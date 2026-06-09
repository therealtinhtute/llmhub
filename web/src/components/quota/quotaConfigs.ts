/**
 * Quota configuration definitions.
 */

import React from 'react';
import type { ReactNode } from 'react';
import type { TFunction } from 'i18next';
import type {
  AntigravityQuotaGroup,
  AntigravityModelsPayload,
  AntigravityQuotaState,
  AuthFileItem,
  ClaudeExtraUsage,
  ClaudeProfileResponse,
  ClaudeQuotaState,
  ClaudeQuotaWindow,
  ClaudeUsagePayload,
  CodexRateLimitInfo,
  CodexQuotaState,
  CodexUsageWindow,
  CodexQuotaWindow,
  CodexUsagePayload,
  GeminiCliCodeAssistPayload,
  GeminiCliCredits,
  GeminiCliParsedBucket,
  GeminiCliQuotaBucketState,
  GeminiCliQuotaState,
  GeminiCliUserTier,
  KiroProviderQuotaState,
  KiroProviderQuotaRow,
  KiroQuotaState,
  KiroRuntimeModelQuotaState,
  KiroRuntimeQuotaState,
  KiroRuntimeStatus,
  KiroRuntimeUsageStats,
  KimiQuotaRow,
  KimiQuotaState,
  XaiBillingConfig,
  XaiBillingSummary,
  XaiQuotaState,
} from '@/types';
import { apiCallApi, authFilesApi, getApiCallErrorMessage } from '@/services/api';
import { useQuotaStore } from '@/stores';
import {
  ANTIGRAVITY_QUOTA_URLS,
  ANTIGRAVITY_REQUEST_HEADERS,
  CLAUDE_PROFILE_URL,
  CLAUDE_USAGE_URL,
  CLAUDE_REQUEST_HEADERS,
  CLAUDE_USAGE_WINDOW_KEYS,
  CODEX_USAGE_URL,
  CODEX_REQUEST_HEADERS,
  GEMINI_CLI_QUOTA_URL,
  GEMINI_CLI_CODE_ASSIST_URL,
  GEMINI_CLI_REQUEST_HEADERS,
  KIMI_USAGE_URL,
  KIMI_REQUEST_HEADERS,
  XAI_BILLING_URL,
  XAI_REQUEST_HEADERS,
  normalizeGeminiCliModelId,
  normalizeNumberValue,
  normalizePlanType,
  normalizeQuotaFraction,
  normalizeStringValue,
  parseAntigravityPayload,
  parseClaudeUsagePayload,
  parseCodexUsagePayload,
  parseGeminiCliQuotaPayload,
  parseGeminiCliCodeAssistPayload,
  parseKimiUsagePayload,
  parseXaiBillingPayload,
  resolveCodexChatgptAccountId,
  resolveCodexPlanType,
  resolveGeminiCliProjectId,
  formatCodexResetLabel,
  formatQuotaResetTime,
  formatKimiResetHint,
  buildAntigravityQuotaGroups,
  buildGeminiCliQuotaBuckets,
  buildKimiQuotaRows,
  createStatusError,
  getStatusFromError,
  isAntigravityFile,
  isClaudeFile,
  isCodexFile,
  isDisabledAuthFile,
  isGeminiCliFile,
  isKiroFile,
  isKimiFile,
  isRuntimeOnlyAuthFile,
  isXaiFile,
} from '@/utils/quota';
import { normalizeAuthIndex } from '@/utils/authIndex';
import type { QuotaRenderHelpers } from './QuotaCard';
import { quotaStyles as styles } from './quotaStyles';

type QuotaUpdater<T> = T | ((prev: T) => T);

type QuotaType = 'antigravity' | 'claude' | 'codex' | 'gemini-cli' | 'kiro' | 'kimi' | 'xai';

const DEFAULT_ANTIGRAVITY_PROJECT_ID = 'bamboo-precept-lgxtn';
const geminiCliSupplementaryRequestIds = new Map<string, number>();
const geminiCliSupplementaryCache = new Map<
  string,
  {
    requestId: number;
    tierLabel: string | null;
    tierId: string | null;
    creditBalance: number | null;
  }
>();

export interface QuotaStore {
  antigravityQuota: Record<string, AntigravityQuotaState>;
  claudeQuota: Record<string, ClaudeQuotaState>;
  codexQuota: Record<string, CodexQuotaState>;
  geminiCliQuota: Record<string, GeminiCliQuotaState>;
  kiroQuota: Record<string, KiroQuotaState>;
  kimiQuota: Record<string, KimiQuotaState>;
  xaiQuota: Record<string, XaiQuotaState>;
  setAntigravityQuota: (updater: QuotaUpdater<Record<string, AntigravityQuotaState>>) => void;
  setClaudeQuota: (updater: QuotaUpdater<Record<string, ClaudeQuotaState>>) => void;
  setCodexQuota: (updater: QuotaUpdater<Record<string, CodexQuotaState>>) => void;
  setGeminiCliQuota: (updater: QuotaUpdater<Record<string, GeminiCliQuotaState>>) => void;
  setKiroQuota: (updater: QuotaUpdater<Record<string, KiroQuotaState>>) => void;
  setKimiQuota: (updater: QuotaUpdater<Record<string, KimiQuotaState>>) => void;
  setXaiQuota: (updater: QuotaUpdater<Record<string, XaiQuotaState>>) => void;
  clearQuotaCache: () => void;
}

export interface QuotaConfig<TState, TData> {
  type: QuotaType;
  i18nPrefix: string;
  cardIdleMessageKey?: string;
  filterFn: (file: AuthFileItem) => boolean;
  fetchQuota: (file: AuthFileItem, t: TFunction) => Promise<TData>;
  buildRuntimeState?: (file: AuthFileItem) => TState;
  storeSelector: (state: QuotaStore) => Record<string, TState>;
  storeSetter: keyof QuotaStore;
  buildLoadingState: () => TState;
  buildSuccessState: (data: TData) => TState;
  buildErrorState: (message: string, status?: number) => TState;
  cardClassName: string;
  controlsClassName: string;
  controlClassName: string;
  gridClassName: string;
  renderQuotaItems: (quota: TState, t: TFunction, helpers: QuotaRenderHelpers) => ReactNode;
}

const resolveAntigravityProjectId = async (file: AuthFileItem): Promise<string> => {
  try {
    const text = await authFilesApi.downloadText(file.name);
    const trimmed = text.trim();
    if (!trimmed) return DEFAULT_ANTIGRAVITY_PROJECT_ID;

    const parsed = JSON.parse(trimmed) as Record<string, unknown>;
    const topLevel = normalizeStringValue(parsed.project_id ?? parsed.projectId);
    if (topLevel) return topLevel;

    const installed =
      parsed.installed && typeof parsed.installed === 'object' && parsed.installed !== null
        ? (parsed.installed as Record<string, unknown>)
        : null;
    const installedProjectId = installed
      ? normalizeStringValue(installed.project_id ?? installed.projectId)
      : null;
    if (installedProjectId) return installedProjectId;

    const web =
      parsed.web && typeof parsed.web === 'object' && parsed.web !== null
        ? (parsed.web as Record<string, unknown>)
        : null;
    const webProjectId = web ? normalizeStringValue(web.project_id ?? web.projectId) : null;
    if (webProjectId) return webProjectId;
  } catch {
    return DEFAULT_ANTIGRAVITY_PROJECT_ID;
  }

  return DEFAULT_ANTIGRAVITY_PROJECT_ID;
};

const fetchAntigravityQuota = async (
  file: AuthFileItem,
  t: TFunction
): Promise<AntigravityQuotaGroup[]> => {
  const rawAuthIndex = file['auth_index'] ?? file.authIndex;
  const authIndex = normalizeAuthIndex(rawAuthIndex);
  if (!authIndex) {
    throw new Error(t('antigravity_quota.missing_auth_index'));
  }

  const projectId = await resolveAntigravityProjectId(file);
  const requestBody = JSON.stringify({ project: projectId });

  let lastError = '';
  let lastStatus: number | undefined;
  let priorityStatus: number | undefined;
  let hadSuccess = false;

  for (const url of ANTIGRAVITY_QUOTA_URLS) {
    try {
      const result = await apiCallApi.request({
        authIndex,
        method: 'POST',
        url,
        header: { ...ANTIGRAVITY_REQUEST_HEADERS },
        data: requestBody,
      });

      if (result.statusCode < 200 || result.statusCode >= 300) {
        lastError = getApiCallErrorMessage(result);
        lastStatus = result.statusCode;
        if (result.statusCode === 403 || result.statusCode === 404) {
          priorityStatus ??= result.statusCode;
        }
        continue;
      }

      hadSuccess = true;
      const payload = parseAntigravityPayload(result.body ?? result.bodyText);
      const models = payload?.models;
      if (!models || typeof models !== 'object' || Array.isArray(models)) {
        lastError = t('antigravity_quota.empty_models');
        continue;
      }

      const groups = buildAntigravityQuotaGroups(models as AntigravityModelsPayload);
      if (groups.length === 0) {
        lastError = t('antigravity_quota.empty_models');
        continue;
      }

      return groups;
    } catch (err: unknown) {
      lastError = err instanceof Error ? err.message : t('common.unknown_error');
      const status = getStatusFromError(err);
      if (status) {
        lastStatus = status;
        if (status === 403 || status === 404) {
          priorityStatus ??= status;
        }
      }
    }
  }

  if (hadSuccess) {
    return [];
  }

  throw createStatusError(lastError || t('common.unknown_error'), priorityStatus ?? lastStatus);
};

