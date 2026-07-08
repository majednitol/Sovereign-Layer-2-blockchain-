"use client";

import React, { useState } from "react";
import Link from "next/link";
import { FileText } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { TxListSchema } from "@/lib/schemas";
import { DataTable } from "@/components/ui/DataTable";
import { Badge } from "@/components/ui/Badge";
import { ColumnDef } from "@tanstack/react-table";

interface Tx {
  hash: string;
  height: number;
  time: string;
  type: string;
  msgTypes: string[];
  status: number;
  fee: number;
  gasUsed?: number;
}

export default function TxsPage() {
  const [cursor, setCursor] = useState("");
  const [selectedType, setSelectedType] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["transactions", cursor, selectedType],
    queryFn: () =>
      apiClient.get(
        `/api/rest/v1/explorer/txs?pagination.limit=12&pagination.cursor=${cursor}&type=${selectedType}`,
        TxListSchema
      ),
  });

  const txs = data?.txs || [];
  const nextCursor = data?.pagination?.nextCursor || "";
  const hasMore = data?.pagination?.hasMore || false;

  const columns: ColumnDef<any>[] = [
    {
      accessorKey: "hash",
      header: "Tx Hash",
      cell: ({ row }) => (
        <Link
          href={`/txs/${row.original.hash}`}
          className="text-cyan-400 hover:text-cyan-300 font-mono text-xs"
        >
          {row.original.hash.slice(0, 16)}...
        </Link>
      ),
    },
    {
      accessorKey: "height",
      header: "Height",
      cell: ({ row }) => (
        <Link
          href={`/blocks/${row.original.height}`}
          className="text-gray-300 hover:text-cyan-400 hover:underline"
        >
          #{row.original.height.toLocaleString()}
        </Link>
      ),
    },
    {
      accessorKey: "time",
      header: "Timestamp",
      cell: ({ row }) => new Date(row.original.time).toLocaleString(),
    },
    {
      accessorKey: "type",
      header: "Type",
      cell: ({ row }) => (
        <Badge variant="neutral" size="sm">
          {row.original.type}
        </Badge>
      ),
    },
    {
      accessorKey: "msgTypes",
      header: "Message Type",
      cell: ({ row }) => (
        <span className="text-xs text-gray-500 font-mono">
          {row.original.msgTypes[0] || "Msg"}
        </span>
      ),
    },
    {
      accessorKey: "fee",
      header: "Fee",
      cell: ({ row }) => `${row.original.fee.toLocaleString()} uSLT`,
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) =>
        row.original.status === 0 ? (
          <Badge variant="success" size="sm">Success</Badge>
        ) : (
          <Badge variant="danger" size="sm">Failed</Badge>
        ),
    },
  ];

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-500 flex items-center space-x-2 font-mono">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-gray-300">Transactions</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-900 pb-4">
        <div className="flex items-center space-x-3">
          <FileText className="text-cyan-400 h-8 w-8" />
          <div>
            <h1 className="text-3xl font-extrabold tracking-tight text-white">Transactions</h1>
            <p className="text-gray-500 mt-1 text-xs uppercase font-mono">Sovereign L1 Transaction ledger</p>
          </div>
        </div>
      </div>

      {/* Filter Options */}
      <div className="flex justify-between items-center bg-gray-950 border border-gray-900 p-4 rounded-xl">
        <span className="text-xs text-gray-400 font-mono">Runtime Filter:</span>
        <select
          value={selectedType}
          onChange={(e) => {
            setSelectedType(e.target.value);
            setCursor("");
          }}
          className="bg-gray-900 border border-gray-800 rounded-lg px-3 py-1.5 text-xs text-white outline-none focus:border-cyan-500 font-mono"
        >
          <option value="">All Transactions</option>
          <option value="cosmos">Cosmos SDK</option>
          <option value="evm">EVM</option>
          <option value="cosmwasm">CosmWasm</option>
        </select>
      </div>

      <DataTable
        columns={columns}
        data={txs}
        loading={isLoading}
        onPaginationChange={setCursor}
        nextCursor={nextCursor}
        hasMore={hasMore}
      />
    </div>
  );
}
