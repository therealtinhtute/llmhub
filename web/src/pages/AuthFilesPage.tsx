import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
} from 'react';
import { createPortal } from 'react-dom';
import { useTranslation } from 'react-i18next';
import { animate } from 'motion/mini';
import type { AnimationPlaybackControlsWithThen } from 'motion-dom';
import { useInterval } from '@/hooks/useInterval';
import { useHeaderRefresh } from '@/hooks/useHeaderRefresh';
import { AppCard as Card } from '@/components/ui/AppCard';
import { Button } from '@/components/ui/Button';
import { FormInput as Input } from '@/components/ui/FormInput';
import { FormSelect as Select } from '@/components/ui/FormSelect';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { IconChevronDown, IconChevronUp, IconFilterAll, IconSearch } from '@/components/ui/icons';
import { EmptyState } from '@/components/ui/EmptyState';
import { ToggleSwitch } from '@/components/ui/ToggleSwitch';
import {
  MAX_CARD_PAGE_SIZE,
  MIN_CARD_PAGE_SIZE,
  clampCardPageSize,
  getAuthFileIcon,
  getTypeLabel,
  hasAuthFileStatusMessage,
  isRuntimeOnlyAuthFile,
  normalizeProviderKey,
  parsePriorityValue,
  type ResolvedTheme,
} from '@/features/authFiles/constants';
import { AuthFileCard } from '@/features/authFiles/components/AuthFileCard';
import { useAuthFilesData } from '@/features/authFiles/hooks/useAuthFilesData';
import { useAuthFilesStatusBarCache } from '@/features/authFiles/hooks/useAuthFilesStatusBarCache';
import {
  isAuthFilesSortMode,
  readAuthFilesUiState,
  readPersistedAuthFilesCompactMode,
  writeAuthFilesUiState,
  writePersistedAuthFilesCompactMode,
  type AuthFilesSortMode,
} from '@/features/authFiles/uiState';
import { useAuthStore, useThemeStore } from '@/stores';

const easePower3Out = (progress: number) => 1 - (1 - progress) ** 4;
const easePower2In = (progress: number) => progress ** 3;
const BATCH_BAR_BASE_TRANSFORM = 'translateX(-50%)';
const BATCH_BAR_HIDDEN_TRANSFORM = 'translateX(-50%) translateY(56px)';
const DEFAULT_REGULAR_PAGE_SIZE = 9;
const DEFAULT_COMPACT_PAGE_SIZE = 12;

const escapeWildcardSearchSegment = (value: string) =>
  value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

const buildWildcardSearch = (value: string): RegExp | null => {
  if (!value.includes('*')) return null;
  const pattern = value.split('*').map(escapeWildcardSearchSegment).join('.*');
  return new RegExp(pattern, 'i');
};

