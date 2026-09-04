export type QuotaAlertProvider =
  | 'claude'
  | 'codex'
  | 'gemini-cli'
  | 'antigravity'
  | 'kimi'
  | 'xai'
  | 'kiro';

export type QuotaAlertLevel = 'healthy' | 'warning' | 'exhausted' | 'unknown';
export type QuotaAlertHealth = 'reliable' | 'unknown';
export type QuotaAlertEventKind = 'warning' | 'exhausted' | 'recovery' | 'reminder';

export interface QuotaAlertProviderOverride {
  provider: QuotaAlertProvider;
  enabled: boolean;
  warningThreshold?: number | null;
}

export interface QuotaAlertTelegramRead {
  enabled: boolean;
  chatId: string;
  tokenConfigured: boolean;
  secretKeyConfigured?: boolean;
}

export interface QuotaAlertSettings {
  revision: number;
  enabled: boolean;
  pollIntervalSeconds: number;
  warningThreshold: number;
  notifyRecovery: boolean;
  reminderIntervalSeconds: number;
  providers: QuotaAlertProviderOverride[];
  telegram: QuotaAlertTelegramRead;
}

export interface QuotaAlertProviderOverrideUpdate {
  provider: QuotaAlertProvider;
  enabled: boolean;
  warningThreshold?: number | null;
}

export interface QuotaAlertSettingsUpdate {
  revision: number;
  enabled: boolean;
  pollIntervalSeconds: number;
  warningThreshold: number;
  notifyRecovery: boolean;
  reminderIntervalSeconds: number;
  providers: QuotaAlertProviderOverrideUpdate[];
}

export interface QuotaAlertTelegramUpdate {
  revision: number;
  enabled: boolean;
  chatId: string;
  token?: string;
  clearToken?: boolean;
}

export interface QuotaAlertState {
  authId: string;
  provider: QuotaAlertProvider;
  resource: string;
  window: string;
  authLabel: string;
  alert: QuotaAlertLevel;
  health: QuotaAlertHealth;
  remaining?: number;
  resetAt?: string;
  observedAt: string;
  transitionedAt: string;
  updatedAt: string;
  revision: number;
}

export interface QuotaAlertEventDelivery {
  status: 'pending' | 'sent' | 'failed' | 'unknown';
  failureCode?: string;
  attemptCount: number;
  sentAt?: string;
}

export interface QuotaAlertEvent {
  id: string;
  authId: string;
  provider: QuotaAlertProvider;
  resource: string;
  window: string;
  authLabel: string;
  kind: QuotaAlertEventKind;
  from: QuotaAlertLevel;
  to: QuotaAlertLevel;
  remaining?: number;
  resetAt?: string;
  occurredAt: string;
  acknowledgedAt?: string;
  delivery?: QuotaAlertEventDelivery;
}

export interface QuotaAlertPage<T> {
  items: T[];
  nextCursor?: string;
}

export interface QuotaAlertPageQuery {
  cursor?: string;
  limit?: number;
}
