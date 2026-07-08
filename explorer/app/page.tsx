"use client";

import React from "react";
import Link from "next/link";
import { Server, Cpu, Database, Activity, ArrowUpRight, TrendingUp } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { BlockListSchema, TxListSchema, StatusResponseSchema, StatsSummarySchema, TpsHistorySchema } from "@/lib/schemas";
import SearchBar from "@/components/SearchBar";
import { Card } from "@/components/ui/Card";
import { Badge } from "@/components/ui/Badge";
import { ResponsiveContainer, AreaChart, Area, XAxis, YAxis, Tooltip } from "recharts";

export default function HomePage() {
  // Query Stats Summary
  const { data: summaryData } = useQuery({
    queryKey: ["stats-summary"],
    queryFn: () => apiClient.get("/api/rest/v1/explorer/stats/summary", StatsSummarySchema),
    refetchInterval: 5000,
  });

  // Query TPS History
  const { data: tpsHistoryData } = useQuery({
    queryKey: ["tps-history"],
    queryFn: () => apiClient.get("/api/rest/v1/explorer/analytics/tps", TpsHistorySchema),
    refetchInterval: 10000,
  });

  // Query Recent Blocks
  const { data: blocksData, isLoading: blocksLoading } = useQuery({
    queryKey: ["recent-blocks"],
    queryFn: () => apiClient.get("/api/rest/v1/explorer/blocks?pagination.limit=6", BlockListSchema),
  });

  // Query Recent Transactions
  const { data: txsData, isLoading: txsLoading } = useQuery({
    queryKey: ["recent-transactions"],
    queryFn: () => apiClient.get("/api/rest/v1/explorer/txs?pagination.limit=6", TxListSchema),
  });

  // Query API health metrics
  const { data: healthData } = useQuery({
    queryKey: ["api-health"],
    queryFn: () => apiClient.get("/api/rest/v1/explorer/status", StatusResponseSchema).catch(() => null),
  });

  const blocks = blocksData?.blocks || [];
  const txs = txsData?.txs || [];

  const tpsHistory = tpsHistoryData?.points?.map(p => ({
    name: new Date(p.time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    tps: p.tps
  })) || [];

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-8 font-sans">
      {/* Search & Intro Hero Banner */}
      <div className="relative bg-gradient-to-tr from-gray-950 via-slate-900 to-indigo-950/80 border border-gray-900 rounded-2xl p-8 md:p-12 overflow-hidden shadow-2xl">
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,_var(--tw-gradient-stops))] from-cyan-950/20 via-transparent to-transparent pointer-events-none" />
        <div className="relative max-w-3xl mx-auto text-center space-y-6">
          <h2 className="text-4xl md:text-5xl font-extrabold tracking-tight text-white font-sans">
            Sovereign L1 Explorer
          </h2>
          <p className="text-gray-400 max-w-xl mx-auto text-sm md:text-base font-medium">
            Search block height, transaction hash, smart contract address, or validator nodes instantly.
          </p>

          <div className="pt-2 max-w-lg mx-auto">
            <SearchBar />
          </div>
        </div>
      </div>

      {/* Network Stats Dashboard Row */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
        <Card className="hover:border-cyan-500/20 transition-all">
          <div className="flex items-center justify-between">
            <div>
              <span className="text-xs text-gray-500 font-bold uppercase tracking-wider">Live TPS</span>
              <div className="text-3xl font-bold text-white mt-1.5 font-mono">
                {summaryData !== undefined ? summaryData.liveTps.toFixed(2) : "..."}
              </div>
            </div>
            <div className="h-10 w-10 rounded-xl bg-cyan-950/40 border border-cyan-900/30 flex items-center justify-center text-cyan-400">
              <Activity className="h-5 w-5" />
            </div>
          </div>
          <div className="flex items-center text-[10px] text-green-400 font-semibold mt-2.5 space-x-1">
            <TrendingUp className="h-3 w-3" />
            <span>Peak 24.5 tps today</span>
          </div>
        </Card>

        <Card>
          <div className="flex items-center justify-between">
            <div>
              <span className="text-xs text-gray-500 font-bold uppercase tracking-wider">Active Validators</span>
              <div className="text-3xl font-bold text-white mt-1.5 font-mono">
                {summaryData ? `${summaryData.activeValidators} / ${summaryData.totalValidators}` : "..."}
              </div>
            </div>
            <div className="h-10 w-10 rounded-xl bg-green-950/40 border border-green-900/30 flex items-center justify-center text-green-400">
              <Server className="h-5 w-5" />
            </div>
          </div>
          <div className="flex items-center text-[10px] text-gray-500 font-medium mt-2.5">
            <span>100% consensus uptime</span>
          </div>
        </Card>

        <Card>
          <div className="flex items-center justify-between">
            <div>
              <span className="text-xs text-gray-500 font-bold uppercase tracking-wider">Avg Block Time</span>
              <div className="text-3xl font-bold text-white mt-1.5 font-mono">
                {summaryData ? `${summaryData.avgBlockTimeSec.toFixed(2)}s` : "..."}
              </div>
            </div>
            <div className="h-10 w-10 rounded-xl bg-purple-950/40 border border-purple-900/30 flex items-center justify-center text-purple-400">
              <Cpu className="h-5 w-5" />
            </div>
          </div>
          <div className="flex items-center text-[10px] text-gray-500 font-medium mt-2.5">
            <span>CometBFT engine consensus</span>
          </div>
        </Card>

        <Card>
          <div className="flex items-center justify-between">
            <div>
              <span className="text-xs text-gray-500 font-bold uppercase tracking-wider">Latest Height</span>
              <div className="text-3xl font-bold text-white mt-1.5 font-mono">
                {blocks.length > 0 ? blocks[0].height.toLocaleString() : "..."}
              </div>
            </div>
            <div className="h-10 w-10 rounded-xl bg-indigo-950/40 border border-indigo-900/30 flex items-center justify-center text-indigo-400">
              <Database className="h-5 w-5" />
            </div>
          </div>
          <div className="flex items-center text-[10px] text-gray-500 font-medium mt-2.5">
            <span>Synchronized {healthData ? `${healthData.indexerLagSeconds}s ago` : "just now"}</span>
          </div>
        </Card>
      </div>

      {/* Recharts Performance Area Chart */}
      <div className="rounded-xl border border-gray-900 bg-gray-950/30 p-6">
        <div className="flex items-center justify-between pb-6">
          <div>
            <h3 className="text-base font-bold text-white uppercase tracking-wider">TPS Metrics History</h3>
            <p className="text-xs text-gray-500 mt-1">Transaction processing speed over the last 24 hours.</p>
          </div>
        </div>
        <div className="h-48 w-full font-mono text-[10px]">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={tpsHistory}>
              <defs>
                <linearGradient id="tpsGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#06b6d4" stopOpacity={0.2} />
                  <stop offset="95%" stopColor="#06b6d4" stopOpacity={0} />
                </linearGradient>
              </defs>
              <XAxis dataKey="name" stroke="#4b5563" />
              <YAxis stroke="#4b5563" />
              <Tooltip
                contentStyle={{ background: "#0a0f1d", border: "1px solid #1f2937", borderRadius: "8px" }}
                labelClassName="text-white font-bold"
              />
              <Area type="monotone" dataKey="tps" stroke="#06b6d4" strokeWidth={2} fillOpacity={1} fill="url(#tpsGrad)" />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Double Column Page Feeds */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Recent Blocks Feed */}
        <div className="rounded-xl border border-gray-900 bg-gray-950/40 p-6 space-y-4">
          <div className="flex justify-between items-center pb-2 border-b border-gray-900">
            <h3 className="text-base font-bold text-white uppercase tracking-wider">Recent Blocks</h3>
            <Link href="/blocks" className="text-xs text-cyan-400 hover:text-cyan-300 font-semibold flex items-center space-x-0.5">
              <span>View All</span>
              <ArrowUpRight className="h-3 w-3" />
            </Link>
          </div>

          <div className="divide-y divide-gray-950">
            {blocksLoading ? (
              <div className="py-8 text-center text-xs text-gray-500 font-mono">Loading blocks data feed...</div>
            ) : blocks.map((block) => (
              <div key={block.height} className="py-4 flex justify-between items-center text-sm font-mono">
                <div>
                  <Link href={`/blocks/${block.height}`} className="text-cyan-400 hover:text-cyan-300 font-bold transition">
                    #{block.height}
                  </Link>
                  <div className="text-[10px] text-gray-500 mt-1 truncate max-w-[220px]">
                    Proposer: {block.proposer}
                  </div>
                </div>
                <div className="text-right">
                  <div className="text-white font-semibold">{block.txCount} Transactions</div>
                  <div className="text-[10px] text-gray-500 mt-1">
                    {new Date(block.time).toLocaleTimeString()}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Recent Transactions Feed */}
        <div className="rounded-xl border border-gray-900 bg-gray-950/40 p-6 space-y-4">
          <div className="flex justify-between items-center pb-2 border-b border-gray-900">
            <h3 className="text-base font-bold text-white uppercase tracking-wider">Recent Transactions</h3>
            <Link href="/txs" className="text-xs text-cyan-400 hover:text-cyan-300 font-semibold flex items-center space-x-0.5">
              <span>View All</span>
              <ArrowUpRight className="h-3 w-3" />
            </Link>
          </div>

          <div className="divide-y divide-gray-950">
            {txsLoading ? (
              <div className="py-8 text-center text-xs text-gray-500 font-mono">Loading transactions data feed...</div>
            ) : txs.map((tx) => (
              <div key={tx.hash} className="py-4 flex justify-between items-center text-sm font-mono">
                <div>
                  <Link href={`/txs/${tx.hash}`} className="text-cyan-400 hover:text-cyan-300 font-medium transition">
                    {tx.hash.slice(0, 16)}...
                  </Link>
                  <div className="flex items-center space-x-2 text-[10px] text-gray-500 mt-1">
                    <Badge variant="neutral" size="sm">
                      {tx.type}
                    </Badge>
                    <span className="truncate max-w-[150px]">{tx.msgTypes[0] || "Msg"}</span>
                  </div>
                </div>
                <div className="text-right">
                  <div className="text-white font-semibold">{tx.fee} uSLT</div>
                  <div className="mt-1">
                    {tx.status === 0 ? (
                      <Badge variant="success" size="sm">Success</Badge>
                    ) : (
                      <Badge variant="danger" size="sm">Failed</Badge>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
