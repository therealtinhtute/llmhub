import { apiClient } from './client';

export interface ResetUsageResponse {
  status: string;
  auth_index: string;
}

export const usageApi = {
  resetUsage: (authIndex: string) =>
    apiClient.post<ResetUsageResponse>('/reset-usage', { auth_index: authIndex }),
};
