import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useHeaderRefresh } from '@/hooks/useHeaderRefresh';
import { useAuthStore, useThemeStore } from '@/stores';
import { authFilesApi, configFileApi } from '@/services/api';
import {
  AllQuotaSection,
  QuotaSection,
  ANTIGRAVITY_CONFIG,
  CLAUDE_CONFIG,
  CODEX_CONFIG,
  GEMINI_CLI_CONFIG,
  KIRO_CONFIG,
  KIMI_CONFIG,
  XAI_CONFIG,
} from '@/components/quota';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { Button } from '@/components/ui/Button';
import { IconRefreshCw, IconFilterAll } from '@/components/ui/icons';
import { getAuthFileIcon } from '@/features/authFiles/constants';
import type { AuthFileItem, ResolvedTheme } from '@/types';
import { quotaStyles as styles } from '@/components/quota/quotaStyles';

type ViewMode = 'paged' | 'all';

const ALL_CONFIGS = [
  CLAUDE_CONFIG,
  ANTIGRAVITY_CONFIG,
  CODEX_CONFIG,
  XAI_CONFIG,
  GEMINI_CLI_CONFIG,
  KIRO_CONFIG,
  KIMI_CONFIG,
];

export function QuotaPage() {
  const { t } = useTranslation();
  const connectionStatus = useAuthStore((state) => state.connectionStatus);
  const resolvedTheme: ResolvedTheme = useThemeStore((state) => state.resolvedTheme);

  const [files, setFiles] = useState<AuthFileItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [activeTab, setActiveTab] = useState('all');
  const [viewMode, setViewMode] = useState<ViewMode>('paged');
  const [refreshSignal, setRefreshSignal] = useState(0);

  const disableControls = connectionStatus !== 'connected';

  const loadConfig = useCallback(async () => {
    try {
      await configFileApi.fetchConfigYaml();
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : t('notification.refresh_failed');
      setError((prev) => prev || errorMessage);
    }
  }, [t]);

  const loadFiles = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await authFilesApi.list();
      setFiles(data?.files || []);
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : t('notification.refresh_failed');
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  }, [t]);

  const handleHeaderRefresh = useCallback(async () => {
    await Promise.all([loadConfig(), loadFiles()]);
  }, [loadConfig, loadFiles]);

  const handleTabRefresh = useCallback(async () => {
    setRefreshSignal((prev) => prev + 1);
    await handleHeaderRefresh();
  }, [handleHeaderRefresh]);

  useHeaderRefresh(handleHeaderRefresh);

  useEffect(() => {
    loadFiles();
    loadConfig();
  }, [loadFiles, loadConfig]);

  const allCount = useMemo(
    () =>
      ALL_CONFIGS.reduce((acc, cfg) => {
        files.forEach((f) => {
          if (cfg.filterFn(f)) acc.add(f.name);
        });
        return acc;
      }, new Set<string>()).size,
    [files]
  );

  const providerTabs = useMemo(
    () =>
      ALL_CONFIGS.map((config) => ({
        config,
        count: files.filter(config.filterFn).length,
        iconSrc: getAuthFileIcon(config.type, resolvedTheme),
      })).filter(({ count }) => count > 0),
    [files, resolvedTheme]
  );

  return (
    <div className={styles.container}>
      <div className={styles.pageHeader}>
        <h1 className={styles.pageTitle}>{t('quota_management.title')}</h1>
        <p className={styles.description}>{t('quota_management.description')}</p>
      </div>

      <div className="rounded-md border border-border bg-muted/40 p-3 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div className="text-sm font-semibold text-foreground">
            {t('quota_management.monitoring_title')}
          </div>
          <div className="text-xs text-muted-foreground">
            {t('quota_management.monitoring_desc')}
          </div>
        </div>
        <Button asChild variant="secondary" size="sm" className="w-fit">
          <Link to="/quota/monitoring">{t('quota_management.monitoring_open')}</Link>
        </Button>
      </div>

      {error && <div className={styles.errorBox}>{error}</div>}

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <div className="flex items-end gap-2 border-b border-border">
          <TabsList className="flex justify-start items-start gap-0 p-0 bg-transparent overflow-x-auto flex-1 min-w-0">
            <TabsTrigger
              value="all"
              className="min-h-[32px] px-3 py-1 gap-1.5 text-muted-foreground border-b-2 border-transparent -mb-px data-[state=active]:border-primary data-[state=active]:text-primary hover:text-foreground"
            >
              <span className="inline-flex items-center gap-1.5">
                <IconFilterAll data-icon="inline-start" />
                {t('quota_management.tab_all')}
              </span>
              {allCount > 0 && <span className={styles.tabCountBadge}>{allCount}</span>}
            </TabsTrigger>
            {providerTabs.map(({ config, count, iconSrc }) => (
              <TabsTrigger
                key={config.type}
                value={config.type}
                className="min-h-[32px] px-3 py-1 gap-1.5 text-muted-foreground border-b-2 border-transparent -mb-px data-[state=active]:border-primary data-[state=active]:text-primary hover:text-foreground"
              >
                <span className="inline-flex items-center gap-1.5">
                  {iconSrc ? (
                    <img src={iconSrc} alt="" className="size-4 shrink-0 object-contain" />
                  ) : (
                    <span className="inline-flex size-4 items-center justify-center text-[11px] font-bold leading-none">
                      {t(`${config.i18nPrefix}.title`).slice(0, 1).toUpperCase()}
                    </span>
                  )}
                  <span className="truncate">{t(`${config.i18nPrefix}.title`)}</span>
                </span>
                <span className={styles.tabCountBadge}>{count}</span>
              </TabsTrigger>
            ))}
          </TabsList>

          <div className="flex items-center gap-1 shrink-0 pb-px mb-1">
            <div className={styles.viewModeToggle}>
              <Button
                variant="secondary"
                size="sm"
                className={`${styles.viewModeButton} ${viewMode === 'paged' ? styles.viewModeButtonActive : ''}`}
                onClick={() => setViewMode('paged')}
              >
                {t('auth_files.view_mode_paged')}
              </Button>
              <Button
                variant="secondary"
                size="sm"
                className={`${styles.viewModeButton} ${viewMode === 'all' ? styles.viewModeButtonActive : ''}`}
                onClick={() => setViewMode('all')}
              >
                {t('auth_files.view_mode_all')}
              </Button>
            </div>
            <button
              type="button"
              className="p-1.5 text-muted-foreground hover:text-foreground hover:bg-muted rounded disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
              onClick={() => void handleTabRefresh()}
              disabled={loading}
              title={t('quota_management.refresh_all_credentials')}
              aria-label={t('quota_management.refresh_all_credentials')}
            >
              <IconRefreshCw size={13} className={loading ? 'animate-spin' : ''} />
            </button>
          </div>
        </div>

        <TabsContent value="all" className="mt-3">
          <AllQuotaSection
            configs={ALL_CONFIGS}
            files={files}
            loading={loading}
            disabled={disableControls}
            viewMode={viewMode}
            onViewModeChange={setViewMode}
            refreshSignal={refreshSignal}
          />
        </TabsContent>

        {providerTabs.map(({ config }) => (
          <TabsContent key={config.type} value={config.type} className="mt-3">
            <QuotaSection
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              config={config as any}
              files={files}
              loading={loading}
              disabled={disableControls}
              viewMode={viewMode}
              onViewModeChange={setViewMode}
              refreshSignal={refreshSignal}
            />
          </TabsContent>
        ))}
      </Tabs>
    </div>
  );
}
