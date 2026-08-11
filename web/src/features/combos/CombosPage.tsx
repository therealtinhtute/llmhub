import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { parseDocument, type Document as YamlDocument } from 'yaml';
import { toast } from 'sonner';
import { AppCard } from '@/components/ui/AppCard';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/Button';
import { EmptyState } from '@/components/ui/EmptyState';
import { FormInput } from '@/components/ui/FormInput';
import { FormSelect } from '@/components/ui/FormSelect';
import { LoadingSpinner } from '@/components/ui/LoadingSpinner';
import { IconChevronDown, IconChevronUp, IconTrash2, IconPlus, IconCheck } from '@/components/ui/icons';
import { configFileApi } from '@/services/api/configFile';

interface ComboDraft {
  name: string;
  strategy: 'fallback' | 'round-robin';
  stickyLimit: number;
  models: string[];
}

const emptyDraft = (): ComboDraft => ({ name: '', strategy: 'fallback', stickyLimit: 1, models: [''] });

function readCombos(doc: YamlDocument): ComboDraft[] {
  // parseDocument yields YAML node wrappers, not plain JS values; toJS on the
  // document unwraps everything so the Array.isArray checks below work.
  const data = doc.toJS() as { combos?: unknown } | null;
  const items = data && Array.isArray(data.combos) ? data.combos : [];
  if (items.length === 0) return [];
  const drafts: ComboDraft[] = [];
  for (const item of items) {
    if (item === null || typeof item !== 'object' || Array.isArray(item)) continue;
    const record = item as Record<string, unknown>;
    const models = Array.isArray(record.models)
      ? (record.models as unknown[]).map((m) => String(m ?? '').trim()).filter(Boolean)
      : [];
    drafts.push({
      name: String(record.name ?? '').trim(),
      strategy: record.strategy === 'round-robin' ? 'round-robin' : 'fallback',
      stickyLimit: typeof record['sticky-limit'] === 'number' ? record['sticky-limit'] : 1,
      models,
    });
  }
  return drafts;
}

function toYamlNode(combos: ComboDraft[]): unknown {
  return combos.map((combo) => ({
    name: combo.name.trim(),
    ...(combo.strategy === 'round-robin' ? { strategy: 'round-robin' } : {}),
    ...(combo.strategy === 'round-robin' ? { 'sticky-limit': Math.max(1, combo.stickyLimit) } : {}),
    models: combo.models.map((m) => m.trim()).filter(Boolean),
  }));
}

