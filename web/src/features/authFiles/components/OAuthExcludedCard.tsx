import { useTranslation } from 'react-i18next';
import { AppCard as Card } from '@/components/ui/AppCard';
import { Button } from '@/components/ui/Button';
import { EmptyState } from '@/components/ui/EmptyState';

type UnsupportedError = 'unsupported' | null;

export type OAuthExcludedCardProps = {
  disableControls: boolean;
  excludedError: UnsupportedError;
  excluded: Record<string, string[]>;
  onAdd: () => void;
  onEdit: (provider: string) => void;
  onDelete: (provider: string) => void;
};

export function OAuthExcludedCard(props: OAuthExcludedCardProps) {
  const { t } = useTranslation();
  const { disableControls, excludedError, excluded, onAdd, onEdit, onDelete } = props;

  return (
    <Card
      title={t('oauth_excluded.title')}
      extra={
        <Button size="sm" onClick={onAdd} disabled={disableControls || excludedError === 'unsupported'}>
          {t('oauth_excluded.add')}
        </Button>
      }
    >
      {excludedError === 'unsupported' ? (
        <EmptyState
          title={t('oauth_excluded.upgrade_required_title')}
          description={t('oauth_excluded.upgrade_required_desc')}
        />
      ) : Object.keys(excluded).length === 0 ? (
        <EmptyState title={t('oauth_excluded.list_empty_all')} />
      ) : (
        <div className="flex flex-col gap-2">
          {Object.entries(excluded).map(([provider, models]) => (
            <div key={provider} className="flex justify-between items-center p-3 bg-muted border border-border gap-3 max-md:flex-col max-md:items-start">
              <div className="flex flex-col gap-[2px] min-w-0 flex-1">
                <div className="font-semibold text-foreground text-[14px]">{provider}</div>
                <div className="text-[12px] text-muted-foreground">
                  {models?.length
                    ? t('oauth_excluded.model_count', { count: models.length })
                    : t('oauth_excluded.no_models')}
                </div>
              </div>
              <div className="flex gap-1 shrink-0">
                <Button variant="secondary" size="sm" onClick={() => onEdit(provider)}>
                  {t('common.edit')}
                </Button>
                <Button variant="danger" size="sm" onClick={() => onDelete(provider)}>
                  {t('oauth_excluded.delete')}
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}

