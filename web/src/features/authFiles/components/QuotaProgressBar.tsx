export type QuotaProgressBarProps = {
  percent: number | null;
  highThreshold: number;
  mediumThreshold: number;
};

export function QuotaProgressBar({ percent, highThreshold, mediumThreshold }: QuotaProgressBarProps) {
  const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));
  const normalized = percent === null ? null : clamp(percent, 0, 100);
  const fillColor =
    normalized === null
      ? 'bg-[var(--quota-medium-color,#e0aa14)]'
      : normalized >= highThreshold
        ? 'bg-success'
        : normalized >= mediumThreshold
          ? 'bg-[var(--quota-medium-color,#e0aa14)]'
          : 'bg-destructive';
  const widthPercent = Math.round(normalized ?? 0);

  return (
    <div className="h-2 bg-secondary overflow-hidden">
      <div className={`h-full transition-[width] duration-200 ease-out ${fillColor}`} style={{ width: `${widthPercent}%` }} />
    </div>
  );
}

