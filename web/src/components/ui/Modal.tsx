import type { PropsWithChildren, ReactNode } from 'react';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from './dialog';

interface ModalProps {
  open: boolean;
  title?: ReactNode;
  onClose: () => void;
  footer?: ReactNode;
  width?: number | string;
  className?: string;
  closeDisabled?: boolean;
}

export function Modal({
  open,
  title,
  onClose,
  footer,
  width,
  className,
  closeDisabled,
  children,
}: PropsWithChildren<ModalProps>) {
  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen && !closeDisabled) {
      onClose();
    }
  };

  const widthStyle = width
    ? { maxWidth: typeof width === 'number' ? `${width}px` : width }
    : undefined;

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent
        className={className}
        style={widthStyle}
        showCloseButton={!closeDisabled}
        onPointerDownOutside={closeDisabled ? (e) => e.preventDefault() : undefined}
        onEscapeKeyDown={closeDisabled ? (e) => e.preventDefault() : undefined}
      >
        {title != null && (
          <DialogHeader>
            <DialogTitle>{title}</DialogTitle>
          </DialogHeader>
        )}
        {children}
        {footer != null && <DialogFooter>{footer}</DialogFooter>}
      </DialogContent>
    </Dialog>
  );
}
