import { useCallback, useEffect, useId, useImperativeHandle, useMemo, useState, type Ref } from 'react';
import { useTranslation } from 'react-i18next';
import { AppSheet as Sheet } from '@/components/ui/AppSheet';
import { IconLoader2, IconPencil } from '@/components/ui/icons';
import {
  getOpenAIProviderRecentWindowStats,
  type ProviderRecentUsageMap,
} from '@/components/providers/utils';
import type { OpenAIProviderConfig } from '@/types';
import { toast } from 'sonner';
import { useConfirmationStore } from '@/stores';
import { PROVIDER_DESCRIPTORS } from '../descriptors';
import {
  type ProviderEntry,
  type ProviderEntryOAuthMeta,
  type ProviderOAuthState,
} from '../entries';
import type {
  NativeProviderBrand,
  NativeProviderFormInput,
  ProviderBrand,
  ProviderEntryFormInput,
  ProviderGroup,
  ProviderResource,
} from '../types';
import type { UseProviderWorkbenchResult } from '../useProviderWorkbench';
import { AuthFileMiniTable } from '../panels/AuthFileMiniTable';
import { OAuthLoginPanel } from '../panels/OAuthLoginPanel';
import { ProviderResourcePanel, type OpenAIPanelControls } from '../components/ProviderResourcePanel';
import type { OpenAISortBy, SortDir } from '../components/OpenAIBrandToolbar';
import { BaseProviderForm } from './forms/BaseProviderForm';
import { NativeProviderForm } from './forms/NativeProviderForm';
import { ResourceDetailView } from './ResourceDetailView';

type SheetMode = 'list' | 'oauth' | 'detail' | 'create' | 'edit';

export interface ProviderSheetState {
  open: boolean;
  entryKey: string | null;
  brand: ProviderBrand;
  mode: SheetMode;
  resourceId: string | null;
}

export interface ProviderSheetHandle {
  confirmDiscardIfDirty: () => Promise<boolean>;
}

interface ProviderSheetProps {
  state: ProviderSheetState;
  entries: ProviderEntry[];
  onClose: () => void;
  onSwitchToEdit: () => void;
  onBackToList: () => void;
  onViewResource: (resource: ProviderResource) => void;
  onEditResource: (resource: ProviderResource) => void;
  onCreateResource: () => void;
  workbench: UseProviderWorkbenchResult;
  onCreated: () => void;
  onUpdated: () => void;
  onDirtyChange?: (dirty: boolean) => void;
  disableMutations?: boolean;
  usageByProvider?: ProviderRecentUsageMap;
  oauthStates: Partial<Record<ProviderEntryOAuthMeta['id'], ProviderOAuthState>>;
  onStartOAuth: (providerId: ProviderEntryOAuthMeta['id'], projectId?: string) => void;
  onSubmitOAuthCallback: (
    providerId: ProviderEntryOAuthMeta['id'],
    callbackInput: string
  ) => void;
  onResetOAuth: (providerId: ProviderEntryOAuthMeta['id']) => void;
  onCopyOAuthLink: (url?: string) => void;
  onAuthFilesChanged: () => void | Promise<void>;
  authFilesRevision: number;
  ref?: Ref<ProviderSheetHandle>;
}

const matchesFilter = (resource: ProviderResource, normalized: string): boolean => {
  if (!normalized) return true;
  const haystack = [
    resource.identifier,
    resource.name,
    resource.authIndex,
    resource.apiKeyPreview,
    resource.apiKey,
    resource.baseUrl,
    resource.proxyUrl,
    resource.prefix,
  ]
    .filter(Boolean)
    .map((value) => String(value).toLowerCase());
  return haystack.some((value) => value.includes(normalized));
};

