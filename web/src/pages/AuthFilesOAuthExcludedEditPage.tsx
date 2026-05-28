import { useCallback, useEffect, useMemo, useState } from 'react';
import { useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Card } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { LoadingSpinner } from '@/components/ui/LoadingSpinner';
import { SelectionCheckbox } from '@/components/ui/SelectionCheckbox';
import { AutocompleteInput } from '@/components/ui/AutocompleteInput';
import { EmptyState } from '@/components/ui/EmptyState';
import { IconInfo } from '@/components/ui/icons';
import { SecondaryScreenShell } from '@/components/common/SecondaryScreenShell';
import { useEdgeSwipeBack } from '@/hooks/useEdgeSwipeBack';
import { toast } from 'sonner';
import { useAuthStore } from '@/stores';
import { authFilesApi } from '@/services/api';
import type { AuthFileItem, OAuthModelAliasEntry } from '@/types';

type AuthFileModelItem = { id: string; display_name?: string; type?: string; owned_by?: string };

type LocationState = { fromAuthFiles?: boolean } | null;

const OAUTH_PROVIDER_PRESETS = [
  'gemini-cli',
  'vertex',
  'aistudio',
  'antigravity',
  'claude',
  'codex',
  'qwen',
  'kimi',
  'iflow',
];

const OAUTH_PROVIDER_EXCLUDES = new Set(['all', 'unknown', 'empty']);

const normalizeProviderKey = (value: string) => value.trim().toLowerCase();

export function AuthFilesOAuthExcludedEditPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();
  const connectionStatus = useAuthStore((state) => state.connectionStatus);
  const disableControls = connectionStatus !== 'connected';

  const [searchParams, setSearchParams] = useSearchParams();
  const providerFromParams = searchParams.get('provider') ?? '';

  const [provider, setProvider] = useState(providerFromParams);
  const [files, setFiles] = useState<AuthFileItem[]>([]);
  const [excluded, setExcluded] = useState<Record<string, string[]>>({});
  const [modelAlias, setModelAlias] = useState<Record<string, OAuthModelAliasEntry[]>>({});
  const [initialLoading, setInitialLoading] = useState(true);
  const [excludedUnsupported, setExcludedUnsupported] = useState(false);

  const [selectedModels, setSelectedModels] = useState<Set<string>>(new Set());
  const [modelsList, setModelsList] = useState<AuthFileModelItem[]>([]);
  const [modelsLoading, setModelsLoading] = useState(false);
  const [modelsError, setModelsError] = useState<'unsupported' | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setProvider(providerFromParams);
  }, [providerFromParams]);

  const providerOptions = useMemo(() => {
    const extraProviders = new Set<string>();
    Object.keys(excluded).forEach((value) => extraProviders.add(value));
    Object.keys(modelAlias).forEach((value) => extraProviders.add(value));
    files.forEach((file) => {
      if (typeof file.type === 'string') {
        extraProviders.add(file.type);
      }
      if (typeof file.provider === 'string') {
        extraProviders.add(file.provider);
      }
    });

    const normalizedExtras = Array.from(extraProviders)
      .map((value) => value.trim())
      .filter((value) => value && !OAUTH_PROVIDER_EXCLUDES.has(value.toLowerCase()));

    const baseSet = new Set(OAUTH_PROVIDER_PRESETS.map((value) => value.toLowerCase()));
    const extraList = normalizedExtras
      .filter((value) => !baseSet.has(value.toLowerCase()))
      .sort((a, b) => a.localeCompare(b));

    return [...OAUTH_PROVIDER_PRESETS, ...extraList];
  }, [excluded, files, modelAlias]);

  const getTypeLabel = useCallback(
    (type: string): string => {
      const key = `auth_files.filter_${type}`;
      const translated = t(key);
      if (translated !== key) return translated;
      if (type.toLowerCase() === 'iflow') return 'iFlow';
      return type.charAt(0).toUpperCase() + type.slice(1);
    },
    [t]
  );

  const resolvedProviderKey = useMemo(() => normalizeProviderKey(provider), [provider]);
  const isEditing = useMemo(() => {
    if (!resolvedProviderKey) return false;
    return Object.prototype.hasOwnProperty.call(excluded, resolvedProviderKey);
  }, [excluded, resolvedProviderKey]);

  const title = useMemo(() => {
    if (isEditing) {
      return t('oauth_excluded.edit_title', { provider: provider.trim() || resolvedProviderKey });
    }
    return t('oauth_excluded.add_title');
  }, [isEditing, provider, resolvedProviderKey, t]);

  const handleBack = useCallback(() => {
    const state = location.state as LocationState;
    if (state?.fromAuthFiles) {
      navigate(-1);
      return;
    }
    navigate('/auth-files', { replace: true });
  }, [location.state, navigate]);

  const swipeRef = useEdgeSwipeBack({ onBack: handleBack });

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        handleBack();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [handleBack]);

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      setInitialLoading(true);
      setExcludedUnsupported(false);
      try {
        const [filesResult, excludedResult, aliasResult] = await Promise.allSettled([
          authFilesApi.list(),
          authFilesApi.getOauthExcludedModels(),
          authFilesApi.getOauthModelAlias(),
        ]);

        if (cancelled) return;

        if (filesResult.status === 'fulfilled') {
          setFiles(filesResult.value?.files ?? []);
        }

        if (aliasResult.status === 'fulfilled') {
          setModelAlias(aliasResult.value ?? {});
        }

        if (excludedResult.status === 'fulfilled') {
          setExcluded(excludedResult.value ?? {});
          return;
        }

        const err = excludedResult.status === 'rejected' ? excludedResult.reason : null;
        const status =
          typeof err === 'object' && err !== null && 'status' in err
            ? (err as { status?: unknown }).status
            : undefined;

        if (status === 404) {
          setExcludedUnsupported(true);
          return;
        }
      } finally {
        if (!cancelled) {
          setInitialLoading(false);
        }
      }
    };

    load().catch(() => {
      if (!cancelled) {
        setInitialLoading(false);
      }
    });

    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!resolvedProviderKey) {
      setSelectedModels(new Set());
      return;
    }
    const existing = excluded[resolvedProviderKey] ?? [];
    setSelectedModels(new Set(existing));
  }, [excluded, resolvedProviderKey]);

  useEffect(() => {
    if (!resolvedProviderKey || excludedUnsupported) {
      setModelsList([]);
      setModelsError(null);
      setModelsLoading(false);
      return;
    }

    let cancelled = false;
    setModelsLoading(true);
    setModelsError(null);

    authFilesApi
      .getModelDefinitions(resolvedProviderKey)
      .then((models) => {
        if (cancelled) return;
        setModelsList(models);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        const status =
          typeof err === 'object' && err !== null && 'status' in err
            ? (err as { status?: unknown }).status
            : undefined;

        if (status === 404) {
          setModelsList([]);
          setModelsError('unsupported');
          return;
        }

        const errorMessage = err instanceof Error ? err.message : '';
        toast.error(`${t('notification.load_failed')}: ${errorMessage}`);
      })
      .finally(() => {
        if (cancelled) return;
        setModelsLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [excludedUnsupported, resolvedProviderKey, t]);

  const updateProvider = useCallback(
    (value: string) => {
      setProvider(value);
      const next = new URLSearchParams(searchParams);
      const trimmed = value.trim();
      if (trimmed) {
        next.set('provider', trimmed);
      } else {
        next.delete('provider');
      }
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  const toggleModel = useCallback((modelId: string, checked: boolean) => {
    setSelectedModels((prev) => {
      const next = new Set(prev);
      if (checked) {
        next.add(modelId);
      } else {
        next.delete(modelId);
      }
      return next;
    });
  }, []);

  const handleSave = useCallback(async () => {
    const normalizedProvider = normalizeProviderKey(provider);
    if (!normalizedProvider) {
      toast.error(t('oauth_excluded.provider_required'));
      return;
    }

    const models = [...selectedModels];
    setSaving(true);
    try {
      if (models.length) {
        await authFilesApi.saveOauthExcludedModels(normalizedProvider, models);
      } else {
        await authFilesApi.deleteOauthExcludedEntry(normalizedProvider);
      }
      toast.success(t('oauth_excluded.save_success'));
      handleBack();
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : '';
      toast.error(`${t('oauth_excluded.save_failed')}: ${errorMessage}`);
    } finally {
      setSaving(false);
    }
  }, [handleBack, provider, selectedModels, t]);

  const canSave = !disableControls && !saving && !excludedUnsupported;

  return (
    <SecondaryScreenShell
      ref={swipeRef}
      title={title}
      onBack={handleBack}
      backLabel={t('common.back')}
      backAriaLabel={t('common.back')}
      contentClassName="w-full max-w-[1000px] mx-auto px-4 pb-12 max-md:px-4"
      rightAction={
        <Button size="sm" onClick={handleSave} loading={saving} disabled={!canSave}>
          {t('oauth_excluded.save')}
        </Button>
      }
      isLoading={initialLoading}
      loadingLabel={t('common.loading')}
    >
      {excludedUnsupported ? (
        <Card>
          <EmptyState
            title={t('oauth_excluded.upgrade_required_title')}
            description={t('oauth_excluded.upgrade_required_desc')}
          />
        </Card>
      ) : (
        <>
          <Card className="p-0 overflow-visible">
            <div className="flex flex-col gap-1 px-4 py-3 border-b border-border max-md:px-4">
              <div className="inline-flex items-center gap-1 font-bold text-foreground">
                <IconInfo size={16} />
                <span>{t('oauth_excluded.title')}</span>
              </div>
              <div className="text-[13px] text-muted-foreground">{t('oauth_excluded.description')}</div>
            </div>

            <div className="flex flex-col gap-2 px-4 py-3 pb-4 max-md:px-4">
              <div className="flex items-start justify-between gap-4 max-md:flex-col max-md:items-stretch max-md:gap-2">
                <div className="flex-1 min-w-0 flex flex-col gap-1">
                  <div className="text-[14px] font-semibold text-foreground">{t('oauth_excluded.provider_label')}</div>
                  <div className="text-[13px] text-muted-foreground">{t('oauth_excluded.provider_hint')}</div>
                </div>
                <div className="shrink-0 w-[min(360px,45%)] min-w-[220px] max-md:w-full max-md:min-w-0">
                  <AutocompleteInput
                    id="oauth-excluded-provider"
                    placeholder={t('oauth_excluded.provider_placeholder')}
                    value={provider}
                    onChange={updateProvider}
                    options={providerOptions}
                    disabled={disableControls || saving}
                    wrapperStyle={{ marginBottom: 0 }}
                  />
                </div>
              </div>

              {providerOptions.length > 0 && (
                <div className="flex flex-wrap gap-1">
                  {providerOptions.map((option) => {
                    const isActive = normalizeProviderKey(provider) === option.toLowerCase();
                    return (
                      <button
                        key={option}
                        type="button"
                        className={`inline-flex items-center px-[10px] py-1 border text-[12px] cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed ${isActive ? 'bg-primary border-primary text-white hover:bg-primary hover:border-primary hover:text-white' : 'border-border bg-muted text-muted-foreground hover:border-primary hover:text-foreground hover:bg-secondary'}`}
                        onClick={() => updateProvider(option)}
                        disabled={disableControls || saving}
                      >
                        {getTypeLabel(option)}
                      </button>
                    );
                  })}
                </div>
              )}
            </div>
          </Card>

          <Card className="p-0 overflow-visible">
            <div className="flex flex-col gap-1 px-4 py-3 border-b border-border max-md:px-4">
              <div className="inline-flex items-center gap-1 font-bold text-foreground">{t('oauth_excluded.models_label')}</div>
              {resolvedProviderKey && (
                <div className="flex items-center gap-1 text-[13px] text-muted-foreground">
                  {modelsLoading ? (
                    <>
                      <LoadingSpinner size={14} />
                      <span>{t('oauth_excluded.models_loading')}</span>
                    </>
                  ) : modelsError === 'unsupported' ? (
                    <span>{t('oauth_excluded.models_unsupported')}</span>
                  ) : modelsList.length > 0 ? (
                    <span>{t('oauth_excluded.models_loaded', { count: modelsList.length })}</span>
                  ) : (
                    <span>{t('oauth_excluded.no_models_available')}</span>
                  )}
                </div>
              )}
            </div>

            {modelsLoading ? (
              <div className="flex items-center justify-center gap-2 py-6 text-muted-foreground">
                <LoadingSpinner size={16} />
                <span>{t('common.loading')}</span>
              </div>
            ) : modelsList.length > 0 ? (
              <div className="max-h-[520px] overflow-auto px-4 py-3 pb-4 max-md:px-4">
                {modelsList.map((model) => {
                  const checked = selectedModels.has(model.id);
                  return (
                    <SelectionCheckbox
                      key={model.id}
                      checked={checked}
                      disabled={disableControls || saving}
                      onChange={(value) => toggleModel(model.id, value)}
                      className="w-full items-start py-[10px] border-b border-border last:border-b-0 hover:bg-accent"
                      labelClassName="flex flex-col gap-[2px] min-w-0 flex-1"
                      label={
                        <>
                          <span className="text-[13px] font-semibold text-foreground break-all">{model.id}</span>
                          {model.display_name && model.display_name !== model.id && (
                            <span className="text-[12px] text-muted-foreground break-all">{model.display_name}</span>
                          )}
                        </>
                      }
                    />
                  );
                })}
              </div>
            ) : resolvedProviderKey ? (
              <div className="px-4 py-6 text-muted-foreground text-[13px] text-center max-md:px-4">
                {modelsError === 'unsupported'
                  ? t('oauth_excluded.models_unsupported')
                  : t('oauth_excluded.no_models_available')}
              </div>
            ) : (
              <div className="px-4 py-6 text-muted-foreground text-[13px] text-center max-md:px-4">{t('oauth_excluded.provider_required')}</div>
            )}
          </Card>
        </>
      )}
    </SecondaryScreenShell>
  );
}
