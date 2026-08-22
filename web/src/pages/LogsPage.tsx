import { useDeferredValue, useEffect, useMemo, useRef, useState } from 'react';
import type { PointerEvent as ReactPointerEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { AppCard as Card } from '@/components/ui/AppCard';
import { Button } from '@/components/ui/Button';
import { EmptyState } from '@/components/ui/EmptyState';
import { FormInput as Input } from '@/components/ui/FormInput';
import { Modal } from '@/components/ui/Modal';
import { ToggleSwitch } from '@/components/ui/ToggleSwitch';
import {
  IconChevronDown,
  IconChevronUp,
  IconCode,
  IconDownload,
  IconEyeOff,
  IconRefreshCw,
  IconSearch,
  IconSlidersHorizontal,
  IconTimer,
  IconTrash2,
  IconX,
} from '@/components/ui/icons';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useHeaderRefresh } from '@/hooks/useHeaderRefresh';
import { useLocalStorage } from '@/hooks/useLocalStorage';
import { toast } from 'sonner';
import { useAuthStore, useConfigStore, useConfirmationStore } from '@/stores';
import { logsApi } from '@/services/api/logs';
import { copyToClipboard } from '@/utils/clipboard';
import { downloadBlob } from '@/utils/download';
import { MANAGEMENT_API_PREFIX } from '@/utils/constants';
import { formatUnixTimestamp } from '@/utils/format';
import {
  HTTP_METHODS,
  STATUS_GROUPS,
  resolveStatusGroup,
  type LogState,
} from './hooks/logTypes';
import { parseLogLine } from './hooks/logParsing';
import { useLogFilters } from './hooks/useLogFilters';
import { isNearBottom, useLogScroller } from './hooks/useLogScroller';

interface ErrorLogItem {
  name: string;
  size?: number;
  modified?: number;
}

// 初始只渲染最近 100 行，滚动到顶部再逐步加载更多（避免一次性渲染过多导致卡顿）
const INITIAL_DISPLAY_LINES = 100;
const MAX_BUFFER_LINES = 10000;
const LONG_PRESS_MS = 650;
const LONG_PRESS_MOVE_THRESHOLD = 10;

const getErrorMessage = (err: unknown): string => {
  if (err instanceof Error) return err.message;
  if (typeof err === 'string') return err;
  if (typeof err !== 'object' || err === null) return '';
  if (!('message' in err)) return '';

  const message = (err as { message?: unknown }).message;
  return typeof message === 'string' ? message : '';
};

type TabType = 'logs' | 'errors';

