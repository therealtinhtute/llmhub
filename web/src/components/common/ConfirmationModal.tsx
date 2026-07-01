import { useTranslation } from 'react-i18next';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { useConfirmationStore } from '@/stores';

export function ConfirmationModal() {
  const { t } = useTranslation();
  const confirmation = useConfirmationStore((state) => state.confirmation);
  const hideConfirmation = useConfirmationStore((state) => state.hideConfirmation);
  const setConfirmationLoading = useConfirmationStore((state) => state.setConfirmationLoading);

  const { isOpen, isLoading, options } = confirmation;

  if (!isOpen || !options) {
    return null;
  }

  const { title, message, onConfirm, onCancel, confirmText, cancelText, variant = 'primary' } = options;

  const handleConfirm = async () => {
    try {
      setConfirmationLoading(true);
      await onConfirm();
      hideConfirmation();
    } catch (error) {
      console.error('Confirmation action failed:', error);
      // Optional: show error notification here if needed,
      // but usually the calling component handles specific errors.
    } finally {
      setConfirmationLoading(false);
    }
  };

  const handleCancel = () => {
    if (isLoading) {
      return;
    }
    if (onCancel) {
      onCancel();
    }
    hideConfirmation();
  };

  return (
    <AlertDialog
      open={isOpen}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) {
          handleCancel();
        }
      }}
    >
      <AlertDialogContent
        onEscapeKeyDown={isLoading ? (event) => event.preventDefault() : undefined}
      >
        <AlertDialogHeader>
          <AlertDialogTitle>{title || t('common.confirm')}</AlertDialogTitle>
          {typeof message === 'string' ? (
            <AlertDialogDescription>{message}</AlertDialogDescription>
          ) : (
            <div className="text-sm text-muted-foreground">{message}</div>
          )}
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel
            onClick={(event) => {
              event.preventDefault();
              handleCancel();
            }}
            disabled={isLoading}
          >
            {cancelText || t('common.cancel')}
          </AlertDialogCancel>
          <AlertDialogAction
            variant={variant === 'danger' ? 'destructive' : variant}
            onClick={(event) => {
              event.preventDefault();
              void handleConfirm();
            }}
            disabled={isLoading}
          >
            {isLoading ? t('common.loading') : confirmText || t('common.confirm')}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
