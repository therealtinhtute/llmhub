import { apiClient } from './client';
import type {
  QuotaAlertEvent,
  QuotaAlertPage,
  QuotaAlertPageQuery,
  QuotaAlertProvider,
  QuotaAlertProviderOverride,
  QuotaAlertProviderOverrideUpdate,
  QuotaAlertSettings,
  QuotaAlertSettingsUpdate,
  QuotaAlertState,
  QuotaAlertTelegramRead,
  QuotaAlertTelegramUpdate,
} from '@/types/quotaAlert';

type RawQuotaAlertProviderOverride = {
  provider?: string;
  enabled?: boolean;
  warning_threshold?: number | null;
};

type RawQuotaAlertTelegramRead = {
  enabled?: boolean;
  chat_id?: string;
  token_configured?: boolean;
};

type RawQuotaAlertSettings = {
  revision?: number;
  enabled?: boolean;
  poll_interval_seconds?: number;
  warning_threshold?: number;
  notify_recovery?: boolean;
  reminder_interval_seconds?: number;
  providers?: RawQuotaAlertProviderOverride[];
  telegram?: RawQuotaAlertTelegramRead;
};

type RawQuotaAlertState = {
  auth_id?: string;
  provider?: string;
  resource?: string;
  window?: string;
  auth_label?: string;
  alert?: string;
  health?: string;
  remaining?: number;
  reset_at?: string;
  observed_at?: string;
  transitioned_at?: string;
  updated_at?: string;
  revision?: number;
};

type RawQuotaAlertEventDelivery = {
  status?: string;
  failure_code?: string;
  attempt_count?: number;
  sent_at?: string;
};

type RawQuotaAlertEvent = {
  id?: string;
  auth_id?: string;
  provider?: string;
  resource?: string;
  window?: string;
  auth_label?: string;
  kind?: string;
  from?: string;
  to?: string;
  remaining?: number;
  reset_at?: string;
  occurred_at?: string;
  acknowledged_at?: string;
  delivery?: RawQuotaAlertEventDelivery;
};

type RawQuotaAlertPage<T> = {
  items?: T[];
  next_cursor?: string;
};

const quotaAlertPath = '/quota-alerts';

const normalizeProvider = (value?: string): QuotaAlertProvider =>
  (value || 'claude') as QuotaAlertProvider;

const normalizeProviderOverride = (
  override: RawQuotaAlertProviderOverride
): QuotaAlertProviderOverride => ({
  provider: normalizeProvider(override.provider),
  enabled: Boolean(override.enabled),
  warningThreshold: override.warning_threshold ?? null,
});

const normalizeTelegram = (telegram?: RawQuotaAlertTelegramRead): QuotaAlertTelegramRead => ({
  enabled: Boolean(telegram?.enabled),
  chatId: telegram?.chat_id ?? '',
  tokenConfigured: Boolean(telegram?.token_configured),
});

const normalizeSettings = (settings: RawQuotaAlertSettings): QuotaAlertSettings => ({
  revision: settings.revision ?? 0,
  enabled: Boolean(settings.enabled),
  pollIntervalSeconds: settings.poll_interval_seconds ?? 0,
  warningThreshold: settings.warning_threshold ?? 0,
  notifyRecovery: Boolean(settings.notify_recovery),
  reminderIntervalSeconds: settings.reminder_interval_seconds ?? 0,
  providers: (settings.providers ?? []).map(normalizeProviderOverride),
  telegram: normalizeTelegram(settings.telegram),
});

const normalizeState = (state: RawQuotaAlertState): QuotaAlertState => ({
  authId: state.auth_id ?? '',
  provider: normalizeProvider(state.provider),
  resource: state.resource ?? '',
  window: state.window ?? '',
  authLabel: state.auth_label ?? '',
  alert: (state.alert || 'unknown') as QuotaAlertState['alert'],
  health: (state.health || 'unknown') as QuotaAlertState['health'],
  remaining: state.remaining,
  resetAt: state.reset_at,
  observedAt: state.observed_at ?? '',
  transitionedAt: state.transitioned_at ?? '',
  updatedAt: state.updated_at ?? '',
  revision: state.revision ?? 0,
});

