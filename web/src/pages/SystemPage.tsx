import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { AppCard as Card } from '@/components/ui/AppCard';
import { Button } from '@/components/ui/Button';
import { Modal } from '@/components/ui/Modal';
import { ToggleSwitch } from '@/components/ui/ToggleSwitch';
import { FormInput as Input } from '@/components/ui/FormInput';
import { IconGithub, IconBookOpen, IconExternalLink, IconCode } from '@/components/ui/icons';
import { toast } from 'sonner';
import {
  useAuthStore,
  useConfigStore,
  useConfirmationStore,
} from '@/stores';
import { configApi, runtimeControlsApi, versionApi } from '@/services/api';
import type { RuntimeControlSettings } from '@/types';
import { STORAGE_KEY_AUTH } from '@/utils/constants';
import { INLINE_LOGO_JPEG } from '@/assets/logoInline';

const linkIconClass = (type: 'github' | 'docs' | 'primary') =>
  `flex items-center justify-center w-11 h-11 shrink-0 text-white ${type === 'github' ? 'bg-[#24292f]' : type === 'docs' ? 'bg-success' : 'bg-primary'}`;

const runtimeInputNumber = (value: string, fallback: number): number => {
  const parsed = Number.parseInt(value.trim(), 10);
  return Number.isFinite(parsed) ? parsed : fallback;
};

