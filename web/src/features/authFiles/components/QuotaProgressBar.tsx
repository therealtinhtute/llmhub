export type QuotaProgressBarProps = {
  percent: number | null;
};

export function QuotaProgressBar({ percent }: QuotaProgressBarProps) {
  const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));
  const normalized = percent === null ? null : clamp(percent, 0, 100);
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
  const widthPercent = Math.round(normalized ?? 0);

  return (
    <div className="h-2 bg-secondary overflow-hidden">
      <div className={`h-full ${fillColor}`} style={{ width: `${widthPercent}%` }} />
    </div>
  );
}

