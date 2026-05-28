import { forwardRef, type HTMLAttributes, type PropsWithChildren, type ReactNode } from 'react';

interface ConfigSectionProps extends Omit<HTMLAttributes<HTMLElement>, 'title'> {
  title: ReactNode;
  description?: ReactNode;
  indexLabel?: ReactNode;
  icon?: ReactNode;
}

export const ConfigSection = forwardRef<HTMLElement, PropsWithChildren<ConfigSectionProps>>(
  function ConfigSection(
    { title, description, indexLabel, icon, className, children, ...rest },
    ref
  ) {
    const sectionClassName = [
      'flex flex-col gap-[clamp(16px,2vw,22px)] h-[clamp(520px,calc(100dvh-var(--header-height,64px)-250px),780px)] max-md:h-[clamp(420px,calc(100dvh-var(--header-height,64px)-260px),680px)] min-w-0 box-border overflow-y-auto overscroll-auto p-[clamp(20px,2.4vw,28px)] max-md:p-4 border border-border bg-[color-mix(in_srgb,var(--background)_82%,transparent)] scroll-mt-[104px] max-md:scroll-mt-[92px] snap-start [scroll-snap-stop:always] [scrollbar-width:thin] max-md:gap-[14px]',
      className,
    ]
      .filter(Boolean)
      .join(' ');

    return (
      <section ref={ref} className={sectionClassName} {...rest}>
        <header className="grid grid-cols-[auto_minmax(0,1fr)] max-md:grid-cols-[minmax(0,1fr)] gap-[10px] items-start pb-[14px] max-md:pb-3 border-b border-border bg-[color-mix(in_srgb,var(--background)_88%,transparent)] max-md:bg-transparent">
          <div className="flex items-center gap-2 min-w-0">
            {indexLabel ? (
              <span className="inline-flex items-center justify-center min-w-[32px] h-[28px] px-2 border border-border text-muted-foreground text-[11px] font-[750] tracking-[0.08em]">
                {indexLabel}
              </span>
            ) : null}
            {icon ? (
              <span className="inline-flex items-center justify-center w-[28px] h-[28px] text-muted-foreground flex-none">
                {icon}
              </span>
            ) : null}
          </div>
          <div className="flex flex-col gap-1.5 min-w-0">
            <h2 className="m-0 text-foreground text-[clamp(18px,1.6vw,22px)] font-[680] leading-[1.18] tracking-normal">
              {title}
            </h2>
            {description ? (
              <p className="m-0 max-w-[72ch] max-md:max-w-none text-muted-foreground text-[13px] leading-[1.65]">
                {description}
              </p>
            ) : null}
          </div>
        </header>
        <div className="flex flex-col gap-4 w-full min-w-0">{children}</div>
      </section>
    );
  }
);
