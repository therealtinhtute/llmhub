import { Suspense, lazy, useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { createPortal } from 'react-dom';
import type { ReactCodeMirrorRef } from '@uiw/react-codemirror';
import { parse as parseYaml, parseDocument } from 'yaml';
import { Button } from '@/components/ui/Button';
import { FormInput as Input } from '@/components/ui/FormInput';
import {
  IconCheck,
  IconChevronDown,
  IconChevronUp,
  IconRefreshCw,
  IconSearch,
} from '@/components/ui/icons';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { VisualConfigEditor } from '@/components/config/VisualConfigEditor';
import { DiffModal } from '@/components/config/DiffModal';
import { useMediaQuery } from '@/hooks/useMediaQuery';
import { useUnsavedChangesGuard } from '@/hooks/useUnsavedChangesGuard';
import { useVisualConfig } from '@/hooks/useVisualConfig';
import { toast } from 'sonner';
import { useConfirmationStore, useAuthStore, useThemeStore, useConfigStore } from '@/stores';
import { configFileApi } from '@/services/api/configFile';

type ConfigEditorTab = 'visual' | 'source';

const LazyConfigSourceEditor = lazy(() => import('@/components/config/ConfigSourceEditor'));

function readCommercialModeFromYaml(yamlContent: string): boolean {
  try {
    const parsed = parseYaml(yamlContent);
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return false;
    return Boolean((parsed as Record<string, unknown>)['commercial-mode']);
  } catch {
    return false;
  }
}

function normalizeYamlForVisualDiff(yamlContent: string): string {
  try {
    const doc = parseDocument(yamlContent);
    return doc.toString({ indent: 2, lineWidth: 120, minContentWidth: 0 });
  } catch {
    return yamlContent;
  }
}

export function ConfigPage() {
  const { t } = useTranslation();
  const showConfirmation = useConfirmationStore((state) => state.showConfirmation);
  const connectionStatus = useAuthStore((state) => state.connectionStatus);
  const resolvedTheme = useThemeStore((state) => state.resolvedTheme);
  const isMobile = useMediaQuery('(max-width: 768px)');

  const {
    visualValues,
    visualDirty,
    visualParseError,
    visualValidationErrors,
    visualHasPayloadValidationErrors,
    loadVisualValuesFromYaml,
    applyVisualChangesToYaml,
    setVisualValues,
  } = useVisualConfig();

  const [activeTab, setActiveTab] = useState<ConfigEditorTab>(() => {
    const saved = localStorage.getItem('config-management:tab');
    if (saved === 'visual' || saved === 'source') return saved;
    return 'visual';
  });

  const [content, setContent] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [dirty, setDirty] = useState(false);
  const [diffModalOpen, setDiffModalOpen] = useState(false);
  const [serverYaml, setServerYaml] = useState('');
  const [mergedYaml, setMergedYaml] = useState('');

  // Search state
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<{ current: number; total: number }>({
    current: 0,
    total: 0,
  });
  const [lastSearchedQuery, setLastSearchedQuery] = useState('');
  const editorRef = useRef<ReactCodeMirrorRef | null>(null);
  const floatingActionsRef = useRef<HTMLDivElement>(null);

  const disableControls = connectionStatus !== 'connected';
  const isDirty = dirty || visualDirty;

  const { allowNextNavigation } = useUnsavedChangesGuard({
    shouldBlock: isDirty,
    dialog: {
      title: t('config_management.unsaved_dialog_title'),
      message: t('config_management.unsaved_dialog_message'),
      confirmText: t('config_management.unsaved_dialog_confirm'),
      cancelText: t('config_management.unsaved_dialog_cancel'),
      variant: 'danger',
    },
  });

  const hasVisualModeError = !!visualParseError;
  const hasVisualValidationErrors =
    activeTab === 'visual' &&
    (Object.values(visualValidationErrors).some(Boolean) || visualHasPayloadValidationErrors);

  const loadConfig = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await configFileApi.fetchConfigYaml();
      setContent(data);
      setDirty(false);
      setDiffModalOpen(false);
      setServerYaml(data);
      setMergedYaml(data);
      loadVisualValuesFromYaml(data);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : t('notification.refresh_failed');
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [loadVisualValuesFromYaml, t]);

  useEffect(() => {
    loadConfig();
  }, [loadConfig]);

  useEffect(() => {
    if (activeTab !== 'visual' || !visualParseError) return;

    setActiveTab('source');
    localStorage.setItem('config-management:tab', 'source');
    toast.error(
      t('config_management.visual_mode_unavailable_detail', { message: visualParseError })
    );
  }, [activeTab, t, visualParseError]);

  const handleConfirmSave = async () => {
    setSaving(true);
    try {
      const previousCommercialMode = readCommercialModeFromYaml(serverYaml);
      const nextCommercialMode = readCommercialModeFromYaml(mergedYaml);
      const commercialModeChanged = previousCommercialMode !== nextCommercialMode;

      await configFileApi.saveConfigYaml(mergedYaml);
      const latestContent = await configFileApi.fetchConfigYaml();
      setDirty(false);
      setDiffModalOpen(false);
      setContent(latestContent);
      setServerYaml(latestContent);
      setMergedYaml(latestContent);
      loadVisualValuesFromYaml(latestContent);

      // Keep the global config store in sync so sidebar / other pages reflect YAML changes immediately.
      try {
        useConfigStore.getState().clearCache();
        await useConfigStore.getState().fetchConfig(undefined, true);
      } catch (refreshError: unknown) {
        const message =
          refreshError instanceof Error
            ? refreshError.message
            : typeof refreshError === 'string'
              ? refreshError
              : '';
        toast.error(`${t('notification.refresh_failed')}${message ? `: ${message}` : ''}`);
      }

      toast.success(t('config_management.save_success'));
      allowNextNavigation();
      if (commercialModeChanged) {
        toast.warning(t('notification.commercial_mode_restart_required'));
      }
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : '';
      toast.error(`${t('notification.save_failed')}: ${message}`);
    } finally {
      setSaving(false);
    }
  };

  const handleSave = async () => {
    if (activeTab === 'visual' && visualParseError) {
      toast.error(t('config_management.visual_mode_save_blocked'));
      return;
    }

    setSaving(true);
    try {
      const latestServerYaml = await configFileApi.fetchConfigYaml();

      const visualBaseYaml = dirty ? content : latestServerYaml;

      if (activeTab !== 'source') {
        const latestDocument = parseDocument(latestServerYaml);
        if (latestDocument.errors.length > 0) {
          toast.error(
            t('config_management.visual_mode_latest_yaml_invalid', {
              message:
                latestDocument.errors[0]?.message ??
                t('config_management.visual_mode_save_blocked'),
            })
          );
          return;
        }

        if (visualBaseYaml !== latestServerYaml) {
          const visualBaseDocument = parseDocument(visualBaseYaml);
          if (visualBaseDocument.errors.length > 0) {
            toast.error(
              t('config_management.visual_mode_latest_yaml_invalid', {
                message:
                  visualBaseDocument.errors[0]?.message ??
                  t('config_management.visual_mode_save_blocked'),
              })
            );
            return;
          }
        }
      }

      // In source mode, save exactly what the user edited. In visual mode, preserve the
      // local source draft when it has unsaved edits so source-only backend fields are not dropped.
      const nextMergedYaml =
        activeTab === 'source' ? content : applyVisualChangesToYaml(visualBaseYaml);

      // In visual mode, applyVisualChangesToYaml re-serializes YAML via parseDocument → toString,
      // which may reformat comments/whitespace. Normalize the server YAML through the same pipeline
      // so the diff only shows actual value changes, not cosmetic reformatting.
      let diffOriginal = latestServerYaml;
      if (activeTab !== 'source') {
        diffOriginal = normalizeYamlForVisualDiff(latestServerYaml);
      }

      if (diffOriginal === nextMergedYaml) {
        setDirty(false);
        setContent(latestServerYaml);
        setServerYaml(latestServerYaml);
        setMergedYaml(nextMergedYaml);
        loadVisualValuesFromYaml(latestServerYaml);
        toast.info(t('config_management.diff.no_changes'));
        return;
      }

      setServerYaml(diffOriginal);
      setMergedYaml(nextMergedYaml);
      setDiffModalOpen(true);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : '';
      toast.error(`${t('notification.save_failed')}: ${message}`);
    } finally {
      setSaving(false);
    }
  };

  const handleChange = useCallback((value: string) => {
    setContent(value);
    setDirty(true);
  }, []);

  const handleTabChange = useCallback(
    (tab: ConfigEditorTab) => {
      if (tab === activeTab) return;

      if (tab === 'source') {
        // Only rewrite YAML when there are pending visual changes; otherwise preserve raw YAML + comments.
        if (visualDirty) {
          const nextContent = applyVisualChangesToYaml(content);
          if (nextContent !== content) {
            setContent(nextContent);
            setDirty(true);
          }
        }
      } else {
        const result = loadVisualValuesFromYaml(content);
        if (!result.ok) {
          toast.error(
            t('config_management.visual_mode_unavailable_detail', { message: result.error })
          );
          return;
        }
      }

      setActiveTab(tab);
      localStorage.setItem('config-management:tab', tab);
    },
    [activeTab, applyVisualChangesToYaml, content, loadVisualValuesFromYaml, t, visualDirty]
  );

  // Search functionality
  const performSearch = useCallback((query: string, direction: 'next' | 'prev' = 'next') => {
    if (!query || !editorRef.current?.view) return;

    const view = editorRef.current.view;
    const doc = view.state.doc.toString();
    const matches: number[] = [];
    const lowerQuery = query.toLowerCase();
    const lowerDoc = doc.toLowerCase();

    let pos = 0;
    while (pos < lowerDoc.length) {
      const index = lowerDoc.indexOf(lowerQuery, pos);
      if (index === -1) break;
      matches.push(index);
      pos = index + 1;
    }

    if (matches.length === 0) {
      setSearchResults({ current: 0, total: 0 });
      return;
    }

    // Find current match based on cursor position
    const selection = view.state.selection.main;
    const cursorPos = direction === 'prev' ? selection.from : selection.to;
    let currentIndex = 0;

    if (direction === 'next') {
      // Find next match after cursor
      for (let i = 0; i < matches.length; i++) {
        if (matches[i] > cursorPos) {
          currentIndex = i;
          break;
        }
        // If no match after cursor, wrap to first
        if (i === matches.length - 1) {
          currentIndex = 0;
        }
      }
    } else {
      // Find previous match before cursor
      for (let i = matches.length - 1; i >= 0; i--) {
        if (matches[i] < cursorPos) {
          currentIndex = i;
          break;
        }
        // If no match before cursor, wrap to last
        if (i === 0) {
          currentIndex = matches.length - 1;
        }
      }
    }

    const matchPos = matches[currentIndex];
    setSearchResults({ current: currentIndex + 1, total: matches.length });

    // Scroll to and select the match
    view.dispatch({
      selection: { anchor: matchPos, head: matchPos + query.length },
      scrollIntoView: true,
    });
    view.focus();
  }, []);

  const handleSearchChange = useCallback((value: string) => {
    setSearchQuery(value);
    // Do not auto-search on each keystroke. Clear previous results when query changes.
    if (!value) {
      setSearchResults({ current: 0, total: 0 });
      setLastSearchedQuery('');
    } else {
      setSearchResults({ current: 0, total: 0 });
    }
  }, []);

  const executeSearch = useCallback(
    (direction: 'next' | 'prev' = 'next') => {
      if (!searchQuery) return;
      setLastSearchedQuery(searchQuery);
      performSearch(searchQuery, direction);
    },
    [searchQuery, performSearch]
  );

  const handleSearchKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        executeSearch(e.shiftKey ? 'prev' : 'next');
      }
    },
    [executeSearch]
  );

  const handlePrevMatch = useCallback(() => {
    if (!lastSearchedQuery) return;
    performSearch(lastSearchedQuery, 'prev');
  }, [lastSearchedQuery, performSearch]);

  const handleNextMatch = useCallback(() => {
    if (!lastSearchedQuery) return;
    performSearch(lastSearchedQuery, 'next');
  }, [lastSearchedQuery, performSearch]);

  // Keep bottom floating actions from covering page content by syncing its height to a CSS variable.
  useLayoutEffect(() => {
    if (typeof window === 'undefined') return;

    const actionsEl = floatingActionsRef.current;
    if (!actionsEl) return;

    const updatePadding = () => {
      const height = actionsEl.getBoundingClientRect().height;
      document.documentElement.style.setProperty('--config-action-bar-height', `${height}px`);
    };

    updatePadding();
    window.addEventListener('resize', updatePadding);

    const ro = typeof ResizeObserver === 'undefined' ? null : new ResizeObserver(updatePadding);
    ro?.observe(actionsEl);

    return () => {
      ro?.disconnect();
      window.removeEventListener('resize', updatePadding);
      document.documentElement.style.removeProperty('--config-action-bar-height');
    };
  }, []);

  // Status text
  const getStatusText = () => {
    if (disableControls) return t('config_management.status_disconnected');
    if (loading) return t('config_management.status_loading');
    if (error) return t('config_management.status_load_failed');
    if (hasVisualModeError) return t('config_management.visual_mode_unavailable');
    if (hasVisualValidationErrors)
      return t('config_management.visual.validation.validation_blocked');
    if (saving) return t('config_management.status_saving');
    if (isDirty) return t('config_management.status_dirty');
    return t('config_management.status_loaded');
  };

  const getStatusClass = () => {
    if (error || hasVisualModeError || hasVisualValidationErrors)
      return 'text-amber-700 bg-amber-100 border-amber-400/30';
    if (isDirty) return 'text-amber-700 bg-amber-100 border-amber-400/30';
    if (!loading && !saving) return 'text-emerald-700 bg-emerald-100/10 border-emerald-500/34';
    return '';
  };

  const getFloatingStatusText = () => {
    if (!isMobile) return getStatusText();
    if (disableControls)
      return t('config_management.status_disconnected_short', { defaultValue: 'Disconnected' });
    if (loading) return t('config_management.status_loading_short', { defaultValue: 'Loading' });
    if (error) return t('config_management.status_load_failed_short', { defaultValue: 'Failed' });
    if (hasVisualModeError)
      return t('config_management.visual_mode_unavailable_short', { defaultValue: 'YAML issue' });
    if (hasVisualValidationErrors)
      return t('config_management.visual.validation_blocked_short', { defaultValue: 'Fix errors' });
    if (saving) return t('config_management.status_saving_short', { defaultValue: 'Saving' });
    if (isDirty) return t('config_management.status_dirty_short', { defaultValue: 'Unsaved' });
    return t('config_management.status_loaded_short', { defaultValue: 'Loaded' });
  };

  const handleReload = useCallback(() => {
    if (!isDirty) {
      void loadConfig();
      return;
    }

    showConfirmation({
      title: t('common.unsaved_changes_title'),
      message: t('config_management.reload_confirm_message'),
      confirmText: t('config_management.reload'),
      cancelText: t('common.cancel'),
      variant: 'danger',
      onConfirm: async () => {
        await loadConfig();
      },
    });
  }, [isDirty, loadConfig, showConfirmation, t]);

  const floatingActions = (
    <div
      className="fixed left-[var(--content-center-x,50%)] bottom-[calc(16px+env(safe-area-inset-bottom))] -translate-x-1/2 z-50 pointer-events-auto w-fit max-w-[calc(100vw-24px)]"
      ref={floatingActionsRef}
    >
      <div className="inline-flex items-center gap-1.5 p-1.5 max-w-[inherit] overflow-x-auto border border-border bg-[color-mix(in_srgb,hsl(var(--background))_92%,transparent)] shadow-lg scrollbar-none [&::-webkit-scrollbar]:hidden">
        <div
          className={`inline-flex items-center min-w-0 min-h-[34px] px-2.5 border text-foreground text-[11px] font-bold leading-tight text-center overflow-hidden text-ellipsis whitespace-nowrap ${
            isMobile ? 'max-w-[112px] px-2 text-[10px]' : 'max-w-[min(300px,46vw)]'
          } ${getStatusClass()}`}
        >
          {getFloatingStatusText()}
        </div>
        <button
          type="button"
          className="relative flex items-center justify-center w-[38px] h-[38px] text-foreground cursor-pointer hover:not-disabled:bg-foreground hover:not-disabled:text-background disabled:opacity-45 disabled:cursor-not-allowed bg-transparent border-0 p-0 m-0 appearance-none"
          onClick={handleReload}
          disabled={loading || saving}
          title={t('config_management.reload')}
          aria-label={t('config_management.reload')}
        >
          <IconRefreshCw size={16} />
        </button>
        <button
          type="button"
          className="relative flex items-center justify-center w-[38px] h-[38px] text-foreground cursor-pointer hover:not-disabled:bg-foreground hover:not-disabled:text-background disabled:opacity-45 disabled:cursor-not-allowed bg-transparent border-0 p-0 m-0 appearance-none"
          onClick={handleSave}
          disabled={
            disableControls ||
            loading ||
            saving ||
            !isDirty ||
            diffModalOpen ||
            hasVisualModeError ||
            hasVisualValidationErrors
          }
          title={t('config_management.save')}
          aria-label={t('config_management.save')}
        >
          <IconCheck size={16} />
          {isDirty && (
            <span
              className="absolute top-2 right-2 w-[7px] h-[7px] rounded-full bg-amber-400"
              aria-hidden="true"
            />
          )}
        </button>
      </div>
    </div>
  );

  return (
    <div className="w-[min(100%,1480px)] min-h-full flex flex-col gap-[6px] mx-auto overflow-y-auto pb-[calc(var(--config-action-bar-height,0px)+16px+env(safe-area-inset-bottom)+12px)]">
      <div className="flex items-start">
        <div className="flex flex-col gap-[10px] min-w-0 w-[min(100%,360px)]">
          <h1 className="text-[28px] font-bold text-foreground m-0">
            {t('config_management.title')}
          </h1>
          <Tabs value={activeTab} onValueChange={(v) => handleTabChange(v as ConfigEditorTab)}>
            <TabsList className="flex justify-start items-start gap-0 p-0 border-b border-border bg-transparent">
              <TabsTrigger
                value="visual"
                disabled={saving || loading}
                className="px-3 text-muted-foreground border-b-4 border-transparent transition-colors duration-150 data-[state=active]:border-primary data-[state=active]:text-primary hover:text-foreground"
              >
                {t('config_management.tabs.visual', { defaultValue: '可视化编辑' })}
              </TabsTrigger>
              <TabsTrigger
                value="source"
                disabled={saving || loading}
                className="px-3 text-muted-foreground border-b-4 border-transparent transition-colors duration-150 data-[state=active]:border-primary data-[state=active]:text-primary hover:text-foreground"
              >
                {t('config_management.tabs.source', { defaultValue: '源代码编辑' })}
              </TabsTrigger>
            </TabsList>
          </Tabs>
        </div>
      </div>

      <div className="flex flex-col gap-4 min-w-0">
        <div className="flex flex-col gap-4 min-h-0">
          {error && (
            <div className="p-[10px_14px] mb-2 bg-destructive/10 border border-destructive/35 text-destructive text-sm leading-[1.5]">
              {error}
            </div>
          )}
          {!error && visualParseError && (
            <div className="p-[10px_14px] mb-2 bg-destructive/10 border border-destructive/35 text-destructive text-sm leading-[1.5]">
              {t('config_management.visual_mode_unavailable_detail', { message: visualParseError })}
            </div>
          )}

          {activeTab === 'visual' ? (
            <VisualConfigEditor
              values={visualValues}
              validationErrors={visualValidationErrors}
              hasPayloadValidationErrors={visualHasPayloadValidationErrors}
              disabled={disableControls || loading}
              onChange={setVisualValues}
            />
          ) : (
            <div className="flex flex-col gap-3 min-h-0">
              <div className="grid grid-cols-[minmax(0,1fr)_auto] max-md:grid-cols-[minmax(0,1fr)] gap-2 items-center p-2 border border-border bg-[color-mix(in_srgb,hsl(var(--background))_76%,transparent)]">
                <div className="min-w-0 relative flex items-center">
                  <Input
                    value={searchQuery}
                    onChange={(e) => handleSearchChange(e.target.value)}
                    onKeyDown={handleSearchKeyDown}
                    placeholder={t('config_management.search_placeholder', {
                      defaultValue: '搜索配置内容...',
                    })}
                    disabled={disableControls || loading}
                    className="min-h-[38px]! pr-[128px]! bg-muted!"
                    rightElement={
                      <div className="inline-flex items-center gap-1.5">
                        {searchQuery && lastSearchedQuery === searchQuery && (
                          <span className="inline-flex items-center min-h-[26px] px-2 border border-border text-muted-foreground text-[12px] font-[650] pointer-events-none whitespace-nowrap">
                            {searchResults.total > 0
                              ? `${searchResults.current} / ${searchResults.total}`
                              : t('config_management.search_no_results', {
                                  defaultValue: '无结果',
                                })}
                          </span>
                        )}
                        <button
                          type="button"
                          className="inline-flex items-center justify-center w-[30px] h-[30px] border border-foreground bg-foreground text-background hover:not-disabled:bg-[var(--primary-hover)] hover:not-disabled:border-[var(--primary-hover)] disabled:opacity-45 disabled:cursor-not-allowed p-0 m-0 appearance-none cursor-pointer"
                          onClick={() => executeSearch('next')}
                          disabled={!searchQuery || disableControls || loading}
                          title={t('config_management.search_button', { defaultValue: '搜索' })}
                        >
                          <IconSearch size={16} />
                        </button>
                      </div>
                    }
                  />
                </div>

                <div className="flex gap-1.5 flex-shrink-0 max-md:justify-stretch [&_button]:min-w-[38px] [&_button]:w-[38px] [&_button]:h-[38px] [&_button]:p-0! max-md:[&_button]:flex-1 max-md:[&_button]:w-auto">
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={handlePrevMatch}
                    disabled={
                      !searchQuery || lastSearchedQuery !== searchQuery || searchResults.total === 0
                    }
                    title={t('config_management.search_prev', { defaultValue: '上一个' })}
                  >
                    <IconChevronUp size={16} />
                  </Button>
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={handleNextMatch}
                    disabled={
                      !searchQuery || lastSearchedQuery !== searchQuery || searchResults.total === 0
                    }
                    title={t('config_management.search_next', { defaultValue: '下一个' })}
                  >
                    <IconChevronDown size={16} />
                  </Button>
                </div>
              </div>

              <div className="w-full flex-none h-[clamp(500px,70vh,1040px)] supports-[height:100dvh]:h-[clamp(500px,70dvh,1040px)] overflow-hidden bg-background [&_.cm-editor]:h-full [&_.cm-editor]:text-[13px] [&_.cm-editor]:font-mono [&_.cm-editor]:bg-transparent [&_.cm-scroller]:overflow-auto [&_.cm-scroller]:[-webkit-overflow-scrolling:touch] [&_.cm-scroller]:[touch-action:pan-x_pan-y] [&_.cm-scroller]:[overscroll-behavior:contain] [&_.cm-gutters]:border-r [&_.cm-gutters]:border-border [&_.cm-gutters]:bg-[color-mix(in_srgb,hsl(var(--muted))_86%,transparent)] [&_.cm-lineNumbers_.cm-gutterElement]:px-2 [&_.cm-lineNumbers_.cm-gutterElement]:min-w-[40px] [&_.cm-lineNumbers_.cm-gutterElement]:text-muted-foreground [&_.cm-activeLine]:bg-foreground/[0.05] [&_.cm-activeLineGutter]:bg-foreground/[0.05] [&_.cm-selectionMatch]:bg-[rgba(224,170,20,0.24)] [&_.cm-searchMatch]:bg-[rgba(224,170,20,0.32)] [&_.cm-searchMatch]:outline [&_.cm-searchMatch]:outline-1 [&_.cm-searchMatch]:outline-[rgba(224,170,20,0.48)] [&_.cm-searchMatch-selected]:bg-[rgba(198,87,70,0.32)]">
                <Suspense fallback={null}>
                  <LazyConfigSourceEditor
                    editorRef={editorRef}
                    value={content}
                    onChange={handleChange}
                    theme={resolvedTheme}
                    editable={!disableControls && !loading}
                    placeholder={t('config_management.editor_placeholder')}
                  />
                </Suspense>
              </div>
            </div>
          )}
        </div>
      </div>

      {typeof document !== 'undefined' ? createPortal(floatingActions, document.body) : null}
      <DiffModal
        open={diffModalOpen}
        original={serverYaml}
        modified={mergedYaml}
        onConfirm={handleConfirmSave}
        onCancel={() => setDiffModalOpen(false)}
        loading={saving}
      />
    </div>
  );
}
