import { normalizePlanType } from './parsers';

export const PREMIUM_CODEX_PLAN_TYPES: Record<string, true> = {
  prolite: true,
  'pro-lite': true,
  pro_lite: true,
};

export type PlanTier = 'elite' | 'premium' | 'plain';

export function resolvePlanTier(planType?: string | null): PlanTier {
  const normalized = normalizePlanType(planType);
  if (!normalized) return 'plain';
  if (normalized === 'pro') return 'elite';
  if (PREMIUM_CODEX_PLAN_TYPES[normalized]) return 'premium';
  return 'plain';
}
