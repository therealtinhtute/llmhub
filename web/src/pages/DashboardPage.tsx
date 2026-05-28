import { useCallback, useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  IconKey,
  IconBot,
  IconFileText,
  IconSatellite
} from '@/components/ui/icons';
import { useAuthStore, useConfigStore, useModelsStore } from '@/stores';
import { apiKeysApi, providersApi, authFilesApi } from '@/services/api';

interface QuickStat {
  label: string;
  value: number | string;
  icon: React.ReactNode;
  path: string;
  loading?: boolean;
  sublabel?: string;
}

interface ProviderStats {
  gemini: number | null;
  codex: number | null;
  claude: number | null;
  openai: number | null;
}

type TimeOfDay = 'morning' | 'afternoon' | 'evening' | 'night';

function getTimeOfDay(): TimeOfDay {
  const hour = new Date().getHours();
  if (hour >= 5 && hour < 12) return 'morning';
  if (hour >= 12 && hour < 17) return 'afternoon';
  if (hour >= 17 && hour < 21) return 'evening';
  return 'night';
}

export function DashboardPage() {
  const { t, i18n } = useTranslation();
  const connectionStatus = useAuthStore((state) => state.connectionStatus);
  const serverVersion = useAuthStore((state) => state.serverVersion);
  const serverBuildDate = useAuthStore((state) => state.serverBuildDate);
  const apiBase = useAuthStore((state) => state.apiBase);
  const config = useConfigStore((state) => state.config);

  const models = useModelsStore((state) => state.models);
  const modelsLoading = useModelsStore((state) => state.loading);
  const fetchModelsFromStore = useModelsStore((state) => state.fetchModels);

  const [stats, setStats] = useState<{
    apiKeys: number | null;
    authFiles: number | null;
  }>({
    apiKeys: null,
    authFiles: null
  });

  const [providerStats, setProviderStats] = useState<ProviderStats>({
    gemini: null,
    codex: null,
    claude: null,
    openai: null
  });

  const [loading, setLoading] = useState(true);

  // Time-of-day state for dynamic greeting
  const [timeOfDay, setTimeOfDay] = useState<TimeOfDay>(getTimeOfDay);
  const [currentTime, setCurrentTime] = useState(() => new Date());

  const apiKeysCache = useRef<string[]>([]);

  useEffect(() => {
    apiKeysCache.current = [];
  }, [apiBase, config?.apiKeys]);

  // Update time every 60 seconds
  useEffect(() => {
    const id = setInterval(() => {
      setTimeOfDay(getTimeOfDay());
      setCurrentTime(new Date());
    }, 60_000);
    return () => clearInterval(id);
  }, []);

  const normalizeApiKeyList = (input: unknown): string[] => {
    if (!Array.isArray(input)) return [];
    const seen = new Set<string>();
    const keys: string[] = [];

    input.forEach((item) => {
      const record =
        item !== null && typeof item === 'object' && !Array.isArray(item)
          ? (item as Record<string, unknown>)
          : null;
      const value =
        typeof item === 'string'
          ? item
          : record
            ? (record['api-key'] ?? record['apiKey'] ?? record.key ?? record.Key)
            : '';
      const trimmed = String(value ?? '').trim();
      if (!trimmed || seen.has(trimmed)) return;
      seen.add(trimmed);
      keys.push(trimmed);
    });

    return keys;
  };

  const resolveApiKeysForModels = useCallback(async () => {
    if (apiKeysCache.current.length) {
      return apiKeysCache.current;
    }

    const configKeys = normalizeApiKeyList(config?.apiKeys);
    if (configKeys.length) {
      apiKeysCache.current = configKeys;
      return configKeys;
    }

    try {
      const list = await apiKeysApi.list();
      const normalized = normalizeApiKeyList(list);
      if (normalized.length) {
        apiKeysCache.current = normalized;
      }
      return normalized;
    } catch {
      return [];
    }
  }, [config?.apiKeys]);

  const fetchModels = useCallback(async () => {
    if (connectionStatus !== 'connected' || !apiBase) {
      return;
    }

    try {
      const apiKeys = await resolveApiKeysForModels();
      const primaryKey = apiKeys[0];
      await fetchModelsFromStore(apiBase, primaryKey);
    } catch {
      // Ignore model fetch errors on dashboard
    }
  }, [connectionStatus, apiBase, resolveApiKeysForModels, fetchModelsFromStore]);

  useEffect(() => {
    const fetchStats = async () => {
      setLoading(true);
      try {
        const [keysRes, filesRes, geminiRes, codexRes, claudeRes, openaiRes] = await Promise.allSettled([
          apiKeysApi.list(),
          authFilesApi.list(),
          providersApi.getGeminiKeys(),
          providersApi.getCodexConfigs(),
          providersApi.getClaudeConfigs(),
          providersApi.getOpenAIProviders()
        ]);

        setStats({
          apiKeys: keysRes.status === 'fulfilled' ? keysRes.value.length : null,
          authFiles: filesRes.status === 'fulfilled' ? filesRes.value.files.length : null
        });

        setProviderStats({
          gemini: geminiRes.status === 'fulfilled' ? geminiRes.value.length : null,
          codex: codexRes.status === 'fulfilled' ? codexRes.value.length : null,
          claude: claudeRes.status === 'fulfilled' ? claudeRes.value.length : null,
          openai: openaiRes.status === 'fulfilled' ? openaiRes.value.length : null
        });
      } finally {
        setLoading(false);
      }
    };

    if (connectionStatus === 'connected') {
      fetchStats();
      fetchModels();
    } else {
      setLoading(false);
    }
  }, [connectionStatus, fetchModels]);

  // Calculate total provider keys only when all provider stats are available.
  const providerStatsReady =
    providerStats.gemini !== null &&
    providerStats.codex !== null &&
    providerStats.claude !== null &&
    providerStats.openai !== null;
  const hasProviderStats =
    providerStats.gemini !== null ||
    providerStats.codex !== null ||
    providerStats.claude !== null ||
    providerStats.openai !== null;
  const totalProviderKeys = providerStatsReady
    ? (providerStats.gemini ?? 0) +
      (providerStats.codex ?? 0) +
      (providerStats.claude ?? 0) +
      (providerStats.openai ?? 0)
    : 0;

  const quickStats: QuickStat[] = [
    {
      label: t('dashboard.management_keys'),
      value: stats.apiKeys ?? '-',
      icon: <IconKey size={24} />,
      path: '/config',
      loading: loading && stats.apiKeys === null,
      sublabel: t('nav.config_management')
    },
    {
      label: t('nav.ai_providers'),
      value: loading ? '-' : providerStatsReady ? totalProviderKeys : '-',
      icon: <IconBot size={24} />,
      path: '/ai-providers',
      loading: loading,
      sublabel: hasProviderStats
        ? t('dashboard.provider_keys_detail', {
            gemini: providerStats.gemini ?? '-',
            codex: providerStats.codex ?? '-',
            claude: providerStats.claude ?? '-',
            openai: providerStats.openai ?? '-'
          })
        : undefined
    },
    {
      label: t('nav.auth_files'),
      value: stats.authFiles ?? '-',
      icon: <IconFileText size={24} />,
      path: '/auth-files',
      loading: loading && stats.authFiles === null,
      sublabel: t('dashboard.oauth_credentials')
    },
    {
      label: t('dashboard.available_models'),
      value: modelsLoading ? '-' : models.length,
      icon: <IconSatellite size={24} />,
      path: '/system',
      loading: modelsLoading,
      sublabel: t('dashboard.available_models_desc')
    }
  ];

  const routingStrategyRaw = config?.routingStrategy?.trim() || '';
  const routingStrategyDisplay = !routingStrategyRaw
    ? '-'
    : routingStrategyRaw === 'round-robin'
      ? t('basic_settings.routing_strategy_round_robin')
      : routingStrategyRaw === 'fill-first'
        ? t('basic_settings.routing_strategy_fill_first')
        : routingStrategyRaw;
  const routingStrategyBadgeClass = !routingStrategyRaw
    ? 'text-muted-foreground bg-background'
    : routingStrategyRaw === 'round-robin'
      ? 'text-primary border-primary/25 bg-primary/12'
      : routingStrategyRaw === 'fill-first'
        ? 'text-success border-success/30 bg-success/12'
        : 'text-muted-foreground bg-background';

  // Derived time-based values
  const greetingKey = `dashboard.greeting_${timeOfDay}`;
  const caringKey = `dashboard.caring_${timeOfDay}`;

  const formattedDate = currentTime.toLocaleDateString(i18n.language, {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric'
  });

  const formattedTime = currentTime.toLocaleTimeString(i18n.language, {
    hour: '2-digit',
    minute: '2-digit'
  });

  return (
    <div className="flex flex-col gap-8 max-w-[1000px] mx-auto relative">
      {/* Decorative background orbs */}
      <div
        className="absolute inset-0 pointer-events-none z-0 overflow-hidden opacity-[0.42] blur-[16px]"
        aria-hidden="true"
      >
        <div
          className="absolute w-[420px] h-[420px] rounded-full top-[-140px] right-[-80px]"
          style={{
            background: 'radial-gradient(circle, color-mix(in srgb, var(--primary) 6%, transparent), transparent 70%)',
            animation: 'orbFloat 22s ease-in-out infinite alternate'
          }}
        />
        <div
          className="absolute w-[320px] h-[320px] rounded-full bottom-[18%] left-[-100px]"
          style={{
            background: 'radial-gradient(circle, color-mix(in srgb, var(--success) 4%, transparent), transparent 70%)',
            animation: 'orbFloat 28s ease-in-out infinite alternate-reverse'
          }}
        />
      </div>

      {/* Hero welcome section */}
      <section
        className="relative z-[1] flex items-end justify-between gap-6 px-8 py-12 overflow-hidden max-md:flex-col max-md:items-start max-md:px-6 max-md:py-8"
        style={{
          background: 'linear-gradient(135deg, color-mix(in srgb, var(--background) 92%, transparent), color-mix(in srgb, var(--muted) 80%, transparent))',
          border: '1px solid color-mix(in srgb, var(--border) 60%, transparent)',
          backdropFilter: 'blur(12px)',
          WebkitBackdropFilter: 'blur(12px)',
          animation: 'heroEnter 0.6s ease-out both'
        }}
      >
        <span
          className="absolute top-1/2 left-8 -translate-y-1/2 text-[104px] font-black leading-none uppercase text-foreground opacity-[0.04] whitespace-nowrap pointer-events-none select-none max-md:text-[58px] max-md:left-6"
          style={{ animation: 'watermarkEnter 0.8s ease-out 0.1s both' }}
          aria-hidden="true"
        >
          OVERVIEW
        </span>
        <div
          className="relative z-[1] flex flex-col gap-1"
          style={{ animation: 'fadeSlideUp 0.5s ease-out 0.1s both' }}
        >
          <span className="text-[13px] font-semibold uppercase text-primary">
            {t(greetingKey)}
          </span>
          <h1
            className="m-0 text-[44px] font-extrabold leading-[1.1] text-foreground max-md:text-[34px]"
            style={{ animation: 'fadeSlideUp 0.5s ease-out 0.2s both' }}
          >
            {t('dashboard.welcome_back')}
          </h1>
          <p
            className="mt-1 text-[15px] text-muted-foreground leading-[1.5]"
            style={{ animation: 'fadeSlideUp 0.5s ease-out 0.3s both' }}
          >
            {t(caringKey)}
          </p>
        </div>
        <div
          className="relative z-[1] flex flex-col items-end gap-2 shrink-0 max-md:items-start max-md:flex-row max-md:flex-wrap max-md:gap-2"
          style={{ animation: 'fadeSlideUp 0.5s ease-out 0.35s both' }}
        >
          <div className="flex flex-col items-end gap-[2px] max-md:items-start">
            <span className="text-[22px] font-bold text-foreground tabular-nums leading-[1.2]">
              {formattedTime}
            </span>
            <span className="text-[12px] text-muted-foreground">{formattedDate}</span>
          </div>
          <div
            className="inline-flex items-center gap-[6px] px-3 py-[5px] rounded-full text-[12px]"
            style={{
              background: 'color-mix(in srgb, var(--muted) 80%, transparent)',
              border: '1px solid color-mix(in srgb, var(--border) 60%, transparent)',
              backdropFilter: 'blur(8px)',
              WebkitBackdropFilter: 'blur(8px)'
            }}
          >
            <span
              className={[
                'w-2 h-2 rounded-full shrink-0',
                connectionStatus === 'connected'
                  ? 'bg-success [box-shadow:0_0_6px_color-mix(in_srgb,var(--success)_50%,transparent)]'
                  : connectionStatus === 'connecting'
                    ? 'bg-warning'
                    : 'bg-destructive'
              ].join(' ')}
              style={
                connectionStatus === 'connecting'
                  ? { animation: 'dotPulse 1s ease-in-out infinite' }
                  : undefined
              }
            />
            <span className="font-semibold text-foreground">
              {serverVersion
                ? `v${serverVersion.trim().replace(/^[vV]+/, '')}`
                : t(
                    connectionStatus === 'connected'
                      ? 'common.connected'
                      : connectionStatus === 'connecting'
                        ? 'common.connecting'
                        : 'common.disconnected'
                  )}
            </span>
          </div>
          {serverBuildDate && (
            <span className="text-[11px] text-muted-foreground text-right max-md:text-left">
              {new Date(serverBuildDate).toLocaleDateString(i18n.language)}
            </span>
          )}
        </div>
      </section>

      {/* Bento stats grid */}
      <section className="relative z-[1]">
        <h2 className="text-[12px] font-bold uppercase text-muted-foreground m-0 mb-3">
          {t('dashboard.system_overview')}
        </h2>
        <div className="grid grid-cols-3 gap-4 max-[900px]:grid-cols-2 max-[500px]:grid-cols-1">
          {quickStats.map((stat, index) => (
            <Link
              key={stat.path}
              to={stat.path}
              className={[
                'flex flex-col gap-4 p-6 text-decoration-none transition-all duration-150',
                'hover:-translate-y-[2px]',
                index === 0 ? 'row-span-2 justify-center max-[900px]:row-auto' : ''
              ].join(' ')}
              style={{
                background:
                  index === 0
                    ? 'linear-gradient(160deg, color-mix(in srgb, var(--primary) 8%, var(--background)), color-mix(in srgb, var(--background) 84%, transparent))'
                    : 'linear-gradient(145deg, color-mix(in srgb, var(--background) 86%, transparent), color-mix(in srgb, var(--muted) 72%, transparent))',
                border: '1px solid color-mix(in srgb, var(--border) 66%, transparent)',
                backdropFilter: 'blur(12px)',
                WebkitBackdropFilter: 'blur(12px)',
                boxShadow: '0 18px 42px rgb(0 0 0 / 0.12), inset 0 1px 0 rgb(255 255 255 / 0.03)',
                textDecoration: 'none',
                animationDelay: `${index * 80}ms`,
                animation: 'cardEnter 0.4s ease-out both'
              }}
            >
              <div
                className="flex items-center justify-center w-11 h-11 text-primary transition-colors duration-150"
                style={{
                  background: 'color-mix(in srgb, var(--primary) 10%, var(--muted))'
                }}
              >
                {stat.icon}
              </div>
              <div className="flex flex-col gap-[2px]">
                <span
                  className={[
                    'font-extrabold text-foreground tabular-nums leading-[1.2]',
                    index === 0 ? 'text-[44px] max-[900px]:text-[32px]' : 'text-[28px]'
                  ].join(' ')}
                >
                  {stat.loading ? '...' : stat.value}
                </span>
                <span className="text-[13px] text-muted-foreground">{stat.label}</span>
                {stat.sublabel && !stat.loading && (
                  <span className="text-[11px] text-muted-foreground opacity-70 mt-[2px]">
                    {stat.sublabel}
                  </span>
                )}
              </div>
            </Link>
          ))}
        </div>
      </section>

      {/* Config pills section */}
      {config && (
        <section
          className="relative z-[1] flex flex-col gap-4"
          style={{ animation: 'cardEnter 0.4s ease-out 0.5s both' }}
        >
          <h2 className="text-[12px] font-bold uppercase text-muted-foreground m-0 mb-3">
            {t('dashboard.current_config')}
          </h2>
          <div className="flex flex-wrap gap-2">
            <div
              className="inline-flex items-center gap-2 px-[14px] py-[6px] rounded-full text-[13px] transition-colors duration-150"
              style={{
                background: 'color-mix(in srgb, var(--background) 68%, transparent)',
                border: '1px solid color-mix(in srgb, var(--border) 68%, transparent)',
                backdropFilter: 'blur(10px)',
                WebkitBackdropFilter: 'blur(10px)'
              }}
            >
              <span className="text-muted-foreground whitespace-nowrap">
                {t('basic_settings.debug_enable')}
              </span>
              <span className={`font-semibold ${config.debug ? 'text-success' : 'text-muted-foreground'}`}>
                {config.debug ? t('common.yes') : t('common.no')}
              </span>
            </div>
            <div
              className="inline-flex items-center gap-2 px-[14px] py-[6px] rounded-full text-[13px] transition-colors duration-150"
              style={{
                background: 'color-mix(in srgb, var(--background) 68%, transparent)',
                border: '1px solid color-mix(in srgb, var(--border) 68%, transparent)',
                backdropFilter: 'blur(10px)',
                WebkitBackdropFilter: 'blur(10px)'
              }}
            >
              <span className="text-muted-foreground whitespace-nowrap">
                {t('basic_settings.logging_to_file_enable')}
              </span>
              <span className={`font-semibold ${config.loggingToFile ? 'text-success' : 'text-muted-foreground'}`}>
                {config.loggingToFile ? t('common.yes') : t('common.no')}
              </span>
            </div>
            <div
              className="inline-flex items-center gap-2 px-[14px] py-[6px] rounded-full text-[13px] transition-colors duration-150"
              style={{
                background: 'color-mix(in srgb, var(--background) 68%, transparent)',
                border: '1px solid color-mix(in srgb, var(--border) 68%, transparent)',
                backdropFilter: 'blur(10px)',
                WebkitBackdropFilter: 'blur(10px)'
              }}
            >
              <span className="text-muted-foreground whitespace-nowrap">
                {t('basic_settings.retry_count_label')}
              </span>
              <span className="font-semibold text-foreground">{config.requestRetry ?? 0}</span>
            </div>
            <div
              className="inline-flex items-center gap-2 px-[14px] py-[6px] rounded-full text-[13px] transition-colors duration-150"
              style={{
                background: 'color-mix(in srgb, var(--background) 68%, transparent)',
                border: '1px solid color-mix(in srgb, var(--border) 68%, transparent)',
                backdropFilter: 'blur(10px)',
                WebkitBackdropFilter: 'blur(10px)'
              }}
            >
              <span className="text-muted-foreground whitespace-nowrap">
                {t('basic_settings.ws_auth_enable')}
              </span>
              <span className={`font-semibold ${config.wsAuth ? 'text-success' : 'text-muted-foreground'}`}>
                {config.wsAuth ? t('common.yes') : t('common.no')}
              </span>
            </div>
            <div
              className="inline-flex items-center gap-2 px-[14px] py-[6px] rounded-full text-[13px] transition-colors duration-150"
              style={{
                background: 'color-mix(in srgb, var(--background) 68%, transparent)',
                border: '1px solid color-mix(in srgb, var(--border) 68%, transparent)',
                backdropFilter: 'blur(10px)',
                WebkitBackdropFilter: 'blur(10px)'
              }}
            >
              <span className="text-muted-foreground whitespace-nowrap">
                {t('dashboard.routing_strategy')}
              </span>
              <span
                className={`inline-flex items-center justify-center px-2 py-[2px] rounded-full border text-[11px] font-semibold leading-[1.2] max-w-full overflow-hidden text-ellipsis whitespace-nowrap border-border ${routingStrategyBadgeClass}`}
              >
                {routingStrategyDisplay}
              </span>
            </div>
            {config.proxyUrl && (
              <div
                className="inline-flex items-center gap-2 px-[14px] py-[6px] text-[13px] transition-colors duration-150 basis-full"
                style={{
                  background: 'color-mix(in srgb, var(--background) 68%, transparent)',
                  border: '1px solid color-mix(in srgb, var(--border) 68%, transparent)',
                  backdropFilter: 'blur(10px)',
                  WebkitBackdropFilter: 'blur(10px)'
                }}
              >
                <span className="text-muted-foreground whitespace-nowrap">
                  {t('basic_settings.proxy_url_label')}
                </span>
                <span className="text-[12px] font-mono text-muted-foreground break-all">
                  {config.proxyUrl}
                </span>
              </div>
            )}
          </div>
          <Link to="/config" className="inline-flex items-center text-[13px] text-primary no-underline mt-1 hover:underline transition-colors duration-150">
            {t('dashboard.edit_settings')} →
          </Link>
        </section>
      )}
    </div>
  );
}
