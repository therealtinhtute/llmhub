import { useTranslation } from 'react-i18next';
import claudeLogo from '@/assets/icons/claude.svg';
import codexLogo from '@/assets/icons/codex.svg';
import geminiLogo from '@/assets/icons/gemini.svg';
import openaiLogo from '@/assets/icons/openai-light.svg';
import vertexLogo from '@/assets/icons/vertex.svg';
import opencodeLightLogo from '@/assets/icons/opencode-light.svg';
import opencodeDarkLogo from '@/assets/icons/opencode-dark.svg';
import openrouterLightLogo from '@/assets/icons/openrouter-light.svg';
import openrouterDarkLogo from '@/assets/icons/openrouter-dark.svg';
import nvidiaLightLogo from '@/assets/icons/nvidia-light.svg';
import nvidiaDarkLogo from '@/assets/icons/nvidia-dark.svg';
import { IconAlertTriangle } from '@/components/ui/icons';
import { useThemeStore } from '@/stores';
import type { OAuthProvider } from '@/services/api/oauth';
import type { ProviderBrand } from '../types';
import { getOAuthIcon, type ProviderEntry, type ProviderEntryCategory } from '../entries';

const CATEGORY_ORDER: ProviderEntryCategory[] = ['oauth', 'free', 'freeTier', 'apikey', 'custom'];

type LogoSource = string | { light: string; dark: string };
interface LogoAsset {
  src: LogoSource;
  invertOnDark?: boolean;
}

const resolveLogoSource = (source: LogoSource, theme: 'light' | 'dark'): string =>
  typeof source === 'string' ? source : theme === 'dark' ? source.dark : source.light;

const CONFIG_LOGOS: Record<ProviderBrand, LogoAsset> = {
  gemini: { src: geminiLogo },
  claude: { src: claudeLogo },
  codex: { src: codexLogo },
  vertex: { src: vertexLogo },
  openaiCompatibility: { src: openaiLogo, invertOnDark: true },
  openrouter: { src: openrouterLightLogo },
  opencode: { src: opencodeLightLogo },
};

const PRESET_LOGOS: Record<string, LogoAsset> = {
  opencode: { src: { light: opencodeLightLogo, dark: opencodeDarkLogo } },
  openrouter: { src: { light: openrouterLightLogo, dark: openrouterDarkLogo } },
  nvidia: { src: { light: nvidiaLightLogo, dark: nvidiaDarkLogo } },
};

interface ProviderCategoryGridProps {
  entries: ProviderEntry[];
  onOpen: (key: string) => void;
  oauthPendingStatus?: Partial<Record<OAuthProvider, string>>;
}

export function ProviderCategoryGrid({
  entries,
  onOpen,
  oauthPendingStatus,
}: ProviderCategoryGridProps) {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col gap-6">
      {CATEGORY_ORDER.map((category) => {
        const items = entries.filter((entry) => entry.category === category);
        if (items.length === 0) return null;
        return (
          <section key={category} className="min-w-0">
            <p className="mx-2 mt-1 mb-2 text-[11px] font-medium tracking-[0.05em] uppercase text-muted-foreground">
              {t(`providersPage.categories.sections.${category}`)}
            </p>
            <div className="grid grid-cols-[repeat(auto-fit,minmax(240px,1fr))] gap-3 max-md:grid-cols-[minmax(0,1fr)]">
              {items.map((entry) => (
                <ProviderEntryCard
                  key={entry.key}
                  entry={entry}
                  onOpen={onOpen}
                  pendingStatus={
                    entry.kind === 'oauth' ? oauthPendingStatus?.[entry.oauthId] : undefined
                  }
                />
              ))}
            </div>
          </section>
        );
      })}
    </div>
  );
}

