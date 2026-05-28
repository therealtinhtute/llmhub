import { useMemo, useRef, useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  IconChevronDown,
  IconChevronUp,
  IconSlidersHorizontal,
} from '@/components/ui/icons';
import { FormSelect as Select } from '@/components/ui/FormSelect';
import { SelectionCheckbox } from '@/components/ui/SelectionCheckbox';

export type OpenAISortBy = 'name' | 'priority' | 'recent-success';
export type SortDir = 'asc' | 'desc';

interface OpenAIBrandToolbarProps {
  sortBy: OpenAISortBy;
  sortDir: SortDir;
  onSortBy: (value: OpenAISortBy) => void;
  onSortDir: (value: SortDir) => void;
  availableModels: ReadonlyArray<string>;
  selectedModels: ReadonlySet<string>;
  onSelectedModelsChange: (next: Set<string>) => void;
}

export function OpenAIBrandToolbar({
  sortBy,
  sortDir,
  onSortBy,
  onSortDir,
  availableModels,
  selectedModels,
  onSelectedModelsChange,
}: OpenAIBrandToolbarProps) {
  const { t } = useTranslation();
  const [filterOpen, setFilterOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const sortOptions = useMemo(
    () => [
      { value: 'name', label: t('providersPage.toolbar.sort.name') },
      { value: 'priority', label: t('providersPage.toolbar.sort.priority') },
      {
        value: 'recent-success',
        label: t('providersPage.toolbar.sort.recentSuccess'),
      },
    ],
    [t]
  );

  useEffect(() => {
    if (!filterOpen) return;
    const onClickOutside = (e: PointerEvent) => {
      if (
        containerRef.current &&
        !containerRef.current.contains(e.target as Node)
      ) {
        setFilterOpen(false);
      }
    };
    document.addEventListener('pointerdown', onClickOutside);
    return () => document.removeEventListener('pointerdown', onClickOutside);
  }, [filterOpen]);

  const toggleModel = (name: string) => {
    const next = new Set(selectedModels);
    if (next.has(name)) next.delete(name);
    else next.add(name);
    onSelectedModelsChange(next);
  };

  const selectAll = () => onSelectedModelsChange(new Set(availableModels));
  const clearAll = () => onSelectedModelsChange(new Set());

  const filterLabel =
    selectedModels.size === 0
      ? t('providersPage.toolbar.filter.allModels')
      : t('providersPage.toolbar.filter.selectedModels', {
          selected: selectedModels.size,
          total: availableModels.length,
        });

  return (
    <div className="flex items-center gap-2 flex-wrap">
      <div className="flex items-center gap-1.5">
        <span className="text-[11px] text-muted-foreground whitespace-nowrap">{t('providersPage.toolbar.sortBy')}</span>
        <Select
          value={sortBy}
          options={sortOptions}
          onChange={(value) => onSortBy(value as OpenAISortBy)}
          ariaLabel={t('providersPage.toolbar.sortBy')}
          size="sm"
        />
        <button
          type="button"
          className="inline-flex items-center justify-center w-7 h-7 border border-border bg-background text-muted-foreground cursor-pointer hover:bg-secondary hover:text-foreground transition-colors"
          onClick={() => onSortDir(sortDir === 'asc' ? 'desc' : 'asc')}
          aria-label={
            sortDir === 'asc'
              ? t('providersPage.toolbar.sort.directionAsc')
              : t('providersPage.toolbar.sort.directionDesc')
          }
          title={
            sortDir === 'asc'
              ? t('providersPage.toolbar.sort.directionAsc')
              : t('providersPage.toolbar.sort.directionDesc')
          }
        >
          {sortDir === 'asc' ? (
            <IconChevronUp size={14} />
          ) : (
            <IconChevronDown size={14} />
          )}
        </button>
      </div>

      <div className="relative" ref={containerRef}>
        <button
          type="button"
          className="inline-flex items-center gap-1.5 h-7 px-[10px] border border-border bg-background text-foreground text-[12px] cursor-pointer hover:border-primary hover:text-primary disabled:opacity-60 disabled:cursor-not-allowed transition-colors"
          onClick={() => setFilterOpen((v) => !v)}
          disabled={availableModels.length === 0}
        >
          <IconSlidersHorizontal size={14} />
          <span>{filterLabel}</span>
          <IconChevronDown size={12} />
        </button>
        {filterOpen ? (
          <div className="absolute top-[calc(100%+6px)] right-0 min-w-[220px] max-w-[320px] z-50 bg-background border border-border shadow-[0_8px_20px_rgba(0,0,0,0.08)] p-2 flex flex-col gap-1.5">
            <div className="flex items-center justify-end gap-1.5">
              <button
                type="button"
                className="px-2 py-0.5 border border-border bg-background text-muted-foreground text-[11px] cursor-pointer hover:border-primary hover:text-primary disabled:opacity-50 disabled:cursor-not-allowed"
                onClick={selectAll}
                disabled={availableModels.length === 0}
              >
                {t('providersPage.toolbar.filter.selectAll')}
              </button>
              <button
                type="button"
                className="px-2 py-0.5 border border-border bg-background text-muted-foreground text-[11px] cursor-pointer hover:border-primary hover:text-primary disabled:opacity-50 disabled:cursor-not-allowed"
                onClick={clearAll}
                disabled={selectedModels.size === 0}
              >
                {t('providersPage.toolbar.filter.clear')}
              </button>
            </div>
            {availableModels.length === 0 ? (
              <div className="p-3 text-center text-[12px] text-muted-foreground">
                {t('providersPage.toolbar.filter.empty')}
              </div>
            ) : (
              <ul className="list-none m-0 p-0 max-h-[220px] overflow-y-auto flex flex-col gap-0.5">
                {availableModels.map((name) => (
                  <li key={name} className="px-1.5 py-1 hover:bg-secondary">
                    <SelectionCheckbox
                      checked={selectedModels.has(name)}
                      onChange={() => toggleModel(name)}
                      label={<span className="font-mono text-[12px] break-all">{name}</span>}
                    />
                  </li>
                ))}
              </ul>
            )}
          </div>
        ) : null}
      </div>
    </div>
  );
}
