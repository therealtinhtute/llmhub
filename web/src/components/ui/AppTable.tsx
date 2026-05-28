import {
  type HTMLAttributes,
  type PropsWithChildren,
  type ReactNode,
  type TableHTMLAttributes,
  type TdHTMLAttributes,
  type ThHTMLAttributes,
} from 'react';
import {
  Table as ShadcnTable,
  TableHeader as ShadcnTableHeader,
  TableBody as ShadcnTableBody,
  TableRow as ShadcnTableRow,
  TableHead as ShadcnTableHead,
  TableCell as ShadcnTableCell,
} from './table';
import { cn } from '@/lib/utils';

interface AppTableProps extends TableHTMLAttributes<HTMLTableElement> {
  className?: string;
  cols?: ReactNode;
}

function AppTable({ children, cols, className, ...rest }: PropsWithChildren<AppTableProps>) {
  return (
    <div className="overflow-auto">
      <ShadcnTable className={className} {...rest}>
        {cols ? <colgroup>{cols}</colgroup> : null}
        {children}
      </ShadcnTable>
    </div>
  );
}

function AppTableHeader({
  children,
  className,
  ...rest
}: PropsWithChildren<HTMLAttributes<HTMLTableSectionElement>>) {
  return (
    <ShadcnTableHeader className={className} {...rest}>
      {children}
    </ShadcnTableHeader>
  );
}

function AppTableBody({
  children,
  className,
  ...rest
}: PropsWithChildren<HTMLAttributes<HTMLTableSectionElement>>) {
  return (
    <ShadcnTableBody className={className} {...rest}>
      {children}
    </ShadcnTableBody>
  );
}

interface AppTableRowProps extends HTMLAttributes<HTMLTableRowElement> {
  selected?: boolean;
}

function AppTableRow({ children, className, selected, ...rest }: PropsWithChildren<AppTableRowProps>) {
  return (
    <ShadcnTableRow
      className={cn(selected && 'bg-muted', className)}
      data-state={selected ? 'selected' : undefined}
      {...rest}
    >
      {children}
    </ShadcnTableRow>
  );
}

interface AppTableHeadProps extends ThHTMLAttributes<HTMLTableCellElement> {
  alignRight?: boolean;
}

function AppTableHead({
  children,
  className,
  alignRight,
  ...rest
}: PropsWithChildren<AppTableHeadProps>) {
  return (
    <ShadcnTableHead className={cn(alignRight && 'text-right', className)} {...rest}>
      {children}
    </ShadcnTableHead>
  );
}

interface AppTableCellProps extends TdHTMLAttributes<HTMLTableCellElement> {
  alignRight?: boolean;
}

function AppTableCell({
  children,
  className,
  alignRight,
  ...rest
}: PropsWithChildren<AppTableCellProps>) {
  return (
    <ShadcnTableCell className={cn(alignRight && 'text-right', className)} {...rest}>
      {children}
    </ShadcnTableCell>
  );
}

export { AppTable, AppTableHeader, AppTableBody, AppTableRow, AppTableHead, AppTableCell };
