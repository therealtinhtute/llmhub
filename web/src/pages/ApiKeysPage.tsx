import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { AppCard as Card } from '@/components/ui/AppCard';
import { Button } from '@/components/ui/Button';
import { ApiKeysManager } from '@/components/apiKeys/ApiKeysManager';
import { apiKeysApi } from '@/services/api/apiKeys';
import { useAuthStore, useConfigStore } from '@/stores';

function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message.trim()) return error.message;
  if (typeof error === 'string' && error.trim()) return error;
  return fallback;
}

export function ApiKeysPage() {
  const { t } = useTranslation();
  const connectionStatus = useAuthStore((state) => state.connectionStatus);
  const [keys, setKeys] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState('');

  const disableControls = connectionStatus !== 'connected';

  const syncConfigStore = useCallback((nextKeys: string[]) => {
    const store = useConfigStore.getState();
    store.updateConfigValue('api-keys', nextKeys);
    store.clearCache('api-keys');
    void store.fetchConfig(undefined, true).catch(() => {});
  }, []);

  const loadKeys = useCallback(
    async ({ background = false }: { background?: boolean } = {}) => {
      if (background) {
        setRefreshing(true);
      } else {
        setLoading(true);
      }
      setError('');
      try {
        const nextKeys = await apiKeysApi.list();
        setKeys(nextKeys);
        syncConfigStore(nextKeys);
        return nextKeys;
      } catch (err: unknown) {
        const message = getErrorMessage(err, t('notification.refresh_failed'));
        setError(message);
        throw err;
      } finally {
        if (background) {
          setRefreshing(false);
        } else {
          setLoading(false);
        }
      }
    },
    [syncConfigStore, t]
  );

  useEffect(() => {
    if (disableControls) {
      setLoading(false);
      return;
    }
    void loadKeys().catch(() => {});
  }, [disableControls, loadKeys]);

  const handleAdd = useCallback(
    async (value: string) => {
      try {
        const nextKeys = [...keys, value];
        await apiKeysApi.replace(nextKeys);
        await loadKeys({ background: true });
        toast.success(t('notification.saved'));
      } catch (err: unknown) {
        toast.error(getErrorMessage(err, t('notification.save_failed')));
        throw err;
      }
    },
    [keys, loadKeys, t]
  );

  const handleEdit = useCallback(
    async (index: number, value: string) => {
      try {
        await apiKeysApi.update(index, value);
        await loadKeys({ background: true });
        toast.success(t('notification.saved'));
      } catch (err: unknown) {
        toast.error(getErrorMessage(err, t('notification.save_failed')));
        throw err;
      }
    },
    [loadKeys, t]
  );

  const handleDelete = useCallback(
    async (index: number) => {
      try {
        await apiKeysApi.delete(index);
        await loadKeys({ background: true });
        toast.success(t('notification.deleted'));
      } catch (err: unknown) {
        toast.error(getErrorMessage(err, t('notification.delete_failed')));
        throw err;
      }
    },
    [loadKeys, t]
  );

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-start justify-between gap-4 max-md:flex-col">
        <div className="flex min-w-0 flex-col gap-2">
          <h1 className="m-0 text-[28px] font-bold text-foreground">{t('api_keys.title')}</h1>
          <p className="m-0 max-w-[760px] text-sm leading-6 text-muted-foreground">
            {t('api_keys.proxy_auth_title')}
          </p>
        </div>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => void loadKeys({ background: true }).catch(() => {})}
          disabled={disableControls || loading}
          loading={refreshing}
        >
          {t('common.refresh')}
        </Button>
      </div>

      <Card>
        <div className="flex flex-col gap-3">
          {disableControls ? (
            <div className="border border-warning/30 bg-warning/12 p-[10px_14px] text-sm leading-[1.5] text-warning">
              {t('notification.connection_required')}
            </div>
          ) : null}

          {error ? (
            <div className="border border-destructive/35 bg-destructive/10 p-[10px_14px] text-sm leading-[1.5] text-destructive">
              {error}
            </div>
          ) : null}

          {loading ? (
            <div className="text-[13px] leading-[1.55] text-muted-foreground">
              {t('common.loading')}
            </div>
          ) : (
            <ApiKeysManager
              keys={keys}
              disabled={disableControls || refreshing}
              onAdd={handleAdd}
              onEdit={handleEdit}
              onDelete={handleDelete}
            />
          )}
        </div>
      </Card>
    </div>
  );
}
