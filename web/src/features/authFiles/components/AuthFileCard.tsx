import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/Button';
import { LoadingSpinner } from '@/components/ui/LoadingSpinner';
import { SelectionCheckbox } from '@/components/ui/SelectionCheckbox';
import { ToggleSwitch } from '@/components/ui/ToggleSwitch';
import {
  IconDownload,
  IconInfo,
  IconTrash2,
} from '@/components/ui/icons';
import { ProviderStatusBar } from '@/components/providers/ProviderStatusBar';
import type { AuthFileItem } from '@/types';
import {
  normalizeRecentRequestAuthIndex,
  normalizeRecentRequestBuckets,
  normalizeUsageTotal,
  statusBarDataFromRecentRequests,
} from '@/utils/recentRequests';
import { formatFileSize } from '@/utils/format';
import {
  formatModified,
  getAuthFileIcon,
  getAuthFileStatusMessage,
  getTypeColor,
  getTypeLabel,
  HEALTHY_AUTH_FILE_STATUS_MESSAGES,
  isRuntimeOnlyAuthFile,
  normalizeProviderKey,
  parsePriorityValue,
  type ResolvedTheme,
} from '@/features/authFiles/constants';
import type { AuthFileStatusBarData } from '@/features/authFiles/hooks/useAuthFilesStatusBarCache';

export type AuthFileCardProps = {
  file: AuthFileItem;
  compact: boolean;
  selected: boolean;
  resolvedTheme: ResolvedTheme;
  disableControls: boolean;
  deleting: string | null;
  statusUpdating: Record<string, boolean>;
  statusBarCache: Map<string, AuthFileStatusBarData>;
  onDownload: (name: string) => void;
  onDelete: (name: string) => void;
  onToggleStatus: (file: AuthFileItem, enabled: boolean) => void;
  onToggleSelect: (name: string) => void;
};

