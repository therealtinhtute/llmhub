import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { AppCard as Card } from '@/components/ui/AppCard';
import { Button } from '@/components/ui/Button';
import { FormInput } from '@/components/ui/FormInput';
import { LoadingSpinner } from '@/components/ui/LoadingSpinner';
import { ToggleSwitch } from '@/components/ui/ToggleSwitch';
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
const tableCellClass = 'px-3 py-2 align-middle';
const tableRowClass = 'border-b border-border/60 last:border-0';
const badgeClass = 'inline-flex rounded-full border px-2 py-0.5 text-xs font-medium capitalize';

const eventBadgeClass = (value: string) => {
  switch (value) {
    case 'healthy':
    case 'recovery':
    case 'sent':
      return 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700';
    case 'warning':
    case 'pending':
      return 'border-amber-500/30 bg-amber-500/10 text-amber-700';
    case 'exhausted':
    case 'failed':
      return 'border-red-500/30 bg-red-500/10 text-red-700';
    default:
      return 'border-border bg-muted text-muted-foreground';
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
    <div className={styles.container}>
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

  const groupedStates = useMemo(() => {
    const groups = new Map<QuotaAlertProvider, QuotaAlertState[]>();
    states.forEach((state) => {
      const list = groups.get(state.provider) ?? [];
      list.push(state);
      groups.set(state.provider, list);
    });
    return PROVIDERS.map(({ provider, label }) => ({
      provider,
      label,
      states: groups.get(provider) ?? [],
    })).filter((group) => group.states.length > 0);
  }, [states]);

  const healthSummary = useMemo(() => {
    const counts = { exhausted: 0, warning: 0, healthy: 0, unknown: 0 };
    states.forEach((state) => {
      counts[state.alert] += 1;
    });
    return counts;
  }, [states]);

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
    setError('');
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
            <h2 className="text-lg font-semibold text-foreground">
              {t('quota_monitoring.unavailable_title', { defaultValue: 'Quota monitoring is unavailable' })}
            </h2>
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
        <>
          <Button variant="secondary" size="sm" onClick={() => void loadAll()} disabled={loading || saving}>
            {loading ? <LoadingSpinner size={14} /> : null}
            {t('common.refresh', { defaultValue: 'Refresh' })}
          </Button>
          <Button variant="secondary" size="sm" onClick={handleReset} disabled={!dirty || saving}>
            {t('common.reset', { defaultValue: 'Reset' })}
          </Button>
          <Button size="sm" onClick={() => void handleSave()} loading={saving} disabled={!dirty}>
            {t('common.save', { defaultValue: 'Save' })}
          </Button>
        </>
      }
    >
      {error ? <div className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</div> : null}

      <div className="grid gap-3 md:grid-cols-4">
        {[
          { label: t('quota_monitoring.summary_total', { defaultValue: 'Tracked states' }), value: states.length, className: 'text-foreground' },
          { label: t('quota_monitoring.summary_exhausted', { defaultValue: 'Exhausted' }), value: healthSummary.exhausted, className: 'text-red-600' },
          { label: t('quota_monitoring.summary_warning', { defaultValue: 'Warning' }), value: healthSummary.warning, className: 'text-amber-600' },
          { label: t('quota_monitoring.summary_healthy', { defaultValue: 'Healthy' }), value: healthSummary.healthy, className: 'text-emerald-600' },
        ].map((stat, index) => (
          <Reveal key={stat.label} delay={index * 45}>
            <Card className="gap-2 py-4">
              <div className="text-xs font-medium text-muted-foreground">{stat.label}</div>
              <StatNumber value={stat.value} className={stat.className} />
            </Card>
          </Reveal>
        ))}
      </div>

      <Card title={t('quota_monitoring.settings_title', { defaultValue: 'Alert settings' })}>
        <div className="grid gap-4 md:grid-cols-2">
          <ToggleSwitch
            checked={activeSettings.enabled}
            onChange={(enabled) => updateSettings({ enabled })}
            label={t('quota_monitoring.enabled', { defaultValue: 'Enable quota monitoring' })}
          />
          <ToggleSwitch
            checked={activeSettings.notifyRecovery}
            onChange={(notifyRecovery) => updateSettings({ notifyRecovery })}
            label={t('quota_monitoring.notify_recovery', { defaultValue: 'Notify on recovery' })}
          />
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

      <Card title={t('quota_monitoring.providers_title', { defaultValue: 'Provider overrides' })}>
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          {PROVIDERS.map(({ provider, label }) => {
            const override = providerOverrides.get(provider) ?? { provider, enabled: true, warningThreshold: null };
            return (
              <div key={provider} className="flex flex-col gap-3 rounded-lg border border-border bg-card p-3 shadow-sm">
                <div className="flex items-center justify-between gap-3">
                  <div className="text-sm font-semibold text-foreground">{label}</div>
                  <ToggleSwitch
                    checked={override.enabled}
                    onChange={(enabled) => updateProvider(provider, { enabled })}
                    ariaLabel={`${label} enabled`}
                  />
                </div>
                <FormInput
                  id={`quota-provider-${provider}`}
                  type="number"
                  min={0}
                  max={100}
                  className="mt-2"
                  label={t('quota_monitoring.provider_threshold', { defaultValue: 'Threshold override (%)' })}
                  placeholder={String(activeSettings.warningThreshold)}
                  value={override.warningThreshold ?? ''}
                  onChange={(event) =>
                    updateProvider(provider, {
                      warningThreshold: event.target.value === '' ? null : Number(event.target.value),
                    })
                  }
                />
              </div>
            );
          })}
        </div>
      </Card>

      <Card title={t('quota_monitoring.telegram_title', { defaultValue: 'Telegram destination' })}>
        <div className="grid gap-4 md:grid-cols-2">
          <ToggleSwitch
            checked={activeSettings.telegram.enabled}
            onChange={(enabled) => updateTelegram({ enabled })}
            label={t('quota_monitoring.telegram_enabled', { defaultValue: 'Enable Telegram notifications' })}
          />
          <div className="flex items-center">
            <span
              className={cn(
                badgeClass,
                activeSettings.telegram.tokenConfigured
                  ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700'
                  : 'border-border bg-muted text-muted-foreground'
              )}
            >
              {activeSettings.telegram.tokenConfigured
                ? t('quota_monitoring.telegram_token_configured', { defaultValue: 'Bot token configured' })
                : t('quota_monitoring.telegram_token_missing', { defaultValue: 'Bot token not configured' })}
            </span>
          </div>
          <FormInput
            id="quota-telegram-chat"
            label={t('quota_monitoring.telegram_chat_id', { defaultValue: 'Chat ID' })}
            value={activeSettings.telegram.chatId}
            onChange={(event) => updateTelegram({ chatId: event.target.value })}
          />
          <FormInput
            id="quota-telegram-token"
            type="password"
            label={t('quota_monitoring.telegram_token', { defaultValue: 'New bot token' })}
            hint={t('quota_monitoring.telegram_token_hint', {
              defaultValue: 'Leave blank to preserve the stored write-only token.',
            })}
            value={tokenDraft}
            onChange={(event) => {
              setTokenDraft(event.target.value);
              if (event.target.value.trim()) setClearToken(false);
            }}
          />
          <ToggleSwitch
            checked={clearToken}
            onChange={(value) => {
              setClearToken(value);
              if (value) setTokenDraft('');
            }}
            label={t('quota_monitoring.telegram_clear_token', { defaultValue: 'Clear stored token on save' })}
          />
          <Button
            type="button"
            variant="secondary"
            className="w-fit"
            onClick={() => void handleTelegramTest()}
            loading={testingTelegram}
            disabled={dirty || saving || !activeSettings.telegram.enabled || !activeSettings.telegram.tokenConfigured}
          >
            {t('quota_monitoring.telegram_test', { defaultValue: 'Send test message' })}
          </Button>
        </div>
      </Card>

      <Card title={t('quota_monitoring.current_state_title', { defaultValue: 'Current states' })}>
        {groupedStates.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t('quota_monitoring.empty_states', { defaultValue: 'No quota states have been collected yet.' })}</p>
        ) : (
          <div className="flex flex-col gap-4">
            {groupedStates.map((group) => (
              <div key={group.provider} className="overflow-x-auto">
                <h3 className="mb-2 text-sm font-semibold text-foreground">{group.label}</h3>
                <table className={cn(tableClass, 'min-w-[760px]')}>
                  <thead className={tableHeadClass}>
                    <tr>
                      <th className={tableCellClass}>{t('quota_monitoring.auth_label', { defaultValue: 'Auth' })}</th>
                      <th className={tableCellClass}>{t('quota_monitoring.resource', { defaultValue: 'Resource' })}</th>
                      <th className={tableCellClass}>{t('quota_monitoring.window', { defaultValue: 'Window' })}</th>
                      <th className={tableCellClass}>{t('quota_monitoring.alert', { defaultValue: 'Alert' })}</th>
                      <th className={tableCellClass}>{t('quota_monitoring.remaining', { defaultValue: 'Remaining' })}</th>
                      <th className={tableCellClass}>{t('quota_monitoring.reset_at', { defaultValue: 'Reset at' })}</th>
                      <th className={tableCellClass}>{t('quota_monitoring.updated_at', { defaultValue: 'Updated' })}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {group.states.map((state) => (
                      <tr key={`${state.authId}:${state.resource}:${state.window}`} className={tableRowClass}>
                        <td className={cn(tableCellClass, 'font-medium text-foreground')}>{state.authLabel}</td>
                        <td className={tableCellClass}>{state.resource}</td>
                        <td className={tableCellClass}>{state.window}</td>
                        <td className={tableCellClass}>
                          <span className={cn(badgeClass, eventBadgeClass(state.alert))}>{state.alert}</span>
                        </td>
                        <td className={cn(tableCellClass, 'tabular-nums')}>{formatPercent(state.remaining)}</td>
                        <td className={tableCellClass}>{formatTime(state.resetAt)}</td>
                        <td className={tableCellClass}>{formatTime(state.updatedAt)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
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
      </Card>

      <Card title={t('quota_monitoring.events_title', { defaultValue: 'Recent events' })}>
        <p className="mb-3 text-xs text-muted-foreground">
          {t('quota_monitoring.delivery_note', {
            defaultValue: 'Telegram delivery uses the durable outbox; this API exposes in-app acknowledgement state.',
          })}
        </p>
        {events.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t('quota_monitoring.empty_events', { defaultValue: 'No alert events yet.' })}</p>
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
                    <td className={cn(tableCellClass, 'tabular-nums')}>{formatPercent(event.remaining)}</td>
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
                      <span className={cn(badgeClass, event.acknowledgedAt ? 'border-border bg-muted text-muted-foreground' : 'border-primary/30 bg-primary/10 text-primary')}>
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
              <Button
                type="button"
                variant="secondary"
                className="mt-3 w-fit"
                loading={loadingMoreEvents}
                onClick={() => void handleLoadMoreEvents()}
              >
                {t('quota_monitoring.load_more_events', { defaultValue: 'Load more events' })}
              </Button>
            ) : null}
          </div>
        )}
      </Card>
    </QuotaMonitoringShell>
  );
}
