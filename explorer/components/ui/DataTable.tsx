"use client";

import React from "react";
import {
  ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable,
  getPaginationRowModel,
  getSortedRowModel,
  SortingState,
} from "@tanstack/react-table";
import { ArrowUpDown, ChevronLeft, ChevronRight, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";

interface DataTableProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  loading?: boolean;
  onPaginationChange?: (cursor: string) => void;
  nextCursor?: string;
  hasMore?: boolean;
}

export function DataTable<TData, TValue>({
  columns,
  data,
  loading = false,
  onPaginationChange,
  nextCursor,
  hasMore = false,
}: DataTableProps<TData, TValue>) {
  const [sorting, setSorting] = React.useState<SortingState>([]);
  
  const table = useReactTable({
    data,
    columns,
    state: {
      sorting,
    },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
  });

  return (
    <div className="space-y-4">
      <div className="rounded-xl border border-gray-900 bg-gray-950/40 overflow-hidden">
        <table className="w-full text-left border-collapse">
          <thead>
            {table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id} className="border-b border-gray-900 bg-gray-950/80">
                {headerGroup.headers.map((header) => (
                  <th
                    key={header.id}
                    className="px-4 py-3 text-xs font-bold text-gray-500 uppercase tracking-wider select-none font-sans"
                  >
                    {header.isPlaceholder ? null : (
                      <div
                        className={cn(
                          header.column.getCanSort() &&
                            "flex items-center space-x-1 cursor-pointer hover:text-gray-300"
                        )}
                        onClick={header.column.getToggleSortingHandler()}
                      >
                        <span>{flexRender(header.column.columnDef.header, header.getContext())}</span>
                        {header.column.getCanSort() && <ArrowUpDown className="h-3 w-3" />}
                      </div>
                    )}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          
          <tbody className="divide-y divide-gray-950">
            {loading ? (
              <tr>
                <td colSpan={columns.length} className="h-32 text-center">
                  <div className="flex items-center justify-center space-x-2 text-gray-500">
                    <Loader2 className="h-5 w-5 animate-spin text-cyan-500" />
                    <span className="text-sm font-medium font-mono">Loading data feed...</span>
                  </div>
                </td>
              </tr>
            ) : data.length === 0 ? (
              <tr>
                <td colSpan={columns.length} className="h-24 text-center text-sm text-gray-600 font-mono">
                  No records found.
                </td>
              </tr>
            ) : (
              table.getRowModel().rows.map((row) => (
                <tr
                  key={row.id}
                  className="hover:bg-gray-900/30 transition-colors duration-150"
                >
                  {row.getVisibleCells().map((cell) => (
                    <td
                      key={cell.id}
                      className="px-4 py-3.5 text-sm font-medium text-gray-300 font-mono"
                    >
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {onPaginationChange && (
        <div className="flex items-center justify-between px-2">
          <div className="text-xs text-gray-500 font-mono">
            Showing {data.length} items
          </div>
          <div className="flex space-x-2">
            <button
              onClick={() => onPaginationChange("")}
              disabled={loading}
              className="p-1.5 rounded-lg border border-gray-800 bg-gray-950 text-gray-400 hover:text-white hover:border-gray-700 disabled:opacity-50 transition"
            >
              <ChevronLeft className="h-4 w-4" />
            </button>
            <button
              onClick={() => nextCursor && onPaginationChange(nextCursor)}
              disabled={loading || !hasMore}
              className="p-1.5 rounded-lg border border-gray-800 bg-gray-950 text-gray-400 hover:text-white hover:border-gray-700 disabled:opacity-50 transition"
            >
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
