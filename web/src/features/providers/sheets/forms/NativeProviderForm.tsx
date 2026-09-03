import { useEffect, useId, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  IconAlertTriangle,
  IconDownload,
  IconLoader2,
  IconPlus,
  IconX,
} from '@/components/ui/icons';
import { providersApi } from '@/services/api';
import type {
  NativeProviderBrand,
  NativeProviderFormInput,
  NativeProviderModelInput,
  NativeProviderResource,
  ProviderResource,
} from '../../types';

export interface NativeProviderFormHandle {
  submit: () => Promise<void>;
}

interface NativeProviderFormProps {
  brand: NativeProviderBrand;
  resource: ProviderResource | null;
  mode: 'create' | 'edit';
  mutating: boolean;
  formId: string;
  onSubmit: (input: NativeProviderFormInput) => Promise<void>;
  onDirtyChange?: (dirty: boolean) => void;
}

const emptyModel = (): NativeProviderModelInput => ({ name: '', alias: '' });

const normalizeModels = (models: NativeProviderModelInput[]): NativeProviderModelInput[] => {
  const seen = new Set<string>();
  return models
    .map((model) => ({
      name: model.name.trim(),
      alias: model.alias?.trim() || undefined,
      displayName: model.displayName?.trim() || undefined,
    }))
    .filter((model) => {
      if (!model.name) return false;
      const key = (model.alias || model.name).toLowerCase();
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    });
};