function ProviderEntryCard({
  entry,
  onOpen,
  pendingStatus,
}: {
  entry: ProviderEntry;
  onOpen: (key: string) => void;
  pendingStatus?: string;
}) {
  const { t } = useTranslation();
  const resolvedTheme = useThemeStore((s) => s.resolvedTheme);

  let name: string;
  let logoSrc: string | null = null;
  let invertOnDark = false;
  let count: number;
  let statusLabel: string;
  let statusTone: 'success' | 'warning' | 'error' | 'neutral' = 'neutral';
  let showAmberCount = false;
  let warning = false;
  let warningLabel = '';

  if (entry.kind === 'oauth') {
    name = t(entry.titleKey);
    logoSrc = getOAuthIcon(entry.icon, resolvedTheme);
    count = entry.accountCount;
    warning = entry.hasIssue;
    warningLabel = t('providersPage.table.providerIssue');
    statusLabel = pendingStatus ??
      (entry.hasIssue
        ? t('providersPage.status.error')
        : count > 0
          ? t('providersPage.status.active')
          : t('providersPage.status.notConfigured'));
    statusTone = pendingStatus || count === 0 ? 'warning' : entry.hasIssue ? 'error' : 'success';
    showAmberCount = count === 0;
  } else if (entry.kind === 'preset') {
    name = entry.preset.displayName;
    const logo = PRESET_LOGOS[entry.preset.id];
    logoSrc = logo ? resolveLogoSource(logo.src, resolvedTheme) : null;
    invertOnDark = Boolean(logo?.invertOnDark);
    count = entry.resources.length;
    statusLabel = entry.preset.verified
      ? t('providersPage.presets.verified')
      : t('providersPage.presets.unverified');
    statusTone = entry.preset.verified ? 'success' : 'warning';
    showAmberCount = count === 0;
    warning = !entry.preset.verified;
    warningLabel = t('providersPage.presets.unverified');
  } else {
    const realResources = entry.resources;
    const activeResourceCount = realResources.filter((r) => !r.disabled).length;
    count = realResources.length;
    const logo = CONFIG_LOGOS[entry.group.id];
    logoSrc = logo ? resolveLogoSource(logo.src, resolvedTheme) : null;
    invertOnDark = Boolean(logo?.invertOnDark);
    name = t(`providersPage.providerNames.${entry.group.id}`);
    warning = Boolean(entry.group.issue);
    warningLabel = t('providersPage.table.providerIssue');
    statusLabel = entry.group.issue
      ? t('providersPage.status.error')
      : count === 0
        ? t('providersPage.status.notConfigured')
        : activeResourceCount > 0
          ? t('providersPage.status.active')
          : t('providersPage.status.disabled');
    statusTone = entry.group.issue
      ? 'error'
      : count === 0 || activeResourceCount === 0
        ? 'warning'
        : 'success';
    showAmberCount = count === 0;
  }

  const statusClassName = [
    'inline-flex w-fit items-center border px-2 py-0.5 text-[11px] font-medium',
    statusTone === 'success'
      ? 'border-emerald-400/40 bg-emerald-100 text-emerald-700'
      : statusTone === 'warning'
        ? 'border-amber-400/40 bg-amber-100 text-amber-700'
        : statusTone === 'error'
          ? 'border-destructive/30 bg-destructive/10 text-destructive'
          : 'border-border bg-muted text-muted-foreground',
  ].join(' ');

  return (
    <button
      type="button"
      className={[
        'flex items-center justify-between gap-3 p-3 border text-left w-full cursor-pointer min-w-0',
        'border-border bg-background hover:bg-[color-mix(in_srgb,var(--accent-bg)_50%,transparent)] hover:border-[var(--primary-30)]',
        'focus-visible:outline-2 focus-visible:outline-primary focus-visible:-outline-offset-2',
      ].join(' ')}
      onClick={() => onOpen(entry.key)}
    >
      <span className="flex items-center gap-[10px] min-w-0 flex-1">
        {logoSrc ? (
          <img
            src={logoSrc}
            alt=""
            aria-hidden="true"
            className={[
              'w-6 h-6 flex-shrink-0 object-contain bg-secondary p-0.5',
              invertOnDark ? '[data-theme=dark]_&:invert [data-theme=dark]_&:hue-rotate-180' : '',
            ].join(' ')}
          />
        ) : null}
        <span className="flex flex-col min-w-0 gap-1">
          <span className="text-[13px] font-medium whitespace-nowrap overflow-hidden text-ellipsis">
            {name}
          </span>
          <span className={statusClassName}>{statusLabel}</span>
        </span>
      </span>
      <span className="flex items-center gap-1.5 shrink-0">
        {warning ? (
          <span title={warningLabel} aria-label={warningLabel}>
            <IconAlertTriangle size={16} className="text-amber-700" />
          </span>
        ) : null}
        <span
          className={[
            'inline-flex items-center justify-center min-w-6 px-2 py-0.5 text-[11px] font-medium border flex-shrink-0',
            showAmberCount
              ? 'bg-[rgba(245,158,11,0.10)] border-[rgba(245,158,11,0.30)] text-amber-700'
              : 'bg-background border-border text-muted-foreground',
          ].join(' ')}
        >
          {count}
        </span>
      </span>
    </button>
  );
}
