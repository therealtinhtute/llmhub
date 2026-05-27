import { type HTMLAttributes, type PropsWithChildren, type ReactNode } from 'react';
import { ChevronDown } from 'lucide-react';
import { cn } from '@/lib/utils';

interface CollapsibleProps extends HTMLAttributes<HTMLDetailsElement> {
  label: ReactNode;
  hint?: ReactNode;
  defaultOpen?: boolean;
  open?: boolean;
  onToggle?: (event: React.SyntheticEvent<HTMLDetailsElement>) => void;
  flush?: boolean;
}

function Collapsible({
  label,
  hint,
  defaultOpen,
  open,
  onToggle,
  flush,
  children,
  className,
  ...rest
}: PropsWithChildren<CollapsibleProps>) {
  return (
    <details
      className={cn('group border border-border rounded-md', flush && 'border-0', className)}
      open={open ?? defaultOpen}
      onToggle={onToggle}
      {...rest}
    >
      <summary className="flex cursor-pointer list-none items-center gap-2 px-3 py-2 text-sm font-medium select-none [&::-webkit-details-marker]:hidden">
        <ChevronDown
          className="size-4 shrink-0 transition-transform duration-200 group-open:rotate-180"
          aria-hidden="true"
        />
        <span className="flex-1">{label}</span>
        {hint != null ? (
          <span className="text-xs text-muted-foreground">{hint}</span>
        ) : null}
      </summary>
      <div className="px-3 pb-3">{children}</div>
    </details>
  );
}

export { Collapsible };