export function AuthFileCard(props: AuthFileCardProps) {
  const { t } = useTranslation();
  const {
    file,
    compact,
    selected,
    resolvedTheme,
    disableControls,
    deleting,
    statusUpdating,
    statusBarCache,
    onDownload,
    onDelete,
    onToggleStatus,
    onToggleSelect,
  } = props;

  const recentBuckets = normalizeRecentRequestBuckets(file.recent_requests ?? file.recentRequests);
  const fileStats = {
    success: normalizeUsageTotal(file.success),
    failure: normalizeUsageTotal(file.failed),
  };
  const isRuntimeOnly = isRuntimeOnlyAuthFile(file);
  const providerKey = normalizeProviderKey(String(file.type ?? file.provider ?? 'unknown'));
  const typeColor = getTypeColor(providerKey, resolvedTheme);
  const typeLabel = getTypeLabel(t, providerKey);
  const providerIcon = getAuthFileIcon(providerKey, resolvedTheme);

  const providerCardBgImage =
    providerKey === 'antigravity'
      ? 'linear-gradient(180deg,rgba(224,247,250,0.06),transparent)'
      : providerKey === 'claude'
        ? 'linear-gradient(180deg,rgba(251,236,228,0.08),transparent)'
        : providerKey === 'codex'
          ? 'linear-gradient(180deg,rgba(234,231,255,0.08),transparent)'
          : providerKey === 'gemini-cli'
            ? 'linear-gradient(180deg,rgba(224,232,255,0.08),transparent)'
            : providerKey === 'kiro'
              ? 'linear-gradient(180deg,rgba(239,231,255,0.08),transparent)'
              : providerKey === 'kimi'
                ? 'linear-gradient(180deg,rgba(220,232,255,0.08),transparent)'
                : providerKey === 'xai'
                  ? 'linear-gradient(180deg,rgba(243,244,246,0.08),transparent)'
                  : undefined;

  const rawAuthIndex = file['auth_index'] ?? file.authIndex;
  const authIndexKey = normalizeRecentRequestAuthIndex(rawAuthIndex);
  const statusData =
    (authIndexKey && statusBarCache.get(authIndexKey)) ||
    statusBarDataFromRecentRequests(recentBuckets);
  const rawStatusMessage = getAuthFileStatusMessage(file);
  const hasStatusWarning =
    Boolean(rawStatusMessage) && !HEALTHY_AUTH_FILE_STATUS_MESSAGES.has(rawStatusMessage.toLowerCase());

  const priorityValue = parsePriorityValue(file.priority ?? file['priority']);
  const noteValue = typeof file.note === 'string' ? file.note.trim() : '';
  const stateLabel = isRuntimeOnly
    ? t('auth_files.type_virtual') || '虚拟认证文件'
    : file.disabled
      ? t('auth_files.health_status_disabled')
      : hasStatusWarning
        ? t('auth_files.health_status_warning')
        : rawStatusMessage
          ? t('auth_files.health_status_healthy')
          : t('auth_files.status_toggle_label');
  const stateBadgeClass = isRuntimeOnly
    ? 'text-muted-foreground bg-secondary/80 border border-dashed border-border/90'
    : file.disabled
      ? 'text-muted-foreground bg-secondary/85 border border-border'
      : hasStatusWarning
        ? 'text-amber-700 bg-amber-100 border border-amber-400/30'
        : 'text-emerald-700 bg-emerald-100 border border-emerald-400/40';

  return (
    <div
      className={`relative overflow-hidden border border-border/70 ${compact ? 'p-3' : 'p-4'} flex flex-col min-h-full bg-background/85 backdrop-blur-[8px] ${selected ? 'border-primary/60' : ''} ${file.disabled ? 'border-border/62' : ''}`}
      style={providerCardBgImage ? { backgroundImage: providerCardBgImage } : undefined}
    >
      <div className="flex items-stretch gap-3 min-h-full">
        <div className={`flex flex-col min-w-0 flex-1 ${compact ? 'gap-2' : 'gap-[10px]'}`}>
          <div className={`flex items-start ${compact ? 'gap-2' : 'gap-[10px]'}`}>
            {!isRuntimeOnly && (
              <SelectionCheckbox
                checked={selected}
                onChange={() => onToggleSelect(file.name)}
                className={`shrink-0 ${compact ? 'mt-[6px]' : 'mt-2'}`}
                aria-label={
                  selected ? t('auth_files.batch_deselect') : t('auth_files.batch_select_all')
                }
                title={selected ? t('auth_files.batch_deselect') : t('auth_files.batch_select_all')}
              />
            )}
            <div
              className={`shrink-0 inline-flex items-center justify-center border border-border/60 ${compact ? 'w-[34px] h-[34px]' : 'w-10 h-10'}`}
              style={{
                backgroundColor: typeColor.bg,
                color: typeColor.text,
                ...(typeColor.border ? { border: typeColor.border } : {}),
              }}
            >
              {providerIcon ? (
                <img
                  src={providerIcon}
                  alt=""
                  className={`object-contain ${compact ? 'w-4 h-4' : 'w-[22px] h-[22px]'}`}
                />
              ) : (
                <span
                  className={`font-bold leading-none ${compact ? 'text-[14px]' : 'text-[18px]'}`}
                >
                  {typeLabel.slice(0, 1).toUpperCase()}
                </span>
              )}
            </div>
            <div className={`flex flex-col flex-1 min-w-0 ${compact ? 'gap-[3px]' : 'gap-[6px]'}`}>
              <div className={`flex flex-wrap items-center ${compact ? 'gap-1' : 'gap-2'}`}>
                <span
                  className={`inline-flex items-center border border-transparent font-semibold leading-none whitespace-nowrap shrink-0 ${compact ? 'py-[3px] px-[7px] text-[10px]' : 'py-[3px] px-2 text-[11px]'}`}
                  style={{
                    backgroundColor: typeColor.bg,
                    color: typeColor.text,
                    ...(typeColor.border ? { border: typeColor.border } : {}),
                  }}
                >
                  {typeLabel}
                </span>
                <span
                  className={`inline-flex items-center font-bold leading-none whitespace-nowrap tracking-[0.02em] ${compact ? 'py-[3px] px-[7px] text-[10px]' : 'py-[3px] px-2 text-[10px]'} ${stateBadgeClass}`}
                >
                  {stateLabel}
                </span>
              </div>
              <span
                className={`font-extrabold text-foreground break-all leading-[1.4] tracking-[-0.01em] overflow-hidden [-webkit-box-orient:vertical] [display:-webkit-box] ${compact ? 'text-[13px] [-webkit-line-clamp:1]' : 'text-[14px] [-webkit-line-clamp:2]'}`}
                title={file.name}
              >
                {file.name}
              </span>
              {!compact && noteValue && (
                <div
                  className="text-[12px] text-muted-foreground leading-[1.5] flex gap-[6px] items-start min-w-0"
                  title={noteValue}
                >
                  <span className="shrink-0 text-muted-foreground/60 font-semibold">
                    {t('auth_files.note_display')}
                  </span>
                  <span className="min-w-0 break-words overflow-hidden text-ellipsis [-webkit-box-orient:vertical] [display:-webkit-box] [-webkit-line-clamp:2]">
                    {noteValue}
                  </span>
                </div>
              )}
            </div>
          </div>

          <div
            className={`flex flex-wrap items-baseline px-[2px] ${compact ? 'gap-[3px_12px]' : 'gap-1 gap-x-[14px]'}`}
          >
            <div className="inline-flex items-baseline gap-[5px] min-w-0 p-0 border-none bg-none">
              <span
                className={`font-medium text-muted-foreground/60 leading-[1.2] whitespace-nowrap after:content-[':'] ${compact ? 'text-[10px]' : 'text-[11px]'}`}
              >
                {t('auth_files.file_size')}
              </span>
              <span
                className={`min-w-0 font-semibold text-muted-foreground leading-[1.4] break-words ${compact ? 'text-[11px]' : 'text-[12px]'}`}
              >
                {file.size ? formatFileSize(file.size) : '-'}
              </span>
            </div>
            <div className="inline-flex items-baseline gap-[5px] min-w-0 p-0 border-none bg-none">
              <span
                className={`font-medium text-muted-foreground/60 leading-[1.2] whitespace-nowrap after:content-[':'] ${compact ? 'text-[10px]' : 'text-[11px]'}`}
              >
                {t('auth_files.file_modified')}
              </span>
              <span
                className={`min-w-0 font-semibold text-muted-foreground leading-[1.4] break-words ${compact ? 'text-[11px]' : 'text-[12px]'}`}
              >
                {formatModified(file)}
              </span>
            </div>
            {priorityValue !== undefined && (
              <div className="inline-flex items-baseline gap-[5px] min-w-0 p-0 border-none bg-none">
                <span
                  className={`font-medium text-muted-foreground/60 leading-[1.2] whitespace-nowrap after:content-[':'] ${compact ? 'text-[10px]' : 'text-[11px]'}`}
                >
                  {t('auth_files.priority_display')}
                </span>
                <span
                  className={`min-w-0 font-semibold text-muted-foreground leading-[1.4] break-words tabular-nums ${compact ? 'text-[11px]' : 'text-[12px]'}`}
                >
                  {priorityValue}
                </span>
              </div>
            )}
          </div>

          {rawStatusMessage && hasStatusWarning && (
            <div
              className={`flex items-start gap-[6px] text-[11px] text-amber-700 bg-amber-100/60 border border-amber-400/30 p-[8px_10px] break-words ${compact ? 'text-[11px] p-[6px_8px]' : ''}`}
              title={rawStatusMessage}
            >
              <IconInfo className="shrink-0 mt-[1px]" size={14} />
              <span>{rawStatusMessage}</span>
            </div>
          )}

          <div className={`flex flex-col p-[0_2px] ${compact ? 'gap-[6px]' : 'gap-2'}`}>
            <div className={`flex items-center flex-wrap ${compact ? 'gap-[6px]' : 'gap-2'}`}>
              <div
                className={`inline-flex items-baseline gap-[5px] min-w-0 bg-emerald-100/50 text-emerald-700 ${compact ? 'py-[3px] px-2' : 'py-1 px-[10px]'}`}
              >
                <span
                  className={`font-semibold leading-none ${compact ? 'text-[10px]' : 'text-[11px]'}`}
                >
                  {t('stats.success')}
                </span>
                <span
                  className={`font-bold leading-none tabular-nums ${compact ? 'text-[12px]' : 'text-[13px]'}`}
                >
                  {fileStats.success}
                </span>
              </div>
              <div
                className={`inline-flex items-baseline gap-[5px] min-w-0 bg-destructive/10 text-destructive ${compact ? 'py-[3px] px-2' : 'py-1 px-[10px]'}`}
              >
                <span
                  className={`font-semibold leading-none ${compact ? 'text-[10px]' : 'text-[11px]'}`}
                >
                  {t('stats.failure')}
                </span>
                <span
                  className={`font-bold leading-none tabular-nums ${compact ? 'text-[12px]' : 'text-[13px]'}`}
                >
                  {fileStats.failure}
                </span>
              </div>
            </div>

            <div className={`flex flex-col ${compact ? 'gap-[3px]' : 'gap-1'}`}>
              {!compact && (
                <div className="inline-flex items-center gap-1 text-[10px] font-medium text-muted-foreground/60 uppercase tracking-[0.03em]">
                  <span>{t('auth_files.health_status_label')}</span>
                </div>
              )}
              <ProviderStatusBar statusData={statusData} />
            </div>
          </div>

          <div
            className={`flex items-center justify-between gap-2 mt-auto pt-2 border-t border-border/50 max-md:flex-wrap`}
          >
            <div className="flex items-center gap-[6px] min-w-0 flex-1 max-md:flex-wrap max-md:w-full">
              {!isRuntimeOnly && (
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => onDownload(file.name)}
                  className="w-8 h-8 min-w-[32px] p-0 box-border gap-0"
                  title={t('auth_files.download_button')}
                  disabled={disableControls}
                >
                  <IconDownload className="block" size={16} />
                </Button>
              )}
              {!isRuntimeOnly && (
                <Button
                  variant="danger"
                  size="sm"
                  onClick={() => onDelete(file.name)}
                  className="w-8 h-8 min-w-[32px] p-0 box-border gap-0"
                  title={t('auth_files.delete_button')}
                  disabled={disableControls || deleting === file.name}
                >
                  {deleting === file.name ? (
                    <LoadingSpinner size={14} />
                  ) : (
                    <IconTrash2 className="block" size={16} />
                  )}
                </Button>
              )}
            </div>
            {!isRuntimeOnly && (
              <div
                className={`inline-flex items-center gap-[6px] shrink-0 ml-auto max-md:w-full max-md:justify-between`}
              >
                <span className="text-[11px] font-medium text-muted-foreground/60">
                  {t('auth_files.status_toggle_label')}
                </span>
                <ToggleSwitch
                  ariaLabel={t('auth_files.status_toggle_label')}
                  checked={!file.disabled}
                  disabled={disableControls || statusUpdating[file.name] === true}
                  onChange={(value) => onToggleStatus(file, value)}
                />
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
