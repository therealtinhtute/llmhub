import { useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/Button';
import { ToggleSwitch } from '@/components/ui/ToggleSwitch';
import { LoadingSpinner } from '@/components/ui/LoadingSpinner';
import { IconDownload, IconTrash2 } from '@/components/ui/icons';
import type { OAuthProvider } from '@/services/api/oauth';
import {
  getAuthFileStatusMessage,
  hasAuthFileHealthIssue,
  isAuthFileUnavailable,
} from '@/features/authFiles/constants';
import { useAuthFilesData } from '@/features/authFiles/hooks/useAuthFilesData';
import { getOAuthAuthFileTypes } from '../entries';

interface AuthFileMiniTableProps {
  oauthId: OAuthProvider;
  refreshRevision: number;
  onFilesChanged?: () => void | Promise<void>;
}

const formatLastRefresh = (value: string | number | undefined, locale: string): string => {
  if (value === undefined || value === null || value === '') return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return String(value);
  return new Intl.DateTimeFormat(locale, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date);
};

export function AuthFileMiniTable({
  oauthId,
  refreshRevision,
  onFilesChanged,
}: AuthFileMiniTableProps) {
  const { t, i18n } = useTranslation();
  const {
    files,
    loading,
    error,
    loadFiles,
    deleting,
    statusUpdating,
    handleStatusToggle,
    handleDownload,
    handleDelete,
  } = useAuthFilesData(onFilesChanged);

  useEffect(() => {
    void loadFiles();
  }, [loadFiles, oauthId, refreshRevision]);

  const items = useMemo(() => {
    const fileTypes = getOAuthAuthFileTypes(oauthId);
    return files.filter((file) => fileTypes.includes(String(file.type ?? '')));
  }, [files, oauthId]);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-6">
        <LoadingSpinner size={18} />
      </div>
    );
  }

  if (error && items.length === 0) {
    return (
      <p role="alert" className="border border-[var(--destructive-30)] bg-[var(--destructive-10)] px-3 py-2 text-sm text-destructive">
        {error}
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      {error ? (
        <p role="alert" className="border border-[var(--destructive-30)] bg-[var(--destructive-10)] px-3 py-2 text-sm text-destructive">
          {error}
        </p>
      ) : null}
      {items.length === 0 ? (
        <p className="py-4 text-sm text-muted-foreground">
          {t('providersPage.oauthAccounts.empty')}
        </p>
      ) : (
        <div className="overflow-x-auto border border-border">
      <div className="min-w-[560px]">
        <div className="grid grid-cols-[minmax(180px,1.5fr)_minmax(100px,0.8fr)_minmax(160px,1fr)_auto] gap-3 border-b border-border bg-muted px-3 py-2 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
          <span>{t('providersPage.oauthAccounts.columns.name')}</span>
          <span>{t('providersPage.oauthAccounts.columns.status')}</span>
          <span>{t('providersPage.oauthAccounts.columns.lastRefresh')}</span>
          <span className="text-right">{t('providersPage.oauthAccounts.columns.actions')}</span>
        </div>
        <div className="flex flex-col">
          {items.map((file) => {
            const statusMessage = getAuthFileStatusMessage(file);
            const unavailable = isAuthFileUnavailable(file);
            const hasIssue = hasAuthFileHealthIssue(file);
            const status = file.disabled
              ? t('providersPage.oauthAccounts.disabled')
              : statusMessage || file.status ||
                (unavailable ? t('auth_files.health_status_warning') : t('providersPage.oauthAccounts.active'));
            return (
              <div
                key={file.name}
                className="grid grid-cols-[minmax(180px,1.5fr)_minmax(100px,0.8fr)_minmax(160px,1fr)_auto] items-center gap-3 border-b border-border px-3 py-2 last:border-b-0"
              >
                <span className="truncate text-sm font-medium text-foreground">{file.name}</span>
                <span
                  className={[
                    'inline-flex w-fit items-center border px-2 py-0.5 text-[11px] font-medium',
                    file.disabled
                      ? 'border-amber-400/40 bg-amber-100 text-amber-700'
                      : unavailable
                        ? 'border-destructive/30 bg-destructive/10 text-destructive'
                        : hasIssue
                          ? 'border-amber-400/40 bg-amber-100 text-amber-700'
                          : 'border-emerald-400/40 bg-emerald-100 text-emerald-700',
                  ].join(' ')}
                >
                  {status}
                </span>
                <span className="text-[11px] text-muted-foreground">
                  {formatLastRefresh(file.lastRefresh, i18n.language)}
                </span>
                <div className="flex justify-end gap-1">
                  <ToggleSwitch
                    checked={!file.disabled}
                    onChange={(enabled) => void handleStatusToggle(file, enabled)}
                    disabled={Boolean(statusUpdating[file.name])}
                    ariaLabel={t('auth_files.status_toggle_label')}
                  />
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => void handleDownload(file.name)}
                    title={t('providersPage.oauthAccounts.download')}
                  >
                    <IconDownload size={14} />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => handleDelete(file.name)}
                    loading={deleting === file.name}
                    title={t('common.delete')}
                  >
                    <IconTrash2 size={14} />
                  </Button>
                </div>
              </div>
            );
          })}
        </div>
      </div>
        </div>
      )}
    </div>
  );
}
