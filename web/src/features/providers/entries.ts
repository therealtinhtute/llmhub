import type { OAuthProvider } from '@/services/api/oauth';
import {
  hasAuthFileHealthIssue,
  isGeminiVirtualPrimaryAuthFile,
} from '@/features/authFiles/constants';
import type { AuthFileItem } from '@/types';
import type { ProviderPreset } from '@/types';
import iconCodex from '@/assets/icons/codex.svg';
import iconClaude from '@/assets/icons/claude.svg';
import iconAntigravity from '@/assets/icons/antigravity.svg';
import iconGemini from '@/assets/icons/gemini.svg';
import iconKimiLight from '@/assets/icons/kimi-light.svg';
import iconKimiDark from '@/assets/icons/kimi-dark.svg';
import iconGrok from '@/assets/icons/grok.svg';
import iconGrokDark from '@/assets/icons/grok-dark.svg';
import type { ProviderBrand, ProviderGroup, ProviderResource } from './types';

export type ProviderEntryCategory = 'oauth' | 'free' | 'freeTier' | 'apikey' | 'custom';

export interface ProviderEntryOAuthMeta {
  id: OAuthProvider;
  titleKey: string;
  hintKey: string;
  urlLabelKey: string;
  icon: string | { light: string; dark: string };
}

export type ProviderEntry =
  | {
      kind: 'oauth';
      key: string;
      category: 'oauth';
      oauthId: OAuthProvider;
      titleKey: string;
      hintKey: string;
      urlLabelKey: string;
      icon: string | { light: string; dark: string };
      accountCount: number;
      hasIssue: boolean;
    }
  | {
      kind: 'preset';
      key: string;
      category: 'free' | 'freeTier' | 'apikey';
      preset: ProviderPreset;
      resources: ProviderResource[];
    }
  | {
      kind: 'config';
      key: string;
      category: 'apikey' | 'custom';
      group: ProviderGroup;
      resources: ProviderResource[];
    };

export const PROVIDERS: ProviderEntryOAuthMeta[] = [
  {
    id: 'codex',
    titleKey: 'auth_login.codex_oauth_title',
    hintKey: 'auth_login.codex_oauth_hint',
    urlLabelKey: 'auth_login.codex_oauth_url_label',
    icon: iconCodex,
  },
  {
    id: 'anthropic',
    titleKey: 'auth_login.anthropic_oauth_title',
    hintKey: 'auth_login.anthropic_oauth_hint',
    urlLabelKey: 'auth_login.anthropic_oauth_url_label',
    icon: iconClaude,
  },
  {
    id: 'antigravity',
    titleKey: 'auth_login.antigravity_oauth_title',
    hintKey: 'auth_login.antigravity_oauth_hint',
    urlLabelKey: 'auth_login.antigravity_oauth_url_label',
    icon: iconAntigravity,
  },
  {
    id: 'gemini-cli',
    titleKey: 'auth_login.gemini_cli_oauth_title',
    hintKey: 'auth_login.gemini_cli_oauth_hint',
    urlLabelKey: 'auth_login.gemini_cli_oauth_url_label',
    icon: iconGemini,
  },
  {
    id: 'kimi',
    titleKey: 'auth_login.kimi_oauth_title',
    hintKey: 'auth_login.kimi_oauth_hint',
    urlLabelKey: 'auth_login.kimi_oauth_url_label',
    icon: { light: iconKimiLight, dark: iconKimiDark },
  },
  {
    id: 'xai',
    titleKey: 'auth_login.xai_oauth_title',
    hintKey: 'auth_login.xai_oauth_hint',
    urlLabelKey: 'auth_login.xai_oauth_url_label',
    icon: { light: iconGrok, dark: iconGrokDark },
  },
];

export const CALLBACK_SUPPORTED: OAuthProvider[] = [
  'codex',
  'anthropic',
  'antigravity',
  'gemini-cli',
  'xai',
];

export const OAUTH_TO_AUTH_FILE_TYPE: Record<OAuthProvider, string> = {
  codex: 'codex',
  anthropic: 'claude',
  antigravity: 'antigravity',
  'gemini-cli': 'gemini',
  kimi: 'kimi',
  xai: 'xai',
};

export const getOAuthAuthFileTypes = (provider: OAuthProvider): string[] =>
  provider === 'gemini-cli'
    ? ['gemini', 'gemini-cli']
    : [OAUTH_TO_AUTH_FILE_TYPE[provider]];

export const normalizeBaseUrl = (u: string | null | undefined): string =>
  (u ?? '').trim().toLowerCase().replace(/\/+$/, '');

export interface ProviderOAuthState {
  url?: string;
  state?: string;
  status?: 'idle' | 'waiting' | 'success' | 'error';
  error?: string;
  polling?: boolean;
  callbackSubmitting?: boolean;
  callbackStatus?: 'success' | 'error';
  callbackError?: string;
}

export const getProviderI18nPrefix = (provider: OAuthProvider) => provider.replace('-', '_');
export const getAuthKey = (provider: OAuthProvider, suffix: string) =>
  `auth_login.${getProviderI18nPrefix(provider)}_${suffix}`;

const XAI_CALLBACK_URL = 'http://127.0.0.1:56121/callback';

const isAbsoluteUrl = (value: string): boolean => {
  try {
    new URL(value);
    return true;
  } catch {
    return false;
  }
};

