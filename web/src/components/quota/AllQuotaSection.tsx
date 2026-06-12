import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useShallow } from 'zustand/shallow';
import { Button } from '@/components/ui/Button';
import { EmptyState } from '@/components/ui/EmptyState';
import { toast } from 'sonner';
import { useQuotaStore, useThemeStore } from '@/stores';
import type { AuthFileItem, KiroQuotaState, ResolvedTheme } from '@/types';
import { authFilesApi } from '@/services/api';
import { getStatusFromError } from '@/utils/quota';
import { normalizeAuthIndex } from '@/utils/authIndex';
import { QuotaCard } from './QuotaCard';
import type { QuotaStatusState } from './QuotaCard';
import { buildKiroQuotaStateFromProvider, type QuotaConfig, type QuotaStore } from './quotaConfigs';
import { useGridColumns } from './useGridColumns';
import { quotaStyles as styles } from './quotaStyles';

type ViewMode = 'paged' | 'all';

const MAX_ITEMS_PER_PAGE = 25;
const MAX_SHOW_ALL_THRESHOLD = 30;

interface CombinedItem {
  item: AuthFileItem;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  config: QuotaConfig<any, any>;
}

export interface AllQuotaSectionProps {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  configs: QuotaConfig<any, any>[];
  files: AuthFileItem[];
  loading: boolean;
  disabled: boolean;
  viewMode?: ViewMode;
  onViewModeChange?: (mode: ViewMode) => void;
  refreshSignal?: number;
}

type QuotaUpdater<T> = T | ((prev: T) => T);
type QuotaSetter<T> = (updater: QuotaUpdater<T>) => void;