const buildCodexQuotaWindows = (payload: CodexUsagePayload, t: TFunction): CodexQuotaWindow[] => {
  const FIVE_HOUR_SECONDS = 18000;
  const WEEK_SECONDS = 604800;
  const WINDOW_META = {
    codeFiveHour: { id: 'five-hour', labelKey: 'codex_quota.primary_window' },
    codeWeekly: { id: 'weekly', labelKey: 'codex_quota.secondary_window' },
    codeReviewFiveHour: {
      id: 'code-review-five-hour',
      labelKey: 'codex_quota.code_review_primary_window',
    },
    codeReviewWeekly: {
      id: 'code-review-weekly',
      labelKey: 'codex_quota.code_review_secondary_window',
    },
  } as const;

  const rateLimit = payload.rate_limit ?? payload.rateLimit ?? undefined;
  const codeReviewLimit =
    payload.code_review_rate_limit ?? payload.codeReviewRateLimit ?? undefined;
  const additionalRateLimits = payload.additional_rate_limits ?? payload.additionalRateLimits ?? [];
  const windows: CodexQuotaWindow[] = [];

  const addWindow = (
    id: string,
    label: string,
    labelKey: string | undefined,
    labelParams: Record<string, string | number> | undefined,
    window?: CodexUsageWindow | null,
    limitReached?: boolean,
    allowed?: boolean
  ) => {
    if (!window) return;
    const resetLabel = formatCodexResetLabel(window);
    const usedPercentRaw = normalizeNumberValue(window.used_percent ?? window.usedPercent);
    const isLimitReached = Boolean(limitReached) || allowed === false;
    const usedPercent = usedPercentRaw ?? (isLimitReached && resetLabel !== '-' ? 100 : null);
    windows.push({
      id,
      label,
      labelKey,
      labelParams,
      usedPercent,
      resetLabel,
    });
  };

  const getWindowSeconds = (window?: CodexUsageWindow | null): number | null => {
    if (!window) return null;
    return normalizeNumberValue(window.limit_window_seconds ?? window.limitWindowSeconds);
  };

  const rawLimitReached = rateLimit?.limit_reached ?? rateLimit?.limitReached;
  const rawAllowed = rateLimit?.allowed;

  const pickClassifiedWindows = (
    limitInfo?: CodexRateLimitInfo | null,
    options?: { allowOrderFallback?: boolean }
  ): { fiveHourWindow: CodexUsageWindow | null; weeklyWindow: CodexUsageWindow | null } => {
    const allowOrderFallback = options?.allowOrderFallback ?? true;
    const primaryWindow = limitInfo?.primary_window ?? limitInfo?.primaryWindow ?? null;
    const secondaryWindow = limitInfo?.secondary_window ?? limitInfo?.secondaryWindow ?? null;
    const rawWindows = [primaryWindow, secondaryWindow];

    let fiveHourWindow: CodexUsageWindow | null = null;
    let weeklyWindow: CodexUsageWindow | null = null;

    for (const window of rawWindows) {
      if (!window) continue;
      const seconds = getWindowSeconds(window);
      if (seconds === FIVE_HOUR_SECONDS && !fiveHourWindow) {
        fiveHourWindow = window;
      } else if (seconds === WEEK_SECONDS && !weeklyWindow) {
        weeklyWindow = window;
      }
    }

    // For legacy payloads without window duration, fallback to primary/secondary ordering.
    if (allowOrderFallback) {
      if (!fiveHourWindow) {
        fiveHourWindow = primaryWindow && primaryWindow !== weeklyWindow ? primaryWindow : null;
      }
      if (!weeklyWindow) {
        weeklyWindow =
          secondaryWindow && secondaryWindow !== fiveHourWindow ? secondaryWindow : null;
      }
    }

    return { fiveHourWindow, weeklyWindow };
  };

  const rateWindows = pickClassifiedWindows(rateLimit);
  addWindow(
    WINDOW_META.codeFiveHour.id,
    t(WINDOW_META.codeFiveHour.labelKey),
    WINDOW_META.codeFiveHour.labelKey,
    undefined,
    rateWindows.fiveHourWindow,
    rawLimitReached,
    rawAllowed
  );
  addWindow(
    WINDOW_META.codeWeekly.id,
    t(WINDOW_META.codeWeekly.labelKey),
    WINDOW_META.codeWeekly.labelKey,
    undefined,
    rateWindows.weeklyWindow,
    rawLimitReached,
    rawAllowed
  );

  const codeReviewWindows = pickClassifiedWindows(codeReviewLimit);
  const codeReviewLimitReached = codeReviewLimit?.limit_reached ?? codeReviewLimit?.limitReached;
  const codeReviewAllowed = codeReviewLimit?.allowed;
  addWindow(
    WINDOW_META.codeReviewFiveHour.id,
    t(WINDOW_META.codeReviewFiveHour.labelKey),
    WINDOW_META.codeReviewFiveHour.labelKey,
    undefined,
    codeReviewWindows.fiveHourWindow,
    codeReviewLimitReached,
    codeReviewAllowed
  );
  addWindow(
    WINDOW_META.codeReviewWeekly.id,
    t(WINDOW_META.codeReviewWeekly.labelKey),
    WINDOW_META.codeReviewWeekly.labelKey,
    undefined,
    codeReviewWindows.weeklyWindow,
    codeReviewLimitReached,
    codeReviewAllowed
  );

  const normalizeWindowId = (raw: string) =>
    raw
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '');

  if (Array.isArray(additionalRateLimits)) {
    additionalRateLimits.forEach((limitItem, index) => {
      const rateInfo = limitItem?.rate_limit ?? limitItem?.rateLimit ?? null;
      if (!rateInfo) return;

      const limitName =
        normalizeStringValue(limitItem?.limit_name ?? limitItem?.limitName) ??
        normalizeStringValue(limitItem?.metered_feature ?? limitItem?.meteredFeature) ??
        `additional-${index + 1}`;

      const idPrefix = normalizeWindowId(limitName) || `additional-${index + 1}`;
      const additionalPrimaryWindow = rateInfo.primary_window ?? rateInfo.primaryWindow ?? null;
      const additionalSecondaryWindow =
        rateInfo.secondary_window ?? rateInfo.secondaryWindow ?? null;
      const additionalLimitReached = rateInfo.limit_reached ?? rateInfo.limitReached;
      const additionalAllowed = rateInfo.allowed;

      addWindow(
        `${idPrefix}-five-hour-${index}`,
        t('codex_quota.additional_primary_window', { name: limitName }),
        'codex_quota.additional_primary_window',
        { name: limitName },
        additionalPrimaryWindow,
        additionalLimitReached,
        additionalAllowed
      );
      addWindow(
        `${idPrefix}-weekly-${index}`,
        t('codex_quota.additional_secondary_window', { name: limitName }),
        'codex_quota.additional_secondary_window',
        { name: limitName },
        additionalSecondaryWindow,
        additionalLimitReached,
        additionalAllowed
      );
    });
  }

  return windows;
};

const fetchCodexQuota = async (
  file: AuthFileItem,
  t: TFunction
): Promise<{ planType: string | null; windows: CodexQuotaWindow[] }> => {
  const rawAuthIndex = file['auth_index'] ?? file.authIndex;
  const authIndex = normalizeAuthIndex(rawAuthIndex);
  if (!authIndex) {
    throw new Error(t('codex_quota.missing_auth_index'));
  }

  const planTypeFromFile = resolveCodexPlanType(file);
  const accountId = resolveCodexChatgptAccountId(file);

  const requestHeader: Record<string, string> = {
    ...CODEX_REQUEST_HEADERS,
  };
  if (accountId) {
    requestHeader['Chatgpt-Account-Id'] = accountId;
  }

  const result = await apiCallApi.request({
    authIndex,
    method: 'GET',
    url: CODEX_USAGE_URL,
    header: requestHeader,
  });

  if (result.statusCode < 200 || result.statusCode >= 300) {
    throw createStatusError(getApiCallErrorMessage(result), result.statusCode);
  }

  const payload = parseCodexUsagePayload(result.body ?? result.bodyText);
  if (!payload) {
    throw new Error(t('codex_quota.empty_windows'));
  }

  const planTypeFromUsage = normalizePlanType(payload.plan_type ?? payload.planType);
  const windows = buildCodexQuotaWindows(payload, t);
  return { planType: planTypeFromUsage ?? planTypeFromFile, windows };
};

const GEMINI_CLI_G1_CREDIT_TYPE = 'GOOGLE_ONE_AI';

const GEMINI_CLI_TIER_LABELS: Record<string, string> = {
  'free-tier': 'tier_free',
  'legacy-tier': 'tier_legacy',
  'standard-tier': 'tier_standard',
  'g1-pro-tier': 'tier_pro',
  'g1-ultra-tier': 'tier_ultra',
};

const resolveGeminiCliTierLabel = (
  payload: GeminiCliCodeAssistPayload | null,
  t: TFunction
): string | null => {
  if (!payload) return null;
  const currentTier: GeminiCliUserTier | null | undefined =
    payload.currentTier ?? payload.current_tier;
  const paidTier: GeminiCliUserTier | null | undefined = payload.paidTier ?? payload.paid_tier;
  const rawId = normalizeStringValue(paidTier?.id) ?? normalizeStringValue(currentTier?.id);
  if (!rawId) return null;
  const tierId = rawId.toLowerCase();
  const labelKey = GEMINI_CLI_TIER_LABELS[tierId];
  return labelKey ? t(`gemini_cli_quota.${labelKey}`) : rawId;
};

const resolveGeminiCliTierId = (payload: GeminiCliCodeAssistPayload | null): string | null => {
  if (!payload) return null;
  const currentTier: GeminiCliUserTier | null | undefined =
    payload.currentTier ?? payload.current_tier;
  const paidTier: GeminiCliUserTier | null | undefined = payload.paidTier ?? payload.paid_tier;
  const rawId = normalizeStringValue(paidTier?.id) ?? normalizeStringValue(currentTier?.id);
  return rawId ? rawId.toLowerCase() : null;
};

const resolveGeminiCliCreditBalance = (
  payload: GeminiCliCodeAssistPayload | null
): number | null => {
  if (!payload) return null;
  const paidTier: GeminiCliUserTier | null | undefined = payload.paidTier ?? payload.paid_tier;
  const currentTier: GeminiCliUserTier | null | undefined =
    payload.currentTier ?? payload.current_tier;
  const tier = paidTier ?? currentTier;
  if (!tier) return null;
  const credits: GeminiCliCredits[] = tier.availableCredits ?? tier.available_credits ?? [];
  let total = 0;
  let found = false;
  for (const credit of credits) {
    const creditType = normalizeStringValue(credit.creditType ?? credit.credit_type);
    if (creditType !== GEMINI_CLI_G1_CREDIT_TYPE) continue;
    const amount = normalizeNumberValue(credit.creditAmount ?? credit.credit_amount);
    if (amount !== null) {
      total += amount;
      found = true;
    }
  }
  return found ? total : null;
};