const readQueryLikeCallbackInput = (value: string) => {
  const trimmed = value.trim();
  if (!trimmed) return null;
  const queryStart = trimmed.indexOf('?');
  const hashStart = trimmed.indexOf('#');
  const rawParams =
    queryStart >= 0
      ? trimmed.slice(queryStart + 1)
      : hashStart >= 0
        ? trimmed.slice(hashStart + 1)
        : trimmed;

  if (!/(^|[&#?])(code|state|error)=/i.test(rawParams)) return null;
  return new URLSearchParams(rawParams.replace(/^[?#]/, ''));
};

const extractDisplayedXaiCode = (value: string): string => {
  const trimmed = value.trim();
  const codeMatch = trimmed.match(/\bcode\s*[:=]\s*([^\s&]+)/i);
  return (codeMatch?.[1] ?? trimmed).trim();
};

const buildXaiCallbackUrl = (input: string, state?: string): string | null => {
  const trimmed = input.trim();
  if (!trimmed) return null;
  if (isAbsoluteUrl(trimmed)) return trimmed;

  const params = readQueryLikeCallbackInput(trimmed);
  if (params) {
    const code = params.get('code')?.trim();
    const error = params.get('error')?.trim();
    const errorDescription = params.get('error_description')?.trim();
    const callbackState = params.get('state')?.trim() || state?.trim();
    if (!callbackState) return null;

    const callbackUrl = new URL(XAI_CALLBACK_URL);
    callbackUrl.searchParams.set('state', callbackState);
    if (code) callbackUrl.searchParams.set('code', code);
    if (error) callbackUrl.searchParams.set('error', error);
    if (errorDescription) callbackUrl.searchParams.set('error_description', errorDescription);
    return callbackUrl.toString();
  }

  const code = extractDisplayedXaiCode(trimmed);
  const callbackState = state?.trim();
  if (!code || !callbackState) return null;

  const callbackUrl = new URL(XAI_CALLBACK_URL);
  callbackUrl.searchParams.set('code', code);
  callbackUrl.searchParams.set('state', callbackState);
  return callbackUrl.toString();
};

export const resolveCallbackUrl = (
  provider: OAuthProvider,
  input: string,
  state?: string
): string | null => {
  if (provider !== 'xai') return input.trim();
  return buildXaiCallbackUrl(input, state);
};

export const getOAuthIcon = (
  icon: string | { light: string; dark: string },
  theme: 'light' | 'dark'
): string => (typeof icon === 'string' ? icon : theme === 'dark' ? icon.dark : icon.light);

const APIKEY_CONFIG_BRANDS: Exclude<ProviderBrand, 'openaiCompatibility' | 'ampcode'>[] = [
  'gemini',
  'codex',
  'claude',
  'vertex',
];

interface BuildEntriesInput {
  groups: ProviderGroup[];
  presets: ProviderPreset[];
  authFiles: AuthFileItem[];
}

export function buildEntries({ groups, presets, authFiles }: BuildEntriesInput): ProviderEntry[] {
  const entries: ProviderEntry[] = [];

  const oauthEntries: ProviderEntry[] = PROVIDERS.map((p) => {
    const authFileTypes = getOAuthAuthFileTypes(p.id);
    const matchingAuthFiles = authFiles.filter((f) => authFileTypes.includes(String(f.type ?? '')));
    return {
      kind: 'oauth',
      key: `oauth:${p.id}`,
      category: 'oauth',
      oauthId: p.id,
      titleKey: p.titleKey,
      hintKey: p.hintKey,
      urlLabelKey: p.urlLabelKey,
      icon: p.icon,
      accountCount: matchingAuthFiles.length,
      hasIssue: matchingAuthFiles.some(
        (file) =>
          !isGeminiVirtualPrimaryAuthFile(file) && hasAuthFileHealthIssue(file)
      ),
    };
  });
  entries.push(...oauthEntries);

  const openaiGroup = groups.find((g) => g.id === 'openaiCompatibility') ?? null;
  const presetBaseUrlMap = new Map<string, string>();
  presets.forEach((preset) => {
    presetBaseUrlMap.set(normalizeBaseUrl(preset.baseUrl), preset.id);
  });

  const presetResources = new Map<string, ProviderResource[]>();
  const customOpenAIResources: ProviderResource[] = [];
  (openaiGroup?.resources ?? []).forEach((resource) => {
    if (resource.flags.isPlaceholder) return;
    const presetId = presetBaseUrlMap.get(normalizeBaseUrl(resource.baseUrl));
    if (presetId) {
      const bucket = presetResources.get(presetId) ?? [];
      bucket.push(resource);
      presetResources.set(presetId, bucket);
    } else {
      customOpenAIResources.push(resource);
    }
  });

  const presetsByCategory = (category: 'free' | 'freeTier' | 'apikey'): ProviderEntry[] =>
    presets
      .filter((preset) => preset.category === category)
      .map((preset) => ({
        kind: 'preset' as const,
        key: `preset:${preset.id}`,
        category,
        preset,
        resources: presetResources.get(preset.id) ?? [],
      }));

  entries.push(...presetsByCategory('free'));
  entries.push(...presetsByCategory('freeTier'));

  APIKEY_CONFIG_BRANDS.forEach((brand) => {
    const group = groups.find((g) => g.id === brand);
    if (!group) return;
    entries.push({
      kind: 'config',
      key: `config:${brand}`,
      category: 'apikey',
      group,
      resources: group.resources,
    });
  });
  entries.push(...presetsByCategory('apikey'));

  if (openaiGroup) {
    entries.push({
      kind: 'config',
      key: 'config:openaiCompatibility',
      category: 'custom',
      group: openaiGroup,
      resources: customOpenAIResources,
    });
  }
  const ampcodeGroup = groups.find((g) => g.id === 'ampcode');
  if (ampcodeGroup) {
    entries.push({
      kind: 'config',
      key: 'config:ampcode',
      category: 'custom',
      group: ampcodeGroup,
      resources: ampcodeGroup.resources,
    });
  }

  return entries;
}
