import { useTranslation } from 'react-i18next';
import { Modal } from '@/components/ui/Modal';
import { Button } from '@/components/ui/Button';
import { LoadingSpinner } from '@/components/ui/LoadingSpinner';
import { FormInput as Input } from '@/components/ui/FormInput';
import { ToggleSwitch } from '@/components/ui/ToggleSwitch';
import type {
  PrefixProxyEditorField,
  PrefixProxyEditorFieldValue,
  PrefixProxyEditorState,
} from '@/features/authFiles/hooks/useAuthFilesPrefixProxyEditor';

export type AuthFilesPrefixProxyEditorModalProps = {
  disableControls: boolean;
  editor: PrefixProxyEditorState | null;
  updatedText: string;
  dirty: boolean;
  onClose: () => void;
  onCopyText: (text: string) => void | Promise<void>;
  onSave: () => void;
  onChange: (field: PrefixProxyEditorField, value: PrefixProxyEditorFieldValue) => void;
};

export function AuthFilesPrefixProxyEditorModal(props: AuthFilesPrefixProxyEditorModalProps) {
  const { t } = useTranslation();
  const { disableControls, editor, updatedText, dirty, onClose, onCopyText, onSave, onChange } =
    props;
  const formatJsonText = (text: string) => {
    if (!text) return '';
    try {
      return JSON.stringify(JSON.parse(text), null, 2);
    } catch {
      return text;
    }
  };
  const previewText = formatJsonText(updatedText);
  const invalidContentPreview = editor?.invalidContentPreview ?? '';

  return (
    <Modal
      open={Boolean(editor)}
      onClose={onClose}
      closeDisabled={editor?.saving === true}
      width={720}
      title={
        editor?.fileName
          ? t('auth_files.auth_field_editor_title', { name: editor.fileName })
          : t('auth_files.prefix_proxy_button')
      }
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={editor?.saving === true}>
            {dirty ? t('common.cancel') : t('common.close')}
          </Button>
          <Button
            variant="secondary"
            onClick={() => {
              if (!updatedText) return;
              void onCopyText(updatedText);
            }}
            disabled={editor?.saving === true || !updatedText}
          >
            {t('common.copy')}
          </Button>
          <Button
            onClick={onSave}
            loading={editor?.saving === true}
            disabled={
              disableControls ||
              editor?.saving === true ||
              !dirty ||
              !editor?.json ||
              Boolean(editor?.headersTouched && editor.headersError)
            }
          >
            {t('common.save')}
          </Button>
        </>
      }
    >
      {editor && (
        <div className="flex flex-col gap-3 max-h-[60vh] overflow-auto">
          {editor.loading ? (
            <div className="flex items-center justify-center gap-2 text-[12px] text-muted-foreground py-2">
              <LoadingSpinner size={14} />
              <span>{t('auth_files.prefix_proxy_loading')}</span>
            </div>
          ) : (
            <>
              {editor.error && <div className="py-2 px-3 border border-destructive bg-destructive/10 text-destructive text-[12px]">{editor.error}</div>}
              <div className="flex flex-col gap-[6px]">
                <label className="text-[12px] text-muted-foreground font-semibold">
                  {t('auth_files.prefix_proxy_info_label')}
                </label>
                <textarea
                  className="w-full p-2 px-3 border border-border bg-muted text-foreground text-[12px] font-mono resize-y min-h-[120px] box-border focus:outline-none focus:border-primary"
                  rows={8}
                  readOnly
                  value={editor.fileInfoText}
                />
              </div>
              <div className="flex flex-col gap-[6px]">
                <label className="text-[12px] text-muted-foreground font-semibold">
                  {editor.json
                    ? t('auth_files.prefix_proxy_source_label')
                    : t('auth_files.prefix_proxy_invalid_content_label')}
                </label>
                {editor.json ? (
                  <textarea
                    className="w-full p-2 px-3 border border-border bg-muted text-foreground text-[12px] font-mono resize-y min-h-[120px] box-border focus:outline-none focus:border-primary"
                    rows={10}
                    readOnly
                    value={previewText}
                  />
                ) : (
                  <pre className="w-full max-h-[220px] m-0 p-2 px-3 border border-border bg-muted text-muted-foreground text-[12px] font-mono leading-[1.5] whitespace-pre-wrap overflow-auto box-border">
                    {invalidContentPreview}
                  </pre>
                )}
              </div>
              {editor.json && (
                <div className="grid [grid-template-columns:1fr] gap-2">
                  <Input
                    label={t('auth_files.prefix_label')}
                    value={editor.prefix}
                    disabled={disableControls || editor.saving || !editor.json}
                    onChange={(e) => onChange('prefix', e.target.value)}
                  />
                  <Input
                    label={t('auth_files.proxy_url_label')}
                    value={editor.proxyUrl}
                    placeholder={t('auth_files.proxy_url_placeholder')}
                    disabled={disableControls || editor.saving || !editor.json}
                    onChange={(e) => onChange('proxyUrl', e.target.value)}
                  />
                  <Input
                    label={t('auth_files.priority_label')}
                    value={editor.priority}
                    placeholder={t('auth_files.priority_placeholder')}
                    hint={t('auth_files.priority_hint')}
                    disabled={disableControls || editor.saving || !editor.json}
                    onChange={(e) => onChange('priority', e.target.value)}
                  />
                  {editor.providerKey === 'codex' && (
                    <div className="flex flex-col gap-1.5 mb-3">
                      <label>{t('auth_files.codex_websockets_label')}</label>
                      <ToggleSwitch
                        checked={editor.websockets}
                        onChange={(value) => onChange('websockets', value)}
                        disabled={disableControls || editor.saving || !editor.json}
                        ariaLabel={t('auth_files.codex_websockets_label')}
                      />
                      <div className="text-[13px] text-muted-foreground leading-[1.55]">{t('auth_files.codex_websockets_hint')}</div>
                    </div>
                  )}
                  <div className="flex flex-col gap-1.5 mb-3">
                    <label>{t('auth_files.headers_label')}</label>
                    <textarea
                      className={`min-h-[42px] bg-muted border border-border shadow-none focus:bg-background focus:border-foreground min-h-[112px] w-full px-3 py-2 text-sm ${editor.headersError ? 'border-destructive' : ''}`}
                      value={editor.headersText}
                      placeholder={t('auth_files.headers_placeholder')}
                      rows={4}
                      aria-invalid={Boolean(editor.headersError)}
                      disabled={disableControls || editor.saving || !editor.json}
                      onChange={(e) => onChange('headersText', e.target.value)}
                    />
                    {editor.headersError && <div className="p-[10px_14px] mb-0 bg-destructive/10 border border-destructive/35 text-destructive text-sm leading-[1.5]">{editor.headersError}</div>}
                    <div className="text-[13px] text-muted-foreground leading-[1.55]">{t('auth_files.headers_hint')}</div>
                  </div>
                  <Input
                    label={t('auth_files.note_label')}
                    value={editor.note}
                    placeholder={t('auth_files.note_placeholder')}
                    hint={t('auth_files.note_hint')}
                    disabled={disableControls || editor.saving || !editor.json}
                    onChange={(e) => onChange('note', e.target.value)}
                  />
                </div>
              )}
            </>
          )}
        </div>
      )}
    </Modal>
  );
}
