import { useTranslation } from 'react-i18next';
import { DetailsCollapsible as Collapsible } from '@/components/ui/DetailsCollapsible';
import { IconCheck, IconX } from '@/components/ui/icons';
import {
  getProviderTotalStats,
  type ProviderRecentUsageMap,
} from '@/components/providers/utils';
import type { OpenAIProviderConfig } from '@/types';
import { maskApiKey } from '@/utils/format';
import type { ProviderResource } from '../types';

interface ResourceDetailViewProps {
  resource: ProviderResource;
  usageByProvider?: ProviderRecentUsageMap;
}

export function ResourceDetailView({ resource, usageByProvider }: ResourceDetailViewProps) {
  const { t } = useTranslation();

  const primary: Array<[string, string]> = [
    ['identifier', resource.identifier],
    ['baseUrl', resource.baseUrl ?? t('providersPage.status.notSet')],
    ['proxyUrl', resource.proxyUrl ?? t('providersPage.status.notSet')],
    ['prefix', resource.prefix ?? t('providersPage.status.none')],
    ['models', String(resource.modelCount)],
    ['headers', String(resource.headerCount)],
  ];

  const metadata: Array<[string, string]> = [
    ['authIndex', resource.authIndex ?? t('providersPage.status.notSet')],
    ['excludedModels', String(resource.excludedModelCount)],
    ['apiKeyEntries', String(resource.apiKeyEntryCount)],
  ];

  const openaiConfig =
    resource.brand === 'openaiCompatibility'
      ? (resource.raw as OpenAIProviderConfig)
      : null;
  const apiKeyEntries = openaiConfig?.apiKeyEntries ?? [];

  return (
    <div>
      <div className="mb-1">
        <div className="text-[13px] font-semibold text-foreground m-0 flex items-center gap-2">
          {resource.name ?? resource.identifier}
        </div>
      </div>

      <dl className="grid grid-cols-2 gap-x-4 gap-y-3 m-0">
        {primary.map(([key, value]) => (
          <div key={key}>
            <dt className="text-[11px] text-muted-foreground font-medium">{t(`providersPage.detail.fields.${key}`)}</dt>
            <dd className="m-0 mt-0.5 text-[12px] font-mono text-foreground break-words">{value}</dd>
          </div>
        ))}
      </dl>

      {openaiConfig && apiKeyEntries.length > 0 ? (
        <div className="mt-4 min-w-0 max-w-full">
          <div className="text-[12px] font-semibold text-muted-foreground mb-1.5">
            {t('providersPage.form.apiKeyEntriesSection')}: {apiKeyEntries.length}
          </div>
          <div className="flex flex-col gap-1.5 min-w-0 max-w-full">
            {apiKeyEntries.map((entry, entryIndex) => {
              const entryStats = usageByProvider
                ? getProviderTotalStats(
                    usageByProvider,
                    openaiConfig.name,
                    entry.apiKey,
                    openaiConfig.baseUrl
                  )
                : { success: 0, failure: 0 };
              return (
                <div
                  key={`${entry.apiKey}-${entryIndex}`}
                  className="flex flex-wrap items-center gap-2 min-w-0 max-w-full px-3 py-2 bg-muted border border-[var(--border-secondary)] text-[12px]"
                >
                  <span className="inline-flex items-center justify-center w-5 h-5 rounded-full bg-primary text-primary-foreground text-[11px] font-semibold flex-shrink-0">
                    {entryIndex + 1}
                  </span>
                  <span className="font-mono font-semibold text-foreground min-w-0 max-w-full overflow-wrap-anywhere break-all">
                    {maskApiKey(entry.apiKey)}
                  </span>
                  {entry.proxyUrl ? (
                    <span className="text-[var(--text-tertiary)] text-[11px] min-w-0 max-w-full overflow-wrap-anywhere break-words before:content-['|_Proxy:_'] before:text-[var(--text-quaternary)]">
                      {entry.proxyUrl}
                    </span>
                  ) : null}
                  <div className="flex gap-1.5 ml-auto">
                    <span className="inline-flex items-center gap-[3px] px-1.5 py-0.5 rounded-[10px] text-[10px] font-semibold bg-success/12 text-success">
                      <IconCheck size={12} /> {entryStats.success}
                    </span>
                    <span className="inline-flex items-center gap-[3px] px-1.5 py-0.5 rounded-[10px] text-[10px] font-semibold bg-destructive/10 text-destructive">
                      <IconX size={12} /> {entryStats.failure}
                    </span>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      ) : null}

      <div style={{ marginTop: 16 }}>
        <Collapsible label={t('providersPage.detail.metadataTitle')}>
          <dl className="grid grid-cols-2 gap-x-4 gap-y-3 m-0">
            {metadata.map(([key, value]) => (
              <div key={key}>
                <dt className="text-[11px] text-muted-foreground font-medium">{t(`providersPage.detail.fields.${key}`)}</dt>
                <dd className="m-0 mt-0.5 text-[12px] font-mono text-foreground break-words">{value}</dd>
              </div>
            ))}
          </dl>
        </Collapsible>
      </div>
    </div>
  );
}
