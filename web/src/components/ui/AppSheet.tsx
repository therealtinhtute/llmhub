import type { ReactNode } from 'react';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from './sheet';
import { cn } from '@/lib/utils';

const SIZE_MAP: Record<string, string> = {
  md: 'sm:max-w-md',
  lg: 'sm:max-w-lg',
  xl: 'sm:max-w-xl',
  '2xl': 'sm:max-w-4xl',
};

interface AppSheetProps {
  open: boolean;
  onClose: () => void;
  size?: 'md' | 'lg' | 'xl' | '2xl';
  eyebrow?: ReactNode;
  title?: ReactNode;
  description?: ReactNode;
  footer?: ReactNode;
  closeDisabled?: boolean;
  confirmClose?: () => Promise<boolean>;
  children?: ReactNode;
  className?: string;
}

export function AppSheet({
  open,
  onClose,
  size = 'md',
  eyebrow,
  title,
  description,
  footer,
  closeDisabled,
  confirmClose,
  children,
  className,
}: AppSheetProps) {
  const handleOpenChange = async (nextOpen: boolean) => {
    if (nextOpen) return;
    if (closeDisabled) return;
    if (confirmClose) {
      const ok = await confirmClose();
      if (!ok) return;
    }
    onClose();
  };

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetContent
        side="right"
        showCloseButton={!closeDisabled}
        className={cn(SIZE_MAP[size], className)}
        onPointerDownOutside={(e) => {
          if (closeDisabled) e.preventDefault();
        }}
        onEscapeKeyDown={(e) => {
          if (closeDisabled) e.preventDefault();
        }}
      >
        {(eyebrow || title || description) && (
          <SheetHeader>
            {eyebrow && (
              <span className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                {eyebrow}
              </span>
            )}
            {title && <SheetTitle>{title}</SheetTitle>}
            {description && <SheetDescription>{description}</SheetDescription>}
          </SheetHeader>
        )}
        <div className="flex-1 overflow-y-auto px-4">{children}</div>
        {footer && <SheetFooter>{footer}</SheetFooter>}
      </SheetContent>
    </Sheet>
  );
}
