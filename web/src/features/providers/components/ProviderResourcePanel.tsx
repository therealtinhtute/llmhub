import { useTranslation } from 'react-i18next';
import opencodeLogo from '@/assets/icons/opencode-light.svg';
import openrouterLogo from '@/assets/icons/openrouter-light.svg';
import claudeLogo from '@/assets/icons/claude.svg';
import codexLogo from '@/assets/icons/codex.svg';
import geminiLogo from '@/assets/icons/gemini.svg';
import openaiLogo from '@/assets/icons/openai-light.svg';
import vertexLogo from '@/assets/icons/vertex.svg';
import { IconPlus, IconSearch } from '@/components/ui/icons';
import type { ProviderRecentUsageMap } from '@/components/providers/utils';
import type { ProviderBrand, ProviderGroup, ProviderResource } from '../types';
import { ProviderResourceTable } from './ProviderResourceTable';
import {
  OpenAIBrandToolbar,
  type OpenAISortBy,
  type SortDir,
} from './OpenAIBrandToolbar';

const LOGOS: Record<ProviderBrand, { src: string; invertOnDark?: boolean }> = {
  gemini: { src: geminiLogo },
  claude: { src: claudeLogo },
  codex: { src: codexLogo },
  vertex: { src: vertexLogo },
  openaiCompatibility: { src: openaiLogo, invertOnDark: true },
  openrouter: { src: openrouterLogo },
  opencode: { src: opencodeLogo },
};

export interface OpenAIPanelControls {
  sortBy: OpenAISortBy;
  sortDir: SortDir;
  onSortBy: (value: OpenAISortBy) => void;
  onSortDir: (value: SortDir) => void;
  availableModels: ReadonlyArray<string>;
  selectedModels: ReadonlySet<string>;
  onSelectedModelsChange: (next: Set<string>) => void;
}

interface ProviderResourcePanelProps {
  group: ProviderGroup;
  filter: string;
  onFilterChange: (value: string) => void;
  filteredResources: ProviderResource[];
  selectedId: string | null;
  disableMutations?: boolean;
  usageByProvider?: ProviderRecentUsageMap;
  openaiControls?: OpenAIPanelControls;
  onView: (resource: ProviderResource) => void;
  onEdit: (resource: ProviderResource) => void;
  onDelete: (resource: ProviderResource) => void;
  onToggleDisabled?: (resource: ProviderResource, disabled: boolean) => void;
  onCreate: () => void;
}

export function ProviderResourcePanel({
  group,
  filter,
  onFilterChange,
  filteredResources,
  selectedId,
  disableMutations,
  usageByProvider,
  openaiControls,
  onView,
  onEdit,
  onDelete,
  onToggleDisabled,
  onCreate,
}: ProviderResourcePanelProps) {
  const { t } = useTranslation();
  const logo = LOGOS[group.id];

  const realResources = filteredResources;

  return (
    <section className="bg-background border border-border shadow p-5 min-w-0 flex flex-col gap-4">
      <div className="flex flex-col gap-3">
        <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
          <div className="min-w-0 flex flex-col">
            <div className="flex items-center gap-3">
              {logo ? (
                <img
                  src={logo.src}
                  alt=""
                  aria-hidden="true"
                  className="w-6 h-6 object-contain flex-shrink-0"
                />
              ) : null}
              <h2 className="text-[20px] font-semibold text-foreground m-0 tracking-[-0.005em]">
                {t(`providersPage.providerNames.${group.id}`)}
              </h2>
            </div>
          </div>
          <div className="relative min-w-0 md:w-[280px]">
            <span className="absolute top-1/2 left-3 -translate-y-1/2 text-muted-foreground pointer-events-none inline-flex" aria-hidden="true">
              <IconSearch size={14} />
            </span>
            <input
              type="search"
              className="w-full h-9 pl-9 pr-3 border border-border bg-background text-foreground text-[13px] box-border placeholder:text-[var(--text-tertiary)] focus:outline-none focus:border-primary"
              value={filter}
              onChange={(event) => onFilterChange(event.target.value)}
              placeholder={t('providersPage.table.filterPlaceholder')}
            />
          </div>
        </div>
        {openaiControls ? (
          <div className="flex justify-end items-center flex-wrap gap-2">
            <OpenAIBrandToolbar
              sortBy={openaiControls.sortBy}
              sortDir={openaiControls.sortDir}
              onSortBy={openaiControls.onSortBy}
              onSortDir={openaiControls.onSortDir}
              availableModels={openaiControls.availableModels}
              selectedModels={openaiControls.selectedModels}
              onSelectedModelsChange={openaiControls.onSelectedModelsChange}
            />
          </div>
        ) : null}
      </div>

      {group.issue ? (
        <div className="border border-[rgba(245,158,11,0.30)] bg-[rgba(245,158,11,0.10)] p-3 px-[14px] text-[13px] text-amber-700 leading-[1.5]">
          <div className="font-semibold mb-1">
            {t('providersPage.table.providerIssue')}
            {group.issue.status ? ` · ${group.issue.status}` : ''}
          </div>
          <div>{group.issue.message}</div>
        </div>
      ) : null}

      {realResources.length === 0 ? (
        <div className="border border-dashed border-border p-8 text-center text-muted-foreground text-[13px]">
          <div>{t('providersPage.table.empty')}</div>
          <div className="mt-3">
            <button
              type="button"
              onClick={onCreate}
              className="inline-flex items-center gap-1.5 py-1.5 px-3 border border-border bg-background cursor-pointer text-[13px]"
            >
              <IconPlus size={14} />
              <span>{t('providersPage.actions.new')}</span>
            </button>
          </div>
        </div>
      ) : (
        <ProviderResourceTable
          resources={filteredResources}
          selectedId={selectedId}
          disableMutations={disableMutations}
          usageByProvider={usageByProvider}
          onView={onView}
          onEdit={onEdit}
          onDelete={onDelete}
          onToggleDisabled={onToggleDisabled}
        />
      )}
    </section>
  );
}
