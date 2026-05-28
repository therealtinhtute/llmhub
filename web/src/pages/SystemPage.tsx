import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { AppCard as Card } from '@/components/ui/AppCard';
import { Button } from '@/components/ui/Button';
import { Modal } from '@/components/ui/Modal';
import { ToggleSwitch } from '@/components/ui/ToggleSwitch';
import { IconGithub, IconBookOpen, IconExternalLink, IconCode } from '@/components/ui/icons';
import { toast } from 'sonner';
import {
  useAuthStore,
  useConfigStore,
  useNotificationStore,
  useModelsStore,
  useThemeStore,
} from '@/stores';
import { configApi, versionApi } from '@/services/api';
import { apiKeysApi } from '@/services/api/apiKeys';
import { classifyModels } from '@/utils/models';
import { STORAGE_KEY_AUTH } from '@/utils/constants';
import { INLINE_LOGO_JPEG } from '@/assets/logoInline';
import iconGemini from '@/assets/icons/gemini.svg';
import iconClaude from '@/assets/icons/claude.svg';
import iconOpenaiLight from '@/assets/icons/openai-light.svg';
import iconOpenaiDark from '@/assets/icons/openai-dark.svg';
import iconQwen from '@/assets/icons/qwen.svg';
import iconKimiLight from '@/assets/icons/kimi-light.svg';
import iconKimiDark from '@/assets/icons/kimi-dark.svg';
import iconGlm from '@/assets/icons/glm.svg';
import iconGrok from '@/assets/icons/grok.svg';
import iconGrokDark from '@/assets/icons/grok-dark.svg';
import iconDeepseek from '@/assets/icons/deepseek.svg';
import iconMinimax from '@/assets/icons/minimax.svg';

const MODEL_CATEGORY_ICONS: Record<string, string | { light: string; dark: string }> = {
  gpt: { light: iconOpenaiLight, dark: iconOpenaiDark },
  claude: iconClaude,
  gemini: iconGemini,
  qwen: iconQwen,
  kimi: { light: iconKimiLight, dark: iconKimiDark },
  glm: iconGlm,
  grok: { light: iconGrok, dark: iconGrokDark },
  deepseek: iconDeepseek,
  minimax: iconMinimax,
};

const linkIconClass = (type: 'github' | 'docs' | 'primary') =>
  `flex items-center justify-center w-11 h-11 shrink-0 text-white ${type === 'github' ? 'bg-[#24292f]' : type === 'docs' ? 'bg-[#10b981]' : 'bg-primary'}`;

const parseVersionSegments = (version?: string | null) => {
  if (!version) return null;
  const cleaned = version.trim().replace(/^v/i, '');
  if (!cleaned) return null;
  const parts = cleaned
    .split(/[^0-9]+/)
    .filter(Boolean)
    .map((segment) => Number.parseInt(segment, 10))
    .filter(Number.isFinite);
  return parts.length ? parts : null;
};

const compareVersions = (latest?: string | null, current?: string | null) => {
  const latestParts = parseVersionSegments(latest);
  const currentParts = parseVersionSegments(current);
  if (!latestParts || !currentParts) return null;
  const length = Math.max(latestParts.length, currentParts.length);
  for (let i = 0; i < length; i++) {
    const l = latestParts[i] || 0;
    const c = currentParts[i] || 0;
    if (l > c) return 1;
    if (l < c) return -1;
  }
  return 0;
};