const fetchGeminiCliCodeAssist = async (
  authIndex: string,
  projectId: string,
  t: TFunction
): Promise<{ tierLabel: string | null; tierId: string | null; creditBalance: number | null }> => {
  try {
    const result = await apiCallApi.request({
      authIndex,
      method: 'POST',
      url: GEMINI_CLI_CODE_ASSIST_URL,
      header: { ...GEMINI_CLI_REQUEST_HEADERS },
      data: JSON.stringify({
        cloudaicompanionProject: projectId,
        metadata: {
          ideType: 'IDE_UNSPECIFIED',
          platform: 'PLATFORM_UNSPECIFIED',
          pluginType: 'GEMINI',
          duetProject: projectId,
        },
      }),
    });

    if (result.statusCode < 200 || result.statusCode >= 300) {
      return { tierLabel: null, tierId: null, creditBalance: null };
    }

    const payload = parseGeminiCliCodeAssistPayload(result.body ?? result.bodyText);
    return {
      tierLabel: resolveGeminiCliTierLabel(payload, t),
      tierId: resolveGeminiCliTierId(payload),
      creditBalance: resolveGeminiCliCreditBalance(payload),
    };
  } catch {
    return { tierLabel: null, tierId: null, creditBalance: null };
  }
};

const readGeminiCliSupplementarySnapshot = (
  fileName: string,
  requestId: number
): { tierLabel: string | null; tierId: string | null; creditBalance: number | null } => {
  const cached = geminiCliSupplementaryCache.get(fileName);
  if (!cached || cached.requestId !== requestId) {
    return { tierLabel: null, tierId: null, creditBalance: null };
  }

  return {
    tierLabel: cached.tierLabel,
    tierId: cached.tierId,
    creditBalance: cached.creditBalance,
  };
};

const scheduleGeminiCliSupplementaryRefresh = (
  fileName: string,
  authIndex: string,
  projectId: string,
  t: TFunction
): number => {
  const requestId = (geminiCliSupplementaryRequestIds.get(fileName) ?? 0) + 1;
  geminiCliSupplementaryRequestIds.set(fileName, requestId);
  geminiCliSupplementaryCache.delete(fileName);

  void (async () => {
    const supplementary = await fetchGeminiCliCodeAssist(authIndex, projectId, t);
    if (geminiCliSupplementaryRequestIds.get(fileName) !== requestId) {
      return;
    }

    geminiCliSupplementaryCache.set(fileName, { requestId, ...supplementary });

    useQuotaStore.getState().setGeminiCliQuota((prev) => {
      const current = prev[fileName];
      if (!current || current.status !== 'success') {
        return prev;
      }

      if (
        current.tierLabel === supplementary.tierLabel &&
        current.tierId === supplementary.tierId &&
        current.creditBalance === supplementary.creditBalance
      ) {
        return prev;
      }

      return {
        ...prev,
        [fileName]: {
          ...current,
          tierLabel: supplementary.tierLabel,
          tierId: supplementary.tierId,
          creditBalance: supplementary.creditBalance,
        },
      };
    });
  })();

  return requestId;
};

const fetchGeminiCliQuota = async (
  file: AuthFileItem,
  t: TFunction
): Promise<{
  fileName: string;
  supplementaryRequestId: number;
  buckets: GeminiCliQuotaBucketState[];
  tierLabel: string | null;
  tierId: string | null;
  creditBalance: number | null;
}> => {
  const rawAuthIndex = file['auth_index'] ?? file.authIndex;
  const authIndex = normalizeAuthIndex(rawAuthIndex);
  if (!authIndex) {
    throw new Error(t('gemini_cli_quota.missing_auth_index'));
  }

  const projectId = resolveGeminiCliProjectId(file);
  if (!projectId) {
    throw new Error(t('gemini_cli_quota.missing_project_id'));
  }

  const quotaResponse = await apiCallApi.request({
    authIndex,
    method: 'POST',
    url: GEMINI_CLI_QUOTA_URL,
    header: { ...GEMINI_CLI_REQUEST_HEADERS },
    data: JSON.stringify({ project: projectId }),
  });
  if (quotaResponse.statusCode < 200 || quotaResponse.statusCode >= 300) {
    throw createStatusError(getApiCallErrorMessage(quotaResponse), quotaResponse.statusCode);
  }

  const payload = parseGeminiCliQuotaPayload(quotaResponse.body ?? quotaResponse.bodyText);
  const buckets = Array.isArray(payload?.buckets) ? payload?.buckets : [];

  const parsedBuckets = buckets
    .map((bucket) => {
      const modelId = normalizeGeminiCliModelId(bucket.modelId ?? bucket.model_id);
      if (!modelId) return null;
      const tokenType = normalizeStringValue(bucket.tokenType ?? bucket.token_type);
      const remainingFractionRaw = normalizeQuotaFraction(
        bucket.remainingFraction ?? bucket.remaining_fraction
      );
      const remainingAmount = normalizeNumberValue(
        bucket.remainingAmount ?? bucket.remaining_amount
      );
      const resetTime = normalizeStringValue(bucket.resetTime ?? bucket.reset_time) ?? undefined;
      let fallbackFraction: number | null = null;
      if (remainingAmount !== null) {
        fallbackFraction = remainingAmount <= 0 ? 0 : null;
      } else if (resetTime) {
        fallbackFraction = 0;
      }
      const remainingFraction = remainingFractionRaw ?? fallbackFraction;
      return {
        modelId,
        tokenType,
        remainingFraction,
        remainingAmount,
        resetTime,
      };
    })
    .filter((bucket): bucket is GeminiCliParsedBucket => bucket !== null);

  const builtBuckets = buildGeminiCliQuotaBuckets(parsedBuckets);
  const supplementaryRequestId = scheduleGeminiCliSupplementaryRefresh(
    file.name,
    authIndex,
    projectId,
    t
  );
  const supplementarySnapshot = readGeminiCliSupplementarySnapshot(
    file.name,
    supplementaryRequestId
  );

  return {
    fileName: file.name,
    supplementaryRequestId,
    buckets: builtBuckets,
    tierLabel: supplementarySnapshot.tierLabel,
    tierId: supplementarySnapshot.tierId,
    creditBalance: supplementarySnapshot.creditBalance,
  };
};

const renderAntigravityItems = (
  quota: AntigravityQuotaState,
  t: TFunction,
  helpers: QuotaRenderHelpers
): ReactNode => {
  const { styles: styleMap, QuotaProgressBar } = helpers;
  const { createElement: h } = React;
  const groups = quota.groups ?? [];

  if (groups.length === 0) {
    return h('div', { className: styleMap.quotaMessage }, t('antigravity_quota.empty_models'));
  }

  return groups.map((group) => {
    const clamped = Math.max(0, Math.min(1, group.remainingFraction));
    const percent = Math.round(clamped * 100);
    const resetLabel = formatQuotaResetTime(group.resetTime);

    return h(
      'div',
      { key: group.id, className: styleMap.quotaRow },
      h(
        'div',
        { className: styleMap.quotaRowHeader },
        h('span', { className: styleMap.quotaModel, title: group.models.join(', ') }, group.label),
        h(
          'div',
          { className: styleMap.quotaMeta },
          h('span', { className: styleMap.quotaPercent }, `${percent}%`),
          h('span', { className: styleMap.quotaReset }, resetLabel)
        )
      ),
      h(QuotaProgressBar, { percent })
    );
  });
};

const PREMIUM_GEMINI_CLI_TIER_IDS = new Set(['g1-ultra-tier']);
const PREMIUM_CODEX_PLAN_TYPES = new Set(['pro', 'prolite', 'pro-lite', 'pro_lite']);

const renderCodexItems = (
  quota: CodexQuotaState,
  t: TFunction,
  helpers: QuotaRenderHelpers
): ReactNode => {
  const { styles: styleMap, QuotaProgressBar } = helpers;
  const { createElement: h, Fragment } = React;
  const windows = quota.windows ?? [];
  const planType = quota.planType ?? null;

  const getPlanLabel = (pt?: string | null): string | null => {
    const normalized = normalizePlanType(pt);
    if (!normalized) return null;
    if (normalized === 'pro') return t('codex_quota.plan_pro');
    if (PREMIUM_CODEX_PLAN_TYPES.has(normalized) && normalized !== 'pro') {
      return t('codex_quota.plan_prolite');
    }
    if (normalized === 'plus') return t('codex_quota.plan_plus');
    if (normalized === 'team') return t('codex_quota.plan_team');
    if (normalized === 'free') return t('codex_quota.plan_free');
    return pt || normalized;
  };

  const planLabel = getPlanLabel(planType);
  const isPremiumPlan = PREMIUM_CODEX_PLAN_TYPES.has(normalizePlanType(planType) ?? '');
  const nodes: ReactNode[] = [];

  if (planLabel) {
    const valueClass = isPremiumPlan ? styleMap.premiumPlanValue : styleMap.codexPlanValue;
    nodes.push(
      h(
        'div',
        { key: 'plan', className: styleMap.codexPlan },
        h('span', { className: styleMap.codexPlanLabel }, t('codex_quota.plan_label')),
        h('span', { className: valueClass }, planLabel)
      )
    );
  }

  if (windows.length === 0) {
    nodes.push(
      h('div', { key: 'empty', className: styleMap.quotaMessage }, t('codex_quota.empty_windows'))
    );
    return h(Fragment, null, ...nodes);
  }

  nodes.push(
    ...windows.map((window) => {
      const used = window.usedPercent;
      const clampedUsed = used === null ? null : Math.max(0, Math.min(100, used));
      const remaining = clampedUsed === null ? null : Math.max(0, Math.min(100, 100 - clampedUsed));
      const percentLabel = remaining === null ? '--' : `${Math.round(remaining)}%`;
      const windowLabel = window.labelKey
        ? t(window.labelKey, window.labelParams as Record<string, string | number>)
        : window.label;

      return h(
        'div',
        { key: window.id, className: styleMap.quotaRow },
        h(
          'div',
          { className: styleMap.quotaRowHeader },
          h('span', { className: styleMap.quotaModel }, windowLabel),
          h(
            'div',
            { className: styleMap.quotaMeta },
            h('span', { className: styleMap.quotaPercent }, percentLabel),
            h('span', { className: styleMap.quotaReset }, window.resetLabel)
          )
        ),
        h(QuotaProgressBar, { percent: remaining })
      );
    })
  );

  return h(Fragment, null, ...nodes);
};

