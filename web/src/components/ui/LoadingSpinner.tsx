import { Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';

function LoadingSpinner({ size = 20, className = '' }: { size?: number; className?: string }) {
  return <Loader2 className={cn('animate-spin', className)} size={size} aria-hidden="true" />;
}

export { LoadingSpinner };
