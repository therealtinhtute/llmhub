import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  IconLoader2,
  IconRefreshCw,
  IconSearch,
} from '@/components/ui/icons';
import { SelectionCheckbox } from '@/components/ui/SelectionCheckbox';
import type { ModelInfo } from '@/utils/models';

interface ModelDiscoveryPanelProps {
  loading: boolean;
  error: string | null;
  models: ModelInfo[];
  hasFetched: boolean;
  existingNames: Set<string>;
  mutating?: boolean;
  onApply: (picked: ModelInfo[]) => void;
  onReload: () => void;
  onClose: () => void;
}

export function ModelDiscoveryPanel({
  loading,
  error,
  models,
  hasFetched,
  existingNames,
  mutating,
  onApply,
  onReload,
  onClose,
}: ModelDiscoveryPanelProps) {
  const { t } = useTranslation();
  const [search, setSearch] = useState('');
  const [selected, setSelected] = useState<Set<string>>(new Set());

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return models;
    return models.filter((m) =>
      `${m.name} ${m.alias ?? ''}`.toLowerCase().includes(q)
    );
  }, [models, search]);

  const selectable = useMemo(
    () => filtered.filter((m) => !existingNames.has(m.name)),
    [filtered, existingNames]
  );

  const allSelectableChecked =
    selectable.length > 0 && selectable.every((m) => selected.has(m.name));

  const toggle = (name: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  };

  const toggleAll = () => {
    if (allSelectableChecked) {
      setSelected(new Set());
    } else {
      setSelected(new Set(selectable.map((m) => m.name)));
    }
  };

  const handleApply = () => {
    const picked = models.filter(
      (m) => selected.has(m.name) && !existingNames.has(m.name)
    );
    if (!picked.length) return;
    onApply(picked);
    setSelected(new Set());
  };

  const connectivityBtnCls = "inline-flex items-center gap-1.5 h-7 px-3 border border-border bg-background text-foreground text-[12px] font-medium cursor-pointer hover:border-primary hover:text-primary disabled:opacity-60 disabled:cursor-not-allowed";
  const connectivityBtnGhostCls = "inline-flex items-center gap-1.5 h-6 px-2 border border-transparent bg-transparent text-muted-foreground text-[11px] font-medium cursor-pointer hover:bg-secondary hover:text-foreground disabled:opacity-60 disabled:cursor-not-allowed";

  return (
    <div className="flex flex-col gap-[10px] p-3 border border-border bg-muted">
      <div className="flex items-center gap-2">
        <div className="relative flex-1 flex items-center">
          <span className="absolute left-[10px] inline-flex items-center text-muted-foreground pointer-events-none" aria-hidden="true">
            <IconSearch size={14} />
          </span>
          <input
            type="search"
            className="w-full h-8 py-1.5 px-[10px] pl-[30px] border border-border bg-background text-foreground text-[12px] box-border placeholder:text-[var(--text-tertiary)] focus:outline-none focus:border-primary"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t('providersPage.discovery.searchPlaceholder')}
          />
        </div>
        <button
          type="button"
          className={connectivityBtnCls}
          onClick={onReload}
          disabled={loading}
          aria-label={t('providersPage.discovery.reload')}
        >
          {loading ? (
            <span className="inline-flex items-center justify-center w-[14px] h-[14px] animate-spin text-muted-foreground">
              <IconLoader2 size={14} />
            </span>
          ) : (
            <IconRefreshCw size={14} />
          )}
          <span>{t('providersPage.discovery.reload')}</span>
        </button>
      </div>

      {loading && !models.length ? (
        <div className="p-4 text-center text-[12px] text-muted-foreground border border-dashed border-border bg-background">
          {t('providersPage.discovery.loading')}
        </div>
      ) : error ? (
        <div className="border border-[var(--destructive-30)] bg-[var(--destructive-10)] text-destructive px-[10px] py-2 text-[11px] leading-[1.4] break-words">
          {error}
        </div>
      ) : hasFetched && !models.length ? (
        <div className="p-4 text-center text-[12px] text-muted-foreground border border-dashed border-border bg-background">
          {t('providersPage.discovery.empty')}
        </div>
      ) : models.length ? (
        <>
          <div className="flex items-center justify-between px-1 py-0.5">
            <SelectionCheckbox
              checked={allSelectableChecked}
              onChange={toggleAll}
              disabled={selectable.length === 0}
              label={
                <span className="text-[12px] font-medium text-foreground">
                  {allSelectableChecked
                    ? t('providersPage.discovery.clearAll')
                    : t('providersPage.discovery.selectAll')}
                </span>
              }
            />
            <span className="text-[11px] text-muted-foreground tabular-nums">
              {t('providersPage.discovery.selectedCount', {
                selected: selected.size,
                total: selectable.length,
              })}
            </span>
          </div>
          <ul className="list-none m-0 p-0 flex flex-col gap-0.5 max-h-[240px] overflow-y-auto border border-border bg-background">
            {filtered.map((m) => {
              const existing = existingNames.has(m.name);
              return (
                <li
                  key={m.name}
                  className={[
                    'flex items-center justify-between gap-2 px-[10px] py-2 border-b border-border text-[12px] text-foreground last:border-b-0',
                    existing ? 'bg-muted text-muted-foreground' : '',
                  ].join(' ')}
                >
                  {existing ? (
                    <>
                      <span className="font-mono text-[12px] break-all">{m.name}</span>
                      <span className="text-[11px] text-muted-foreground px-2 py-0.5 border border-border bg-background">
                        {t('providersPage.discovery.alreadyAdded')}
                      </span>
                    </>
                  ) : (
                    <SelectionCheckbox
                      checked={selected.has(m.name)}
                      onChange={() => toggle(m.name)}
                      label={
                        <span className="font-mono text-[12px] break-all">{m.name}</span>
                      }
                    />
                  )}
                </li>
              );
            })}
          </ul>
        </>
      ) : (
        <div className="p-4 text-center text-[12px] text-muted-foreground border border-dashed border-border bg-background">
          {t('providersPage.discovery.notLoaded')}
        </div>
      )}

      <div className="flex items-center justify-end gap-2">
        <button
          type="button"
          className={connectivityBtnGhostCls}
          onClick={onClose}
          disabled={mutating}
        >
          {t('providersPage.discovery.close')}
        </button>
        <button
          type="button"
          className="inline-flex items-center gap-1.5 h-[30px] px-[14px] border border-transparent bg-primary text-primary-foreground text-[12px] font-medium cursor-pointer hover:bg-[var(--primary-hover)] disabled:opacity-60 disabled:cursor-not-allowed"
          onClick={handleApply}
          disabled={mutating || selected.size === 0}
        >
          {t('providersPage.discovery.apply', { count: selected.size })}
        </button>
      </div>
    </div>
  );
}
