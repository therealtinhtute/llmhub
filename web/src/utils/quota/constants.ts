/**
 * Quota constants for API URLs, headers, and theme colors.
 */

import type {
  AntigravityQuotaGroupDefinition,
  GeminiCliQuotaGroupDefinition,
  TypeColorSet,
} from '@/types';

// Theme colors for type badges — 与 authFiles/constants.ts 保持同步
export const TYPE_COLORS: Record<string, TypeColorSet> = {
  qwen: {
    light: { bg: '#ede5fd', text: '#5530c7' },
  },
  gemini: {
    light: { bg: '#e3f2fd', text: '#1565c0' },
  },
  'gemini-cli': {
    light: { bg: '#e0e8ff', text: '#1e4fa3' },
  },
  aistudio: {
    light: { bg: '#f0f2f5', text: '#2f343c' },
  },
  claude: {
    light: { bg: '#fbece4', text: '#c05621' },
  },
  codex: {
    light: { bg: '#eae7ff', text: '#3538d4' },
  },
  kimi: {
    light: { bg: '#dce8ff', text: '#0560cf' },
  },
  antigravity: {
    light: { bg: '#e0f7fa', text: '#006064' },
  },
  xai: {
    light: { bg: '#f3f4f6', text: '#111827', border: '1px solid #d1d5db' },
  },
  iflow: {
    light: { bg: '#f5e3fc', text: '#9025c8' },
  },
  vertex: {
    light: { bg: '#e4edfd', text: '#2b5fbc' },
  },
  empty: {
    light: { bg: '#f5f5f5', text: '#616161' },
  },
  unknown: {
    light: { bg: '#f0f0f0', text: '#666666', border: '1px dashed #999999' },
  },
};

// Antigravity API configuration
export const ANTIGRAVITY_QUOTA_URLS = [
  'https://daily-cloudcode-pa.googleapis.com/v1internal:fetchAvailableModels',
  'https://daily-cloudcode-pa.sandbox.googleapis.com/v1internal:fetchAvailableModels',
  'https://cloudcode-pa.googleapis.com/v1internal:fetchAvailableModels',
];

export const ANTIGRAVITY_REQUEST_HEADERS = {
  Authorization: 'Bearer $TOKEN$',
  'Content-Type': 'application/json',
  'User-Agent': 'antigravity/1.11.5 windows/amd64',
};

export const ANTIGRAVITY_QUOTA_GROUPS: AntigravityQuotaGroupDefinition[] = [
  {
    id: 'claude-gpt',
    label: 'Claude/GPT',
    identifiers: ['claude-sonnet-4-6', 'claude-opus-4-6-thinking', 'gpt-oss-120b-medium'],
  },
  {
    id: 'gemini-3-pro',
    label: 'Gemini 3 Pro',
    identifiers: ['gemini-3-pro-high', 'gemini-3-pro-low'],
  },
  {
    id: 'gemini-3-1-pro-series',
    label: 'Gemini 3.1 Pro Series',
    identifiers: ['gemini-3.1-pro-high', 'gemini-3.1-pro-low'],
  },
  {
    id: 'gemini-2-5-flash',
    label: 'Gemini 2.5 Flash',
    identifiers: ['gemini-2.5-flash', 'gemini-2.5-flash-thinking'],
  },
  {
    id: 'gemini-2-5-flash-lite',
    label: 'Gemini 2.5 Flash Lite',
    identifiers: ['gemini-2.5-flash-lite'],
  },
  {
    id: 'gemini-2-5-cu',
    label: 'Gemini 2.5 CU',
    identifiers: ['rev19-uic3-1p'],
  },
  {
    id: 'gemini-3-flash',
    label: 'Gemini 3 Flash',
    identifiers: ['gemini-3-flash'],
  },
  {
    id: 'gemini-image',
    label: 'gemini-3.1-flash-image',
    identifiers: ['gemini-3.1-flash-image'],
    labelFromModel: true,
  },
];

// Gemini CLI API configuration
export const GEMINI_CLI_QUOTA_URL =
  'https://cloudcode-pa.googleapis.com/v1internal:retrieveUserQuota';

