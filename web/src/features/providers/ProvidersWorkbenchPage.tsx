import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useHeaderRefresh } from '@/hooks/useHeaderRefresh';
import { useUnsavedChangesGuard } from '@/hooks/useUnsavedChangesGuard';
import { AppSkeleton as Skeleton } from '@/components/ui/AppSkeleton';
import { toast } from 'sonner';
import { useAuthStore } from '@/stores';
import { useProviderRecentRequests } from '@/components/providers/hooks/useProviderRecentRequests';
import { authFilesApi, providersApi } from '@/services/api';
import { oauthApi, type OAuthProvider } from '@/services/api/oauth';
import { copyToClipboard } from '@/utils/clipboard';
import type { AuthFileItem, ProviderPreset } from '@/types';
import { ProviderHeaderCard } from './components/ProviderHeaderCard';
import { ProviderCategoryGrid } from './components/ProviderCategoryGrid';
import {
  buildEntries,
  getAuthKey,
  PROVIDERS,
  resolveCallbackUrl,
  type ProviderEntry,
  type ProviderOAuthState,
} from './entries';
import {
  ProviderSheet,
  type ProviderSheetHandle,
  type ProviderSheetState,
} from './sheets/ProviderSheet';
import { useProviderWorkbench } from './useProviderWorkbench';
import type { ProviderResource } from './types';

const SUCCESS_RESET_DELAY_MS = 5000;

