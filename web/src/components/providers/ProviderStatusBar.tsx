import { useState, useCallback, useRef, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import type { StatusBarData, StatusBlockDetail } from '@/utils/recentRequests';

const COLOR_STOPS = [
  { r: 239, g: 68, b: 68 },
  { r: 250, g: 204, b: 21 },
  { r: 34, g: 197, b: 94 },
] as const;

function rateToColor(rate: number): string {
  const t = Math.max(0, Math.min(1, rate));
  const segment = t < 0.5 ? 0 : 1;
  const localT = segment === 0 ? t * 2 : (t - 0.5) * 2;
  const from = COLOR_STOPS[segment];
  const to = COLOR_STOPS[segment + 1];
  const r = Math.round(from.r + (to.r - from.r) * localT);
  const g = Math.round(from.g + (to.g - from.g) * localT);
  const b = Math.round(from.b + (to.b - from.b) * localT);
  return `rgb(${r}, ${g}, ${b})`;
}

function formatTime(timestamp: number): string {
  const date = new Date(timestamp);
  const h = date.getHours().toString().padStart(2, '0');
  const m = date.getMinutes().toString().padStart(2, '0');
  return `${h}:${m}`;
}

function formatSuccessRate(rate: number): string {
  const rounded = rate.toFixed(1);
  return `${rounded.endsWith('.0') ? rounded.slice(0, -2) : rounded}%`;
}

interface ProviderStatusBarProps {
  statusData: StatusBarData;
  styles?: Record<string, string>;
}

export function ProviderStatusBar({ statusData }: ProviderStatusBarProps) {
  const { t } = useTranslation();
  const [activeTooltip, setActiveTooltip] = useState<number | null>(null);
  const blocksRef = useRef<HTMLDivElement>(null);

  const hasData = statusData.totalSuccess + statusData.totalFailure > 0;
  const rateClass = !hasData
    ? 'inline-flex items-center text-[11px] font-semibold whitespace-nowrap px-1.5 py-px bg-muted text-muted-foreground tabular-nums'
    : statusData.successRate >= 90
      ? 'inline-flex items-center text-[11px] font-semibold whitespace-nowrap px-1.5 py-px tabular-nums text-success bg-success/12'
      : statusData.successRate >= 50
        ? 'inline-flex items-center text-[11px] font-semibold whitespace-nowrap px-1.5 py-px tabular-nums text-warning bg-warning/12'
        : 'inline-flex items-center text-[11px] font-semibold whitespace-nowrap px-1.5 py-px tabular-nums text-destructive bg-destructive/10';

  useEffect(() => {
    if (activeTooltip === null) return;
    const handler = (e: PointerEvent) => {
      if (blocksRef.current && !blocksRef.current.contains(e.target as Node)) {
        setActiveTooltip(null);
      }
    };
    document.addEventListener('pointerdown', handler);
    return () => document.removeEventListener('pointerdown', handler);
  }, [activeTooltip]);

  const handlePointerEnter = useCallback((e: React.PointerEvent, idx: number) => {
    if (e.pointerType === 'mouse') setActiveTooltip(idx);
  }, []);

  const handlePointerLeave = useCallback((e: React.PointerEvent) => {
    if (e.pointerType === 'mouse') setActiveTooltip(null);
  }, []);

  const handlePointerDown = useCallback((e: React.PointerEvent, idx: number) => {
    if (e.pointerType === 'touch') {
      e.preventDefault();
      setActiveTooltip((prev) => (prev === idx ? null : idx));
    }
  }, []);

  const getTooltipPositionClass = (idx: number, total: number): string => {
    if (idx <= 2) return 'left-0 translate-x-0 [--tooltip-arrow-left:8px] [--tooltip-arrow-transform:none]';
    if (idx >= total - 3) return 'left-auto! right-0 translate-x-0 [--tooltip-arrow-right:8px] [--tooltip-arrow-transform:none] [--tooltip-arrow-left:auto]';
    return '';
  };

  const renderTooltip = (detail: StatusBlockDetail, idx: number) => {
    const total = detail.success + detail.failure;
    const posClass = getTooltipPositionClass(idx, statusData.blockDetails.length);
    const timeRange = `${formatTime(detail.startTime)} – ${formatTime(detail.endTime)}`;

    return (
      <div className={`status-bar-tooltip absolute bottom-[calc(100%+8px)] left-1/2 -translate-x-1/2 bg-background border border-border px-2.5 py-1.5 text-[11px] leading-snug whitespace-nowrap shadow-[0_4px_12px_rgba(0,0,0,0.12)] z-50 pointer-events-none text-foreground ${posClass}`}>
        <span className="text-muted-foreground block mb-0.5">{timeRange}</span>
        {total > 0 ? (
          <span className="flex items-center gap-2">
            <span className="text-success">{t('status_bar.success_short')} {detail.success}</span>
            <span className="text-destructive">{t('status_bar.failure_short')} {detail.failure}</span>
            <span className="text-muted-foreground ml-0.5">({(detail.rate * 100).toFixed(1)}%)</span>
          </span>
        ) : (
          <span className="flex items-center gap-2">{t('status_bar.no_requests')}</span>
        )}
      </div>
    );
  };

  return (
    <div className="flex items-center gap-1.5 max-w-full">
      <div className="flex gap-0.5 flex-1 min-w-[120px] relative" ref={blocksRef}>
        {statusData.blockDetails.map((detail, idx) => {
          const isIdle = detail.rate === -1;
          const blockStyle = isIdle ? undefined : { backgroundColor: rateToColor(detail.rate) };
          const isActive = activeTooltip === idx;

          return (
            <div
              key={idx}
              className={`flex-1 min-w-[4px] relative cursor-pointer [-webkit-tap-highlight-color:transparent] group${isActive ? ' is-active' : ''}`}
              onPointerEnter={(e) => handlePointerEnter(e, idx)}
              onPointerLeave={handlePointerLeave}
              onPointerDown={(e) => handlePointerDown(e, idx)}
            >
              <div
                className={`w-full h-1.5 group-hover:scale-y-[1.8] group-hover:opacity-90${isActive ? ' scale-y-[1.8] opacity-90' : ''}${isIdle ? ' bg-border' : ''}`}
                style={blockStyle}
              />
              {isActive && renderTooltip(detail, idx)}
            </div>
          );
        })}
      </div>
      <span className={rateClass}>
        {hasData ? formatSuccessRate(statusData.successRate) : '--'}
      </span>
    </div>
  );
}
