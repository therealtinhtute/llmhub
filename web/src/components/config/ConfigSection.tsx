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
      'flex flex-col gap-[clamp(16px,2vw,22px)] pt-3 h-[clamp(520px,calc(100dvh-var(--header-height,64px)-250px),780px)] max-md:h-[clamp(420px,calc(100dvh-var(--header-height,64px)-260px),680px)] min-w-0 box-border overflow-y-auto overscroll-auto bg-[color-mix(in_srgb,var(--background)_82%,transparent)] scroll-mt-[104px] max-md:scroll-mt-[92px] snap-start [scroll-snap-stop:always] [scrollbar-width:thin] max-md:gap-[14px]',
      className,
    ]
      .filter(Boolean)
      .join(' ');

    return (
      <section ref={ref} className={sectionClassName} {...rest}>
        <div className="flex flex-col gap-4 w-full min-w-0">{children}</div>
      </section>
    );
  }
);
