import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { AppCard as Card } from '@/components/ui/AppCard';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/Button';
import { EmptyState } from '@/components/ui/EmptyState';
import { Input } from '@/components/ui/Input';
import { OAuthModelAliasCard } from '@/features/authFiles/components/OAuthModelAliasCard';
import { useAuthFilesOauth } from '@/features/authFiles/hooks/useAuthFilesOauth';
import { authFilesApi } from '@/services/api';
import { apiKeysApi } from '@/services/api/apiKeys';
import { useAuthStore, useConfigStore, useModelsStore, useThemeStore } from '@/stores';
import type { AuthFileItem } from '@/types';
import { classifyModels } from '@/utils/models';
import iconClaude from '@/assets/icons/claude.svg';
import iconDeepseek from '@/assets/icons/deepseek.svg';
import iconGemini from '@/assets/icons/gemini.svg';
import iconGlm from '@/assets/icons/glm.svg';
import iconGrok from '@/assets/icons/grok.svg';
import iconGrokDark from '@/assets/icons/grok-dark.svg';
import iconKimiLight from '@/assets/icons/kimi-light.svg';
import iconKimiDark from '@/assets/icons/kimi-dark.svg';
import iconMinimax from '@/assets/icons/minimax.svg';
import iconOpenaiLight from '@/assets/icons/openai-light.svg';
import iconOpenaiDark from '@/assets/icons/openai-dark.svg';
import iconQwen from '@/assets/icons/qwen.svg';

type AliasViewMode = 'diagram' | 'list';
type StatusType = 'success' | 'warning' | 'error' | 'muted';

const MODEL_CATEGORY_ICONS: Record<string, string | { light: string; dark: string }> = {
  gpt: { light: iconOpenaiLight, dark: iconOpenaiDark },
  claude: iconClaude,
  gemini: iconGemini,
  qwen: iconQwen,
  kimi: { light: iconKimiLight, dark: iconKimiDark },
  glm: iconGlm,
  grok: { light: iconGrok, dark: iconGrokDark },
  deepseek: iconDeepseek,
  minimax: iconMinimax,
};

const normalizeApiKeyList = (input: unknown): string[] => {
  if (!Array.isArray(input)) return [];
  const seen = new Set<string>();
  const keys: string[] = [];

  input.forEach((item) => {
    const record =
      item !== null && typeof item === 'object' && !Array.isArray(item)
        ? (item as Record<string, unknown>)
        : null;
    const value =
      typeof item === 'string'
        ? item
        : record
          ? (record['api-key'] ?? record['apiKey'] ?? record.key ?? record.Key)
          : '';
    const trimmed = String(value ?? '').trim();
    if (!trimmed || seen.has(trimmed)) return;
    seen.add(trimmed);
    keys.push(trimmed);
  });

  return keys;
};

const statusClassName = (type: StatusType) => {
  if (type === 'success') return 'text-emerald-700 bg-emerald-100 border-emerald-400/40';
  if (type === 'error') return 'text-destructive bg-destructive/10 border-destructive/30';
  if (type === 'warning') return 'text-amber-700 bg-amber-100 border-amber-400/30';
  return 'border-border text-muted-foreground bg-muted';
};

