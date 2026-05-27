import type { CSSProperties, HTMLAttributes } from 'react';
import { Skeleton as ShadcnSkeleton } from './skeleton';
import { cn } from '@/lib/utils';

interface SkeletonProps extends HTMLAttributes<HTMLDivElement> {
  width?: number | string;
  height?: number | string;
  rounded?: number | string;
}

function Skeleton({ width, height, rounded, className, style, ...rest }: SkeletonProps) {
  const inlineStyle: CSSProperties = { ...style };
  if (width !== undefined) inlineStyle.width = typeof width === 'number' ? `${width}px` : width;
  if (height !== undefined)
    inlineStyle.height = typeof height === 'number' ? `${height}px` : height;
  if (rounded !== undefined)
    inlineStyle.borderRadius = typeof rounded === 'number' ? `${rounded}px` : rounded;

  return <ShadcnSkeleton className={cn(className)} style={inlineStyle} {...rest} />;
}

export { Skeleton };
