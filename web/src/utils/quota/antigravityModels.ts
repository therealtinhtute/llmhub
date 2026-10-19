/**
 * Model-level Antigravity quota fallback.
 *
 * `retrieveUserQuotaSummary` can omit the Gemini group (for example when the
 * upstream only serves it to newer clients, or when a staging host returns a
 * partial payload). `fetchAvailableModels` still reports per-model quota, so
 * these helpers synthesize summary groups from it.
 *
 * Models that match no `ANTIGRAVITY_QUOTA_GROUPS` identifier are never
 * dropped: every remaining `gemini-*` model is collected into one fallback
 * group so a future model-ID rename cannot hide Gemini quota again.
 */

import type {
  AntigravityQuotaGroupDefinition,
  AntigravityQuotaSummaryBucketPayload,
  AntigravityQuotaSummaryGroupPayload,
} from '@/types';
import { ANTIGRAVITY_QUOTA_GROUPS } from './constants';

export const ANTIGRAVITY_FALLBACK_GROUP_LABEL = 'Gemini (other)';

type UnknownRecord = Record<string, unknown>;

function asRecord(value: unknown): UnknownRecord | null {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as UnknownRecord)
    : null;
}

function normalizeString(value: unknown): string | null {
  if (typeof value === 'string') {
    const trimmed = value.trim();
    return trimmed || null;
  }
  if (typeof value === 'number' && Number.isFinite(value)) return String(value);
  return null;
}

function normalizeFraction(value: unknown): number | null {
  if (typeof value === 'number' && Number.isFinite(value)) return value;
  if (typeof value !== 'string') return null;
  const trimmed = value.trim();
  if (!trimmed) return null;
  if (trimmed.endsWith('%')) {
    const parsed = Number(trimmed.slice(0, -1));
    return Number.isFinite(parsed) ? parsed / 100 : null;
  }
  const parsed = Number(trimmed);
  return Number.isFinite(parsed) ? parsed : null;
}

type ModelQuota = {
  id: string;
  displayName?: string;
  remainingFraction: number;
  resetTime?: string;
};

function readModelQuota(id: string, raw: unknown): ModelQuota | null {
  const model = asRecord(raw);
  if (!model) return null;
  const quota = asRecord(model.quotaInfo) ?? asRecord(model.quota_info);
  if (!quota) return null;

  const resetTime = normalizeString(quota.resetTime ?? quota.reset_time) ?? undefined;
  let remainingFraction = normalizeFraction(
    quota.remainingFraction ?? quota.remaining_fraction ?? quota.remaining
  );
  if (remainingFraction === null) {
    // proto3 JSON omits zero values, so an exhausted model can arrive with
    // only a reset time. Treat it as 0 instead of dropping the model.
    if (!resetTime) return null;
    remainingFraction = 0;
  }

  return {
    id,
    displayName: normalizeString(model.displayName ?? model.display_name) ?? undefined,
    remainingFraction,
    resetTime,
  };
}

function findModelQuota(
  models: Record<string, unknown>,
  identifier: string
): ModelQuota | null {
  const direct = readModelQuota(identifier, models[identifier]);
  if (direct) return direct;

  const wanted = identifier.toLowerCase();
  for (const [id, raw] of Object.entries(models)) {
    const model = asRecord(raw);
    const name = normalizeString(model?.displayName ?? model?.display_name);
    if (name && name.toLowerCase() === wanted) {
      return readModelQuota(id, raw);
    }
  }
  return null;
}

function toBucket(model: ModelQuota, fallbackLabel: string): AntigravityQuotaSummaryBucketPayload {
  return {
    bucketId: model.id,
    displayName: model.displayName ?? fallbackLabel,
    remainingFraction: model.remainingFraction,
    resetTime: model.resetTime,
  };
}

function toGroup(
  definition: AntigravityQuotaGroupDefinition,
  models: Record<string, unknown>,
  matched: Set<string>
): AntigravityQuotaSummaryGroupPayload | null {
  const quotas: ModelQuota[] = [];

  definition.identifiers.forEach((identifier) => {
    const quota = findModelQuota(models, identifier);
    if (!quota) return;
    matched.add(quota.id);
    quotas.push(quota);
  });

  if (quotas.length === 0) return null;

  const modelLabel = definition.labelFromModel
    ? quotas.map((quota) => quota.displayName).find(Boolean)
    : undefined;

  return {
    displayName: modelLabel ?? definition.label,
    buckets: quotas.map((quota) => toBucket(quota, definition.label)),
  };
}

export function buildAntigravitySummaryGroupsFromModels(
  models: Record<string, unknown>
): AntigravityQuotaSummaryGroupPayload[] {
  const matched = new Set<string>();
  const groups: AntigravityQuotaSummaryGroupPayload[] = [];

  ANTIGRAVITY_QUOTA_GROUPS.forEach((definition) => {
    const group = toGroup(definition, models, matched);
    if (group) groups.push(group);
  });

  const fallbackBuckets: AntigravityQuotaSummaryBucketPayload[] = [];
  Object.entries(models).forEach(([id, raw]) => {
    if (matched.has(id) || !/^gemini-/i.test(id.trim())) return;
    const quota = readModelQuota(id, raw);
    if (quota) fallbackBuckets.push(toBucket(quota, ANTIGRAVITY_FALLBACK_GROUP_LABEL));
  });

  if (fallbackBuckets.length > 0) {
    groups.push({
      displayName: ANTIGRAVITY_FALLBACK_GROUP_LABEL,
      buckets: fallbackBuckets,
    });
  }

  return groups;
}

export function hasAntigravityGeminiGroup(groups: unknown[]): boolean {
  return groups.some((raw) => {
    const group = asRecord(raw);
    if (!group) return false;

    const name = normalizeString(group.displayName ?? group.display_name) ?? '';
    const description = normalizeString(group.description) ?? '';
    if (/gemini/i.test(name) || /gemini/i.test(description)) return true;

    const buckets = Array.isArray(group.buckets) ? group.buckets : [];
    return buckets.some((bucketRaw) => {
      const bucket = asRecord(bucketRaw);
      if (!bucket) return false;
      const bucketId = normalizeString(bucket.bucketId ?? bucket.bucket_id) ?? '';
      const bucketName = normalizeString(bucket.displayName ?? bucket.display_name) ?? '';
      return /^gemini-/i.test(bucketId) || /gemini/i.test(bucketName);
    });
  });
}
