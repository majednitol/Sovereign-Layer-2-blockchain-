"use client";

import React from "react";
import Link from "next/link";
import { Server, Activity, ShieldCheck } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { ValidatorListSchema } from "@/lib/schemas";

export default function ValidatorsPage() {
  const { data, isLoading } = useQuery({
    queryKey: ["validators"],
    queryFn: () => apiClient.get("/api/rest/v1/explorer/validators", ValidatorListSchema),
    refetchInterval: 10000,
  });

  const validators = data?.validators || [];

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white">Validators</span>
      </nav>

      <div>
        <h1 className="text-3xl font-bold tracking-tight text-white">Validator Slot Grid</h1>
        <p className="text-gray-400 mt-1">
          Sovereign L1 validator sets. 30 active slots with equal consensus power.
        </p>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 flex items-center space-x-4">
          <Server className="h-8 w-8 text-blue-500" />
          <div>
            <div className="text-xs text-gray-500 uppercase font-bold">Total Active Slots</div>
            <div className="text-2xl font-semibold text-white">{validators.length} / 30</div>
          </div>
        </div>
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 flex items-center space-x-4">
          <ShieldCheck className="h-8 w-8 text-green-500" />
          <div>
            <div className="text-xs text-gray-500 uppercase font-bold">Avg Attestation Score</div>
            <div className="text-2xl font-semibold text-white">
              {validators.length > 0
                ? (validators.reduce((s, v) => s + v.certificationScore, 0) / validators.length).toFixed(1)
                : "..."}%
            </div>
          </div>
        </div>
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 flex items-center space-x-4">
          <Activity className="h-8 w-8 text-indigo-500" />
          <div>
            <div className="text-xs text-gray-500 uppercase font-bold">Consensus Mode</div>
            <div className="text-2xl font-semibold text-white">Equal Power</div>
          </div>
        </div>
      </div>

      {isLoading ? (
        <div className="py-12 text-center text-gray-500">Loading validator grid...</div>
      ) : validators.length === 0 ? (
        <div className="py-12 text-center text-gray-500">No validators found</div>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 lg:grid-cols-6 gap-4">
          {validators.map((val) => (
            <Link
              key={val.slotIndex}
              href={`/validators/${val.address}`}
              className="group block bg-gray-950 border border-gray-900 hover:border-blue-600 rounded-xl p-4 transition-all duration-300 transform hover:-translate-y-1 shadow-md hover:shadow-lg hover:shadow-blue-900/10"
            >
              <div className="flex justify-between items-start mb-2">
                <span className="text-xs font-bold text-gray-500 font-mono group-hover:text-blue-400">
                  SLOT #{val.slotIndex.toString().padStart(2, "0")}
                </span>
                <span className={`h-2 w-2 rounded-full ${val.status === "active" ? "bg-green-500 animate-pulse" : "bg-red-500"}`}></span>
              </div>
              <div className="space-y-1">
                <div className="text-sm font-medium text-white font-mono truncate">
                  {val.address.slice(0, 12)}...
                </div>
                <div className="flex justify-between items-center text-xs mt-3">
                  <span className="text-gray-400">Attest Score</span>
                  <span className="font-semibold text-green-400">{val.certificationScore}%</span>
                </div>
                <div className="flex justify-between items-center text-xs">
                  <span className="text-gray-400">Missed</span>
                  <span className="font-mono text-gray-300">{val.missedBlocks} blocks</span>
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}