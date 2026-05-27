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

interface TableProps extends TableHTMLAttributes<HTMLTableElement> {
  className?: string;
  cols?: ReactNode;
}

function Table({ children, cols, className, ...rest }: PropsWithChildren<TableProps>) {
  return (
    <div className="overflow-auto">
      <ShadcnTable className={className} {...rest}>
        {cols ? <colgroup>{cols}</colgroup> : null}
        {children}
      </ShadcnTable>
    </div>
  );
}

function TableHeader({
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

function TableBody({
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

interface TableRowProps extends HTMLAttributes<HTMLTableRowElement> {
  selected?: boolean;
}

function TableRow({ children, className, selected, ...rest }: PropsWithChildren<TableRowProps>) {
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

interface TableHeadProps extends ThHTMLAttributes<HTMLTableCellElement> {
  alignRight?: boolean;
}

function TableHead({
  children,
  className,
  alignRight,
  ...rest
}: PropsWithChildren<TableHeadProps>) {
  return (
    <ShadcnTableHead className={cn(alignRight && 'text-right', className)} {...rest}>
      {children}
    </ShadcnTableHead>
  );
}

interface TableCellProps extends TdHTMLAttributes<HTMLTableCellElement> {
  alignRight?: boolean;
}

function TableCell({
  children,
  className,
  alignRight,
  ...rest
}: PropsWithChildren<TableCellProps>) {
  return (
    <ShadcnTableCell className={cn(alignRight && 'text-right', className)} {...rest}>
      {children}
    </ShadcnTableCell>
  );
}

export { Table, TableHeader, TableBody, TableRow, TableHead, TableCell };
