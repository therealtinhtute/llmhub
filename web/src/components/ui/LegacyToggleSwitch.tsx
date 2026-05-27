import type { ReactNode } from 'react';
import { Switch } from './switch';
import { cn } from '@/lib/utils';

interface ToggleSwitchProps {
  checked: boolean;
  onChange: (value: boolean) => void;
  label?: ReactNode;
  ariaLabel?: string;
  disabled?: boolean;
  labelPosition?: 'left' | 'right';
}

function ToggleSwitch({
  checked,
  onChange,
  label,
  ariaLabel,
  disabled,
  labelPosition = 'right',
}: ToggleSwitchProps) {
  const switchElement = (
    <Switch
      checked={checked}
      onCheckedChange={onChange}
      disabled={disabled}
      aria-label={ariaLabel}
    />
  );

  if (!label) return switchElement;

  return (
    <label className={cn('inline-flex items-center gap-2', disabled && 'opacity-50')}>
      {labelPosition === 'left' ? (
        <>
          <span className="text-sm">{label}</span>
          {switchElement}
        </>
      ) : (
        <>
          {switchElement}
          <span className="text-sm">{label}</span>
        </>
      )}
    </label>
  );
}

export { ToggleSwitch };
