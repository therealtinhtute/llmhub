/**
 * 认证文件相关类型
 * 基于原项目 src/modules/auth-files.js
 */

import type { RecentRequestBucket } from '@/utils/recentRequests';

export type AuthFileType =
  | 'qwen'
  | 'kimi'
  | 'gemini'
  | 'gemini-cli'
  | 'aistudio'
  | 'claude'
  | 'codex'
  | 'antigravity'
  | 'kiro'
  | 'xai'
  | 'iflow'
  | 'vertex'
  | 'empty'
  | 'unknown';

export interface AuthFileItem {
  name: string;
  type?: AuthFileType | string;
  provider?: string;
  size?: number;
  authIndex?: string | number | null;
  runtimeOnly?: boolean | string;
  disabled?: boolean;
  unavailable?: boolean;
  status?: string;
  statusMessage?: string;
  quota?: RuntimeQuotaState;
  model_states?: Record<string, RuntimeModelState>;
  modelStates?: Record<string, RuntimeModelState>;
  lastRefresh?: string | number;
  modified?: number;
  success?: unknown;
  failed?: unknown;
  recent_requests?: RecentRequestBucket[];
  recentRequests?: RecentRequestBucket[];
  [key: string]: unknown;
}

export interface AuthFilesResponse {
  files: AuthFileItem[];
  total?: number;
}

export interface RuntimeQuotaState {
  exceeded?: boolean;
  reason?: string;
  next_recover_at?: string;
  nextRecoverAt?: string;
  backoff_level?: number;
  backoffLevel?: number;
}

export interface RuntimeModelState {
  status?: string;
  status_message?: string;
  statusMessage?: string;
  unavailable?: boolean;
  next_retry_after?: string;
  nextRetryAfter?: string;
  last_error?: unknown;
  lastError?: unknown;
  quota?: RuntimeQuotaState;
  updated_at?: string;
  updatedAt?: string;
}
