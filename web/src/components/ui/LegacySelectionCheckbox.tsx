import type { ReactNode } from 'react';
import { Checkbox } from './checkbox';
import { cn } from '@/lib/utils';

interface SelectionCheckboxProps {
  checked: boolean;
  onChange: (value: boolean) => void;
  label?: ReactNode;
  ariaLabel?: string;
  title?: string;
  disabled?: boolean;
  className?: string;
  labelClassName?: string;
}

function SelectionCheckbox({
  checked,
  onChange,
  label,
  ariaLabel,
  title,
  disabled,
  className,
  labelClassName,
}: SelectionCheckboxProps) {
  const checkbox = (
    <Checkbox
      checked={checked}
      onCheckedChange={(value) => onChange(value === true)}
      disabled={disabled}
      aria-label={ariaLabel ?? (typeof label === 'string' ? label : undefined)}
      title={title}
    />
  );

  if (!label) return <span className={className}>{checkbox}</span>;

  return (
    <label
      className={cn('inline-flex items-center gap-2', disabled && 'opacity-50', className)}
      title={title}
    >
      {checkbox}
      <span className={cn(labelClassName)}>{label}</span>
    </label>
  );
}

export { SelectionCheckbox };
