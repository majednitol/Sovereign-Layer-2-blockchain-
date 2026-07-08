"use client";

import React, { useState } from "react";
import Link from "next/link";
import { Clock } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { SlotEventsSchema } from "@/lib/schemas";
import { DataTable } from "@/components/ui/DataTable";
import { Badge } from "@/components/ui/Badge";
import { ColumnDef } from "@tanstack/react-table";

interface SlotEvent {
  slot: number;
  eventType: string;
  blockHeight: number;
  time: string;
  validatorAddress: string;
}

export default function StakingHistoryPage() {
  const [cursor, setCursor] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["staking-history", cursor],
    queryFn: () =>
      apiClient.get(
        `/api/rest/v1/explorer/staking/slot-events?limit=50`,
        SlotEventsSchema
      ),
  });

  const events = data?.events || [];

  const columns: ColumnDef<any>[] = [
    {
      accessorKey: "slot",
      header: "Validator Slot",
      cell: ({ row }) => (
        <span className="font-mono text-white font-bold">
          Slot #{row.original.slot}
        </span>
      ),
    },
    {
      accessorKey: "blockHeight",
      header: "Block Height",
      cell: ({ row }) => (
        <Link
          href={`/blocks/${row.original.blockHeight}`}
          className="text-cyan-400 hover:text-cyan-300 font-mono text-xs font-semibold"
        >
          #{row.original.blockHeight.toLocaleString()}
        </Link>
      ),
    },
    {
      accessorKey: "eventType",
      header: "Event Action",
      cell: ({ row }) => {
        const type = row.original.eventType.toLowerCase();
        const variant = type === "filled" ? "success" : type === "slashed" ? "danger" : "neutral";
        return (
          <Badge variant={variant} size="sm">
            {row.original.eventType.toUpperCase()}
          </Badge>
        );
      },
    },
    {
      accessorKey: "validatorAddress",
      header: "Validator Node",
      cell: ({ row }) => (
        <Link
          href={`/validators/${row.original.validatorAddress}`}
          className="font-mono text-xs text-cyan-500 hover:text-cyan-400"
        >
          {row.original.validatorAddress ? `${row.original.validatorAddress.slice(0, 14)}...${row.original.validatorAddress.slice(-8)}` : "—"}
        </Link>
      ),
    },
    {
      accessorKey: "time",
      header: "Timestamp",
      cell: ({ row }) => (
        <span className="font-mono text-xs text-gray-500">
          {new Date(row.original.time).toLocaleString()}
        </span>
      ),
    },
  ];

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-500 flex items-center space-x-2 font-mono">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/staking" className="hover:text-white transition">Staking</Link>
        <span>/</span>
        <span className="text-gray-300">History</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-900 pb-4">
        <div className="flex items-center space-x-3">
          <Clock className="text-cyan-400 h-8 w-8" />
          <div>
            <h1 className="text-3xl font-extrabold tracking-tight text-white">Staking History</h1>
            <p className="text-gray-500 mt-1 text-xs uppercase font-mono">Sovereign L1 Staking and Slot History logs</p>
          </div>
        </div>
      </div>

      <DataTable
        columns={columns}
        data={events}
        loading={isLoading}
      />
    </div>
  );
}
