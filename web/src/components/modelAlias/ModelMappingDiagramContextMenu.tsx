import { useEffect, useRef } from 'react';
import { createPortal } from 'react-dom';
import type { TFunction } from 'i18next';
import type { ContextMenuState } from './ModelMappingDiagramTypes';

interface DiagramContextMenuProps {
  contextMenu: ContextMenuState | null;
  t: TFunction;
  onRequestClose: () => void;
  onAddAlias: () => void;
  onRenameAlias: (alias: string) => void;
  onOpenAliasSettings: (alias: string) => void;
  onDeleteAlias: (alias: string) => void;
  onEditProvider: (provider: string) => void;
  onDeleteProvider: (provider: string) => void;
  onOpenSourceSettings: (sourceId: string) => void;
}

export function DiagramContextMenu({
  contextMenu,
  t,
  onRequestClose,
  onAddAlias,
  onRenameAlias,
  onOpenAliasSettings,
  onDeleteAlias,
  onEditProvider,
  onDeleteProvider,
  onOpenSourceSettings
}: DiagramContextMenuProps) {
  const menuRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!contextMenu) return;
    const handleClick = (event: globalThis.MouseEvent) => {
      if (!menuRef.current?.contains(event.target as Node)) {
        onRequestClose();
      }
    };
    const handleKeyDown = (event: globalThis.KeyboardEvent) => {
      if (event.key === 'Escape') {
        onRequestClose();
      }
    };
    document.addEventListener('mousedown', handleClick);
    document.addEventListener('keydown', handleKeyDown);
    // Move focus into the menu so keyboard users can operate it immediately
    menuRef.current?.querySelector<HTMLButtonElement>('button')?.focus();
    return () => {
      document.removeEventListener('mousedown', handleClick);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [contextMenu, onRequestClose]);

  if (!contextMenu) return null;

  const { type, data } = contextMenu;

  const menuItemCls =
    'block w-full px-3 py-2 text-left text-[13px] text-foreground cursor-pointer flex items-center gap-2 hover:bg-muted';
  const menuItemDangerCls =
    'block w-full px-3 py-2 text-left text-[13px] text-destructive cursor-pointer flex items-center gap-2 hover:bg-destructive/10';

  const renderBackground = () => (
    <button type="button" className={menuItemCls} onClick={onAddAlias}>
      <span>{t('oauth_model_alias.diagram_add_alias')}</span>
    </button>
  );

  const renderAlias = () => {
    if (!data) return null;
    return (
      <>
        <button type="button" className={menuItemCls} onClick={() => onRenameAlias(data)}>
          <span>{t('oauth_model_alias.diagram_rename')}</span>
        </button>
        <button type="button" className={menuItemCls} onClick={() => onOpenAliasSettings(data)}>
          <span>{t('oauth_model_alias.diagram_settings')}</span>
        </button>
        <div className="h-px my-1 bg-border pointer-events-none" />
        <button type="button" className={menuItemDangerCls} onClick={() => onDeleteAlias(data)}>
          <span>{t('oauth_model_alias.diagram_delete_alias')}</span>
        </button>
      </>
    );
  };

  const renderProvider = () => {
    if (!data) return null;
    return (
      <>
        <button type="button" className={menuItemCls} onClick={() => onEditProvider(data)}>
          <span>{t('common.edit')}</span>
        </button>
        <div className="h-px my-1 bg-border pointer-events-none" />
        <button type="button" className={menuItemDangerCls} onClick={() => onDeleteProvider(data)}>
          <span>{t('oauth_model_alias.delete')}</span>
        </button>
      </>
    );
  };

  const renderSource = () => {
    if (!data) return null;
    return (
      <button type="button" className={menuItemCls} onClick={() => onOpenSourceSettings(data)}>
        <span>{t('oauth_model_alias.diagram_settings')}</span>
      </button>
    );
  };

  return createPortal(
    <div
      ref={menuRef}
      className="fixed bg-background border border-border shadow-[0_4px_12px_rgba(0,0,0,0.15)] z-[9999] min-w-[120px] overflow-hidden py-1"
      style={{ top: contextMenu.y, left: contextMenu.x }}
      onClick={(e) => e.stopPropagation()}
    >
      {type === 'background' && renderBackground()}
      {type === 'alias' && renderAlias()}
      {type === 'provider' && renderProvider()}
      {type === 'source' && renderSource()}
    </div>,
    document.body
  );
}
