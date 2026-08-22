import { useId, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { Button } from '@/components/ui/Button';
import { Modal } from '@/components/ui/Modal';
import { copyToClipboard } from '@/utils/clipboard';
import { maskApiKey } from '@/utils/format';
import { isValidApiKeyCharset } from '@/utils/validation';
import { makeClientId } from '@/types/visualConfig';

export type ApiKeysManagerProps = {
  keys: string[];
  disabled?: boolean;
  onAdd: (value: string) => Promise<void>;
  onEdit: (index: number, value: string) => Promise<void>;
  onDelete: (index: number) => Promise<void>;
};

export function ApiKeysManager(props: ApiKeysManagerProps) {
  const { t } = useTranslation();
  const { keys, disabled = false, onAdd, onEdit, onDelete } = props;

  const [keyIds, setKeyIds] = useState(() => keys.map(() => makeClientId()));
  const renderKeyIds = useMemo(() => {
    if (keyIds.length === keys.length) return keyIds;
    if (keyIds.length > keys.length) return keyIds.slice(0, keys.length);
    return [
      ...keyIds,
      ...Array.from({ length: keys.length - keyIds.length }, () => makeClientId()),
    ];
  }, [keyIds, keys.length]);

  const apiKeyInputId = useId();
  const apiKeyHintId = `${apiKeyInputId}-hint`;
  const apiKeyErrorId = `${apiKeyInputId}-error`;

  const [modalOpen, setModalOpen] = useState(false);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [inputValue, setInputValue] = useState('');
  const [formError, setFormError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [deletingIndex, setDeletingIndex] = useState<number | null>(null);

  const controlsDisabled = disabled || submitting || deletingIndex !== null;

  function generateSecureApiKey(): string {
    const charset = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    const array = new Uint8Array(17);
    crypto.getRandomValues(array);
    return `sk-${Array.from(array, (value) => charset[value % charset.length]).join('')}`;
  }

  const openAddModal = () => {
    setEditingIndex(null);
    setInputValue('');
    setFormError('');
    setModalOpen(true);
  };

  const openEditModal = (index: number) => {
    setEditingIndex(index);
    setInputValue(keys[index] ?? '');
    setFormError('');
    setModalOpen(true);
  };

  const closeModal = () => {
    if (submitting) return;
    setModalOpen(false);
    setEditingIndex(null);
    setInputValue('');
    setFormError('');
  };

  const resetModalState = () => {
    setModalOpen(false);
    setEditingIndex(null);
    setInputValue('');
    setFormError('');
  };

  const handleCopy = async (apiKey: string) => {
    const copied = await copyToClipboard(apiKey);
    if (copied) {
      toast.success(t('notification.link_copied'));
    } else {
      toast.error(t('notification.copy_failed'));
    }
  };

  const handleGenerate = () => {
    setInputValue(generateSecureApiKey());
    setFormError('');
  };

  const handleSave = async () => {
    const trimmed = inputValue.trim();
    if (!trimmed) {
      setFormError(t('config_management.visual.api_keys.error_empty'));
      return;
    }
    if (!isValidApiKeyCharset(trimmed)) {
      setFormError(t('config_management.visual.api_keys.error_invalid'));
      return;
    }

    setSubmitting(true);
    try {
      if (editingIndex === null) {
        await onAdd(trimmed);
        setKeyIds((current) => [...current, makeClientId()]);
      } else {
        await onEdit(editingIndex, trimmed);
      }
      resetModalState();
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (index: number) => {
    setDeletingIndex(index);
    try {
      await onDelete(index);
      setKeyIds((current) => current.filter((_, currentIndex) => currentIndex !== index));
    } finally {
      setDeletingIndex(null);
    }
  };

  return (
    <div className="flex flex-col gap-[7px]">
      <div className="flex items-center justify-between gap-[10px] flex-wrap">
        <label style={{ margin: 0 }}>{t('config_management.visual.api_keys.label')}</label>
        <Button size="sm" onClick={openAddModal} disabled={controlsDisabled}>
          {t('config_management.visual.api_keys.add')}
        </Button>
      </div>

      {keys.length === 0 ? (
        <div className="border border-dashed border-border p-4 text-center text-muted-foreground bg-transparent">
          {t('config_management.visual.api_keys.empty')}
        </div>
      ) : (
        <div className="flex flex-col gap-2 mt-2">
          {keys.map((key, index) => (
            <div
              key={renderKeyIds[index] ?? `${key}-${index}`}
              className="flex items-center justify-between gap-2 p-3 border-b border-border last:border-b-0 bg-transparent transition-colors duration-150 hover:bg-muted/50 max-md:flex-col max-md:items-stretch"
            >
              <div className="flex flex-col gap-1 min-w-0 flex-1">
                <div className="inline-flex items-center px-2 py-[2px] text-[0.6875rem] font-semibold border border-border bg-transparent text-muted-foreground leading-[1.5] rounded-sm w-fit">
                  #{index + 1}
                </div>
                <div className="text-sm font-medium text-foreground">
                  {t('config_management.visual.api_keys.input_label')}
                </div>
                <div className="text-xs text-muted-foreground break-all">
                  {maskApiKey(String(key || ''))}
                </div>
              </div>
              <div className="flex items-center flex-wrap gap-1.5 shrink-0">
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => void handleCopy(key)}
                  disabled={controlsDisabled}
                >
                  {t('common.copy')}
                </Button>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => openEditModal(index)}
                  disabled={controlsDisabled}
                >
                  {t('config_management.visual.common.edit')}
                </Button>
                <Button
                  variant="danger"
                  size="sm"
                  onClick={() => void handleDelete(index)}
                  disabled={controlsDisabled}
                  loading={deletingIndex === index}
                >
                  {t('config_management.visual.common.delete')}
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="text-muted-foreground text-[12px] leading-[1.55]">
        {t('config_management.visual.api_keys.hint')}
      </div>

      <Modal
        open={modalOpen}
        onClose={closeModal}
        title={
          editingIndex !== null
            ? t('config_management.visual.api_keys.edit_title')
            : t('config_management.visual.api_keys.add_title')
        }
        footer={
          <>
            <Button variant="secondary" onClick={closeModal} disabled={controlsDisabled}>
              {t('config_management.visual.common.cancel')}
            </Button>
            <Button onClick={() => void handleSave()} disabled={controlsDisabled} loading={submitting}>
              {editingIndex !== null
                ? t('config_management.visual.common.update')
                : t('config_management.visual.common.add')}
            </Button>
          </>
        }
      >
        <div className="flex flex-col gap-[7px]">
          <label
            htmlFor={apiKeyInputId}
            className="text-muted-foreground text-[12px] font-bold tracking-[0.02em]"
          >
            {t('config_management.visual.api_keys.input_label')}
          </label>
          <div className="flex gap-2 items-center max-[900px]:flex-col max-[900px]:items-stretch [&_input]:flex-1">
            <input
              id={apiKeyInputId}
              className="min-h-[42px] bg-muted border border-border shadow-none focus:bg-background focus:border-foreground w-full px-3 py-2 text-sm"
              placeholder={t('config_management.visual.api_keys.input_placeholder')}
              value={inputValue}
              onChange={(event) => setInputValue(event.target.value)}
              disabled={controlsDisabled}
              aria-describedby={formError ? `${apiKeyErrorId} ${apiKeyHintId}` : apiKeyHintId}
              aria-invalid={Boolean(formError)}
            />
            <Button
              type="button"
              variant="secondary"
              size="sm"
              onClick={handleGenerate}
              disabled={controlsDisabled}
            >
              {t('config_management.visual.api_keys.generate')}
            </Button>
          </div>
          <div id={apiKeyHintId} className="text-muted-foreground text-[12px] leading-[1.55]">
            {t('config_management.visual.api_keys.input_hint')}
          </div>
          {formError && (
            <div
              id={apiKeyErrorId}
              className="p-[10px_14px] mb-0 bg-destructive/10 border border-destructive/35 text-destructive text-sm leading-[1.5]"
            >
              {formError}
            </div>
          )}
        </div>
      </Modal>
    </div>
  );
}
