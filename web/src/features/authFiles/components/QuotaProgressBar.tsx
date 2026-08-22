import { useEffect, useState } from 'react';

export type QuotaProgressBarProps = {
  percent: number | null;
};

const clampPercent = (value: number) => Math.min(100, Math.max(0, value));

const prefersReducedMotion = () =>
  typeof window !== 'undefined' &&
  window.matchMedia('(prefers-reduced-motion: reduce)').matches;

export function QuotaProgressBar({ percent }: QuotaProgressBarProps) {
  const normalized = percent === null ? null : clampPercent(percent);
  const targetWidth = Math.round(normalized ?? 0);
  const [renderedWidth, setRenderedWidth] = useState(() =>
    prefersReducedMotion() ? targetWidth : 0
  );

  useEffect(() => {
    if (prefersReducedMotion()) {
      const frame = requestAnimationFrame(() => setRenderedWidth(targetWidth));
      return () => cancelAnimationFrame(frame);
    }
    // Double rAF: guarantee one painted frame at the start width so the
    // width transition plays from the previous value (or 0 on mount).
    let innerFrame = 0;
    const outerFrame = requestAnimationFrame(() => {
      innerFrame = requestAnimationFrame(() => setRenderedWidth(targetWidth));
    });
    return () => {
      cancelAnimationFrame(outerFrame);
      cancelAnimationFrame(innerFrame);
    };
  }, [targetWidth]);

  const fillColor =
    normalized === null
      ? 'bg-amber-500'
      : normalized > 80
        ? 'bg-green-500'
        : normalized > 50
          ? 'bg-lime-500'
          : normalized > 20
            ? 'bg-amber-500'
            : normalized > 10
              ? 'bg-orange-500'
              : 'bg-destructive';

  return (
    <div className="h-2 bg-secondary overflow-hidden">
      <div
        className={`h-full transition-[width] duration-[400ms] ease-out motion-reduce:transition-none ${fillColor}`}
        style={{ width: `${renderedWidth}%` }}
      />
    </div>
  );
}
