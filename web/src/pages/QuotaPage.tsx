import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useHeaderRefresh } from '@/hooks/useHeaderRefresh';
import { useAuthStore } from '@/stores';
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
import type { AuthFileItem } from '@/types';
import { quotaStyles as styles } from '@/components/quota/quotaStyles';

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

  const [files, setFiles] = useState<AuthFileItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [activeTab, setActiveTab] = useState('all');

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
      })).filter(({ count }) => count > 0),
    [files]
  );

  return (
    <div className={styles.container}>
      <div className={styles.pageHeader}>
        <h1 className={styles.pageTitle}>{t('quota_management.title')}</h1>
        <p className={styles.description}>{t('quota_management.description')}</p>
      </div>

      {error && <div className={styles.errorBox}>{error}</div>}

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className="flex justify-start items-start gap-0 p-0 border-b border-border bg-transparent">
          <TabsTrigger
            value="all"
            className="min-h-[32px] px-3 py-1 gap-1.5 text-muted-foreground border-b-2 border-transparent -mb-px data-[state=active]:border-primary data-[state=active]:text-primary hover:text-foreground"
          >
            {t('quota_management.tab_all')}
            {allCount > 0 && <span className={styles.tabCountBadge}>{allCount}</span>}
          </TabsTrigger>
          {providerTabs.map(({ config, count }) => (
            <TabsTrigger
              key={config.type}
              value={config.type}
              className="min-h-[32px] px-3 py-1 gap-1.5 text-muted-foreground border-b-2 border-transparent -mb-px data-[state=active]:border-primary data-[state=active]:text-primary hover:text-foreground"
            >
              {t(`${config.i18nPrefix}.title`)}
              <span className={styles.tabCountBadge}>{count}</span>
            </TabsTrigger>
          ))}
        </TabsList>

        <TabsContent value="all">
          <AllQuotaSection
            configs={ALL_CONFIGS}
            files={files}
            loading={loading}
            disabled={disableControls}
          />
        </TabsContent>

        {providerTabs.map(({ config }) => (
          <TabsContent key={config.type} value={config.type}>
            <QuotaSection
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              config={config as any}
              files={files}
              loading={loading}
              disabled={disableControls}
            />
          </TabsContent>
        ))}
      </Tabs>
    </div>
  );
}
