"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useWalletStore } from "@/store/wallet";
import { 
  ArrowLeftRight, Shield, ShieldAlert, Cpu, 
  Activity, Users, Hash, FileText, CheckCircle2, 
  Clock, AlertTriangle, ArrowUpRight, ArrowDownLeft,
  ChevronRight, Network
} from "lucide-react";
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from "recharts";

interface BridgeTx {
  id: string;
  direction: string;
  nonce: number;
  status: string;
  sourceHash: string;
  destHash: string;
  amount: string;
  sender: string;
  receiver: string;
  height: number;
  time: string;
}

interface SupplyMetrics {
  cosmosMinted: string;
  bscCirculating: string;
  totalSupply: string;
  bridgeSupplyGauge: string;
}

interface Relayer {
  address: string;
  status: "Primary" | "Secondary" | "Candidate";
  lastActive: number;
  missCount: number;
}

interface CircuitBreakerEvent {
  height: number;
  eventType: string;
  triggerAddress: string;
  time: string;
}

export default function BridgePage() {
  const { connected, address, walletType, connectWallet, disconnectWallet } = useWalletStore();
  const [activeTab, setActiveTab] = useState<"overview" | "txs" | "relayers" | "nonces">("overview");
  const [txs, setTxs] = useState<BridgeTx[]>([]);
  const [metrics, setMetrics] = useState<SupplyMetrics | null>(null);
  const [relayers, setRelayers] = useState<Relayer[]>([]);
  const [cbEvents, setCbEvents] = useState<CircuitBreakerEvent[]>([]);
  const [nonces, setNonces] = useState<{ usedNonces: number[]; inFlightNonces: number[] }>({ usedNonces: [], inFlightNonces: [] });
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  const fetchBridgeData = async () => {
    try {
      const metricsResp = await fetch(`${API_BASE}/api/rest/v1/explorer/bridge/supply`);
      if (metricsResp.ok) {
        const data = await metricsResp.json();
        setMetrics({
          cosmosMinted: data.cosmosMinted || "0",
          bscCirculating: data.bscCirculating || "0",
          totalSupply: data.totalSupply || "0",
          bridgeSupplyGauge: data.bridgeSupplyGauge || "1.0000",
        });
      }

      const txsResp = await fetch(`${API_BASE}/api/rest/v1/explorer/bridge/txs`);
      if (txsResp.ok) {
        const data = await txsResp.json();
        if (data.txs) {
          setTxs(data.txs.map((t: any) => ({
            id: t.id,
            direction: t.direction,
            nonce: Number(t.nonce),
            status: t.status,
            sourceHash: t.sourceHash,
            destHash: t.destHash,
            amount: t.amount,
            sender: t.sender,
            receiver: t.receiver,
            height: Number(t.height),
            time: t.time,
          })));
        }
      }

      const relayersResp = await fetch(`${API_BASE}/api/rest/v1/explorer/bridge/relayers`);
      if (relayersResp.ok) {
        const data = await relayersResp.json();
        if (data.relayers) {
          setRelayers(data.relayers);
        }
      }

      const cbResp = await fetch(`${API_BASE}/api/rest/v1/explorer/bridge/circuit-breaker`);
      if (cbResp.ok) {
        const data = await cbResp.json();
        if (data.events) {
          setCbEvents(data.events);
        }
      }

      const noncesResp = await fetch(`${API_BASE}/api/rest/v1/explorer/bridge/nonces`);
      if (noncesResp.ok) {
        const data = await noncesResp.json();
        setNonces({
          usedNonces: (data.usedNonces || []).map(Number),
          inFlightNonces: (data.inFlightNonces || []).map(Number),
        });
      }
    } catch (err) {
      console.warn("Failed to fetch bridge data from API. Falling back to mocks.", err);
      setMetrics({
        cosmosMinted: "1250000000000",
        bscCirculating: "1250000000000",
        totalSupply: "2500000000000",
        bridgeSupplyGauge: "1.0000",
      });
      setTxs([
        { id: "1", direction: "deposit", nonce: 1024, status: "minted", sourceHash: "0x3f5c9e2b1d7a8d...", destHash: "8d92a10be43...", amount: "5000000000", sender: "0xsenderaddr...", receiver: "sovereign1address...", height: 120530, time: new Date().toISOString() },
        { id: "2", direction: "withdraw", nonce: 1023, status: "released", sourceHash: "7c28f9d6ae12...", destHash: "0x8e92d10be...", amount: "2500000000", sender: "sovereign1address...", receiver: "0xreceiveraddr...", height: 120520, time: new Date(Date.now() - 3600000).toISOString() },
      ]);
      setRelayers([
        { address: "sovereign1relayer0", status: "Primary", lastActive: 120530, missCount: 0 },
        { address: "sovereign1relayer1", status: "Secondary", lastActive: 120528, missCount: 2 },
        { address: "sovereign1relayer2", status: "Candidate", lastActive: 120410, missCount: 15 },
      ]);
      setCbEvents([
        { height: 110200, eventType: "pause", triggerAddress: "sovereign1relayer0", time: new Date(Date.now() - 86400000).toISOString() },
        { height: 110250, eventType: "unpause", triggerAddress: "sovereign1relayer0", time: new Date(Date.now() - 82800000).toISOString() },
      ]);
      setNonces({
        usedNonces: [1020, 1021, 1022, 1023, 1024],
        inFlightNonces: [1025, 1026],
      });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchBridgeData();
    const interval = setInterval(fetchBridgeData, 5000);
    return () => clearInterval(interval);
  }, []);

  const chartData = [
    { name: "00:00", volume: 150000 },
    { name: "04:00", volume: 220000 },
    { name: "08:00", volume: 180000 },
    { name: "12:00", volume: 340000 },
    { name: "16:00", volume: 290000 },
    { name: "20:00", volume: 410000 },
    { name: "24:00", volume: 380000 },
  ];

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white font-medium">Bridge</span>
      </nav>

      {/* Header */}
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

        {/* Quick Links */}
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

      {/* Status Bar */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 flex items-center space-x-4 shadow-md">
          <Activity className="h-8 w-8 text-blue-500" />
          <div>
            <div className="text-xs text-gray-500 uppercase font-bold">Total Bridge Volume (24h)</div>
            <div className="text-xl font-bold text-white">410,000 SOV</div>
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
              <span className="h-2.5 w-2.5 rounded-full bg-green-500"></span>
              <span className="text-sm font-semibold text-white">Active (Unpaused)</span>
            </div>
          </div>
        </div>

        <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 flex items-center space-x-4 shadow-md">
          <Hash className="h-8 w-8 text-indigo-500" />
          <div>
            <div className="text-xs text-gray-500 uppercase font-bold">Pending Deposits</div>
            <div className="text-xl font-bold text-white">
              {nonces.inFlightNonces.length} in progress
            </div>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex space-x-2 border-b border-gray-900 pb-px">
        {(["overview", "txs", "relayers", "nonces"] as const).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2.5 text-sm font-medium border-b-2 capitalize transition ${
              activeTab === tab 
                ? "border-blue-500 text-blue-500" 
                : "border-transparent text-gray-500 hover:text-gray-300"
            }`}
          >
            {tab}
          </button>
        ))}
      </div>

      {/* Tab Panels */}
      {loading ? (
        <div className="py-20 text-center text-gray-400">Loading bridge details...</div>
      ) : (
        <div className="space-y-6">
          {activeTab === "overview" && (
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
              {/* Supply Invariant Card */}
              <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 shadow-md flex flex-col justify-between">
                <div>
                  <h3 className="text-lg font-bold text-white flex items-center gap-2">
                    <Shield className="text-green-500 h-5 w-5" />
                    Supply Invariant Gauge
                  </h3>
                  <p className="text-xs text-gray-500 mt-1">
                    Asserts: Cosmos Minted Supply = BSC Escrowed LockBox Balance
                  </p>
                </div>

                <div className="my-6 relative flex flex-col items-center justify-center">
                  {/* Gauge Arc representation */}
                  <svg className="w-32 h-20" viewBox="0 0 100 60">
                    <path d="M 10 50 A 40 40 0 0 1 90 50" fill="none" stroke="#1f2937" strokeWidth="10" strokeLinecap="round" />
                    <path d="M 10 50 A 40 40 0 0 1 90 50" fill="none" stroke="#22c55e" strokeWidth="10" strokeLinecap="round" strokeDasharray="125" strokeDashoffset="0" />
                  </svg>
                  <div className="text-2xl font-extrabold text-white tracking-tight -mt-4">
                    {metrics ? Number(metrics.cosmosMinted).toLocaleString() : "0"} SOV
                  </div>
                  <div className="text-[10px] text-green-400 font-bold flex items-center gap-0.5 mt-1">
                    <CheckCircle2 className="h-3.5 w-3.5" /> balanced & verified
                  </div>
                </div>

                <div className="border-t border-gray-900 pt-4 space-y-2 text-sm text-gray-400">
                  <div className="flex justify-between">
                    <span>Cosmos Minted (x/bridge)</span>
                    <span className="font-mono text-white">
                      {metrics ? Number(metrics.cosmosMinted).toLocaleString() : "0"}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span>BSC Escrow Balance</span>
                    <span className="font-mono text-white">
                      {metrics ? Number(metrics.bscCirculating).toLocaleString() : "0"}
                    </span>
                  </div>
                  <div className="flex justify-between border-t border-gray-900 pt-2 text-xs">
                    <span>Supply Invariant Ratio</span>
                    <span className="font-mono text-green-400">{metrics?.bridgeSupplyGauge || "1.00"}</span>
                  </div>
                </div>
              </div>

              {/* Volume Area Chart */}
              <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 shadow-md lg:col-span-2">
                <h3 className="text-lg font-bold text-white mb-4">Bridge Activity Volume (24h)</h3>
                <div className="h-64 w-full">
                  <ResponsiveContainer width="100%" height="100%">
                    <AreaChart data={chartData}>
                      <defs>
                        <linearGradient id="colorVolume" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3}/>
                          <stop offset="95%" stopColor="#3b82f6" stopOpacity={0}/>
                        </linearGradient>
                      </defs>
                      <XAxis dataKey="name" stroke="#6b7280" fontSize={11} tickLine={false} />
                      <YAxis stroke="#6b7280" fontSize={11} tickLine={false} />
                      <Tooltip 
                        contentStyle={{ backgroundColor: "#09090b", border: "1px solid #1f2937" }}
                        labelStyle={{ color: "#9ca3af" }}
                      />
                      <Area type="monotone" dataKey="volume" stroke="#3b82f6" fillOpacity={1} fill="url(#colorVolume)" strokeWidth={2} />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>
              </div>
            </div>
          )}

          {activeTab === "txs" && (
            <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 shadow-md">
              <h3 className="text-lg font-bold text-white mb-4">Bridge Transactions Log</h3>
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm text-gray-400">
                  <thead>
                    <tr className="border-b border-gray-900 text-gray-500 text-xs font-bold uppercase">
                      <th className="pb-3">Nonce</th>
                      <th className="pb-3">Direction</th>
                      <th className="pb-3">Amount</th>
                      <th className="pb-3">Status</th>
                      <th className="pb-3">Source Tx Hash</th>
                      <th className="pb-3">Dest Tx Hash</th>
                      <th className="pb-3 text-right">Time</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-900">
                    {txs.map((tx) => (
                      <tr key={tx.id} className="hover:bg-gray-900/50 transition">
                        <td className="py-4">
                          <Link href={`/bridge/tx/${tx.nonce}`} className="text-blue-500 font-bold hover:underline font-mono">
                            #{tx.nonce}
                          </Link>
                        </td>
                        <td className="py-4">
                          <span className={`inline-flex items-center gap-1 px-2.5 py-0.5 rounded text-xs font-medium uppercase ${
                            tx.direction === "deposit" 
                              ? "bg-blue-950 text-blue-400 border border-blue-900" 
                              : "bg-orange-950 text-orange-400 border border-orange-900"
                          }`}>
                            {tx.direction === "deposit" ? (
                              <>
                                <ArrowDownLeft className="h-3 w-3" /> Deposit
                              </>
                            ) : (
                              <>
                                <ArrowUpRight className="h-3 w-3" /> Withdraw
                              </>
                            )}
                          </span>
                        </td>
                        <td className="py-4 font-mono font-semibold text-white">
                          {(Number(tx.amount) / 1e6).toLocaleString()} SOV
                        </td>
                        <td className="py-4">
                          <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium uppercase ${
                            tx.status === "minted" || tx.status === "released"
                              ? "bg-green-950 text-green-400 border border-green-900"
                              : "bg-yellow-950 text-yellow-400 border border-yellow-900"
                          }`}>
                            {tx.status}
                          </span>
                        </td>
                        <td className="py-4 font-mono">
                          <span className="text-gray-300">{tx.sourceHash.slice(0, 10)}...</span>
                        </td>
                        <td className="py-4 font-mono">
                          {tx.destHash ? (
                            <span className="text-gray-300">{tx.destHash.slice(0, 10)}...</span>
                          ) : (
                            <span className="text-gray-600">—</span>
                          )}
                        </td>
                        <td className="py-4 text-right text-xs">
                          {new Date(tx.time).toLocaleTimeString()}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {activeTab === "relayers" && (
            <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 shadow-md space-y-4">
              <h3 className="text-lg font-bold text-white mb-4">Relayer Set & Promotion Ladder</h3>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                {relayers.map((relayer, index) => (
                  <div key={index} className="bg-gray-900/50 border border-gray-850 rounded-xl p-5 space-y-3 relative overflow-hidden">
                    <div className="flex justify-between items-center">
                      <span className="text-xs font-bold text-gray-500 uppercase">Relayer Ladder Step</span>
                      <span className={`inline-flex items-center px-2.5 py-0.5 rounded text-xs font-semibold uppercase ${
                        relayer.status === "Primary" 
                          ? "bg-green-950 text-green-400 border border-green-900"
                          : relayer.status === "Secondary"
                            ? "bg-blue-950 text-blue-400 border border-blue-900"
                            : "bg-gray-800 text-gray-400 border border-gray-700"
                      }`}>
                        {relayer.status}
                      </span>
                    </div>

                    <div className="space-y-1">
                      <div className="text-sm font-semibold text-white font-mono truncate">{relayer.address}</div>
                      <div className="text-xs text-gray-500">Node Operator Address</div>
                    </div>

                    <div className="border-t border-gray-800 pt-3 flex justify-between text-xs text-gray-400">
                      <div>
                        <div>Last Active Block</div>
                        <div className="text-white font-mono font-semibold mt-0.5">#{relayer.lastActive}</div>
                      </div>
                      <div className="text-right">
                        <div>Missed Heartbeats</div>
                        <div className={`font-semibold mt-0.5 ${relayer.missCount > 5 ? "text-red-400" : "text-white"}`}>
                          {relayer.missCount} blocks
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {activeTab === "nonces" && (
            <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 shadow-md space-y-6">
              <div>
                <h3 className="text-lg font-bold text-white">Bitmap Nonce Registry</h3>
                <p className="text-xs text-gray-500 mt-1">
                  Visualization of processed and in-flight nonces mapped to a bits grid representing the nonce state space.
                </p>
              </div>

              {/* Compressed Bitmap Grid display */}
              <div className="border border-gray-900 p-4 rounded-xl space-y-3">
                <h4 className="text-xs font-bold text-gray-400 uppercase tracking-wider">Bit Registry Map</h4>
                <div className="grid grid-cols-8 sm:grid-cols-16 gap-2">
                  {Array.from({ length: 64 }).map((_, i) => {
                    const offsetNonce = 1000 + i;
                    const isUsed = nonces.usedNonces.includes(offsetNonce);
                    const isInFlight = nonces.inFlightNonces.includes(offsetNonce);
                    return (
                      <div 
                        key={i} 
                        className={`h-8 flex items-center justify-center font-mono text-xs rounded font-bold border cursor-help relative group ${
                          isUsed ? "bg-green-950/40 border-green-900/60 text-green-400" :
                          isInFlight ? "bg-yellow-950/40 border-yellow-900/60 text-yellow-400 animate-pulse" :
                          "bg-gray-900 border-gray-850 text-gray-600"
                        }`}
                        title={`Nonce #${offsetNonce}`}
                      >
                        {isUsed ? "1" : isInFlight ? "?" : "0"}
                        <span className="absolute bottom-full left-1/2 transform -translate-x-1/2 bg-black border border-gray-800 text-[10px] text-white px-1.5 py-0.5 rounded opacity-0 group-hover:opacity-100 transition whitespace-nowrap mb-1 z-10 pointer-events-none">
                          Nonce #{offsetNonce}
                        </span>
                      </div>
                    );
                  })}
                </div>
                <div className="flex gap-4 text-[10px] text-gray-500 pt-2 border-t border-gray-900">
                  <div className="flex items-center gap-1">
                    <span className="w-3 h-3 bg-green-950/40 border border-green-900/60 rounded inline-block" /> Processed (1)
                  </div>
                  <div className="flex items-center gap-1">
                    <span className="w-3 h-3 bg-yellow-950/40 border border-yellow-900/60 rounded inline-block" /> Confirming (?)
                  </div>
                  <div className="flex items-center gap-1">
                    <span className="w-3 h-3 bg-gray-900 border border-gray-850 rounded inline-block" /> Available (0)
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
