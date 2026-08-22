import { useEffect, useRef, useState } from 'react';

function easeOutExpo(t: number): number {
  return t >= 1 ? 1 : 1 - Math.pow(2, -10 * t);
}

export function useAnimatedNumber(value: number, duration = 700): number {
  const [display, setDisplay] = useState(value);
  const previousRef = useRef(value);

  useEffect(() => {
    const from = previousRef.current;
    const to = value;
    previousRef.current = value;
    if (from === to) return;

    const reduceMotion =
      typeof window !== 'undefined' &&
      window.matchMedia('(prefers-reduced-motion: reduce)').matches;

    if (typeof requestAnimationFrame === 'undefined' || reduceMotion) {
      const frame = requestAnimationFrame(() => setDisplay(to));
      return () => cancelAnimationFrame(frame);
    }

    let frame = 0;
    const start = performance.now();
    const tick = (now: number) => {
      const progress = Math.min((now - start) / duration, 1);
      setDisplay(from + (to - from) * easeOutExpo(progress));
      if (progress < 1) frame = requestAnimationFrame(tick);
    };
    frame = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(frame);
  }, [value, duration]);

  return display;
}
