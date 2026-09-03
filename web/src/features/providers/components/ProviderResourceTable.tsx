import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import {
  IconAlertTriangle,
  IconCheckCircle2,
  IconEye,
  IconPencil,
  IconTrash2,
} from '@/components/ui/icons';
import {
  AppTable as Table,
  AppTableBody as TableBody,
  AppTableCell as TableCell,
  AppTableHead as TableHead,
  AppTableHeader as TableHeader,
  AppTableRow as TableRow,
} from '@/components/ui/AppTable';
import { ToggleSwitch } from '@/components/ui/ToggleSwitch';
import { ProviderStatusBar } from '@/components/providers/ProviderStatusBar';
import {
  getOpenAIProviderRecentStatusData,
  getOpenAIProviderTotalStats,
  getProviderRecentStatusData,
  getProviderTotalStats,
  type ProviderRecentUsageMap,
} from '@/components/providers/utils';
import type { OpenAIProviderConfig } from '@/types';
import type { StatusBarData } from '@/utils/recentRequests';
import type { ProviderResource } from '../types';

interface ProviderResourceTableProps {
  resources: ProviderResource[];
  selectedId?: string | null;
  disableMutations?: boolean;
  usageByProvider?: ProviderRecentUsageMap;
  onView: (resource: ProviderResource) => void;
  onEdit: (resource: ProviderResource) => void;
  onDelete: (resource: ProviderResource) => void;
  onToggleDisabled?: (resource: ProviderResource, disabled: boolean) => void;
}

const columnWidths = ['18%', '18%', '6%', '14%', '24%', '20%'];

const resolveStatusBarData = (
  resource: ProviderResource,
  usageByProvider: ProviderRecentUsageMap
): StatusBarData => {
  if (resource.brand === 'openaiCompatibility') {
    return getOpenAIProviderRecentStatusData(
      resource.raw as OpenAIProviderConfig,
      usageByProvider
    );
  }
  return getProviderRecentStatusData(
    usageByProvider,
    resource.brand,
    resource.apiKey ?? undefined,
    resource.baseUrl ?? undefined
  );
};

const resolveTotalStats = (
  resource: ProviderResource,
  usageByProvider: ProviderRecentUsageMap
): { success: number; failure: number } => {
  if (resource.brand === 'openaiCompatibility') {
    return getOpenAIProviderTotalStats(
      resource.raw as OpenAIProviderConfig,
      usageByProvider
    );
  }
  return getProviderTotalStats(
    usageByProvider,
    resource.brand,
    resource.apiKey ?? undefined,
    resource.baseUrl ?? undefined
  );
};