export function ProviderSheet({
  state,
  entries,
  onClose,
  onSwitchToEdit,
  onBackToList,
  onViewResource,
  onEditResource,
  onCreateResource,
  workbench,
  onCreated,
  onUpdated,
  onDirtyChange,
  disableMutations = false,
  usageByProvider,
  oauthStates,
  onStartOAuth,
  onSubmitOAuthCallback,
  onResetOAuth,
  onCopyOAuthLink,
  onAuthFilesChanged,
  authFilesRevision,
  ref,
}: ProviderSheetProps) {
  const { t } = useTranslation();
  const { showConfirmation } = useConfirmationStore();
  const formId = useId();
  const [submitting, setSubmitting] = useState(false);
  const [isDirty, setIsDirty] = useState(false);
  const [filter, setFilter] = useState('');
  const [openaiSortBy, setOpenaiSortBy] = useState<OpenAISortBy>('name');
  const [openaiSortDir, setOpenaiSortDir] = useState<SortDir>('asc');
  const [openaiSelectedModels, setOpenaiSelectedModels] = useState<Set<string>>(
    () => new Set()
  );

  useEffect(() => {
    setIsDirty(false);
    onDirtyChange?.(false);
  }, [onDirtyChange, state.brand, state.mode, state.resourceId, state.open]);

  useEffect(() => {
    setFilter('');
    setOpenaiSortBy('name');
    setOpenaiSortDir('asc');
    setOpenaiSelectedModels(new Set());
  }, [state.entryKey, state.open]);

  const handleDirtyChange = useCallback(
    (dirty: boolean) => {
      setIsDirty(dirty);
      onDirtyChange?.(dirty);
    },
    [onDirtyChange]
  );

  const liveEntry = useMemo(
    () => entries.find((entry) => entry.key === state.entryKey) ?? null,
    [entries, state.entryKey]
  );

  const listGroup = useMemo<ProviderGroup | null>(() => {
    if (!liveEntry || liveEntry.kind === 'oauth') return null;
    if (liveEntry.kind === 'config') {
      return { ...liveEntry.group, resources: liveEntry.resources };
    }
    return {
      id: 'openaiCompatibility',
      resources: liveEntry.resources,
      issue: null,
      path: '/ai-providers/openai',
    };
  }, [liveEntry]);

  const resource = useMemo(
    () =>
      listGroup?.resources.find((candidate) => candidate.id === state.resourceId) ?? null,
    [listGroup, state.resourceId]
  );

  const descriptor = PROVIDER_DESCRIPTORS[state.brand];
  const isNative = state.brand === 'openrouter' || state.brand === 'opencode';
  const isEditingForm = state.mode === 'create' || state.mode === 'edit';
  const hasListView = Boolean(listGroup);
  const isOpenAIList = listGroup?.id === 'openaiCompatibility';

  const filteredResources = useMemo(() => {
    const normalized = filter.trim().toLowerCase();
    return (listGroup?.resources ?? []).filter((candidate) =>
      matchesFilter(candidate, normalized)
    );
  }, [filter, listGroup]);

  const availableOpenAIModels = useMemo(() => {
    if (!isOpenAIList || !listGroup) return [];
    const seen = new Set<string>();
    listGroup.resources.forEach((candidate) => {
      const config = candidate.raw as OpenAIProviderConfig;
      config.models?.forEach((model) => {
        const name = (model.name ?? '').trim();
        if (name) seen.add(name);
      });
    });
    return Array.from(seen).sort();
  }, [isOpenAIList, listGroup]);

  const visibleResources = useMemo(() => {
    if (!isOpenAIList) return filteredResources;

    let visible = filteredResources;
    if (openaiSelectedModels.size > 0) {
      visible = visible.filter((candidate) => {
        const config = candidate.raw as OpenAIProviderConfig;
        return Boolean(
          config.models?.some((model) =>
            openaiSelectedModels.has((model.name ?? '').trim())
          )
        );
      });
    }

    const sorted = [...visible].sort((a, b) => {
      let diff = 0;
      if (openaiSortBy === 'name') {
        diff = (a.name ?? a.identifier ?? '').localeCompare(b.name ?? b.identifier ?? '');
      } else if (openaiSortBy === 'priority') {
        diff =
          ((a.raw as OpenAIProviderConfig).priority ?? 0) -
          ((b.raw as OpenAIProviderConfig).priority ?? 0);
      } else {
        diff =
          getOpenAIProviderRecentWindowStats(
            a.raw as OpenAIProviderConfig,
            usageByProvider ?? new Map()
          ).success -
          getOpenAIProviderRecentWindowStats(
            b.raw as OpenAIProviderConfig,
            usageByProvider ?? new Map()
          ).success;
      }
      return openaiSortDir === 'asc' ? diff : -diff;
    });

    return sorted;
  }, [
    filteredResources,
    isOpenAIList,
    openaiSelectedModels,
    openaiSortBy,
    openaiSortDir,
    usageByProvider,
  ]);

  const openaiControls = useMemo<OpenAIPanelControls | undefined>(() => {
    if (!isOpenAIList) return undefined;
    return {
      sortBy: openaiSortBy,
      sortDir: openaiSortDir,
      onSortBy: setOpenaiSortBy,
      onSortDir: setOpenaiSortDir,
      availableModels: availableOpenAIModels,
      selectedModels: openaiSelectedModels,
      onSelectedModelsChange: setOpenaiSelectedModels,
    };
  }, [availableOpenAIModels, isOpenAIList, openaiSelectedModels, openaiSortBy, openaiSortDir]);

  const confirmDiscardIfDirty = useCallback((): Promise<boolean> => {
    if (!isEditingForm || !isDirty || submitting) {
      return Promise.resolve(true);
    }
    return new Promise<boolean>((resolve) => {
      showConfirmation({
        title: t('providersPage.unsavedChanges.title'),
        message: t('providersPage.unsavedChanges.message'),
        variant: 'danger',
        confirmText: t('providersPage.unsavedChanges.discard'),
        cancelText: t('providersPage.unsavedChanges.keepEditing'),
        onConfirm: () => resolve(true),
        onCancel: () => resolve(false),
      });
    });
  }, [isDirty, isEditingForm, showConfirmation, submitting, t]);

  useImperativeHandle(ref, () => ({ confirmDiscardIfDirty }), [confirmDiscardIfDirty]);

  const handleBackClick = useCallback(() => {
    void confirmDiscardIfDirty().then((ok) => {
      if (ok) onBackToList();
    });
  }, [confirmDiscardIfDirty, onBackToList]);

  const handleCloseClick = useCallback(() => {
    void confirmDiscardIfDirty().then((ok) => {
      if (ok) onClose();
    });
  }, [confirmDiscardIfDirty, onClose]);

  const handleCreate = useCallback(
    async (input: ProviderEntryFormInput) => {
      setSubmitting(true);
      try {
        await workbench.createProvider(state.brand, input);
        onCreated();
      } finally {
        setSubmitting(false);
      }
    },
    [onCreated, state.brand, workbench]
  );

  const handleUpdate = useCallback(
    async (input: ProviderEntryFormInput) => {
      if (!resource) return;
      setSubmitting(true);
      try {
        await workbench.updateProvider(resource, input);
        onUpdated();
      } finally {
        setSubmitting(false);
      }
    },
    [onUpdated, resource, workbench]
  );

  const handleNativeSubmit = useCallback(
    async (input: NativeProviderFormInput) => {
      if (!isNative) return;
      setSubmitting(true);
      try {
        if (state.mode === 'create') {
          await workbench.createNativeProvider(state.brand as NativeProviderBrand, input);
          onCreated();
        } else if (resource) {
          await workbench.updateNativeProvider(resource, input);
          onUpdated();
        }
      } finally {
        setSubmitting(false);
      }
    },
    [isNative, onCreated, onUpdated, resource, state.brand, state.mode, workbench]
  );

  const handleDelete = useCallback(
    (candidate: ProviderResource) => {
      const name = candidate.name ?? candidate.apiKeyPreview ?? candidate.identifier ?? '';
      showConfirmation({
        title: t('providersPage.delete.title'),
        message: t('providersPage.delete.confirm', { name }),
        variant: 'danger',
        confirmText: t('providersPage.actions.delete'),
        onConfirm: async () => {
          try {
            await workbench.deleteProvider(candidate);
            toast.success(t('providersPage.toast.deleted'));
          } catch (error) {
            const message = error instanceof Error ? error.message : String(error);
            toast.error(`${t('notification.delete_failed')}: ${message}`);
          }
        },
      });
    },
    [showConfirmation, t, workbench]
  );

  const handleToggleDisabled = useCallback(
    async (candidate: ProviderResource, disabled: boolean) => {
      try {
        await workbench.toggleDisabled(candidate, disabled);
        toast.success(
          disabled ? t('providersPage.toast.disabled') : t('providersPage.toast.enabled')
        );
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        toast.error(`${t('providersPage.toast.toggleFailed')}: ${message}`);
      }
    },
    [t, workbench]
  );

  const presetHeader =
    liveEntry?.kind === 'preset' ? (
      <div className="flex flex-col gap-2 border border-border bg-muted p-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h3 className="m-0 text-base font-semibold text-foreground">
            {liveEntry.preset.displayName}
          </h3>
          <span
            className={[
              'inline-flex items-center border px-2 py-0.5 text-[11px] font-medium',
              liveEntry.preset.verified
                ? 'border-emerald-400/40 bg-emerald-100 text-emerald-700'
                : 'border-amber-400/40 bg-amber-100 text-amber-700',
            ].join(' ')}
          >
            {liveEntry.preset.verified
              ? t('providersPage.presets.verified')
              : t('providersPage.presets.unverified')}
          </span>
        </div>
        <span className="break-all font-mono text-[12px] text-muted-foreground">
          {liveEntry.preset.baseUrl}
        </span>
        {liveEntry.preset.freeTierNote ? (
          <p className="m-0 text-[12px] text-muted-foreground">{liveEntry.preset.freeTierNote}</p>
        ) : null}
        {liveEntry.preset.signupUrl ? (
          <a
            href={liveEntry.preset.signupUrl}
            target="_blank"
            rel="noreferrer"
            className="text-[12px] text-primary underline underline-offset-2"
          >
            {t('providersPage.presets.signup')}
          </a>
        ) : null}
        {liveEntry.resources.length === 0 ? (
          <p className="m-0 text-[12px] text-muted-foreground">
            {t('providersPage.presets.empty')}
          </p>
        ) : null}
      </div>
    ) : null;

  const renderList = () => {
    if (!listGroup) return null;
    return (
      <div className="flex flex-col gap-4">
        {presetHeader}
        <ProviderResourcePanel
          group={listGroup}
          filter={filter}
          onFilterChange={setFilter}
          filteredResources={visibleResources}
          selectedId={state.resourceId}
          disableMutations={disableMutations}
          usageByProvider={usageByProvider}
          openaiControls={openaiControls}
          onView={onViewResource}
          onEdit={onEditResource}
          onDelete={handleDelete}
          onToggleDisabled={handleToggleDisabled}
          onCreate={onCreateResource}
        />
      </div>
    );
  };

  const renderOAuth = () => {
    if (liveEntry?.kind !== 'oauth') return null;
    const provider: ProviderEntryOAuthMeta = {
      id: liveEntry.oauthId,
      titleKey: liveEntry.titleKey,
      hintKey: liveEntry.hintKey,
      urlLabelKey: liveEntry.urlLabelKey,
      icon: liveEntry.icon,
    };
    return (
      <div className="flex flex-col gap-6">
        <OAuthLoginPanel
          key={`${provider.id}:${oauthStates[provider.id]?.state ?? ''}:${oauthStates[provider.id]?.callbackStatus === 'success' ? 'callback-success' : ''}:${oauthStates[provider.id]?.callbackError ? 'callback-error' : ''}`}
          provider={provider}
          state={oauthStates[provider.id] ?? {}}
          onStart={onStartOAuth}
          onSubmitCallback={onSubmitOAuthCallback}
          onReset={onResetOAuth}
          onCopyLink={onCopyOAuthLink}
        />
        <div className="flex flex-col gap-2 border-t border-border pt-4">
          <h3 className="m-0 text-sm font-semibold text-foreground">
            {t('providersPage.oauthAccounts.title')}
          </h3>
          <AuthFileMiniTable
            oauthId={provider.id}
            refreshRevision={authFilesRevision}
            onFilesChanged={onAuthFilesChanged}
          />
          <a
            href="#/auth-files"
            className="text-[12px] text-primary underline underline-offset-2"
          >
            {t('providersPage.oauthAccounts.manage')}
          </a>
          <p className="m-0 text-[11px] text-muted-foreground">
            {t('providersPage.oauthAccounts.unknownHint')}
          </p>
        </div>
      </div>
    );
  };

  const renderBody = () => {
    if (state.mode === 'list') return renderList();
    if (state.mode === 'oauth') return renderOAuth();
    if (state.mode === 'detail') {
      return resource ? <ResourceDetailView resource={resource} usageByProvider={usageByProvider} /> : null;
    }

    const initialPresetId =
      state.mode === 'create' && liveEntry?.kind === 'preset' ? liveEntry.preset.id : undefined;
    const formKey = `${state.entryKey ?? state.brand}:${state.brand}:${state.resourceId ?? 'new'}:${state.mode}:${initialPresetId ?? ''}`;
    if (isNative) {
      return (
        <NativeProviderForm
          key={formKey}
          brand={state.brand as NativeProviderBrand}
          resource={resource}
          mode={state.mode}
          mutating={submitting || workbench.mutating}
          formId={formId}
          onSubmit={handleNativeSubmit}
          onDirtyChange={handleDirtyChange}
        />
      );
    }
    return (
      <BaseProviderForm
        key={formKey}
        brand={state.brand}
        resource={resource}
        mode={state.mode}
        initialPresetId={initialPresetId}
        mutating={submitting || workbench.mutating}
        formId={formId}
        onSubmit={state.mode === 'create' ? handleCreate : handleUpdate}
        onDirtyChange={handleDirtyChange}
      />
    );
  };

  const footerBtnBase =
    'inline-flex items-center gap-1.5 h-8 px-[14px] text-[13px] font-medium cursor-pointer border disabled:opacity-60 disabled:cursor-not-allowed';
  const footerBtnGhost = `${footerBtnBase} bg-transparent border-transparent text-foreground hover:bg-secondary`;
  const footerBtnPrimary = `${footerBtnBase} bg-primary border-transparent text-primary-foreground hover:bg-[var(--primary-hover)]`;

  const closeButton = (
    <button
      type="button"
      className={footerBtnGhost}
      onClick={handleCloseClick}
      disabled={submitting}
    >
      {t('providersPage.actions.cancel')}
    </button>
  );
  const backButton = (
    <button type="button" className={footerBtnGhost} onClick={handleBackClick} disabled={submitting}>
      {t('common.back')}
    </button>
  );

  const footer =
    state.mode === 'list' || state.mode === 'oauth' ? (
      <button type="button" className={footerBtnPrimary} onClick={onClose}>
        {t('providersPage.actions.cancel')}
      </button>
    ) : state.mode === 'detail' ? (
      <>
        {hasListView ? backButton : closeButton}
        {resource ? (
          <button type="button" className={footerBtnPrimary} onClick={onSwitchToEdit}>
            <IconPencil size={14} />
            {t('providersPage.actions.edit')}
          </button>
        ) : null}
      </>
    ) : (
      <>
        {hasListView ? backButton : closeButton}
        <button
          type="submit"
          form={formId}
          className={footerBtnPrimary}
          disabled={submitting}
        >
          {submitting ? <IconLoader2 size={14} /> : null}
          {state.mode === 'create'
            ? t('providersPage.actions.create')
            : t('providersPage.actions.save')}
        </button>
      </>
    );

  const titleText =
    state.mode === 'oauth' && liveEntry?.kind === 'oauth'
      ? t(liveEntry.titleKey)
      : state.mode === 'list' && liveEntry?.kind === 'preset'
        ? liveEntry.preset.displayName
        : state.mode === 'create'
          ? `${t('providersPage.form.createEyebrow')} · ${t(
              `providersPage.providerNames.${state.brand}`
            )}`
          : state.mode === 'edit'
            ? `${t('providersPage.form.editEyebrow')} · ${t(
                `providersPage.providerNames.${state.brand}`
              )}`
            : state.mode === 'list'
              ? t(`providersPage.providerNames.${state.brand}`)
              : `${t('providersPage.detail.title')} · ${t(
                  `providersPage.providerNames.${state.brand}`
                )}`;

  const eyebrow =
    state.mode === 'oauth'
      ? t('providersPage.oauthAccounts.eyebrow')
      : state.mode === 'list'
        ? t('providersPage.detail.title')
        : state.mode === 'create'
          ? t('providersPage.form.createEyebrow')
          : state.mode === 'edit'
            ? t('providersPage.form.editEyebrow')
            : t('providersPage.detail.title');

  return (
    <Sheet
      open={state.open}
      onClose={onClose}
      size={state.mode === 'list' || state.mode === 'oauth' ? '2xl' : descriptor.sheetSize}
      eyebrow={eyebrow}
      title={titleText}
      description={
        state.mode === 'oauth'
          ? t('providersPage.oauthAccounts.description')
          : t('providersPage.table.description', {
              route: `/ai-providers/${state.brand === 'openaiCompatibility' ? 'openai' : state.brand}`,
            })
      }
      footer={footer}
      closeDisabled={submitting}
      confirmClose={confirmDiscardIfDirty}
    >
      {renderBody()}
    </Sheet>
  );
}