function getDefaultSheetState(entry: ProviderEntry): ProviderSheetState {
  if (entry.kind === 'oauth') {
    return {
      open: true,
      entryKey: entry.key,
      brand: 'gemini',
      mode: 'oauth',
      resourceId: null,
    };
  }
  if (entry.kind === 'preset') {
    return {
      open: true,
      entryKey: entry.key,
      brand: 'openaiCompatibility',
      mode: 'list',
      resourceId: null,
    };
  }
  if (entry.group.id === 'ampcode') {
    return {
      open: true,
      entryKey: entry.key,
      brand: 'ampcode',
      mode: 'edit',
      resourceId: entry.resources[0]?.id ?? null,
    };
  }
  return {
    open: true,
    entryKey: entry.key,
    brand: entry.group.id,
    mode: 'list',
    resourceId: null,
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object';
}

function getErrorMessage(error: unknown): string {
  if (error instanceof Error) return error.message;
  if (isRecord(error) && typeof error.message === 'string') return error.message;
  return typeof error === 'string' ? error : '';
}

function getErrorStatus(error: unknown): number | undefined {
  if (!isRecord(error)) return undefined;
  return typeof error.status === 'number' ? error.status : undefined;
}

const formatDateTime = (iso: string, locale?: string) => {
  try {
    const date = new Date(iso);
    if (Number.isNaN(date.getTime())) return iso;
    return new Intl.DateTimeFormat(locale, {
      dateStyle: 'medium',
      timeStyle: 'short',
    }).format(date);
  } catch {
    return iso;
  }
};

export function ProvidersWorkbenchPage() {
  const { t, i18n } = useTranslation();
  const connectionStatus = useAuthStore((s) => s.connectionStatus);

  const workbench = useProviderWorkbench();
  const [searchParams, setSearchParams] = useSearchParams();
  const requestedEntryKey = searchParams.get('entry')?.trim() || null;
  const hasEntryParam = searchParams.has('entry');
  const [sheetState, setSheetState] = useState<ProviderSheetState>({
    open: false,
    entryKey: null,
    brand: 'gemini',
    mode: 'list',
    resourceId: null,
  });
  const [sheetDirty, setSheetDirty] = useState(false);
  const sheetRef = useRef<ProviderSheetHandle>(null);

  const [presets, setPresets] = useState<ProviderPreset[]>([]);
  const [authFiles, setAuthFiles] = useState<AuthFileItem[]>([]);
  const [authFilesRevision, setAuthFilesRevision] = useState(0);
  const [presetsLoaded, setPresetsLoaded] = useState(false);
  const [authFilesLoaded, setAuthFilesLoaded] = useState(false);
  const [oauthStates, setOauthStates] = useState<
    Partial<Record<OAuthProvider, ProviderOAuthState>>
  >({});
  const pollingTimers = useRef<Partial<Record<OAuthProvider, number>>>({});
  const successResetTimers = useRef<Partial<Record<OAuthProvider, number>>>({});

  const refreshAuthFiles = useCallback(async () => {
    try {
      const data = await authFilesApi.list();
      setAuthFiles(data?.files ?? []);
      setAuthFilesRevision((revision) => revision + 1);
    } catch {
      // Preserve the last known account list when a refresh fails.
    } finally {
      setAuthFilesLoaded(true);
    }
  }, []);

  const setEntryQuery = useCallback(
    (entryKey: string | null, replace = false) => {
      setSearchParams(entryKey ? { entry: entryKey } : {}, { replace });
    },
    [setSearchParams]
  );

  useEffect(() => {
    providersApi
      .getProviderPresets()
      .then(setPresets)
      .catch(() => setPresets([]))
      .finally(() => setPresetsLoaded(true));
    void refreshAuthFiles();
  }, [refreshAuthFiles]);

  const clearTimers = useCallback(() => {
    Object.values(pollingTimers.current).forEach((timer) => {
      if (timer !== undefined) window.clearInterval(timer);
    });
    Object.values(successResetTimers.current).forEach((timer) => {
      if (timer !== undefined) window.clearTimeout(timer);
    });
    pollingTimers.current = {};
    successResetTimers.current = {};
  }, []);

  useEffect(() => clearTimers, [clearTimers]);

  const updateProviderState = useCallback(
    (provider: OAuthProvider, next: Partial<ProviderOAuthState>) => {
      setOauthStates((previous) => ({
        ...previous,
        [provider]: { ...(previous[provider] ?? {}), ...next },
      }));
    },
    []
  );

  const clearPollingTimer = useCallback((provider: OAuthProvider) => {
    const timer = pollingTimers.current[provider];
    if (timer !== undefined) {
      window.clearInterval(timer);
      delete pollingTimers.current[provider];
    }
  }, []);

  const clearSuccessResetTimer = useCallback((provider: OAuthProvider) => {
    const timer = successResetTimers.current[provider];
    if (timer !== undefined) {
      window.clearTimeout(timer);
      delete successResetTimers.current[provider];
    }
  }, []);

  const clearProviderTimers = useCallback(
    (provider: OAuthProvider) => {
      clearPollingTimer(provider);
      clearSuccessResetTimer(provider);
    },
    [clearPollingTimer, clearSuccessResetTimer]
  );

  const resetProviderAttempt = useCallback(
    (provider: OAuthProvider) => {
      clearProviderTimers(provider);
      setOauthStates((previous) => ({ ...previous, [provider]: {} }));
    },
    [clearProviderTimers]
  );

  const completeProviderAuth = useCallback(
    (provider: OAuthProvider) => {
      clearPollingTimer(provider);
      clearSuccessResetTimer(provider);
      void refreshAuthFiles();
      updateProviderState(provider, {
        url: undefined,
        state: undefined,
        status: 'success',
        error: undefined,
        polling: false,
        callbackSubmitting: false,
        callbackStatus: undefined,
        callbackError: undefined,
      });
      successResetTimers.current[provider] = window.setTimeout(() => {
        resetProviderAttempt(provider);
      }, SUCCESS_RESET_DELAY_MS);
    },
    [
      clearPollingTimer,
      clearSuccessResetTimer,
      refreshAuthFiles,
      resetProviderAttempt,
      updateProviderState,
    ]
  );

  const startPolling = useCallback(
    (provider: OAuthProvider, state: string) => {
      clearPollingTimer(provider);
      const timer = window.setInterval(async () => {
        try {
          const response = await oauthApi.getAuthStatus(state);
          if (response.status === 'ok') {
            completeProviderAuth(provider);
            toast.success(t(getAuthKey(provider, 'oauth_status_success')));
          } else if (response.status === 'error') {
            updateProviderState(provider, {
              status: 'error',
              error: response.error,
              polling: false,
            });
            toast.error(
              `${t(getAuthKey(provider, 'oauth_status_error'))} ${response.error || ''}`
            );
            window.clearInterval(timer);
            delete pollingTimers.current[provider];
          }
        } catch (error: unknown) {
          updateProviderState(provider, {
            status: 'error',
            error: getErrorMessage(error),
            polling: false,
          });
          window.clearInterval(timer);
          delete pollingTimers.current[provider];
        }
      }, 3000);
      pollingTimers.current[provider] = timer;
    },
    [clearPollingTimer, completeProviderAuth, t, updateProviderState]
  );

  const startOAuth = useCallback(
    async (provider: OAuthProvider, projectId?: string) => {
      clearProviderTimers(provider);
      const rawProjectId = provider === 'gemini-cli' ? (projectId ?? '').trim() : '';
      const normalizedProjectId = rawProjectId
        ? rawProjectId.toUpperCase() === 'ALL'
          ? 'ALL'
          : rawProjectId
        : undefined;

      updateProviderState(provider, {
        url: undefined,
        state: undefined,
        status: 'waiting',
        polling: true,
        error: undefined,
        callbackStatus: undefined,
        callbackError: undefined,
      });
      try {
        const response = await oauthApi.startAuth(
          provider,
          provider === 'gemini-cli'
            ? { projectId: normalizedProjectId }
            : undefined
        );
        if (!response.state) {
          const message = t('auth_login.missing_state');
          updateProviderState(provider, {
            url: response.url,
            state: undefined,
            status: 'error',
            error: message,
            polling: false,
          });
          toast.error(message);
          return;
        }
        updateProviderState(provider, {
          url: response.url,
          state: response.state,
          status: 'waiting',
          polling: true,
        });
        startPolling(provider, response.state);
      } catch (error: unknown) {
        const message = getErrorMessage(error);
        updateProviderState(provider, {
          status: 'error',
          error: message,
          polling: false,
        });
        toast.error(
          `${t(getAuthKey(provider, 'oauth_start_error'))}${message ? ` ${message}` : ''}`
        );
      }
    },
    [clearProviderTimers, startPolling, t, updateProviderState]
  );

  const copyOAuthLink = useCallback(
    async (url?: string) => {
      if (!url) return;
      const copied = await copyToClipboard(url);
      if (copied) {
        toast.success(t('notification.link_copied'));
      } else {
        toast.error(t('notification.copy_failed'));
      }
    },
    [t]
  );

  const submitOAuthCallback = useCallback(
    async (provider: OAuthProvider, callbackInput: string) => {
      const input = callbackInput.trim();
      if (!input) {
        toast.warning(
          t(
            provider === 'xai'
              ? 'auth_login.xai_callback_required'
              : 'auth_login.oauth_callback_required'
          )
        );
        return;
      }
      const redirectUrl = resolveCallbackUrl(provider, input, oauthStates[provider]?.state);
      if (!redirectUrl) {
        toast.warning(
          t(
            provider === 'xai'
              ? 'auth_login.xai_callback_state_missing'
              : 'auth_login.missing_state'
          )
        );
        return;
      }
      updateProviderState(provider, {
        callbackSubmitting: true,
        callbackStatus: undefined,
        callbackError: undefined,
      });
      try {
        await oauthApi.submitCallback(provider, redirectUrl);
        updateProviderState(provider, {
          callbackSubmitting: false,
          callbackStatus: 'success',
        });
        toast.success(t('auth_login.oauth_callback_success'));
      } catch (error: unknown) {
        const status = getErrorStatus(error);
        const message = getErrorMessage(error);
        const errorMessage =
          status === 404
            ? t('auth_login.oauth_callback_upgrade_hint')
            : message || undefined;
        updateProviderState(provider, {
          callbackSubmitting: false,
          callbackStatus: 'error',
          callbackError: errorMessage,
        });
        toast.error(
          errorMessage
            ? `${t('auth_login.oauth_callback_error')} ${errorMessage}`
            : t('auth_login.oauth_callback_error')
        );
      }
    },
    [oauthStates, t, updateProviderState]
  );

  const connected = connectionStatus === 'connected';
  const { usageByProvider, refreshRecentRequests } = useProviderRecentRequests({
    enabled: connected,
  });

  const handleRefresh = useCallback(async () => {
    await Promise.allSettled([
      workbench.refetch(),
      refreshRecentRequests().catch(() => undefined),
      refreshAuthFiles(),
    ]);
  }, [refreshAuthFiles, refreshRecentRequests, workbench]);

  useHeaderRefresh(handleRefresh);

  const disableMutations = connectionStatus !== 'connected' || workbench.mutating;
  const groups = useMemo(() => workbench.snapshot?.groups ?? [], [workbench.snapshot]);
  const entries = useMemo(
    () => buildEntries({ groups, presets, authFiles }),
    [groups, presets, authFiles]
  );
  const requestedEntry = useMemo(
    () =>
      requestedEntryKey
        ? entries.find((entry) => entry.key === requestedEntryKey) ?? null
        : null,
    [entries, requestedEntryKey]
  );
  const entriesReady = !workbench.isPending && presetsLoaded && authFilesLoaded;

  useEffect(() => {
    if (!hasEntryParam || !entriesReady) return;
    if (!requestedEntry) setEntryQuery(null, true);
  }, [entriesReady, hasEntryParam, requestedEntry, setEntryQuery]);

  const effectiveSheetState = useMemo<ProviderSheetState>(() => {
    if (!requestedEntry) {
      return {
        ...sheetState,
        open: false,
        entryKey: null,
        mode: 'list',
        resourceId: null,
      };
    }
    if (sheetState.entryKey === requestedEntry.key) {
      return { ...sheetState, open: true };
    }
    return getDefaultSheetState(requestedEntry);
  }, [requestedEntry, sheetState]);

  const { allowNextNavigation } = useUnsavedChangesGuard({
    enabled: effectiveSheetState.open && sheetDirty,
    shouldBlock: effectiveSheetState.open && sheetDirty,
    dialog: {
      title: t('providersPage.unsavedChanges.title'),
      message: t('providersPage.unsavedChanges.message'),
      confirmText: t('providersPage.unsavedChanges.discard'),
      cancelText: t('providersPage.unsavedChanges.keepEditing'),
      variant: 'danger',
    },
  });

  const oauthPendingStatus = useMemo(
    () =>
      Object.fromEntries(
        PROVIDERS.flatMap((provider) =>
          oauthStates[provider.id]?.status === 'waiting'
            ? [
                [
                  provider.id,
                  t('providersPage.card.pendingAuthorization'),
                ],
              ]
            : []
        )
      ) as Partial<Record<OAuthProvider, string>>,
    [oauthStates, t]
  );

  const totalResources = useMemo(
    () =>
      groups.reduce(
        (sum, group) =>
          sum + group.resources.filter((resource) => !resource.flags.isPlaceholder).length,
        0
      ),
    [groups]
  );

  const totalActive = useMemo(
    () =>
      groups.reduce(
        (sum, group) =>
          sum +
          group.resources.filter(
            (resource) => !resource.disabled && !resource.flags.isPlaceholder
          ).length,
        0
      ),
    [groups]
  );

  const providerFamilies = useMemo(
    () =>
      groups.filter((group) =>
        group.resources.some((resource) => !resource.flags.isPlaceholder)
      ).length,
    [groups]
  );

  const updatedAtLabel = workbench.snapshot
    ? formatDateTime(workbench.snapshot.fetchedAt, i18n.language)
    : t('providersPage.modelCatalog.notLoaded');

  const confirmSheetNavigation = useCallback(async () => {
    const ok = await (sheetRef.current?.confirmDiscardIfDirty() ?? Promise.resolve(true));
    if (ok) allowNextNavigation();
    return ok;
  }, [allowNextNavigation]);

  const openCreate = useCallback(() => {
    const entry = entries.find((candidate) => candidate.key === 'config:openaiCompatibility');
    if (!entry) return;
    void confirmSheetNavigation().then((ok) => {
      if (!ok) return;
      setSheetState({
        ...getDefaultSheetState(entry),
        mode: 'create',
        resourceId: null,
      });
      setEntryQuery(entry.key);
    });
  }, [confirmSheetNavigation, entries, setEntryQuery]);

  const closeSheet = useCallback(() => {
    allowNextNavigation();
    setSheetDirty(false);
    setSheetState({
      open: false,
      entryKey: null,
      brand: 'gemini',
      mode: 'list',
      resourceId: null,
    });
    setEntryQuery(null, true);
  }, [allowNextNavigation, setEntryQuery]);

  const handleOpenEntry = useCallback(
    (key: string) => {
      const entry = entries.find((candidate) => candidate.key === key);
      if (!entry) return;
      void confirmSheetNavigation().then((ok) => {
        if (!ok) return;
        setSheetState(getDefaultSheetState(entry));
        setEntryQuery(entry.key);
      });
    },
    [confirmSheetNavigation, entries, setEntryQuery]
  );

  const handleBackToList = useCallback(() => {
    setSheetState((previous) => ({
      ...previous,
      open: true,
      entryKey: requestedEntry?.key ?? previous.entryKey,
      mode: 'list',
      resourceId: null,
    }));
  }, [requestedEntry]);

  const handleViewResource = useCallback(
    (resource: ProviderResource) => {
      setSheetState((previous) => ({
        ...previous,
        open: true,
        entryKey: requestedEntry?.key ?? previous.entryKey,
        brand: resource.brand,
        mode: 'detail',
        resourceId: resource.id,
      }));
    },
    [requestedEntry]
  );

  const handleEditResource = useCallback(
    (resource: ProviderResource) => {
      setSheetState((previous) => ({
        ...previous,
        open: true,
        entryKey: requestedEntry?.key ?? previous.entryKey,
        brand: resource.brand,
        mode: 'edit',
        resourceId: resource.id,
      }));
    },
    [requestedEntry]
  );

  const handleCreateResource = useCallback(() => {
    setSheetState((previous) => ({
      ...previous,
      open: true,
      entryKey: requestedEntry?.key ?? previous.entryKey,
      mode: 'create',
      resourceId: null,
    }));
  }, [requestedEntry]);

  const handleCreated = useCallback(() => {
    toast.success(t('providersPage.toast.created'));
    const entryKey = requestedEntry?.key ?? sheetState.entryKey;
    const entry = entries.find((candidate) => candidate.key === entryKey);
    const canReturnToList =
      entry?.kind === 'preset' ||
      (entry?.kind === 'config' && entry.group.id !== 'ampcode');
    if (canReturnToList) {
      setSheetState((previous) => ({ ...previous, open: true, mode: 'list', resourceId: null }));
    } else {
      closeSheet();
    }
  }, [closeSheet, entries, requestedEntry, sheetState.entryKey, t]);

  const handleUpdated = useCallback(() => {
    toast.success(t('providersPage.toast.updated'));
    const entryKey = requestedEntry?.key ?? sheetState.entryKey;
    const entry = entries.find((candidate) => candidate.key === entryKey);
    const canReturnToList =
      entry?.kind === 'preset' ||
      (entry?.kind === 'config' && entry.group.id !== 'ampcode');
    if (canReturnToList) {
      setSheetState((previous) => ({ ...previous, open: true, mode: 'list', resourceId: null }));
    } else {
      closeSheet();
    }
  }, [closeSheet, entries, requestedEntry, sheetState.entryKey, t]);

  if (!workbench.snapshot && workbench.isPending) {
    return (
      <div className="flex flex-col gap-5 w-full p-6 box-border max-md:p-4 max-md:gap-4">
        <Skeleton height={120} />
        <Skeleton height={420} />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-5 w-full p-6 box-border max-md:p-4 max-md:gap-4">
      <ProviderHeaderCard
        totalActive={totalActive}
        totalResources={totalResources}
        providerFamilies={providerFamilies}
        updatedAtLabel={updatedAtLabel}
        issueCount={workbench.snapshot?.issues.length ?? 0}
        isFetching={workbench.isFetching}
        isNewDisabled={disableMutations}
        onRefresh={() => void handleRefresh()}
        onNew={openCreate}
      />

      <ProviderCategoryGrid
        entries={entries}
        onOpen={handleOpenEntry}
        oauthPendingStatus={oauthPendingStatus}
      />

      <ProviderSheet
        ref={sheetRef}
        state={effectiveSheetState}
        entries={entries}
        onClose={closeSheet}
        onSwitchToEdit={() => {
          setSheetState((previous) =>
            previous.resourceId ? { ...previous, mode: 'edit' } : previous
          );
        }}
        onBackToList={handleBackToList}
        onViewResource={handleViewResource}
        onEditResource={handleEditResource}
        onCreateResource={handleCreateResource}
        workbench={workbench}
        onCreated={handleCreated}
        onUpdated={handleUpdated}
        onDirtyChange={setSheetDirty}
        disableMutations={disableMutations}
        usageByProvider={usageByProvider}
        oauthStates={oauthStates}
        onStartOAuth={(providerId, projectId) => void startOAuth(providerId, projectId)}
        onSubmitOAuthCallback={(providerId, callbackInput) =>
          void submitOAuthCallback(providerId, callbackInput)
        }
        onResetOAuth={resetProviderAttempt}
        onCopyOAuthLink={(url) => void copyOAuthLink(url)}
        onAuthFilesChanged={refreshAuthFiles}
        authFilesRevision={authFilesRevision}
      />
    </div>
  );
}