const renderGeminiCliItems = (
  quota: GeminiCliQuotaState,
  t: TFunction,
  helpers: QuotaRenderHelpers
): ReactNode => {
  const { styles: styleMap, QuotaProgressBar } = helpers;
  const { createElement: h, Fragment } = React;
  const buckets = quota.buckets ?? [];
  const tierLabel = quota.tierLabel ?? null;
  const tierId = quota.tierId ?? null;
  const creditBalance = quota.creditBalance ?? null;
  const isPremiumTier = tierId !== null && PREMIUM_GEMINI_CLI_TIER_IDS.has(tierId);
  const nodes: ReactNode[] = [];

  if (tierLabel) {
    const valueClass = isPremiumTier ? styleMap.premiumPlanValue : styleMap.codexPlanValue;
    nodes.push(
      h(
        'div',
        { key: 'tier', className: styleMap.codexPlan },
        h('span', { className: styleMap.codexPlanLabel }, t('gemini_cli_quota.tier_label')),
        h('span', { className: valueClass }, tierLabel)
      )
    );
  }

  if (creditBalance !== null) {
    nodes.push(
      h(
        'div',
        { key: 'credits', className: styleMap.codexPlan },
        h('span', { className: styleMap.codexPlanLabel }, t('gemini_cli_quota.credit_label')),
        h(
          'span',
          { className: styleMap.codexPlanValue },
          t('gemini_cli_quota.credit_amount', { count: creditBalance })
        )
      )
    );
  }

  if (buckets.length === 0) {
    nodes.push(
      h(
        'div',
        { key: 'empty', className: styleMap.quotaMessage },
        t('gemini_cli_quota.empty_buckets')
      )
    );
    return h(Fragment, null, ...nodes);
  }

  nodes.push(
    ...buckets.map((bucket) => {
      const fraction = bucket.remainingFraction;
      const clamped = fraction === null ? null : Math.max(0, Math.min(1, fraction));
      const percent = clamped === null ? null : Math.round(clamped * 100);
      const percentLabel = percent === null ? '--' : `${percent}%`;
      const remainingAmountLabel =
        bucket.remainingAmount === null || bucket.remainingAmount === undefined
          ? null
          : t('gemini_cli_quota.remaining_amount', {
              count: bucket.remainingAmount,
            });
      const titleBase =
        bucket.modelIds && bucket.modelIds.length > 0 ? bucket.modelIds.join(', ') : bucket.label;
      const title = bucket.tokenType ? `${titleBase} (${bucket.tokenType})` : titleBase;

      const resetLabel = formatQuotaResetTime(bucket.resetTime);

      return h(
        'div',
        { key: bucket.id, className: styleMap.quotaRow },
        h(
          'div',
          { className: styleMap.quotaRowHeader },
          h('span', { className: styleMap.quotaModel, title }, bucket.label),
          h(
            'div',
            { className: styleMap.quotaMeta },
            h('span', { className: styleMap.quotaPercent }, percentLabel),
            remainingAmountLabel
              ? h('span', { className: styleMap.quotaAmount }, remainingAmountLabel)
              : null,
            h('span', { className: styleMap.quotaReset }, resetLabel)
          )
        ),
        h(QuotaProgressBar, { percent })
      );
    })
  );

  return h(Fragment, null, ...nodes);
};

const buildClaudeQuotaWindows = (
  payload: ClaudeUsagePayload,
  t: TFunction
): ClaudeQuotaWindow[] => {
  const windows: ClaudeQuotaWindow[] = [];

  for (const { key, id, labelKey } of CLAUDE_USAGE_WINDOW_KEYS) {
    const window = payload[key as keyof ClaudeUsagePayload];
    if (!window || typeof window !== 'object' || !('utilization' in window)) continue;
    const typedWindow = window as { utilization: number; resets_at: string };
    const usedPercent = normalizeNumberValue(typedWindow.utilization);
    const resetLabel = formatQuotaResetTime(typedWindow.resets_at);
    windows.push({
      id,
      label: t(labelKey),
      labelKey,
      usedPercent,
      resetLabel,
    });
  }

  return windows;
};

const normalizeFlagValue = (value: unknown): boolean | undefined => {
  if (value === undefined || value === null) return undefined;
  if (typeof value === 'boolean') return value;
  if (typeof value === 'number') return value !== 0;
  if (typeof value === 'string') {
    const trimmed = value.trim().toLowerCase();
    if (['true', '1', 'yes', 'y', 'on'].includes(trimmed)) return true;
    if (['false', '0', 'no', 'n', 'off'].includes(trimmed)) return false;
  }
  return undefined;
};

const parseClaudeProfilePayload = (payload: unknown): ClaudeProfileResponse | null => {
  if (payload === undefined || payload === null) return null;
  if (typeof payload === 'string') {
    const trimmed = payload.trim();
    if (!trimmed) return null;
    try {
      return JSON.parse(trimmed) as ClaudeProfileResponse;
    } catch {
      return null;
    }
  }
  if (typeof payload === 'object') {
    return payload as ClaudeProfileResponse;
  }
  return null;
};

const resolveClaudePlanType = (profile: ClaudeProfileResponse | null): string | null => {
  if (!profile) return null;

  const hasClaudeMax = normalizeFlagValue(profile.account?.has_claude_max);
  if (hasClaudeMax) return 'plan_max';

  const hasClaudePro = normalizeFlagValue(profile.account?.has_claude_pro);
  if (hasClaudePro) return 'plan_pro';

  const organizationType = normalizeStringValue(
    profile.organization?.organization_type
  )?.toLowerCase();
  const subscriptionStatus = normalizeStringValue(
    profile.organization?.subscription_status
  )?.toLowerCase();

  if (organizationType === 'claude_team' && subscriptionStatus === 'active') {
    return 'plan_team';
  }

  if (hasClaudeMax === false && hasClaudePro === false) return 'plan_free';

  return null;
};

const fetchClaudeQuota = async (
  file: AuthFileItem,
  t: TFunction
): Promise<{
  windows: ClaudeQuotaWindow[];
  extraUsage?: ClaudeExtraUsage | null;
  planType?: string | null;
}> => {
  const rawAuthIndex = file['auth_index'] ?? file.authIndex;
  const authIndex = normalizeAuthIndex(rawAuthIndex);
  if (!authIndex) {
    throw new Error(t('claude_quota.missing_auth_index'));
  }

  const [usageResult, profileResult] = await Promise.allSettled([
    apiCallApi.request({
      authIndex,
      method: 'GET',
      url: CLAUDE_USAGE_URL,
      header: { ...CLAUDE_REQUEST_HEADERS },
    }),
    apiCallApi.request({
      authIndex,
      method: 'GET',
      url: CLAUDE_PROFILE_URL,
      header: { ...CLAUDE_REQUEST_HEADERS },
    }),
  ]);

  if (usageResult.status === 'rejected') {
    throw usageResult.reason;
  }

  const result = usageResult.value;

  if (result.statusCode < 200 || result.statusCode >= 300) {
    throw createStatusError(getApiCallErrorMessage(result), result.statusCode);
  }

  const payload = parseClaudeUsagePayload(result.body ?? result.bodyText);
  if (!payload) {
    throw new Error(t('claude_quota.empty_windows'));
  }

  const windows = buildClaudeQuotaWindows(payload, t);
  const planType =
    profileResult.status === 'fulfilled' &&
    profileResult.value.statusCode >= 200 &&
    profileResult.value.statusCode < 300
      ? resolveClaudePlanType(
          parseClaudeProfilePayload(profileResult.value.body ?? profileResult.value.bodyText)
        )
      : null;

  return { windows, extraUsage: payload.extra_usage, planType };
};

const renderClaudeItems = (
  quota: ClaudeQuotaState,
  t: TFunction,
  helpers: QuotaRenderHelpers
): ReactNode => {
  const { styles: styleMap, QuotaProgressBar } = helpers;
  const { createElement: h, Fragment } = React;
  const windows = quota.windows ?? [];
  const extraUsage = quota.extraUsage ?? null;
  const planType = quota.planType ?? null;
  const nodes: ReactNode[] = [];

  if (planType) {
    nodes.push(
      h(
        'div',
        { key: 'plan', className: styleMap.codexPlan },
        h('span', { className: styleMap.codexPlanLabel }, t('claude_quota.plan_label')),
        h('span', { className: styleMap.codexPlanValue }, t(`claude_quota.${planType}`))
      )
    );
  }

  if (extraUsage && extraUsage.is_enabled) {
    const usedLabel = `$${(extraUsage.used_credits / 100).toFixed(2)} / $${(extraUsage.monthly_limit / 100).toFixed(2)}`;
    nodes.push(
      h(
        'div',
        { key: 'extra', className: styleMap.codexPlan },
        h('span', { className: styleMap.codexPlanLabel }, t('claude_quota.extra_usage_label')),
        h('span', { className: styleMap.codexPlanValue }, usedLabel)
      )
    );
  }

  if (windows.length === 0) {
    nodes.push(
      h('div', { key: 'empty', className: styleMap.quotaMessage }, t('claude_quota.empty_windows'))
    );
    return h(Fragment, null, ...nodes);
  }

  nodes.push(
    ...windows.map((window) => {
      const used = window.usedPercent;
      const clampedUsed = used === null ? null : Math.max(0, Math.min(100, used));
      const remaining = clampedUsed === null ? null : Math.max(0, Math.min(100, 100 - clampedUsed));
      const percentLabel = remaining === null ? '--' : `${Math.round(remaining)}%`;
      const windowLabel = window.labelKey ? t(window.labelKey) : window.label;

      return h(
        'div',
        { key: window.id, className: styleMap.quotaRow },
        h(
          'div',
          { className: styleMap.quotaRowHeader },
          h('span', { className: styleMap.quotaModel }, windowLabel),
          h(
            'div',
            { className: styleMap.quotaMeta },
            h('span', { className: styleMap.quotaPercent }, percentLabel),
            h('span', { className: styleMap.quotaReset }, window.resetLabel)
          )
        ),
        h(QuotaProgressBar, { percent: remaining })
      );
    })
  );

  return h(Fragment, null, ...nodes);
};

export const CLAUDE_CONFIG: QuotaConfig<
  ClaudeQuotaState,
  { windows: ClaudeQuotaWindow[]; extraUsage?: ClaudeExtraUsage | null; planType?: string | null }
> = {
  type: 'claude',
  i18nPrefix: 'claude_quota',
  cardIdleMessageKey: 'quota_management.card_idle_hint',
  filterFn: (file) => isClaudeFile(file) && !isDisabledAuthFile(file),
  fetchQuota: fetchClaudeQuota,
  storeSelector: (state) => state.claudeQuota,
  storeSetter: 'setClaudeQuota',
  buildLoadingState: () => ({ status: 'loading', windows: [] }),
  buildSuccessState: (data) => ({
    status: 'success',
    windows: data.windows,
    extraUsage: data.extraUsage,
    planType: data.planType,
  }),
  buildErrorState: (message, status) => ({
    status: 'error',
    windows: [],
    error: message,
    errorStatus: status,
  }),
  cardClassName: styles.claudeCard,
  controlsClassName: styles.claudeControls,
  controlClassName: styles.claudeControl,
  gridClassName: styles.claudeGrid,
  renderQuotaItems: renderClaudeItems,
};

