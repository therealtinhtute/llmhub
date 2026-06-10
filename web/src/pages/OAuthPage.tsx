import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { AppCard as Card } from '@/components/ui/AppCard';
import { Button } from '@/components/ui/Button';
import { FormInput as Input } from '@/components/ui/FormInput';
import { IconPlus } from '@/components/ui/icons';
import { toast } from 'sonner';
import { useThemeStore } from '@/stores';
import { oauthApi, type OAuthProvider } from '@/services/api/oauth';
import { copyToClipboard } from '@/utils/clipboard';
import iconCodex from '@/assets/icons/codex.svg';
import iconClaude from '@/assets/icons/claude.svg';
import iconAntigravity from '@/assets/icons/antigravity.svg';
import iconGemini from '@/assets/icons/gemini.svg';
import iconKimiLight from '@/assets/icons/kimi-light.svg';
import iconKimiDark from '@/assets/icons/kimi-dark.svg';
import iconGrok from '@/assets/icons/grok.svg';
import iconGrokDark from '@/assets/icons/grok-dark.svg';

interface ProviderState {
  url?: string;
  state?: string;
  status?: 'idle' | 'waiting' | 'success' | 'error';
  error?: string;
  polling?: boolean;
  projectId?: string;
  projectIdError?: string;
  callbackUrl?: string;
  callbackSubmitting?: boolean;
  callbackStatus?: 'success' | 'error';
  callbackError?: string;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object';
}

function getErrorMessage(error: unknown): string {
  if (error instanceof Error) return error.message;
  if (isRecord(error) && typeof error.message === 'string') return error.message;
  return typeof error === 'string' ? error : '';
}

function getErrorStatus(error: unknown): number | undefined {
  if (!isRecord(error)) return undefined;
  return typeof error.status === 'number' ? error.status : undefined;
}

const PROVIDERS: { id: OAuthProvider; titleKey: string; hintKey: string; urlLabelKey: string; icon: string | { light: string; dark: string } }[] = [
  { id: 'codex', titleKey: 'auth_login.codex_oauth_title', hintKey: 'auth_login.codex_oauth_hint', urlLabelKey: 'auth_login.codex_oauth_url_label', icon: iconCodex },
  { id: 'anthropic', titleKey: 'auth_login.anthropic_oauth_title', hintKey: 'auth_login.anthropic_oauth_hint', urlLabelKey: 'auth_login.anthropic_oauth_url_label', icon: iconClaude },
  { id: 'antigravity', titleKey: 'auth_login.antigravity_oauth_title', hintKey: 'auth_login.antigravity_oauth_hint', urlLabelKey: 'auth_login.antigravity_oauth_url_label', icon: iconAntigravity },
  { id: 'gemini-cli', titleKey: 'auth_login.gemini_cli_oauth_title', hintKey: 'auth_login.gemini_cli_oauth_hint', urlLabelKey: 'auth_login.gemini_cli_oauth_url_label', icon: iconGemini },
  { id: 'kimi', titleKey: 'auth_login.kimi_oauth_title', hintKey: 'auth_login.kimi_oauth_hint', urlLabelKey: 'auth_login.kimi_oauth_url_label', icon: { light: iconKimiLight, dark: iconKimiDark } },
  { id: 'xai', titleKey: 'auth_login.xai_oauth_title', hintKey: 'auth_login.xai_oauth_hint', urlLabelKey: 'auth_login.xai_oauth_url_label', icon: { light: iconGrok, dark: iconGrokDark } }
];

const CALLBACK_SUPPORTED: OAuthProvider[] = [
  'codex',
  'anthropic',
  'antigravity',
  'gemini-cli',
  'xai'
];
const XAI_CALLBACK_URL = 'http://127.0.0.1:56121/callback';
const SUCCESS_RESET_DELAY_MS = 5000;
const getProviderI18nPrefix = (provider: OAuthProvider) => provider.replace('-', '_');
const getAuthKey = (provider: OAuthProvider, suffix: string) =>
  `auth_login.${getProviderI18nPrefix(provider)}_${suffix}`;

