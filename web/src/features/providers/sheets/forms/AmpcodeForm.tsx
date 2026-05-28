import { useEffect, useId, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { DetailsCollapsible as Collapsible } from '@/components/ui/DetailsCollapsible';
import { IconPlus, IconX } from '@/components/ui/icons';
import type {
  AmpcodeConfig,
  AmpcodeModelMapping,
  AmpcodeUpstreamApiKeyMapping,
} from '@/types';
import type { ProviderResource } from '../../types';

interface AmpcodeFormState {
  upstreamUrl: string;
  upstreamApiKey: string;
  forceModelMappings: boolean;
  upstreamMappings: Array<{ upstreamApiKey: string; clientKeysText: string }>;
  modelMappings: Array<{ from: string; to: string }>;
}

const emptyUpstream = () => ({ upstreamApiKey: '', clientKeysText: '' });
const emptyModelMapping = () => ({ from: '', to: '' });

function buildState(config?: AmpcodeConfig | null): AmpcodeFormState {
  const safe = config ?? {};
  const upstreamMappings = (safe.upstreamApiKeys ?? []).length
    ? (safe.upstreamApiKeys ?? []).map((m) => ({
        upstreamApiKey: m.upstreamApiKey ?? '',
        clientKeysText: (m.apiKeys ?? []).join('\n'),
      }))
    : [emptyUpstream()];
  const modelMappings = (safe.modelMappings ?? []).length
    ? (safe.modelMappings ?? []).map((m) => ({ from: m.from ?? '', to: m.to ?? '' }))
    : [emptyModelMapping()];
  return {
    upstreamUrl: safe.upstreamUrl ?? '',
    upstreamApiKey: safe.upstreamApiKey ?? '',
    forceModelMappings: safe.forceModelMappings === true,
    upstreamMappings,
    modelMappings,
  };
}

const parseClientKeys = (text: string): string[] =>
  text
    .split(/[\n,]+/)
    .map((s) => s.trim())
    .filter(Boolean);

interface AmpcodeFormProps {
  resource: ProviderResource | null;
  mutating: boolean;
  formId: string;
  onSubmit: (config: AmpcodeConfig) => Promise<void>;
  onDirtyChange?: (dirty: boolean) => void;
}

export function AmpcodeForm({
  resource,
  mutating,
  formId,
  onSubmit,
  onDirtyChange,
}: AmpcodeFormProps) {
  const { t } = useTranslation();
  const fid = useId();
  const initialConfig = (resource?.raw as AmpcodeConfig | undefined) ?? {};
  const [form, setForm] = useState<AmpcodeFormState>(() => buildState(initialConfig));
  const [initialFormSignature] = useState<string>(() =>
    JSON.stringify(buildState(initialConfig))
  );
  const [error, setError] = useState<string | null>(null);

  const isDirty = useMemo(
    () => JSON.stringify(form) !== initialFormSignature,
    [form, initialFormSignature]
  );

  useEffect(() => {
    onDirtyChange?.(isDirty);
  }, [isDirty, onDirtyChange]);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    try {
      setError(null);
      const upstreamApiKeys: AmpcodeUpstreamApiKeyMapping[] = [];
      const seen = new Set<string>();
      form.upstreamMappings.forEach((m) => {
        const key = m.upstreamApiKey.trim();
        if (!key || seen.has(key)) return;
        const clientKeys = parseClientKeys(m.clientKeysText);
        if (!clientKeys.length) return;
        seen.add(key);
        upstreamApiKeys.push({ upstreamApiKey: key, apiKeys: clientKeys });
      });

      const modelMappings: AmpcodeModelMapping[] = [];
      const seenFrom = new Set<string>();
      form.modelMappings.forEach((m) => {
        const from = m.from.trim();
        const to = m.to.trim();
        if (!from || !to) return;
        const id = from.toLowerCase();
        if (seenFrom.has(id)) return;
        seenFrom.add(id);
        modelMappings.push({ from, to });
      });

      const next: AmpcodeConfig = {
        upstreamUrl: form.upstreamUrl.trim() || undefined,
        upstreamApiKey: form.upstreamApiKey.trim() || undefined,
        upstreamApiKeys: upstreamApiKeys.length ? upstreamApiKeys : undefined,
        modelMappings: modelMappings.length ? modelMappings : undefined,
        forceModelMappings: form.forceModelMappings,
      };
      await onSubmit(next);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  const inputCls = "w-full h-9 px-3 py-2 border border-border bg-background text-foreground text-[13px] font-[inherit] box-border placeholder:text-[var(--text-tertiary)] focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_var(--primary-10)] disabled:opacity-60 disabled:cursor-not-allowed";
  const textareaCls = "w-full px-3 py-2 border border-border bg-background text-foreground text-[13px] font-mono leading-[1.5] box-border resize-y min-h-[80px] placeholder:text-[var(--text-tertiary)] focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_var(--primary-10)] disabled:opacity-60 disabled:cursor-not-allowed";
  const removeBtnCls = "inline-flex items-center gap-1 px-2 py-1 border border-transparent bg-transparent text-destructive cursor-pointer text-[12px] hover:bg-[var(--destructive-10)] disabled:opacity-50 disabled:cursor-not-allowed";
  const addBtnCls = "inline-flex items-center gap-1.5 px-3 py-1.5 border border-dashed border-border bg-background text-muted-foreground cursor-pointer text-[12px] font-medium self-start hover:border-primary hover:text-primary";

  return (
    <form id={formId} className="flex flex-col gap-4" onSubmit={handleSubmit} noValidate>
      <div className="flex flex-col gap-3">
        <div className="grid gap-1.5">
          <label className="text-[12px] font-medium text-foreground" htmlFor={`${fid}-url`}>
            {t('providersPage.ampcode.upstreamUrl')}
          </label>
          <input
            id={`${fid}-url`}
            className={inputCls}
            value={form.upstreamUrl}
            onChange={(e) => setForm((s) => ({ ...s, upstreamUrl: e.target.value }))}
            placeholder="https://api.ampcode.com"
            disabled={mutating}
          />
        </div>
        <div className="grid gap-1.5">
          <label className="text-[12px] font-medium text-foreground" htmlFor={`${fid}-key`}>
            {t('providersPage.ampcode.upstreamApiKey')}
            <span className="text-[11px] text-muted-foreground font-normal">
              {' '}
              · {t('providersPage.ampcode.upstreamApiKeyHint')}
            </span>
          </label>
          <input
            id={`${fid}-key`}
            className={inputCls}
            type="password"
            value={form.upstreamApiKey}
            onChange={(e) =>
              setForm((s) => ({ ...s, upstreamApiKey: e.target.value }))
            }
            disabled={mutating}
          />
        </div>
        <label className="flex items-start gap-[10px] cursor-pointer select-none">
          <input
            type="checkbox"
            className="mt-0.5 w-4 h-4 border border-border bg-background cursor-pointer appearance-none relative checked:bg-primary checked:border-primary focus-visible:outline-2 focus-visible:outline-primary focus-visible:outline-offset-2"
            checked={form.forceModelMappings}
            disabled={mutating}
            onChange={(e) =>
              setForm((s) => ({ ...s, forceModelMappings: e.target.checked }))
            }
          />
          <span className="flex flex-col gap-0.5 text-[13px] text-foreground">
            <span>{t('providersPage.ampcode.forceModelMappings')}</span>
            <small className="text-muted-foreground text-[11px]">{t('providersPage.ampcode.forceModelMappingsHint')}</small>
          </span>
        </label>
      </div>

      <Collapsible label={t('providersPage.ampcode.keyMappingsSection')} defaultOpen>
        <div className="flex flex-col gap-[10px]">
          {form.upstreamMappings.map((m, idx) => (
            <div key={idx} className="border border-border p-3 flex flex-col gap-[10px] bg-muted">
              <div className="flex items-center justify-between text-[12px] font-medium text-muted-foreground">
                <span>{t('providersPage.ampcode.mappingRow', { index: idx + 1 })}</span>
                <button
                  type="button"
                  className={removeBtnCls}
                  disabled={mutating || form.upstreamMappings.length <= 1}
                  onClick={() =>
                    setForm((s) => ({
                      ...s,
                      upstreamMappings: s.upstreamMappings.filter((_, i) => i !== idx),
                    }))
                  }
                >
                  <IconX size={12} />
                </button>
              </div>
              <div className="grid gap-1.5">
                <label className="text-[12px] font-medium text-foreground">
                  {t('providersPage.ampcode.upstreamApiKey')}
                </label>
                <input
                  className={inputCls}
                  value={m.upstreamApiKey}
                  onChange={(e) =>
                    setForm((s) => ({
                      ...s,
                      upstreamMappings: s.upstreamMappings.map((it, i) =>
                        i === idx ? { ...it, upstreamApiKey: e.target.value } : it
                      ),
                    }))
                  }
                  disabled={mutating}
                />
              </div>
              <div className="grid gap-1.5">
                <label className="text-[12px] font-medium text-foreground">
                  {t('providersPage.ampcode.clientKeys')}
                  <span className="text-[11px] text-muted-foreground font-normal">
                    {' '}
                    · {t('providersPage.ampcode.clientKeysHint')}
                  </span>
                </label>
                <textarea
                  className={textareaCls}
                  rows={3}
                  value={m.clientKeysText}
                  onChange={(e) =>
                    setForm((s) => ({
                      ...s,
                      upstreamMappings: s.upstreamMappings.map((it, i) =>
                        i === idx ? { ...it, clientKeysText: e.target.value } : it
                      ),
                    }))
                  }
                  disabled={mutating}
                />
              </div>
            </div>
          ))}
          <button
            type="button"
            className={addBtnCls}
            disabled={mutating}
            onClick={() =>
              setForm((s) => ({
                ...s,
                upstreamMappings: [...s.upstreamMappings, emptyUpstream()],
              }))
            }
          >
            <IconPlus size={12} />
            <span>{t('providersPage.ampcode.addMapping')}</span>
          </button>
        </div>
      </Collapsible>

      <Collapsible label={t('providersPage.ampcode.modelMappingsSection')}>
        <div className="flex flex-col gap-[10px]">
          {form.modelMappings.map((m, idx) => (
            <div
              key={idx}
              style={{ display: 'grid', gridTemplateColumns: '1fr 1fr auto', gap: 8 }}
            >
              <input
                className={inputCls}
                placeholder="from"
                value={m.from}
                onChange={(e) =>
                  setForm((s) => ({
                    ...s,
                    modelMappings: s.modelMappings.map((it, i) =>
                      i === idx ? { ...it, from: e.target.value } : it
                    ),
                  }))
                }
                disabled={mutating}
              />
              <input
                className={inputCls}
                placeholder="to"
                value={m.to}
                onChange={(e) =>
                  setForm((s) => ({
                    ...s,
                    modelMappings: s.modelMappings.map((it, i) =>
                      i === idx ? { ...it, to: e.target.value } : it
                    ),
                  }))
                }
                disabled={mutating}
              />
              <button
                type="button"
                className={removeBtnCls}
                disabled={mutating || form.modelMappings.length <= 1}
                onClick={() =>
                  setForm((s) => ({
                    ...s,
                    modelMappings: s.modelMappings.filter((_, i) => i !== idx),
                  }))
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
            onClick={() =>
              setForm((s) => ({
                ...s,
                modelMappings: [...s.modelMappings, emptyModelMapping()],
              }))
            }
          >
            <IconPlus size={12} />
            <span>{t('providersPage.ampcode.addModelMapping')}</span>
          </button>
        </div>
      </Collapsible>

      {error ? <div className="border border-[var(--destructive-30)] bg-[var(--destructive-10)] text-destructive px-3 py-[10px] text-[12px] leading-[1.5]">{error}</div> : null}
    </form>
  );
}
