import { useEffect, useId, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  IconAlertTriangle,
  IconCheckCircle2,
  IconDownload,
  IconLoader2,
  IconPlus,
  IconX,
} from '@/components/ui/icons';
import { DetailsCollapsible as Collapsible } from '@/components/ui/DetailsCollapsible';
import { FormSelect as Select } from '@/components/ui/FormSelect';
import { hasDisableAllModelsRule } from '@/components/providers/utils';
import type {
  GeminiKeyConfig,
  OpenAIProviderConfig,
  ProviderKeyConfig,
  ProviderPreset,
} from '@/types';
import type { ModelInfo } from '@/utils/models';
import { providersApi } from '@/services/api/providers';
import { PROVIDER_DESCRIPTORS } from '../../descriptors';
import type {
  ApiKeyEntryInput,
  ModelEntryInput,
  ProviderBrand,
  ProviderEntryFormInput,
  ProviderResource,
} from '../../types';
import {
  useConnectivityTest,
  type ConnectivityErrorMessages,
  type ConnectivityState,
} from './useConnectivityTest';
import { useModelDiscovery } from './useModelDiscovery';
import { ModelDiscoveryPanel } from './ModelDiscoveryPanel';

export interface BaseProviderFormHandle {
  submit: () => Promise<void>;
}

interface BaseProviderFormProps {
  brand: Exclude<ProviderBrand, 'ampcode'>;
  resource: ProviderResource | null;
  mode: 'create' | 'edit';
  mutating: boolean;
  formId: string;
  onSubmit: (input: ProviderEntryFormInput) => Promise<void>;
  onDirtyChange?: (dirty: boolean) => void;
}

const emptyHeader = () => ({ key: '', value: '' });
const emptyModel = (): ModelEntryInput => ({ name: '', alias: '' });
const emptyApiKeyEntry = (): ApiKeyEntryInput => ({
  apiKey: '',
  proxyUrl: '',
  headersText: '',
});

const headersObjectToText = (headers?: Record<string, string>): string =>
  Object.entries(headers ?? {})
    .filter(([k]) => k.trim())
    .map(([k, v]) => `${k}: ${v}`)
    .join('\n');

const stripDisableAllRule = (list?: string[]): string[] =>
  (list ?? []).filter((s) => s.trim() !== '*');

function buildInitialForm(
  brand: Exclude<ProviderBrand, 'ampcode'>,
  resource: ProviderResource | null,
  mode: 'create' | 'edit'
): ProviderEntryFormInput {
  if (mode === 'create' || !resource) {
    return {
      apiKey: '',
      name: '',
      baseUrl: '',
      proxyUrl: '',
      prefix: '',
      disabled: false,
      priority: undefined,
      weight: undefined,
      models: [emptyModel()],
      headers: [emptyHeader()],
      excludedModelsText: '',
      websockets: brand === 'codex' ? false : undefined,
      cloak:
        brand === 'claude'
          ? { mode: '', strictMode: false, sensitiveWordsText: '' }
          : undefined,
      testModel:
        brand === 'openaiCompatibility' || brand === 'claude' ? '' : undefined,
      apiKeyEntries:
        brand === 'openaiCompatibility' ? [emptyApiKeyEntry()] : undefined,
    };
  }

  const raw = resource.raw;
  if (brand === 'openaiCompatibility') {
    const cfg = raw as OpenAIProviderConfig;
    return {
      apiKey: '',
      name: cfg.name ?? '',
      baseUrl: cfg.baseUrl ?? '',
      proxyUrl: '',
      prefix: cfg.prefix ?? '',
      disabled: cfg.disabled === true,
      priority: cfg.priority,
      weight: cfg.weight,
      models: cfg.models?.length
        ? cfg.models.map((m) => ({
            name: m.name,
            alias: m.alias ?? '',
            priority: m.priority,
            testModel: m.testModel,
          }))
        : [emptyModel()],
      headers: cfg.headers
        ? Object.entries(cfg.headers).map(([k, v]) => ({ key: k, value: String(v) }))
        : [emptyHeader()],
      excludedModelsText: '',
      testModel: cfg.testModel ?? '',
      apiKeyEntries: cfg.apiKeyEntries?.length
        ? cfg.apiKeyEntries.map((entry) => ({
            apiKey: entry.apiKey,
            proxyUrl: entry.proxyUrl ?? '',
            headersText: headersObjectToText(entry.headers),
            authIndex: entry.authIndex,
          }))
        : [emptyApiKeyEntry()],
    };
  }

  const cfg = raw as GeminiKeyConfig & ProviderKeyConfig;
  const disabled = hasDisableAllModelsRule(cfg.excludedModels);
  const excludedList = stripDisableAllRule(cfg.excludedModels);
  return {
    apiKey: '',
    name: '',
    baseUrl: cfg.baseUrl ?? '',
    proxyUrl: cfg.proxyUrl ?? '',
    prefix: cfg.prefix ?? '',
    disabled,
    priority: cfg.priority,
    weight: cfg.weight,
    models: cfg.models?.length
      ? cfg.models.map((m) => ({
          name: m.name,
          alias: m.alias ?? '',
          priority: m.priority,
          testModel: m.testModel,
        }))
      : [emptyModel()],
    headers: cfg.headers
      ? Object.entries(cfg.headers).map(([k, v]) => ({ key: k, value: String(v) }))
      : [emptyHeader()],
    excludedModelsText: excludedList.join('\n'),
    websockets:
      brand === 'codex' ? (cfg as ProviderKeyConfig).websockets === true : undefined,
    cloak:
      brand === 'claude'
        ? {
            mode: (cfg as ProviderKeyConfig).cloak?.mode ?? '',
            strictMode: (cfg as ProviderKeyConfig).cloak?.strictMode === true,
            sensitiveWordsText:
              (cfg as ProviderKeyConfig).cloak?.sensitiveWords?.join('\n') ?? '',
          }
        : undefined,
    testModel: brand === 'claude' ? '' : undefined,
  };
}