export function LogsPage() {
  const { t } = useTranslation();
  const { showConfirmation } = useConfirmationStore();
  const connectionStatus = useAuthStore((state) => state.connectionStatus);
  const config = useConfigStore((state) => state.config);
  const requestLogEnabled = config?.requestLog ?? false;

  const [activeTab, setActiveTab] = useState<TabType>('logs');
  const [logState, setLogState] = useState<LogState>({ buffer: [], visibleFrom: 0 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [autoRefresh, setAutoRefresh] = useLocalStorage('logsPage.autoRefresh', false);
  const [searchQuery, setSearchQuery] = useState('');
  const deferredSearchQuery = useDeferredValue(searchQuery);
  const [hideManagementLogs, setHideManagementLogs] = useLocalStorage(
    'logsPage.hideManagementLogs',
    true
  );
  const [showRawLogs, setShowRawLogs] = useLocalStorage('logsPage.showRawLogs', false);
  const [structuredFiltersExpanded, setStructuredFiltersExpanded] = useLocalStorage(
    'logsPage.structuredFiltersExpanded',
    true
  );
  const [errorLogs, setErrorLogs] = useState<ErrorLogItem[]>([]);
  const [loadingErrors, setLoadingErrors] = useState(false);
  const [errorLogsError, setErrorLogsError] = useState('');
  const [requestLogId, setRequestLogId] = useState<string | null>(null);
  const [requestLogDownloading, setRequestLogDownloading] = useState(false);

  const logScrollerRef = useRef<ReturnType<typeof useLogScroller> | null>(null);
  const longPressRef = useRef<{
    timer: number | null;
    startX: number;
    startY: number;
    fired: boolean;
  } | null>(null);
  const logRequestInFlightRef = useRef(false);
  const pendingFullReloadRef = useRef(false);

  // 保存最新时间戳用于增量获取
  const latestTimestampRef = useRef<number>(0);

  const disableControls = connectionStatus !== 'connected';

  const loadLogs = async (incremental = false) => {
    if (connectionStatus !== 'connected') {
      setLoading(false);
      return;
    }

    if (logRequestInFlightRef.current) {
      if (!incremental) {
        pendingFullReloadRef.current = true;
      }
      return;
    }

    logRequestInFlightRef.current = true;

    if (!incremental) {
      setLoading(true);
    }
    setError('');

    try {
      const scrollerInstance = logScrollerRef.current;
      const stickToBottom =
        !incremental || isNearBottom(scrollerInstance?.logViewerRef.current ?? null);
      if (stickToBottom) {
        scrollerInstance?.requestScrollToBottom();
      }

      const params =
        incremental && latestTimestampRef.current > 0 ? { after: latestTimestampRef.current } : {};
      const data = await logsApi.fetchLogs(params);

      // 更新时间戳
      if (data['latest-timestamp']) {
        latestTimestampRef.current = data['latest-timestamp'];
      }

      const newLines = Array.isArray(data.lines) ? data.lines : [];

      if (incremental && newLines.length > 0) {
        // 增量更新：追加新日志并限制缓冲区大小（避免内存与渲染膨胀）
        setLogState((prev) => {
          const prevRenderedCount = prev.buffer.length - prev.visibleFrom;
          const combined = [...prev.buffer, ...newLines];
          const dropCount = Math.max(combined.length - MAX_BUFFER_LINES, 0);
          const buffer = dropCount > 0 ? combined.slice(dropCount) : combined;
          let visibleFrom = Math.max(prev.visibleFrom - dropCount, 0);

          // 若用户停留在底部（跟随最新日志），则保持"渲染窗口"大小不变，避免无限增长
          if (stickToBottom) {
            visibleFrom = Math.max(buffer.length - prevRenderedCount, 0);
          }

          return { buffer, visibleFrom };
        });
      } else if (!incremental) {
        // 全量加载：默认只渲染最后 100 行，向上滚动再展开更多
        const buffer = newLines.slice(-MAX_BUFFER_LINES);
        const visibleFrom = Math.max(buffer.length - INITIAL_DISPLAY_LINES, 0);
        setLogState({ buffer, visibleFrom });
      }
    } catch (err: unknown) {
      console.error('Failed to load logs:', err);
      if (!incremental) {
        setError(getErrorMessage(err) || t('logs.load_error'));
      }
    } finally {
      if (!incremental) {
        setLoading(false);
      }
      logRequestInFlightRef.current = false;
      if (pendingFullReloadRef.current) {
        pendingFullReloadRef.current = false;
        void loadLogs(false);
      }
    }
  };

  useHeaderRefresh(() => loadLogs(false));

  const clearLogs = async () => {
    showConfirmation({
      title: t('logs.clear_confirm_title', { defaultValue: 'Clear Logs' }),
      message: t('logs.clear_confirm'),
      variant: 'danger',
      confirmText: t('common.confirm'),
      onConfirm: async () => {
        try {
          await logsApi.clearLogs();
          setLogState({ buffer: [], visibleFrom: 0 });
          latestTimestampRef.current = 0;
          toast.success(t('logs.clear_success'));
        } catch (err: unknown) {
          const message = getErrorMessage(err);
          toast.error(
            `${t('notification.delete_failed')}${message ? `: ${message}` : ''}`
          );
        }
      },
    });
  };

  const downloadLogs = () => {
    const text = logState.buffer.join('\n');
    downloadBlob({ filename: 'logs.txt', blob: new Blob([text], { type: 'text/plain' }) });
    toast.success(t('logs.download_success'));
  };

  const loadErrorLogs = async () => {
    if (connectionStatus !== 'connected') {
      setLoadingErrors(false);
      return;
    }

    setLoadingErrors(true);
    setErrorLogsError('');
    try {
      const res = await logsApi.fetchErrorLogs();
      // API 返回 { files: [...] }
      setErrorLogs(Array.isArray(res.files) ? res.files : []);
    } catch (err: unknown) {
      console.error('Failed to load error logs:', err);
      setErrorLogs([]);
      const message = getErrorMessage(err);
      setErrorLogsError(
        message ? `${t('logs.error_logs_load_error')}: ${message}` : t('logs.error_logs_load_error')
      );
    } finally {
      setLoadingErrors(false);
    }
  };

  const downloadErrorLog = async (name: string) => {
    try {
      const response = await logsApi.downloadErrorLog(name);
      downloadBlob({ filename: name, blob: new Blob([response.data], { type: 'text/plain' }) });
      toast.success(t('logs.error_log_download_success'));
    } catch (err: unknown) {
      const message = getErrorMessage(err);
      toast.error(
        `${t('notification.download_failed')}${message ? `: ${message}` : ''}`
      );
    }
  };

  useEffect(() => {
    if (connectionStatus === 'connected') {
      latestTimestampRef.current = 0;
      loadLogs(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [connectionStatus]);

  useEffect(() => {
    if (activeTab !== 'errors') return;
    if (connectionStatus !== 'connected') return;
    void loadErrorLogs();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTab, connectionStatus, requestLogEnabled]);

  useEffect(() => {
    if (!autoRefresh || connectionStatus !== 'connected') {
      return;
    }
    const id = window.setInterval(() => {
      loadLogs(true);
    }, 8000);
    return () => window.clearInterval(id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [autoRefresh, connectionStatus]);

  const visibleLines = useMemo(
    () => logState.buffer.slice(logState.visibleFrom),
    [logState.buffer, logState.visibleFrom]
  );

  const trimmedSearchQuery = deferredSearchQuery.trim();
  const isSearching = trimmedSearchQuery.length > 0;
  const baseLines = isSearching ? logState.buffer : visibleLines;

  const parsedSearchLines = useMemo(() => {
    let working = baseLines;

    if (hideManagementLogs) {
      working = working.filter((line) => !line.includes(MANAGEMENT_API_PREFIX));
    }

    if (trimmedSearchQuery) {
      const queryLowered = trimmedSearchQuery.toLowerCase();
      working = working.filter((line) => line.toLowerCase().includes(queryLowered));
    }

    return working.map((line) => parseLogLine(line));
  }, [baseLines, hideManagementLogs, trimmedSearchQuery]);

  const filters = useLogFilters({ parsedLines: parsedSearchLines });
  const structuredFiltersPanelId = 'logs-structured-filters';
  const structuredFilterCount =
    filters.methodFilters.length + filters.statusFilters.length + filters.pathFilters.length;

  const { filteredParsedLines, filteredLines, removedCount } = useMemo(() => {
    const filteredParsed = parsedSearchLines.filter((line) => {
      if (
        filters.methodFilterSet.size > 0 &&
        (!line.method || !filters.methodFilterSet.has(line.method))
      ) {
        return false;
      }

      const statusGroup = resolveStatusGroup(line.statusCode);
      if (
        filters.statusFilterSet.size > 0 &&
        (!statusGroup || !filters.statusFilterSet.has(statusGroup))
      ) {
        return false;
      }

      if (filters.pathFilterSet.size > 0 && (!line.path || !filters.pathFilterSet.has(line.path))) {
        return false;
      }

      return true;
    });

    return {
      filteredParsedLines: filteredParsed,
      filteredLines: filteredParsed.map((line) => line.raw),
      removedCount: Math.max(baseLines.length - filteredParsed.length, 0)
    };
  }, [
    baseLines,
    filters.methodFilterSet,
    filters.pathFilterSet,
    filters.statusFilterSet,
    parsedSearchLines
  ]);

  const parsedVisibleLines = useMemo(
    () => (showRawLogs ? [] : filteredParsedLines),
    [filteredParsedLines, showRawLogs]
  );

  const rawVisibleText = useMemo(() => filteredLines.join('\n'), [filteredLines]);

  const scroller = useLogScroller({
    logState,
    setLogState,
    loading,
    isSearching,
    filteredLineCount: filteredLines.length,
    hasStructuredFilters: filters.hasStructuredFilters,
    showRawLogs
  });

  logScrollerRef.current = scroller;

  const copyLogLine = async (raw: string) => {
    const ok = await copyToClipboard(raw);
    if (ok) {
      toast.success(t('logs.copy_success', { defaultValue: 'Copied to clipboard' }));
    } else {
      toast.error(t('logs.copy_failed', { defaultValue: 'Copy failed' }));
    }
  };

  const clearLongPressTimer = () => {
    if (longPressRef.current?.timer) {
      window.clearTimeout(longPressRef.current.timer);
      longPressRef.current.timer = null;
    }
  };

  const startLongPress = (event: ReactPointerEvent<HTMLDivElement>, id?: string) => {
    if (!requestLogEnabled) return;
    if (!id) return;
    if (requestLogId) return;
    clearLongPressTimer();
    longPressRef.current = {
      timer: window.setTimeout(() => {
        setRequestLogId(id);
        if (longPressRef.current) {
          longPressRef.current.fired = true;
          longPressRef.current.timer = null;
        }
      }, LONG_PRESS_MS),
      startX: event.clientX,
      startY: event.clientY,
      fired: false,
    };
  };

  const cancelLongPress = () => {
    clearLongPressTimer();
    longPressRef.current = null;
  };

  const handleLongPressMove = (event: ReactPointerEvent<HTMLDivElement>) => {
    const current = longPressRef.current;
    if (!current || current.timer === null || current.fired) return;
    const deltaX = Math.abs(event.clientX - current.startX);
    const deltaY = Math.abs(event.clientY - current.startY);
    if (deltaX > LONG_PRESS_MOVE_THRESHOLD || deltaY > LONG_PRESS_MOVE_THRESHOLD) {
      cancelLongPress();
    }
  };

  const closeRequestLogModal = () => {
    if (requestLogDownloading) return;
    setRequestLogId(null);
  };

  const downloadRequestLog = async (id: string) => {
    setRequestLogDownloading(true);
    try {
      const response = await logsApi.downloadRequestLogById(id);
      downloadBlob({
        filename: `request-${id}.log`,
        blob: new Blob([response.data], { type: 'text/plain' })
      });
      toast.success(t('logs.request_log_download_success'));
      setRequestLogId(null);
    } catch (err: unknown) {
      const message = getErrorMessage(err);
      toast.error(
        `${t('notification.download_failed')}${message ? `: ${message}` : ''}`
      );
    } finally {
      setRequestLogDownloading(false);
    }
  };

  useEffect(() => {
    return () => {
      if (longPressRef.current?.timer) {
        window.clearTimeout(longPressRef.current.timer);
        longPressRef.current.timer = null;
      }
    };
  }, []);

  return (
    // container: width 100%, flex col, flex-1, min-h-0; mobile: min-h-auto overflow-visible
    <div className="w-full flex-1 flex flex-col min-h-0 max-md:min-h-auto max-md:overflow-visible">
      {/* pageTitle */}
      <h1 className="text-[28px] font-bold text-foreground m-0 mb-4 max-[820px]:text-2xl max-[820px]:mb-3 max-[600px]:text-xl max-[600px]:mb-2">
        {t('logs.title')}
      </h1>

      {/* tabBar */}
      <Tabs
        value={activeTab}
        onValueChange={(v) => setActiveTab(v as TabType)}
        className="mb-4 max-[820px]:mb-3 max-[600px]:mb-2"
      >
        <TabsList className="flex gap-0 p-0 border-b border-border bg-transparent">
          <TabsTrigger
            value="logs"
            className="text-muted-foreground px-5 py-3 text-sm font-medium border-b-4 border-transparent -mb-[3px] data-[state=active]:border-foreground data-[state=active]:text-foreground hover:text-foreground max-[820px]:px-4 max-[820px]:py-[10px] max-[600px]:px-3 max-[600px]:py-2 max-[600px]:text-[13px]"
          >
            {t('logs.log_content')}
          </TabsTrigger>
          <TabsTrigger
            value="errors"
            className="text-muted-foreground px-5 py-3 text-sm font-medium border-b-4 border-transparent -mb-[3px] data-[state=active]:border-foreground data-[state=active]:text-foreground hover:text-foreground max-[820px]:px-4 max-[820px]:py-[10px] max-[600px]:px-3 max-[600px]:py-2 max-[600px]:text-[13px]"
          >
            {t('logs.error_logs_modal_title')}
          </TabsTrigger>
        </TabsList>
      </Tabs>

      {/* content: flex col, flex-1, min-h-0, gap-4; mobile: gap-3 min-h-auto */}
      <div className="flex flex-col gap-4 flex-1 min-h-0 max-md:gap-3 max-md:min-h-auto max-[820px]:gap-3 max-[600px]:gap-2">
        {activeTab === 'logs' && (
          // logCard: flex col, flex-1, min-h-0, overflow-hidden; mobile: flex-none min-h-auto overflow-visible
          <Card className="flex flex-col flex-1 min-h-0 overflow-hidden max-md:flex-none max-md:min-h-auto max-md:overflow-visible max-[820px]:p-3 max-[600px]:p-2">
            {error && <div className="p-[10px_14px] mb-2 bg-destructive/10 border border-destructive/35 text-destructive text-sm leading-[1.5]">{error}</div>}

            {/* filters: flex wrap, items-center, gap-3, mb-3; mobile: gap-2 mb-2 */}
            <div className="flex items-center gap-3 flex-wrap mb-3 max-md:gap-2 max-md:mb-2 max-[600px]:gap-2 max-[600px]:mb-2">
              {/* searchWrapper: flex-1, min-w-[220px], max-w-[420px] */}
              <div className="flex-1 min-w-[220px] max-w-[420px]">
                <Input
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder={t('logs.search_placeholder')}
                  className="pr-11!"
                  rightElement={
                    searchQuery ? (
                      <button
                        type="button"
                        className="appearance-none border-0 inline-flex items-center justify-center w-7 h-7 rounded-full text-muted-foreground hover:bg-muted cursor-pointer bg-transparent p-0"
                        onClick={() => setSearchQuery('')}
                        title="Clear"
                        aria-label="Clear"
                      >
                        <IconX size={16} />
                      </button>
                    ) : (
                      <IconSearch size={16} className="[color:var(--text-tertiary)] pointer-events-none" />
                    )
                  }
                />
              </div>

              {/* filterPanelHeader: flex items-center, flex-[1_1_100%]; mobile: w-full */}
              <div className="flex items-center flex-[1_1_100%] max-md:w-full">
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  className="whitespace-nowrap max-md:w-full"
                  onClick={() => setStructuredFiltersExpanded((prev) => !prev)}
                  aria-expanded={structuredFiltersExpanded}
                  aria-controls={structuredFiltersPanelId}
                  title={
                    structuredFiltersExpanded
                      ? t('logs.filter_panel_collapse')
                      : t('logs.filter_panel_expand')
                  }
                >
                  {/* filterPanelButtonContent */}
                  <span className="inline-flex items-center gap-[6px] max-md:w-full max-md:justify-between max-md:flex-wrap">
                    <IconSlidersHorizontal size={16} />
                    <span>{t('logs.filter_panel_title')}</span>
                    {structuredFilterCount > 0 && (
                      // filterPanelCount: badge with primary color
                      <span className="inline-flex items-center px-2 py-0.5 rounded-full bg-primary/12 text-primary text-[12px] leading-[1.2]">
                        {t('logs.filter_panel_active_count', { count: structuredFilterCount })}
                      </span>
                    )}
                    {structuredFiltersExpanded ? (
                      <IconChevronUp size={16} />
                    ) : (
                      <IconChevronDown size={16} />
                    )}
                  </span>
                </Button>
              </div>

              {structuredFiltersExpanded && (
                // structuredFilters: flex col, gap-2, flex-[1_1_100%], p-2 px-[10px], bg-muted, border, rounded
                <div
                  id={structuredFiltersPanelId}
                  className="flex flex-col gap-2 flex-[1_1_100%] py-2 px-[10px] bg-muted border border-border"
                >
                  {/* filterChipGroup: flex, items-start, gap-2, flex-wrap */}
                  <div className="flex items-start gap-2 flex-wrap max-md:flex-col max-md:gap-[6px]">
                    <span className="text-[12px] text-muted-foreground font-bold leading-7 min-w-[68px] max-md:min-w-0 max-md:leading-[1.2]">
                      {t('logs.filter_method')}
                    </span>
                    <div className="flex items-center gap-[6px] flex-wrap flex-1">
                      {HTTP_METHODS.map((method) => {
                        const active = filters.methodFilters.includes(method);
                        const count = filters.methodCounts[method] ?? 0;
                        return (
                          <button
                            key={method}
                            type="button"
                            className={[
                              'appearance-none inline-flex items-center gap-1 px-[10px] py-1 border text-[12px] leading-[1.3] cursor-pointer',
                              'max-w-[280px] whitespace-nowrap overflow-hidden text-ellipsis',
                              'disabled:opacity-55 disabled:cursor-not-allowed',
                              active
                                ? 'text-primary border-primary/45 bg-primary/14'
                                : 'border-border bg-background text-muted-foreground hover:enabled:text-foreground hover:enabled:border-primary',
                            ].join(' ')}
                            onClick={() => filters.toggleMethodFilter(method)}
                            disabled={count === 0 && !active}
                            aria-pressed={active}
                          >
                            {method} ({count})
                          </button>
                        );
                      })}
                    </div>
                  </div>

                  <div className="flex items-start gap-2 flex-wrap max-md:flex-col max-md:gap-[6px]">
                    <span className="text-[12px] text-muted-foreground font-bold leading-7 min-w-[68px] max-md:min-w-0 max-md:leading-[1.2]">
                      {t('logs.filter_status')}
                    </span>
                    <div className="flex items-center gap-[6px] flex-wrap flex-1">
                      {STATUS_GROUPS.map((statusGroup) => {
                        const active = filters.statusFilters.includes(statusGroup);
                        const count = filters.statusCounts[statusGroup] ?? 0;
                        return (
                          <button
                            key={statusGroup}
                            type="button"
                            className={[
                              'appearance-none inline-flex items-center gap-1 px-[10px] py-1 border text-[12px] leading-[1.3] cursor-pointer',
                              'max-w-[280px] whitespace-nowrap overflow-hidden text-ellipsis',
                              'disabled:opacity-55 disabled:cursor-not-allowed',
                              active
                                ? 'text-primary border-primary/45 bg-primary/14'
                                : 'border-border bg-background text-muted-foreground hover:enabled:text-foreground hover:enabled:border-primary',
                            ].join(' ')}
                            onClick={() => filters.toggleStatusFilter(statusGroup)}
                            disabled={count === 0 && !active}
                            aria-pressed={active}
                          >
                            {t(`logs.filter_status_${statusGroup}`)} ({count})
                          </button>
                        );
                      })}
                    </div>
                  </div>

                  <div className="flex items-start gap-2 flex-wrap max-md:flex-col max-md:gap-[6px]">
                    <span className="text-[12px] text-muted-foreground font-bold leading-7 min-w-[68px] max-md:min-w-0 max-md:leading-[1.2]">
                      {t('logs.filter_path')}
                    </span>
                    <div className="flex items-center gap-[6px] flex-wrap flex-1">
                      {filters.pathOptions.length === 0 ? (
                        <span className="text-[12px] [color:var(--text-tertiary)]">
                          {t('logs.filter_path_empty')}
                        </span>
                      ) : (
                        filters.pathOptions.map(({ path, count }) => {
                          const active = filters.pathFilters.includes(path);
                          return (
                            <button
                              key={path}
                              type="button"
                              className={[
                                'appearance-none inline-flex items-center gap-1 px-[10px] py-1 border text-[12px] leading-[1.3] cursor-pointer',
                                'max-w-[280px] whitespace-nowrap overflow-hidden text-ellipsis',
                                active
                                  ? 'text-primary border-primary/45 bg-primary/14'
                                  : 'border-border bg-background text-muted-foreground hover:text-foreground hover:border-primary',
                              ].join(' ')}
                              onClick={() => filters.togglePathFilter(path)}
                              aria-pressed={active}
                              title={path}
                            >
                              {path} ({count})
                            </button>
                          );
                        })
                      )}
                    </div>
                  </div>

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={filters.clearStructuredFilters}
                    disabled={!filters.hasStructuredFilters}
                  >
                    {t('logs.clear_filters')}
                  </Button>
                </div>
              )}

              <ToggleSwitch
                checked={hideManagementLogs}
                onChange={setHideManagementLogs}
                label={
                  <span className="inline-flex items-center gap-[6px] [&_svg]:shrink-0">
                    <IconEyeOff size={16} />
                    {t('logs.hide_management_logs', { prefix: MANAGEMENT_API_PREFIX })}
                  </span>
                }
              />

              <ToggleSwitch
                checked={showRawLogs}
                onChange={setShowRawLogs}
                label={
                  <span
                    className="inline-flex items-center gap-[6px] [&_svg]:shrink-0"
                    title={t('logs.show_raw_logs_hint', {
                      defaultValue: 'Show original log text for easier multi-line copy',
                    })}
                  >
                    <IconCode size={16} />
                    {t('logs.show_raw_logs', { defaultValue: 'Show raw logs' })}
                  </span>
                }
              />

              {/* toolbar: flex items-center, gap-2, flex-wrap, ml-auto; mobile: ml-0 w-full items-start */}
              <div className="flex items-center gap-2 flex-wrap ml-auto max-md:ml-0 max-md:w-full max-md:items-start">
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => loadLogs(false)}
                  disabled={disableControls || loading}
                  className="whitespace-nowrap"
                >
                  <span className="inline-flex items-center gap-[6px] [&_svg]:shrink-0">
                    <IconRefreshCw size={16} />
                    {t('logs.refresh_button')}
                  </span>
                </Button>
                <ToggleSwitch
                  checked={autoRefresh}
                  onChange={(value) => setAutoRefresh(value)}
                  disabled={disableControls}
                  label={
                    <span className="inline-flex items-center gap-[6px] [&_svg]:shrink-0">
                      <IconTimer size={16} />
                      {t('logs.auto_refresh')}
                    </span>
                  }
                />
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={downloadLogs}
                  disabled={logState.buffer.length === 0}
                  className="whitespace-nowrap"
                >
                  <span className="inline-flex items-center gap-[6px] [&_svg]:shrink-0">
                    <IconDownload size={16} />
                    {t('logs.download_button')}
                  </span>
                </Button>
                <Button
                  variant="danger"
                  size="sm"
                  onClick={clearLogs}
                  disabled={disableControls}
                  className="whitespace-nowrap"
                >
                  <span className="inline-flex items-center gap-[6px] [&_svg]:shrink-0">
                    <IconTrash2 size={16} />
                    {t('logs.clear_button')}
                  </span>
                </Button>
              </div>
            </div>

            {loading ? (
              <div className="text-[13px] text-muted-foreground leading-[1.55]">{t('logs.loading')}</div>
            ) : logState.buffer.length > 0 && filteredLines.length > 0 ? (
              // logPanel: scroll container — preserving all scroll/overflow/height behavior
              // flex-[1_1_auto], min-h-[280px], max-h-[calc(100vh-320px)], overflow-auto, resize-y
              // mobile: flex-none, min-h-[360px], h-[420px], max-h-[480px]
              <div
                ref={scroller.logViewerRef}
                className={[
                  'bg-muted border border-border',
                  'flex-[1_1_auto] min-h-[280px] max-h-[calc(100vh-320px)] h-auto overflow-auto resize-y relative',
                  '[webkit-overflow-scrolling:touch] touch-pan-y overscroll-contain',
                  'max-md:flex-none max-md:min-h-[360px] max-md:h-[420px] max-md:max-h-[480px]',
                  'max-[960px]:min-[769px]:min-h-[160px] max-[960px]:min-[769px]:max-h-[calc(100vh-300px)]',
                ].join(' ')}
                onScroll={scroller.handleLogScroll}
              >
                {scroller.canLoadMore && (
                  // loadMoreBanner: sticky top, z-1, flex between, gap-2, p-2 px-3, border-b, bg-background
                  <div className="sticky top-0 z-[1] flex items-center justify-between gap-2 py-2 px-3 border-b border-border bg-background text-muted-foreground text-[12px] max-md:flex-col max-md:items-start max-md:justify-start max-md:gap-1 max-[600px]:py-[6px] max-[600px]:px-[10px]">
                    <span className="max-md:w-full">{t('logs.load_more_hint')}</span>
                    {/* loadMoreStats: flex items-center gap-3; mobile: w-full flex-wrap gap-2 */}
                    <div className="flex items-center gap-3 max-md:w-full max-md:flex-wrap max-md:gap-2">
                      <span>
                        {t('logs.loaded_lines', { count: filteredLines.length })}
                      </span>
                      {removedCount > 0 && (
                        <span className="[color:var(--text-tertiary)] whitespace-nowrap">
                          {t('logs.filtered_lines', { count: removedCount })}
                        </span>
                      )}
                      <span className="[color:var(--text-tertiary)] whitespace-nowrap">
                        {t('logs.hidden_lines', { count: logState.visibleFrom })}
                      </span>
                    </div>
                  </div>
                )}
                {showRawLogs ? (
                  // rawLog: pre, font-mono, text-foreground, selectable
                  <pre
                    className="m-0 py-[10px] px-3 cursor-text select-text whitespace-pre text-foreground font-mono text-[12.5px] leading-[1.45] max-[960px]:py-2 max-[960px]:px-[10px] max-[960px]:text-[12px] max-md:py-2 max-md:px-[10px] max-md:text-[11.5px]"
                    spellCheck={false}
                  >
                    {rawVisibleText}
                  </pre>
                ) : (
                  // logList: flex col
                  <div className="flex flex-col">
                    {parsedVisibleLines.map((line, index) => {
                      const rowClasses = [
                        // logRow base: grid [170px 1fr], gap-3, p-[10px_12px], border-b, border-l-[3px], cursor-copy, font-mono
                        'grid grid-cols-[170px_1fr] gap-3 py-[10px] px-3 border-b border-border',
                        'border-l-[3px] border-l-transparent cursor-copy',
                        'font-mono text-[12.5px] leading-[1.45] text-foreground',
                        'transition-colors duration-100 hover:bg-primary/[0.06]',
                        // tablet
                        'max-[960px]:grid-cols-[140px_1fr] max-[960px]:gap-2 max-[960px]:py-2 max-[960px]:px-[10px] max-[960px]:text-[12px]',
                        // mobile
                        'max-md:grid-cols-1 max-md:gap-1 max-md:py-2 max-md:px-[10px] max-md:text-[11.5px]',
                        // viewport-height compact
                        'max-[820px]:py-2 max-[820px]:px-[10px] max-[820px]:text-[12px]',
                        'max-[600px]:py-[6px] max-[600px]:px-2 max-[600px]:text-[11px] max-[600px]:grid-cols-[130px_1fr] max-[600px]:gap-1',
                      ];
                      if (line.level === 'warn') rowClasses.push('border-l-warning');
                      if (line.level === 'error' || line.level === 'fatal')
                        rowClasses.push('border-l-destructive');
                      return (
                        <div
                          key={`${logState.visibleFrom + index}-${line.raw}`}
                          className={rowClasses.join(' ')}
                          onDoubleClick={() => {
                            void copyLogLine(line.raw);
                          }}
                          onPointerDown={(event) => startLongPress(event, line.requestId)}
                          onPointerUp={cancelLongPress}
                          onPointerLeave={cancelLongPress}
                          onPointerCancel={cancelLongPress}
                          onPointerMove={handleLongPressMove}
                          title={t('logs.double_click_copy_hint', {
                            defaultValue: 'Double-click to copy',
                          })}
                        >
                          {/* timestamp: text-tertiary, whitespace-nowrap, pt-0.5; mobile: whitespace-normal */}
                          <div className="[color:var(--text-tertiary)] whitespace-nowrap pt-0.5 max-md:whitespace-normal">
                            {line.timestamp || ''}
                          </div>
                          {/* rowMain: flex wrap, items-baseline, gap-[6px], min-w-0 */}
                          <div className="flex flex-wrap items-baseline gap-[6px] min-w-0">
                            {line.level && (
                              <span
                                className={[
                                  // badge base: inline-flex, items-center, px-2 py-0.5, rounded-full, text-[12px], font-extrabold, border
                                  'inline-flex items-center px-2 py-0.5 text-[12px] font-extrabold border whitespace-nowrap',
                                  'bg-background text-muted-foreground border-border',
                                  line.level === 'info'
                                    ? 'text-primary bg-primary/12 border-primary/25'
                                    : '',
                                  line.level === 'warn'
                                    ? 'text-amber-700 bg-amber-100 border-amber-400/30'
                                    : '',
                                  line.level === 'error' || line.level === 'fatal'
                                    ? 'text-destructive bg-destructive/12 border-destructive/25'
                                    : '',
                                  line.level === 'debug' || line.level === 'trace'
                                    ? 'text-muted-foreground bg-gray-500/12 border-gray-500/25'
                                    : '',
                                ]
                                  .filter(Boolean)
                                  .join(' ')}
                              >
                                {line.level.toUpperCase()}
                              </span>
                            )}

                            {line.source && (
                              <span
                                className="text-muted-foreground max-w-[240px] overflow-hidden text-ellipsis whitespace-nowrap max-md:max-w-full"
                                title={line.source}
                              >
                                {line.source}
                              </span>
                            )}

                            {line.requestId && (
                              <span
                                className="inline-flex items-center px-2 py-0.5 text-[11px] font-extrabold border whitespace-nowrap font-mono text-primary bg-primary/10 border-primary/25"
                                title={line.requestId}
                              >
                                {line.requestId}
                              </span>
                            )}

                            {typeof line.statusCode === 'number' && (
                              <span
                                className={[
                                  'inline-flex items-center px-2 py-0.5 text-[12px] font-extrabold border whitespace-nowrap [font-variant-numeric:tabular-nums]',
                                  line.statusCode >= 200 && line.statusCode < 300
                                    ? 'text-emerald-700 bg-emerald-100 border-emerald-400/40'
                                    : line.statusCode >= 300 && line.statusCode < 400
                                      ? 'text-primary bg-primary/12 border-primary/25'
                                      : line.statusCode >= 400 && line.statusCode < 500
                                        ? 'text-amber-700 bg-amber-100 border-amber-400/30'
                                        : 'text-destructive bg-destructive/10 border-destructive/30',
                                ].join(' ')}
                              >
                                {line.statusCode}
                              </span>
                            )}

                            {line.latency && (
                              <span className="inline-flex items-center px-2 py-0.5 text-[12px] font-semibold border border-border bg-background text-muted-foreground whitespace-nowrap">
                                {line.latency}
                              </span>
                            )}
                            {line.ip && (
                              <span className="inline-flex items-center px-2 py-0.5 text-[12px] font-semibold border border-border bg-background text-muted-foreground whitespace-nowrap">
                                {line.ip}
                              </span>
                            )}

                            {line.method && (
                              <span className="inline-flex items-center px-2 py-0.5 text-[12px] font-extrabold border whitespace-nowrap text-foreground bg-primary/[0.08] border-primary/22">
                                {line.method}
                              </span>
                            )}

                            {line.path && (
                              <span
                                className="text-foreground font-bold max-w-[520px] overflow-hidden text-ellipsis whitespace-nowrap max-md:max-w-full max-md:basis-full"
                                title={line.path}
                              >
                                {line.path}
                              </span>
                            )}

                            {line.message && (
                              <span className="text-muted-foreground break-words max-md:basis-full">
                                {line.message}
                              </span>
                            )}
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            ) : logState.buffer.length > 0 ? (
              <EmptyState
                title={t('logs.search_empty_title')}
                description={t('logs.search_empty_desc')}
              />
            ) : (
              <EmptyState title={t('logs.empty_title')} description={t('logs.empty_desc')} />
            )}
          </Card>
        )}

        {activeTab === 'errors' && (
          <Card
            extra={
              <Button
                variant="secondary"
                size="sm"
                onClick={loadErrorLogs}
                loading={loadingErrors}
                disabled={disableControls}
              >
                {t('common.refresh')}
              </Button>
            }
          >
            <div className="stack">
              <div className="text-[13px] text-muted-foreground leading-[1.55]">{t('logs.error_logs_description')}</div>

              {requestLogEnabled && (
                <div>
                  <div className="inline-flex items-center text-[0.8125rem] font-medium px-[10px] py-[2px] border rounded-sm text-amber-700 bg-amber-100 border-amber-400/30 leading-[1.5]">{t('logs.error_logs_request_log_enabled')}</div>
                </div>
              )}

              {errorLogsError && <div className="p-[10px_14px] mb-2 bg-destructive/10 border border-destructive/35 text-destructive text-sm leading-[1.5]">{errorLogsError}</div>}

              {/* errorPanel: h-[480px], overflow-auto; compact viewport: h-[360px]/h-[280px] */}
              <div className="h-[480px] overflow-auto [webkit-overflow-scrolling:touch] overscroll-contain max-[820px]:h-[360px] max-[600px]:h-[280px]">
                {loadingErrors ? (
                  <div className="text-[13px] text-muted-foreground leading-[1.55]">{t('common.loading')}</div>
                ) : errorLogs.length === 0 ? (
                  <div className="text-[13px] text-muted-foreground leading-[1.55]">{t('logs.error_logs_empty')}</div>
                ) : (
                  <div className="flex flex-col">
                    {errorLogs.map((item) => (
                      <div key={item.name} className="flex items-center justify-between gap-2 py-2.5 border-b border-border last:border-b-0">
                        <div className="flex flex-col gap-0.5 min-w-0 flex-1">
                          <div className="text-sm font-medium text-foreground">{item.name}</div>
                          <div className="text-xs text-muted-foreground">
                            {item.size ? `${(item.size / 1024).toFixed(1)} KB` : ''}{' '}
                            {item.modified ? formatUnixTimestamp(item.modified) : ''}
                          </div>
                        </div>
                        <div className="flex items-center gap-1.5 shrink-0">
                          <Button
                            variant="secondary"
                            size="sm"
                            onClick={() => downloadErrorLog(item.name)}
                            disabled={disableControls}
                          >
                            {t('logs.error_logs_download')}
                          </Button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </Card>
        )}
      </div>

      <Modal
        open={Boolean(requestLogId)}
        onClose={closeRequestLogModal}
        title={t('logs.request_log_download_title')}
        footer={
          <>
            <Button variant="secondary" onClick={closeRequestLogModal} disabled={requestLogDownloading}>
              {t('common.cancel')}
            </Button>
            <Button
              onClick={() => {
                if (requestLogId) {
                  void downloadRequestLog(requestLogId);
                }
              }}
              loading={requestLogDownloading}
              disabled={!requestLogId}
            >
              {t('common.confirm')}
            </Button>
          </>
        }
      >
        {requestLogId ? t('logs.request_log_download_confirm', { id: requestLogId }) : null}
      </Modal>
    </div>
  );
}