export function CombosPage() {
  const { t } = useTranslation();
  const [combos, setCombos] = useState<ComboDraft[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const yamlContent = await configFileApi.fetchConfigYaml();
      const doc = parseDocument(yamlContent);
      setCombos(readCombos(doc));
    } catch (err) {
      toast.error(t('combos.load_error'), { description: err instanceof Error ? err.message : String(err) });
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void load();
  }, [load]);

  const validationErrors = useMemo(() => {
    const errors: string[] = [];
    const names = new Set<string>();
    for (const combo of combos) {
      const name = combo.name.trim();
      if (!name) {
        errors.push(t('combos.err_empty_name'));
        continue;
      }
      if (names.has(name)) errors.push(t('combos.err_duplicate_name', { name }));
      names.add(name);
      const models = combo.models.map((m) => m.trim()).filter(Boolean);
      if (models.length === 0) errors.push(t('combos.err_no_models', { name }));
      for (const model of models) {
        if (!model.includes('/')) errors.push(t('combos.err_model_format', { name, model }));
      }
    }
    return errors;
  }, [combos, t]);

  const updateCombo = (index: number, patch: Partial<ComboDraft>) => {
    setCombos((prev) => prev.map((combo, i) => (i === index ? { ...combo, ...patch } : combo)));
  };

  const moveCandidate = (comboIndex: number, candidateIndex: number, delta: -1 | 1) => {
    setCombos((prev) => {
      const combo = prev[comboIndex];
      if (!combo) return prev;
      const target = candidateIndex + delta;
      if (target < 0 || target >= combo.models.length) return prev;
      const models = [...combo.models];
      [models[candidateIndex], models[target]] = [models[target], models[candidateIndex]];
      const next = [...prev];
      next[comboIndex] = { ...combo, models };
      return next;
    });
  };

  const save = async () => {
    if (validationErrors.length > 0) {
      toast.error(validationErrors[0]);
      return;
    }
    setSaving(true);
    try {
      const yamlContent = await configFileApi.fetchConfigYaml();
      const doc = parseDocument(yamlContent);
      doc.setIn(['combos'], doc.createNode(toYamlNode(combos)));
      await configFileApi.saveConfigYaml(doc.toString({ indent: 2, lineWidth: 120, minContentWidth: 0 }));
      toast.success(t('combos.saved'));
    } catch (err) {
      toast.error(t('combos.save_error'), { description: err instanceof Error ? err.message : String(err) });
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <LoadingSpinner />
      </div>
    );
  }

  return (
    <div className="space-y-6 p-4 md:p-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold">{t('combos.title')}</h1>
          <p className="mt-1 text-sm text-muted-foreground">{t('combos.subtitle')}</p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={() => setCombos((prev) => [...prev, emptyDraft()])}>
            <IconPlus />
            {t('combos.add')}
          </Button>
          <Button onClick={() => void save()} disabled={saving || validationErrors.length > 0} title={validationErrors[0]}>
            {saving ? <LoadingSpinner className="size-4" /> : <IconCheck />}
            {t('combos.save')}
          </Button>
        </div>
      </div>

      {validationErrors.length > 0 && (
        <div className="rounded-md border border-destructive/40 bg-destructive/5 p-3 text-sm text-destructive">
          <ul className="list-inside list-disc space-y-1">
            {validationErrors.map((error, i) => (
              <li key={i}>{error}</li>
            ))}
          </ul>
        </div>
      )}

      {combos.length === 0 && (
        <EmptyState title={t('combos.empty_title')} description={t('combos.empty_description')} />
      )}

      {combos.map((combo, comboIndex) => (
        <AppCard key={comboIndex} className="gap-3">
          <div className="flex flex-wrap items-end gap-4">
            <FormInput
              label={t('combos.name')}
              value={combo.name}
              placeholder="daily"
              onChange={(event) => updateCombo(comboIndex, { name: event.target.value })}
            />
            <FormSelect
              value={combo.strategy}
              ariaLabel={t('combos.strategy')}
              size="sm"
              options={[
                { value: 'fallback', label: t('combos.strategy_fallback') },
                { value: 'round-robin', label: t('combos.strategy_round_robin') },
              ]}
              onChange={(value) => updateCombo(comboIndex, { strategy: value as ComboDraft['strategy'] })}
            />
            <FormInput
              label={t('combos.sticky_limit')}
              type="number"
              min={1}
              value={String(combo.stickyLimit)}
              onChange={(event) => updateCombo(comboIndex, { stickyLimit: Number(event.target.value) || 1 })}
            />
            <Badge variant="secondary">
              {combo.strategy === 'round-robin' ? 'round-robin' : 'fallback'}
            </Badge>
          </div>
          <div className="space-y-2">
            {combo.models.map((candidate, candidateIndex) => (
              <div key={candidateIndex} className="flex items-center gap-2">
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label={t('combos.move_up')}
                  disabled={candidateIndex === 0}
                  onClick={() => moveCandidate(comboIndex, candidateIndex, -1)}
                >
                  <IconChevronUp />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label={t('combos.move_down')}
                  disabled={candidateIndex === combo.models.length - 1}
                  onClick={() => moveCandidate(comboIndex, candidateIndex, 1)}
                >
                  <IconChevronDown />
                </Button>
                <FormInput
                  className="flex-1"
                  aria-label={`${t('combos.candidate')} ${candidateIndex + 1}`}
                  value={candidate}
                  placeholder="provider/model"
                  onChange={(event) =>
                    updateCombo(comboIndex, {
                      models: combo.models.map((m, i) => (i === candidateIndex ? event.target.value : m)),
                    })
                  }
                />
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label={t('combos.remove_candidate')}
                  onClick={() =>
                    updateCombo(comboIndex, { models: combo.models.filter((_, i) => i !== candidateIndex) })
                  }
                >
                  <IconTrash2 />
                </Button>
              </div>
            ))}
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => updateCombo(comboIndex, { models: [...combo.models, ''] })}
          >
            <IconPlus />
            {t('combos.add_candidate')}
          </Button>
          <div className="flex justify-end">
            <Button
              variant="ghost"
              size="sm"
              className="text-destructive"
              onClick={() => setCombos((prev) => prev.filter((_, i) => i !== comboIndex))}
            >
              <IconTrash2 />
              {t('combos.delete')}
            </Button>
          </div>
        </AppCard>
      ))}
    </div>
  );
}
