import { useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { AppCard as Card } from '@/components/ui/AppCard';
import { Button } from '@/components/ui/Button';
import { EmptyState } from '@/components/ui/EmptyState';
import { ModelMappingDiagram, type ModelMappingDiagramRef } from '@/components/modelAlias';
import { IconChevronUp } from '@/components/ui/icons';
import type { OAuthModelAliasEntry } from '@/types';
import type { AuthFileModelItem } from '@/features/authFiles/constants';

type UnsupportedError = 'unsupported' | null;
type ViewMode = 'diagram' | 'list';

export type OAuthModelAliasCardProps = {
  disableControls: boolean;
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
  onAdd: () => void;
  onEditProvider: (provider?: string) => void;
  onDeleteProvider: (provider: string) => void;
  modelAliasError: UnsupportedError;
  modelAlias: Record<string, OAuthModelAliasEntry[]>;
  allProviderModels: Record<string, AuthFileModelItem[]>;
  onUpdate: (provider: string, sourceModel: string, newAlias: string) => Promise<void>;
  onDeleteLink: (provider: string, sourceModel: string, alias: string) => void;
  onToggleFork: (provider: string, sourceModel: string, alias: string, fork: boolean) => Promise<void>;
  onRenameAlias: (oldAlias: string, newAlias: string) => Promise<void>;
  onDeleteAlias: (aliasName: string) => void;
};

export function OAuthModelAliasCard(props: OAuthModelAliasCardProps) {
  const { t } = useTranslation();
  const diagramRef = useRef<ModelMappingDiagramRef | null>(null);
  const {
    disableControls,
    viewMode,
    onViewModeChange,
    onAdd,
    onEditProvider,
    onDeleteProvider,
    modelAliasError,
    modelAlias,
    allProviderModels,
    onUpdate,
    onDeleteLink,
    onToggleFork,
    onRenameAlias,
    onDeleteAlias
  } = props;

  return (
    <Card
      title={t('oauth_model_alias.title')}
      extra={
        <div className="flex gap-2 items-center flex-wrap min-w-0 max-w-full max-md:w-full">
          <div className="inline-flex items-center gap-1 flex-wrap min-w-0 max-w-full p-[2px] bg-muted border border-border max-md:w-full max-md:justify-stretch">
            <Button
              variant={viewMode === 'list' ? 'secondary' : 'ghost'}
              size="sm"
              onClick={() => onViewModeChange('list')}
              disabled={disableControls || modelAliasError === 'unsupported'}
            >
              {t('oauth_model_alias.view_mode_list')}
            </Button>
            <Button
              variant={viewMode === 'diagram' ? 'secondary' : 'ghost'}
              size="sm"
              onClick={() => onViewModeChange('diagram')}
              disabled={disableControls || modelAliasError === 'unsupported'}
            >
              {t('oauth_model_alias.view_mode_diagram')}
            </Button>
          </div>
          <Button
            size="sm"
            onClick={onAdd}
            disabled={disableControls || modelAliasError === 'unsupported'}
          >
            {t('oauth_model_alias.add')}
          </Button>
        </div>
      }
    >
      {modelAliasError === 'unsupported' ? (
        <EmptyState
          title={t('oauth_model_alias.upgrade_required_title')}
          description={t('oauth_model_alias.upgrade_required_desc')}
        />
      ) : viewMode === 'diagram' ? (
        Object.keys(modelAlias).length === 0 ? (
          <EmptyState title={t('oauth_model_alias.list_empty_all')} />
        ) : (
          <div className="mb-4 pb-4 border-b border-border">
            <div className="flex items-center justify-between gap-2 mb-2">
              <h4 className="m-0 text-[14px] font-semibold text-muted-foreground">{t('oauth_model_alias.chart_title')}</h4>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => diagramRef.current?.collapseAll()}
                disabled={disableControls || modelAliasError === 'unsupported'}
                title={t('oauth_model_alias.diagram_collapse')}
                aria-label={t('oauth_model_alias.diagram_collapse')}
              >
                <IconChevronUp size={16} />
              </Button>
            </div>
            <ModelMappingDiagram
              ref={diagramRef}
              modelAlias={modelAlias}
              allProviderModels={allProviderModels}
              onUpdate={onUpdate}
              onDeleteLink={onDeleteLink}
              onToggleFork={onToggleFork}
              onRenameAlias={onRenameAlias}
              onDeleteAlias={onDeleteAlias}
              onEditProvider={onEditProvider}
              onDeleteProvider={onDeleteProvider}
              className="w-full min-h-[120px]"
            />
          </div>
        )
      ) : Object.keys(modelAlias).length === 0 ? (
        <EmptyState title={t('oauth_model_alias.list_empty_all')} />
      ) : (
        <div className="flex flex-col gap-2">
          {Object.entries(modelAlias).map(([provider, mappings]) => (
            <div key={provider} className="flex justify-between items-center p-3 bg-muted border border-border gap-3 max-md:flex-col max-md:items-start">
              <div className="flex flex-col gap-[2px] min-w-0 flex-1">
                <div className="font-semibold text-foreground text-[14px]">{provider}</div>
                <div className="text-[12px] text-muted-foreground">
                  {mappings?.length
                    ? t('oauth_model_alias.model_count', { count: mappings.length })
                    : t('oauth_model_alias.no_models')}
                </div>
              </div>
              <div className="flex gap-1 shrink-0">
                <Button variant="secondary" size="sm" onClick={() => onEditProvider(provider)}>
                  {t('common.edit')}
                </Button>
                <Button variant="danger" size="sm" onClick={() => onDeleteProvider(provider)}>
                  {t('oauth_model_alias.delete')}
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}