export const ANTIGRAVITY_CONFIG: QuotaConfig<AntigravityQuotaState, AntigravityQuotaGroup[]> = {
  type: 'antigravity',
  i18nPrefix: 'antigravity_quota',
  cardIdleMessageKey: 'quota_management.card_idle_hint',
  filterFn: (file) => isAntigravityFile(file) && !isDisabledAuthFile(file),
  fetchQuota: fetchAntigravityQuota,
  storeSelector: (state) => state.antigravityQuota,
  storeSetter: 'setAntigravityQuota',
  buildLoadingState: () => ({ status: 'loading', groups: [] }),
  buildSuccessState: (groups) => ({ status: 'success', groups }),
  buildErrorState: (message, status) => ({
    status: 'error',
    groups: [],
    error: message,
    errorStatus: status,
  }),
  cardClassName: styles.antigravityCard,
  controlsClassName: styles.antigravityControls,
  controlClassName: styles.antigravityControl,
  gridClassName: styles.antigravityGrid,
  renderQuotaItems: renderAntigravityItems,
};

export const CODEX_CONFIG: QuotaConfig<
  CodexQuotaState,
  { planType: string | null; windows: CodexQuotaWindow[] }
> = {
  type: 'codex',
  i18nPrefix: 'codex_quota',
  cardIdleMessageKey: 'quota_management.card_idle_hint',
  filterFn: (file) => isCodexFile(file) && !isDisabledAuthFile(file),
  fetchQuota: fetchCodexQuota,
  storeSelector: (state) => state.codexQuota,
  storeSetter: 'setCodexQuota',
  buildLoadingState: () => ({ status: 'loading', windows: [] }),
  buildSuccessState: (data) => ({
    status: 'success',
    windows: data.windows,
    planType: data.planType,
  }),
  buildErrorState: (message, status) => ({
    status: 'error',
    windows: [],
    error: message,
    errorStatus: status,
  }),
  cardClassName: styles.codexCard,
  controlsClassName: styles.codexControls,
  controlClassName: styles.codexControl,
  gridClassName: styles.codexGrid,
  renderQuotaItems: renderCodexItems,
};

export const GEMINI_CLI_CONFIG: QuotaConfig<
  GeminiCliQuotaState,
  {
    fileName: string;
    supplementaryRequestId: number;
    buckets: GeminiCliQuotaBucketState[];
    tierLabel: string | null;
    tierId: string | null;
    creditBalance: number | null;
  }
> = {
  type: 'gemini-cli',
  i18nPrefix: 'gemini_cli_quota',
  cardIdleMessageKey: 'quota_management.card_idle_hint',
  filterFn: (file) =>
    isGeminiCliFile(file) && !isRuntimeOnlyAuthFile(file) && !isDisabledAuthFile(file),
  fetchQuota: fetchGeminiCliQuota,
  storeSelector: (state) => state.geminiCliQuota,
  storeSetter: 'setGeminiCliQuota',
  buildLoadingState: () => ({
    status: 'loading',
    buckets: [],
    tierLabel: null,
    tierId: null,
    creditBalance: null,
  }),
  buildSuccessState: (data) => {
    const supplementarySnapshot = readGeminiCliSupplementarySnapshot(
      data.fileName,
      data.supplementaryRequestId
    );

    return {
      status: 'success',
      buckets: data.buckets,
      tierLabel: supplementarySnapshot.tierLabel ?? data.tierLabel,
      tierId: supplementarySnapshot.tierId ?? data.tierId,
      creditBalance: supplementarySnapshot.creditBalance ?? data.creditBalance,
    };
  },
  buildErrorState: (message, status) => ({
    status: 'error',
    buckets: [],
    error: message,
    errorStatus: status,
  }),
  cardClassName: styles.geminiCliCard,
  controlsClassName: styles.geminiCliControls,
  controlClassName: styles.geminiCliControl,
  gridClassName: styles.geminiCliGrid,
  renderQuotaItems: renderGeminiCliItems,
};

const fetchKimiQuota = async (file: AuthFileItem, t: TFunction): Promise<KimiQuotaRow[]> => {
  const rawAuthIndex = file['auth_index'] ?? file.authIndex;
  const authIndex = normalizeAuthIndex(rawAuthIndex);
  if (!authIndex) {
    throw new Error(t('kimi_quota.missing_auth_index'));
  }

  const result = await apiCallApi.request({
    authIndex,
    method: 'GET',
    url: KIMI_USAGE_URL,
    header: { ...KIMI_REQUEST_HEADERS },
  });

  if (result.statusCode < 200 || result.statusCode >= 300) {
    throw createStatusError(getApiCallErrorMessage(result), result.statusCode);
  }

  const payload = parseKimiUsagePayload(result.body ?? result.bodyText);
  if (!payload) {
    throw new Error(t('kimi_quota.empty_data'));
  }

  return buildKimiQuotaRows(payload);
};

const renderKimiItems = (
  quota: KimiQuotaState,
  t: TFunction,
  helpers: QuotaRenderHelpers
): ReactNode => {
  const { styles: styleMap, QuotaProgressBar } = helpers;
  const { createElement: h } = React;
  const rows = quota.rows ?? [];

  if (rows.length === 0) {
    return h('div', { className: styleMap.quotaMessage }, t('kimi_quota.empty_data'));
  }

  return rows.map((row) => {
    const limit = row.limit;
    const used = row.used;
    const remaining =
      limit > 0
        ? Math.max(0, Math.min(100, Math.round(((limit - used) / limit) * 100)))
        : used > 0
          ? 0
          : null;
    const percentLabel = remaining === null ? '--' : `${remaining}%`;
    const rowLabel = row.labelKey
      ? t(row.labelKey, (row.labelParams ?? {}) as Record<string, string | number>)
      : (row.label ?? '');
    const resetLabel = formatKimiResetHint(t, row.resetHint);

    return h(
      'div',
      { key: row.id, className: styleMap.quotaRow },
      h(
        'div',
        { className: styleMap.quotaRowHeader },
        h('span', { className: styleMap.quotaModel }, rowLabel),
        h(
          'div',
          { className: styleMap.quotaMeta },
          h('span', { className: styleMap.quotaPercent }, percentLabel),
          limit > 0 ? h('span', { className: styleMap.quotaAmount }, `${used} / ${limit}`) : null,
          resetLabel ? h('span', { className: styleMap.quotaReset }, resetLabel) : null
        )
      ),
      h(QuotaProgressBar, { percent: remaining })
    );
  });
};

const readRuntimeString = (value: unknown): string | undefined => {
  if (typeof value !== 'string') return undefined;
  const trimmed = value.trim();
  return trimmed || undefined;
};

const readRuntimeTimestampString = (value: unknown): string | undefined => {
  const timestamp = readRuntimeString(value);
  if (!timestamp) return undefined;
  if (timestamp.startsWith('0001-01-01')) return undefined;
  return timestamp;
};

const readRuntimeBoolean = (value: unknown): boolean => {
  if (typeof value === 'boolean') return value;
  if (typeof value === 'number') return value !== 0;
  if (typeof value === 'string') {
    const trimmed = value.trim().toLowerCase();
    return trimmed === 'true' || trimmed === '1' || trimmed === 'yes' || trimmed === 'on';
  }
  return false;
};

const normalizeKiroProviderQuotaRowEntries = (
  raw: unknown
): Array<{ key?: string; value: unknown }> => {
  if (Array.isArray(raw)) {
    return raw.map((value) => ({ value }));
  }
  if (!raw || typeof raw !== 'object') {
    return [];
  }

  return Object.keys(raw as Record<string, unknown>)
    .sort((left, right) => left.localeCompare(right))
    .map((key) => ({
      key,
      value: (raw as Record<string, unknown>)[key],
    }));
};

const normalizeKiroProviderQuotaRows = (raw: unknown): KiroProviderQuotaRow[] => {
  return normalizeKiroProviderQuotaRowEntries(raw)
    .map(({ key, value }, index): KiroProviderQuotaRow | null => {
      const source = value && typeof value === 'object' ? (value as Record<string, unknown>) : null;
      if (!source) return null;
      const resourceType = readRuntimeString(source.resource_type ?? source.resourceType);
      const id = readRuntimeString(source.id) ?? key ?? resourceType ?? `quota-${index}`;
      const name =
        readRuntimeString(source.name ?? source.display_name ?? source.displayName) ??
        resourceType ??
        key ??
        id;
      const current = normalizeNumberValue(source.current);
      const limit = normalizeNumberValue(source.limit);
      const used = normalizeNumberValue(source.used) ?? current;
      const total = normalizeNumberValue(source.total) ?? limit;
      const remaining = normalizeNumberValue(source.remaining);
      const percent = normalizeNumberValue(source.percent);
      const remainingPercent = normalizeNumberValue(
        source.remaining_percent ?? source.remainingPercent
      );

      return {
        id,
        resourceType: resourceType ?? undefined,
        name,
        displayNamePlural: readRuntimeString(
          source.display_name_plural ?? source.displayNamePlural
        ),
        currency: readRuntimeString(source.currency),
        unit: readRuntimeString(source.unit),
        current,
        limit,
        used,
        total,
        remaining,
        percent,
        remainingPercent,
        resetAt: readRuntimeTimestampString(source.reset_at ?? source.resetAt),
        unlimited: readRuntimeBoolean(source.unlimited),
        freeTrial: readRuntimeBoolean(source.free_trial ?? source.freeTrial),
        trialStatus: readRuntimeString(source.trial_status ?? source.trialStatus),
        subscriptionTitle: readRuntimeString(
          source.subscription_title ?? source.subscriptionTitle
        ),
        subscriptionType: readRuntimeString(
          source.subscription_type ?? source.subscriptionType
        ),
        overageStatus: readRuntimeString(source.overage_status ?? source.overageStatus),
        overageCap: normalizeNumberValue(source.overage_cap ?? source.overageCap),
        overageRate: normalizeNumberValue(source.overage_rate ?? source.overageRate),
        currentOverages: normalizeNumberValue(
          source.current_overages ?? source.currentOverages
        ),
        overageCharges: normalizeNumberValue(
          source.overage_charges ?? source.overageCharges
        ),
        overageChargesWithPrecision: normalizeNumberValue(
          source.overage_charges_with_precision ?? source.overageChargesWithPrecision
        ),
        bonuses: source.bonuses,
      };
    })
    .filter((row): row is KiroProviderQuotaRow => row !== null);
};

