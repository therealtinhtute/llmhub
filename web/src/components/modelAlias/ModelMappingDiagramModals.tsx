import type { KeyboardEvent } from 'react';
import type { TFunction } from 'i18next';
import { Modal } from '@/components/ui/Modal';
import { FormInput as Input } from '@/components/ui/FormInput';
import { Button } from '@/components/ui/Button';
import { ToggleSwitch } from '@/components/ui/ToggleSwitch';
import { IconTrash2 } from '@/components/ui/icons';
import type { AliasNode, SourceNode } from './ModelMappingDiagramTypes';

interface RenameAliasModalProps {
  open: boolean;
  t: TFunction;
  value: string;
  error: string;
  onChange: (value: string) => void;
  onClose: () => void;
  onSubmit: () => void;
}

export function RenameAliasModal({
  open,
  t,
  value,
  error,
  onChange,
  onClose,
  onSubmit
}: RenameAliasModalProps) {
  return (
    <Modal
      open={open}
      onClose={onClose}
      title={t('oauth_model_alias.diagram_rename_alias_title')}
      width={400}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button onClick={onSubmit}>{t('oauth_model_alias.diagram_rename_btn')}</Button>
        </>
      }
    >
      <Input
        label={t('oauth_model_alias.diagram_rename_alias_label')}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e: KeyboardEvent<HTMLInputElement>) => {
          if (e.key === 'Enter') onSubmit();
        }}
        error={error}
        placeholder={t('oauth_model_alias.diagram_rename_placeholder')}
        autoFocus
      />
    </Modal>
  );
}

interface AddAliasModalProps {
  open: boolean;
  t: TFunction;
  value: string;
  error: string;
  onChange: (value: string) => void;
  onClose: () => void;
  onSubmit: () => void;
}

export function AddAliasModal({
  open,
  t,
  value,
  error,
  onChange,
  onClose,
  onSubmit
}: AddAliasModalProps) {
  return (
    <Modal
      open={open}
      onClose={onClose}
      title={t('oauth_model_alias.diagram_add_alias_title')}
      width={400}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button onClick={onSubmit}>{t('oauth_model_alias.diagram_add_btn')}</Button>
        </>
      }
    >
      <Input
        label={t('oauth_model_alias.diagram_add_alias_label')}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e: KeyboardEvent<HTMLInputElement>) => {
          if (e.key === 'Enter') onSubmit();
        }}
        error={error}
        placeholder={t('oauth_model_alias.diagram_add_placeholder')}
        autoFocus
      />
    </Modal>
  );
}

interface SettingsAliasModalProps {
  open: boolean;
  t: TFunction;
  alias: string | null;
  aliasNodes: AliasNode[];
  onClose: () => void;
  onToggleFork: (provider: string, sourceModel: string, alias: string, fork: boolean) => void;
  onUnlink: (provider: string, sourceModel: string, alias: string) => void;
}

