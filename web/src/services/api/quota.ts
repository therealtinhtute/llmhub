import { apiClient } from './client';

export interface ResetQuotaResponse {
  status: string;
  auth_index: string;
  models: string[];
}

export const quotaApi = {
  resetQuota: (authIndex: string) =>
    apiClient.post<ResetQuotaResponse>('/reset-quota', { auth_index: authIndex }),
};