const normalizeKiroProviderQuota = (raw: unknown): KiroProviderQuotaState | null => {
  const source = raw && typeof raw === 'object' ? (raw as Record<string, unknown>) : null;
  if (!source) return null;

  const current = normalizeNumberValue(source.current);
  const limit = normalizeNumberValue(source.limit);
  const quotas = normalizeKiroProviderQuotaRows(source.quotas);
  const providerQuotaAvailable =
    readRuntimeBoolean(
      source.provider_quota_available ?? source.providerQuotaAvailable ?? source.available
    ) || current !== null || limit !== null || quotas.length > 0;

  return {
    providerQuotaAvailable,
    message: readRuntimeString(source.message),
    plan: readRuntimeString(source.plan),
    quotas,
    current,
    limit,
    percent: normalizeNumberValue(source.percent),
    remaining: normalizeNumberValue(source.remaining),
    nextResetAt: readRuntimeTimestampString(source.next_reset_at ?? source.nextResetAt),
    subscriptionType: readRuntimeString(
      source.subscription_type ?? source.subscriptionType
    ),
    subscriptionTitle: readRuntimeString(
      source.subscription_title ?? source.subscriptionTitle
    ),
    trialCurrent: normalizeNumberValue(source.trial_current ?? source.trialCurrent),
    trialLimit: normalizeNumberValue(source.trial_limit ?? source.trialLimit),
    trialPercent: normalizeNumberValue(source.trial_percent ?? source.trialPercent),
    trialStatus: readRuntimeString(source.trial_status ?? source.trialStatus),
    trialExpiresAt: readRuntimeTimestampString(
      source.trial_expires_at ?? source.trialExpiresAt
    ),
    overageStatus: readRuntimeString(source.overage_status ?? source.overageStatus),
    overageCap: normalizeNumberValue(source.overage_cap ?? source.overageCap),
    overageRate: normalizeNumberValue(source.overage_rate ?? source.overageRate),
    currentOverages: normalizeNumberValue(
      source.current_overages ?? source.currentOverages
    ),
    overageCharges: normalizeNumberValue(source.overage_charges ?? source.overageCharges),
    overageChargesWithPrecision: normalizeNumberValue(
      source.overage_charges_with_precision ?? source.overageChargesWithPrecision
    ),
    overageCapability: source.overage_capability ?? source.overageCapability,
    subscriptionManagementTarget: readRuntimeString(
      source.subscription_management_target ?? source.subscriptionManagementTarget
    ),
    upgradeCapability: source.upgrade_capability ?? source.upgradeCapability,
    raw:
      source.raw && typeof source.raw === 'object'
        ? (source.raw as Record<string, unknown>)
        : undefined,
    checkedAt: readRuntimeTimestampString(source.checked_at ?? source.checkedAt),
  };
};

const normalizeKiroRuntimeUsageStats = (raw: unknown): KiroRuntimeUsageStats | null => {
  const source = raw && typeof raw === 'object' ? (raw as Record<string, unknown>) : null;
  if (!source) return null;
  const requests = normalizeNumberValue(source.requests) ?? 0;
  const promptTokens = normalizeNumberValue(source.prompt_tokens ?? source.promptTokens) ?? 0;
  const completionTokens =
    normalizeNumberValue(source.completion_tokens ?? source.completionTokens) ?? 0;
  const totalTokens = normalizeNumberValue(source.total_tokens ?? source.totalTokens) ?? 0;
  const estimatedTokens =
    normalizeNumberValue(source.estimated_tokens ?? source.estimatedTokens) ?? 0;
  const lastModel = readRuntimeString(source.last_model ?? source.lastModel);
  const lastUsedAt = readRuntimeTimestampString(source.last_used_at ?? source.lastUsedAt);
  const updatedAt = readRuntimeTimestampString(source.updated_at ?? source.updatedAt);
  if (
    requests === 0 &&
    promptTokens === 0 &&
    completionTokens === 0 &&
    totalTokens === 0 &&
    estimatedTokens === 0 &&
    !lastModel &&
    !lastUsedAt &&
    !updatedAt
  ) {
    return null;
  }
  return {
    requests,
    promptTokens,
    completionTokens,
    totalTokens,
    estimatedTokens,
    lastModel: lastModel ?? undefined,
    lastUsedAt,
    updatedAt,
  };
};

const normalizeKiroRuntimeStatus = (
  status: unknown,
  options?: { disabled?: boolean; unavailable?: boolean; quotaExceeded?: boolean; error?: unknown }
): KiroRuntimeStatus => {
  if (options?.disabled) return 'disabled';
  const rawStatus = readRuntimeString(status)?.toLowerCase();
  if (rawStatus === 'disabled') return 'disabled';
  if (rawStatus === 'error' || options?.error) return 'error';
  if (rawStatus === 'unavailable' || options?.unavailable || options?.quotaExceeded) {
    return 'unavailable';
  }
  return 'active';
};

const normalizeKiroRuntimeQuota = (raw: unknown): KiroRuntimeQuotaState => {
  const source = raw && typeof raw === 'object' ? (raw as Record<string, unknown>) : {};
  return {
    exceeded: readRuntimeBoolean(source.exceeded),
    reason: readRuntimeString(source.reason),
    nextRecoverAt: readRuntimeTimestampString(source.next_recover_at ?? source.nextRecoverAt),
  };
};

const normalizeKiroModelStates = (file: AuthFileItem): KiroRuntimeModelQuotaState[] => {
  const rawStates = file.model_states ?? file.modelStates;
  if (!rawStates || typeof rawStates !== 'object') return [];

  return Object.entries(rawStates)
    .map((entry): KiroRuntimeModelQuotaState | null => {
      const [id, rawState] = entry;
      if (!rawState || typeof rawState !== 'object') return null;
      const state = rawState as Record<string, unknown>;
      const quota = normalizeKiroRuntimeQuota(state.quota);
      const unavailable = readRuntimeBoolean(state.unavailable);
      return {
        id,
        status: normalizeKiroRuntimeStatus(state.status, {
          unavailable,
          quotaExceeded: quota.exceeded,
          error: state.last_error ?? state.lastError,
        }),
        statusMessage: readRuntimeString(state.status_message ?? state.statusMessage),
        unavailable,
        nextRetryAfter: readRuntimeTimestampString(state.next_retry_after ?? state.nextRetryAfter),
        quota,
      };
    })
    .filter((state): state is KiroRuntimeModelQuotaState => state !== null)
    .sort((left, right) => left.id.localeCompare(right.id));
};

const buildKiroRuntimeState = (file: AuthFileItem): KiroQuotaState => {
  const quota = normalizeKiroRuntimeQuota(file.quota);
  const disabled = isDisabledAuthFile(file);
  const unavailable = readRuntimeBoolean(file.unavailable);
  const providerQuota = normalizeKiroProviderQuota(file.kiro_quota ?? file.kiroQuota);
  const runtimeUsageStats = normalizeKiroRuntimeUsageStats(
    file.kiro_usage_stats ?? file.kiroUsageStats
  );
  return {
    status: providerQuota?.providerQuotaAvailable ? 'success' : 'runtime-only',
    runtimeStatus: normalizeKiroRuntimeStatus(file.status, {
      disabled,
      unavailable,
      quotaExceeded: quota.exceeded,
    }),
    statusMessage: readRuntimeString(file.status_message ?? file.statusMessage),
    unavailable,
    disabled,
    quota,
    modelStates: normalizeKiroModelStates(file),
    providerQuotaAvailable: Boolean(providerQuota?.providerQuotaAvailable),
    providerQuota,
    runtimeUsageStats,
  };
};

export const buildKiroQuotaStateFromProvider = (
  file: AuthFileItem,
  rawQuota: KiroProviderQuotaState | unknown
): KiroQuotaState => {
  const providerQuota = normalizeKiroProviderQuota(rawQuota) ?? (rawQuota as KiroProviderQuotaState);
  const providerQuotaAvailable = Boolean(providerQuota?.providerQuotaAvailable);
  return {
    ...buildKiroRuntimeState(file),
    status: providerQuotaAvailable ? 'success' : 'runtime-only',
    providerQuotaAvailable,
    providerQuota,
    overageUpdating: false,
  };
};

const fetchKiroQuota = async (file: AuthFileItem): Promise<KiroQuotaState> => {
  const rawAuthIndex = file['auth_index'] ?? file.authIndex;
  const authIndex = normalizeAuthIndex(rawAuthIndex);
  const quota = await authFilesApi.refreshKiroQuota({
    name: file.name,
    authIndex: authIndex ?? undefined,
  });
  return buildKiroQuotaStateFromProvider(file, quota);
};

const renderKiroRuntimeStatusBadge = (
  status: KiroRuntimeStatus,
  t: TFunction,
  helpers: QuotaRenderHelpers
): ReactNode => {
  const { createElement: h } = React;
  const label = t(`kiro_quota.status_${status}`);
  const className =
    status === 'active'
      ? `${helpers.styles.codexPlanValue} text-emerald-700`
      : status === 'disabled'
        ? `${helpers.styles.codexPlanValue} text-muted-foreground`
        : status === 'error'
          ? `${helpers.styles.codexPlanValue} text-destructive`
          : `${helpers.styles.codexPlanValue} text-amber-700`;
  return h('span', { className }, label);
};

const formatKiroQuotaNumber = (value?: number | null): string => {
  if (value === undefined || value === null || !Number.isFinite(value)) return '-';
  if (Number.isInteger(value)) return String(value);
  return value.toFixed(2).replace(/\.?0+$/, '');
};

const formatKiroQuotaMetadataValue = (value: unknown): string | null => {
  if (value === undefined || value === null) return null;
  if (typeof value === 'string') {
    const trimmed = value.trim();
    return trimmed || null;
  }
  if (typeof value === 'number' || typeof value === 'boolean') return String(value);
  try {
    const serialized = JSON.stringify(value);
    return serialized === '{}' || serialized === '[]' ? null : serialized;
  } catch {
    return null;
  }
};

const formatKiroQuotaUnitSuffix = (
  unit?: string,
  currency?: string
): string => {
  const parts = [unit, currency]
    .map((part) => part?.trim())
    .filter((part): part is string => Boolean(part));
  const deduped = parts.filter((part, index) => parts.indexOf(part) === index);
  return deduped.length > 0 ? ` ${deduped.join(' ')}` : '';
};