export function SettingsAliasModal({
  open,
  t,
  alias,
  aliasNodes,
  onClose,
  onToggleFork,
  onUnlink
}: SettingsAliasModalProps) {
  return (
    <Modal
      open={open}
      onClose={onClose}
      title={t('oauth_model_alias.diagram_settings_title', { alias: alias ?? '' })}
      width={720}
      footer={
        <Button variant="secondary" onClick={onClose}>
          {t('common.close')}
        </Button>
      }
    >
      {alias ? (
        (() => {
          const node = aliasNodes.find((n) => n.alias === alias);
          if (!node || node.sources.length === 0) {
            return <div className="text-muted-foreground/60 text-[13px] text-center py-4">{t('oauth_model_alias.diagram_settings_empty')}</div>;
          }
          return (
            <div className="flex flex-col gap-2">
              {node.sources.map((source) => {
                const entry = source.aliases.find((item) => item.alias === alias);
                const forkEnabled = entry?.fork === true;
                return (
                  <div key={source.id} className="grid [grid-template-columns:minmax(200px,1fr)_auto] gap-3 items-center py-2 px-3 border border-border bg-muted max-md:[grid-template-columns:1fr] max-md:items-start">
                    <div className="flex items-center gap-1 text-[13px] text-foreground min-w-0">
                      <span className="whitespace-nowrap overflow-hidden text-ellipsis max-w-[220px]">{source.name}</span>
                      <span className="text-muted-foreground/60">→</span>
                      <span className="whitespace-nowrap overflow-hidden text-ellipsis max-w-[220px]">{alias}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-[12px] text-muted-foreground">
                        {t('oauth_model_alias.alias_fork_label')}
                      </span>
                      <ToggleSwitch
                        checked={forkEnabled}
                        onChange={(value) => onToggleFork(source.provider, source.name, alias, value)}
                        ariaLabel={t('oauth_model_alias.alias_fork_label')}
                      />
                      <button
                        type="button"
                        className="border-0 bg-transparent text-destructive p-[6px] cursor-pointer hover:bg-destructive/10"
                        onClick={() => onUnlink(source.provider, source.name, alias)}
                        aria-label={t('oauth_model_alias.diagram_delete_link', {
                          provider: source.provider,
                          name: source.name
                        })}
                        title={t('oauth_model_alias.diagram_delete_link', {
                          provider: source.provider,
                          name: source.name
                        })}
                      >
                        <IconTrash2 size={14} />
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          );
        })()
      ) : null}
    </Modal>
  );
}

interface SettingsSourceModalProps {
  open: boolean;
  t: TFunction;
  source: SourceNode | null;
  onClose: () => void;
  onToggleFork: (provider: string, sourceModel: string, alias: string, fork: boolean) => void;
  onUnlink: (provider: string, sourceModel: string, alias: string) => void;
}

export function SettingsSourceModal({
  open,
  t,
  source,
  onClose,
  onToggleFork,
  onUnlink
}: SettingsSourceModalProps) {
  return (
    <Modal
      open={open}
      onClose={onClose}
      title={t('oauth_model_alias.diagram_settings_source_title')}
      width={720}
      footer={
        <Button variant="secondary" onClick={onClose}>
          {t('common.close')}
        </Button>
      }
    >
      {source ? (
        source.aliases.length === 0 ? (
          <div className="text-muted-foreground/60 text-[13px] text-center py-4">{t('oauth_model_alias.diagram_settings_empty')}</div>
        ) : (
          <div className="flex flex-col gap-2">
            {source.aliases.map((entry) => (
              <div key={`${source.id}-${entry.alias}`} className="grid [grid-template-columns:minmax(200px,1fr)_auto] gap-3 items-center py-2 px-3 border border-border bg-muted max-md:[grid-template-columns:1fr] max-md:items-start">
                <div className="flex items-center gap-1 text-[13px] text-foreground min-w-0">
                  <span className="whitespace-nowrap overflow-hidden text-ellipsis max-w-[220px]">{source.name}</span>
                  <span className="text-muted-foreground/60">→</span>
                  <span className="whitespace-nowrap overflow-hidden text-ellipsis max-w-[220px]">{entry.alias}</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-[12px] text-muted-foreground">
                    {t('oauth_model_alias.alias_fork_label')}
                  </span>
                  <ToggleSwitch
                    checked={entry.fork === true}
                    onChange={(value) => onToggleFork(source.provider, source.name, entry.alias, value)}
                    ariaLabel={t('oauth_model_alias.alias_fork_label')}
                  />
                  <button
                    type="button"
                    className="border-0 bg-transparent text-destructive p-[6px] cursor-pointer hover:bg-destructive/10"
                    onClick={() => onUnlink(source.provider, source.name, entry.alias)}
                    aria-label={t('oauth_model_alias.diagram_delete_link', {
                      provider: source.provider,
                      name: source.name
                    })}
                    title={t('oauth_model_alias.diagram_delete_link', {
                      provider: source.provider,
                      name: source.name
                    })}
                  >
                    <IconTrash2 size={14} />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )
      ) : null}
    </Modal>
  );
}