export function NativeProviderForm({
  brand,
  resource,
  mode,
  mutating,
  formId,
  onSubmit,
  onDirtyChange,
}: NativeProviderFormProps) {
  const { t } = useTranslation();
  const fid = useId();
  const stored = resource?.raw as NativeProviderResource | undefined;
  const initial = useMemo(
    () => ({
      apiKey: '',
      apiKeyTouched: false,
      enabled: mode === 'create' ? true : !resource?.disabled,
      useApiKey: brand === 'openrouter' || stored?.apiKeyPresent === true,
      models: stored?.models ?? [],
    }),
    [brand, mode, resource?.disabled, stored?.apiKeyPresent, stored?.models]
  );
  const [apiKey, setApiKey] = useState(initial.apiKey);
  const [apiKeyTouched, setApiKeyTouched] = useState(initial.apiKeyTouched);
  const [enabled, setEnabled] = useState(initial.enabled);
  const [useApiKey, setUseApiKey] = useState(initial.useApiKey);
  const [models, setModels] = useState<NativeProviderModelInput[]>(initial.models);
  const [initialSignature] = useState(() => JSON.stringify(initial));
  const [loadingModels, setLoadingModels] = useState(false);
  const [modelSource, setModelSource] = useState<'remote' | 'fallback' | null>(null);
  const [modelError, setModelError] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const formSignature = JSON.stringify({ apiKey, apiKeyTouched, enabled, useApiKey, models });
  const isDirty = formSignature !== initialSignature;

  useEffect(() => {
    onDirtyChange?.(isDirty);
  }, [isDirty, onDirtyChange]);

  const inputCls =
    'w-full h-9 px-3 py-2 border border-border bg-background text-foreground text-[13px] font-[inherit] box-border placeholder:text-[var(--text-tertiary)] focus:outline-none focus:border-primary disabled:opacity-60 disabled:cursor-not-allowed';
  const addBtnCls =
    'inline-flex items-center gap-1.5 px-3 py-1.5 border border-dashed border-border bg-background text-muted-foreground cursor-pointer text-[12px] font-medium self-start hover:border-primary hover:text-primary disabled:opacity-60 disabled:cursor-not-allowed';
  const removeBtnCls =
    'inline-flex items-center gap-1 px-2 py-1 border border-transparent bg-transparent text-destructive cursor-pointer text-[12px] hover:bg-[var(--destructive-10)] disabled:opacity-50 disabled:cursor-not-allowed';
  const actionBtnCls =
    'inline-flex items-center gap-1.5 h-7 px-3 border border-border bg-background text-foreground text-[12px] font-medium cursor-pointer hover:border-primary hover:text-primary disabled:opacity-60 disabled:cursor-not-allowed';

  const providerName = t(`providersPage.providerNames.${brand}`);
  const endpoint =
    brand === 'openrouter'
      ? 'https://openrouter.ai/api/v1'
      : 'https://opencode.ai/zen/v1';
  const keyPreview = resource?.apiKeyPreview;

  const updateModels = (next: NativeProviderModelInput[]) => {
    setModels(next);
    setModelError(null);
  };

  const loadModels = async () => {
    if (!resource?.id) {
      setModelError(t('providersPage.native.modelsSaveFirst'));
      return;
    }
    setLoadingModels(true);
    setModelError(null);
    try {
      const response = await providersApi.getNativeProviderModels(brand, resource.id);
      updateModels(
        response.models.map((model) => ({
          name: model.name,
          alias: model.alias,
          displayName: model.display_name,
        }))
      );
      setModelSource(response.source);
      if (response.error) setModelError(response.error);
    } catch (err) {
      setModelError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoadingModels(false);
    }
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const normalizedKey = apiKey.trim();
    if (
      (brand === 'openrouter' && !normalizedKey && (mode === 'create' || apiKeyTouched)) ||
      (brand === 'opencode' && useApiKey && !normalizedKey && (mode === 'create' || apiKeyTouched))
    ) {
      setError(t('providersPage.native.apiKeyRequired'));
      return;
    }
    try {
      setError(null);
      await onSubmit({
        apiKey: useApiKey ? normalizedKey : '',
        apiKeyTouched,
        enabled,
        models: normalizeModels(models),
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  return (
    <form id={formId} className="flex flex-col gap-4" onSubmit={handleSubmit} noValidate>
      <div className="flex flex-col gap-3 border border-border bg-muted p-3">
        <div className="flex flex-col gap-1">
          <span className="text-[12px] font-medium text-foreground">
            {t('providersPage.native.endpoint')}
          </span>
          <span className="font-mono text-[12px] text-muted-foreground break-all">{endpoint}</span>
          <span className="text-[11px] text-muted-foreground">
            {t('providersPage.native.endpointHint', { provider: providerName })}
          </span>
        </div>

        {brand === 'openrouter' ? (
          <div className="grid gap-1.5">
            <label className="text-[12px] font-medium text-foreground" htmlFor={`${fid}-apiKey`}>
              {t('providersPage.form.apiKey')}
            </label>
            <input
              id={`${fid}-apiKey`}
              className={inputCls}
              type="password"
              value={apiKey}
              onChange={(event) => {
                setApiKey(event.target.value);
                setApiKeyTouched(true);
              }}
              placeholder={keyPreview || t('providersPage.form.apiKeyCreatePlaceholder')}
              disabled={mutating}
              autoComplete="new-password"
            />
            {mode === 'edit' && keyPreview ? (
              <span className="text-[11px] text-muted-foreground">
                {t('providersPage.native.keyStored', { preview: keyPreview })}
              </span>
            ) : null}
          </div>
        ) : (
          <div className="flex flex-col gap-2">
            <label className="flex items-start gap-[10px] cursor-pointer select-none">
              <input
                type="checkbox"
                className="mt-0.5 w-4 h-4 border border-border bg-background cursor-pointer appearance-none relative checked:bg-primary checked:border-primary focus-visible:outline-2 focus-visible:outline-primary focus-visible:outline-offset-2"
                checked={useApiKey}
                disabled={mutating}
                onChange={(event) => {
                  const next = event.target.checked;
                  setUseApiKey(next);
                  if (!next) {
                    setApiKey('');
                    setApiKeyTouched(true);
                  }
                }}
              />
              <span className="flex flex-col gap-0.5 text-[13px] text-foreground">
                <span>{t('providersPage.native.useApiKey')}</span>
                <small className="text-muted-foreground text-[11px]">
                  {t('providersPage.native.zeroKeyHint')}
                </small>
              </span>
            </label>
            {useApiKey ? (
              <input
                id={`${fid}-apiKey`}
                className={inputCls}
                type="password"
                value={apiKey}
                onChange={(event) => {
                  setApiKey(event.target.value);
                  setApiKeyTouched(true);
                }}
                placeholder={keyPreview || t('providersPage.form.apiKeyEditPlaceholder')}
                disabled={mutating}
                autoComplete="new-password"
              />
            ) : (
              <div className="flex items-center gap-2 border border-success/30 bg-success/12 px-3 py-2 text-[11px] text-success">
                {t('providersPage.native.zeroKeyEnabled')}
              </div>
            )}
            {mode === 'edit' && keyPreview && useApiKey ? (
              <span className="text-[11px] text-muted-foreground">
                {t('providersPage.native.keyStored', { preview: keyPreview })}
              </span>
            ) : null}
          </div>
        )}

        <label className="flex items-start gap-[10px] cursor-pointer select-none">
          <input
            type="checkbox"
            className="mt-0.5 w-4 h-4 border border-border bg-background cursor-pointer appearance-none relative checked:bg-primary checked:border-primary focus-visible:outline-2 focus-visible:outline-primary focus-visible:outline-offset-2"
            checked={enabled}
            disabled={mutating}
            onChange={(event) => setEnabled(event.target.checked)}
          />
          <span className="flex flex-col gap-0.5 text-[13px] text-foreground">
            <span>{t('providersPage.native.enabled')}</span>
            <small className="text-muted-foreground text-[11px]">
              {t('providersPage.form.disabledHint')}
            </small>
          </span>
        </label>
      </div>

      <div className="flex flex-col gap-[10px] border border-border bg-background p-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div className="flex flex-col gap-0.5">
            <span className="text-[12px] font-semibold text-foreground">
              {t('providersPage.native.modelsTitle')}
            </span>
            <span className="text-[11px] text-muted-foreground">
              {t('providersPage.native.modelsHint')}
            </span>
          </div>
          <button
            type="button"
            className={actionBtnCls}
            onClick={() => void loadModels()}
            disabled={mutating || loadingModels || mode === 'create'}
          >
            {loadingModels ? <IconLoader2 size={14} className="animate-spin" /> : <IconDownload size={14} />}
            {t('providersPage.native.loadModels')}
          </button>
        </div>

        {modelSource ? (
          <span className="text-[11px] text-muted-foreground">
            {t('providersPage.native.modelsSource', { source: modelSource })}
          </span>
        ) : null}
        {modelError ? (
          <div className="flex items-start gap-1.5 border border-[var(--destructive-30)] bg-[var(--destructive-10)] px-[10px] py-2 text-[11px] leading-[1.4] text-destructive break-words">
            <IconAlertTriangle size={13} className="mt-0.5 shrink-0" />
            <span>{modelError}</span>
          </div>
        ) : null}

        {models.map((model, index) => (
          <div key={`${model.name}-${index}`} className="grid grid-cols-[1fr_1fr_auto] gap-2">
            <input
              className={inputCls}
              value={model.name}
              placeholder="model-name"
              disabled={mutating}
              onChange={(event) =>
                updateModels(
                  models.map((item, itemIndex) =>
                    itemIndex === index ? { ...item, name: event.target.value } : item
                  )
                )
              }
            />
            <input
              className={inputCls}
              value={model.alias ?? ''}
              placeholder="alias (optional)"
              disabled={mutating}
              onChange={(event) =>
                updateModels(
                  models.map((item, itemIndex) =>
                    itemIndex === index ? { ...item, alias: event.target.value } : item
                  )
                )
              }
            />
            <button
              type="button"
              className={removeBtnCls}
              disabled={mutating}
              onClick={() => updateModels(models.filter((_, itemIndex) => itemIndex !== index))}
              aria-label={t('providersPage.native.removeModel')}
            >
              <IconX size={12} />
            </button>
          </div>
        ))}
        <button
          type="button"
          className={addBtnCls}
          disabled={mutating}
          onClick={() => updateModels([...models, emptyModel()])}
        >
          <IconPlus size={12} />
          <span>{t('providersPage.form.addModel')}</span>
        </button>
      </div>

      {mode === 'create' ? (
        <p className="m-0 text-[11px] text-muted-foreground">
          {t('providersPage.native.modelsCreateHint')}
        </p>
      ) : null}
      {error ? (
        <div className="border border-[var(--destructive-30)] bg-[var(--destructive-10)] px-3 py-[10px] text-[12px] leading-[1.5] text-destructive">
          {error}
        </div>
      ) : null}
    </form>
  );
}
