import { apiClient } from './client';
import type { RuntimeControlSettings } from '@/types/runtimeControls';

export const runtimeControlsApi = {
  get: () => apiClient.get<RuntimeControlSettings>('/runtime-controls'),
  save: (settings: RuntimeControlSettings) =>
    apiClient.put<RuntimeControlSettings>('/runtime-controls', settings),
};
