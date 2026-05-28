import type { DragEvent, MouseEvent as ReactMouseEvent, RefObject } from 'react';
import type { AliasNode, ProviderNode, SourceNode } from './ModelMappingDiagramTypes';

interface ProviderColumnProps {
  providerNodes: ProviderNode[];
  collapsedProviders: Set<string>;
  getProviderColor: (provider: string) => string;
  providerGroupHeights?: Record<string, number>;
  providerRefs: RefObject<Map<string, HTMLDivElement>>;
  onToggleCollapse: (provider: string) => void;
  onContextMenu: (e: ReactMouseEvent, type: 'provider' | 'background', data?: string) => void;
  label: string;
  expandLabel: string;
  collapseLabel: string;
}

export function ProviderColumn({
  providerNodes,
  collapsedProviders,
  getProviderColor,
  providerGroupHeights = {},
  providerRefs,
  onToggleCollapse,
  onContextMenu,
  label,
  expandLabel,
  collapseLabel
}: ProviderColumnProps) {
  return (
    <div
      className="flex flex-col gap-3 z-[2] shrink-0 items-end min-w-[140px]"
      onContextMenu={(e) => {
        e.preventDefault();
        e.stopPropagation();
        onContextMenu(e, 'background');
      }}
    >
      <div className="text-[13px] font-semibold text-muted-foreground uppercase mb-2 px-1">{label}</div>
      {providerNodes.map(({ provider, sources }) => {
        const collapsed = collapsedProviders.has(provider);
        const groupHeight = collapsed ? undefined : providerGroupHeights[provider];
        return (
          <div
            key={provider}
            className="flex items-center justify-end w-full"
            style={groupHeight ? { height: groupHeight } : undefined}
          >
            <div
              ref={(el) => {
                if (el) providerRefs.current?.set(provider, el);
                else providerRefs.current?.delete(provider);
              }}
              className="bg-background border border-border p-[10px_14px] text-[13px] text-foreground flex items-center justify-between w-full max-w-[280px] relative transition-all duration-200 shadow-[0_1px_2px_rgba(0,0,0,0.05)] hover:border-primary hover:-translate-y-px hover:shadow-[0_4px_6px_rgba(0,0,0,0.05)] hover:z-10 border-l-[3px] pl-2 gap-2 cursor-pointer"
              style={{ borderLeftColor: getProviderColor(provider) }}
              onContextMenu={(e) => {
                e.preventDefault();
                e.stopPropagation();
                onContextMenu(e, 'provider', provider);
              }}
            >
              <button
                type="button"
                className="shrink-0 w-6 h-6 flex items-center justify-center border-none bg-muted cursor-pointer text-muted-foreground transition-[background-color,color] duration-150 hover:bg-border hover:text-foreground"
                onClick={() => onToggleCollapse(provider)}
                aria-label={collapsed ? expandLabel : collapseLabel}
                title={collapsed ? expandLabel : collapseLabel}
              >
                {collapsed ? (
                  <span className="inline-block w-0 h-0 border-solid border-y-[4px] border-y-transparent border-l-[5px] border-l-current" />
                ) : (
                  <span className="inline-block w-0 h-0 border-solid border-x-[4px] border-x-transparent border-t-[5px] border-t-current" />
                )}
              </button>
              <span className="font-semibold text-[13px] flex-1 overflow-hidden text-ellipsis whitespace-nowrap" style={{ color: getProviderColor(provider) }}>
                {provider}
              </span>
              <span className="text-[11px] text-muted-foreground/60 ml-2 bg-muted px-[6px] py-[1px]">{sources.length}</span>
            </div>
          </div>
        );
      })}
    </div>
  );
}

interface SourceColumnProps {
  providerNodes: ProviderNode[];
  collapsedProviders: Set<string>;
  sourceRefs: RefObject<Map<string, HTMLDivElement>>;
  getProviderColor: (provider: string) => string;
  selectedSourceId?: string | null;
  onSelectSource?: (source: SourceNode) => void;
  draggedSource: SourceNode | null;
  dropTargetSource: string | null;
  draggable: boolean;
  onDragStart: (e: DragEvent, source: SourceNode) => void;
  onDragEnd: () => void;
  onDragOver: (e: DragEvent, source: SourceNode) => void;
  onDragLeave: () => void;
  onDrop: (e: DragEvent, source: SourceNode) => void;
  onContextMenu: (e: ReactMouseEvent, type: 'source' | 'background', data?: string) => void;
  label: string;
}