const normalizeEventDelivery = (
  delivery?: RawQuotaAlertEventDelivery
): QuotaAlertEvent['delivery'] =>
  delivery
    ? {
        status: (delivery.status || 'unknown') as NonNullable<QuotaAlertEvent['delivery']>['status'],
        failureCode: delivery.failure_code,
        attemptCount: delivery.attempt_count ?? 0,
        sentAt: delivery.sent_at,
      }
    : undefined;

const normalizeEvent = (event: RawQuotaAlertEvent): QuotaAlertEvent => ({
  id: event.id ?? '',
  authId: event.auth_id ?? '',
  provider: normalizeProvider(event.provider),
  resource: event.resource ?? '',
  window: event.window ?? '',
  authLabel: event.auth_label ?? '',
  kind: (event.kind || 'warning') as QuotaAlertEvent['kind'],
  from: (event.from || 'unknown') as QuotaAlertEvent['from'],
  to: (event.to || 'unknown') as QuotaAlertEvent['to'],
  remaining: event.remaining,
  resetAt: event.reset_at,
  occurredAt: event.occurred_at ?? '',
  acknowledgedAt: event.acknowledged_at,
  delivery: normalizeEventDelivery(event.delivery),
});

const normalizePage = <TRaw, TItem>(
  page: RawQuotaAlertPage<TRaw>,
  normalizeItem: (item: TRaw) => TItem
): QuotaAlertPage<TItem> => ({
  items: (page.items ?? []).map(normalizeItem),
  nextCursor: page.next_cursor,
});

const serializeProviderOverride = (override: QuotaAlertProviderOverrideUpdate) => ({
  provider: override.provider,
  enabled: override.enabled,
  warning_threshold: override.warningThreshold ?? null,
});

const serializeSettingsUpdate = (settings: QuotaAlertSettingsUpdate) => ({
  revision: settings.revision,
  enabled: settings.enabled,
  poll_interval_seconds: settings.pollIntervalSeconds,
  warning_threshold: settings.warningThreshold,
  notify_recovery: settings.notifyRecovery,
  reminder_interval_seconds: settings.reminderIntervalSeconds,
  providers: settings.providers.map(serializeProviderOverride),
});

const serializeTelegramUpdate = (telegram: QuotaAlertTelegramUpdate) => ({
  revision: telegram.revision,
  enabled: telegram.enabled,
  chat_id: telegram.chatId,
  token: telegram.token,
  clear_token: telegram.clearToken ?? false,
});

export const quotaAlertsApi = {
  async getSettings(): Promise<QuotaAlertSettings> {
    const raw = await apiClient.get<RawQuotaAlertSettings>(`${quotaAlertPath}/settings`);
    return normalizeSettings(raw);
  },

  async updateSettings(settings: QuotaAlertSettingsUpdate): Promise<QuotaAlertSettings> {
    const raw = await apiClient.put<RawQuotaAlertSettings>(
      `${quotaAlertPath}/settings`,
      serializeSettingsUpdate(settings)
    );
    return normalizeSettings(raw);
  },

  async updateTelegram(telegram: QuotaAlertTelegramUpdate): Promise<QuotaAlertSettings> {
    const raw = await apiClient.put<RawQuotaAlertSettings>(
      `${quotaAlertPath}/telegram`,
      serializeTelegramUpdate(telegram)
    );
    return normalizeSettings(raw);
  },

  async listStates(query: QuotaAlertPageQuery = {}): Promise<QuotaAlertPage<QuotaAlertState>> {
    const raw = await apiClient.get<RawQuotaAlertPage<RawQuotaAlertState>>(
      `${quotaAlertPath}/state`,
      { params: query }
    );
    return normalizePage(raw, normalizeState);
  },

  async listEvents(query: QuotaAlertPageQuery = {}): Promise<QuotaAlertPage<QuotaAlertEvent>> {
    const raw = await apiClient.get<RawQuotaAlertPage<RawQuotaAlertEvent>>(
      `${quotaAlertPath}/events`,
      { params: query }
    );
    return normalizePage(raw, normalizeEvent);
  },

  acknowledgeEvent: (id: string) =>
    apiClient.post<{ status: string }>(`${quotaAlertPath}/events/${encodeURIComponent(id)}/ack`),

  testTelegram: () => apiClient.post<{ status: string }>(`${quotaAlertPath}/telegram/test`),
};
