import type { PropsWithChildren, ReactNode } from 'react';
import {
  Card,
  CardAction,
  CardContent,
  CardHeader,
  CardTitle,
} from './Card';
import { cn } from '@/lib/utils';

interface AppCardProps {
  title?: ReactNode;
  extra?: ReactNode;
  className?: string;
}

export function AppCard({
  title,
  extra,
  children,
  className,
}: PropsWithChildren<AppCardProps>) {
  return (
    <Card className={cn(className)}>
      {(title || extra) && (
        <CardHeader>
          {title && <CardTitle>{title}</CardTitle>}
          {extra && <CardAction>{extra}</CardAction>}
        </CardHeader>
      )}
      <CardContent>{children}</CardContent>
    </Card>
  );
}