const updateRuntimeDraft = (
  settings: RuntimeControlSettings | null,
  update: (draft: RuntimeControlSettings) => void
): RuntimeControlSettings | null => {
  if (!settings) return settings;
  const next: RuntimeControlSettings = {
    ...settings,
    credential_routing: {
      ...settings.credential_routing,
      weights: [...(settings.credential_routing?.weights ?? [])],
    },
    cloaking: { ...settings.cloaking },
    codex_live: {
      ...settings.codex_live,
      ice_servers: [...(settings.codex_live?.ice_servers ?? [])],
    },
    home: { ...settings.home },
  };
  update(next);
  return next;
};

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
  const { showConfirmation } = useConfirmationStore();
  const auth = useAuthStore();
  const config = useConfigStore((state) => state.config);
  const fetchConfig = useConfigStore((state) => state.fetchConfig);
  const clearCache = useConfigStore((state) => state.clearCache);
  const updateConfigValue = useConfigStore((state) => state.updateConfigValue);

  const [requestLogModalOpen, setRequestLogModalOpen] = useState(false);
  const [requestLogDraft, setRequestLogDraft] = useState(false);
  const [requestLogTouched, setRequestLogTouched] = useState(false);
  const [requestLogSaving, setRequestLogSaving] = useState(false);
  const [checkingVersion, setCheckingVersion] = useState(false);
  const [availableVersion, setAvailableVersion] = useState<string | null>(null);
  const [stagingUpdate, setStagingUpdate] = useState(false);
  const [stagedVersion, setStagedVersion] = useState<string | null>(null);
  const [applyingUpdate, setApplyingUpdate] = useState(false);
  const [runtimeDraft, setRuntimeDraft] = useState<RuntimeControlSettings | null>(null);
  const [runtimeLoading, setRuntimeLoading] = useState(false);
  const [runtimeSaving, setRuntimeSaving] = useState(false);
  const [runtimeError, setRuntimeError] = useState<string | null>(null);
  const [codexLiveMaxSessionsText, setCodexLiveMaxSessionsText] = useState('');
  const [codexLivePublicIPText, setCodexLivePublicIPText] = useState('');
  const [codexLiveUDPMinText, setCodexLiveUDPMinText] = useState('');
  const [codexLiveUDPMaxText, setCodexLiveUDPMaxText] = useState('');

  const versionTapCount = useRef(0);
  const versionTapTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const requestLogEnabled = config?.requestLog ?? false;
  const requestLogDirty = requestLogDraft !== requestLogEnabled;
  const canEditRequestLog = auth.connectionStatus === 'connected' && Boolean(config);

  const appVersion = __APP_VERSION__ || t('system_info.version_unknown');
  const apiVersion = auth.serverVersion || t('system_info.version_unknown');
  const buildTime = auth.serverBuildDate
    ? new Date(auth.serverBuildDate).toLocaleString(i18n.language)
    : t('system_info.version_unknown');

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

  const syncRuntimeText = useCallback((settings: RuntimeControlSettings) => {
    setCodexLiveMaxSessionsText(String(settings.codex_live?.max_sessions ?? 32));
    setCodexLivePublicIPText(settings.codex_live?.public_ip ?? '');
    setCodexLiveUDPMinText(String(settings.codex_live?.udp_port_min ?? 0));
    setCodexLiveUDPMaxText(String(settings.codex_live?.udp_port_max ?? 0));
  }, []);

  const loadRuntimeControls = useCallback(async () => {
    setRuntimeLoading(true);
    setRuntimeError(null);
    try {
      const settings = await runtimeControlsApi.get();
      setRuntimeDraft(settings);
      syncRuntimeText(settings);
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : typeof error === 'string' ? error : '';
      setRuntimeError(message || t('system_info.runtime_controls_load_failed'));
    } finally {
      setRuntimeLoading(false);
    }
  }, [syncRuntimeText, t]);

  const handleRuntimeSave = async () => {
    if (!runtimeDraft) return;
    const next = updateRuntimeDraft(runtimeDraft, (draft) => {
      draft.codex_live.max_sessions = runtimeInputNumber(codexLiveMaxSessionsText, draft.codex_live.max_sessions);
      draft.codex_live.public_ip = codexLivePublicIPText.trim() || undefined;
      draft.codex_live.udp_port_min = runtimeInputNumber(codexLiveUDPMinText, draft.codex_live.udp_port_min);
      draft.codex_live.udp_port_max = runtimeInputNumber(codexLiveUDPMaxText, draft.codex_live.udp_port_max);
    });
    if (!next) return;
    setRuntimeSaving(true);
    setRuntimeError(null);
    try {
      const saved = await runtimeControlsApi.save(next);
      setRuntimeDraft(saved);
      syncRuntimeText(saved);
      toast.success(t('system_info.runtime_controls_saved'));
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : typeof error === 'string' ? error : '';
      toast.error(`${t('notification.update_failed')}${message ? `: ${message}` : ''}`);
    } finally {
      setRuntimeSaving(false);
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
        setAvailableVersion(latest);
        toast.warning(t('system_info.version_update_available', { version: latest }));
      } else {
        setAvailableVersion(null);
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

  const handleStageUpdate = useCallback(async () => {
    setStagingUpdate(true);
    try {
      const data = await versionApi.stageUpdate();
      const status = data?.status;
      if (status === 'staged') {
        const version = typeof data?.version === 'string' ? data.version : '';
        setStagedVersion(version);
        toast.success(t('system_info.update_staged', { version }));
      } else if (status === 'current') {
        setAvailableVersion(null);
        toast.success(t('system_info.version_is_latest'));
      } else if (status === 'unsupported') {
        toast.error(t('system_info.update_unsupported'));
      } else if (status === 'rejected') {
        const reason = typeof data?.reason === 'string' ? data.reason : '';
        toast.error(t('system_info.update_rejected', { reason }));
      } else {
        toast.error(t('system_info.update_stage_failed'));
      }
    } catch (error: unknown) {
      const message =
        error instanceof Error ? error.message : typeof error === 'string' ? error : '';
      const suffix = message ? `: ${message}` : '';
      toast.error(`${t('system_info.update_stage_failed')}${suffix}`);
    } finally {
      setStagingUpdate(false);
    }
  }, [t]);

  const handleApplyUpdate = useCallback(() => {
    showConfirmation({
      title: t('system_info.update_apply_button'),
      message: t('system_info.update_restart_confirm'),
      variant: 'danger',
      confirmText: t('common.confirm'),
      onConfirm: () => {
        void (async () => {
          setApplyingUpdate(true);
          try {
            await versionApi.applyUpdate();
            // The server terminates itself; poll until it returns on the
            // staged version. Each response refreshes the store version via
            // the server-version-update event.
            const target = stagedVersion;
            const started = Date.now();
            const POLL_INTERVAL_MS = 3000;
            const POLL_TIMEOUT_MS = 90000;
            for (;;) {
              await new Promise((resolve) => setTimeout(resolve, POLL_INTERVAL_MS));
              try {
                await versionApi.checkLatest();
              } catch {
                // Server still down; keep polling.
              }
              const current = useAuthStore.getState().serverVersion;
              if (target && current === target) {
                toast.success(t('system_info.update_restarted', { version: target }));
                setAvailableVersion(null);
                setStagedVersion(null);
                return;
              }
              if (Date.now() - started > POLL_TIMEOUT_MS) {
                toast.warning(t('system_info.update_restart_timeout'));
                return;
              }
            }
          } catch (error: unknown) {
            const message =
              error instanceof Error ? error.message : typeof error === 'string' ? error : '';
            const suffix = message ? `: ${message}` : '';
            toast.error(`${t('system_info.update_restart_failed')}${suffix}`);
          } finally {
            setApplyingUpdate(false);
          }
        })();
      },
    });
  }, [showConfirmation, stagedVersion, t]);

  useEffect(() => {
    fetchConfig().catch(() => {
      // ignore
    });
  }, [fetchConfig]);

  useEffect(() => {
    void loadRuntimeControls();
  }, [loadRuntimeControls]);

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

  return (
    <div className="w-full">
      <h1 className="text-[28px] font-bold text-foreground mb-6">{t('system_info.title')}</h1>
      <div className="flex flex-col gap-6">
        <Card className="overflow-hidden">
          <div className="flex flex-col items-center justify-center w-full gap-3 py-4 px-0 pb-6">
            <img src={INLINE_LOGO_JPEG} alt="LLMHUB" className="w-24 h-24 object-contain drop-shadow-none" />
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
              {availableVersion && !stagedVersion && !applyingUpdate && (
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-xs text-muted-foreground">
                    {t('system_info.version_update_available', { version: availableVersion })}
                  </span>
                  <Button
                    type="button"
                    size="sm"
                    className="shrink-0"
                    onClick={() => void handleStageUpdate()}
                    loading={stagingUpdate}
                  >
                    {t('system_info.update_button')}
                  </Button>
                </div>
              )}
              {stagingUpdate && (
                <div className="text-xs text-muted-foreground">{t('system_info.update_staging')}</div>
              )}
              {stagedVersion && (
                <div className="flex flex-col gap-1.5">
                  <div className="text-xs text-foreground">
                    {t('system_info.update_staged', { version: stagedVersion })}
                  </div>
                  <Button
                    type="button"
                    size="sm"
                    variant="danger"
                    className="shrink-0 self-start"
                    onClick={() => void handleApplyUpdate()}
                    loading={applyingUpdate}
                    disabled={applyingUpdate}
                  >
                    {t('system_info.update_apply_button')}
                  </Button>
                </div>
              )}
              {applyingUpdate && (
                <div className="text-xs text-muted-foreground">{t('system_info.update_restarting')}</div>
              )}
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
              className="flex items-center gap-3 px-4 py-3 bg-muted border border-border no-underline text-inherit transition-colors duration-150 hover:bg-accent hover:border-primary"
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
              className="flex items-center gap-3 px-4 py-3 bg-muted border border-border no-underline text-inherit transition-colors duration-150 hover:bg-accent hover:border-primary"
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
              className="flex items-center gap-3 px-4 py-3 bg-muted border border-border no-underline text-inherit transition-colors duration-150 hover:bg-accent hover:border-primary"
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

        <Card title={t('system_info.runtime_controls_title')}>
          <p className="text-sm text-muted-foreground mb-3">{t('system_info.runtime_controls_desc')}</p>
          {runtimeLoading ? (
            <div className="text-sm text-muted-foreground">{t('common.loading')}</div>
          ) : runtimeError ? (
            <div className="flex flex-col gap-3">
              <div className="py-2 px-3 border border-destructive bg-destructive/10 text-destructive text-sm">{runtimeError}</div>
              <div className="flex justify-end">
                <Button variant="secondary" onClick={() => void loadRuntimeControls()}>
                  {t('common.refresh')}
                </Button>
              </div>
            </div>
          ) : runtimeDraft ? (
            <div className="flex flex-col gap-4">
              <div className="grid grid-cols-[repeat(auto-fit,minmax(240px,1fr))] gap-3">
                <label className="flex flex-col gap-1 text-sm text-muted-foreground">
                  <span>{t('system_info.runtime_routing_strategy')}</span>
                  <select
                    className="h-10 px-3 border border-border bg-background text-foreground"
                    value={runtimeDraft.credential_routing.strategy}
                    disabled={runtimeSaving}
                    onChange={(event) =>
                      setRuntimeDraft((current) =>
                        updateRuntimeDraft(current, (draft) => {
                          draft.credential_routing.strategy = event.target.value;
                        })
                      )
                    }
                  >
                    <option value="round-robin">round-robin</option>
                    <option value="weighted-round-robin">weighted-round-robin</option>
                    <option value="fill-first">fill-first</option>
                  </select>
                </label>
                <Input
                  label={t('system_info.runtime_codex_live_max_sessions')}
                  value={codexLiveMaxSessionsText}
                  disabled={runtimeSaving}
                  onChange={(event) => setCodexLiveMaxSessionsText(event.target.value)}
                />
                <Input
                  label={t('system_info.runtime_codex_live_public_ip')}
                  value={codexLivePublicIPText}
                  placeholder="203.0.113.10"
                  disabled={runtimeSaving}
                  onChange={(event) => setCodexLivePublicIPText(event.target.value)}
                />
                <Input
                  label={t('system_info.runtime_codex_live_udp_min')}
                  value={codexLiveUDPMinText}
                  disabled={runtimeSaving}
                  onChange={(event) => setCodexLiveUDPMinText(event.target.value)}
                />
                <Input
                  label={t('system_info.runtime_codex_live_udp_max')}
                  value={codexLiveUDPMaxText}
                  disabled={runtimeSaving}
                  onChange={(event) => setCodexLiveUDPMaxText(event.target.value)}
                />
              </div>
              <div className="grid grid-cols-[repeat(auto-fit,minmax(260px,1fr))] gap-3">
                <ToggleSwitch
                  label={t('system_info.runtime_codex_live_enabled')}
                  labelPosition="left"
                  checked={runtimeDraft.codex_live.enabled}
                  disabled={runtimeSaving}
                  onChange={(value) =>
                    setRuntimeDraft((current) =>
                      updateRuntimeDraft(current, (draft) => {
                        draft.codex_live.enabled = value;
                      })
                    )
                  }
                />
                <ToggleSwitch
                  label={t('system_info.runtime_codex_live_private_ips')}
                  labelPosition="left"
                  checked={runtimeDraft.codex_live.disable_private_remote_ips}
                  disabled={runtimeSaving}
                  onChange={(value) =>
                    setRuntimeDraft((current) =>
                      updateRuntimeDraft(current, (draft) => {
                        draft.codex_live.disable_private_remote_ips = value;
                      })
                    )
                  }
                />
                <ToggleSwitch
                  label={t('system_info.runtime_cloak_claude_models')}
                  labelPosition="left"
                  checked={runtimeDraft.cloaking.disable_claude_model_list}
                  disabled={runtimeSaving}
                  onChange={(value) =>
                    setRuntimeDraft((current) =>
                      updateRuntimeDraft(current, (draft) => {
                        draft.cloaking.disable_claude_model_list = value;
                      })
                    )
                  }
                />
                <ToggleSwitch
                  label={t('system_info.runtime_cloak_codex')}
                  labelPosition="left"
                  checked={runtimeDraft.cloaking.disable_codex}
                  disabled={runtimeSaving}
                  onChange={(value) =>
                    setRuntimeDraft((current) =>
                      updateRuntimeDraft(current, (draft) => {
                        draft.cloaking.disable_codex = value;
                      })
                    )
                  }
                />
                <ToggleSwitch
                  label={t('system_info.runtime_home_enabled')}
                  labelPosition="left"
                  checked={runtimeDraft.home.enabled}
                  disabled={runtimeSaving}
                  onChange={(value) =>
                    setRuntimeDraft((current) =>
                      updateRuntimeDraft(current, (draft) => {
                        draft.home.enabled = value;
                      })
                    )
                  }
                />
                <ToggleSwitch
                  label={t('system_info.runtime_home_discovery')}
                  labelPosition="left"
                  checked={runtimeDraft.home.disable_cluster_discovery}
                  disabled={runtimeSaving}
                  onChange={(value) =>
                    setRuntimeDraft((current) =>
                      updateRuntimeDraft(current, (draft) => {
                        draft.home.disable_cluster_discovery = value;
                      })
                    )
                  }
                />
                <ToggleSwitch
                  label={t('system_info.runtime_cooldown_persistence')}
                  labelPosition="left"
                  checked={runtimeDraft.cooldown_persistence_enabled}
                  disabled={runtimeSaving}
                  onChange={(value) =>
                    setRuntimeDraft((current) =>
                      updateRuntimeDraft(current, (draft) => {
                        draft.cooldown_persistence_enabled = value;
                      })
                    )
                  }
                />
              </div>
              <div className="flex justify-between gap-3 items-center max-sm:flex-col max-sm:items-stretch">
                <div className="text-xs text-muted-foreground">
                  {t('system_info.runtime_controls_revision', { revision: runtimeDraft.revision })}
                </div>
                <div className="flex gap-2 justify-end">
                  <Button variant="secondary" onClick={() => void loadRuntimeControls()} disabled={runtimeSaving}>
                    {t('common.refresh')}
                  </Button>
                  <Button onClick={() => void handleRuntimeSave()} loading={runtimeSaving}>
                    {t('common.save')}
                  </Button>
                </div>
              </div>
            </div>
          ) : null}
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
          <div className="inline-flex items-center text-[0.8125rem] font-medium px-[10px] py-[2px] border rounded-sm text-warning bg-warning/12 border-warning/30 leading-[1.5]">{t('basic_settings.request_log_warning')}</div>
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