function ConnectivityStatusIcon({ state }: { state: ConnectivityState }) {
  if (state === 'loading') {
    return (
      <span className="inline-flex items-center justify-center w-[14px] h-[14px] text-muted-foreground animate-spin">
        <IconLoader2 size={14} />
      </span>
    );
  }
  if (state === 'success') {
    return (
      <span className="inline-flex items-center justify-center w-[14px] h-[14px] text-emerald-600">
        <IconCheckCircle2 size={14} />
      </span>
    );
  }
  if (state === 'error') {
    return (
      <span className="inline-flex items-center justify-center w-[14px] h-[14px] text-destructive">
        <IconAlertTriangle size={14} />
      </span>
    );
  }
  return null;
}

export function BaseProviderForm({
  brand,
  resource,
  mode,
  mutating,
  formId,
  onSubmit,
  onDirtyChange,
}: BaseProviderFormProps) {
  const { t } = useTranslation();
  const descriptor = PROVIDER_DESCRIPTORS[brand];
  const fid = useId();
  const [form, setForm] = useState<ProviderEntryFormInput>(() =>
    buildInitialForm(brand, resource, mode)
  );
  const [initialFormSignature] = useState<string>(() =>
    JSON.stringify(buildInitialForm(brand, resource, mode))
  );
  const [error, setError] = useState<string | null>(null);

  const showPresetPicker = brand === 'openaiCompatibility' && mode === 'create';
  const [presets, setPresets] = useState<ProviderPreset[]>([]);
  const [selectedPresetId, setSelectedPresetId] = useState('');

  useEffect(() => {
    if (!showPresetPicker) return;
    let cancelled = false;
    void providersApi
      .getProviderPresets()
      .then((list) => {
        if (!cancelled) setPresets(list);
      })
      .catch(() => {
        if (!cancelled) setPresets([]);
      });
    return () => {
      cancelled = true;
    };
  }, [showPresetPicker]);

  const selectedPreset = useMemo(
    () => presets.find((p) => p.id === selectedPresetId) ?? null,
    [presets, selectedPresetId]
  );

  const applyPreset = (presetId: string) => {
    setSelectedPresetId(presetId);
    const preset = presets.find((p) => p.id === presetId);
    if (!preset) return;
    setForm((prev) => ({
      ...prev,
      baseUrl: preset.baseUrl,
      headers: preset.headers && Object.keys(preset.headers).length
        ? Object.entries(preset.headers).map(([key, value]) => ({ key, value }))
        : prev.headers,
    }));
  };

  const isDirty = useMemo(
    () => JSON.stringify(form) !== initialFormSignature,
    [form, initialFormSignature]
  );

  useEffect(() => {
    onDirtyChange?.(isDirty);
  }, [isDirty, onDirtyChange]);

  const fallbackApiKey = useMemo(() => {
    if (mode !== 'edit' || !resource) return '';
    if (brand === 'openaiCompatibility') return '';
    return (resource.raw as { apiKey?: string } | undefined)?.apiKey ?? '';
  }, [brand, mode, resource]);

  const fallbackAuthIndex = useMemo(() => {
    if (mode !== 'edit' || !resource) return '';
    return (
      (resource.raw as { authIndex?: string } | undefined)?.authIndex ?? ''
    );
  }, [mode, resource]);

  const connectivityMessages = useMemo<ConnectivityErrorMessages>(
    () => ({
      baseUrlRequired: t('providersPage.connectivity.baseUrlRequired'),
      endpointInvalid: t('providersPage.connectivity.endpointInvalid'),
      apiKeyRequired: t('providersPage.connectivity.apiKeyRequired'),
      modelRequired: t('providersPage.connectivity.modelRequired'),
      timeout: (seconds: number) =>
        t('providersPage.connectivity.timeout', { seconds }),
      requestFailed: t('providersPage.connectivity.requestFailed'),
    }),
    [t]
  );

  const connectivity = useConnectivityTest(
    {
      brand,
      baseUrl: form.baseUrl,
      testModel: form.testModel,
      models: form.models,
      formHeaders: form.headers,
      apiKeyEntries: form.apiKeyEntries,
      apiKey: form.apiKey,
      fallbackApiKey,
      authIndex: fallbackAuthIndex,
    },
    connectivityMessages
  );

  const discovery = useModelDiscovery({
    brand,
    baseUrl: form.baseUrl,
    formHeaders: form.headers,
    apiKeyEntries: form.apiKeyEntries,
    apiKey: form.apiKey,
    fallbackApiKey,
    authIndex: fallbackAuthIndex,
  });
  const [discoveryOpen, setDiscoveryOpen] = useState(false);

  const existingModelNames = useMemo(() => {
    const set = new Set<string>();
    form.models.forEach((m) => {
      const name = (m.name ?? '').trim();
      if (name) set.add(name);
    });
    return set;
  }, [form.models]);

  const testModelOptions = useMemo(() => {
    const seen = new Set<string>();
    const names: string[] = [];
    form.models.forEach((m) => {
      const name = (m.name ?? '').trim();
      if (!name || seen.has(name)) return;
      seen.add(name);
      names.push(name);
    });
    const firstName = names[0];
    const autoLabel = firstName
      ? t('providersPage.form.testModelAutoWith', { name: firstName })
      : t('providersPage.form.testModelAutoEmpty');
    const opts: Array<{ value: string; label: string }> = [
      { value: '', label: autoLabel },
    ];
    names.forEach((n) => opts.push({ value: n, label: n }));
    const tm = (form.testModel ?? '').trim();
    if (tm && !seen.has(tm)) {
      opts.push({
        value: tm,
        label: t('providersPage.form.testModelCustom', { name: tm }),
      });
    }
    return opts;
  }, [form.models, form.testModel, t]);

  const openDiscovery = () => {
    setDiscoveryOpen(true);
    if (!discovery.loading && !discovery.hasFetched) {
      void discovery.fetch();
    }
  };

  const closeDiscovery = () => {
    setDiscoveryOpen(false);
  };

  const applyDiscoveredModels = (incoming: ModelInfo[]) => {
    if (!incoming.length) return;
    setForm((prev) => {
      const seen = new Set<string>();
      const next: ModelEntryInput[] = [];
      prev.models.forEach((entry) => {
        const trimmed = (entry.name ?? '').trim();
        if (trimmed) {
          if (seen.has(trimmed)) return;
          seen.add(trimmed);
        }
        next.push(entry);
      });
      // If the existing list is just an empty placeholder row, drop it.
      const placeholderIdx = next.findIndex(
        (it) => !(it.name ?? '').trim() && !(it.alias ?? '').trim()
      );
      if (placeholderIdx !== -1) {
        next.splice(placeholderIdx, 1);
      }
      incoming.forEach((info) => {
        const trimmed = info.name.trim();
        if (!trimmed || seen.has(trimmed)) return;
        seen.add(trimmed);
        next.push({
          name: trimmed,
          alias: (info.alias ?? '').trim(),
        });
      });
      return { ...prev, models: next };
    });
  };

  const updateField = <K extends keyof ProviderEntryFormInput>(
    key: K,
    value: ProviderEntryFormInput[K]
  ) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  const updateCloak = <K extends keyof NonNullable<ProviderEntryFormInput['cloak']>>(
    key: K,
    value: NonNullable<ProviderEntryFormInput['cloak']>[K]
  ) => {
    setForm((prev) => ({
      ...prev,
      cloak: {
        ...(prev.cloak ?? { mode: '', strictMode: false, sensitiveWordsText: '' }),
        [key]: value,
      },
    }));
  };

  const validate = (): string | null => {
    if (descriptor.supportsName && !form.name.trim()) {
      return t('providersPage.form.validation.nameRequired');
    }
    if (
      descriptor.supportsApiKey &&
      mode === 'create' &&
      !form.apiKey.trim()
    ) {
      return t('providersPage.form.validation.apiKeyRequired');
    }
    if (descriptor.baseUrlRequired && !form.baseUrl.trim()) {
      return t('providersPage.form.validation.baseUrlRequired');
    }
    if (
      brand === 'openaiCompatibility' &&
      mode === 'create' &&
      !form.apiKeyEntries?.some((e) => e.apiKey.trim())
    ) {
      return t('providersPage.form.validation.apiKeyRequired');
    }
    return null;
  };

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const v = validate();
    if (v) {
      setError(v);
      return;
    }
    try {
      setError(null);
      await onSubmit(form);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  /* ------------------ entries helpers ------------------ */

  const headersList = useMemo(
    () => (form.headers.length ? form.headers : [emptyHeader()]),
    [form.headers]
  );
  const modelsList = useMemo(
    () => (form.models.length ? form.models : [emptyModel()]),
    [form.models]
  );
  const apiKeyEntries = useMemo(
    () =>
      form.apiKeyEntries && form.apiKeyEntries.length
        ? form.apiKeyEntries
        : [emptyApiKeyEntry()],
    [form.apiKeyEntries]
  );

  const inputCls = "w-full h-9 px-3 py-2 border border-border bg-background text-foreground text-[13px] font-[inherit] box-border placeholder:text-[var(--text-tertiary)] focus:outline-none focus:border-primary disabled:opacity-60 disabled:cursor-not-allowed";
  const textareaCls = "w-full px-3 py-2 border border-border bg-background text-foreground text-[13px] font-mono leading-[1.5] box-border resize-y min-h-[80px] placeholder:text-[var(--text-tertiary)] focus:outline-none focus:border-primary disabled:opacity-60 disabled:cursor-not-allowed";
  const removeBtnCls = "inline-flex items-center gap-1 px-2 py-1 border border-transparent bg-transparent text-destructive cursor-pointer text-[12px] hover:bg-[var(--destructive-10)] disabled:opacity-50 disabled:cursor-not-allowed";
  const addBtnCls = "inline-flex items-center gap-1.5 px-3 py-1.5 border border-dashed border-border bg-background text-muted-foreground cursor-pointer text-[12px] font-medium self-start hover:border-primary hover:text-primary";
  const connectivityBtnCls = "inline-flex items-center gap-1.5 h-7 px-3 border border-border bg-background text-foreground text-[12px] font-medium cursor-pointer hover:border-primary hover:text-primary disabled:opacity-60 disabled:cursor-not-allowed";
  const connectivityBtnGhostCls = "inline-flex items-center gap-1.5 h-6 px-2 border border-transparent bg-transparent text-muted-foreground text-[11px] font-medium cursor-pointer hover:bg-secondary hover:text-foreground disabled:opacity-60 disabled:cursor-not-allowed";

  return (
    <form id={formId} className="flex flex-col gap-4" onSubmit={handleSubmit} noValidate>
      {/* Base fields */}
      <div className="flex flex-col gap-3">
        {showPresetPicker && presets.length ? (
          <div className="grid gap-1.5">
            <label className="text-[12px] font-medium text-foreground" htmlFor={`${fid}-preset`}>
              {t('providersPage.form.presetLabel')}
            </label>
            <Select
              id={`${fid}-preset`}
              value={selectedPresetId}
              options={[
                { value: '', label: t('providersPage.form.presetNone') },
                ...presets.map((p) => ({
                  value: p.id,
                  label: p.verified
                    ? p.displayName
                    : `${p.displayName} (${t('providersPage.form.presetUnverified')})`,
                })),
              ]}
              onChange={applyPreset}
              disabled={mutating}
              ariaLabel={t('providersPage.form.presetLabel')}
            />
            {selectedPreset ? (
              <div className="flex flex-col gap-1 text-[11px] text-muted-foreground">
                {!selectedPreset.verified ? (
                  <span className="inline-flex items-center gap-1 self-start px-1.5 py-0.5 border border-[var(--destructive-30)] text-destructive font-medium">
                    {t('providersPage.form.presetUnverified')}
                  </span>
                ) : null}
                {selectedPreset.freeTierNote ? <span>{selectedPreset.freeTierNote}</span> : null}
                {selectedPreset.signupUrl ? (
                  <a
                    href={selectedPreset.signupUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="text-primary hover:underline self-start"
                  >
                    {t('providersPage.form.presetSignup')}
                  </a>
                ) : null}
              </div>
            ) : null}
          </div>
        ) : null}

        {descriptor.supportsName ? (
          <div className="grid gap-1.5">
            <label className="text-[12px] font-medium text-foreground" htmlFor={`${fid}-name`}>
              {t('providersPage.form.name')}
            </label>
            <input
              id={`${fid}-name`}
              className={inputCls}
              value={form.name}
              onChange={(e) => updateField('name', e.target.value)}
              disabled={mutating}
            />
          </div>
        ) : null}

        {descriptor.supportsApiKey ? (
          <div className="grid gap-1.5">
            <label className="text-[12px] font-medium text-foreground" htmlFor={`${fid}-apiKey`}>
              {t('providersPage.form.apiKey')}
            </label>
            <input
              id={`${fid}-apiKey`}
              className={inputCls}
              type="password"
              value={form.apiKey}
              onChange={(e) => updateField('apiKey', e.target.value)}
              placeholder={
                mode === 'edit'
                  ? t('providersPage.form.apiKeyEditPlaceholder')
                  : t('providersPage.form.apiKeyCreatePlaceholder')
              }
              disabled={mutating}
            />
          </div>
        ) : null}

        {descriptor.supportsBaseUrl ? (
          <div className="grid gap-1.5">
            <label className="text-[12px] font-medium text-foreground" htmlFor={`${fid}-baseUrl`}>
              {t('providersPage.form.baseUrl')}
              {descriptor.baseUrlRequired ? (
                <span className="text-[11px] text-muted-foreground font-normal">
                  {' '}
                  · {t('providersPage.form.baseUrlRequiredHint')}
                </span>
              ) : null}
            </label>
            <input
              id={`${fid}-baseUrl`}
              className={inputCls}
              value={form.baseUrl}
              onChange={(e) => updateField('baseUrl', e.target.value)}
              placeholder="https://api.example.com"
              disabled={mutating}
            />
          </div>
        ) : null}

        {descriptor.supportsProxyUrl ? (
          <div className="grid gap-1.5">
            <label className="text-[12px] font-medium text-foreground" htmlFor={`${fid}-proxy`}>
              {t('providersPage.form.proxyUrl')}
            </label>
            <input
              id={`${fid}-proxy`}
              className={inputCls}
              value={form.proxyUrl}
              onChange={(e) => updateField('proxyUrl', e.target.value)}
              placeholder="http://127.0.0.1:7890"
              disabled={mutating}
            />
          </div>
        ) : null}

        {descriptor.supportsPrefix ? (
          <div className="grid grid-cols-2 gap-3">
            <div className="grid gap-1.5">
              <label className="text-[12px] font-medium text-foreground" htmlFor={`${fid}-prefix`}>
                {t('providersPage.form.prefix')}
              </label>
              <input
                id={`${fid}-prefix`}
                className={inputCls}
                value={form.prefix}
                onChange={(e) => updateField('prefix', e.target.value)}
                disabled={mutating}
              />
            </div>
            {descriptor.supportsPriority ? (
              <div className="grid gap-1.5">
                <label className="text-[12px] font-medium text-foreground" htmlFor={`${fid}-prio`}>
                  {t('providersPage.form.priority')}
                </label>
                <input
                  id={`${fid}-prio`}
                  type="number"
                  className={inputCls}
                  value={form.priority ?? ''}
                  onChange={(e) =>
                    updateField(
                      'priority',
                      e.target.value === '' ? undefined : Number(e.target.value)
                    )
                  }
                  disabled={mutating}
                />
              </div>
            ) : null}
            {descriptor.supportsWeight ? (
              <div className="grid gap-1.5">
                <label className="text-[12px] font-medium text-foreground" htmlFor={`${fid}-weight`}>
                  {t('providersPage.form.weight')}
                </label>
                <input
                  id={`${fid}-weight`}
                  type="number"
                  min={1}
                  className={inputCls}
                  value={form.weight ?? ''}
                  onChange={(e) =>
                    updateField(
                      'weight',
                      e.target.value === '' ? undefined : Number(e.target.value)
                    )
                  }
                  disabled={mutating}
                />
              </div>
            ) : null}
          </div>
        ) : null}

        {descriptor.supportsTestModel ? (
          <div className="grid gap-1.5">
            <label className="text-[12px] font-medium text-foreground" htmlFor={`${fid}-testModel`}>
              {t('providersPage.form.testModel')}
              {brand === 'claude' ? (
                <span className="text-[11px] text-muted-foreground font-normal">
                  {' '}
                  · {t('providersPage.form.testModelClaudeHint')}
                </span>
              ) : null}
            </label>
            <Select
              id={`${fid}-testModel`}
              value={form.testModel ?? ''}
              options={testModelOptions}
              onChange={(value) => updateField('testModel', value)}
              disabled={mutating}
              ariaLabel={t('providersPage.form.testModel')}
            />
            {brand === 'claude' ? (
              <div className="flex items-center gap-2 mt-1">
                <button
                  type="button"
                  className={connectivityBtnCls}
                  disabled={mutating || connectivity.isTestingAny}
                  onClick={() => void connectivity.runClaude()}
                >
                  {connectivity.claudeStatus.state === 'loading' ? (
                    <span className="inline-flex items-center justify-center w-[14px] h-[14px] animate-spin text-muted-foreground">
                      <IconLoader2 size={14} />
                    </span>
                  ) : null}
                  <span>{t('providersPage.connectivity.test')}</span>
                </button>
                <ConnectivityStatusIcon state={connectivity.claudeStatus.state} />
                {connectivity.claudeStatus.state === 'success' ? (
                  <span className="text-[11px] text-primary">
                    {t('providersPage.connectivity.success')}
                  </span>
                ) : null}
              </div>
            ) : null}
            {brand === 'claude' && connectivity.claudeStatus.state === 'error' ? (
              <div className="border border-[var(--destructive-30)] bg-[var(--destructive-10)] text-destructive px-[10px] py-2 text-[11px] leading-[1.4] break-words">
                {connectivity.claudeStatus.message}
              </div>
            ) : null}
          </div>
        ) : null}

        {descriptor.supportsWebsockets ? (
          <label className="flex items-start gap-[10px] cursor-pointer select-none">
            <input
              type="checkbox"
              className="mt-0.5 w-4 h-4 border border-border bg-background cursor-pointer appearance-none relative checked:bg-primary checked:border-primary focus-visible:outline-2 focus-visible:outline-primary focus-visible:outline-offset-2"
              checked={form.websockets ?? false}
              disabled={mutating}
              onChange={(e) => updateField('websockets', e.target.checked)}
            />
            <span className="flex flex-col gap-0.5 text-[13px] text-foreground">
              <span>{t('providersPage.form.websockets')}</span>
            </span>
          </label>
        ) : null}

        {descriptor.supportsDisabled ? (
          <label className="flex items-start gap-[10px] cursor-pointer select-none">
            <input
              type="checkbox"
              className="mt-0.5 w-4 h-4 border border-border bg-background cursor-pointer appearance-none relative checked:bg-primary checked:border-primary focus-visible:outline-2 focus-visible:outline-primary focus-visible:outline-offset-2"
              checked={form.disabled}
              disabled={mutating}
              onChange={(e) => updateField('disabled', e.target.checked)}
            />
            <span className="flex flex-col gap-0.5 text-[13px] text-foreground">
              <span>{t('providersPage.form.disabled')}</span>
              <small className="text-muted-foreground text-[11px]">{t('providersPage.form.disabledHint')}</small>
            </span>
          </label>
        ) : null}
      </div>

      {/* Advanced collapsible sections */}
      {descriptor.supportsApiKeyEntries && form.apiKeyEntries ? (
        <Collapsible
          label={t('providersPage.form.apiKeyEntriesSection')}
          hint={`${apiKeyEntries.filter((e) => e.apiKey.trim()).length}`}
          defaultOpen
        >
          <div className="flex flex-col gap-[10px]">
            <div className="flex items-center justify-end gap-2 mb-0.5">
              <button
                type="button"
                className={connectivityBtnCls}
                disabled={mutating || connectivity.isTestingAny}
                onClick={() => void connectivity.runOpenAIAllKeys()}
              >
                {connectivity.isTestingAny ? (
                  <span className="inline-flex items-center justify-center w-[14px] h-[14px] animate-spin text-muted-foreground">
                    <IconLoader2 size={14} />
                  </span>
                ) : null}
                <span>{t('providersPage.connectivity.testAll')}</span>
              </button>
            </div>
            {apiKeyEntries.map((entry, idx) => {
              const status = connectivity.openaiStatuses[idx] ?? {
                state: 'idle' as ConnectivityState,
                message: '',
              };
              return (
                <div key={idx} className="border border-border p-3 flex flex-col gap-[10px] bg-muted">
                  <div className="flex items-center justify-between text-[12px] font-medium text-muted-foreground">
                    <span>
                      {t('providersPage.form.apiKeyEntry', { index: idx + 1 })}
                    </span>
                    <div className="flex items-center gap-1.5">
                      <ConnectivityStatusIcon state={status.state} />
                      <button
                        type="button"
                        className={connectivityBtnGhostCls}
                        disabled={mutating || status.state === 'loading'}
                        onClick={() => void connectivity.runOpenAIKey(idx)}
                      >
                        {status.state === 'loading' ? (
                          <span className="inline-flex items-center justify-center w-[14px] h-[14px] animate-spin text-muted-foreground">
                            <IconLoader2 size={14} />
                          </span>
                        ) : null}
                        <span>{t('providersPage.connectivity.test')}</span>
                      </button>
                      <button
                        type="button"
                        className={removeBtnCls}
                        disabled={mutating || apiKeyEntries.length <= 1}
                        onClick={() =>
                          updateField(
                            'apiKeyEntries',
                            apiKeyEntries.filter((_, i) => i !== idx)
                          )
                        }
                      >
                        <IconX size={12} />
                      </button>
                    </div>
                  </div>
                  <div className="grid gap-1.5">
                    <label className="text-[12px] font-medium text-foreground">
                      {t('providersPage.form.apiKey')}
                    </label>
                    <input
                      className={inputCls}
                      type="password"
                      value={entry.apiKey}
                      onChange={(e) =>
                        updateField(
                          'apiKeyEntries',
                          apiKeyEntries.map((it, i) =>
                            i === idx ? { ...it, apiKey: e.target.value } : it
                          )
                        )
                      }
                      disabled={mutating}
                      placeholder={t('providersPage.form.apiKeyCreatePlaceholder')}
                    />
                  </div>
                  <div className="grid gap-1.5">
                    <label className="text-[12px] font-medium text-foreground">
                      {t('providersPage.form.proxyUrl')}
                    </label>
                    <input
                      className={inputCls}
                      value={entry.proxyUrl}
                      onChange={(e) =>
                        updateField(
                          'apiKeyEntries',
                          apiKeyEntries.map((it, i) =>
                            i === idx ? { ...it, proxyUrl: e.target.value } : it
                          )
                        )
                      }
                      disabled={mutating}
                      placeholder="http://127.0.0.1:7890"
                    />
                  </div>
                  <div className="grid gap-1.5">
                    <label className="text-[12px] font-medium text-foreground">
                      {t('providersPage.form.headers')}
                      <span className="text-[11px] text-muted-foreground font-normal">
                        {' '}
                        · {t('providersPage.form.headersHint')}
                      </span>
                    </label>
                    <textarea
                      className={textareaCls}
                      value={entry.headersText}
                      rows={3}
                      onChange={(e) =>
                        updateField(
                          'apiKeyEntries',
                          apiKeyEntries.map((it, i) =>
                            i === idx ? { ...it, headersText: e.target.value } : it
                          )
                        )
                      }
                      disabled={mutating}
                      placeholder="X-Custom-Header: value"
                    />
                  </div>
                  {status.state === 'error' ? (
                    <div className="border border-[var(--destructive-30)] bg-[var(--destructive-10)] text-destructive px-[10px] py-2 text-[11px] leading-[1.4] break-words">
                      {status.message}
                    </div>
                  ) : null}
                </div>
              );
            })}
            <button
              type="button"
              className={addBtnCls}
              disabled={mutating}
              onClick={() =>
                updateField('apiKeyEntries', [...apiKeyEntries, emptyApiKeyEntry()])
              }
            >
              <IconPlus size={12} />
              <span>{t('providersPage.form.addApiKeyEntry')}</span>
            </button>
          </div>
        </Collapsible>
      ) : null}

      {descriptor.supportsHeaders ? (
        <Collapsible label={t('providersPage.form.headersSection')}>
          <div className="flex flex-col gap-[10px]">
            {headersList.map((entry, idx) => (
              <div
                key={idx}
                style={{ display: 'grid', gridTemplateColumns: '1fr 1fr auto', gap: 8 }}
              >
                <input
                  className={inputCls}
                  placeholder="X-Custom-Header"
                  value={entry.key}
                  onChange={(e) =>
                    updateField(
                      'headers',
                      headersList.map((it, i) =>
                        i === idx ? { ...it, key: e.target.value } : it
                      )
                    )
                  }
                  disabled={mutating}
                />
                <input
                  className={inputCls}
                  placeholder="value"
                  value={entry.value}
                  onChange={(e) =>
                    updateField(
                      'headers',
                      headersList.map((it, i) =>
                        i === idx ? { ...it, value: e.target.value } : it
                      )
                    )
                  }
                  disabled={mutating}
                />
                <button
                  type="button"
                  className={removeBtnCls}
                  disabled={mutating || headersList.length <= 1}
                  onClick={() =>
                    updateField(
                      'headers',
                      headersList.filter((_, i) => i !== idx)
                    )
                  }
                >
                  <IconX size={12} />
                </button>
              </div>
            ))}
            <button
              type="button"
              className={addBtnCls}
              disabled={mutating}
              onClick={() => updateField('headers', [...headersList, emptyHeader()])}
            >
              <IconPlus size={12} />
              <span>{t('providersPage.form.addHeader')}</span>
            </button>
          </div>
        </Collapsible>
      ) : null}

      {descriptor.supportsModels ? (
        <Collapsible label={t('providersPage.form.modelsSection')}>
          <div className="flex flex-col gap-[10px]">
            {discovery.available ? (
              <div className="flex items-center justify-end gap-2 mb-0.5">
                <button
                  type="button"
                  className={connectivityBtnCls}
                  onClick={openDiscovery}
                  disabled={mutating}
                >
                  <IconDownload size={14} />
                  <span>{t('providersPage.discovery.openButton')}</span>
                </button>
              </div>
            ) : null}
            {discovery.available && discoveryOpen ? (
              <ModelDiscoveryPanel
                loading={discovery.loading}
                error={discovery.error}
                models={discovery.models}
                hasFetched={discovery.hasFetched}
                existingNames={existingModelNames}
                mutating={mutating}
                onApply={(names) => {
                  applyDiscoveredModels(names);
                }}
                onReload={() => void discovery.fetch()}
                onClose={closeDiscovery}
              />
            ) : null}
            {modelsList.map((entry, idx) => (
              <div
                key={idx}
                style={{ display: 'grid', gridTemplateColumns: '1fr 1fr auto', gap: 8 }}
              >
                <input
                  className={inputCls}
                  placeholder="model-name"
                  value={entry.name}
                  onChange={(e) =>
                    updateField(
                      'models',
                      modelsList.map((it, i) =>
                        i === idx ? { ...it, name: e.target.value } : it
                      )
                    )
                  }
                  disabled={mutating}
                />
                <input
                  className={inputCls}
                  placeholder="alias (optional)"
                  value={entry.alias ?? ''}
                  onChange={(e) =>
                    updateField(
                      'models',
                      modelsList.map((it, i) =>
                        i === idx ? { ...it, alias: e.target.value } : it
                      )
                    )
                  }
                  disabled={mutating}
                />
                <button
                  type="button"
                  className={removeBtnCls}
                  disabled={mutating || modelsList.length <= 1}
                  onClick={() =>
                    updateField(
                      'models',
                      modelsList.filter((_, i) => i !== idx)
                    )
                  }
                >
                  <IconX size={12} />
                </button>
              </div>
            ))}
            <button
              type="button"
              className={addBtnCls}
              disabled={mutating}
              onClick={() => updateField('models', [...modelsList, emptyModel()])}
            >
              <IconPlus size={12} />
              <span>{t('providersPage.form.addModel')}</span>
            </button>
          </div>
        </Collapsible>
      ) : null}

      {descriptor.supportsExcludedModels ? (
        <Collapsible label={t('providersPage.form.excludedSection')}>
          <div className="grid gap-1.5">
            <span className="text-[11px] text-muted-foreground">
              {t('providersPage.form.excludedHint')}
            </span>
            <textarea
              className={textareaCls}
              rows={4}
              value={form.excludedModelsText}
              onChange={(e) => updateField('excludedModelsText', e.target.value)}
              disabled={mutating}
              placeholder="model-1&#10;model-2"
            />
          </div>
        </Collapsible>
      ) : null}

      {descriptor.supportsCloak && form.cloak ? (
        <Collapsible label={t('providersPage.form.cloakSection')}>
          <div className="flex flex-col gap-3">
            <div className="grid gap-1.5">
              <label className="text-[12px] font-medium text-foreground">
                {t('providersPage.form.cloakMode')}
              </label>
              <input
                className={inputCls}
                value={form.cloak.mode}
                onChange={(e) => updateCloak('mode', e.target.value)}
                placeholder="auto / always / never"
                disabled={mutating}
              />
            </div>
            <label className="flex items-start gap-[10px] cursor-pointer select-none">
              <input
                type="checkbox"
                className="mt-0.5 w-4 h-4 border border-border bg-background cursor-pointer appearance-none relative checked:bg-primary checked:border-primary focus-visible:outline-2 focus-visible:outline-primary focus-visible:outline-offset-2"
                checked={form.cloak.strictMode}
                disabled={mutating}
                onChange={(e) => updateCloak('strictMode', e.target.checked)}
              />
              <span className="flex flex-col gap-0.5 text-[13px] text-foreground">
                <span>{t('providersPage.form.cloakStrict')}</span>
              </span>
            </label>
            <div className="grid gap-1.5">
              <label className="text-[12px] font-medium text-foreground">
                {t('providersPage.form.cloakSensitiveWords')}
              </label>
              <textarea
                className={textareaCls}
                rows={3}
                value={form.cloak.sensitiveWordsText}
                onChange={(e) =>
                  updateCloak('sensitiveWordsText', e.target.value)
                }
                disabled={mutating}
              />
            </div>
          </div>
        </Collapsible>
      ) : null}

      {error ? <div className="border border-[var(--destructive-30)] bg-[var(--destructive-10)] text-destructive px-3 py-[10px] text-[12px] leading-[1.5]">{error}</div> : null}
    </form>
  );
}
