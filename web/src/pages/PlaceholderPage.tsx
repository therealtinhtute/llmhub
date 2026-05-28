import { useTranslation } from 'react-i18next';
import { AppCard as Card } from '@/components/ui/AppCard';

export function PlaceholderPage({ titleKey }: { titleKey: string }) {
  const { t } = useTranslation();

  return (
    <Card title={t(titleKey)}>
      <p className="text-muted-foreground">{t('common.loading')}</p>
    </Card>
  );
}