const formatKiroQuotaAmount = (
  value?: number | null,
  meta?: Pick<KiroProviderQuotaRow, 'unit' | 'currency'>
): string => `${formatKiroQuotaNumber(value)}${formatKiroQuotaUnitSuffix(meta?.unit, meta?.currency)}`;

const humanizeKiroQuotaName = (name: string, t: TFunction, freeTrial?: boolean): string => {
  const normalized = name.trim().toLowerCase();
  const base =
    normalized === 'agentic_request'
      ? t('kiro_quota.provider_quota')
      : normalized === 'free_trial'
        ? t('kiro_quota.trial_quota')
        : name
            .split('_')
            .filter(Boolean)
            .map((part) => part.charAt(0).toUpperCase() + part.slice(1).toLowerCase())
            .join(' ');
  return freeTrial && normalized !== 'free_trial' ? `${base} (${t('kiro_quota.trial')})` : base;
};

const renderKiroProviderQuotaRow = (
  row: KiroProviderQuotaRow,
  t: TFunction,
  helpers: QuotaRenderHelpers
): ReactNode => {
  const { styles: styleMap, QuotaProgressBar } = helpers;
  const { createElement: h } = React;
  const used = row.used ?? row.current ?? null;
  const total = row.total ?? row.limit ?? null;
  const usedPercent =
    row.percent ?? (used != null && total && total > 0 ? (used / total) * 100 : null);
  const remainingPercent =
    row.remainingPercent ?? (usedPercent === null ? null : Math.max(0, 100 - usedPercent));
  const amountLabel =
    used != null && total != null
      ? `${formatKiroQuotaAmount(used, row)} / ${formatKiroQuotaAmount(total, row)}`
      : row.remaining != null
        ? t('kiro_quota.remaining_amount', {
            amount: formatKiroQuotaAmount(row.remaining, row),
          })
        : undefined;
  const label = humanizeKiroQuotaName(row.name || row.resourceType || row.id, t, row.freeTrial);
  const title = [row.resourceType ?? row.name, row.displayNamePlural, row.unit, row.currency]
    .filter(Boolean)
    .join(' / ');

  return h(
    'div',
    { key: `provider-quota-${row.id}`, className: styleMap.quotaRow },
    h(
      'div',
      { className: styleMap.quotaRowHeader },
      h('span', { className: styleMap.quotaModel, title }, label),
      h(
        'div',
        { className: styleMap.quotaMeta },
        row.trialStatus ? h('span', { className: styleMap.quotaAmount }, row.trialStatus) : null,
        h(
          'span',
          { className: styleMap.quotaPercent },
          usedPercent === null ? '-' : `${Math.round(usedPercent)}%`
        ),
        amountLabel ? h('span', { className: styleMap.quotaAmount }, amountLabel) : null,
        h('span', { className: styleMap.quotaReset }, formatQuotaResetTime(row.resetAt))
      )
    ),
    h(QuotaProgressBar, { percent: remainingPercent })
  );
};

const renderKiroItems = (
  quota: KiroQuotaState,
  t: TFunction,
  helpers: QuotaRenderHelpers
): ReactNode => {
  const { styles: styleMap, QuotaProgressBar } = helpers;
  const { createElement: h, Fragment } = React;
  const nodes: ReactNode[] = [];
  const providerQuota = quota.providerQuota;

  if (providerQuota?.providerQuotaAvailable) {
    const planParts = [
      providerQuota.subscriptionTitle || providerQuota.plan,
      providerQuota.subscriptionType,
      formatKiroQuotaMetadataValue(providerQuota.overageCapability),
      providerQuota.subscriptionManagementTarget,
      formatKiroQuotaMetadataValue(providerQuota.upgradeCapability),
    ].filter(Boolean);
    const planLabel = planParts.length > 0 ? planParts.join(' / ') : null;

    if (planLabel) {
      nodes.push(
        h(
          'div',
          { key: 'subscription', className: styleMap.codexPlan },
          h('span', { className: styleMap.codexPlanLabel }, t('kiro_quota.subscription')),
          h('span', { className: styleMap.codexPlanValue }, planLabel)
        )
      );
    }

    if (providerQuota.quotas && providerQuota.quotas.length > 0) {
      nodes.push(
        ...providerQuota.quotas.map((row) => renderKiroProviderQuotaRow(row, t, helpers))
      );
    } else {
      nodes.push(
        renderKiroProviderQuotaRow(
          {
            id: 'provider',
            name: 'agentic_request',
            current: providerQuota.current,
            limit: providerQuota.limit,
            used: providerQuota.current,
            total: providerQuota.limit,
            remaining: providerQuota.remaining,
            percent: providerQuota.percent,
            resetAt: providerQuota.nextResetAt,
          },
          t,
          helpers
        )
      );
    }

    if (
      (!providerQuota.quotas || providerQuota.quotas.length === 0) &&
      (providerQuota.trialCurrent != null || providerQuota.trialLimit != null)
    ) {
      const trialUsedPercent =
        providerQuota.trialPercent ??
        (providerQuota.trialCurrent != null && providerQuota.trialLimit
          ? (providerQuota.trialCurrent / providerQuota.trialLimit) * 100
          : null);
      nodes.push(
        h(
          'div',
          { key: 'trial-quota', className: styleMap.quotaRow },
          h(
            'div',
            { className: styleMap.quotaRowHeader },
            h('span', { className: styleMap.quotaModel }, t('kiro_quota.trial_quota')),
            h(
              'div',
              { className: styleMap.quotaMeta },
              providerQuota.trialStatus
                ? h('span', { className: styleMap.quotaAmount }, providerQuota.trialStatus)
                : null,
              h(
                'span',
                { className: styleMap.quotaPercent },
                trialUsedPercent === null ? '-' : `${Math.round(trialUsedPercent)}%`
              ),
              h(
                'span',
                { className: styleMap.quotaAmount },
                `${formatKiroQuotaNumber(providerQuota.trialCurrent)} / ${formatKiroQuotaNumber(providerQuota.trialLimit)}`
              ),
              h(
                'span',
                { className: styleMap.quotaReset },
                formatQuotaResetTime(providerQuota.trialExpiresAt)
              )
            )
          ),
          h(QuotaProgressBar, {
            percent: trialUsedPercent === null ? null : Math.max(0, 100 - trialUsedPercent),
          })
        )
      );
    }

    if (
      providerQuota.overageStatus ||
      providerQuota.currentOverages != null ||
      providerQuota.overageCap != null ||
      providerQuota.overageRate != null ||
      providerQuota.overageCharges != null ||
      providerQuota.overageChargesWithPrecision != null ||
      providerQuota.quotas?.some(
        (row) => row.overageCharges != null || row.overageChargesWithPrecision != null
      )
    ) {
      const primaryQuota =
        providerQuota.quotas?.find((row) => !row.freeTrial) ?? providerQuota.quotas?.[0];
      const overageCharges =
        providerQuota.overageChargesWithPrecision ??
        providerQuota.overageCharges ??
        primaryQuota?.overageChargesWithPrecision ??
        primaryQuota?.overageCharges ??
        null;
      const overageParts = [
        providerQuota.currentOverages != null
          ? t('kiro_quota.current_overages', {
              amount: formatKiroQuotaAmount(providerQuota.currentOverages, primaryQuota),
            })
          : null,
        providerQuota.overageCap != null
          ? t('kiro_quota.overage_cap', {
              amount: formatKiroQuotaAmount(providerQuota.overageCap, primaryQuota),
            })
          : null,
        providerQuota.overageRate != null
          ? t('kiro_quota.overage_rate', {
              amount: formatKiroQuotaNumber(providerQuota.overageRate),
            })
          : null,
        overageCharges != null
          ? t('kiro_quota.overage_charges', {
              amount: formatKiroQuotaAmount(overageCharges, {
                currency: primaryQuota?.currency,
              }),
            })
          : null,
      ].filter(Boolean);
      const overageEnabled =
        providerQuota.overageStatus?.trim().toUpperCase() === 'ENABLED';
      const canToggleOverage =
        Boolean(helpers.item) &&
        Boolean(helpers.onSetKiroOverage) &&
        quota.status !== 'loading' &&
        !quota.disabled &&
        !helpers.quotaDisabled &&
        Boolean(providerQuota.providerQuotaAvailable) &&
        Boolean(providerQuota.overageStatus);
      nodes.push(
        h(
          'div',
          { key: 'overage', className: styleMap.codexPlan },
          h('span', { className: styleMap.codexPlanLabel }, t('kiro_quota.overage')),
          h(
            'div',
            { className: styleMap.overagePlanValue },
            h(
              'span',
              { className: styleMap.codexPlanValue },
              [providerQuota.overageStatus, ...overageParts].filter(Boolean).join(' / ')
            ),
            canToggleOverage
              ? h(
                  'button',
                  {
                    type: 'button',
                    className: styleMap.overageToggle,
                    disabled: quota.overageUpdating,
                    onClick: () => {
                      if (!helpers.item || !helpers.onSetKiroOverage) return;
                      void helpers.onSetKiroOverage(helpers.item, !overageEnabled);
                    },
                  },
                  quota.overageUpdating
                    ? t('kiro_quota.overage_updating')
                    : overageEnabled
                      ? t('kiro_quota.overage_disable')
                      : t('kiro_quota.overage_enable')
                )
              : null
          )
        )
      );
    }
  } else {
    nodes.push(
      h(
        'div',
        { key: 'provider-quota-unavailable', className: styleMap.quotaWarning },
        providerQuota?.message || t('kiro_quota.provider_unavailable')
      )
    );
  }

  if (quota.runtimeUsageStats) {
    const stats = quota.runtimeUsageStats;
    const tokenParts = [
      t('kiro_quota.prompt_tokens', { count: stats.promptTokens }),
      t('kiro_quota.completion_tokens', { count: stats.completionTokens }),
      t('kiro_quota.total_tokens', { count: stats.totalTokens }),
      stats.estimatedTokens > 0
        ? t('kiro_quota.estimated_tokens', { count: stats.estimatedTokens })
        : null,
    ].filter(Boolean);
    nodes.push(
      h(
        'div',
        { key: 'runtime-usage-requests', className: styleMap.codexPlan },
        h('span', { className: styleMap.codexPlanLabel }, t('kiro_quota.runtime_usage')),
        h(
          'span',
          { className: styleMap.codexPlanValue },
          t('kiro_quota.requests', { count: stats.requests })
        )
      ),
      h(
        'div',
        { key: 'runtime-usage-tokens', className: styleMap.codexPlan },
        h('span', { className: styleMap.codexPlanLabel }, t('kiro_quota.runtime_tokens')),
        h('span', { className: styleMap.codexPlanValue }, tokenParts.join(' / '))
      )
    );
    if (stats.lastModel || stats.lastUsedAt) {
      const lastUsedParts = [
        stats.lastModel,
        stats.lastUsedAt ? formatQuotaResetTime(stats.lastUsedAt) : null,
      ].filter(Boolean);
      nodes.push(
        h(
          'div',
          { key: 'runtime-usage-last', className: styleMap.codexPlan },
          h('span', { className: styleMap.codexPlanLabel }, t('kiro_quota.last_used')),
          h(
            'span',
            { className: styleMap.codexPlanValue },
            lastUsedParts.join(' / ')
          )
        )
      );
    }
  }

  nodes.push(
    h(
      'div',
      { key: 'runtime-status', className: styleMap.codexPlan },
      h('span', { className: styleMap.codexPlanLabel }, t('kiro_quota.runtime_status')),
      renderKiroRuntimeStatusBadge(quota.runtimeStatus, t, helpers)
    ),
  );

  if (quota.statusMessage) {
    nodes.push(
      h(
        'div',
        { key: 'status-message', className: styleMap.codexPlan },
        h('span', { className: styleMap.codexPlanLabel }, t('kiro_quota.runtime_reason')),
        h('span', { className: styleMap.codexPlanValue }, quota.statusMessage)
      )
    );
  }

  if (quota.quota.exceeded || quota.quota.reason || quota.quota.nextRecoverAt) {
    nodes.push(
      h(
        'div',
        { key: 'runtime-quota', className: styleMap.quotaRow },
        h(
          'div',
          { className: styleMap.quotaRowHeader },
          h('span', { className: styleMap.quotaModel }, t('kiro_quota.runtime_quota')),
          h(
            'div',
            { className: styleMap.quotaMeta },
            h(
              'span',
              { className: styleMap.quotaPercent },
              quota.quota.exceeded ? t('kiro_quota.exceeded') : t('kiro_quota.available')
            ),
            quota.quota.reason
              ? h('span', { className: styleMap.quotaAmount }, quota.quota.reason)
              : null,
            h(
              'span',
              { className: styleMap.quotaReset },
              formatQuotaResetTime(quota.quota.nextRecoverAt)
            )
          )
        )
      )
    );
  }

  return h(Fragment, null, ...nodes);
};