export function ProviderResourceTable({
  resources,
  selectedId,
  disableMutations,
  usageByProvider,
  onView,
  onEdit,
  onDelete,
  onToggleDisabled,
}: ProviderResourceTableProps) {
  const { t } = useTranslation();

  const renderMetric = (key: string, label: string, value: number) => (
    <span key={key} className="inline-flex items-center gap-1 px-2 py-0.5 border border-border bg-muted text-[11px] leading-[1.4] whitespace-nowrap">
      <span className="text-muted-foreground">{label}</span>
      <span className="text-foreground font-semibold tabular-nums">{value}</span>
    </span>
  );

  const renderFlagTag = (key: string, label: string) => (
    <span key={key} className="inline-flex items-center px-2 py-0.5 border border-[var(--primary-30)] bg-[var(--primary-10)] text-primary text-[11px] font-medium leading-[1.4] whitespace-nowrap">
      {label}
    </span>
  );

  const renderModelsSummary = (r: ProviderResource) => {
    const items: ReactNode[] = [];
    if (r.brand === 'openaiCompatibility') {
      items.push(
        renderMetric('models', t('providersPage.table.metrics.models'), r.modelCount),
        renderMetric('keys', t('providersPage.table.metrics.keys'), r.apiKeyEntryCount),
        renderMetric('headers', t('providersPage.table.metrics.headers'), r.headerCount),
      );
    } else {
      items.push(
        renderMetric('models', t('providersPage.table.metrics.models'), r.modelCount),
        renderMetric('headers', t('providersPage.table.metrics.headers'), r.headerCount),
      );
      if (r.brand === 'codex' && r.flags.websockets) {
        items.push(renderFlagTag('ws', t('providersPage.table.websocketsTag')));
      }
      if (r.brand === 'claude' && r.flags.cloakEnabled) {
        items.push(renderFlagTag('cloak', t('providersPage.table.cloakTag')));
      }
    }
    return <div className="flex flex-wrap gap-1.5 items-center">{items}</div>;
  };

  const renderStatus = (r: ProviderResource) => {
    if (r.disabled) {
      return (
        <span className="inline-flex items-center gap-1 px-2 py-0.5 border border-warning/30 bg-warning/12 text-warning text-[11px] font-medium whitespace-nowrap">
          <IconAlertTriangle size={12} />
          {t('providersPage.status.disabled')}
        </span>
      );
    }
    return (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 border border-success/30 bg-success/12 text-success text-[11px] font-medium whitespace-nowrap">
        <IconCheckCircle2 size={12} />
        {t('providersPage.status.active')}
      </span>
    );
  };

  const renderPrimary = (r: ProviderResource) => {
    if (r.brand === 'openaiCompatibility') {
      const extra = r.apiKeyEntryCount > 1 ? ` · +${r.apiKeyEntryCount - 1}` : '';
      return (
        <div className="flex flex-col gap-1 min-w-0">
          <span className="font-medium text-foreground overflow-hidden text-ellipsis whitespace-nowrap max-w-[220px]">{r.name ?? r.identifier}</span>
          <span className="text-[11px] text-muted-foreground font-mono overflow-hidden text-ellipsis whitespace-nowrap max-w-[220px]">
            {(r.apiKeyPreview ?? '—') + extra}
          </span>
        </div>
      );
    }
    return (
      <div className="flex flex-col gap-1 min-w-0">
        <span className="font-medium text-foreground overflow-hidden text-ellipsis whitespace-nowrap max-w-[220px]">{r.apiKeyPreview ?? '—'}</span>
        {r.authIndex ? (
          <span className="text-[11px] text-muted-foreground font-mono overflow-hidden text-ellipsis whitespace-nowrap max-w-[220px]">auth: {r.authIndex}</span>
        ) : null}
      </div>
    );
  };

  const renderBaseUrl = (r: ProviderResource) => {
    if (r.brand === 'claude' && !r.baseUrl) {
      return (
        <span className="font-mono text-[11px] text-muted-foreground overflow-hidden text-ellipsis whitespace-nowrap max-w-[220px] block">
          https://api.anthropic.com {t('providersPage.status.defaultSuffix')}
        </span>
      );
    }
    return (
      <span className="font-mono text-[11px] text-muted-foreground overflow-hidden text-ellipsis whitespace-nowrap max-w-[220px] block">
        {r.baseUrl ?? t('providersPage.status.notSet')}
      </span>
    );
  };

  return (
    <Table
      cols={columnWidths.map((w, i) => (
        <col key={i} style={{ width: w }} />
      ))}
    >
      <TableHeader>
        <TableRow>
          <TableHead>{t('providersPage.table.key')}</TableHead>
          <TableHead>{t('providersPage.table.baseUrl')}</TableHead>
          <TableHead>{t('providersPage.table.prefix')}</TableHead>
          <TableHead>{t('providersPage.table.models')}</TableHead>
          <TableHead>{t('providersPage.table.status')}</TableHead>
          <TableHead alignRight>{t('providersPage.table.actions')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {resources.map((resource) => {
          return (
            <TableRow key={resource.id} selected={resource.id === selectedId}>
              <TableCell>{renderPrimary(resource)}</TableCell>
              <TableCell>{renderBaseUrl(resource)}</TableCell>
              <TableCell>
                {resource.prefix ? (
                  <span className="inline-flex items-center px-2 py-0.5 border border-border bg-muted text-muted-foreground text-[11px]">{resource.prefix}</span>
                ) : (
                  <span className="font-mono text-[11px] text-muted-foreground">{t('providersPage.status.none')}</span>
                )}
              </TableCell>
              <TableCell>{renderModelsSummary(resource)}</TableCell>
              <TableCell>
                <div className="flex flex-col gap-1.5 items-start min-w-0">
                  {renderStatus(resource)}
                  {usageByProvider ? (
                    <>
                      {(() => {
                        const stats = resolveTotalStats(resource, usageByProvider);
                        return (
                          <div className="flex flex-wrap gap-1.5">
                            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-semibold leading-[1.4] border whitespace-nowrap tabular-nums bg-success/12 text-success border-success/30">
                              {t('stats.success')}: {stats.success}
                            </span>
                            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-semibold leading-[1.4] border whitespace-nowrap tabular-nums bg-destructive/10 text-destructive border-destructive/30">
                              {t('stats.failure')}: {stats.failure}
                            </span>
                          </div>
                        );
                      })()}
                      <ProviderStatusBar
                        statusData={resolveStatusBarData(resource, usageByProvider)}
                      />
                    </>
                  ) : null}
                </div>
              </TableCell>
              <TableCell alignRight>
                <div className="flex gap-1 items-center justify-end">
                  {onToggleDisabled ? (
                    <span
                      className="inline-flex items-center mr-1"
                      onClick={(e) => e.stopPropagation()}
                    >
                      <ToggleSwitch
                        checked={!resource.disabled}
                        disabled={disableMutations}
                        onChange={(value) =>
                          onToggleDisabled(resource, !value)
                        }
                        ariaLabel={
                          resource.disabled
                            ? t('providersPage.actions.enable')
                            : t('providersPage.actions.disable')
                        }
                      />
                    </span>
                  ) : null}
                  <button
                    type="button"
                    className="inline-flex items-center justify-center w-7 h-7 p-0 border border-transparent bg-transparent text-muted-foreground cursor-pointer hover:bg-secondary hover:text-foreground disabled:opacity-50 disabled:cursor-not-allowed focus-visible:outline-2 focus-visible:outline-primary focus-visible:-outline-offset-1"
                    aria-label={t('providersPage.actions.view')}
                    title={t('providersPage.actions.view')}
                    onClick={(e) => {
                      e.stopPropagation();
                      onView(resource);
                    }}
                  >
                    <IconEye size={14} />
                  </button>
                  <button
                    type="button"
                    className="inline-flex items-center justify-center w-7 h-7 p-0 border border-transparent bg-transparent text-muted-foreground cursor-pointer hover:bg-secondary hover:text-foreground disabled:opacity-50 disabled:cursor-not-allowed focus-visible:outline-2 focus-visible:outline-primary focus-visible:-outline-offset-1"
                    aria-label={t('providersPage.actions.edit')}
                    title={t('providersPage.actions.edit')}
                    disabled={disableMutations}
                    onClick={(e) => {
                      e.stopPropagation();
                      onEdit(resource);
                    }}
                  >
                    <IconPencil size={14} />
                  </button>
                  <button
                    type="button"
                    className="inline-flex items-center justify-center w-7 h-7 p-0 border border-transparent bg-transparent text-destructive cursor-pointer hover:bg-[var(--destructive-10)] hover:text-destructive disabled:opacity-50 disabled:cursor-not-allowed focus-visible:outline-2 focus-visible:outline-primary focus-visible:-outline-offset-1"
                    aria-label={t('providersPage.actions.delete')}
                    title={t('providersPage.actions.delete')}
                    disabled={disableMutations}
                    onClick={(e) => {
                      e.stopPropagation();
                      onDelete(resource);
                    }}
                  >
                    <IconTrash2 size={14} />
                  </button>
                </div>
              </TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}
