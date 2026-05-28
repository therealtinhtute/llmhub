import { useTranslation } from 'react-i18next';
import ampcodeLogo from '@/assets/icons/amp.svg';
import claudeLogo from '@/assets/icons/claude.svg';
import codexLogo from '@/assets/icons/codex.svg';
import geminiLogo from '@/assets/icons/gemini.svg';
import openaiLogo from '@/assets/icons/openai-light.svg';
import vertexLogo from '@/assets/icons/vertex.svg';
import { IconAlertTriangle } from '@/components/ui/icons';
import type { ProviderBrand, ProviderGroup } from '../types';

const PROVIDER_LOGOS: Record<ProviderBrand, { src: string; invertOnDark?: boolean }> = {
  gemini: { src: geminiLogo },
  claude: { src: claudeLogo },
  codex: { src: codexLogo },
  vertex: { src: vertexLogo },
  openaiCompatibility: { src: openaiLogo, invertOnDark: true },
  ampcode: { src: ampcodeLogo },
};

interface ProviderCategoryListProps {
  groups: ProviderGroup[];
  activeBrand: ProviderBrand;
  onSelect: (brand: ProviderBrand) => void;
}

export function ProviderCategoryList({
  groups,
  activeBrand,
  onSelect,
}: ProviderCategoryListProps) {
  const { t } = useTranslation();

  return (
    <aside className="bg-background border border-border shadow p-3 min-w-0">
      <p className="mx-2 mt-1 mb-2 text-[11px] font-medium tracking-[0.05em] uppercase text-muted-foreground">
        {t('providersPage.categories.title')}
      </p>
      <div className="flex flex-col gap-1">
        {groups.map((group) => {
          const active = group.id === activeBrand;
          const realResources = group.resources.filter(
            (r) => !r.flags.isPlaceholder
          );
          const total = realResources.length || (group.id === 'ampcode' ? 1 : 0);
          const activeCount = realResources.filter((r) => !r.disabled).length;
          const logo = PROVIDER_LOGOS[group.id];

          return (
            <button
              key={group.id}
              type="button"
              className={[
                'flex items-center justify-between gap-3 px-3 py-[10px] border text-left w-full cursor-pointer min-w-0 transition-colors',
                active
                  ? 'border-[var(--primary-30)] bg-[var(--primary-10)] text-primary'
                  : 'border-transparent bg-transparent text-foreground hover:bg-[color-mix(in_srgb,var(--accent-bg)_50%,transparent)] hover:border-border',
                'focus-visible:outline-2 focus-visible:outline-primary focus-visible:-outline-offset-2',
              ].join(' ')}
              onClick={() => onSelect(group.id)}
              aria-current={active ? 'page' : undefined}
            >
              <span className="flex items-center gap-[10px] min-w-0 flex-1">
                {logo ? (
                  <img
                    src={logo.src}
                    alt=""
                    aria-hidden="true"
                    className={[
                      'w-6 h-6 flex-shrink-0 object-contain bg-secondary p-0.5',
                      logo.invertOnDark ? '[data-theme=dark]_&:invert [data-theme=dark]_&:hue-rotate-180' : '',
                    ].join(' ')}
                  />
                ) : null}
                <span className="flex flex-col min-w-0">
                  <span className="text-[13px] font-medium whitespace-nowrap overflow-hidden text-ellipsis">
                    {t(`providersPage.providerNames.${group.id}`)}
                  </span>
                  <span className={[
                    'text-[11px] mt-0.5 font-normal',
                    active ? 'text-primary opacity-85' : 'text-muted-foreground',
                  ].join(' ')}>
                    {group.id === 'ampcode'
                      ? t(
                          group.resources[0]?.disabled
                            ? 'providersPage.categories.ampcodeInactive'
                            : 'providersPage.categories.ampcodeActive'
                        )
                      : t('providersPage.categories.activeCount', {
                          active: activeCount,
                          total,
                        })}
                  </span>
                </span>
              </span>
              {group.issue ? (
                <IconAlertTriangle
                  size={16}
                  className="text-amber-700 dark:text-amber-400 shrink-0"
                />
              ) : (
                <span
                  className={[
                    'inline-flex items-center justify-center min-w-6 px-2 py-0.5 text-[11px] font-medium border flex-shrink-0',
                    group.id !== 'ampcode' && total === 0
                      ? 'bg-[rgba(245,158,11,0.10)] border-[rgba(245,158,11,0.30)] text-amber-700 dark:text-amber-400'
                      : 'bg-background border-border text-muted-foreground',
                  ].join(' ')}
                >
                  {group.id === 'ampcode' ? (group.resources[0]?.disabled ? '—' : '1') : total}
                </span>
              )}
            </button>
          );
        })}
      </div>
    </aside>
  );
}
