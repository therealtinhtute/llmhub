/**
 * 版本相关 API
 */

import { apiClient } from './client';

export const versionApi = {
  checkLatest: () => apiClient.get<Record<string, unknown>>('/latest-version'),
  // Stage the latest verified release candidate for the next restart.
  stageUpdate: () => apiClient.post<Record<string, unknown>>('/self-update'),
  // Trigger the service restart that applies the staged update. The server
  // responds 202 before the process terminates, so no retry logic belongs
  // here.
  applyUpdate: () => apiClient.post<Record<string, unknown>>('/self-update/apply')
};
