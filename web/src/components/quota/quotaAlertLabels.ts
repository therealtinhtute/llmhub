// Maps canonical quota-monitoring (provider, resource, window) identities onto
// the same i18n label keys used by the Quota Management cards
// (src/utils/quota/constants.ts CLAUDE_USAGE_WINDOW_KEYS and
// src/components/quota/quotaConfigs.ts buildCodexQuotaWindows). Monitoring
// state keeps raw durable identities; this module is presentation-only.

export interface QuotaAlertLabelRef {
  labelKey: string;
  labelParams?: Record<string, string>;
}

const CLAUDE_WINDOW_LABELS: Record<string, string> = {
  'messages/five-hour': 'claude_quota.five_hour',
  'messages/seven-day': 'claude_quota.seven_day',
  'oauth-apps/seven-day': 'claude_quota.seven_day_oauth_apps',
  'opus/seven-day': 'claude_quota.seven_day_opus',
  'sonnet/seven-day': 'claude_quota.seven_day_sonnet',
  'cowork/seven-day': 'claude_quota.seven_day_cowork',
  'iguana-necktie/default': 'claude_quota.seven_day_fable',
  'fable/seven-day': 'claude_quota.seven_day_fable',
};

const CODEX_WINDOW_LABELS: Record<string, string> = {
  'code/five-hour': 'codex_quota.primary_window',
  'code/weekly': 'codex_quota.secondary_window',
  'code/monthly': 'codex_quota.team_secondary_window',
  'code-review/five-hour': 'codex_quota.code_review_primary_window',
  'code-review/weekly': 'codex_quota.code_review_secondary_window',
  'code-review/monthly': 'codex_quota.code_review_team_secondary_window',
};

const CODEX_ADDITIONAL_WINDOW_LABELS: Record<string, string> = {
  'five-hour': 'codex_quota.additional_primary_window',
  weekly: 'codex_quota.additional_secondary_window',
  monthly: 'codex_quota.additional_team_secondary_window',
};

const humanizeSlug = (slug: string) => slug.replace(/[-_]+/g, ' ').trim();

export function quotaAlertWindowLabel(
  provider: string,
  resource: string,
  window: string
): QuotaAlertLabelRef | undefined {
  const identity = `${resource}/${window}`;
  if (provider === 'claude') {
    const labelKey = CLAUDE_WINDOW_LABELS[identity];
    return labelKey ? { labelKey } : undefined;
  }
  if (provider === 'codex') {
    const labelKey = CODEX_WINDOW_LABELS[identity];
    if (labelKey) return { labelKey };
    if (resource.startsWith('additional-')) {
      const additionalKey = CODEX_ADDITIONAL_WINDOW_LABELS[window];
      if (!additionalKey) return undefined;
      const name = humanizeSlug(resource.slice('additional-'.length));
      return { labelKey: additionalKey, labelParams: { name } };
    }
  }
  return undefined;
}
