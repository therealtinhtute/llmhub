import { CSSProperties, ReactNode, useEffect, useRef } from 'react';
import { cn } from '@/lib/utils';

interface RevealProps {
  children: ReactNode;
  delay?: number;
  className?: string;
}

export function Reveal({ children, delay = 0, className }: RevealProps) {
  const ref = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    if (
      typeof IntersectionObserver === 'undefined' ||
      window.matchMedia('(prefers-reduced-motion: reduce)').matches
    ) {
      el.classList.add('is-inview');
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            el.classList.add('is-inview');
            observer.disconnect();
          }
        }
      },
      { threshold: 0.05, rootMargin: '0px 0px -6% 0px' }
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  const style = delay
    ? ({ '--reveal-delay': `${Math.min(delay, 400)}ms` } as CSSProperties)
    : undefined;

  return (
    <div ref={ref} className={cn('reveal', className)} style={style}>
      {children}
    </div>
  );
}