export function SystemPage() {
  const { t, i18n } = useTranslation();
  const { showConfirmation } = useNotificationStore();
  const resolvedTheme = useThemeStore((state) => state.resolvedTheme);
  const auth = useAuthStore();
  const config = useConfigStore((state) => state.config);
  const fetchConfig = useConfigStore((state) => state.fetchConfig);
  const clearCache = useConfigStore((state) => state.clearCache);
  const updateConfigValue = useConfigStore((state) => state.updateConfigValue);

  const models = useModelsStore((state) => state.models);
  const modelsLoading = useModelsStore((state) => state.loading);
  const modelsError = useModelsStore((state) => state.error);
  const fetchModelsFromStore = useModelsStore((state) => state.fetchModels);

  const [modelStatus, setModelStatus] = useState<{
    type: 'success' | 'warning' | 'error' | 'muted';
    message: string;
  }>();
  const [requestLogModalOpen, setRequestLogModalOpen] = useState(false);
  const [requestLogDraft, setRequestLogDraft] = useState(false);
  const [requestLogTouched, setRequestLogTouched] = useState(false);
  const [requestLogSaving, setRequestLogSaving] = useState(false);
  const [checkingVersion, setCheckingVersion] = useState(false);

  const apiKeysCache = useRef<string[]>([]);
  const versionTapCount = useRef(0);
  const versionTapTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const otherLabel = useMemo(
    () => (i18n.language?.toLowerCase().startsWith('zh') ? '其他' : 'Other'),
    [i18n.language]
  );
  const groupedModels = useMemo(() => classifyModels(models, { otherLabel }), [models, otherLabel]);
  const requestLogEnabled = config?.requestLog ?? false;
  const requestLogDirty = requestLogDraft !== requestLogEnabled;
  const canEditRequestLog = auth.connectionStatus === 'connected' && Boolean(config);

  const appVersion = __APP_VERSION__ || t('system_info.version_unknown');
  const apiVersion = auth.serverVersion || t('system_info.version_unknown');
  const buildTime = auth.serverBuildDate
    ? new Date(auth.serverBuildDate).toLocaleString(i18n.language)
    : t('system_info.version_unknown');

  const getIconForCategory = (categoryId: string): string | null => {
    const iconEntry = MODEL_CATEGORY_ICONS[categoryId];
    if (!iconEntry) return null;
    if (typeof iconEntry === 'string') return iconEntry;
    return resolvedTheme === 'dark' ? iconEntry.dark : iconEntry.light;
  };

  const normalizeApiKeyList = (input: unknown): string[] => {
    if (!Array.isArray(input)) return [];
    const seen = new Set<string>();
    const keys: string[] = [];

    input.forEach((item) => {
      const record =
        item !== null && typeof item === 'object' && !Array.isArray(item)
          ? (item as Record<string, unknown>)
          : null;
      const value =
        typeof item === 'string'
          ? item
          : record
            ? (record['api-key'] ?? record['apiKey'] ?? record.key ?? record.Key)
            : '';
      const trimmed = String(value ?? '').trim();
      if (!trimmed || seen.has(trimmed)) return;
      seen.add(trimmed);
      keys.push(trimmed);
    });

    return keys;
  };

  const resolveApiKeysForModels = useCallback(async () => {
    if (apiKeysCache.current.length) {
      return apiKeysCache.current;
    }

    const configKeys = normalizeApiKeyList(config?.apiKeys);
    if (configKeys.length) {
      apiKeysCache.current = configKeys;
      return configKeys;
    }

    try {
      const list = await apiKeysApi.list();
      const normalized = normalizeApiKeyList(list);
      if (normalized.length) {
        apiKeysCache.current = normalized;
      }
      return normalized;
    } catch (err) {
      console.warn('Auto loading API keys for models failed:', err);
      return [];
    }
  }, [config?.apiKeys]);

  const fetchModels = async ({ forceRefresh = false }: { forceRefresh?: boolean } = {}) => {
    if (auth.connectionStatus !== 'connected') {
      setModelStatus({
        type: 'warning',
        message: t('notification.connection_required'),
      });
      return;
    }

    if (!auth.apiBase) {
      toast.warning(t('notification.connection_required'));
      return;
    }

    if (forceRefresh) {
      apiKeysCache.current = [];
    }

    setModelStatus({ type: 'muted', message: t('system_info.models_loading') });
    try {
      const apiKeys = await resolveApiKeysForModels();
      const primaryKey = apiKeys[0];
      const list = await fetchModelsFromStore(auth.apiBase, primaryKey, forceRefresh);
      const hasModels = list.length > 0;
      setModelStatus({
        type: hasModels ? 'success' : 'warning',
        message: hasModels
          ? t('system_info.models_count', { count: list.length })
          : t('system_info.models_empty'),
      });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : typeof err === 'string' ? err : '';
      const suffix = message ? `: ${message}` : '';
      const text = `${t('system_info.models_error')}${suffix}`;
      setModelStatus({ type: 'error', message: text });
    }
  };

  const handleClearLoginStorage = () => {
    showConfirmation({
      title: t('system_info.clear_login_title', { defaultValue: 'Clear Login Storage' }),
      message: t('system_info.clear_login_confirm'),
      variant: 'danger',
      confirmText: t('common.confirm'),
      onConfirm: () => {
        auth.logout();
        if (typeof localStorage === 'undefined') return;
        const keysToRemove = [STORAGE_KEY_AUTH, 'isLoggedIn', 'apiBase', 'apiUrl', 'managementKey'];
        keysToRemove.forEach((key) => localStorage.removeItem(key));
        toast.success(t('notification.login_storage_cleared'));
      },
    });
  };

  const openRequestLogModal = useCallback(() => {
    setRequestLogTouched(false);
    setRequestLogDraft(requestLogEnabled);
    setRequestLogModalOpen(true);
  }, [requestLogEnabled]);

  const handleInfoVersionTap = useCallback(() => {
    versionTapCount.current += 1;
    if (versionTapTimer.current) {
      clearTimeout(versionTapTimer.current);
    }

    if (versionTapCount.current >= 7) {
      versionTapCount.current = 0;
      versionTapTimer.current = null;
      openRequestLogModal();
      return;
    }

    versionTapTimer.current = setTimeout(() => {
      versionTapCount.current = 0;
      versionTapTimer.current = null;
    }, 1500);
  }, [openRequestLogModal]);

  const handleRequestLogClose = useCallback(() => {
    setRequestLogModalOpen(false);
    setRequestLogTouched(false);
  }, []);

  const handleRequestLogSave = async () => {
    if (!canEditRequestLog) return;
    if (!requestLogDirty) {
      setRequestLogModalOpen(false);
      return;
    }

    const previous = requestLogEnabled;
    setRequestLogSaving(true);
    updateConfigValue('request-log', requestLogDraft);

    try {
      await configApi.updateRequestLog(requestLogDraft);
      clearCache('request-log');
      toast.success(t('notification.request_log_updated'));
      setRequestLogModalOpen(false);
    } catch (error: unknown) {
      const message =
        error instanceof Error ? error.message : typeof error === 'string' ? error : '';
      updateConfigValue('request-log', previous);
      toast.error(
        `${t('notification.update_failed')}${message ? `: ${message}` : ''}`
      );
    } finally {
      setRequestLogSaving(false);
    }
  };

  const handleVersionCheck = useCallback(async () => {
    setCheckingVersion(true);
    try {
      const data = await versionApi.checkLatest();
      const latestRaw = data?.['latest-version'] ?? data?.latest_version ?? data?.latest ?? '';
      const latest = typeof latestRaw === 'string' ? latestRaw : String(latestRaw ?? '');
      const comparison = compareVersions(latest, auth.serverVersion);

      if (!latest) {
        toast.error(t('system_info.version_check_error'));
        return;
      }

      if (comparison === null) {
        toast.warning(t('system_info.version_current_missing'));
        return;
      }

      if (comparison > 0) {
        toast.warning(t('system_info.version_update_available', { version: latest }));
      } else {
        toast.success(t('system_info.version_is_latest'));
      }
    } catch (error: unknown) {
      const message =
        error instanceof Error ? error.message : typeof error === 'string' ? error : '';
      const suffix = message ? `: ${message}` : '';
      toast.error(`${t('system_info.version_check_error')}${suffix}`);
    } finally {
      setCheckingVersion(false);
    }
  }, [auth.serverVersion, t]);

  useEffect(() => {
    fetchConfig().catch(() => {
      // ignore
    });
  }, [fetchConfig]);

  useEffect(() => {
    if (requestLogModalOpen && !requestLogTouched) {
      setRequestLogDraft(requestLogEnabled);
    }
  }, [requestLogModalOpen, requestLogTouched, requestLogEnabled]);

  useEffect(() => {
    return () => {
      if (versionTapTimer.current) {
        clearTimeout(versionTapTimer.current);
      }
    };
  }, []);

  useEffect(() => {
    fetchModels();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [auth.connectionStatus, auth.apiBase]);

  return (
    <div className="w-full">
      <h1 className="text-[28px] font-bold text-foreground mb-6">{t('system_info.title')}</h1>
      <div className="flex flex-col gap-6">
        <Card className="overflow-hidden">
          <div className="flex flex-col items-center justify-center w-full gap-3 py-4 px-0 pb-6">
            <img src={INLINE_LOGO_JPEG} alt="CPAMC" className="w-[108px] h-[108px] object-cover shadow-[0_12px_32px_rgba(0,0,0,0.16)]" />
            <div className="w-[min(100%,920px)] text-[clamp(28px,4.2vw,44px)] font-extrabold leading-tight text-foreground tracking-tight text-center text-balance break-words">{t('system_info.about_title')}</div>
          </div>

          <div className="grid grid-cols-2 gap-3 max-[900px]:grid-cols-1">
            <button
              type="button"
              className="flex flex-col gap-1.5 min-h-[120px] px-4 py-3 border border-border bg-muted/82 text-left cursor-pointer"
              onClick={handleInfoVersionTap}
            >
              <div className="flex items-start justify-between gap-2 min-h-[40px]">
                <div className="text-[13px] font-semibold text-muted-foreground">{t('footer.version')}</div>
              </div>
              <div className="text-[22px] font-bold text-foreground leading-tight break-words">{appVersion}</div>
            </button>

            <div className="flex flex-col gap-1.5 min-h-[120px] px-4 py-3 border border-border bg-muted/82 text-left">
              <div className="flex items-start justify-between gap-2 min-h-[40px]">
                <div className="text-[13px] font-semibold text-muted-foreground">{t('footer.api_version')}</div>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="shrink-0 whitespace-nowrap -mt-1 -mr-2"
                  onClick={() => void handleVersionCheck()}
                  loading={checkingVersion}
                  title={t('system_info.version_check_button')}
                  aria-label={t('system_info.version_check_button')}
                >
                  {t('system_info.version_check_button')}
                </Button>
              </div>
              <div className="text-[22px] font-bold text-foreground leading-tight break-words">{apiVersion}</div>
            </div>

            <div className="flex flex-col gap-1.5 min-h-[120px] px-4 py-3 border border-border bg-muted/82 text-left">
              <div className="text-[13px] font-semibold text-muted-foreground">{t('footer.build_date')}</div>
              <div className="text-[22px] font-bold text-foreground leading-tight break-words">{buildTime}</div>
            </div>

            <div className="flex flex-col gap-1.5 min-h-[120px] px-4 py-3 border border-border bg-muted/82 text-left">
              <div className="text-[13px] font-semibold text-muted-foreground">{t('connection.status')}</div>
              <div className="text-[22px] font-bold text-foreground leading-tight break-words">{t(`common.${auth.connectionStatus}_status`)}</div>
              <div className="text-xs text-muted-foreground/70 leading-snug">{auth.apiBase || '-'}</div>
            </div>
          </div>
        </Card>

        <Card title={t('system_info.quick_links_title')}>
          <p className="text-sm text-muted-foreground mb-3">{t('system_info.quick_links_desc')}</p>
          <div className="grid grid-cols-[repeat(auto-fit,minmax(280px,1fr))] gap-3">
            <a
              href="https://github.com/therealtinhtute/llmhub"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-3 px-4 py-3 bg-muted border border-border no-underline text-inherit hover:bg-accent hover:border-primary"
            >
              <div className={linkIconClass('github')}>
                <IconGithub size={22} />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-1 text-[15px] font-semibold text-foreground mb-0.5">
                  {t('system_info.link_main_repo')}
                  <IconExternalLink size={14} />
                </div>
                <div className="text-[13px] text-muted-foreground whitespace-nowrap overflow-hidden text-ellipsis">{t('system_info.link_main_repo_desc')}</div>
              </div>
            </a>

            <a
              href="https://github.com/therealtinhtute/llmhub"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-3 px-4 py-3 bg-muted border border-border no-underline text-inherit hover:bg-accent hover:border-primary"
            >
              <div className={linkIconClass('github')}>
                <IconCode size={22} />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-1 text-[15px] font-semibold text-foreground mb-0.5">
                  {t('system_info.link_webui_repo')}
                  <IconExternalLink size={14} />
                </div>
                <div className="text-[13px] text-muted-foreground whitespace-nowrap overflow-hidden text-ellipsis">{t('system_info.link_webui_repo_desc')}</div>
              </div>
            </a>

            <a
              href="https://github.com/therealtinhtute/llmhub"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-3 px-4 py-3 bg-muted border border-border no-underline text-inherit hover:bg-accent hover:border-primary"
            >
              <div className={linkIconClass('docs')}>
                <IconBookOpen size={22} />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-1 text-[15px] font-semibold text-foreground mb-0.5">
                  {t('system_info.link_docs')}
                  <IconExternalLink size={14} />
                </div>
                <div className="text-[13px] text-muted-foreground whitespace-nowrap overflow-hidden text-ellipsis">{t('system_info.link_docs_desc')}</div>
              </div>
            </a>
          </div>
        </Card>

        <Card
          title={t('system_info.models_title')}
          extra={
            <Button
              variant="secondary"
              size="sm"
              onClick={() => fetchModels({ forceRefresh: true })}
              loading={modelsLoading}
            >
              {t('common.refresh')}
            </Button>
          }
        >
          <p className="text-sm text-muted-foreground mb-3">{t('system_info.models_desc')}</p>
          {modelStatus && (
            <div className={`inline-flex items-center text-[0.8125rem] font-medium px-[10px] py-[2px] border leading-[1.5] rounded-sm ${modelStatus.type === 'success' ? 'text-emerald-700 bg-emerald-100 border-emerald-400/40' : modelStatus.type === 'error' ? 'text-destructive bg-destructive/10 border-destructive/30' : modelStatus.type === 'warning' ? 'text-amber-700 bg-amber-100 border-amber-400/30' : 'border-border text-muted-foreground bg-muted'}`}>{modelStatus.message}</div>
          )}
          {modelsError && <div className="p-[10px_14px] mb-2 bg-destructive/10 border border-destructive/35 text-destructive text-sm leading-[1.5]">{modelsError}</div>}
          {modelsLoading ? (
            <div className="text-[13px] text-muted-foreground leading-[1.55]">{t('common.loading')}</div>
          ) : models.length === 0 ? (
            <div className="text-[13px] text-muted-foreground leading-[1.55]">{t('system_info.models_empty')}</div>
          ) : (
            <div className="flex flex-col">
              {groupedModels.map((group) => {
                const iconSrc = getIconForCategory(group.id);
                return (
                  <div key={group.id} className="flex items-center justify-between gap-2 py-2.5 border-b border-border last:border-b-0">
                    <div className="flex flex-col gap-0.5 min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        {iconSrc && <img src={iconSrc} alt="" className="w-[18px] h-[18px] shrink-0" />}
                        <span className="text-sm font-medium text-foreground">{group.label}</span>
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {t('system_info.models_count', { count: group.items.length })}
                      </div>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      {group.items.map((model) => (
                        <span
                          key={`${model.name}-${model.alias ?? 'default'}`}
                          className="inline-flex items-center gap-1 px-2.5 py-1 border border-border bg-muted font-mono text-sm"
                          title={model.description || ''}
                        >
                          <span className="text-foreground font-semibold">{model.name}</span>
                          {model.alias && <span className="text-muted-foreground text-xs">{model.alias}</span>}
                        </span>
                      ))}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </Card>

        <Card title={t('system_info.clear_login_title')}>
          <p className="text-sm text-muted-foreground mb-3">{t('system_info.clear_login_desc')}</p>
          <div className="flex justify-end items-center">
            <Button variant="danger" onClick={handleClearLoginStorage}>
              {t('system_info.clear_login_button')}
            </Button>
          </div>
        </Card>
      </div>

      <Modal
        open={requestLogModalOpen}
        onClose={handleRequestLogClose}
        title={t('basic_settings.request_log_title')}
        footer={
          <>
            <Button variant="secondary" onClick={handleRequestLogClose} disabled={requestLogSaving}>
              {t('common.cancel')}
            </Button>
            <Button
              onClick={handleRequestLogSave}
              loading={requestLogSaving}
              disabled={!canEditRequestLog || !requestLogDirty}
            >
              {t('common.save')}
            </Button>
          </>
        }
      >
        <div className="request-log-modal">
          <div className="inline-flex items-center text-[0.8125rem] font-medium px-[10px] py-[2px] border rounded-sm text-amber-700 bg-amber-100 border-amber-400/30 leading-[1.5]">{t('basic_settings.request_log_warning')}</div>
          <ToggleSwitch
            label={t('basic_settings.request_log_enable')}
            labelPosition="left"
            checked={requestLogDraft}
            disabled={!canEditRequestLog || requestLogSaving}
            onChange={(value) => {
              setRequestLogDraft(value);
              setRequestLogTouched(true);
            }}
          />
        </div>
      </Modal>
    </div>
  );
}
