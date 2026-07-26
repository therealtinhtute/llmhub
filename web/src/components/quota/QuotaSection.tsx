/**
 * Generic quota section component.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { AppCard as Card } from '@/components/ui/AppCard';
import { Button } from '@/components/ui/Button';
import { EmptyState } from '@/components/ui/EmptyState';
import { triggerHeaderRefresh } from '@/hooks/useHeaderRefresh';
import { toast } from 'sonner';
import { useConfirmationStore, useQuotaStore, useThemeStore } from '@/stores';
import type { AuthFileItem, ResolvedTheme } from '@/types';
import { getStatusFromError } from '@/utils/quota';
import { normalizeAuthIndex } from '@/utils/authIndex';
import { quotaApi } from '@/services/api';
import { QuotaCard } from './QuotaCard';
import type { QuotaStatusState } from './QuotaCard';
import { useQuotaLoader } from './useQuotaLoader';
import type { QuotaConfig } from './quotaConfigs';
import { useGridColumns } from './useGridColumns';
import { IconRefreshCw } from '@/components/ui/icons';
import { quotaStyles as styles } from './quotaStyles';

type QuotaUpdater<T> = T | ((prev: T) => T);

type QuotaSetter<T> = (updater: QuotaUpdater<T>) => void;

type ViewMode = 'paged' | 'all';

const resolveResetErrorMessageKey = (status: number | undefined): string => {
  switch (status) {
    case 400:
      return 'quota_management.reset_error_400';
    case 404:
      return 'quota_management.reset_error_404';
    case 500:
      return 'quota_management.reset_error_500';
    case 503:
      return 'quota_management.reset_error_503';
    default:
      return 'quota_management.reset_error_generic';
  }
};

const MAX_ITEMS_PER_PAGE = 25;
const MAX_SHOW_ALL_THRESHOLD = 30;

interface QuotaPaginationState<T> {
  pageSize: number;
  totalPages: number;
  currentPage: number;
  pageItems: T[];
  setPageSize: (size: number) => void;
  goToPrev: () => void;
  goToNext: () => void;
  loading: boolean;
  loadingScope: 'page' | 'all' | null;
  setLoading: (loading: boolean, scope?: 'page' | 'all' | null) => void;
}

const useQuotaPagination = <T,>(items: T[], defaultPageSize = 6): QuotaPaginationState<T> => {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSizeState] = useState(defaultPageSize);
  const [loading, setLoadingState] = useState(false);
  const [loadingScope, setLoadingScope] = useState<'page' | 'all' | null>(null);

  const totalPages = useMemo(
    () => Math.max(1, Math.ceil(items.length / pageSize)),
    [items.length, pageSize]
  );

  const currentPage = useMemo(() => Math.min(page, totalPages), [page, totalPages]);

  const pageItems = useMemo(() => {
    const start = (currentPage - 1) * pageSize;
    return items.slice(start, start + pageSize);
  }, [items, currentPage, pageSize]);

  const setPageSize = useCallback((size: number) => {
    setPageSizeState(size);
    setPage(1);
  }, []);

  const goToPrev = useCallback(() => {
    setPage((prev) => Math.max(1, prev - 1));
  }, []);

  const goToNext = useCallback(() => {
    setPage((prev) => Math.min(totalPages, prev + 1));
  }, [totalPages]);

  const setLoading = useCallback((isLoading: boolean, scope?: 'page' | 'all' | null) => {
    setLoadingState(isLoading);
    setLoadingScope(isLoading ? (scope ?? null) : null);
  }, []);

  return {
    pageSize,
    totalPages,
    currentPage,
    pageItems,
    setPageSize,
    goToPrev,
    goToNext,
    loading,
    loadingScope,
    setLoading,
  };
};

interface QuotaSectionProps<TState extends QuotaStatusState, TData> {
  config: QuotaConfig<TState, TData>;
  files: AuthFileItem[];
  loading: boolean;
  disabled: boolean;
  viewMode?: 'paged' | 'all';
  onViewModeChange?: (mode: 'paged' | 'all') => void;
  refreshSignal?: number;
}

export function QuotaSection<TState extends QuotaStatusState, TData>({
  config,
  files,
  loading,
  disabled,
  viewMode: externalViewMode,
  onViewModeChange,
  refreshSignal,
}: QuotaSectionProps<TState, TData>) {
  const { t } = useTranslation();
  const resolvedTheme: ResolvedTheme = useThemeStore((state) => state.resolvedTheme);
  const setQuota = useQuotaStore((state) => state[config.storeSetter]) as QuotaSetter<
    Record<string, TState>
  >;
  const showConfirmation = useConfirmationStore((state) => state.showConfirmation);
  const [resettingNames, setResettingNames] = useState<Set<string>>(new Set());

  /* Removed useRef */
  const [columns, gridRef] = useGridColumns(300);
  const [internalViewMode, setInternalViewMode] = useState<ViewMode>('paged');
  const [showTooManyWarning, setShowTooManyWarning] = useState(false);
  const viewMode = externalViewMode ?? internalViewMode;
  const setViewMode = useCallback(
    (mode: ViewMode) => {
      if (externalViewMode !== undefined) {
        onViewModeChange?.(mode);
      } else {
        setInternalViewMode(mode);
      }
    },
    [externalViewMode, onViewModeChange]
  );

  const filteredFiles = useMemo(
    () => files.filter((file) => config.filterFn(file)),
    [files, config]
  );
  const showAllAllowed = filteredFiles.length <= MAX_SHOW_ALL_THRESHOLD;
  const effectiveViewMode: ViewMode = viewMode === 'all' && !showAllAllowed ? 'paged' : viewMode;

  const {
    pageSize,
    totalPages,
    currentPage,
    pageItems,
    setPageSize,
    goToPrev,
    goToNext,
    loading: sectionLoading,
    setLoading,
  } = useQuotaPagination(filteredFiles);

  useEffect(() => {
    if (showAllAllowed) return;
    if (viewMode !== 'all') return;

    let cancelled = false;
    queueMicrotask(() => {
      if (cancelled) return;
      setViewMode('paged');
      setShowTooManyWarning(true);
    });

    return () => {
      cancelled = true;
    };
  }, [showAllAllowed, viewMode, setViewMode]);

  // Update page size based on view mode and columns
  useEffect(() => {
    if (effectiveViewMode === 'all') {
      setPageSize(Math.max(1, filteredFiles.length));
    } else {
      // Paged mode: 3 rows * columns, capped to avoid oversized pages.
      setPageSize(Math.min(columns * 3, MAX_ITEMS_PER_PAGE));
    }
  }, [effectiveViewMode, columns, filteredFiles.length, setPageSize]);

  const { quota, loadQuota } = useQuotaLoader(config);

  const pendingQuotaRefreshRef = useRef(false);
  const prevFilesLoadingRef = useRef(loading);
  const prevRefreshSignalRef = useRef<number | undefined>(undefined);

  useEffect(() => {
    if (refreshSignal === undefined) return;
    if (prevRefreshSignalRef.current === refreshSignal) return;
    const isFirst = prevRefreshSignalRef.current === undefined;
    prevRefreshSignalRef.current = refreshSignal;
    if (isFirst) return;
    pendingQuotaRefreshRef.current = true;
  }, [refreshSignal]);

  const handleRefresh = useCallback(() => {
    pendingQuotaRefreshRef.current = true;
    void triggerHeaderRefresh();
  }, []);

  useEffect(() => {
    const wasLoading = prevFilesLoadingRef.current;
    prevFilesLoadingRef.current = loading;

    if (!pendingQuotaRefreshRef.current) return;
    if (loading) return;
    if (!wasLoading) return;

    pendingQuotaRefreshRef.current = false;
    const scope = effectiveViewMode === 'all' ? 'all' : 'page';
    const targets = effectiveViewMode === 'all' ? filteredFiles : pageItems;
    if (targets.length === 0) return;
    loadQuota(targets, scope, setLoading);
  }, [loading, effectiveViewMode, filteredFiles, pageItems, loadQuota, setLoading]);

  useEffect(() => {
    if (loading) return;
    if (filteredFiles.length === 0) {
      setQuota({});
      return;
    }
    setQuota((prev) => {
      const nextState: Record<string, TState> = {};
      filteredFiles.forEach((file) => {
        const cached = prev[file.name];
        if (cached) {
          nextState[file.name] = cached;
        } else if (config.buildRuntimeState) {
          nextState[file.name] = config.buildRuntimeState(file);
        }
      });
      return nextState;
    });
  }, [filteredFiles, loading, setQuota]);

  const refreshQuotaForFile = useCallback(
    async (file: AuthFileItem) => {
      if (disabled || file.disabled) return;
      if (quota[file.name]?.status === 'loading') return;

      setQuota((prev) => ({
        ...prev,
        [file.name]: config.buildLoadingState(),
      }));

      try {
        const data = await config.fetchQuota(file, t);
        setQuota((prev) => ({
          ...prev,
          [file.name]: config.buildSuccessState(data),
        }));
        toast.success(t('auth_files.quota_refresh_success', { name: file.name }));
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : t('common.unknown_error');
        const status = getStatusFromError(err);
        setQuota((prev) => ({
          ...prev,
          [file.name]: config.buildErrorState(message, status),
        }));
        toast.error(t('auth_files.quota_refresh_failed', { name: file.name, message }));
      }
    },
    [config, disabled, quota, setQuota, t]
  );

  const resetQuotaForFile = useCallback(
    (item: AuthFileItem) => {
      if (disabled || item.disabled) return;
      const authIndex = normalizeAuthIndex(item['auth_index'] ?? item.authIndex);
      if (!authIndex) return;

      showConfirmation({
        title: t('quota_management.reset_confirm_title'),
        message: t('quota_management.reset_confirm_message', { name: item.name }),
        confirmText: t('quota_management.reset_confirm_action'),
        variant: 'danger',
        onConfirm: async () => {
          setResettingNames((prev) => new Set(prev).add(item.name));
          try {
            await quotaApi.resetQuota(authIndex);
            toast.success(t('quota_management.reset_success', { name: item.name }));
            await refreshQuotaForFile(item);
          } catch (err: unknown) {
            const message = err instanceof Error ? err.message : t('common.unknown_error');
            const status = getStatusFromError(err);
            toast.error(t(resolveResetErrorMessageKey(status), { name: item.name, message }));
          } finally {
            setResettingNames((prev) => {
              const next = new Set(prev);
              next.delete(item.name);
              return next;
            });
          }
        },
      });
    },
    [disabled, refreshQuotaForFile, showConfirmation, t]
  );

  const titleNode = (
    <div className={styles.titleWrapper}>
      <span>{t(`${config.i18nPrefix}.title`)}</span>
      {filteredFiles.length > 0 && (
        <span className={styles.countBadge}>{filteredFiles.length}</span>
      )}
    </div>
  );

  const isRefreshing = sectionLoading || loading;

  return (
    <Card
      title={titleNode}
      extra={
        externalViewMode === undefined ? (
          <div className={styles.headerActions}>
            <div className={styles.viewModeToggle}>
              <Button
                variant="secondary"
                size="sm"
                className={`${styles.viewModeButton} ${
                  effectiveViewMode === 'paged' ? styles.viewModeButtonActive : ''
                }`}
                onClick={() => setViewMode('paged')}
              >
                {t('auth_files.view_mode_paged')}
              </Button>
              <Button
                variant="secondary"
                size="sm"
                className={`${styles.viewModeButton} ${
                  effectiveViewMode === 'all' ? styles.viewModeButtonActive : ''
                }`}
                onClick={() => {
                  if (filteredFiles.length > MAX_SHOW_ALL_THRESHOLD) {
                    setShowTooManyWarning(true);
                  } else {
                    setViewMode('all');
                  }
                }}
              >
                {t('auth_files.view_mode_all')}
              </Button>
            </div>
            <Button
              variant="secondary"
              size="sm"
              className={styles.refreshAllButton}
              onClick={handleRefresh}
              disabled={disabled || isRefreshing}
              loading={isRefreshing}
              title={t('quota_management.refresh_all_credentials')}
              aria-label={t('quota_management.refresh_all_credentials')}
            >
              {!isRefreshing && <IconRefreshCw size={16} />}
              {t('quota_management.refresh_all_credentials')}
            </Button>
          </div>
        ) : undefined
      }
    >
      {filteredFiles.length === 0 ? (
        <EmptyState
          title={t(`${config.i18nPrefix}.empty_title`)}
          description={t(`${config.i18nPrefix}.empty_desc`)}
        />
      ) : (
        <>
          <div ref={gridRef} className={config.gridClassName}>
            {pageItems.map((item) => (
              <QuotaCard
                key={item.name}
                item={item}
                quota={quota[item.name]}
                resolvedTheme={resolvedTheme}
                i18nPrefix={config.i18nPrefix}
                cardIdleMessageKey={config.cardIdleMessageKey}
                cardClassName={config.cardClassName}
                defaultType={config.type}
                canRefresh={!disabled && !item.disabled}
                onRefresh={() => void refreshQuotaForFile(item)}
                canResetQuota={!disabled && !item.disabled}
                resettingQuota={resettingNames.has(item.name)}
                onResetQuota={() => resetQuotaForFile(item)}
                renderQuotaItems={config.renderQuotaItems}
              />
            ))}
          </div>
          {filteredFiles.length > pageSize && effectiveViewMode === 'paged' && (
            <div className={styles.pagination}>
              <Button variant="secondary" size="sm" onClick={goToPrev} disabled={currentPage <= 1}>
                {t('auth_files.pagination_prev')}
              </Button>
              <div className={styles.pageInfo}>
                {t('auth_files.pagination_info', {
                  current: currentPage,
                  total: totalPages,
                  count: filteredFiles.length,
                })}
              </div>
              <Button
                variant="secondary"
                size="sm"
                onClick={goToNext}
                disabled={currentPage >= totalPages}
              >
                {t('auth_files.pagination_next')}
              </Button>
            </div>
          )}
        </>
      )}
      {showTooManyWarning && (
        <div className={styles.warningOverlay} onClick={() => setShowTooManyWarning(false)}>
          <div className={styles.warningModal} onClick={(e) => e.stopPropagation()}>
            <p>{t('auth_files.too_many_files_warning')}</p>
            <Button variant="primary" size="sm" onClick={() => setShowTooManyWarning(false)}>
              {t('common.confirm')}
            </Button>
          </div>
        </div>
      )}
    </Card>
  );
}
