import { ReactNode, useCallback, useLayoutEffect, useRef, useState } from 'react';
import { useLocation, type Location } from 'react-router-dom';
import {
  PAGE_TRANSITION_LAYER_CONTEXT_VALUES,
  PageTransitionLayerContext,
  type LayerStatus,
} from './PageTransitionLayer';

interface PageTransitionProps {
  render: (location: Location) => ReactNode;
  getRouteOrder?: (pathname: string) => number | null;
  getTransitionVariant?: (fromPathname: string, toPathname: string) => TransitionVariant;
  scrollContainerRef?: React.RefObject<HTMLElement | null>;
}

type Layer = {
  key: string;
  location: Location;
  status: LayerStatus;
};

type TransitionDirection = 'forward' | 'backward';

type TransitionVariant = 'vertical' | 'ios';

export function PageTransition({
  render,
  getRouteOrder,
  getTransitionVariant,
  scrollContainerRef,
}: PageTransitionProps) {
  const location = useLocation();
  const scrollPositionsRef = useRef(new Map<string, number>());
  const nextLayersRef = useRef<Layer[] | null>(null);

  const [layers, setLayers] = useState<Layer[]>(() => [
    {
      key: location.key,
      location,
      status: 'current',
    },
  ]);
  const currentLayer =
    layers.find((layer) => layer.status === 'current') ?? layers[layers.length - 1];
  const currentLayerKey = currentLayer?.key ?? location.key;
  const currentLayerPathname = currentLayer?.location.pathname;

  const resolveScrollContainer = useCallback(() => {
    if (scrollContainerRef?.current) return scrollContainerRef.current;
    if (typeof document === 'undefined') return null;
    return document.scrollingElement as HTMLElement | null;
  }, [scrollContainerRef]);

  useLayoutEffect(() => {
    if (location.key === currentLayerKey) return;
    if (currentLayerPathname === location.pathname) return;
    const scrollContainer = resolveScrollContainer();
    const exitScrollOffset = scrollContainer?.scrollTop ?? 0;
    scrollPositionsRef.current.set(currentLayerKey, exitScrollOffset);
    const resolveOrderIndex = (pathname?: string) => {
      if (!getRouteOrder || !pathname) return null;
      const index = getRouteOrder(pathname);
      return typeof index === 'number' && index >= 0 ? index : null;
    };
    const fromIndex = resolveOrderIndex(currentLayerPathname);
    const toIndex = resolveOrderIndex(location.pathname);
    const nextVariant: TransitionVariant = getTransitionVariant
      ? getTransitionVariant(currentLayerPathname ?? '', location.pathname)
      : 'vertical';

    let nextDirection: TransitionDirection =
      fromIndex === null || toIndex === null || fromIndex === toIndex
        ? 'forward'
        : toIndex > fromIndex
          ? 'forward'
          : 'backward';

    // When using iOS-style stacking, history POP within the same "section" can have equal route order.
    // In that case, prefer treating navigation to an existing layer as a backward (pop) transition.
    if (nextVariant === 'ios' && layers.some((layer) => layer.key === location.key)) {
      nextDirection = 'backward';
    }

    const shouldSkipExitLayer = (() => {
      if (nextVariant !== 'ios' || nextDirection !== 'backward') return false;
      const normalizeSegments = (pathname: string) =>
        pathname
          .split('/')
          .filter(Boolean)
          .filter((segment) => segment.length > 0);
      const fromSegments = normalizeSegments(currentLayerPathname ?? '');
      const toSegments = normalizeSegments(location.pathname);
      if (!fromSegments.length || !toSegments.length) return false;
      return fromSegments[0] === toSegments[0] && toSegments.length === 1;
    })();

    setLayers((prev) => {
      const variant = nextVariant;
      const direction = nextDirection;
      const previousCurrentIndex = prev.findIndex((layer) => layer.status === 'current');
      const resolvedCurrentIndex =
        previousCurrentIndex >= 0 ? previousCurrentIndex : prev.length - 1;
      const previousCurrent = prev[resolvedCurrentIndex];
      const previousStack: Layer[] = prev
        .filter((_, idx) => idx !== resolvedCurrentIndex)
        .map((layer): Layer => ({ ...layer, status: 'stacked' }));

      const nextCurrent: Layer = { key: location.key, location, status: 'current' };

      if (!previousCurrent) {
        nextLayersRef.current = [nextCurrent];
        return [nextCurrent];
      }

      if (variant === 'ios') {
        if (direction === 'forward') {
          const exitingLayer: Layer = { ...previousCurrent, status: 'exiting' };
          const stackedLayer: Layer = { ...previousCurrent, status: 'stacked' };

          nextLayersRef.current = [...previousStack, stackedLayer, nextCurrent];
          return [...previousStack, exitingLayer, nextCurrent];
        }

        const targetIndex = prev.findIndex((layer) => layer.key === location.key);
        if (targetIndex !== -1) {
          const targetStack: Layer[] = prev.slice(0, targetIndex + 1).map((layer, idx): Layer => {
            const isTarget = idx === targetIndex;
            return {
              ...layer,
              location: isTarget ? location : layer.location,
              status: isTarget ? 'current' : 'stacked',
            };
          });

          if (shouldSkipExitLayer) {
            nextLayersRef.current = targetStack;
            return targetStack;
          }

          const exitingLayer: Layer = { ...previousCurrent, status: 'exiting' };
          nextLayersRef.current = targetStack;
          return [...targetStack, exitingLayer];
        }
      }

      if (shouldSkipExitLayer) {
        nextLayersRef.current = [nextCurrent];
        return [nextCurrent];
      }

      const exitingLayer: Layer = { ...previousCurrent, status: 'exiting' };

      nextLayersRef.current = [nextCurrent];
      return [exitingLayer, nextCurrent];
    });
  }, [
    location,
    currentLayerKey,
    currentLayerPathname,
    getRouteOrder,
    getTransitionVariant,
    resolveScrollContainer,
    layers,
  ]);

  return (
    <div className={`relative flex-[1_1_auto] flex flex-col min-h-0 overflow-hidden`}>
      {(() => {
        const currentIndex = layers.findIndex((layer) => layer.status === 'current');
        const resolvedCurrentIndex = currentIndex === -1 ? layers.length - 1 : currentIndex;
        const keepStackedIndex = layers
          .slice(0, resolvedCurrentIndex)
          .map((layer, index) => ({ layer, index }))
          .reverse()
          .find(({ layer }) => layer.status === 'stacked')?.index;

        return layers.map((layer, index) => {
          const shouldKeepStacked = layer.status === 'stacked' && index === keepStackedIndex;
          const isExit = layer.status === 'exiting';
          const isStacked = layer.status === 'stacked';
          const isCurrent = layer.status === 'current';

          // Build layer className based on status
          let layerClassName: string;
          if (isExit) {
            // exit layer: absolute, fills container, no pointer events, will-change
            layerClassName =
              'flex flex-col gap-4 min-h-0 flex-1 [backface-visibility:hidden] [transform:translateZ(0)] absolute inset-0 overflow-hidden pointer-events-none [will-change:transform,opacity]';
          } else if (isStacked && shouldKeepStacked) {
            // stacked-keep: visible but invisible, absolute, no pointer events
            layerClassName =
              'flex flex-col gap-4 min-h-0 flex-1 [backface-visibility:hidden] [transform:translateZ(0)] absolute inset-0 overflow-hidden pointer-events-none opacity-0 [will-change:transform,opacity]';
          } else if (isStacked) {
            // stacked: hidden
            layerClassName =
              'hidden flex-col gap-4 min-h-0 flex-1 [backface-visibility:hidden] [transform:translateZ(0)]';
          } else {
            // current layer — no animation, always static
            layerClassName =
              'flex flex-col gap-4 min-h-0 flex-1 [backface-visibility:hidden] [transform:translateZ(0)]';
          }

          return (
            <div
              key={layer.key}
              className={layerClassName}
              aria-hidden={!isCurrent}
              inert={!isCurrent}
            >
              <PageTransitionLayerContext.Provider
                value={{
                  ...PAGE_TRANSITION_LAYER_CONTEXT_VALUES[layer.status],
                  isAnimating: false,
                }}
              >
                {render(layer.location)}
              </PageTransitionLayerContext.Provider>
            </div>
          );
        });
      })()}
    </div>
  );
}