export function AuthFilesPage() {
  const { t } = useTranslation();
  const connectionStatus = useAuthStore((state) => state.connectionStatus);
  const resolvedTheme: ResolvedTheme = useThemeStore((state) => state.resolvedTheme);

  const [filter, setFilter] = useState<'all' | string>('all');
  const [problemOnly, setProblemOnly] = useState(false);
  const [disabledOnly, setDisabledOnly] = useState(false);
  const [compactMode, setCompactMode] = useState(false);
  const [searchPanelCollapsed, setSearchPanelCollapsed] = useState(true);
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [pageSizeByMode, setPageSizeByMode] = useState({
    regular: DEFAULT_REGULAR_PAGE_SIZE,
    compact: DEFAULT_COMPACT_PAGE_SIZE,
  });
  const [pageSizeInput, setPageSizeInput] = useState('9');
  const [sortMode, setSortMode] = useState<AuthFilesSortMode>('default');
  const [batchActionBarVisible, setBatchActionBarVisible] = useState(false);
  const [uiStateHydrated, setUiStateHydrated] = useState(false);
  const floatingBatchActionsRef = useRef<HTMLDivElement>(null);
  const batchActionAnimationRef = useRef<AnimationPlaybackControlsWithThen | null>(null);
  const previousSelectionCountRef = useRef(0);
  const selectionCountRef = useRef(0);

  const {
    files,
    selectedFiles,
    selectionCount,
    loading,
    error,
    uploading,
    deleting,
    deletingAll,
    statusUpdating,
    batchStatusUpdating,
    usageResetting,
    fileInputRef,
    loadFiles,
    handleUploadClick,
    handleFileChange,
    handleDelete,
    handleDeleteAll,
    handleResetUsage,
    handleDownload,
    handleStatusToggle,
    toggleSelect,
    selectAllVisible,
    invertVisibleSelection,
    deselectAll,
    batchDownload,
    batchSetStatus,
    batchDelete,
  } = useAuthFilesData();

  const statusBarCache = useAuthFilesStatusBarCache(files);

  const disableControls = connectionStatus !== 'connected';
  const normalizedFilter = normalizeProviderKey(String(filter));
  const pageSize = compactMode ? pageSizeByMode.compact : pageSizeByMode.regular;

  useEffect(() => {
    const persistedCompactMode = readPersistedAuthFilesCompactMode();
    if (typeof persistedCompactMode === 'boolean') {
      setCompactMode(persistedCompactMode);
    }

    const persisted = readAuthFilesUiState();
    if (persisted) {
      if (typeof persisted.filter === 'string' && persisted.filter.trim()) {
        setFilter(normalizeProviderKey(persisted.filter));
      }
      if (typeof persisted.problemOnly === 'boolean') {
        setProblemOnly(persisted.problemOnly);
      }
      if (typeof persisted.disabledOnly === 'boolean') {
        setDisabledOnly(persisted.disabledOnly);
      }
      if (
        typeof persistedCompactMode !== 'boolean' &&
        typeof persisted.compactMode === 'boolean'
      ) {
        setCompactMode(persisted.compactMode);
      }
      if (typeof persisted.search === 'string') {
        setSearch(persisted.search);
      }
      if (typeof persisted.searchPanelCollapsed === 'boolean') {
        setSearchPanelCollapsed(persisted.searchPanelCollapsed);
      }
      if (typeof persisted.page === 'number' && Number.isFinite(persisted.page)) {
        setPage(Math.max(1, Math.round(persisted.page)));
      }
      const legacyPageSize =
        typeof persisted.pageSize === 'number' && Number.isFinite(persisted.pageSize)
          ? clampCardPageSize(persisted.pageSize)
          : null;
      const regularPageSize =
        typeof persisted.regularPageSize === 'number' && Number.isFinite(persisted.regularPageSize)
          ? clampCardPageSize(persisted.regularPageSize)
          : legacyPageSize ?? DEFAULT_REGULAR_PAGE_SIZE;
      const compactPageSize =
        typeof persisted.compactPageSize === 'number' && Number.isFinite(persisted.compactPageSize)
          ? clampCardPageSize(persisted.compactPageSize)
          : legacyPageSize ?? DEFAULT_COMPACT_PAGE_SIZE;
      setPageSizeByMode({
        regular: regularPageSize,
        compact: compactPageSize,
      });
      if (isAuthFilesSortMode(persisted.sortMode)) {
        setSortMode(persisted.sortMode);
      }
    }

    setUiStateHydrated(true);
  }, []);

  useEffect(() => {
    if (!uiStateHydrated) return;

    writeAuthFilesUiState({
      filter,
      problemOnly,
      disabledOnly,
      compactMode,
      searchPanelCollapsed,
      search,
      page,
      pageSize,
      regularPageSize: pageSizeByMode.regular,
      compactPageSize: pageSizeByMode.compact,
      sortMode,
    });
    writePersistedAuthFilesCompactMode(compactMode);
  }, [
    compactMode,
    disabledOnly,
    filter,
    page,
    pageSize,
    pageSizeByMode,
    problemOnly,
    searchPanelCollapsed,
    search,
    sortMode,
    uiStateHydrated,
  ]);

  useEffect(() => {
    setPageSizeInput(String(pageSize));
  }, [pageSize]);

  const setCurrentModePageSize = useCallback(
    (next: number) => {
      setPageSizeByMode((current) =>
        compactMode ? { ...current, compact: next } : { ...current, regular: next }
      );
    },
    [compactMode]
  );

  const commitPageSizeInput = (rawValue: string) => {
    const trimmed = rawValue.trim();
    if (!trimmed) {
      setPageSizeInput(String(pageSize));
      return;
    }

    const value = Number(trimmed);
    if (!Number.isFinite(value)) {
      setPageSizeInput(String(pageSize));
      return;
    }

    const next = clampCardPageSize(value);
    setCurrentModePageSize(next);
    setPageSizeInput(String(next));
    setPage(1);
  };

  const handlePageSizeChange = (event: ChangeEvent<HTMLInputElement>) => {
    const rawValue = event.currentTarget.value;
    setPageSizeInput(rawValue);

    const trimmed = rawValue.trim();
    if (!trimmed) return;

    const parsed = Number(trimmed);
    if (!Number.isFinite(parsed)) return;

    const rounded = Math.round(parsed);
    if (rounded < MIN_CARD_PAGE_SIZE || rounded > MAX_CARD_PAGE_SIZE) return;

    setCurrentModePageSize(rounded);
    setPage(1);
  };

  const handleSortModeChange = useCallback(
    (value: string) => {
      if (!isAuthFilesSortMode(value) || value === sortMode) return;
      setSortMode(value);
      setPage(1);
      void loadFiles().catch(() => {});
    },
    [loadFiles, sortMode]
  );

  const handleHeaderRefresh = useCallback(async () => {
    await loadFiles();
  }, [loadFiles]);

  useHeaderRefresh(handleHeaderRefresh);

  useEffect(() => {
    loadFiles();
  }, [loadFiles]);

  useInterval(
    () => {
      void loadFiles().catch(() => {});
    },
    240_000
  );

  const existingTypes = useMemo(() => {
    const types = new Set<string>();
    files.forEach((file) => {
      const type = normalizeProviderKey(String(file.type ?? file.provider ?? ''));
      if (type) types.add(type);
    });
    return Array.from(types).sort((a, b) => getTypeLabel(t, a).localeCompare(getTypeLabel(t, b)));
  }, [files, t]);

  const filesMatchingStatusFilters = useMemo(
    () =>
      files.filter((file) => {
        if (problemOnly && !hasAuthFileStatusMessage(file)) return false;
        if (disabledOnly && file.disabled !== true) return false;
        return true;
      }),
    [disabledOnly, files, problemOnly]
  );

  const sortOptions = useMemo(
    () => [
      { value: 'default', label: t('auth_files.sort_default') },
      { value: 'az', label: t('auth_files.sort_az') },
      { value: 'priority', label: t('auth_files.sort_priority') },
    ],
    [t]
  );

  const typeCounts = useMemo(() => {
    const counts: Record<string, number> = { all: filesMatchingStatusFilters.length };
    filesMatchingStatusFilters.forEach((file) => {
      const type = normalizeProviderKey(String(file.type ?? file.provider ?? ''));
      if (!type) return;
      counts[type] = (counts[type] || 0) + 1;
    });
    return counts;
  }, [filesMatchingStatusFilters]);

  const providerTabs = useMemo(
    () =>
      existingTypes
        .map((type) => ({
          type,
          count: typeCounts[type] ?? 0,
          label: getTypeLabel(t, type),
          iconSrc: getAuthFileIcon(type, resolvedTheme),
        }))
        .filter(({ count }) => count > 0),
    [existingTypes, resolvedTheme, t, typeCounts]
  );

  useEffect(() => {
    if (normalizedFilter === 'all') return;
    if (providerTabs.some((tab) => tab.type === normalizedFilter)) return;
    setFilter('all');
    setPage(1);
  }, [normalizedFilter, providerTabs]);

  const normalizedSearch = search.trim();
  const wildcardSearch = useMemo(() => buildWildcardSearch(normalizedSearch), [normalizedSearch]);

  const filtered = useMemo(() => {
    const normalizedTerm = normalizedSearch.toLowerCase();

    return filesMatchingStatusFilters.filter((item) => {
      const type = normalizeProviderKey(String(item.type ?? item.provider ?? ''));
      const matchType = normalizedFilter === 'all' || type === normalizedFilter;
      const matchSearch =
        !normalizedSearch ||
        [item.name, item.type, item.provider].some((value) => {
          const content = (value || '').toString();
          return wildcardSearch
            ? wildcardSearch.test(content)
            : content.toLowerCase().includes(normalizedTerm);
        });
      return matchType && matchSearch;
    });
  }, [filesMatchingStatusFilters, normalizedFilter, normalizedSearch, wildcardSearch]);

  const sorted = useMemo(() => {
    const copy = [...filtered];
    if (sortMode === 'default') {
      copy.sort((a, b) => {
        const providerA = normalizeProviderKey(String(a.provider ?? a.type ?? 'unknown'));
        const providerB = normalizeProviderKey(String(b.provider ?? b.type ?? 'unknown'));
        const providerCompare = providerA.localeCompare(providerB);
        if (providerCompare !== 0) return providerCompare;
        return a.name.localeCompare(b.name);
      });
    } else if (sortMode === 'az') {
      copy.sort((a, b) => a.name.localeCompare(b.name));
    } else if (sortMode === 'priority') {
      copy.sort((a, b) => {
        const pa = parsePriorityValue(a.priority ?? a['priority']) ?? 0;
        const pb = parsePriorityValue(b.priority ?? b['priority']) ?? 0;
        return pb - pa; // 高优先级排前面
      });
    }
    return copy;
  }, [filtered, sortMode]);

  const totalPages = Math.max(1, Math.ceil(sorted.length / pageSize));
  const currentPage = Math.min(page, totalPages);
  const start = (currentPage - 1) * pageSize;
  const pageItems = sorted.slice(start, start + pageSize);
  const selectablePageItems = useMemo(
    () => pageItems.filter((file) => !isRuntimeOnlyAuthFile(file)),
    [pageItems]
  );
  const selectableFilteredItems = useMemo(
    () => sorted.filter((file) => !isRuntimeOnlyAuthFile(file)),
    [sorted]
  );
  const selectedNames = useMemo(() => Array.from(selectedFiles), [selectedFiles]);
  const selectedHasStatusUpdating = useMemo(
    () => selectedNames.some((name) => statusUpdating[name] === true),
    [selectedNames, statusUpdating]
  );
  const batchStatusButtonsDisabled =
    disableControls ||
    selectedNames.length === 0 ||
    batchStatusUpdating ||
    selectedHasStatusUpdating;

  useLayoutEffect(() => {
    if (typeof window === 'undefined') return;

    const actionsEl = floatingBatchActionsRef.current;
    if (!actionsEl) {
      document.documentElement.style.removeProperty('--auth-files-action-bar-height');
      return;
    }

    const updatePadding = () => {
      const height = actionsEl.getBoundingClientRect().height;
      document.documentElement.style.setProperty('--auth-files-action-bar-height', `${height}px`);
    };

    updatePadding();
    window.addEventListener('resize', updatePadding);

    const ro = typeof ResizeObserver === 'undefined' ? null : new ResizeObserver(updatePadding);
    ro?.observe(actionsEl);

    return () => {
      ro?.disconnect();
      window.removeEventListener('resize', updatePadding);
      document.documentElement.style.removeProperty('--auth-files-action-bar-height');
    };
  }, [batchActionBarVisible, selectionCount]);

  useEffect(() => {
    selectionCountRef.current = selectionCount;
    if (selectionCount > 0) {
      setBatchActionBarVisible(true);
    }
  }, [selectionCount]);

  useLayoutEffect(() => {
    if (!batchActionBarVisible) return;
    const currentCount = selectionCount;
    const previousCount = previousSelectionCountRef.current;
    const actionsEl = floatingBatchActionsRef.current;
    if (!actionsEl) return;

    batchActionAnimationRef.current?.stop();
    batchActionAnimationRef.current = null;

    if (currentCount > 0 && previousCount === 0) {
      batchActionAnimationRef.current = animate(
        actionsEl,
        {
          transform: [BATCH_BAR_HIDDEN_TRANSFORM, BATCH_BAR_BASE_TRANSFORM],
          opacity: [0, 1],
        },
        {
          duration: 0.28,
          ease: easePower3Out,
          onComplete: () => {
            actionsEl.style.transform = BATCH_BAR_BASE_TRANSFORM;
            actionsEl.style.opacity = '1';
          },
        }
      );
    } else if (currentCount === 0 && previousCount > 0) {
      batchActionAnimationRef.current = animate(
        actionsEl,
        {
          transform: [BATCH_BAR_BASE_TRANSFORM, BATCH_BAR_HIDDEN_TRANSFORM],
          opacity: [1, 0],
        },
        {
          duration: 0.22,
          ease: easePower2In,
          onComplete: () => {
            if (selectionCountRef.current === 0) {
              setBatchActionBarVisible(false);
            }
          },
        }
      );
    }

    previousSelectionCountRef.current = currentCount;
  }, [batchActionBarVisible, selectionCount]);

  useEffect(
    () => () => {
      batchActionAnimationRef.current?.stop();
      batchActionAnimationRef.current = null;
    },
    []
  );

  const titleNode = (
    <div className="flex items-center gap-2 leading-6">
      <span>{t('auth_files.title_section')}</span>
      {files.length > 0 && <span className="inline-flex items-center justify-center h-6 min-w-[24px] px-2 text-[13px] font-semibold text-primary bg-primary/10 box-border">{files.length}</span>}
    </div>
  );

  const deleteAllButtonLabel = (() => {
    if (disabledOnly) {
      return t('auth_files.delete_filtered_result_button');
    }
    if (problemOnly) {
      return normalizedFilter === 'all'
        ? t('auth_files.delete_problem_button')
        : t('auth_files.delete_problem_button_with_type', {
            type: getTypeLabel(t, normalizedFilter),
          });
    }
    return normalizedFilter === 'all'
      ? t('auth_files.delete_all_button')
      : `${t('common.delete')} ${getTypeLabel(t, normalizedFilter)}`;
  })();

  return (
    <div className="flex flex-col gap-4 pb-[calc(var(--auth-files-action-bar-height,0px)+16px+env(safe-area-inset-bottom))]">
      <div className="flex flex-col gap-2">
        <h1 className="text-[28px] font-bold text-foreground m-0">{t('auth_files.title')}</h1>
        <p className="text-[14px] text-muted-foreground m-0">{t('auth_files.description')}</p>
      </div>

      <Card
        title={titleNode}
        extra={
          <div className="flex gap-2 flex-wrap items-center min-w-0 max-w-full max-md:w-full">
            <Button variant="secondary" size="sm" onClick={handleHeaderRefresh} disabled={loading}>
              {t('common.refresh')}
            </Button>
            <Button
              size="sm"
              onClick={handleUploadClick}
              disabled={disableControls || uploading}
              loading={uploading}
            >
              {t('auth_files.upload_button')}
            </Button>
            <Button
              variant="danger"
              size="sm"
              onClick={() =>
                handleDeleteAll({
                  filter,
                  problemOnly,
                  disabledOnly,
                  onResetFilterToAll: () => setFilter('all'),
                  onResetProblemOnly: () => setProblemOnly(false),
                  onResetDisabledOnly: () => setDisabledOnly(false),
                })
              }
              disabled={disableControls || loading || deletingAll}
              loading={deletingAll}
            >
              {deleteAllButtonLabel}
            </Button>
            <input
              ref={fileInputRef}
              type="file"
              accept=".json,application/json"
              multiple
              style={{ display: 'none' }}
              onChange={handleFileChange}
            />
          </div>
        }
      >
        {error && <div className="p-[10px_14px] mb-2 bg-destructive/10 border border-destructive/35 text-destructive text-sm leading-[1.5]">{error}</div>}

        <div className="flex flex-col gap-3 mb-4">
          <Tabs
            value={normalizedFilter}
            onValueChange={(value) => {
              setFilter(value);
              setPage(1);
            }}
          >
            <TabsList className="flex justify-start items-start gap-0 p-0 border-b border-border bg-transparent overflow-x-auto max-w-full">
              <TabsTrigger
                value="all"
                className="min-h-[32px] px-3 py-1 gap-1.5 text-muted-foreground border-b-2 border-transparent -mb-px data-[state=active]:border-primary data-[state=active]:text-primary hover:text-foreground"
              >
                <span className="inline-flex items-center gap-1.5">
                  <IconFilterAll data-icon="inline-start" />
                  {t('auth_files.filter_all')}
                </span>
                {typeCounts.all > 0 && (
                  <span className="inline-flex min-w-5 items-center justify-center bg-primary/10 px-1.5 py-0.5 text-[11px] font-semibold text-primary">
                    {typeCounts.all}
                  </span>
                )}
              </TabsTrigger>
              {providerTabs.map(({ type, count, label, iconSrc }) => (
                <TabsTrigger
                  key={type}
                  value={type}
                  className="min-h-[32px] px-3 py-1 gap-1.5 text-muted-foreground border-b-2 border-transparent -mb-px data-[state=active]:border-primary data-[state=active]:text-primary hover:text-foreground"
                >
                  <span className="inline-flex items-center gap-1.5">
                    {iconSrc ? (
                      <img src={iconSrc} alt="" className="size-4 shrink-0 object-contain" />
                    ) : (
                      <span className="inline-flex size-4 items-center justify-center text-[11px] font-bold leading-none">
                        {label.slice(0, 1).toUpperCase()}
                      </span>
                    )}
                    <span className="truncate">{label}</span>
                  </span>
                  <span className="inline-flex min-w-5 items-center justify-center bg-primary/10 px-1.5 py-0.5 text-[11px] font-semibold text-primary">
                    {count}
                  </span>
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>

          <div className="flex flex-col gap-3 min-w-0">
            <div className="relative overflow-hidden p-4 border border-border/60 bg-[linear-gradient(135deg,color-mix(in_srgb,hsl(var(--primary))_8%,transparent),transparent_46%),color-mix(in_srgb,hsl(var(--muted))_56%,transparent)] max-md:p-3">
              <div className="flex items-center justify-between gap-3">
                <div className="min-w-0">
                  <div className="text-[11px] font-bold uppercase tracking-[0.08em] text-muted-foreground/60">
                    {t('auth_files.search_label')}
                  </div>
                  <div className="text-[13px] font-semibold text-foreground">
                    {t('auth_files.display_options_label')}
                  </div>
                </div>
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={() => setSearchPanelCollapsed((value) => !value)}
                  aria-expanded={!searchPanelCollapsed}
                  title={
                    searchPanelCollapsed ? t('common.expand') : t('common.collapse')
                  }
                >
                  {searchPanelCollapsed ? (
                    <IconChevronDown data-icon="inline-end" />
                  ) : (
                    <IconChevronUp data-icon="inline-end" />
                  )}
                  {searchPanelCollapsed ? t('common.expand') : t('common.collapse')}
                </Button>
              </div>

              {!searchPanelCollapsed && (
                <div className="mt-4 grid gap-3 items-end [grid-template-columns:minmax(260px,1fr)_minmax(108px,0.35fr)_minmax(148px,0.45fr)] max-md:[grid-template-columns:1fr]">
                  <div className="flex flex-col gap-[6px] min-w-0">
                    <label className="text-[11px] text-muted-foreground/60 font-bold whitespace-nowrap">{t('auth_files.search_label')}</label>
                    <Input
                      className="pr-10"
                      value={search}
                      onChange={(e) => {
                        setSearch(e.target.value);
                        setPage(1);
                      }}
                      placeholder={t('auth_files.search_placeholder')}
                      rightElement={<IconSearch className="block text-muted-foreground/60 pointer-events-none" size={18} />}
                    />
                  </div>
                  <div className="flex flex-col gap-[6px] min-w-0">
                    <label className="text-[11px] text-muted-foreground/60 font-bold whitespace-nowrap">{t('auth_files.page_size_label')}</label>
                    <input
                      className="w-full px-3 py-2 border border-border/70 bg-muted text-foreground text-[14px] cursor-text h-[42px] box-border focus:outline-none focus:border-primary"
                      type="number"
                      min={MIN_CARD_PAGE_SIZE}
                      max={MAX_CARD_PAGE_SIZE}
                      step={1}
                      value={pageSizeInput}
                      onChange={handlePageSizeChange}
                      onBlur={(e) => commitPageSizeInput(e.currentTarget.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') {
                          e.currentTarget.blur();
                        }
                      }}
                    />
                  </div>
                  <div className="flex flex-col gap-[6px] min-w-0">
                    <label className="text-[11px] text-muted-foreground/60 font-bold whitespace-nowrap">{t('auth_files.sort_label')}</label>
                    <Select
                      className="w-full min-w-0"
                      value={sortMode}
                      options={sortOptions}
                      onChange={handleSortModeChange}
                      ariaLabel={t('auth_files.sort_label')}
                      fullWidth
                    />
                  </div>
                  <div className="flex flex-col gap-[6px] min-w-0 [grid-column:1_/_-1]">
                    <label className="text-[11px] text-muted-foreground/60 font-bold whitespace-nowrap">{t('auth_files.display_options_label')}</label>
                    <div className="grid [grid-template-columns:repeat(3,minmax(0,1fr))] gap-2 min-h-[42px] max-md:[grid-template-columns:1fr]">
                      <div className="flex items-center min-w-0 min-h-[42px] px-[10px] border border-border/60 bg-background/56 [&>label]:w-full [&>label]:min-w-0">
                        <ToggleSwitch
                          checked={problemOnly}
                          onChange={(value) => {
                            setProblemOnly(value);
                            setPage(1);
                          }}
                          ariaLabel={t('auth_files.problem_filter_only')}
                          label={
                            <span className="inline-flex items-center text-foreground text-[13px] font-semibold leading-[1.25]">
                              {t('auth_files.problem_filter_only')}
                            </span>
                          }
                        />
                      </div>
                      <div className="flex items-center min-w-0 min-h-[42px] px-[10px] border border-border/60 bg-background/56 [&>label]:w-full [&>label]:min-w-0">
                        <ToggleSwitch
                          checked={disabledOnly}
                          onChange={(value) => {
                            setDisabledOnly(value);
                            setPage(1);
                          }}
                          ariaLabel={t('auth_files.disabled_filter_only')}
                          label={
                            <span className="inline-flex items-center text-foreground text-[13px] font-semibold leading-[1.25]">
                              {t('auth_files.disabled_filter_only')}
                            </span>
                          }
                        />
                      </div>
                      <div className="flex items-center min-w-0 min-h-[42px] px-[10px] border border-border/60 bg-background/56 [&>label]:w-full [&>label]:min-w-0">
                        <ToggleSwitch
                          checked={compactMode}
                          onChange={(value) => setCompactMode(value)}
                          ariaLabel={t('auth_files.compact_mode_label')}
                          label={
                            <span className="inline-flex items-center text-foreground text-[13px] font-semibold leading-[1.25]">
                              {t('auth_files.compact_mode_label')}
                            </span>
                          }
                        />
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>

            {loading ? (
              <div className="text-[13px] text-muted-foreground leading-[1.55]">{t('common.loading')}</div>
            ) : pageItems.length === 0 ? (
              <EmptyState
                title={t('auth_files.search_empty_title')}
                description={t('auth_files.search_empty_desc')}
              />
            ) : (
              <div
                className={`grid gap-3 ${compactMode ? '[grid-template-columns:repeat(auto-fill,minmax(min(100%,280px),1fr))]' : '[grid-template-columns:repeat(auto-fill,minmax(min(100%,320px),1fr))]'}`}
              >
                {pageItems.map((file) => (
                  <AuthFileCard
                    key={file.name}
                    file={file}
                    compact={compactMode}
                    selected={selectedFiles.has(file.name)}
                    resolvedTheme={resolvedTheme}
                    disableControls={disableControls}
                    deleting={deleting}
                    statusUpdating={statusUpdating}
                    usageResetting={usageResetting}
                    statusBarCache={statusBarCache}
                    onDownload={handleDownload}
                    onDelete={handleDelete}
                    onToggleStatus={handleStatusToggle}
                    onToggleSelect={toggleSelect}
                    onResetUsage={handleResetUsage}
                  />
                ))}
              </div>
            )}

            {!loading && sorted.length > pageSize && (
              <div className="flex justify-center items-center gap-4 flex-wrap mt-4 pt-3 border-t border-border">
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => setPage(Math.max(1, currentPage - 1))}
                  disabled={currentPage <= 1}
                >
                  {t('auth_files.pagination_prev')}
                </Button>
                <div className="text-[13px] text-muted-foreground px-4 py-1 bg-muted">
                  {t('auth_files.pagination_info', {
                    current: currentPage,
                    total: totalPages,
                    count: sorted.length,
                  })}
                </div>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => setPage(Math.min(totalPages, currentPage + 1))}
                  disabled={currentPage >= totalPages}
                >
                  {t('auth_files.pagination_next')}
                </Button>
              </div>
            )}
          </div>
        </div>
      </Card>

      {batchActionBarVisible && typeof document !== 'undefined'
        ? createPortal(
            <div
              className="fixed left-[var(--content-center-x,50%)] bottom-[calc(16px+env(safe-area-inset-bottom))] [transform:translateX(-50%)] z-50 w-[min(960px,calc(100vw-24px))] max-w-[calc(100vw-24px)] box-border will-change-transform max-md:w-[calc(100vw-16px)] max-md:max-w-[calc(100vw-16px)] max-md:bottom-[calc(12px+env(safe-area-inset-bottom))]"
              ref={floatingBatchActionsRef}
            >
              <div className="flex items-center justify-between gap-2 p-[10px_12px] border border-border/70 bg-background/84 backdrop-blur-[12px] shadow-lg max-md:flex-col max-md:items-stretch">
                <div className="flex items-center gap-1 flex-wrap max-md:justify-center">
                  <span className="text-[13px] font-semibold text-foreground mr-[2px]">
                    {t('auth_files.batch_selected', { count: selectionCount })}
                  </span>
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => selectAllVisible(pageItems)}
                    disabled={selectablePageItems.length === 0}
                  >
                    {t('auth_files.batch_select_page')}
                  </Button>
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => selectAllVisible(sorted)}
                    disabled={selectableFilteredItems.length === 0}
                  >
                    {t('auth_files.batch_select_filtered')}
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => invertVisibleSelection(pageItems)}
                    disabled={selectablePageItems.length === 0}
                  >
                    {t('auth_files.batch_invert_page')}
                  </Button>
                  <Button variant="ghost" size="sm" onClick={deselectAll}>
                    {t('auth_files.batch_deselect')}
                  </Button>
                </div>
                <div className="flex items-center gap-1 flex-wrap justify-end max-md:justify-center">
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => void batchDownload(selectedNames)}
                    disabled={disableControls || selectedNames.length === 0}
                  >
                    {t('auth_files.batch_download')}
                  </Button>
                  <Button
                    size="sm"
                    onClick={() => batchSetStatus(selectedNames, true)}
                    disabled={batchStatusButtonsDisabled}
                  >
                    {t('auth_files.batch_enable')}
                  </Button>
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => batchSetStatus(selectedNames, false)}
                    disabled={batchStatusButtonsDisabled}
                  >
                    {t('auth_files.batch_disable')}
                  </Button>
                  <Button
                    variant="danger"
                    size="sm"
                    onClick={() => batchDelete(selectedNames)}
                    disabled={disableControls || selectedNames.length === 0}
                  >
                    {t('common.delete')}
                  </Button>
                </div>
              </div>
            </div>,
            document.body
          )
        : null}
    </div>
  );
}