export function ModelsPage() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const resolvedTheme = useThemeStore((state) => state.resolvedTheme);
  const auth = useAuthStore();
  const config = useConfigStore((state) => state.config);
  const models = useModelsStore((state) => state.models);
  const modelsLoading = useModelsStore((state) => state.loading);
  const modelsError = useModelsStore((state) => state.error);
  const fetchModelsFromStore = useModelsStore((state) => state.fetchModels);

  const [modelStatus, setModelStatus] = useState<{
    type: StatusType;
    message: string;
  }>();
  const [search, setSearch] = useState('');
  const [authFiles, setAuthFiles] = useState<AuthFileItem[]>([]);
  const [aliasViewMode, setAliasViewMode] = useState<AliasViewMode>('diagram');
  const [aliasLoading, setAliasLoading] = useState(true);

  const apiKeysCache = useRef<string[]>([]);

  const {
    modelAlias,
    modelAliasError,
    allProviderModels,
    loadModelAlias,
    deleteModelAlias,
    handleMappingUpdate,
    handleDeleteLink,
    handleToggleFork,
    handleRenameAlias,
    handleDeleteAlias,
  } = useAuthFilesOauth({ viewMode: aliasViewMode, files: authFiles });

  const otherLabel = useMemo(
    () => (i18n.language?.toLowerCase().startsWith('vi') ? 'Khác' : 'Other'),
    [i18n.language]
  );

  const filteredModels = useMemo(() => {
    const query = search.trim().toLowerCase();
    if (!query) return models;
    return models.filter((model) => {
      const haystack = `${model.name} ${model.alias ?? ''} ${model.description ?? ''}`.toLowerCase();
      return haystack.includes(query);
    });
  }, [models, search]);

  const groupedModels = useMemo(
    () => classifyModels(filteredModels, { otherLabel }),
    [filteredModels, otherLabel]
  );

  const aliasProviderCount = Object.keys(modelAlias).length;
  const aliasMappingCount = Object.values(modelAlias).reduce(
    (sum, mappings) => sum + mappings.length,
    0
  );
  const disableAliasControls = auth.connectionStatus !== 'connected' || aliasLoading;

  const getIconForCategory = (categoryId: string): string | null => {
    const iconEntry = MODEL_CATEGORY_ICONS[categoryId];
    if (!iconEntry) return null;
    if (typeof iconEntry === 'string') return iconEntry;
    return resolvedTheme === 'dark' ? iconEntry.dark : iconEntry.light;
  };

  const resolveApiKeysForModels = useCallback(async () => {
    if (apiKeysCache.current.length) {
      return apiKeysCache.current;
    }

    const configKeys = normalizeApiKeyList(config?.apiKeys);
    if (configKeys.length) {
      apiKeysCache.current = configKeys;
      return configKeys;
    }

    try {
      const list = await apiKeysApi.list();
      const normalized = normalizeApiKeyList(list);
      if (normalized.length) {
        apiKeysCache.current = normalized;
      }
      return normalized;
    } catch (err) {
      console.warn('Auto loading API keys for models failed:', err);
      return [];
    }
  }, [config?.apiKeys]);

  const fetchModels = useCallback(
    async ({ forceRefresh = false }: { forceRefresh?: boolean } = {}) => {
      if (auth.connectionStatus !== 'connected') {
        setModelStatus({
          type: 'warning',
          message: t('notification.connection_required'),
        });
        return;
      }

      if (!auth.apiBase) {
        toast.warning(t('notification.connection_required'));
        return;
      }

      if (forceRefresh) {
        apiKeysCache.current = [];
      }

      setModelStatus({ type: 'muted', message: t('models_management.models_loading') });
      try {
        const apiKeys = await resolveApiKeysForModels();
        const primaryKey = apiKeys[0];
        const list = await fetchModelsFromStore(auth.apiBase, primaryKey, forceRefresh);
        const hasModels = list.length > 0;
        setModelStatus({
          type: hasModels ? 'success' : 'warning',
          message: hasModels
            ? t('models_management.models_count', { count: list.length })
            : t('models_management.models_empty'),
        });
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : typeof err === 'string' ? err : '';
        const suffix = message ? `: ${message}` : '';
        setModelStatus({
          type: 'error',
          message: `${t('models_management.models_error')}${suffix}`,
        });
      }
    },
    [
      auth.apiBase,
      auth.connectionStatus,
      fetchModelsFromStore,
      resolveApiKeysForModels,
      t,
    ]
  );

  const loadAliasData = useCallback(async () => {
    setAliasLoading(true);
    try {
      const filesResult = await authFilesApi.list();
      setAuthFiles(filesResult.files ?? []);
      await loadModelAlias();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : '';
      if (message) {
        toast.error(`${t('notification.load_failed')}: ${message}`);
      }
    } finally {
      setAliasLoading(false);
    }
  }, [loadModelAlias, t]);

  const openModelAliasEditor = useCallback(
    (provider?: string) => {
      const nextSearch = provider ? `?provider=${encodeURIComponent(provider)}` : '';
      navigate(`/auth-files/oauth-model-alias${nextSearch}`, {
        state: { returnTo: '/models' },
      });
    },
    [navigate]
  );

  useEffect(() => {
    fetchModels();
  }, [fetchModels]);

  useEffect(() => {
    if (auth.connectionStatus !== 'connected') {
      setAliasLoading(false);
      return;
    }
    void loadAliasData();
  }, [auth.connectionStatus, loadAliasData]);

  return (
    <div className="flex w-full flex-col gap-6">
      <div className="flex items-start justify-between gap-4 max-md:flex-col">
        <div className="flex min-w-0 flex-col gap-2">
          <h1 className="m-0 text-[28px] font-bold text-foreground">
            {t('models_management.title')}
          </h1>
          <p className="m-0 max-w-[760px] text-sm leading-6 text-muted-foreground">
            {t('models_management.description')}
          </p>
        </div>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => fetchModels({ forceRefresh: true })}
          loading={modelsLoading}
        >
          {t('common.refresh')}
        </Button>
      </div>

      <div className="grid grid-cols-3 gap-3 max-[780px]:grid-cols-1">
        <Card>
          <div className="flex flex-col gap-1">
            <span className="text-xs font-semibold uppercase text-muted-foreground">
              {t('models_management.total_models')}
            </span>
            <span className="text-[30px] font-bold leading-tight text-foreground">
              {modelsLoading ? '...' : models.length}
            </span>
          </div>
        </Card>
        <Card>
          <div className="flex flex-col gap-1">
            <span className="text-xs font-semibold uppercase text-muted-foreground">
              {t('models_management.provider_count')}
            </span>
            <span className="text-[30px] font-bold leading-tight text-foreground">
              {aliasLoading ? '...' : aliasProviderCount}
            </span>
          </div>
        </Card>
        <Card>
          <div className="flex flex-col gap-1">
            <span className="text-xs font-semibold uppercase text-muted-foreground">
              {t('models_management.alias_count')}
            </span>
            <span className="text-[30px] font-bold leading-tight text-foreground">
              {aliasLoading ? '...' : aliasMappingCount}
            </span>
          </div>
        </Card>
      </div>

      <Card
        title={t('models_management.available_models_title')}
        extra={
          <div className="w-[min(320px,42vw)] max-md:w-full">
            <Input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder={t('models_management.search_placeholder')}
              aria-label={t('models_management.search_placeholder')}
            />
          </div>
        }
      >
        <div className="flex flex-col gap-3">
          {modelStatus && (
            <div
              className={`inline-flex w-fit items-center rounded-sm border px-[10px] py-[2px] text-[0.8125rem] font-medium leading-[1.5] ${statusClassName(modelStatus.type)}`}
            >
              {modelStatus.message}
            </div>
          )}
          {modelsError && (
            <div className="border border-destructive/35 bg-destructive/10 p-[10px_14px] text-sm leading-[1.5] text-destructive">
              {modelsError}
            </div>
          )}
          {modelsLoading ? (
            <div className="text-[13px] leading-[1.55] text-muted-foreground">
              {t('common.loading')}
            </div>
          ) : models.length === 0 ? (
            <EmptyState title={t('models_management.models_empty')} />
          ) : filteredModels.length === 0 ? (
            <EmptyState title={t('models_management.search_empty')} />
          ) : (
            <div className="flex flex-col">
              {groupedModels.map((group) => {
                const iconSrc = getIconForCategory(group.id);
                return (
                  <div
                    key={group.id}
                    className="grid grid-cols-[minmax(160px,220px)_1fr] gap-4 border-b border-border py-3 last:border-b-0 max-[780px]:grid-cols-1"
                  >
                    <div className="flex min-w-0 flex-col gap-1">
                      <div className="flex items-center gap-2">
                        {iconSrc && <img src={iconSrc} alt="" className="size-[18px] shrink-0" />}
                        <span className="text-sm font-semibold text-foreground">
                          {group.label}
                        </span>
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {t('models_management.models_count', { count: group.items.length })}
                      </div>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      {group.items.map((model) => (
                        <Badge
                          key={`${model.name}-${model.alias ?? 'default'}`}
                          variant="outline"
                          className="max-w-full gap-1 rounded-md bg-muted px-2.5 py-1 font-mono"
                          title={model.description || model.alias || model.name}
                        >
                          <span className="truncate font-semibold text-foreground">
                            {model.name}
                          </span>
                          {model.alias && (
                            <span className="truncate text-xs text-muted-foreground">
                              {model.alias}
                            </span>
                          )}
                        </Badge>
                      ))}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </Card>

      <OAuthModelAliasCard
        disableControls={disableAliasControls}
        viewMode={aliasViewMode}
        onViewModeChange={setAliasViewMode}
        onAdd={() => openModelAliasEditor()}
        onEditProvider={openModelAliasEditor}
        onDeleteProvider={deleteModelAlias}
        modelAliasError={modelAliasError}
        modelAlias={modelAlias}
        allProviderModels={allProviderModels}
        onUpdate={handleMappingUpdate}
        onDeleteLink={handleDeleteLink}
        onToggleFork={handleToggleFork}
        onRenameAlias={handleRenameAlias}
        onDeleteAlias={handleDeleteAlias}
      />
    </div>
  );
}
