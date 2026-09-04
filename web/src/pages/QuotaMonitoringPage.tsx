import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import {
  Activity,
  Bell,
  History,
  Search,
  X,
  Clock,
  Key,
  Eye,
  EyeOff,
  AlertTriangle,
  Send,
  SlidersHorizontal,
} from 'lucide-react';
import { AppCard as Card } from '@/components/ui/AppCard';
import { Button } from '@/components/ui/Button';
import { FormInput } from '@/components/ui/FormInput';
import { LoadingSpinner } from '@/components/ui/LoadingSpinner';
import { ToggleSwitch } from '@/components/ui/ToggleSwitch';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { Reveal, useAnimatedNumber } from '@/components/motion';
import { useHeaderRefresh } from '@/hooks/useHeaderRefresh';
import { useUnsavedChangesGuard } from '@/hooks/useUnsavedChangesGuard';
import { quotaAlertsApi } from '@/services/api';
import { cn } from '@/lib/utils';
import { quotaStyles as styles } from '@/components/quota/quotaStyles';
import type {
  ApiError,
  QuotaAlertEvent,
  QuotaAlertPageQuery,
  QuotaAlertProvider,
  QuotaAlertProviderOverride,
  QuotaAlertSettings,
  QuotaAlertState,
} from '@/types';

const PROVIDERS: Array<{ provider: QuotaAlertProvider; label: string }> = [
  { provider: 'claude', label: 'Claude' },
  { provider: 'codex', label: 'Codex' },
  { provider: 'gemini-cli', label: 'Gemini CLI' },
  { provider: 'antigravity', label: 'Antigravity' },
  { provider: 'kimi', label: 'Kimi' },
  { provider: 'xai', label: 'xAI' },
  { provider: 'kiro', label: 'Kiro' },
];

const pageQuery: QuotaAlertPageQuery = { limit: 100 };

function StatNumber({ value, className }: { value: number; className?: string }) {
  const animated = useAnimatedNumber(value);
  return (
    <div className={cn('text-2xl font-bold tabular-nums', className)}>
      {Math.round(animated)}
    </div>
  );
}

const defaultSettings = (): QuotaAlertSettings => ({
  revision: 0,
  enabled: false,
  pollIntervalSeconds: 300,
  warningThreshold: 10,
  notifyRecovery: true,
  reminderIntervalSeconds: 0,
  providers: PROVIDERS.map(({ provider }) => ({ provider, enabled: true, warningThreshold: null })),
  telegram: { enabled: false, chatId: '', tokenConfigured: false },
});

const settingsEqual = (left: QuotaAlertSettings | null, right: QuotaAlertSettings | null) =>
  JSON.stringify(left) === JSON.stringify(right);

const formatPercent = (value?: number) => (typeof value === 'number' ? `${value}%` : '—');
const formatTime = (value?: string) => {
  if (!value) return '—';
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
};

const tableClass = 'w-full text-left text-sm';
const tableHeadClass = 'border-b border-border bg-muted/40 text-xs text-muted-foreground';
const tableCellClass = 'px-3.5 py-2.5 align-middle';
const tableRowClass = 'border-b border-border/60 last:border-0 hover:bg-muted/15 transition-colors';
const badgeClass = 'inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium capitalize';

const eventBadgeClass = (value: string) => {
  switch (value) {
    case 'healthy':
    case 'recovery':
    case 'sent':
      return 'border-success/30 bg-success/12 text-success';
    case 'warning':
    case 'pending':
      return 'border-warning/30 bg-warning/12 text-warning';
    case 'exhausted':
    case 'failed':
      return 'border-destructive/30 bg-destructive/10 text-destructive';
    default:
      return 'border-border bg-muted text-muted-foreground';
  }
};

const alertColorClass = (alert: string) => {
  switch (alert) {
    case 'healthy':
      return 'bg-success';
    case 'warning':
      return 'bg-warning';
    case 'exhausted':
      return 'bg-destructive';
    default:
      return 'bg-muted-foreground';
  }
};

const cloneSettings = (settings: QuotaAlertSettings): QuotaAlertSettings => ({
  ...settings,
  providers: settings.providers.map((provider) => ({ ...provider })),
  telegram: { ...settings.telegram },
});

function QuotaMonitoringShell({
  title,
  description,
  actions,
  children,
}: {
  title: ReactNode;
  description: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className={cn(styles.container, 'flex-1 flex flex-col')}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className={styles.pageHeader}>
          <h1 className={styles.pageTitle}>{title}</h1>
          <p className={styles.description}>{description}</p>
        </div>
        {actions ? <div className={styles.headerActions}>{actions}</div> : null}
      </div>
      {children}
    </div>
  );
}

