import {
  Select as ShadcnSelect,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './Select';
import { cn } from '@/lib/utils';

export interface SelectOption {
  value: string;
  label: string;
}

// Radix throws if a Select.Item receives value="" (that string is reserved for
// clearing the selection). Callers legitimately model "no selection" as '', so
// translate to/from this sentinel at the boundary instead of pushing the
// constraint onto every call site.
const EMPTY_VALUE_SENTINEL = '__form-select-empty__';

interface FormSelectProps {
  value: string;
  options: ReadonlyArray<SelectOption>;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
  disabled?: boolean;
  ariaLabel?: string;
  ariaLabelledBy?: string;
  ariaDescribedBy?: string;
  fullWidth?: boolean;
  size?: 'sm' | 'md';
  id?: string;
}

export function FormSelect({
  value,
  options,
  onChange,
  placeholder,
  className,
  disabled = false,
  ariaLabel,
  ariaLabelledBy,
  ariaDescribedBy,
  fullWidth = true,
  size = 'md',
  id,
}: FormSelectProps) {
  return (
    <ShadcnSelect
      value={value === '' ? EMPTY_VALUE_SENTINEL : value}
      onValueChange={(next) => onChange(next === EMPTY_VALUE_SENTINEL ? '' : next)}
      disabled={disabled}
    >
      <SelectTrigger
        id={id}
        aria-label={ariaLabel}
        aria-labelledby={ariaLabelledBy}
        aria-describedby={ariaDescribedBy}
        className={cn(
          size === 'sm' && 'h-7 text-xs px-2',
          fullWidth && 'w-full',
          className,
        )}
      >
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>
        {options.map((opt) => (
          <SelectItem
            key={opt.value}
            value={opt.value === '' ? EMPTY_VALUE_SENTINEL : opt.value}
          >
            {opt.label}
          </SelectItem>
        ))}
      </SelectContent>
    </ShadcnSelect>
  );
}
