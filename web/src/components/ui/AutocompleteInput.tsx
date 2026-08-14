import { useEffect, useId, useRef, useState, type ChangeEvent, type KeyboardEvent, type ReactNode } from 'react';
import { IconChevronDown } from './icons';
import { Input } from './Input';
import { cn } from '@/lib/utils';

interface AutocompleteInputProps {
  label?: string;
  value: string;
  onChange: (value: string) => void;
  options: string[] | { value: string; label?: string }[];
  placeholder?: string;
  disabled?: boolean;
  hint?: string;
  error?: string;
  className?: string;
  wrapperClassName?: string;
  wrapperStyle?: React.CSSProperties;
  id?: string;
  rightElement?: ReactNode;
}

export function AutocompleteInput({
  label,
  value,
  onChange,
  options,
  placeholder,
  disabled,
  hint,
  error,
  className = '',
  wrapperClassName = '',
  wrapperStyle,
  id,
  rightElement,
}: AutocompleteInputProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const containerRef = useRef<HTMLDivElement>(null);
  const listboxId = useId();
  const errorId = error && id ? `${id}-error` : undefined;
  const hintId = hint && !error && id ? `${id}-hint` : undefined;

  const normalizedOptions = options.map((opt) =>
    typeof opt === 'string'
      ? { value: opt, label: opt }
      : { value: opt.value, label: opt.label || opt.value }
  );

  const filteredOptions = normalizedOptions.filter((opt) => {
    const v = value.toLowerCase();
    return (
      opt.value.toLowerCase().includes(v) ||
      (opt.label && opt.label.toLowerCase().includes(v))
    );
  });

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleInputChange = (e: ChangeEvent<HTMLInputElement>) => {
    onChange(e.target.value);
    setIsOpen(true);
    setHighlightedIndex(-1);
  };

  const handleSelect = (selectedValue: string) => {
    onChange(selectedValue);
    setIsOpen(false);
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (disabled) return;

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (!isOpen) {
        setIsOpen(true);
        return;
      }
      setHighlightedIndex((prev) =>
        prev < filteredOptions.length - 1 ? prev + 1 : prev
      );
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setHighlightedIndex((prev) => (prev > 0 ? prev - 1 : 0));
    } else if (e.key === 'Enter') {
      if (isOpen && highlightedIndex >= 0 && highlightedIndex < filteredOptions.length) {
        e.preventDefault();
        handleSelect(filteredOptions[highlightedIndex].value);
      } else if (isOpen) {
        e.preventDefault();
        setIsOpen(false);
      }
    } else if (e.key === 'Escape') {
      setIsOpen(false);
    } else if (e.key === 'Tab') {
      setIsOpen(false);
    }
  };

  return (
    <div className={cn('grid gap-1.5', wrapperClassName)} ref={containerRef} style={wrapperStyle}>
      {label && (
        <label htmlFor={id} className="text-sm font-medium leading-none">
          {label}
        </label>
      )}
      <div className="relative">
        <Input
          id={id}
          className={cn('pr-8', className)}
          value={value}
          onChange={handleInputChange}
          onFocus={() => setIsOpen(true)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          disabled={disabled}
          autoComplete="off"
          role="combobox"
          aria-expanded={isOpen}
          aria-controls={listboxId}
          aria-autocomplete="list"
          aria-activedescendant={
            isOpen && highlightedIndex >= 0
              ? `${listboxId}-option-${highlightedIndex}`
              : undefined
          }
          aria-invalid={error ? true : undefined}
          aria-describedby={errorId ?? hintId}
        />
        <div
          className={cn(
            'absolute right-2 top-1/2 flex -translate-y-1/2 items-center',
            disabled ? 'pointer-events-none' : 'cursor-pointer'
          )}
          onClick={() => !disabled && setIsOpen(!isOpen)}
        >
          {rightElement}
          <IconChevronDown size={16} className="ml-1 opacity-50" />
        </div>

        {isOpen && filteredOptions.length > 0 && !disabled && (
          <div
            id={listboxId}
            role="listbox"
            className="absolute top-[calc(100%+4px)] left-0 right-0 z-50 max-h-[200px] overflow-y-auto border border-border bg-popover shadow-md"
          >
            {filteredOptions.map((opt, index) => (
              <div
                key={`${opt.value}-${index}`}
                id={`${listboxId}-option-${index}`}
                role="option"
                aria-selected={index === highlightedIndex}
                onClick={() => handleSelect(opt.value)}
                onMouseEnter={() => setHighlightedIndex(index)}
                className={cn(
                  'flex cursor-pointer flex-col px-3 py-2 text-sm text-popover-foreground',
                  index === highlightedIndex && 'bg-accent'
                )}
              >
                <span className="font-medium">{opt.value}</span>
                {opt.label && opt.label !== opt.value && (
                  <span className="text-xs text-muted-foreground">{opt.label}</span>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
      {hint && (
        <div id={hintId} className="text-xs text-muted-foreground">
          {hint}
        </div>
      )}
      {error && (
        <div id={errorId} className="text-xs text-destructive">
          {error}
        </div>
      )}
    </div>
  );
}
