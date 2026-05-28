import { useTranslation } from 'react-i18next';
import {
  IconLoader2,
  IconPlus,
  IconRefreshCw,
} from '@/components/ui/icons';

interface ProviderHeaderCardProps {
  totalActive: number;
  totalResources: number;
  providerFamilies: number;
  updatedAtLabel: string;
  issueCount?: number;
  isFetching?: boolean;
  isNewDisabled?: boolean;
  newLabel?: string;
  onRefresh: () => void;
  onNew: () => void;
}

export function ProviderHeaderCard({
  totalActive,
  totalResources,
  providerFamilies,
  updatedAtLabel,
  issueCount = 0,
  isFetching = false,
  isNewDisabled = false,
  newLabel,
  onRefresh,
  onNew,
}: ProviderHeaderCardProps) {
  const { t } = useTranslation();

  return (
    <section className="flex flex-col gap-3">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div className="min-w-0 flex flex-col gap-1.5">
          <h1 className="m-0 text-[28px] font-bold text-foreground">{t('providersPage.header.title')}</h1>
        </div>
        <div className="flex flex-wrap gap-2 items-center lg:justify-end">
          <button
            type="button"
            className="inline-flex items-center gap-1.5 h-8 px-3 text-[13px] font-medium cursor-pointer border border-border bg-background text-foreground hover:bg-secondary hover:border-border/80 disabled:opacity-60 disabled:cursor-not-allowed focus-visible:outline-2 focus-visible:outline-primary focus-visible:outline-offset-2 transition-colors"
            onClick={onRefresh}
            disabled={isFetching}
            aria-label={
              isFetching
                ? t('providersPage.actions.syncing')
                : t('providersPage.actions.refresh')
            }
          >
            <span className={`inline-flex items-center${isFetching ? ' animate-spin' : ''}`}>
              {isFetching ? <IconLoader2 size={14} /> : <IconRefreshCw size={14} />}
            </span>
            <span>
              {isFetching
                ? t('providersPage.actions.syncing')
                : t('providersPage.actions.refresh')}
            </span>
          </button>
          <button
            type="button"
            className="inline-flex items-center gap-1.5 h-8 px-3 text-[13px] font-medium cursor-pointer border border-transparent bg-primary text-primary-foreground hover:bg-[var(--primary-hover)] active:bg-[var(--primary-active)] disabled:opacity-60 disabled:cursor-not-allowed focus-visible:outline-2 focus-visible:outline-primary focus-visible:outline-offset-2 transition-colors"
            onClick={onNew}
            disabled={isNewDisabled}
          >
            <IconPlus size={14} />
            <span>{newLabel ?? t('providersPage.actions.new')}</span>
          </button>
        </div>
      </div>

      <div className="flex flex-wrap gap-2">
        <span className="inline-flex items-center gap-1.5 py-1 px-[10px] border border-primary/30 bg-primary/10 text-primary text-[12px] font-medium">
          {t('providersPage.header.activeResources', {
            active: totalActive,
            total: totalResources,
          })}
        </span>
        <span className="inline-flex items-center gap-1.5 py-1 px-[10px] border border-border bg-muted text-muted-foreground text-[12px] font-medium">
          {t('providersPage.header.providerFamilies', { count: providerFamilies })}
        </span>
        <span className="inline-flex items-center gap-1.5 py-1 px-[10px] border border-border bg-muted text-muted-foreground text-[12px] font-medium">
          {t('providersPage.header.updatedAt', { time: updatedAtLabel })}
        </span>
        {issueCount > 0 ? (
          <span className="inline-flex items-center gap-1.5 py-1 px-[10px] border border-[rgba(245,158,11,0.30)] bg-[rgba(245,158,11,0.10)] text-amber-700 text-[12px] font-medium">
            {t('providersPage.header.issueCount', { count: issueCount })}
          </span>
        ) : null}
      </div>
    </section>
  );
}
