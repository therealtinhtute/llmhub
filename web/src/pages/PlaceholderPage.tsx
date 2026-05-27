import { useTranslation } from 'react-i18next';
import { LegacyCard as Card } from '@/components/ui/LegacyCard';

export function PlaceholderPage({ titleKey }: { titleKey: string }) {
  const { t } = useTranslation();

  return (
    <Card title={t(titleKey)}>
      <p style={{ color: 'var(--text-secondary)' }}>{t('common.loading')}</p>
    </Card>
  );
}
