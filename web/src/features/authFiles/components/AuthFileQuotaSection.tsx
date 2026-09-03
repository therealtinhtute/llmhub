import { useCallback, useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import type { TFunction } from 'i18next';
import {
  ANTIGRAVITY_CONFIG,
  CLAUDE_CONFIG,
  CODEX_CONFIG,
  GEMINI_CLI_CONFIG,
  KIRO_CONFIG,
  KIMI_CONFIG,
  XAI_CONFIG,
  buildKiroQuotaStateFromProvider,
} from '@/components/quota';
import { toast } from 'sonner';
import { useConfirmationStore, useQuotaStore } from '@/stores';
import { authFilesApi } from '@/services/api';
import type { AuthFileItem, KiroQuotaState } from '@/types';
import { normalizeAuthIndex } from '@/utils/authIndex';
import { getStatusFromError } from '@/utils/quota';
import {
  isRuntimeOnlyAuthFile,
  resolveQuotaErrorMessage,
  type QuotaProviderType,
} from '@/features/authFiles/constants';
import { QuotaProgressBar } from '@/features/authFiles/components/QuotaProgressBar';
import { Button } from '@/components/ui/Button';
import { IconRefreshCw } from '@/components/ui/icons';

// Tailwind class equivalents for quota rendering classes,
// passed to renderQuotaItems as the helpers.styles object.
const quotaStyleMap = {
  quotaMessage: 'text-[12px] text-muted-foreground/60 text-center py-2',
  quotaMessageAction:
    'w-full border-none bg-transparent cursor-pointer underline disabled:cursor-not-allowed disabled:opacity-60 disabled:no-underline text-[12px] text-muted-foreground/60 text-center py-2',
  quotaError:
    'text-[12px] text-destructive bg-destructive/[.08] border border-destructive py-1 px-2',
  quotaWarning: 'text-[12px] text-warning bg-warning/12 border border-warning/30 py-1 px-2',
  quotaRow: 'flex flex-col gap-1',
  quotaRowHeader: 'flex items-center justify-between gap-2 min-w-0',
  quotaModel:
    'text-[13px] font-semibold text-foreground whitespace-nowrap overflow-hidden text-ellipsis flex-1 min-w-0',
  quotaBar: 'h-1 bg-secondary rounded-full overflow-hidden',
  quotaBarFill: 'h-full',
  quotaMeta: 'flex items-center gap-2 text-[12px] text-muted-foreground whitespace-nowrap',
  quotaPercent: 'font-semibold text-foreground tabular-nums',
  quotaReset: 'text-muted-foreground/60',
  quotaAmount: 'text-muted-foreground tabular-nums',
  codexPlan: 'flex flex-wrap items-center gap-x-3 gap-y-1 text-[12px] text-muted-foreground',
  codexPlanItem: 'inline-flex items-center gap-1.5',
  codexPlanLabel: 'text-muted-foreground/60',
  codexPlanValue: 'font-semibold text-foreground capitalize',
  codexResetCredits: 'flex flex-col gap-1 border border-border bg-muted/40 px-2 py-1.5',
  codexResetCreditsTitle: 'text-[11px] font-semibold text-muted-foreground/60',
  codexResetCreditRow: 'flex items-center justify-between gap-2 text-[11px]',
  codexResetCreditLabel: 'text-muted-foreground',
  codexResetCreditTime: 'font-medium text-foreground tabular-nums',
  codexResetCreditsError: 'text-[11px] text-destructive',
  overagePlanValue: 'flex min-w-0 flex-1 items-center justify-between gap-2 font-semibold text-foreground',
  overageToggle:
    'shrink-0 border border-border bg-background px-2 py-0.5 text-[11px] font-semibold text-foreground hover:not-disabled:bg-muted disabled:cursor-not-allowed disabled:opacity-55',
  premiumPlanValue:
    'inline-flex items-center font-bold text-[12px] px-2 py-[2px] bg-warning/12 border border-warning/30 text-warning capitalize',
};

type QuotaState = { status?: string; error?: string; errorStatus?: number } | undefined;

const getQuotaConfig = (type: QuotaProviderType) => {
  if (type === 'antigravity') return ANTIGRAVITY_CONFIG;
  if (type === 'claude') return CLAUDE_CONFIG;
  if (type === 'codex') return CODEX_CONFIG;
  if (type === 'kiro') return KIRO_CONFIG;
  if (type === 'kimi') return KIMI_CONFIG;
  if (type === 'xai') return XAI_CONFIG;
  return GEMINI_CLI_CONFIG;
};

export type AuthFileQuotaSectionProps = {
  file: AuthFileItem;
  quotaType: QuotaProviderType;
  disableControls: boolean;
};

export function AuthFileQuotaSection(props: AuthFileQuotaSectionProps) {
  const { file, quotaType, disableControls } = props;
  const { t } = useTranslation();
  const showConfirmation = useConfirmationStore((state) => state.showConfirmation);
  const [resettingQuota, setResettingQuota] = useState(false);

  const quota = useQuotaStore((state) => {
    if (quotaType === 'antigravity') return state.antigravityQuota[file.name] as QuotaState;
    if (quotaType === 'claude') return state.claudeQuota[file.name] as QuotaState;
    if (quotaType === 'codex') return state.codexQuota[file.name] as QuotaState;
    if (quotaType === 'kiro')
      return (state.kiroQuota[file.name] ?? KIRO_CONFIG.buildRuntimeState?.(file)) as QuotaState;
    if (quotaType === 'kimi') return state.kimiQuota[file.name] as QuotaState;
    if (quotaType === 'xai') return state.xaiQuota[file.name] as QuotaState;
    return state.geminiCliQuota[file.name] as QuotaState;
  });

  const updateQuotaState = useQuotaStore((state) => {
    if (quotaType === 'antigravity')
      return state.setAntigravityQuota as unknown as (updater: unknown) => void;
    if (quotaType === 'claude')
      return state.setClaudeQuota as unknown as (updater: unknown) => void;
    if (quotaType === 'codex') return state.setCodexQuota as unknown as (updater: unknown) => void;
    if (quotaType === 'kiro') return state.setKiroQuota as unknown as (updater: unknown) => void;
    if (quotaType === 'kimi') return state.setKimiQuota as unknown as (updater: unknown) => void;
    if (quotaType === 'xai') return state.setXaiQuota as unknown as (updater: unknown) => void;
    return state.setGeminiCliQuota as unknown as (updater: unknown) => void;
  });

  const refreshQuotaForFile = useCallback(async () => {
    if (disableControls) return;
    if (isRuntimeOnlyAuthFile(file)) return;
    if (file.disabled) return;
    if (quota?.status === 'loading') return;

    const config = getQuotaConfig(quotaType) as unknown as {
      i18nPrefix: string;
      fetchQuota: (file: AuthFileItem, t: TFunction) => Promise<unknown>;
      buildLoadingState: () => unknown;
      buildSuccessState: (data: unknown) => unknown;
      buildErrorState: (message: string, status?: number) => unknown;
      renderQuotaItems: (quota: unknown, t: TFunction, helpers: unknown) => unknown;
    };

    updateQuotaState((prev: Record<string, unknown>) => ({
      ...prev,
      [file.name]: config.buildLoadingState(),
    }));

    try {
      const data = await config.fetchQuota(file, t);
      updateQuotaState((prev: Record<string, unknown>) => ({
        ...prev,
        [file.name]: config.buildSuccessState(data),
      }));
      toast.success(t('auth_files.quota_refresh_success', { name: file.name }));
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('common.unknown_error');
      const status = getStatusFromError(err);
      updateQuotaState((prev: Record<string, unknown>) => ({
        ...prev,
        [file.name]: config.buildErrorState(message, status),
      }));
      toast.error(t('auth_files.quota_refresh_failed', { name: file.name, message }));
    }
  }, [disableControls, file, quota?.status, quotaType, t, updateQuotaState]);

  const resetQuotaForFile = useCallback(() => {
    const config = getQuotaConfig(quotaType) as unknown as {
      resetQuota?: (file: AuthFileItem, t: TFunction) => Promise<unknown>;
      buildSuccessState: (data: unknown) => unknown;
    };
    const resetQuota = config.resetQuota;
    if (!resetQuota || disableControls || file.disabled || quota?.status === 'loading') return;
    if (resettingQuota) return;

    showConfirmation({
      title: t('codex_quota.reset_confirm_title'),
      message: t('codex_quota.reset_confirm_message', { name: file.name }),
      confirmText: t('codex_quota.reset_confirm_button'),
      variant: 'primary',
      onConfirm: async () => {
        setResettingQuota(true);
        try {
          const data = await resetQuota(file, t);
          updateQuotaState((prev: Record<string, unknown>) => ({
            ...prev,
            [file.name]: config.buildSuccessState(data),
          }));
          toast.success(t('codex_quota.reset_success', { name: file.name }));
        } catch (err: unknown) {
          const message = err instanceof Error ? err.message : t('common.unknown_error');
          toast.error(t('codex_quota.reset_failed', { name: file.name, message }));
        } finally {
          setResettingQuota(false);
        }
      },
    });
  }, [
    disableControls,
    file,
    quota?.status,
    quotaType,
    resettingQuota,
    showConfirmation,
    t,
    updateQuotaState,
  ]);

  const setKiroOverageForFile = useCallback(
    async (enabled: boolean) => {
      if (quotaType !== 'kiro') return;
      if (disableControls || file.disabled) return;
      const current = (useQuotaStore.getState().kiroQuota[file.name] ??
        KIRO_CONFIG.buildRuntimeState?.(file)) as KiroQuotaState | undefined;
      if (!current?.providerQuotaAvailable || current.status === 'loading' || current.overageUpdating) {
        return;
      }

      const authIndex = normalizeAuthIndex(file['auth_index'] ?? file.authIndex);
      useQuotaStore.getState().setKiroQuota((prev) => ({
        ...prev,
        [file.name]: {
          ...current,
          overageUpdating: true,
        },
      }));

      try {
        const providerQuota = await authFilesApi.setKiroOverage({
          name: file.name,
          authIndex: authIndex ?? undefined,
          enabled,
        });
        useQuotaStore.getState().setKiroQuota((prev) => ({
          ...prev,
          [file.name]: buildKiroQuotaStateFromProvider(file, providerQuota),
        }));
        toast.success(
          t(enabled ? 'kiro_quota.overage_enable_success' : 'kiro_quota.overage_disable_success', {
            name: file.name,
          })
        );
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : t('common.unknown_error');
        useQuotaStore.getState().setKiroQuota((prev) => ({
          ...prev,
          [file.name]: {
            ...(prev[file.name] ?? current),
            overageUpdating: false,
          },
        }));
        toast.error(t('kiro_quota.overage_toggle_failed', { name: file.name, message }));
      }
    },
    [disableControls, file, quotaType, t]
  );

  const config = getQuotaConfig(quotaType) as unknown as {
    i18nPrefix: string;
    resetQuota?: (file: AuthFileItem, t: TFunction) => Promise<unknown>;
    canResetQuota?: (quota: unknown) => boolean;
    renderQuotaItems: (quota: unknown, t: TFunction, helpers: unknown) => unknown;
  };

  const quotaStatus = quota?.status ?? 'idle';
  const canRefreshQuota = !disableControls && !file.disabled;
  const showResetQuotaAction = quota !== undefined && Boolean(config.canResetQuota?.(quota));
  const quotaErrorMessage = resolveQuotaErrorMessage(
    t,
    quota?.errorStatus,
    quota?.error || t('common.unknown_error')
  );

  return (
    <div className="flex flex-col gap-2 pt-2 mt-[2px] border-t border-dashed border-border/80">
      {quotaStatus === 'loading' ? (
        <div className={quotaStyleMap.quotaMessage}>{t(`${config.i18nPrefix}.loading`)}</div>
      ) : quotaStatus === 'idle' ? (
        <button
          type="button"
          className={quotaStyleMap.quotaMessageAction}
          onClick={() => void refreshQuotaForFile()}
          disabled={!canRefreshQuota}
        >
          {t(`${config.i18nPrefix}.idle`)}
        </button>
      ) : quotaStatus === 'error' ? (
        <div className={quotaStyleMap.quotaError}>
          {t(`${config.i18nPrefix}.load_failed`, {
            message: quotaErrorMessage,
          })}
        </div>
      ) : quota ? (
        (config.renderQuotaItems(quota, t, {
          styles: quotaStyleMap,
          QuotaProgressBar,
          item: file,
          quotaDisabled: file.disabled,
          onSetKiroOverage:
            quotaType === 'kiro'
              ? (_item: AuthFileItem, enabled: boolean) => setKiroOverageForFile(enabled)
              : undefined,
        }) as ReactNode)
      ) : (
        <div className={quotaStyleMap.quotaMessage}>{t(`${config.i18nPrefix}.idle`)}</div>
      )}
      {config.resetQuota && showResetQuotaAction && (
        <div className="flex justify-end">
          <Button
            type="button"
            variant="secondary"
            size="sm"
            onClick={resetQuotaForFile}
            disabled={!canRefreshQuota || quotaStatus === 'loading' || resettingQuota}
            loading={resettingQuota}
            title={t('codex_quota.reset_button')}
            aria-label={t('codex_quota.reset_button')}
          >
            {!resettingQuota && <IconRefreshCw size={14} />}
            {t('codex_quota.reset_button')}
          </Button>
        </div>
      )}
    </div>
  );
}
