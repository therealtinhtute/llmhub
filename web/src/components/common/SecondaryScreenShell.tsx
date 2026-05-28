import { forwardRef, useLayoutEffect, useRef, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { Button } from '@/components/ui/Button';
import { LoadingSpinner } from '@/components/ui/LoadingSpinner';
import { IconChevronLeft } from '@/components/ui/icons';
import { usePageTransitionLayer } from './PageTransitionLayer';

export type SecondaryScreenShellProps = {
  title: ReactNode;
  onBack?: () => void;
  backLabel?: string;
  backAriaLabel?: string;
  rightAction?: ReactNode;
  hideTopBarBackButton?: boolean;
  hideTopBarRightAction?: boolean;
  floatingAction?: ReactNode;
  isLoading?: boolean;
  loadingLabel?: ReactNode;
  className?: string;
  contentClassName?: string;
  children?: ReactNode;
};

export const SecondaryScreenShell = forwardRef<HTMLDivElement, SecondaryScreenShellProps>(
  function SecondaryScreenShell(
    {
      title,
      onBack,
      backLabel = 'Back',
      backAriaLabel,
      rightAction,
      hideTopBarBackButton = false,
      hideTopBarRightAction = false,
      floatingAction,
      isLoading = false,
      loadingLabel = 'Loading...',
      className = '',
      contentClassName = '',
      children,
    },
    ref
  ) {
    const containerClassName = ['flex flex-col gap-4 min-h-0', className].filter(Boolean).join(' ');
    const contentClasses = [
      'flex flex-col gap-4',
      floatingAction
        ? '[padding-bottom:calc(var(--secondary-shell-floating-action-height,56px)+12px+env(safe-area-inset-bottom))]'
        : '',
      contentClassName,
    ]
      .filter(Boolean)
      .join(' ');
    const titleTooltip = typeof title === 'string' ? title : undefined;
    const resolvedBackAriaLabel = backAriaLabel ?? backLabel;
    const pageTransitionLayer = usePageTransitionLayer();
    const isCurrentLayer = pageTransitionLayer ? pageTransitionLayer.isCurrentLayer : true;
    const shouldRenderFloatingAction = Boolean(floatingAction) && isCurrentLayer;
    const floatingActionRef = useRef<HTMLDivElement | null>(null);

    useLayoutEffect(() => {
      if (!shouldRenderFloatingAction) return;

      const element = floatingActionRef.current;
      if (!element) return;

      const updateHeight = () => {
        const height = element.getBoundingClientRect().height;
        document.documentElement.style.setProperty(
          '--secondary-shell-floating-action-height',
          `${height}px`
        );
      };

      updateHeight();
      window.addEventListener('resize', updateHeight);

      const resizeObserver =
        typeof ResizeObserver === 'undefined' ? null : new ResizeObserver(updateHeight);
      resizeObserver?.observe(element);

      return () => {
        resizeObserver?.disconnect();
        window.removeEventListener('resize', updateHeight);
        document.documentElement.style.removeProperty('--secondary-shell-floating-action-height');
      };
    }, [shouldRenderFloatingAction]);

    return (
      <>
        <div className={containerClassName} ref={ref}>
          {/* Top bar: 3-column grid — back | title | right */}
          <div className="sticky top-0 z-5 grid grid-cols-[1fr_auto_1fr] items-center gap-3 px-3 py-2 bg-muted border-b border-border min-h-[44px]">
            {onBack && !hideTopBarBackButton ? (
              <Button
                variant="ghost"
                size="sm"
                onClick={onBack}
                className="justify-self-start gap-0 pl-[6px] pr-[10px]"
                aria-label={resolvedBackAriaLabel}
              >
                <span className="inline-flex items-center justify-center [&_svg]:block">
                  <IconChevronLeft size={18} />
                </span>
                <span className="font-semibold leading-[18px]">{backLabel}</span>
              </Button>
            ) : (
              <div />
            )}
            <div
              className="min-w-0 text-center text-base font-[650] text-foreground overflow-hidden text-ellipsis whitespace-nowrap justify-self-center"
              title={titleTooltip}
            >
              {title}
            </div>
            <div className="justify-self-end flex justify-end">
              {hideTopBarRightAction ? null : rightAction}
            </div>
          </div>

          {isLoading ? (
            <div className="flex items-center justify-center gap-2 py-12 text-muted-foreground">
              <LoadingSpinner size={16} />
              <span>{loadingLabel}</span>
            </div>
          ) : (
            <div className={contentClasses}>{children}</div>
          )}
        </div>
        {shouldRenderFloatingAction && typeof document !== 'undefined'
          ? createPortal(
              <div
                className="fixed z-50 pointer-events-none w-fit max-w-[calc(100vw-24px)] max-[1200px]:max-w-[calc(100vw-16px)]"
                style={{
                  left: 'var(--content-center-x, 50%)',
                  bottom: 'calc(12px + env(safe-area-inset-bottom))',
                  transform: 'translateX(-50%)',
                }}
              >
                <div
                  className="pointer-events-auto inline-flex items-center gap-2 px-3 py-[10px] rounded-full border border-border/50 bg-background/80 backdrop-blur-md shadow-[0_4px_24px_rgba(0,0,0,0.08)] dark:shadow-[0_4px_24px_rgba(0,0,0,0.4)] max-[1200px]:px-[10px] max-[1200px]:py-2"
                  ref={floatingActionRef}
                >
                  {floatingAction}
                </div>
              </div>,
              document.body
            )
          : null}
      </>
    );
  }
);