const getIcon = (icon: string | { light: string; dark: string }, theme: 'light' | 'dark') => {
  return typeof icon === 'string' ? icon : icon[theme];
};

const isAbsoluteUrl = (value: string): boolean => {
  try {
    new URL(value);
    return true;
  } catch {
    return false;
  }
};

const readQueryLikeCallbackInput = (value: string) => {
  const trimmed = value.trim();
  if (!trimmed) return null;
  const queryStart = trimmed.indexOf('?');
  const hashStart = trimmed.indexOf('#');
  const rawParams =
    queryStart >= 0
      ? trimmed.slice(queryStart + 1)
      : hashStart >= 0
        ? trimmed.slice(hashStart + 1)
        : trimmed;

  if (!/(^|[&#?])(code|state|error)=/i.test(rawParams)) return null;
  return new URLSearchParams(rawParams.replace(/^[?#]/, ''));
};

const extractDisplayedXaiCode = (value: string): string => {
  const trimmed = value.trim();
  const codeMatch = trimmed.match(/\bcode\s*[:=]\s*([^\s&]+)/i);
  return (codeMatch?.[1] ?? trimmed).trim();
};

const buildXaiCallbackUrl = (input: string, state?: string): string | null => {
  const trimmed = input.trim();
  if (!trimmed) return null;
  if (isAbsoluteUrl(trimmed)) return trimmed;

  const params = readQueryLikeCallbackInput(trimmed);
  if (params) {
    const code = params.get('code')?.trim();
    const error = params.get('error')?.trim();
    const errorDescription = params.get('error_description')?.trim();
    const callbackState = params.get('state')?.trim() || state?.trim();
    if (!callbackState) return null;

    const callbackUrl = new URL(XAI_CALLBACK_URL);
    callbackUrl.searchParams.set('state', callbackState);
    if (code) callbackUrl.searchParams.set('code', code);
    if (error) callbackUrl.searchParams.set('error', error);
    if (errorDescription) callbackUrl.searchParams.set('error_description', errorDescription);
    return callbackUrl.toString();
  }

  const code = extractDisplayedXaiCode(trimmed);
  const callbackState = state?.trim();
  if (!code || !callbackState) return null;

  const callbackUrl = new URL(XAI_CALLBACK_URL);
  callbackUrl.searchParams.set('code', code);
  callbackUrl.searchParams.set('state', callbackState);
  return callbackUrl.toString();
};

const resolveCallbackUrl = (
  provider: OAuthProvider,
  input: string,
  state?: string
): string | null => {
  if (provider !== 'xai') return input.trim();
  return buildXaiCallbackUrl(input, state);
};

export function OAuthPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const resolvedTheme = useThemeStore((state) => state.resolvedTheme);
  const [states, setStates] = useState<Record<OAuthProvider, ProviderState>>({} as Record<OAuthProvider, ProviderState>);
  const pollingTimers = useRef<Partial<Record<OAuthProvider, number>>>({});
  const successResetTimers = useRef<Partial<Record<OAuthProvider, number>>>({});

  const clearTimers = useCallback(() => {
    Object.values(pollingTimers.current).forEach((timer) => {
      if (timer !== undefined) window.clearInterval(timer);
    });
    Object.values(successResetTimers.current).forEach((timer) => {
      if (timer !== undefined) window.clearTimeout(timer);
    });
    pollingTimers.current = {};
    successResetTimers.current = {};
  }, []);

  useEffect(() => {
    return () => {
      clearTimers();
    };
  }, [clearTimers]);

  const updateProviderState = (provider: OAuthProvider, next: Partial<ProviderState>) => {
    setStates((prev) => ({
      ...prev,
      [provider]: { ...(prev[provider] ?? {}), ...next }
    }));
  };

  const clearPollingTimer = (provider: OAuthProvider) => {
    const timer = pollingTimers.current[provider];
    if (timer !== undefined) {
      window.clearInterval(timer);
      delete pollingTimers.current[provider];
    }
  };

  const clearSuccessResetTimer = (provider: OAuthProvider) => {
    const timer = successResetTimers.current[provider];
    if (timer !== undefined) {
      window.clearTimeout(timer);
      delete successResetTimers.current[provider];
    }
  };

  const clearProviderTimers = (provider: OAuthProvider) => {
    clearPollingTimer(provider);
    clearSuccessResetTimer(provider);
  };

  const resetProviderAttempt = (provider: OAuthProvider) => {
    clearProviderTimers(provider);
    setStates((prev) => {
      const current = prev[provider] ?? {};
      const next: ProviderState = {};
      if (provider === 'gemini-cli' && current.projectId !== undefined) {
        next.projectId = current.projectId;
      }
      return {
        ...prev,
        [provider]: next
      };
    });
  };

  const completeProviderAuth = (provider: OAuthProvider) => {
    clearPollingTimer(provider);
    clearSuccessResetTimer(provider);
    updateProviderState(provider, {
      url: undefined,
      state: undefined,
      status: 'success',
      error: undefined,
      polling: false,
      callbackUrl: '',
      callbackSubmitting: false,
      callbackStatus: undefined,
      callbackError: undefined
    });
    successResetTimers.current[provider] = window.setTimeout(() => {
      resetProviderAttempt(provider);
    }, SUCCESS_RESET_DELAY_MS);
  };

  const startPolling = (provider: OAuthProvider, state: string) => {
    clearPollingTimer(provider);
    const timer = window.setInterval(async () => {
      try {
        const res = await oauthApi.getAuthStatus(state);
        if (res.status === 'ok') {
          completeProviderAuth(provider);
          toast.success(t(getAuthKey(provider, 'oauth_status_success')));
        } else if (res.status === 'error') {
          updateProviderState(provider, { status: 'error', error: res.error, polling: false });
          toast.error(
            `${t(getAuthKey(provider, 'oauth_status_error'))} ${res.error || ''}`
          );
          window.clearInterval(timer);
          delete pollingTimers.current[provider];
        }
      } catch (err: unknown) {
        updateProviderState(provider, { status: 'error', error: getErrorMessage(err), polling: false });
        window.clearInterval(timer);
        delete pollingTimers.current[provider];
      }
    }, 3000);
    pollingTimers.current[provider] = timer;
  };

  const startAuth = async (provider: OAuthProvider) => {
    clearProviderTimers(provider);
    const geminiState = provider === 'gemini-cli' ? states[provider] : undefined;
    const rawProjectId = provider === 'gemini-cli' ? (geminiState?.projectId || '').trim() : '';
    const projectId = rawProjectId
      ? rawProjectId.toUpperCase() === 'ALL'
        ? 'ALL'
        : rawProjectId
      : undefined;
    // 项目 ID 可选：留空自动选择第一个可用项目；输入 ALL 获取全部项目
    if (provider === 'gemini-cli') {
      updateProviderState(provider, { projectIdError: undefined });
    }
    updateProviderState(provider, {
      url: undefined,
      state: undefined,
      status: 'waiting',
      polling: true,
      error: undefined,
      callbackStatus: undefined,
      callbackError: undefined,
      callbackUrl: ''
    });
    try {
      const res = await oauthApi.startAuth(
        provider,
        provider === 'gemini-cli' ? { projectId: projectId || undefined } : undefined
      );
      if (!res.state) {
        const message = t('auth_login.missing_state');
        updateProviderState(provider, {
          url: res.url,
          state: undefined,
          status: 'error',
          error: message,
          polling: false
        });
        toast.error(message);
        return;
      }
      updateProviderState(provider, { url: res.url, state: res.state, status: 'waiting', polling: true });
      startPolling(provider, res.state);
    } catch (err: unknown) {
      const message = getErrorMessage(err);
      updateProviderState(provider, { status: 'error', error: message, polling: false });
      toast.error(
        `${t(getAuthKey(provider, 'oauth_start_error'))}${message ? ` ${message}` : ''}`
      );
    }
  };

  const copyLink = async (url?: string) => {
    if (!url) return;
    const copied = await copyToClipboard(url);
    if (copied) {
      toast.success(t('notification.link_copied'));
    } else {
      toast.error(t('notification.copy_failed'));
    }
  };

  const submitCallback = async (provider: OAuthProvider) => {
    const callbackInput = (states[provider]?.callbackUrl || '').trim();
    if (!callbackInput) {
      toast.warning(
        t(provider === 'xai' ? 'auth_login.xai_callback_required' : 'auth_login.oauth_callback_required')
      );
      return;
    }
    const redirectUrl = resolveCallbackUrl(provider, callbackInput, states[provider]?.state);
    if (!redirectUrl) {
      toast.warning(t(provider === 'xai' ? 'auth_login.xai_callback_state_missing' : 'auth_login.missing_state'));
      return;
    }
    updateProviderState(provider, {
      callbackSubmitting: true,
      callbackStatus: undefined,
      callbackError: undefined
    });
    try {
      await oauthApi.submitCallback(provider, redirectUrl);
      updateProviderState(provider, { callbackSubmitting: false, callbackStatus: 'success' });
      toast.success(t('auth_login.oauth_callback_success'));
    } catch (err: unknown) {
      const status = getErrorStatus(err);
      const message = getErrorMessage(err);
      const errorMessage =
        status === 404
          ? t('auth_login.oauth_callback_upgrade_hint', {
              defaultValue: 'Please update LLMHub or check the connection.'
            })
          : message || undefined;
      updateProviderState(provider, {
        callbackSubmitting: false,
        callbackStatus: 'error',
        callbackError: errorMessage
      });
      const notificationMessage = errorMessage
        ? `${t('auth_login.oauth_callback_error')} ${errorMessage}`
        : t('auth_login.oauth_callback_error');
      toast.error(notificationMessage);
    }
  };

  return (
    <div className="w-full">
      <h1 className="text-[28px] font-bold text-foreground mb-6">{t('nav.oauth', { defaultValue: 'OAuth' })}</h1>

      <div className="flex flex-col gap-6">
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-4">
          {PROVIDERS.map((provider) => {
            const state = states[provider.id] || {};
            const canSubmitCallback = CALLBACK_SUPPORTED.includes(provider.id) && Boolean(state.url);
            const loginButtonLabel =
              state.status === 'success'
                ? t('auth_login.login_another_account')
                : t(getAuthKey(provider.id, 'oauth_button'));
            const statusBadgeClassName = [
              'inline-flex items-center text-[0.8125rem] font-medium px-[10px] py-[2px] border border-border text-muted-foreground bg-muted leading-[1.5] rounded-sm',
              state.status === 'success' ? 'text-emerald-700 bg-emerald-100 border-emerald-400/40' : '',
              state.status === 'error' ? 'text-destructive bg-destructive/10 border-destructive/30' : ''
            ]
              .filter(Boolean)
              .join(' ');
            return (
              <Card key={provider.id} className="h-full border-border/70 bg-card/95 shadow-xs">
                <div className="flex h-full flex-col gap-4">
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex min-w-0 items-start gap-3 pr-2">
                      <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-md border border-border/70 bg-muted/55">
                        <img
                          src={getIcon(provider.icon, resolvedTheme)}
                          alt=""
                          className="h-5 w-5 shrink-0"
                        />
                      </span>
                      <span className="flex min-w-0 flex-col gap-1">
                        <span className="truncate text-sm font-semibold text-foreground">
                          {t(provider.titleKey)}
                        </span>
                        <span className="text-[12px] leading-[1.5] text-muted-foreground">
                          {t(provider.hintKey)}
                        </span>
                      </span>
                    </div>
                    <Button
                      variant="secondary"
                      size="icon-sm"
                      onClick={() => startAuth(provider.id)}
                      loading={state.polling}
                      className="shrink-0 rounded-full border-border/80 bg-background/90 text-foreground shadow-none"
                      title={loginButtonLabel}
                      aria-label={loginButtonLabel}
                    >
                      {!state.polling ? <IconPlus size={14} /> : null}
                    </Button>
                  </div>

                  {provider.id === 'gemini-cli' && (
                    <div className="flex flex-col gap-2">
                      <Input
                        label={t('auth_login.gemini_cli_project_id_label')}
                        hint={t('auth_login.gemini_cli_project_id_hint')}
                        value={state.projectId || ''}
                        error={state.projectIdError}
                        disabled={Boolean(state.polling)}
                        onChange={(e) =>
                          updateProviderState(provider.id, {
                            projectId: e.target.value,
                            projectIdError: undefined
                          })
                        }
                        placeholder={t('auth_login.gemini_cli_project_id_placeholder')}
                      />
                    </div>
                  )}

                  {state.url && (
                    <div className="flex flex-col gap-1 border border-dashed border-border bg-muted p-3">
                      <div className="text-sm text-muted-foreground">{t(provider.urlLabelKey)}</div>
                      <div className="max-w-full break-all font-bold leading-relaxed text-foreground">
                        {state.url}
                      </div>
                      <div className="mt-2 flex flex-wrap items-center gap-2">
                        <Button variant="secondary" size="sm" onClick={() => copyLink(state.url!)}>
                          {t(getAuthKey(provider.id, 'copy_link'))}
                        </Button>
                        <Button
                          variant="secondary"
                          size="sm"
                          onClick={() => window.open(state.url, '_blank', 'noopener,noreferrer')}
                        >
                          {t(getAuthKey(provider.id, 'open_link'))}
                        </Button>
                      </div>
                    </div>
                  )}

                  {canSubmitCallback && (
                    <div className="flex flex-col gap-2">
                      <Input
                        label={t(
                          provider.id === 'xai'
                            ? 'auth_login.xai_callback_label'
                            : 'auth_login.oauth_callback_label'
                        )}
                        hint={t(
                          provider.id === 'xai'
                            ? 'auth_login.xai_callback_hint'
                            : 'auth_login.oauth_callback_hint'
                        )}
                        value={state.callbackUrl || ''}
                        onChange={(e) =>
                          updateProviderState(provider.id, {
                            callbackUrl: e.target.value,
                            callbackStatus: undefined,
                            callbackError: undefined
                          })
                        }
                        placeholder={t(
                          provider.id === 'xai'
                            ? 'auth_login.xai_callback_placeholder'
                            : 'auth_login.oauth_callback_placeholder'
                        )}
                      />
                      <div className="flex gap-3">
                        <Button
                          variant="secondary"
                          size="sm"
                          onClick={() => submitCallback(provider.id)}
                          loading={state.callbackSubmitting}
                        >
                          {t('auth_login.oauth_callback_button')}
                        </Button>
                      </div>
                      {state.callbackStatus === 'success' && state.status === 'waiting' && (
                        <div className="inline-flex items-center rounded-sm border border-emerald-400/40 bg-emerald-100 px-[10px] py-[2px] text-[0.8125rem] font-medium leading-[1.5] text-emerald-700">
                          {t('auth_login.oauth_callback_status_success')}
                        </div>
                      )}
                      {state.callbackStatus === 'error' && (
                        <div className="inline-flex items-center rounded-sm border border-destructive/30 bg-destructive/10 px-[10px] py-[2px] text-[0.8125rem] font-medium leading-[1.5] text-destructive">
                          {t('auth_login.oauth_callback_status_error')} {state.callbackError || ''}
                        </div>
                      )}
                    </div>
                  )}

                  {state.status && state.status !== 'idle' && (
                    <div className={statusBadgeClassName}>
                      {state.status === 'success'
                        ? t(getAuthKey(provider.id, 'oauth_status_success'))
                        : state.status === 'error'
                          ? `${t(getAuthKey(provider.id, 'oauth_status_error'))} ${state.error || ''}`
                          : t(getAuthKey(provider.id, 'oauth_status_waiting'))}
                    </div>
                  )}

                  {state.status === 'success' && (
                    <div className="mt-auto flex flex-wrap items-center gap-2">
                      <Button variant="secondary" size="sm" onClick={() => navigate('/auth-files')}>
                        {t('auth_login.view_auth_files')}
                      </Button>
                    </div>
                  )}
                </div>
              </Card>
            );
          })}
        </div>
      </div>
    </div>
  );
}