const useQuotaPagination = <T,>(items: T[], defaultPageSize = 6) => {
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

export function AllQuotaSection({ configs, files, loading, disabled, viewMode: externalViewMode, onViewModeChange, refreshSignal }: AllQuotaSectionProps) {
  const { t } = useTranslation();
  const resolvedTheme: ResolvedTheme = useThemeStore((state) => state.resolvedTheme);

  const allQuota = useQuotaStore(
    useShallow((state: QuotaStore) => ({
      antigravityQuota: state.antigravityQuota,
      claudeQuota: state.claudeQuota,
      codexQuota: state.codexQuota,
      geminiCliQuota: state.geminiCliQuota,
      kiroQuota: state.kiroQuota,
      kimiQuota: state.kimiQuota,
      xaiQuota: state.xaiQuota,
      setAntigravityQuota: state.setAntigravityQuota,
      setClaudeQuota: state.setClaudeQuota,
      setCodexQuota: state.setCodexQuota,
      setGeminiCliQuota: state.setGeminiCliQuota,
      setKiroQuota: state.setKiroQuota,
      setKimiQuota: state.setKimiQuota,
      setXaiQuota: state.setXaiQuota,
      clearQuotaCache: state.clearQuotaCache,
    }))
  );

  const [columns, gridRef] = useGridColumns(380);
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

  const allItems = useMemo(() => {
    const seen = new Set<string>();
    const result: CombinedItem[] = [];
    for (const config of configs) {
      for (const file of files) {
        if (!seen.has(file.name) && config.filterFn(file)) {
          seen.add(file.name);
          result.push({ item: file, config });
        }
      }
    }
    return result;
  }, [configs, files]);

  const showAllAllowed = allItems.length <= MAX_SHOW_ALL_THRESHOLD;
  const effectiveViewMode: ViewMode = viewMode === 'all' && !showAllAllowed ? 'paged' : viewMode;

  const {
    pageSize,
    totalPages,
    currentPage,
    pageItems,
    setPageSize,
    goToPrev,
    goToNext,
    setLoading,
  } = useQuotaPagination(allItems);

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

  useEffect(() => {
    if (effectiveViewMode === 'all') {
      setPageSize(Math.max(1, allItems.length));
    } else {
      setPageSize(Math.min(columns * 3, MAX_ITEMS_PER_PAGE));
    }
  }, [effectiveViewMode, columns, allItems.length, setPageSize]);

  useEffect(() => {
    if (loading) return;

    for (const config of configs) {
      if (!config.buildRuntimeState) continue;
      const providerFiles = files.filter((file) => config.filterFn(file));
      const setter = useQuotaStore.getState()[
        config.storeSetter as keyof QuotaStore
      ] as QuotaSetter<Record<string, QuotaStatusState>>;
      setter((prev) => {
        const nextState: Record<string, QuotaStatusState> = {};
        providerFiles.forEach((file) => {
          nextState[file.name] = prev[file.name] ?? config.buildRuntimeState!(file);
        });
        return nextState;
      });
    }
  }, [configs, files, loading]);

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

  useEffect(() => {
    const wasLoading = prevFilesLoadingRef.current;
    prevFilesLoadingRef.current = loading;

    if (!pendingQuotaRefreshRef.current) return;
    if (loading) return;
    if (!wasLoading) return;

    pendingQuotaRefreshRef.current = false;
    const scope = effectiveViewMode === 'all' ? 'all' : 'page';
    const targets = effectiveViewMode === 'all' ? allItems : pageItems;
    if (targets.length === 0) return;

    setLoading(true, scope);
    void Promise.all(
      targets.map(async ({ item, config }) => {
        try {
          const data = await config.fetchQuota(item, t);
          const setter = useQuotaStore.getState()[
            config.storeSetter as keyof QuotaStore
          ] as QuotaSetter<Record<string, QuotaStatusState>>;
          setter((prev) => ({
            ...prev,
            [item.name]: config.buildSuccessState(data),
          }));
        } catch (err: unknown) {
          const message = err instanceof Error ? err.message : t('common.unknown_error');
          const errorStatus = getStatusFromError(err);
          const setter = useQuotaStore.getState()[
            config.storeSetter as keyof QuotaStore
          ] as QuotaSetter<Record<string, QuotaStatusState>>;
          setter((prev) => ({
            ...prev,
            [item.name]: config.buildErrorState(message, errorStatus),
          }));
        }
      })
    ).finally(() => {
      setLoading(false);
    });
  }, [loading, effectiveViewMode, allItems, pageItems, setLoading, t]);

  const refreshQuotaForItem = useCallback(
    async (item: AuthFileItem, config: QuotaConfig<QuotaStatusState, unknown>) => {
      if (disabled || item.disabled) return;
      const currentQuota = config.storeSelector(useQuotaStore.getState());
      if (currentQuota[item.name]?.status === 'loading') return;

      const setQuota = useQuotaStore.getState()[
        config.storeSetter as keyof QuotaStore
      ] as QuotaSetter<Record<string, QuotaStatusState>>;
      setQuota((prev) => ({
        ...prev,
        [item.name]: config.buildLoadingState(),
      }));

      try {
        const data = await config.fetchQuota(item, t);
        const setter = useQuotaStore.getState()[
          config.storeSetter as keyof QuotaStore
        ] as QuotaSetter<Record<string, QuotaStatusState>>;
        setter((prev) => ({
          ...prev,
          [item.name]: config.buildSuccessState(data),
        }));
        toast.success(t('auth_files.quota_refresh_success', { name: item.name }));
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : t('common.unknown_error');
        const status = getStatusFromError(err);
        const setter = useQuotaStore.getState()[
          config.storeSetter as keyof QuotaStore
        ] as QuotaSetter<Record<string, QuotaStatusState>>;
        setter((prev) => ({
          ...prev,
          [item.name]: config.buildErrorState(message, status),
        }));
        toast.error(t('auth_files.quota_refresh_failed', { name: item.name, message }));
      }
    },
    [disabled, t]
  );

  const setKiroOverageForItem = useCallback(
    async (item: AuthFileItem, enabled: boolean) => {
      if (disabled || item.disabled) return;
      const current = useQuotaStore.getState().kiroQuota[item.name];
      if (!current?.providerQuotaAvailable || current.status === 'loading' || current.overageUpdating) {
        return;
      }

      const authIndex = normalizeAuthIndex(item['auth_index'] ?? item.authIndex);
      useQuotaStore.getState().setKiroQuota((prev) => ({
        ...prev,
        [item.name]: {
          ...current,
          overageUpdating: true,
        } as KiroQuotaState,
      }));

      try {
        const quota = await authFilesApi.setKiroOverage({
          name: item.name,
          authIndex: authIndex ?? undefined,
          enabled,
        });
        useQuotaStore.getState().setKiroQuota((prev) => ({
          ...prev,
          [item.name]: buildKiroQuotaStateFromProvider(item, quota),
        }));
        toast.success(
          t(enabled ? 'kiro_quota.overage_enable_success' : 'kiro_quota.overage_disable_success', {
            name: item.name,
          })
        );
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : t('common.unknown_error');
        useQuotaStore.getState().setKiroQuota((prev) => ({
          ...prev,
          [item.name]: {
            ...(prev[item.name] ?? current),
            overageUpdating: false,
          } as KiroQuotaState,
        }));
        toast.error(t('kiro_quota.overage_toggle_failed', { name: item.name, message }));
      }
    },
    [disabled, t]
  );

  if (allItems.length === 0) {
    return (
      <EmptyState
        title={t('quota_management.title')}
        description={t('quota_management.description')}
      />
    );
  }

  return (
    <div className="flex flex-col gap-3">
      <div ref={gridRef} className={styles.quotaGrid}>
        {pageItems.map(({ item, config }) => (
          <QuotaCard
            key={item.name}
            item={item}
            quota={config.storeSelector(allQuota)[item.name]}
            resolvedTheme={resolvedTheme}
            i18nPrefix={config.i18nPrefix}
            cardIdleMessageKey={config.cardIdleMessageKey}
            cardClassName={config.cardClassName}
            defaultType={config.type}
            canRefresh={!disabled && !item.disabled}
            onRefresh={() => void refreshQuotaForItem(item, config)}
            onSetKiroOverage={config.type === 'kiro' ? setKiroOverageForItem : undefined}
            renderQuotaItems={config.renderQuotaItems}
          />
        ))}
      </div>
      {allItems.length > pageSize && effectiveViewMode === 'paged' && (
        <div className={styles.pagination}>
          <Button variant="secondary" size="sm" onClick={goToPrev} disabled={currentPage <= 1}>
            {t('auth_files.pagination_prev')}
          </Button>
          <div className={styles.pageInfo}>
            {t('auth_files.pagination_info', {
              current: currentPage,
              total: totalPages,
              count: allItems.length,
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
    </div>
  );
}
