import type { ProviderBrand } from './types';

export interface ProviderDescriptor {
  id: ProviderBrand;
  supportsName: boolean;
  supportsApiKey: boolean;
  supportsDisabled: boolean;
  supportsBaseUrl: boolean;
  baseUrlRequired: boolean;
  supportsProxyUrl: boolean;
  supportsPrefix: boolean;
  supportsModels: boolean;
  supportsHeaders: boolean;
  supportsExcludedModels: boolean;
  supportsPriority: boolean;
  supportsWeight: boolean;
  supportsTestModel: boolean;
  supportsWebsockets: boolean;
  supportsCloak: boolean;
  supportsApiKeyEntries: boolean;
  /** Sheet 默认宽度 */
  sheetSize: 'md' | 'lg' | 'xl';
}

export const PROVIDER_DESCRIPTORS: Record<ProviderBrand, ProviderDescriptor> = {
  gemini: {
    id: 'gemini',
    supportsName: false,
    supportsApiKey: true,
    supportsDisabled: true,
    supportsBaseUrl: true,
    baseUrlRequired: false,
    supportsProxyUrl: true,
    supportsPrefix: true,
    supportsModels: true,
    supportsHeaders: true,
    supportsExcludedModels: true,
    supportsPriority: true,
    supportsWeight: true,
    supportsTestModel: false,
    supportsWebsockets: false,
    supportsCloak: false,
    supportsApiKeyEntries: false,
    sheetSize: 'md',
  },
  codex: {
    id: 'codex',
    supportsName: false,
    supportsApiKey: true,
    supportsDisabled: true,
    supportsBaseUrl: true,
    baseUrlRequired: true,
    supportsProxyUrl: true,
    supportsPrefix: true,
    supportsModels: true,
    supportsHeaders: true,
    supportsExcludedModels: true,
    supportsPriority: true,
    supportsWeight: true,
    supportsTestModel: false,
    supportsWebsockets: true,
    supportsCloak: false,
    supportsApiKeyEntries: false,
    sheetSize: 'md',
  },
  claude: {
    id: 'claude',
    supportsName: false,
    supportsApiKey: true,
    supportsDisabled: true,
    supportsBaseUrl: true,
    baseUrlRequired: false,
    supportsProxyUrl: true,
    supportsPrefix: true,
    supportsModels: true,
    supportsHeaders: true,
    supportsExcludedModels: true,
    supportsPriority: true,
    supportsWeight: true,
    supportsTestModel: true,
    supportsWebsockets: false,
    supportsCloak: true,
    supportsApiKeyEntries: false,
    sheetSize: 'md',
  },
  vertex: {
    id: 'vertex',
    supportsName: false,
    supportsApiKey: true,
    supportsDisabled: true,
    supportsBaseUrl: true,
    baseUrlRequired: false,
    supportsProxyUrl: true,
    supportsPrefix: true,
    supportsModels: true,
    supportsHeaders: true,
    supportsExcludedModels: true,
    supportsPriority: true,
    supportsWeight: true,
    supportsTestModel: false,
    supportsWebsockets: false,
    supportsCloak: false,
    supportsApiKeyEntries: false,
    sheetSize: 'md',
  },
  openaiCompatibility: {
    id: 'openaiCompatibility',
    supportsName: true,
    supportsApiKey: false,
    supportsDisabled: true,
    supportsBaseUrl: true,
    baseUrlRequired: true,
    supportsProxyUrl: false,
    supportsPrefix: true,
    supportsModels: true,
    supportsHeaders: true,
    supportsExcludedModels: false,
    supportsPriority: true,
    supportsWeight: true,
    supportsTestModel: true,
    supportsWebsockets: false,
    supportsCloak: false,
    supportsApiKeyEntries: true,
    sheetSize: 'lg',
  },
  openrouter: {
    id: 'openrouter',
    supportsName: false,
    supportsApiKey: true,
    supportsDisabled: true,
    supportsBaseUrl: false,
    baseUrlRequired: false,
    supportsProxyUrl: false,
    supportsPrefix: false,
    supportsModels: false,
    supportsHeaders: false,
    supportsExcludedModels: false,
    supportsPriority: false,
    supportsWeight: false,
    supportsTestModel: false,
    supportsWebsockets: false,
    supportsCloak: false,
    supportsApiKeyEntries: false,
    sheetSize: 'lg',
  },
  opencode: {
    id: 'opencode',
    supportsName: false,
    supportsApiKey: false,
    supportsDisabled: true,
    supportsBaseUrl: false,
    baseUrlRequired: false,
    supportsProxyUrl: false,
    supportsPrefix: false,
    supportsModels: false,
    supportsHeaders: false,
    supportsExcludedModels: false,
    supportsPriority: false,
    supportsWeight: false,
    supportsTestModel: false,
    supportsWebsockets: false,
    supportsCloak: false,
    supportsApiKeyEntries: false,
    sheetSize: 'lg',
  },
};

export const PROVIDER_BRAND_ORDER: ProviderBrand[] = [
  'gemini',
  'codex',
  'claude',
  'vertex',
  'openrouter',
  'opencode',
  'openaiCompatibility',
];

export const PROVIDER_PATHS: Record<ProviderBrand, string> = {
  gemini: '/ai-providers/gemini',
  codex: '/ai-providers/codex',
  claude: '/ai-providers/claude',
  vertex: '/ai-providers/vertex',
  openaiCompatibility: '/ai-providers/openai',
  openrouter: '/ai-providers/openrouter',
  opencode: '/ai-providers/opencode',
};
