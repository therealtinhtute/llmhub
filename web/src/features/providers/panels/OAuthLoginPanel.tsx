import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/Button';
import { FormInput as Input } from '@/components/ui/FormInput';
import { IconPlus } from '@/components/ui/icons';
import { useThemeStore } from '@/stores';
import type { OAuthProvider } from '@/services/api/oauth';
import {
  CALLBACK_SUPPORTED,
  getAuthKey,
  getOAuthIcon,
  type ProviderEntryOAuthMeta,
  type ProviderOAuthState,
} from '../entries';

interface OAuthLoginPanelProps {
  provider: ProviderEntryOAuthMeta;
  state: ProviderOAuthState;
  onStart: (providerId: OAuthProvider, projectId?: string) => void;
  onSubmitCallback: (providerId: OAuthProvider, callbackInput: string) => void;
  onReset: (providerId: OAuthProvider) => void;
  onCopyLink: (url?: string) => void;
}

export function OAuthLoginPanel({
  provider,
  state,
  onStart,
  onSubmitCallback,
  onReset,
  onCopyLink,
}: OAuthLoginPanelProps) {
  const { t } = useTranslation();
  const resolvedTheme = useThemeStore((s) => s.resolvedTheme);
  const [projectId, setProjectId] = useState('');
  const [callbackInput, setCallbackInput] = useState('');

  const canSubmitCallback = CALLBACK_SUPPORTED.includes(provider.id) && Boolean(state.url);
  const loginButtonLabel =
    state.status === 'success'
      ? t('auth_login.login_another_account')
      : t(getAuthKey(provider.id, 'oauth_button'));
  const statusBadgeClassName = [
    'inline-flex items-center text-[0.8125rem] font-medium px-[10px] py-[2px] border border-border text-muted-foreground bg-muted leading-[1.5] rounded-sm',
    state.status === 'success' ? 'text-success bg-success/12 border-success/30' : '',
    state.status === 'error' ? 'text-destructive bg-destructive/10 border-destructive/30' : '',
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-start gap-3 pr-2">
          <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-md border border-border/70 bg-muted/55">
            <img
              src={getOAuthIcon(provider.icon, resolvedTheme)}
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
          onClick={() => onStart(provider.id, provider.id === 'gemini-cli' ? projectId : undefined)}
          loading={state.polling}
          className="shrink-0 rounded-full border-border/80 bg-background/90 text-foreground shadow-none"
          title={loginButtonLabel}
          aria-label={loginButtonLabel}
        >
          {!state.polling ? <IconPlus size={14} /> : null}
        </Button>
      </div>

      {provider.id === 'gemini-cli' && (
        <Input
          label={t('auth_login.gemini_cli_project_id_label')}
          hint={t('auth_login.gemini_cli_project_id_hint')}
          value={projectId}
          disabled={Boolean(state.polling)}
          onChange={(e) => setProjectId(e.target.value)}
          placeholder={t('auth_login.gemini_cli_project_id_placeholder')}
        />
      )}

      {state.url && (
        <div className="flex flex-col gap-1 border border-dashed border-border bg-muted p-3">
          <div className="text-sm text-muted-foreground">{t(provider.urlLabelKey)}</div>
          <div className="max-w-full break-all font-bold leading-relaxed text-foreground">
            {state.url}
          </div>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <Button variant="secondary" size="sm" onClick={() => onCopyLink(state.url)}>
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
              provider.id === 'xai' ? 'auth_login.xai_callback_label' : 'auth_login.oauth_callback_label'
            )}
            hint={t(
              provider.id === 'xai' ? 'auth_login.xai_callback_hint' : 'auth_login.oauth_callback_hint'
            )}
            value={callbackInput}
            onChange={(e) => setCallbackInput(e.target.value)}
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
              onClick={() => onSubmitCallback(provider.id, callbackInput)}
              loading={state.callbackSubmitting}
            >
              {t('auth_login.oauth_callback_button')}
            </Button>
          </div>
          {state.callbackStatus === 'success' && state.status === 'waiting' && (
            <div className="inline-flex items-center rounded-sm border border-success/30 bg-success/12 px-[10px] py-[2px] text-[0.8125rem] font-medium leading-[1.5] text-success">
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

      {state.status === 'error' && (
        <div>
          <Button variant="secondary" size="sm" onClick={() => onReset(provider.id)}>
            {t('providersPage.oauthAccounts.tryAgain')}
          </Button>
        </div>
      )}
    </div>
  );
}