export const GEMINI_CLI_CODE_ASSIST_URL =
  'https://cloudcode-pa.googleapis.com/v1internal:loadCodeAssist';

export const GEMINI_CLI_REQUEST_HEADERS = {
  Authorization: 'Bearer $TOKEN$',
  'Content-Type': 'application/json',
};

export const GEMINI_CLI_QUOTA_GROUPS: GeminiCliQuotaGroupDefinition[] = [
  {
    id: 'gemini-flash-lite-series',
    label: 'Gemini Flash Lite Series',
    preferredModelId: 'gemini-2.5-flash-lite',
    modelIds: ['gemini-2.5-flash-lite'],
  },
  {
    id: 'gemini-flash-series',
    label: 'Gemini Flash Series',
    preferredModelId: 'gemini-3-flash-preview',
    modelIds: ['gemini-3-flash-preview', 'gemini-2.5-flash'],
  },
  {
    id: 'gemini-pro-series',
    label: 'Gemini Pro Series',
    preferredModelId: 'gemini-3.1-pro-preview',
    modelIds: ['gemini-3.1-pro-preview', 'gemini-3-pro-preview', 'gemini-2.5-pro'],
  },
];

export const GEMINI_CLI_GROUP_ORDER = new Map(
  GEMINI_CLI_QUOTA_GROUPS.map((group, index) => [group.id, index] as const)
);

export const GEMINI_CLI_GROUP_LOOKUP = new Map(
  GEMINI_CLI_QUOTA_GROUPS.flatMap((group) =>
    group.modelIds.map((modelId) => [modelId, group] as const)
  )
);

export const GEMINI_CLI_IGNORED_MODEL_PREFIXES = ['gemini-2.0-flash'];

// Claude API configuration
export const CLAUDE_PROFILE_URL = 'https://api.anthropic.com/api/oauth/profile';

export const CLAUDE_USAGE_URL = 'https://api.anthropic.com/api/oauth/usage';

export const CLAUDE_REQUEST_HEADERS = {
  Authorization: 'Bearer $TOKEN$',
  'Content-Type': 'application/json',
  'anthropic-beta': 'oauth-2025-04-20',
};

export const CLAUDE_USAGE_WINDOW_KEYS = [
  { key: 'five_hour', id: 'five-hour', labelKey: 'claude_quota.five_hour' },
  { key: 'seven_day', id: 'seven-day', labelKey: 'claude_quota.seven_day' },
  {
    key: 'seven_day_oauth_apps',
    id: 'seven-day-oauth-apps',
    labelKey: 'claude_quota.seven_day_oauth_apps',
  },
  { key: 'seven_day_opus', id: 'seven-day-opus', labelKey: 'claude_quota.seven_day_opus' },
  { key: 'seven_day_sonnet', id: 'seven-day-sonnet', labelKey: 'claude_quota.seven_day_sonnet' },
  { key: 'seven_day_cowork', id: 'seven-day-cowork', labelKey: 'claude_quota.seven_day_cowork' },
  { key: 'iguana_necktie', id: 'iguana-necktie', labelKey: 'claude_quota.iguana_necktie' },
] as const;

// Codex API configuration
export const CODEX_USAGE_URL = 'https://chatgpt.com/backend-api/wham/usage';

export const CODEX_REQUEST_HEADERS = {
  Authorization: 'Bearer $TOKEN$',
  'Content-Type': 'application/json',
  'User-Agent': 'codex_cli_rs/0.76.0 (Debian 13.0.0; x86_64) WindowsTerminal',
};

// Kimi API configuration
export const KIMI_USAGE_URL = 'https://api.kimi.com/coding/v1/usages';

export const KIMI_REQUEST_HEADERS = {
  Authorization: 'Bearer $TOKEN$',
};

// xAI/Grok API configuration
export const XAI_BILLING_URL = 'https://cli-chat-proxy.grok.com/v1/billing';

export const XAI_REQUEST_HEADERS = {
  Authorization: 'Bearer $TOKEN$',
};