export function QuotaMonitoringPage() {
  const { t } = useTranslation();
  const [settings, setSettings] = useState<QuotaAlertSettings | null>(null);
  const [savedSettings, setSavedSettings] = useState<QuotaAlertSettings | null>(null);
  const [states, setStates] = useState<QuotaAlertState[]>([]);
  const [events, setEvents] = useState<QuotaAlertEvent[]>([]);
  const [stateCursor, setStateCursor] = useState<string | undefined>();
  const [eventCursor, setEventCursor] = useState<string | undefined>();
  const [loading, setLoading] = useState(true);
  const [loadingMoreStates, setLoadingMoreStates] = useState(false);
  const [loadingMoreEvents, setLoadingMoreEvents] = useState(false);
  const [saving, setSaving] = useState(false);
  const [testingTelegram, setTestingTelegram] = useState(false);
  const [acknowledgingID, setAcknowledgingID] = useState('');
  const [error, setError] = useState('');
  const [upgradeRequired, setUpgradeRequired] = useState(false);
  const [tokenDraft, setTokenDraft] = useState('');
  const [clearToken, setClearToken] = useState(false);
  const [showToken, setShowToken] = useState(false);
  const [providerSearch, setProviderSearch] = useState('');
  const [activeTab, setActiveTab] = useState<'states' | 'settings' | 'events'>('states');
  const [filterStatus, setFilterStatus] = useState<'all' | 'exhausted' | 'warning' | 'healthy'>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const requestSeq = useRef(0);

  const dirty = !settingsEqual(settings, savedSettings) || tokenDraft.trim() !== '' || clearToken;
  const dirtyRef = useRef(false);
  useEffect(() => {
    dirtyRef.current = dirty;
  }, [dirty]);

  useUnsavedChangesGuard({
    shouldBlock: dirty,
    dialog: {
      title: t('quota_monitoring.unsaved_title', { defaultValue: 'Discard quota monitoring changes?' }),
      message: t('quota_monitoring.unsaved_message', {
        defaultValue: 'You have unsaved quota alert settings. Leave without saving?',
      }),
      confirmText: t('common.discard', { defaultValue: 'Discard' }),
      cancelText: t('common.cancel', { defaultValue: 'Cancel' }),
    },
  });

  const loadAll = useCallback(async (options: { force?: boolean } = {}) => {
    if (dirtyRef.current && !options.force) {
      toast.info(t('quota_monitoring.refresh_blocked_dirty', { defaultValue: 'Save or reset changes before refreshing.' }));
      return;
    }

    const seq = requestSeq.current + 1;
    requestSeq.current = seq;
    setLoading(true);
    setError('');
    setUpgradeRequired(false);

    const [settingsResult, statesResult, eventsResult] = await Promise.allSettled([
      quotaAlertsApi.getSettings(),
      quotaAlertsApi.listStates(pageQuery),
      quotaAlertsApi.listEvents(pageQuery),
    ]);
    if (requestSeq.current !== seq) return;

    if (settingsResult.status === 'fulfilled') {
      const nextSettings = cloneSettings(settingsResult.value);
      setSettings(nextSettings);
      setSavedSettings(cloneSettings(nextSettings));
      setTokenDraft('');
      setClearToken(false);
    } else {
      const apiError = settingsResult.reason as ApiError;
      if (apiError.status === 404 || apiError.status === 503) {
        setUpgradeRequired(true);
      }
      setError(apiError.message || t('notification.refresh_failed'));
    }

    if (statesResult.status === 'fulfilled') {
      setStates(statesResult.value.items);
      setStateCursor(statesResult.value.nextCursor);
    }
    if (eventsResult.status === 'fulfilled') {
      setEvents(eventsResult.value.items);
      setEventCursor(eventsResult.value.nextCursor);
    }
    if (statesResult.status === 'rejected' || eventsResult.status === 'rejected') {
      setError((prev) => prev || t('notification.refresh_failed'));
    }
    setLoading(false);
  }, [t]);

  useHeaderRefresh(loadAll);

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  const providerOverrides = useMemo(() => {
    const map = new Map<QuotaAlertProvider, QuotaAlertProviderOverride>();
    settings?.providers.forEach((provider) => map.set(provider.provider, provider));
    return map;
  }, [settings]);
  const filteredProviders = useMemo(() => {
    if (!providerSearch.trim()) return PROVIDERS;
    const q = providerSearch.toLowerCase().trim();
    return PROVIDERS.filter(
      (p) => p.label.toLowerCase().includes(q) || p.provider.toLowerCase().includes(q)
    );
  }, [providerSearch]);


  const healthSummary = useMemo(() => {
    const counts = { exhausted: 0, warning: 0, healthy: 0, unknown: 0 };
    states.forEach((state) => {
      counts[state.alert] += 1;
    });
    return counts;
  }, [states]);

  const filteredStates = useMemo(() => {
    return states.filter((state) => {
      if (filterStatus !== 'all' && state.alert !== filterStatus) {
        return false;
      }
      if (searchQuery.trim()) {
        const query = searchQuery.toLowerCase().trim();
        const matchAuth = state.authLabel.toLowerCase().includes(query);
        const matchResource = state.resource.toLowerCase().includes(query);
        const matchProvider = state.provider.toLowerCase().includes(query);
        if (!matchAuth && !matchResource && !matchProvider) {
          return false;
        }
      }
      return true;
    });
  }, [states, filterStatus, searchQuery]);

  const groupedFilteredStates = useMemo(() => {
    const groups = new Map<QuotaAlertProvider, QuotaAlertState[]>();
    filteredStates.forEach((state) => {
      const list = groups.get(state.provider) ?? [];
      list.push(state);
      groups.set(state.provider, list);
    });
    return PROVIDERS.map(({ provider, label }) => ({
      provider,
      label,
      states: groups.get(provider) ?? [],
    })).filter((group) => group.states.length > 0);
  }, [filteredStates]);

  const updateSettings = (patch: Partial<QuotaAlertSettings>) => {
    setSettings((current) => (current ? { ...current, ...patch } : current));
  };

  const updateTelegram = (patch: Partial<QuotaAlertSettings['telegram']>) => {
    setSettings((current) =>
      current ? { ...current, telegram: { ...current.telegram, ...patch } } : current
    );
  };

  const updateProvider = (
    provider: QuotaAlertProvider,
    patch: Partial<QuotaAlertProviderOverride>
  ) => {
    setSettings((current) => {
      if (!current) return current;
      const seen = new Set<QuotaAlertProvider>();
      const providers = current.providers.map((entry) => {
        if (entry.provider !== provider) return entry;
        seen.add(provider);
        return { ...entry, ...patch };
      });
      if (!seen.has(provider)) {
        providers.push({ provider, enabled: true, warningThreshold: null, ...patch });
      }
      return { ...current, providers };
    });
  };

  const handleSave = async () => {
    if (!settings) return;
    setSaving(true);
    try {
      let nextSettings = await quotaAlertsApi.updateSettings({
        revision: settings.revision,
        enabled: settings.enabled,
        pollIntervalSeconds: settings.pollIntervalSeconds,
        warningThreshold: settings.warningThreshold,
        notifyRecovery: settings.notifyRecovery,
        reminderIntervalSeconds: settings.reminderIntervalSeconds,
        providers: PROVIDERS.map(({ provider }) => {
          const override = providerOverrides.get(provider);
          return {
            provider,
            enabled: override?.enabled ?? true,
            warningThreshold: override?.warningThreshold ?? null,
          };
        }),
      });

      const trimmedToken = tokenDraft.trim();
      const needsTelegramUpdate =
        nextSettings.telegram.enabled !== settings.telegram.enabled ||
        nextSettings.telegram.chatId !== settings.telegram.chatId ||
        trimmedToken !== '' ||
        clearToken;

      if (needsTelegramUpdate) {
        try {
          nextSettings = await quotaAlertsApi.updateTelegram({
            revision: nextSettings.revision,
            enabled: settings.telegram.enabled,
            chatId: settings.telegram.chatId,
            token: trimmedToken || undefined,
            clearToken,
          });
        } catch (err: unknown) {
          const saved = cloneSettings(nextSettings);
          setSavedSettings(saved);
          setSettings({ ...saved, telegram: { ...settings.telegram } });
          const message = err instanceof Error ? err.message : t('notification.update_failed');
          toast.error(message);
          return;
        }
      }

      const cloned = cloneSettings(nextSettings);
      setSettings(cloned);
      setSavedSettings(cloneSettings(cloned));
      setTokenDraft('');
      setClearToken(false);
      toast.success(t('quota_monitoring.save_success', { defaultValue: 'Quota alert settings saved' }));
      void loadAll({ force: true });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('notification.update_failed');
      toast.error(message);
    } finally {
      setSaving(false);
    }
  };

  const handleReset = () => {
    if (!savedSettings) return;
    setSettings(cloneSettings(savedSettings));
    setTokenDraft('');
    setClearToken(false);
  };

  const handleLoadMoreStates = async () => {
    if (!stateCursor) return;
    setLoadingMoreStates(true);
    try {
      const page = await quotaAlertsApi.listStates({ ...pageQuery, cursor: stateCursor });
      setStates((current) => [...current, ...page.items]);
      setStateCursor(page.nextCursor);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('notification.refresh_failed');
      toast.error(message);
    } finally {
      setLoadingMoreStates(false);
    }
  };

  const handleLoadMoreEvents = async () => {
    if (!eventCursor) return;
    setLoadingMoreEvents(true);
    try {
      const page = await quotaAlertsApi.listEvents({ ...pageQuery, cursor: eventCursor });
      setEvents((current) => [...current, ...page.items]);
      setEventCursor(page.nextCursor);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('notification.refresh_failed');
      toast.error(message);
    } finally {
      setLoadingMoreEvents(false);
    }
  };

  const handleTelegramTest = async () => {
    if (dirty) {
      toast.info(t('quota_monitoring.telegram_test_blocked_dirty', { defaultValue: 'Save or reset changes before sending a test.' }));
      return;
    }
    setTestingTelegram(true);
    try {
      await quotaAlertsApi.testTelegram();
      toast.success(t('quota_monitoring.telegram_test_success', { defaultValue: 'Telegram test sent' }));
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('notification.update_failed');
      toast.error(message);
    } finally {
      setTestingTelegram(false);
    }
  };

  const handleAcknowledge = async (event: QuotaAlertEvent) => {
    setAcknowledgingID(event.id);
    try {
      await quotaAlertsApi.acknowledgeEvent(event.id);
      setEvents((current) =>
        current.map((item) =>
          item.id === event.id ? { ...item, acknowledgedAt: new Date().toISOString() } : item
        )
      );
      toast.success(t('quota_monitoring.ack_success', { defaultValue: 'Event acknowledged' }));
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('notification.update_failed');
      toast.error(message);
    } finally {
      setAcknowledgingID('');
    }
  };

  const pageTitle = t('quota_monitoring.title', { defaultValue: 'Quota Monitoring' });
  const pageDescription = t('quota_monitoring.description', {
    defaultValue: 'Configure database-backed quota alerts, provider thresholds, Telegram delivery, and recent event acknowledgement.',
  });

  if (loading && !settings) {
    return (
      <QuotaMonitoringShell title={pageTitle} description={pageDescription}>
        <div className="flex items-center justify-center gap-2 py-12 text-muted-foreground">
          <LoadingSpinner size={16} />
          <span>{t('common.loading', { defaultValue: 'Loading...' })}</span>
        </div>
      </QuotaMonitoringShell>
    );
  }

  if (upgradeRequired) {
    return (
      <QuotaMonitoringShell title={pageTitle} description={pageDescription}>
        <Card>
          <div className="flex flex-col gap-3">
            <h3 className="text-base font-semibold text-foreground">
              {t('quota_monitoring.unavailable_title', { defaultValue: 'Quota monitoring unavailable' })}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t('quota_monitoring.unavailable_desc', {
                defaultValue: 'This server does not expose the database-backed quota alert API yet.',
              })}
            </p>
          </div>
        </Card>
      </QuotaMonitoringShell>
    );
  }

  const activeSettings = settings ?? defaultSettings();

  return (
    <QuotaMonitoringShell
      title={pageTitle}
      description={pageDescription}
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <div className="flex items-center gap-1.5 mr-2">
            <span
              className={cn(
                'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium border',
                activeSettings.enabled
                  ? 'border-success/30 bg-success/12 text-success'
                  : 'border-border bg-muted text-muted-foreground'
              )}
            >
              <Activity className="h-3 w-3" />
              <span>
                {activeSettings.enabled
                  ? t('quota_monitoring.monitoring_active', { defaultValue: 'Monitoring Active' })
                  : t('quota_monitoring.monitoring_paused', { defaultValue: 'Monitoring Paused' })}
              </span>
            </span>
            <span
              className={cn(
                'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium border',
                activeSettings.telegram.secretKeyConfigured !== false
                  ? 'border-success/30 bg-success/12 text-success'
                  : 'border-amber-500/30 bg-amber-500/10 text-amber-600 dark:text-amber-400'
              )}
            >
              <Key className="h-3 w-3" />
              <span>
                {activeSettings.telegram.secretKeyConfigured !== false
                  ? t('quota_monitoring.key_configured', { defaultValue: 'Key Active' })
                  : t('quota_monitoring.key_missing', { defaultValue: 'Key Missing' })}
              </span>
            </span>
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium border border-border bg-muted/40 text-muted-foreground">
              <Clock className="h-3 w-3" />
              <span>{activeSettings.pollIntervalSeconds}s</span>
            </span>
          </div>
          <Button variant="secondary" size="sm" onClick={() => void loadAll()} disabled={loading || saving}>
            {loading ? <LoadingSpinner size={14} /> : null}
            {t('common.refresh', { defaultValue: 'Refresh' })}
          </Button>
        </div>
      }
    >
      {error ? (
        <div className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {error}
        </div>
      ) : null}

      {/* Top KPI Stat Cards */}
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <Reveal delay={0}>
          <Card className="flex flex-col justify-between p-4 border border-border bg-card">
            <div className="text-xs font-medium text-muted-foreground">
              {t('quota_monitoring.summary_total', { defaultValue: 'Tracked states' })}
            </div>
            <div className="mt-2 text-2xl font-bold tabular-nums text-foreground">
              <StatNumber value={states.length} />
            </div>
          </Card>
        </Reveal>
        <Reveal delay={40}>
          <Card className="flex flex-col justify-between p-4 border border-border bg-card border-t-2 border-t-destructive">
            <div className="flex items-center justify-between text-xs font-medium text-muted-foreground">
              <span>{t('quota_monitoring.summary_exhausted', { defaultValue: 'Exhausted' })}</span>
              <span className="h-2 w-2 rounded-full bg-destructive" />
            </div>
            <div className="mt-2 text-2xl font-bold tabular-nums text-destructive">
              <StatNumber value={healthSummary.exhausted} />
            </div>
          </Card>
        </Reveal>
        <Reveal delay={80}>
          <Card className="flex flex-col justify-between p-4 border border-border bg-card border-t-2 border-t-warning">
            <div className="flex items-center justify-between text-xs font-medium text-muted-foreground">
              <span>{t('quota_monitoring.summary_warning', { defaultValue: 'Warning' })}</span>
              <span className="h-2 w-2 rounded-full bg-warning" />
            </div>
            <div className="mt-2 text-2xl font-bold tabular-nums text-warning">
              <StatNumber value={healthSummary.warning} />
            </div>
          </Card>
        </Reveal>
        <Reveal delay={120}>
          <Card className="flex flex-col justify-between p-4 border border-border bg-card border-t-2 border-t-success">
            <div className="flex items-center justify-between text-xs font-medium text-muted-foreground">
              <span>{t('quota_monitoring.summary_healthy', { defaultValue: 'Healthy' })}</span>
              <span className="h-2 w-2 rounded-full bg-success" />
            </div>
            <div className="mt-2 text-2xl font-bold tabular-nums text-success">
              <StatNumber value={healthSummary.healthy} />
            </div>
          </Card>
        </Reveal>
      </div>

      {/* Tabs Layout */}
      <Tabs
        value={activeTab}
        onValueChange={(val) => setActiveTab(val as 'states' | 'settings' | 'events')}
        className="w-full flex-1 flex flex-col space-y-4"
      >
        <div className="flex items-end justify-between border-b border-border">
          <TabsList className="flex justify-start items-start gap-0 p-0 bg-transparent overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden flex-1 min-w-0">
            <TabsTrigger
              value="states"
              className="px-3 gap-1.5 text-muted-foreground border-b-4 border-transparent transition-colors duration-150 data-[state=active]:border-primary data-[state=active]:text-primary hover:text-foreground"
            >
              <span className="inline-flex items-center gap-1.5">
                <Activity className="h-4 w-4 shrink-0" />
                <span>{t('quota_monitoring.tab_states', { defaultValue: 'Live States' })}</span>
              </span>
              <span className={styles.tabCountBadge}>{states.length}</span>
            </TabsTrigger>
            <TabsTrigger
              value="settings"
              className="px-3 gap-1.5 text-muted-foreground border-b-4 border-transparent transition-colors duration-150 data-[state=active]:border-primary data-[state=active]:text-primary hover:text-foreground"
            >
              <span className="inline-flex items-center gap-1.5">
                <SlidersHorizontal className="h-4 w-4 shrink-0" />
                <span>{t('quota_monitoring.tab_settings', { defaultValue: 'Alerts & Channels' })}</span>
              </span>
              {dirty && <span className="h-1.5 w-1.5 rounded-full bg-primary shrink-0" />}
            </TabsTrigger>
            <TabsTrigger
              value="events"
              className="px-3 gap-1.5 text-muted-foreground border-b-4 border-transparent transition-colors duration-150 data-[state=active]:border-primary data-[state=active]:text-primary hover:text-foreground"
            >
              <span className="inline-flex items-center gap-1.5">
                <History className="h-4 w-4 shrink-0" />
                <span>{t('quota_monitoring.tab_events', { defaultValue: 'Event History' })}</span>
              </span>
              {events.length > 0 && <span className={styles.tabCountBadge}>{events.length}</span>}
            </TabsTrigger>
          </TabsList>
        </div>

        {/* Tab 1: Live States */}
        <TabsContent value="states" className="space-y-4 outline-none">
          {/* Filter and Search Bar */}
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
              <input
                type="text"
                placeholder={t('quota_monitoring.search_placeholder', { defaultValue: 'Search auth or resource...' })}
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-8 pr-8 py-1.5 text-xs bg-background border border-input rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              />
              {searchQuery && (
                <button
                  type="button"
                  onClick={() => setSearchQuery('')}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                >
                  <X className="h-3 w-3" />
                </button>
              )}
            </div>
            <div className="flex flex-wrap items-center gap-1.5">
              {[
                { id: 'all', label: t('quota_monitoring.filter_all', { defaultValue: 'All' }), count: states.length },
                { id: 'exhausted', label: t('quota_monitoring.summary_exhausted', { defaultValue: 'Exhausted' }), count: healthSummary.exhausted },
                { id: 'warning', label: t('quota_monitoring.summary_warning', { defaultValue: 'Warning' }), count: healthSummary.warning },
                { id: 'healthy', label: t('quota_monitoring.summary_healthy', { defaultValue: 'Healthy' }), count: healthSummary.healthy },
              ].map((f) => (
                <button
                  key={f.id}
                  type="button"
                  onClick={() => setFilterStatus(f.id as typeof filterStatus)}
                  className={cn(
                    'inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium transition-colors border',
                    filterStatus === f.id
                      ? 'bg-primary text-primary-foreground border-primary'
                      : 'bg-muted/40 text-muted-foreground border-border hover:text-foreground hover:bg-muted'
                  )}
                >
                  <span>{f.label}</span>
                  <span
                    className={cn(
                      'text-[10px] px-1 py-0.2 rounded',
                      filterStatus === f.id ? 'bg-primary-foreground/20 text-primary-foreground' : 'bg-background text-muted-foreground'
                    )}
                  >
                    {f.count}
                  </span>
                </button>
              ))}
            </div>
          </div>

          {/* States Content */}
          {groupedFilteredStates.length === 0 ? (
            <Card>
              <div className="flex flex-col items-center justify-center py-10 text-center">
                <AlertTriangle className="h-8 w-8 text-muted-foreground/60 mb-2" />
                <p className="text-sm font-medium text-foreground">
                  {t('quota_monitoring.no_matching_states', { defaultValue: 'No matching quota states found.' })}
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  {states.length === 0
                    ? t('quota_monitoring.empty_states', { defaultValue: 'No quota states have been collected yet.' })
                    : t('common.try_adjusting_filters', { defaultValue: 'Try adjusting your search or filters.' })}
                </p>
              </div>
            </Card>
          ) : (
            <div className="space-y-4">
              {groupedFilteredStates.map((group) => (
                <Card key={group.provider} className="overflow-hidden p-0 border border-border">
                  <div className="flex items-center justify-between border-b border-border bg-muted/20 px-4 py-2.5">
                    <div className="flex items-center gap-2 font-semibold text-foreground text-sm">
                      <span className="h-2 w-2 rounded-full bg-primary" />
                      <span>{group.label}</span>
                    </div>
                    <span className="text-xs text-muted-foreground font-mono">
                      {group.states.length} {group.states.length === 1 ? 'state' : 'states'}
                    </span>
                  </div>
                  <div className="overflow-x-auto">
                    <table className={cn(tableClass, 'min-w-[760px]')}>
                      <thead className={tableHeadClass}>
                        <tr>
                          <th className={tableCellClass}>{t('quota_monitoring.auth_label', { defaultValue: 'Auth' })}</th>
                          <th className={tableCellClass}>{t('quota_monitoring.resource', { defaultValue: 'Resource' })}</th>
                          <th className={tableCellClass}>{t('quota_monitoring.window', { defaultValue: 'Window' })}</th>
                          <th className={tableCellClass}>{t('quota_monitoring.alert', { defaultValue: 'Alert' })}</th>
                          <th className={cn(tableCellClass, 'w-44')}>{t('quota_monitoring.remaining', { defaultValue: 'Remaining' })}</th>
                          <th className={tableCellClass}>{t('quota_monitoring.reset_at', { defaultValue: 'Reset at' })}</th>
                          <th className={tableCellClass}>{t('quota_monitoring.updated_at', { defaultValue: 'Updated' })}</th>
                        </tr>
                      </thead>
                      <tbody>
                        {group.states.map((state) => (
                          <tr key={`${state.authId}:${state.resource}:${state.window}`} className={tableRowClass}>
                            <td className={cn(tableCellClass, 'font-medium text-foreground')}>
                              <div className="flex flex-col">
                                <span className="font-semibold text-foreground">{state.authLabel}</span>
                                <span className="text-[11px] text-muted-foreground font-mono">{state.resource}</span>
                              </div>
                            </td>
                            <td className={cn(tableCellClass, 'text-xs text-muted-foreground font-mono')}>
                              {state.resource}
                            </td>
                            <td className={cn(tableCellClass, 'text-xs text-muted-foreground font-mono')}>
                              {state.window}
                            </td>
                            <td className={tableCellClass}>
                              <span className={cn(badgeClass, eventBadgeClass(state.alert))}>{state.alert}</span>
                            </td>
                            <td className={cn(tableCellClass, 'w-44')}>
                              <div className="flex flex-col gap-1">
                                <div className="flex justify-between items-center text-xs">
                                  <span className="font-mono font-medium tabular-nums text-foreground">
                                    {formatPercent(state.remaining)}
                                  </span>
                                </div>
                                <div className="h-1.5 w-full overflow-hidden rounded-full bg-secondary">
                                  <div
                                    className={cn('h-full transition-all duration-300', alertColorClass(state.alert))}
                                    style={{ width: `${Math.min(100, Math.max(0, state.remaining ?? 0))}%` }}
                                  />
                                </div>
                              </div>
                            </td>
                            <td className={cn(tableCellClass, 'text-xs text-muted-foreground whitespace-nowrap')}>
                              {formatTime(state.resetAt)}
                            </td>
                            <td className={cn(tableCellClass, 'text-xs text-muted-foreground whitespace-nowrap')}>
                              {formatTime(state.updatedAt)}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </Card>
              ))}
              {stateCursor ? (
                <Button
                  type="button"
                  variant="secondary"
                  className="w-fit"
                  loading={loadingMoreStates}
                  onClick={() => void handleLoadMoreStates()}
                >
                  {t('quota_monitoring.load_more_states', { defaultValue: 'Load more states' })}
                </Button>
              ) : null}
            </div>
          )}
        </TabsContent>

        {/* Tab 2: Alerts & Channels */}
        <TabsContent value="settings" className="space-y-6 outline-none">
          <div className="grid gap-6 lg:grid-cols-2">
            {/* Telegram Card */}
            <Card
              title={
                <div className="flex items-center gap-2">
                  <Send className="h-4 w-4 text-primary" />
                  <span>{t('quota_monitoring.telegram_title', { defaultValue: 'Telegram destination' })}</span>
                </div>
              }
              className="flex flex-col"
            >
              <div className="space-y-4">
                <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border pb-3">
                  <ToggleSwitch
                    checked={activeSettings.telegram.enabled}
                    onChange={(enabled) => updateTelegram({ enabled })}
                    label={t('quota_monitoring.telegram_enabled', { defaultValue: 'Enable Telegram notifications' })}
                  />
                  <div className="flex items-center gap-2">
                    <span
                      className={cn(
                        badgeClass,
                        activeSettings.telegram.tokenConfigured
                          ? 'border-success/30 bg-success/12 text-success'
                          : 'border-border bg-muted text-muted-foreground'
                      )}
                    >
                      {activeSettings.telegram.tokenConfigured
                        ? t('quota_monitoring.telegram_token_configured', { defaultValue: 'Bot token configured' })
                        : t('quota_monitoring.telegram_token_missing', { defaultValue: 'Bot token not configured' })}
                    </span>
                    <Button
                      type="button"
                      variant="secondary"
                      size="sm"
                      onClick={() => void handleTelegramTest()}
                      loading={testingTelegram}
                      disabled={
                        dirty ||
                        saving ||
                        !activeSettings.telegram.enabled ||
                        !activeSettings.telegram.tokenConfigured ||
                        activeSettings.telegram.secretKeyConfigured === false
                      }
                    >
                      <Send className="mr-1.5 h-3.5 w-3.5" />
                      {t('quota_monitoring.telegram_test', { defaultValue: 'Send test message' })}
                    </Button>
                  </div>
                </div>

                {activeSettings.telegram.secretKeyConfigured === false && (
                  <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 p-3 text-xs text-amber-700 dark:text-amber-300">
                    <p className="font-semibold flex items-center gap-1.5">
                      <AlertTriangle className="h-3.5 w-3.5 text-amber-600 dark:text-amber-400" />
                      {t('quota_monitoring.telegram_secret_key_missing_title', {
                        defaultValue: 'Telegram token encryption key is not configured on the server.',
                      })}
                    </p>
                    <p className="mt-1 text-muted-foreground">
                      {t('quota_monitoring.telegram_secret_key_missing_desc', {
                        defaultValue:
                          'To save or test Telegram bot tokens, set the LLMHUB_QUOTA_SECRET_KEY_B64 environment variable (a 32-byte base64 key) on the server.',
                      })}
                    </p>
                  </div>
                )}

                <FormInput
                  id="quota-telegram-chat"
                  label={t('quota_monitoring.telegram_chat_id', { defaultValue: 'Chat ID' })}
                  value={activeSettings.telegram.chatId}
                  onChange={(event) => updateTelegram({ chatId: event.target.value })}
                />

                <div className="space-y-1.5">
                  <label htmlFor="quota-telegram-token" className="text-xs font-medium text-foreground">
                    {t('quota_monitoring.telegram_token', { defaultValue: 'New bot token' })}
                  </label>
                  <div className="relative">
                    <input
                      id="quota-telegram-token"
                      type={showToken ? 'text' : 'password'}
                      placeholder={
                        activeSettings.telegram.tokenConfigured
                          ? '••••••••••••••••••••••••••••••••'
                          : t('quota_monitoring.telegram_token_placeholder', {
                              defaultValue: 'Enter Telegram bot token (123456:ABC...)',
                            })
                      }
                      value={tokenDraft}
                      onChange={(event) => {
                        setTokenDraft(event.target.value);
                        if (event.target.value.trim()) setClearToken(false);
                      }}
                      className="w-full rounded-md border border-input bg-background px-3 py-2 pr-10 text-xs text-foreground font-mono focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    />
                    <button
                      type="button"
                      onClick={() => setShowToken(!showToken)}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors p-1"
                      title={showToken ? 'Hide token' : 'Show token'}
                    >
                      {showToken ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                    </button>
                  </div>
                  <p className="text-[11px] text-muted-foreground">
                    {activeSettings.telegram.tokenConfigured && !tokenDraft
                      ? t('quota_monitoring.telegram_token_configured_hint', {
                          defaultValue: 'Token is saved securely. Type a new token to update.',
                        })
                      : t('quota_monitoring.telegram_token_hint', {
                          defaultValue: 'Leave blank to preserve the stored write-only token.',
                        })}
                  </p>
                </div>

                <ToggleSwitch
                  checked={clearToken}
                  onChange={(value) => {
                    setClearToken(value);
                    if (value) setTokenDraft('');
                  }}
                  label={t('quota_monitoring.telegram_clear_token', { defaultValue: 'Clear stored token on save' })}
                />
              </div>
            </Card>

            {/* Alert Engine Policy Card */}
            <Card
              title={
                <div className="flex items-center gap-2">
                  <Bell className="h-4 w-4 text-primary" />
                  <span>{t('quota_monitoring.settings_title', { defaultValue: 'Alert settings' })}</span>
                </div>
              }
              className="flex flex-col"
            >
              <div className="flex flex-col gap-4">
                <div className="flex items-center justify-between py-1 border-b border-border/50 pb-3">
                  <div className="flex flex-col pr-4">
                    <span className="text-xs font-medium text-foreground">
                      {t('quota_monitoring.enabled', { defaultValue: 'Enable quota monitoring daemon' })}
                    </span>
                  </div>
                  <ToggleSwitch
                    checked={activeSettings.enabled}
                    onChange={(enabled) => updateSettings({ enabled })}
                    ariaLabel="Enable quota monitoring daemon"
                  />
                </div>

                <div className="flex items-center justify-between py-1 border-b border-border/50 pb-3">
                  <div className="flex flex-col pr-4">
                    <span className="text-xs font-medium text-foreground">
                      {t('quota_monitoring.notify_recovery', { defaultValue: 'Notify on quota recovery' })}
                    </span>
                  </div>
                  <ToggleSwitch
                    checked={activeSettings.notifyRecovery}
                    onChange={(notifyRecovery) => updateSettings({ notifyRecovery })}
                    ariaLabel="Notify on quota recovery"
                  />
                </div>

                <div className="grid gap-3 sm:grid-cols-2 pt-1">
                  <FormInput
                    id="quota-poll-interval"
                    type="number"
                    min={60}
                    label={t('quota_monitoring.poll_interval', { defaultValue: 'Poll interval (seconds)' })}
                    value={activeSettings.pollIntervalSeconds}
                    onChange={(event) => updateSettings({ pollIntervalSeconds: Number(event.target.value) })}
                  />
                  <FormInput
                    id="quota-warning-threshold"
                    type="number"
                    min={0}
                    max={100}
                    label={t('quota_monitoring.warning_threshold', { defaultValue: 'Default warning threshold (%)' })}
                    value={activeSettings.warningThreshold}
                    onChange={(event) => updateSettings({ warningThreshold: Number(event.target.value) })}
                  />
                </div>
                <FormInput
                  id="quota-reminder-interval"
                  type="number"
                  min={0}
                  label={t('quota_monitoring.reminder_interval', { defaultValue: 'Reminder interval (seconds)' })}
                  hint={t('quota_monitoring.reminder_hint', { defaultValue: 'Use 0 to disable reminders.' })}
                  value={activeSettings.reminderIntervalSeconds}
                  onChange={(event) => updateSettings({ reminderIntervalSeconds: Number(event.target.value) })}
                />
              </div>
            </Card>
          </div>

          {/* Provider Overrides Matrix */}
          <Card className="flex flex-col gap-4 p-5 border border-border">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between border-b border-border pb-3">
              <div>
                <h3 className="text-sm font-semibold tracking-tight text-foreground flex items-center gap-2">
                  <SlidersHorizontal className="h-4 w-4 text-primary" />
                  <span>{t('quota_monitoring.providers_title', { defaultValue: 'Provider overrides' })}</span>
                </h3>
                <p className="text-xs text-muted-foreground mt-0.5">
                  {t('quota_monitoring.provider_override_hint', {
                    defaultValue: 'Override threshold or disable monitoring per provider',
                  })}
                </p>
              </div>
              <div className="relative w-full sm:w-60">
                <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
                <input
                  type="text"
                  placeholder={t('quota_monitoring.search_provider_placeholder', {
                    defaultValue: 'Filter providers...',
                  })}
                  value={providerSearch}
                  onChange={(e) => setProviderSearch(e.target.value)}
                  className="w-full pl-8 pr-7 py-1.5 text-xs bg-background border border-input rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                />
                {providerSearch && (
                  <button
                    type="button"
                    onClick={() => setProviderSearch('')}
                    className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                  >
                    <X className="h-3 w-3" />
                  </button>
                )}
              </div>
            </div>

            {filteredProviders.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-6 text-center text-xs text-muted-foreground">
                {t('quota_monitoring.no_matching_states', { defaultValue: 'No matching providers found.' })}
              </div>
            ) : (
              <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
                {filteredProviders.map(({ provider, label }) => {
                  const override = providerOverrides.get(provider) ?? { provider, enabled: true, warningThreshold: null };
                  return (
                    <div
                      key={provider}
                      className={cn(
                        'flex flex-col justify-between rounded-lg border p-3 transition-all',
                        override.enabled
                          ? 'border-border bg-card shadow-sm hover:border-primary/40'
                          : 'border-border/60 bg-muted/25 opacity-70'
                      )}
                    >
                      <div className="flex items-center justify-between gap-2 mb-2.5">
                        <div className="flex items-center gap-1.5 min-w-0">
                          <span
                            className={cn(
                              'h-2 w-2 rounded-full shrink-0',
                              override.enabled ? 'bg-primary' : 'bg-muted-foreground'
                            )}
                          />
                          <span className="text-xs font-semibold text-foreground truncate">{label}</span>
                        </div>
                        <ToggleSwitch
                          checked={override.enabled}
                          onChange={(enabled) => updateProvider(provider, { enabled })}
                          ariaLabel={`${label} enabled`}
                        />
                      </div>
                      <div className="space-y-1">
                        <label
                          htmlFor={`quota-provider-${provider}`}
                          className="text-[11px] font-medium text-muted-foreground flex items-center justify-between"
                        >
                          <span>{t('quota_monitoring.provider_threshold', { defaultValue: 'Threshold (%)' })}</span>
                          <span className="text-[10px] text-muted-foreground/80 font-mono">
                            def: {activeSettings.warningThreshold}%
                          </span>
                        </label>
                        <input
                          id={`quota-provider-${provider}`}
                          type="number"
                          min={0}
                          max={100}
                          disabled={!override.enabled}
                          placeholder={String(activeSettings.warningThreshold)}
                          value={override.warningThreshold ?? ''}
                          onChange={(event) =>
                            updateProvider(provider, {
                              warningThreshold: event.target.value === '' ? null : Number(event.target.value),
                            })
                          }
                          className="w-full rounded-md border border-input bg-background px-2.5 py-1 text-xs text-foreground font-mono focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
                        />
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </Card>
        </TabsContent>

        {/* Tab 3: Event History */}
        <TabsContent value="events" className="outline-none flex-1 flex flex-col">
          <Card className="flex-1 flex flex-col p-5 border border-border">
            <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between border-b border-border pb-3 mb-4">
              <h3 className="text-sm font-semibold tracking-tight text-foreground flex items-center gap-2">
                <History className="h-4 w-4 text-primary" />
                <span>{t('quota_monitoring.events_title', { defaultValue: 'Recent events' })}</span>
              </h3>
              <span className="text-xs text-muted-foreground">
                {t('quota_monitoring.delivery_note', {
                  defaultValue: 'Telegram delivery uses the durable outbox; this API exposes in-app acknowledgement state.',
                })}
              </span>
            </div>
            {events.length === 0 ? (
              <div className="flex-1 flex flex-col items-center justify-center py-16 text-center">
                <History className="h-10 w-10 text-muted-foreground/40 mb-3" />
                <p className="text-sm font-medium text-foreground">
                  {t('quota_monitoring.empty_events', { defaultValue: 'No alert events yet.' })}
                </p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className={cn(tableClass, 'min-w-[840px]')}>
                  <thead className={tableHeadClass}>
                    <tr>
                      <th className={tableCellClass}>{t('quota_monitoring.event', { defaultValue: 'Event' })}</th>
                      <th className={tableCellClass}>{t('quota_monitoring.provider', { defaultValue: 'Provider' })}</th>
                      <th className={tableCellClass}>{t('quota_monitoring.auth_label', { defaultValue: 'Auth' })}</th>
                      <th className={tableCellClass}>{t('quota_monitoring.transition', { defaultValue: 'Transition' })}</th>
                      <th className={tableCellClass}>{t('quota_monitoring.remaining', { defaultValue: 'Remaining' })}</th>
                      <th className={tableCellClass}>{t('quota_monitoring.occurred_at', { defaultValue: 'Occurred' })}</th>
                      <th className={tableCellClass}>{t('quota_monitoring.delivery_status', { defaultValue: 'Delivery' })}</th>
                      <th className={tableCellClass}>{t('quota_monitoring.status', { defaultValue: 'Status' })}</th>
                      <th className={tableCellClass} />
                    </tr>
                  </thead>
                  <tbody>
                    {events.map((event) => (
                      <tr key={event.id} className={tableRowClass}>
                        <td className={tableCellClass}>
                          <span className={cn(badgeClass, eventBadgeClass(event.kind))}>{event.kind}</span>
                        </td>
                        <td className={tableCellClass}>{event.provider}</td>
                        <td className={cn(tableCellClass, 'font-medium text-foreground')}>{event.authLabel}</td>
                        <td className={tableCellClass}>{event.from} → {event.to}</td>
                        <td className={cn(tableCellClass, 'tabular-nums font-mono')}>{formatPercent(event.remaining)}</td>
                        <td className={tableCellClass}>{formatTime(event.occurredAt)}</td>
                        <td className={tableCellClass}>
                          {event.delivery ? (
                            <span className={cn(badgeClass, eventBadgeClass(event.delivery.status))}>
                              {event.delivery.status === 'sent'
                                ? t('quota_monitoring.delivery_sent', { defaultValue: 'Sent' })
                                : event.delivery.status === 'failed'
                                  ? event.delivery.failureCode
                                    ? t('quota_monitoring.delivery_failed_code', {
                                        defaultValue: 'Failed: {{code}}',
                                        code: event.delivery.failureCode,
                                      })
                                    : t('quota_monitoring.delivery_failed', { defaultValue: 'Failed' })
                                  : event.delivery.attemptCount > 0
                                    ? t('quota_monitoring.delivery_pending_count', {
                                        defaultValue: 'Pending ({{count}})',
                                        count: event.delivery.attemptCount,
                                      })
                                    : t('quota_monitoring.delivery_pending', { defaultValue: 'Pending' })}
                            </span>
                          ) : (
                            <span className="text-muted-foreground">—</span>
                          )}
                        </td>
                        <td className={tableCellClass}>
                          <span
                            className={cn(
                              badgeClass,
                              event.acknowledgedAt
                                ? 'border-border bg-muted text-muted-foreground'
                                : 'border-primary/30 bg-primary/10 text-primary'
                            )}
                          >
                            {event.acknowledgedAt
                              ? t('quota_monitoring.acknowledged', { defaultValue: 'Acknowledged' })
                              : t('quota_monitoring.open', { defaultValue: 'Open' })}
                          </span>
                        </td>
                        <td className={cn(tableCellClass, 'text-right')}>
                          <Button
                            type="button"
                            variant="secondary"
                            size="sm"
                            disabled={Boolean(event.acknowledgedAt)}
                            loading={acknowledgingID === event.id}
                            onClick={() => void handleAcknowledge(event)}
                          >
                            {t('quota_monitoring.acknowledge', { defaultValue: 'Acknowledge' })}
                          </Button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {eventCursor ? (
                  <div className="p-4 border-t border-border">
                    <Button
                      type="button"
                      variant="secondary"
                      size="sm"
                      className="w-fit"
                      loading={loadingMoreEvents}
                      onClick={() => void handleLoadMoreEvents()}
                    >
                      {t('quota_monitoring.load_more_events', { defaultValue: 'Load more events' })}
                    </Button>
                  </div>
                ) : null}
              </div>
            )}
          </Card>
        </TabsContent>
      </Tabs>

      {/* Floating Save Action Bar when dirty */}
      {dirty && (
        <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 flex items-center gap-3 rounded-lg border border-border bg-card/95 px-4 py-2.5 shadow-xl backdrop-blur-md transition-all duration-200">
          <div className="flex items-center gap-2 text-xs font-medium text-foreground">
            <span className="h-2 w-2 rounded-full bg-primary" />
            <span>{t('quota_monitoring.unsaved_changes', { defaultValue: 'You have unsaved changes' })}</span>
          </div>
          <div className="h-4 w-px bg-border" />
          <Button variant="ghost" size="sm" onClick={handleReset} disabled={saving}>
            {t('common.reset', { defaultValue: 'Reset' })}
          </Button>
          <Button size="sm" onClick={() => void handleSave()} loading={saving}>
            {t('quota_monitoring.save_changes', { defaultValue: 'Save changes' })}
          </Button>
        </div>
      )}
    </QuotaMonitoringShell>
  );
}
