/**
 * Generic quota card component.
 */

import { useTranslation } from 'react-i18next';
import type { ReactElement, ReactNode } from 'react';
import type { TFunction } from 'i18next';
import type { AuthFileItem, ResolvedTheme, ThemeColors } from '@/types';
import { TYPE_COLORS } from '@/utils/quota';
import { normalizeAuthIndex } from '@/utils/authIndex';
import { Skeleton } from '@/components/ui/skeleton';
import { quotaStyles as styles } from './quotaStyles';
import type { QuotaStyleMap } from './quotaStyles';

const resetButtonClassName =
  'ml-auto shrink-0 rounded-md border border-border bg-background px-2 py-0.5 text-[11px] font-semibold text-foreground hover:not-disabled:bg-muted disabled:cursor-not-allowed disabled:opacity-55';

type QuotaStatus = 'idle' | 'loading' | 'success' | 'error' | 'runtime-only';

export interface QuotaStatusState {
  status: QuotaStatus;
  error?: string;
  errorStatus?: number;
}

export interface QuotaProgressBarProps {
  percent: number | null;
  muted?: boolean;
}

export function QuotaProgressBar({ percent, muted = false }: QuotaProgressBarProps) {
  const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));
  const normalized = percent === null ? null : clamp(percent, 0, 100);
  const fillClass =
    muted
      ? 'bg-muted-foreground/25'
      : normalized === null
      ? 'bg-amber-500'
      : normalized > 80
        ? 'bg-green-500'
        : normalized > 50
          ? 'bg-lime-500'
          : normalized > 20
            ? 'bg-amber-500'
            : normalized > 10
              ? 'bg-orange-500'
              : 'bg-destructive';
  const widthPercent = Math.round(normalized ?? 0);

  return (
    <div className={styles.quotaBar}>
      <div
        className={`${styles.quotaBarFill} ${fillClass}`}
        style={{ width: `${widthPercent}%` }}
      />
    </div>
  );
}

export interface QuotaRenderHelpers {
  styles: QuotaStyleMap;
  QuotaProgressBar: (props: QuotaProgressBarProps) => ReactElement;
  item?: AuthFileItem;
  quotaDisabled?: boolean;
  onSetKiroOverage?: (item: AuthFileItem, enabled: boolean) => void | Promise<void>;
}

interface QuotaCardProps<TState extends QuotaStatusState> {
  item: AuthFileItem;
  quota?: TState;
  resolvedTheme: ResolvedTheme;
  i18nPrefix: string;
  cardIdleMessageKey?: string;
  cardClassName: string;
  defaultType: string;
  canRefresh?: boolean;
  onRefresh?: () => void;
  canResetQuota?: boolean;
  resettingQuota?: boolean;
  onResetQuota?: () => void;
  onSetKiroOverage?: (item: AuthFileItem, enabled: boolean) => void | Promise<void>;
  renderQuotaItems: (quota: TState, t: TFunction, helpers: QuotaRenderHelpers) => ReactNode;
}

export function QuotaCard<TState extends QuotaStatusState>({
  item,
  quota,
  resolvedTheme: _resolvedTheme,
  i18nPrefix,
  cardIdleMessageKey,
  cardClassName,
  defaultType,
  canRefresh = false,
  onRefresh,
  canResetQuota = false,
  resettingQuota = false,
  onResetQuota,
  onSetKiroOverage,
  renderQuotaItems,
}: QuotaCardProps<TState>) {
  const { t } = useTranslation();

  const displayType = item.type || item.provider || defaultType;
  const typeColorSet = TYPE_COLORS[displayType] || TYPE_COLORS.unknown;
  const typeColor: ThemeColors = typeColorSet.light;

  const authIndexAvailable =
    normalizeAuthIndex(item['auth_index'] ?? item.authIndex) !== null;
  const resetDisabled = !canResetQuota || !authIndexAvailable || resettingQuota;
  const resetTooltip = !authIndexAvailable
    ? t('quota_management.reset_missing_auth_index')
    : t('quota_management.reset_action');

  const quotaStatus = quota?.status ?? 'idle';
  const quotaErrorMessage = resolveQuotaErrorMessage(
    t,
    quota?.errorStatus,
    quota?.error || t('common.unknown_error')
  );
  const idleMessageKey = onRefresh
    ? `${i18nPrefix}.idle`
    : (cardIdleMessageKey ?? `${i18nPrefix}.idle`);

  const getTypeLabel = (type: string): string => {
    const key = `auth_files.filter_${type}`;
    const translated = t(key);
    if (translated !== key) return translated;
    if (type.toLowerCase() === 'iflow') return 'iFlow';
    return type.charAt(0).toUpperCase() + type.slice(1);
  };

  return (
    <div className={`${styles.fileCard} ${cardClassName}`}>
      <div className={styles.cardHeader}>
        <span
          className={styles.typeBadge}
          style={{
            backgroundColor: typeColor.bg,
            color: typeColor.text,
            ...(typeColor.border ? { border: typeColor.border } : {}),
          }}
        >
          {getTypeLabel(displayType)}
        </span>
        <span className={styles.fileName}>{item.name}</span>
        {onResetQuota && (
          <button
            type="button"
            className={resetButtonClassName}
            onClick={onResetQuota}
            disabled={resetDisabled}
            title={resetTooltip}
            aria-label={resetTooltip}
          >
            {resettingQuota ? t('common.loading') : t('quota_management.reset_action')}
          </button>
        )}
      </div>

      <div className={styles.quotaSection}>
        {quotaStatus === 'loading' ? (
          <div className="flex flex-col gap-2" role="status" aria-label={t(`${i18nPrefix}.loading`)}>
            {[0, 1, 2].map((i) => (
              <div key={i} className="flex flex-col gap-1">
                <div className="flex items-center justify-between gap-2">
                  <Skeleton className="h-3 w-2/3" />
                  <Skeleton className="h-3 w-10" />
                </div>
                <Skeleton className="h-1 w-full rounded-full" />
              </div>
            ))}
          </div>
        ) : quotaStatus === 'idle' ? (
          onRefresh ? (
            <button
              type="button"
              className={`${styles.quotaMessage} ${styles.quotaMessageAction}`}
              onClick={onRefresh}
              disabled={!canRefresh}
            >
              {t(idleMessageKey)}
            </button>
          ) : (
            <div className={styles.quotaMessage}>{t(idleMessageKey)}</div>
          )
        ) : quotaStatus === 'error' ? (
          <div className={styles.quotaError}>
            {t(`${i18nPrefix}.load_failed`, {
              message: quotaErrorMessage,
            })}
          </div>
        ) : quota ? (
          renderQuotaItems(quota, t, {
            styles,
            QuotaProgressBar,
            item,
            quotaDisabled: item.disabled,
            onSetKiroOverage,
          })
        ) : (
          <div className={styles.quotaMessage}>{t(idleMessageKey)}</div>
        )}
      </div>
    </div>
  );
}

const resolveQuotaErrorMessage = (
  t: TFunction,
  status: number | undefined,
  fallback: string
): string => {
  if (status === 404) return t('common.quota_update_required');
  if (status === 403) return t('common.quota_check_credential');
  return fallback;
};