export const KIRO_CONFIG: QuotaConfig<KiroQuotaState, KiroQuotaState> = {
  type: 'kiro',
  i18nPrefix: 'kiro_quota',
  cardIdleMessageKey: 'kiro_quota.idle',
  filterFn: (file) => isKiroFile(file),
  fetchQuota: fetchKiroQuota,
  storeSelector: (state) => state.kiroQuota,
  storeSetter: 'setKiroQuota',
  buildRuntimeState: buildKiroRuntimeState,
  buildLoadingState: () => ({
    status: 'loading',
    runtimeStatus: 'active',
    unavailable: false,
    disabled: false,
    quota: { exceeded: false },
    modelStates: [],
    providerQuotaAvailable: false,
  }),
  buildSuccessState: (data) => data,
  buildErrorState: (message, status) => ({
    status: 'error',
    runtimeStatus: 'error',
    unavailable: true,
    disabled: false,
    quota: { exceeded: false },
    modelStates: [],
    providerQuotaAvailable: false,
    error: message,
    errorStatus: status,
  }),
  cardClassName: styles.codexCard,
  controlsClassName: styles.codexControls,
  controlClassName: styles.codexControl,
  gridClassName: styles.codexGrid,
  renderQuotaItems: renderKiroItems,
};

const normalizeXaiCentValue = (value: XaiBillingConfig['monthlyLimit']): number | null => {
  if (value === undefined || value === null) return null;
  if (typeof value === 'object' && !Array.isArray(value)) {
    return normalizeNumberValue((value as { val?: unknown }).val);
  }
  return normalizeNumberValue(value);
};

const buildXaiBillingSummary = (
  config: XaiBillingConfig | null | undefined
): XaiBillingSummary | null => {
  if (!config || typeof config !== 'object') return null;

  const monthlyLimitCents = normalizeXaiCentValue(config.monthlyLimit ?? config.monthly_limit);
  const usedCents = normalizeXaiCentValue(config.used);
  const onDemandCapCents = normalizeXaiCentValue(config.onDemandCap ?? config.on_demand_cap);
  const billingPeriodStart =
    normalizeStringValue(config.billingPeriodStart ?? config.billing_period_start) ?? undefined;
  const billingPeriodEnd =
    normalizeStringValue(config.billingPeriodEnd ?? config.billing_period_end) ?? undefined;

  if (
    monthlyLimitCents === null &&
    usedCents === null &&
    onDemandCapCents === null &&
    !billingPeriodEnd
  ) {
    return null;
  }

  const usedPercent =
    monthlyLimitCents !== null && monthlyLimitCents > 0 && usedCents !== null
      ? (usedCents / monthlyLimitCents) * 100
      : null;

  return {
    monthlyLimitCents,
    usedCents,
    onDemandCapCents,
    billingPeriodStart,
    billingPeriodEnd,
    usedPercent,
  };
};

const fetchXaiQuota = async (file: AuthFileItem, t: TFunction): Promise<XaiBillingSummary> => {
  const rawAuthIndex = file['auth_index'] ?? file.authIndex;
  const authIndex = normalizeAuthIndex(rawAuthIndex);
  if (!authIndex) {
    throw new Error(t('xai_quota.missing_auth_index'));
  }

  const result = await apiCallApi.request({
    authIndex,
    method: 'GET',
    url: XAI_BILLING_URL,
    header: { ...XAI_REQUEST_HEADERS },
  });

  if (result.statusCode < 200 || result.statusCode >= 300) {
    throw createStatusError(getApiCallErrorMessage(result), result.statusCode);
  }

  const payload = parseXaiBillingPayload(result.body ?? result.bodyText);
  const summary = buildXaiBillingSummary(payload?.config);
  if (!summary) {
    throw new Error(t('xai_quota.empty_data'));
  }

  return summary;
};

const formatUsdFromCents = (cents: number | null): string => {
  if (cents === null) return '--';
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: 'USD',
  }).format(cents / 100);
};

const formatXaiUsageAmount = (billing: XaiBillingSummary): string => {
  const used = formatUsdFromCents(billing.usedCents);
  const limit = formatUsdFromCents(billing.monthlyLimitCents);
  if (billing.monthlyLimitCents === null) return used;
  return `${used} / ${limit}`;
};

const renderXaiItems = (
  quota: XaiQuotaState,
  t: TFunction,
  helpers: QuotaRenderHelpers
): ReactNode => {
  const { styles: styleMap, QuotaProgressBar } = helpers;
  const { createElement: h, Fragment } = React;
  const billing = quota.billing;

  if (!billing) {
    return h('div', { className: styleMap.quotaMessage }, t('xai_quota.empty_data'));
  }

  const clampedUsed =
    billing.usedPercent === null ? null : Math.max(0, Math.min(100, billing.usedPercent));
  const remaining = clampedUsed === null ? null : Math.max(0, Math.min(100, 100 - clampedUsed));
  const percentLabel = remaining === null ? '--' : `${Math.round(remaining)}%`;
  const amountLabel = formatXaiUsageAmount(billing);
  const resetLabel = formatQuotaResetTime(billing.billingPeriodEnd);
  const onDemandCap = billing.onDemandCapCents ?? 0;
  const payAsYouGoLabel =
    onDemandCap > 0
      ? t('xai_quota.pay_as_you_go_enabled', { cap: formatUsdFromCents(onDemandCap) })
      : t('xai_quota.pay_as_you_go_disabled');

  return h(
    Fragment,
    null,
    h(
      'div',
      { key: 'pay-as-you-go', className: styleMap.codexPlan },
      h('span', { className: styleMap.codexPlanLabel }, t('xai_quota.pay_as_you_go_label')),
      h('span', { className: styleMap.codexPlanValue }, payAsYouGoLabel)
    ),
    h(
      'div',
      { key: 'monthly-credits', className: styleMap.quotaRow },
      h(
        'div',
        { className: styleMap.quotaRowHeader },
        h('span', { className: styleMap.quotaModel }, t('xai_quota.monthly_credits')),
        h(
          'div',
          { className: styleMap.quotaMeta },
          h('span', { className: styleMap.quotaPercent }, percentLabel),
          h('span', { className: styleMap.quotaAmount }, amountLabel),
          h('span', { className: styleMap.quotaReset }, resetLabel)
        )
      ),
      h(QuotaProgressBar, { percent: remaining })
    )
  );
};

export const KIMI_CONFIG: QuotaConfig<KimiQuotaState, KimiQuotaRow[]> = {
  type: 'kimi',
  i18nPrefix: 'kimi_quota',
  cardIdleMessageKey: 'quota_management.card_idle_hint',
  filterFn: (file) => isKimiFile(file) && !isDisabledAuthFile(file),
  fetchQuota: fetchKimiQuota,
  storeSelector: (state) => state.kimiQuota,
  storeSetter: 'setKimiQuota',
  buildLoadingState: () => ({ status: 'loading', rows: [] }),
  buildSuccessState: (rows) => ({ status: 'success', rows }),
  buildErrorState: (message, status) => ({
    status: 'error',
    rows: [],
    error: message,
    errorStatus: status,
  }),
  cardClassName: styles.kimiCard,
  controlsClassName: styles.kimiControls,
  controlClassName: styles.kimiControl,
  gridClassName: styles.kimiGrid,
  renderQuotaItems: renderKimiItems,
};

export const XAI_CONFIG: QuotaConfig<XaiQuotaState, XaiBillingSummary> = {
  type: 'xai',
  i18nPrefix: 'xai_quota',
  cardIdleMessageKey: 'quota_management.card_idle_hint',
  filterFn: (file) => isXaiFile(file) && !isDisabledAuthFile(file),
  fetchQuota: fetchXaiQuota,
  storeSelector: (state) => state.xaiQuota,
  storeSetter: 'setXaiQuota',
  buildLoadingState: () => ({ status: 'loading', billing: null }),
  buildSuccessState: (billing) => ({ status: 'success', billing }),
  buildErrorState: (message, status) => ({
    status: 'error',
    billing: null,
    error: message,
    errorStatus: status,
  }),
  cardClassName: styles.xaiCard,
  controlsClassName: styles.xaiControls,
  controlClassName: styles.xaiControl,
  gridClassName: styles.xaiGrid,
  renderQuotaItems: renderXaiItems,
};
