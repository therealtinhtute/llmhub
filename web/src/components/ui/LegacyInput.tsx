import type { InputHTMLAttributes, ReactNode } from 'react';
import { Input as ShadcnInput } from './Input';
import { cn } from '@/lib/utils';

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  hint?: string;
  error?: string;
  rightElement?: ReactNode;
}

function Input({ label, hint, error, rightElement, className, id, ...rest }: InputProps) {
  return (
    <div className="flex flex-col gap-1">
      {label ? (
        <label htmlFor={id} className="text-sm font-medium text-foreground">
          {label}
        </label>
      ) : null}
      <div className={cn('relative', rightElement && 'flex items-center')}>
        <ShadcnInput
          id={id}
          className={cn(error && 'border-destructive', className)}
          aria-invalid={error ? true : undefined}
          {...rest}
        />
        {rightElement ? (
          <div className="absolute right-2 top-1/2 -translate-y-1/2">{rightElement}</div>
        ) : null}
      </div>
      {error ? <p className="text-xs text-destructive">{error}</p> : null}
      {hint && !error ? <p className="text-xs text-muted-foreground">{hint}</p> : null}
    </div>
  );
}

export { Input };
