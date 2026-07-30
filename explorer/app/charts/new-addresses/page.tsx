"use client";

import React from "react";
import Link from "next/link";
import { BarChart3 } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { ChartDataSchema } from "@/lib/schemas";

export default function Page() {
  const { data, isLoading } = useQuery({
    queryKey: ["new addresses Chart"],
    queryFn: () => apiClient.get("/api/rest/v1/explorer/charts/new-addresses", ChartDataSchema),
  });

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white">new addresses Chart</span>
      </nav>
      <div>
        <h1 className="text-3xl font-bold text-white flex items-center gap-3">
          <BarChart3 className="h-8 w-8 text-cyan-400" />
          new addresses Chart
        </h1>
      </div>
      {isLoading ? (
        <div className="py-20 text-center text-gray-400">Loading...</div>
      ) : !data ? (
        <div className="py-20 text-center text-gray-500">No data available</div>
      ) : (
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-6">
          <pre className="text-xs text-gray-300 font-mono overflow-auto">{JSON.stringify(data, null, 2)}</pre>
        </div>
      )}
    </div>
  );
}