export function SourceColumn({
  providerNodes,
  collapsedProviders,
  sourceRefs,
  getProviderColor,
  selectedSourceId,
  onSelectSource,
  draggedSource,
  dropTargetSource,
  draggable,
  onDragStart,
  onDragEnd,
  onDragOver,
  onDragLeave,
  onDrop,
  onContextMenu,
  label
}: SourceColumnProps) {
  return (
    <div
      className="flex flex-col gap-3 z-[2] shrink-0 items-start min-w-[200px]"
      onContextMenu={(e) => {
        e.preventDefault();
        e.stopPropagation();
        onContextMenu(e, 'background');
      }}
    >
      <div className="text-[13px] font-semibold text-muted-foreground uppercase mb-2 px-1">{label}</div>
      {providerNodes.flatMap(({ provider, sources }) => {
        if (collapsedProviders.has(provider)) return [];
        return sources.map((source) => (
          <div
            key={source.id}
            ref={(el) => {
              if (el) sourceRefs.current?.set(source.id, el);
              else sourceRefs.current?.delete(source.id);
            }}
            className={`bg-background border border-border p-[10px_14px] text-[13px] text-foreground flex items-center justify-between w-full max-w-[280px] relative transition-all duration-200 shadow-[0_1px_2px_rgba(0,0,0,0.05)] hover:border-primary hover:-translate-y-px hover:shadow-[0_4px_6px_rgba(0,0,0,0.05)] hover:z-10 cursor-grab active:cursor-grabbing ${draggedSource?.id === source.id ? 'opacity-50 border-dashed' : ''} ${dropTargetSource === source.id ? 'bg-muted border-primary border-[2px]' : ''} ${selectedSourceId === source.id ? 'border-primary bg-muted shadow-[0_0_0_2px_hsl(var(--primary)/0.18)]' : ''}`}
            onClick={() => onSelectSource?.(source)}
            draggable={draggable}
            onDragStart={(e) => onDragStart(e, source)}
            onDragEnd={onDragEnd}
            onDragOver={(e) => onDragOver(e, source)}
            onDragLeave={onDragLeave}
            onDrop={(e) => onDrop(e, source)}
            onContextMenu={(e) => {
              e.preventDefault();
              e.stopPropagation();
              onContextMenu(e, 'source', source.id);
            }}
          >
            <span className="flex-1 overflow-hidden text-ellipsis whitespace-nowrap" title={source.name}>
              {source.name}
            </span>
            <div
              className="w-[6px] h-[6px] absolute top-1/2 -mt-[3px] right-[-3px] shrink-0"
              style={{
                background: getProviderColor(source.provider),
                opacity: source.aliases.length > 0 ? 1 : 0.3,
                borderRadius: '50%'
              }}
            />
          </div>
        ));
      })}
    </div>
  );
}

interface AliasColumnProps {
  aliasNodes: AliasNode[];
  aliasRefs: RefObject<Map<string, HTMLDivElement>>;
  dropTargetAlias: string | null;
  draggedAlias: string | null;
  selectedAlias?: string | null;
  onSelectAlias?: (alias: string) => void;
  draggable: boolean;
  onDragStart: (e: DragEvent, alias: string) => void;
  onDragEnd: () => void;
  onDragOver: (e: DragEvent, alias: string) => void;
  onDragLeave: () => void;
  onDrop: (e: DragEvent, alias: string) => void;
  onContextMenu: (e: ReactMouseEvent, type: 'alias' | 'background', data?: string) => void;
  label: string;
}

export function AliasColumn({
  aliasNodes,
  aliasRefs,
  dropTargetAlias,
  draggedAlias,
  selectedAlias,
  onSelectAlias,
  draggable,
  onDragStart,
  onDragEnd,
  onDragOver,
  onDragLeave,
  onDrop,
  onContextMenu,
  label
}: AliasColumnProps) {
  return (
    <div
      className="flex flex-col gap-3 z-[2] shrink-0 items-start min-w-[200px]"
      onContextMenu={(e) => {
        e.preventDefault();
        e.stopPropagation();
        onContextMenu(e, 'background');
      }}
    >
      <div className="text-[13px] font-semibold text-muted-foreground uppercase mb-2 px-1">{label}</div>
      {aliasNodes.map((node) => (
        <div
          key={node.id}
          ref={(el) => {
            if (el) aliasRefs.current?.set(node.id, el);
            else aliasRefs.current?.delete(node.id);
          }}
          className={`bg-background border border-border p-[10px_14px] text-[13px] text-foreground flex items-center justify-between w-full max-w-[280px] relative transition-all duration-200 shadow-[0_1px_2px_rgba(0,0,0,0.05)] hover:border-primary hover:-translate-y-px hover:shadow-[0_4px_6px_rgba(0,0,0,0.05)] hover:z-10 cursor-grab active:cursor-grabbing ${dropTargetAlias === node.alias ? 'bg-muted border-primary border-[2px]' : ''} ${draggedAlias === node.alias ? 'opacity-50 border-dashed' : ''} ${selectedAlias === node.alias ? 'border-primary bg-muted shadow-[0_0_0_2px_hsl(var(--primary)/0.18)]' : ''}`}
          onClick={() => onSelectAlias?.(node.alias)}
          draggable={draggable}
          onDragStart={(e) => onDragStart(e, node.alias)}
          onDragEnd={onDragEnd}
          onDragOver={(e) => onDragOver(e, node.alias)}
          onDragLeave={onDragLeave}
          onDrop={(e) => onDrop(e, node.alias)}
          onContextMenu={(e) => {
            e.preventDefault();
            e.stopPropagation();
            onContextMenu(e, 'alias', node.alias);
          }}
        >
          <div className="w-[6px] h-[6px] absolute top-1/2 -mt-[3px] left-[-3px] shrink-0 bg-muted-foreground/60" style={{ borderRadius: '50%' }} />
          <span className="flex-1 overflow-hidden text-ellipsis whitespace-nowrap" title={node.alias}>
            {node.alias}
          </span>
          <span className="text-[11px] text-muted-foreground/60 ml-2 bg-muted px-[6px] py-[1px]">{node.sources.length}</span>
        </div>
      ))}
    </div>
  );
}
