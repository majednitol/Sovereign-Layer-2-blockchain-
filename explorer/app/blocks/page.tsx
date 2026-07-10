"use client";

import React, { useState } from "react";
import Link from "next/link";
import { Layers } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { BlockListSchema, StatsSummarySchema } from "@/lib/schemas";
import { DataTable } from "@/components/ui/DataTable";
import { ColumnDef } from "@tanstack/react-table";

interface Block {
  height: number;
  time: string;
  proposer: string;
  txCount: number;
  gasUsed: number;
  gasLimit: number;
  appHash: string;
}

export default function BlocksPage() {
  const [cursor, setCursor] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["blocks", cursor],
    queryFn: () =>
      apiClient.get(
        `/api/rest/v1/explorer/blocks?pagination.limit=12&pagination.cursor=${cursor}`,
        BlockListSchema
      ),
  });

  const blocks = data?.blocks || [];
  const nextCursor = data?.pagination?.nextCursor || "";
  const hasMore = data?.pagination?.hasMore || false;

  const { data: summaryData } = useQuery({
    queryKey: ["stats-summary-blocks"],
    queryFn: () => apiClient.get("/api/rest/v1/explorer/stats/summary", StatsSummarySchema),
  });

  const columns: ColumnDef<any>[] = [
    {
      accessorKey: "height",
      header: "Height",
      cell: ({ row }) => (
        <Link
          href={`/blocks/${row.original.height}`}
          className="text-cyan-400 hover:text-cyan-300 font-bold font-mono"
        >
          #{row.original.height.toLocaleString()}
        </Link>
      ),
    },
    {
      accessorKey: "time",
      header: "Timestamp",
      cell: ({ row }) => (
        <span className="text-gray-300 font-mono text-xs">
          {new Date(row.original.time).toLocaleString()}
        </span>
      ),
    },
    {
      accessorKey: "proposer",
      header: "Proposer Address",
      cell: ({ row }) => (
        <Link
          href={`/validators/${row.original.proposer}`}
          className="text-cyan-500 hover:text-cyan-400 font-mono text-xs"
        >
          {row.original.proposer.slice(0, 14)}...{row.original.proposer.slice(-8)}
        </Link>
      ),
    },
    {
      accessorKey: "txCount",
      header: "Transactions",
      cell: ({ row }) => (
        <span className="text-white font-semibold font-mono">
          {row.original.txCount}
        </span>
      ),
    },
    {
      accessorKey: "gasUsed",
      header: "Gas Used %",
      cell: ({ row }) => {
        const used = row.original.gasUsed || 0;
        const limit = row.original.gasLimit || 1;
        const pct = Math.min((used / limit) * 100, 100);
        return (
          <div className="w-full max-w-[140px] space-y-1 font-mono text-[10px]">
            <div className="flex justify-between text-gray-500">
              <span>{used.toLocaleString()}</span>
              <span>{pct.toFixed(1)}%</span>
            </div>
            <div className="w-full bg-gray-900 rounded-full h-1">
              <div
                className="bg-cyan-500 h-1 rounded-full"
                style={{ width: `${pct}%` }}
              />
            </div>
          </div>
        );
      },
    },
    {
      header: "Base Fee",
      cell: () => (
        <span className="text-gray-400 font-mono text-xs">
          0.025 ucsov
        </span>
      ),
    },
    {
      header: "Burnt Fees",
      cell: ({ row }) => {
        const burnt = (row.original.gasUsed || 0) * 0.025;
        return (
          <span className="text-orange-400 font-mono text-xs font-semibold">
            {burnt > 0 ? `${burnt.toFixed(3)} ucsov` : "0 ucsov"}
          </span>
        );
      },
    },
  ];

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-500 flex items-center space-x-2 font-mono">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-gray-300">Blocks</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-900 pb-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <Layers className="text-cyan-400 h-8 w-8" />
            <div>
              <h1 className="text-3xl font-extrabold tracking-tight text-white">Blocks</h1>
              <p className="text-gray-500 mt-1 text-xs uppercase font-mono">Sovereign L1 Block Ledger</p>
            </div>
          </div>
        </div>
      </div>

      {/* Network Utilization Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="bg-gray-950 border border-gray-900 p-4 rounded-xl space-y-1.5">
          <span className="text-[10px] text-gray-500 font-bold uppercase tracking-wider font-sans">Total Blocks</span>
          <div className="text-xl font-bold text-white font-mono">{summaryData?.latestHeight?.toLocaleString() || "..."}</div>
        </div>
        <div className="bg-gray-950 border border-gray-900 p-4 rounded-xl space-y-1.5">
          <span className="text-[10px] text-gray-500 font-bold uppercase tracking-wider font-sans">Avg Block Time</span>
          <div className="text-xl font-bold text-white font-mono">{summaryData ? `${summaryData.avgBlockTimeSec.toFixed(2)}s` : "..."}</div>
        </div>
        <div className="bg-gray-950 border border-gray-900 p-4 rounded-xl space-y-1.5">
          <span className="text-[10px] text-gray-500 font-bold uppercase tracking-wider font-sans">Active Validators</span>
          <div className="text-xl font-bold text-white font-mono">{summaryData ? `${summaryData.activeValidators} / ${summaryData.totalValidators}` : "..."}</div>
        </div>
        <div className="bg-gray-950 border border-gray-900 p-4 rounded-xl space-y-1.5">
          <span className="text-[10px] text-gray-500 font-bold uppercase tracking-wider font-sans">Live TPS</span>
          <div className="text-xl font-bold text-white font-mono">{summaryData?.liveTps?.toFixed(2) || "..."}</div>
        </div>
      </div>

      <DataTable
        columns={columns}
        data={blocks}
        loading={isLoading}
        onPaginationChange={setCursor}
        nextCursor={nextCursor}
        hasMore={hasMore}
      />
    </div>
  );
}
