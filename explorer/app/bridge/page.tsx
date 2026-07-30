"use client";

import React, { useState } from "react";
import Link from "next/link";
import {
  ArrowLeftRight, Shield, ShieldAlert,
  Activity, Users, Hash, CheckCircle2,
  ArrowUpRight, ArrowDownLeft,
} from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import {
  BridgeSupplyMetricsSchema,
  BridgeTxListSchema,
  RelayerListSchema,
  CircuitBreakerHistorySchema,
  NonceRegistrySchema,
  BridgeVolumeHistorySchema,
} from "@/lib/schemas";
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from "recharts";

export default function BridgePage() {
  const [activeTab, setActiveTab] = useState<"overview" | "txs" | "relayers" | "nonces">("overview");

  const { data: metrics, isLoading: metricsLoading } = useQuery({
    queryKey: ["bridge-supply"],
    queryFn: () => apiClient.get("/api/rest/v1/explorer/bridge/supply", BridgeSupplyMetricsSchema),
    refetchInterval: 5000,
  });

  const { data: txsData, isLoading: txsLoading } = useQuery({
    queryKey: ["bridge-txs"],
    queryFn: () => apiClient.get("/api/rest/v1/explorer/bridge/txs", BridgeTxListSchema),
    refetchInterval: 5000,
  });

  const { data: relayersData } = useQuery({
    queryKey: ["bridge-relayers"],
    queryFn: () => apiClient.get("/api/rest/v1/explorer/bridge/relayers", RelayerListSchema),
    refetchInterval: 10000,
  });

  const { data: cbData } = useQuery({
    queryKey: ["bridge-circuit-breaker"],
    queryFn: () => apiClient.get("/api/rest/v1/explorer/bridge/circuit-breaker", CircuitBreakerHistorySchema),
    refetchInterval: 10000,
  });

  const { data: noncesData } = useQuery({
    queryKey: ["bridge-nonces"],
    queryFn: () => apiClient.get("/api/rest/v1/explorer/bridge/nonces", NonceRegistrySchema),
    refetchInterval: 5000,
  });

  const { data: volumeData } = useQuery({
    queryKey: ["bridge-volume"],
    queryFn: () => apiClient.get("/api/rest/v1/explorer/analytics/bridge-volume", BridgeVolumeHistorySchema),
    refetchInterval: 30000,
  });

  const txs = txsData?.txs || [];
  const relayers = relayersData?.relayers || [];
  const cbEvents = cbData?.events || [];
  const usedNonces = noncesData?.usedNonces || [];
  const inFlightNonces = noncesData?.inFlightNonces || [];
  const chartData = (volumeData?.points || []).map(p => ({
    name: p.time.match(/\d{2}:\d{2}/) ? p.time : new Date(p.time).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", hour12: false }),
    volume: p.volume,
  }));

  const loading = metricsLoading || txsLoading;
  const lastPause = cbEvents.filter(e => e.eventType === "pause").sort((a, b) => Number(b.height) - Number(a.height))[0];
  const lastUnpause = cbEvents.filter(e => e.eventType === "unpause").sort((a, b) => Number(b.height) - Number(a.height))[0];
  const isPaused = lastPause && (!lastUnpause || Number(lastUnpause.height) < Number(lastPause.height));

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white font-medium">Bridge</span>
      </nav>

      <div className="border-b border-gray-800 pb-4 flex flex-col md:flex-row md:justify-between md:items-center gap-4">
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-3">
            <ArrowLeftRight className="text-blue-500 h-8 w-8" />
            Cross-Chain Bridge Dashboard
          </h1>
          <p className="text-gray-400 mt-1">
            Real-time tracking of BSC LockBox deposits and Cosmos MsgBridgeOut withdrawals.
          </p>
        </div>
        <div className="flex gap-2">
          <Link href="/bridge/deposit" className="text-xs px-3 py-2 bg-gray-900 border border-gray-800 text-gray-300 hover:text-white rounded-lg flex items-center gap-1.5 transition">
            <ArrowDownLeft className="h-3.5 w-3.5 text-green-500" /> Deposit Logs
          </Link>
          <Link href="/bridge/withdraw" className="text-xs px-3 py-2 bg-gray-900 border border-gray-800 text-gray-300 hover:text-white rounded-lg flex items-center gap-1.5 transition">
            <ArrowUpRight className="h-3.5 w-3.5 text-indigo-500" /> Withdrawal Logs
          </Link>
          <Link href="/bridge/history" className="text-xs px-3 py-2 bg-gray-900 border border-gray-800 text-gray-300 hover:text-white rounded-lg flex items-center gap-1.5 transition">
            <ShieldAlert className="h-3.5 w-3.5 text-red-500" /> Breaker History
          </Link>
        </div>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 flex items-center space-x-4 shadow-md">
          <Activity className="h-8 w-8 text-blue-500" />
          <div>
            <div className="text-xs text-gray-500 uppercase font-bold">Total Bridge Volume (24h)</div>
            <div className="text-xl font-bold text-white">
              {chartData.length > 0 ? chartData.reduce((s, d) => s + d.volume, 0).toLocaleString() : "..."} WSOV
            </div>
          </div>
        </div>
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 flex items-center space-x-4 shadow-md">
          <Users className="h-8 w-8 text-green-500" />
          <div>
            <div className="text-xs text-gray-500 uppercase font-bold">Active Relayers</div>
            <div className="text-xl font-bold text-white">{relayers.length} Online</div>
          </div>
        </div>
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 flex items-center space-x-4 shadow-md">
          <Shield className="h-8 w-8 text-purple-500" />
          <div>
            <div className="text-xs text-gray-500 uppercase font-bold">Circuit Breaker</div>
            <div className="flex items-center gap-1.5 mt-0.5">
              <span className={`h-2.5 w-2.5 rounded-full ${isPaused ? "bg-red-500" : "bg-green-500"}`}></span>
              <span className="text-sm font-semibold text-white">{isPaused ? "Paused" : "Active"}</span>
            </div>
          </div>
        </div>
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 flex items-center space-x-4 shadow-md">
          <Hash className="h-8 w-8 text-indigo-500" />
          <div>
            <div className="text-xs text-gray-500 uppercase font-bold">Pending Deposits</div>
            <div className="text-xl font-bold text-white">{inFlightNonces.length} in progress</div>
          </div>
        </div>
      </div>

      <div className="flex space-x-2 border-b border-gray-900 pb-px">
        {(["overview", "txs", "relayers", "nonces"] as const).map((tab) => (
          <button key={tab} onClick={() => setActiveTab(tab)}
            className={`px-4 py-2.5 text-sm font-medium border-b-2 capitalize transition ${
              activeTab === tab ? "border-blue-500 text-blue-500" : "border-transparent text-gray-500 hover:text-gray-300"
            }`}>{tab}</button>
        ))}
      </div>

      {loading ? (
        <div className="py-20 text-center text-gray-400">Loading bridge details...</div>
      ) : (
        <div className="space-y-6">
          {activeTab === "overview" && (
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
              <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 shadow-md flex flex-col justify-between">
                <div>
                  <h3 className="text-lg font-bold text-white flex items-center gap-2">
                    <Shield className="text-green-500 h-5 w-5" /> Supply Invariant Gauge
                  </h3>
                  <p className="text-xs text-gray-500 mt-1">Cosmos Minted = BSC Escrowed</p>
                </div>
                <div className="my-6 flex flex-col items-center">
                  <svg className="w-32 h-20" viewBox="0 0 100 60">
                    <path d="M 10 50 A 40 40 0 0 1 90 50" fill="none" stroke="#1f2937" strokeWidth="10" strokeLinecap="round" />
                    <path d="M 10 50 A 40 40 0 0 1 90 50" fill="none" stroke="#22c55e" strokeWidth="10" strokeLinecap="round" strokeDasharray="125" strokeDashoffset="0" />
                  </svg>
                  <div className="text-2xl font-extrabold text-white -mt-4">
                    {metrics ? Number(metrics.cosmosMinted).toLocaleString() : "0"} WSOV
                  </div>
                  <div className="text-[10px] text-green-400 font-bold flex items-center gap-0.5 mt-1">
                    <CheckCircle2 className="h-3.5 w-3.5" /> balanced
                  </div>
                </div>
                <div className="border-t border-gray-900 pt-4 space-y-2 text-sm text-gray-400">
                  <div className="flex justify-between"><span>Cosmos Minted</span><span className="font-mono text-white">{metrics ? Number(metrics.cosmosMinted).toLocaleString() : "0"}</span></div>
                  <div className="flex justify-between"><span>BSC Escrow</span><span className="font-mono text-white">{metrics ? Number(metrics.bscCirculating).toLocaleString() : "0"}</span></div>
                  <div className="flex justify-between border-t border-gray-900 pt-2 text-xs"><span>Ratio</span><span className="font-mono text-green-400">{metrics?.bridgeSupplyGauge || "1.00"}</span></div>
                </div>
              </div>
              <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 shadow-md lg:col-span-2">
                <h3 className="text-lg font-bold text-white mb-4">Bridge Volume (24h)</h3>
                <div className="h-64 w-full">
                  {chartData.length > 0 ? (
                    <ResponsiveContainer width="100%" height="100%">
                      <AreaChart data={chartData}>
                        <defs><linearGradient id="v" x1="0" y1="0" x2="0" y2="1"><stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3}/><stop offset="95%" stopColor="#3b82f6" stopOpacity={0}/></linearGradient></defs>
                        <XAxis dataKey="name" stroke="#6b7280" fontSize={11} /><YAxis stroke="#6b7280" fontSize={11} />
                        <Tooltip contentStyle={{ backgroundColor: "#09090b", border: "1px solid #1f2937" }} />
                        <Area type="monotone" dataKey="volume" stroke="#3b82f6" fillOpacity={1} fill="url(#v)" strokeWidth={2} />
                      </AreaChart>
                    </ResponsiveContainer>
                  ) : <div className="h-full flex items-center justify-center text-gray-500 text-sm">No volume data</div>}
                </div>
              </div>
            </div>
          )}

          {activeTab === "txs" && (
            <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 shadow-md">
              <h3 className="text-lg font-bold text-white mb-4">Bridge Transactions</h3>
              {txs.length === 0 ? <div className="py-12 text-center text-gray-500 text-sm">No transactions found</div> : (
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm text-gray-400">
                    <thead><tr className="border-b border-gray-900 text-gray-500 text-xs font-bold uppercase">
                      <th className="pb-3">Nonce</th><th className="pb-3">Direction</th><th className="pb-3">Amount</th>
                      <th className="pb-3">Status</th><th className="pb-3">Source</th><th className="pb-3">Dest</th><th className="pb-3 text-right">Time</th>
                    </tr></thead>
                    <tbody className="divide-y divide-gray-900">
                      {txs.map((tx, i) => (
                        <tr key={i} className="hover:bg-gray-900/50 transition">
                          <td className="py-4"><Link href={`/bridge/tx/${tx.nonce}`} className="text-blue-500 font-bold hover:underline font-mono">#{tx.nonce}</Link></td>
                          <td className="py-4"><span className={`inline-flex items-center gap-1 px-2.5 py-0.5 rounded text-xs font-medium uppercase ${tx.direction === "deposit" ? "bg-blue-950 text-blue-400 border border-blue-900" : "bg-orange-950 text-orange-400 border border-orange-900"}`}>
                            {tx.direction === "deposit" ? <><ArrowDownLeft className="h-3 w-3" /> Deposit</> : <><ArrowUpRight className="h-3 w-3" /> Withdraw</>}
                          </span></td>
                          <td className="py-4 font-mono font-semibold text-white">{(Number(tx.amount) / 1e6).toLocaleString()} WSOV</td>
                          <td className="py-4"><span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium uppercase ${tx.status === "minted" || tx.status === "released" ? "bg-green-950 text-green-400 border border-green-900" : "bg-yellow-950 text-yellow-400 border border-yellow-900"}`}>{tx.status}</span></td>
                          <td className="py-4 font-mono text-gray-300">{(tx.bscLockHash || "").slice(0, 10)}...</td>
                          <td className="py-4 font-mono text-gray-300">{tx.cosmosMintHash ? `${tx.cosmosMintHash.slice(0, 10)}...` : "—"}</td>
                          <td className="py-4 text-right text-xs">{new Date(tx.time || "").toLocaleTimeString()}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )}

          {activeTab === "relayers" && (
            <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 shadow-md space-y-4">
              <h3 className="text-lg font-bold text-white mb-4">Relayer Promotion Ladder</h3>
              {relayers.length === 0 ? <div className="py-12 text-center text-gray-500 text-sm">No relayers found</div> : (
                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                  {relayers.map((r, i) => (
                    <div key={i} className="bg-gray-900/50 border border-gray-850 rounded-xl p-5 space-y-3">
                      <div className="flex justify-between items-center">
                        <span className="text-xs font-bold text-gray-500 uppercase">Step</span>
                        <span className={`inline-flex items-center px-2.5 py-0.5 rounded text-xs font-semibold uppercase ${r.status === "Primary" ? "bg-green-950 text-green-400 border border-green-900" : r.status === "Secondary" ? "bg-blue-950 text-blue-400 border border-blue-900" : "bg-gray-800 text-gray-400 border border-gray-700"}`}>{r.status}</span>
                      </div>
                      <div className="text-sm font-semibold text-white font-mono truncate">{r.address}</div>
                      <div className="border-t border-gray-800 pt-3 flex justify-between text-xs text-gray-400">
                        <div><div>Last Active</div><div className="text-white font-mono font-semibold mt-0.5">#{r.lastActive}</div></div>
                        <div className="text-right"><div>Missed</div><div className={`font-semibold mt-0.5 ${Number(r.missCount) > 5 ? "text-red-400" : "text-white"}`}>{r.missCount}</div></div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {activeTab === "nonces" && (
            <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 shadow-md space-y-6">
              <div><h3 className="text-lg font-bold text-white">Bitmap Nonce Registry</h3>
                <p className="text-xs text-gray-500 mt-1">Processed and in-flight nonces</p></div>
              <div className="border border-gray-900 p-4 rounded-xl space-y-3">
                <div className="grid grid-cols-8 sm:grid-cols-16 gap-2">
                  {Array.from({ length: 64 }).map((_, i) => {
                    const n = 1000 + i;
                    const used = usedNonces.includes(n);
                    const flight = inFlightNonces.includes(n);
                    return <div key={i} className={`h-8 flex items-center justify-center font-mono text-xs rounded font-bold border ${used ? "bg-green-950/40 border-green-900/60 text-green-400" : flight ? "bg-yellow-950/40 border-yellow-900/60 text-yellow-400 animate-pulse" : "bg-gray-900 border-gray-850 text-gray-600"}`}>{used ? "1" : flight ? "?" : "0"}</div>;
                  })}
                </div>
                <div className="flex gap-4 text-[10px] text-gray-500 pt-2 border-t border-gray-900">
                  <div className="flex items-center gap-1"><span className="w-3 h-3 bg-green-950/40 border border-green-900/60 rounded" /> Processed</div>
                  <div className="flex items-center gap-1"><span className="w-3 h-3 bg-yellow-950/40 border border-yellow-900/60 rounded" /> Confirming</div>
                  <div className="flex items-center gap-1"><span className="w-3 h-3 bg-gray-900 border border-gray-850 rounded" /> Available</div>
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}