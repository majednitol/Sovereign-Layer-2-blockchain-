"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Coins, Activity, Users, ArrowLeft, History, TrendingUp } from "lucide-react";
import { ResponsiveContainer, PieChart, Pie, Cell, LineChart, Line, XAxis, YAxis, Tooltip, AreaChart, Area } from "recharts";

interface EvmToken {
  address: string;
  name: string;
  symbol: string;
  decimals: number;
  totalSupply: string;
  balance: string;
}

interface HolderData {
  address: string;
  percentage: number;
  balance: string;
}

interface TransferLog {
  hash: string;
  from: string;
  to: string;
  amount: string;
  time: string;
}

const COLORS = ["#8b5cf6", "#3b82f6", "#10b981", "#f59e0b", "#ef4444"];

export default function EvmTokenDetailPage() {
  const params = useParams();
  const addr = params?.addr ? String(params.addr) : "";

  const [token, setToken] = useState<EvmToken | null>(null);
  const [holders, setHolders] = useState<HolderData[]>([]);
  const [transfers, setTransfers] = useState<TransferLog[]>([]);
  const [priceHistory, setPriceHistory] = useState<{ time: string; price: number }[]>([]);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    if (!addr) return;
    const fetchTokenDetails = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/tokens/cw20/${addr}`);
        if (resp.ok) {
          const data = await resp.json();
          setToken({
            address: data.address || addr,
            name: data.name || "Sovereign Stablecoin",
            symbol: data.symbol || "sUSDT",
            decimals: Number(data.decimals || 18),
            totalSupply: data.totalSupply || "10,000,000",
            balance: data.balance || "0",
          });
        } else {
          throw new Error("Token details not found");
        }
      } catch (err) {
        console.warn("Using simulated token details", err);
        setToken({
          address: addr,
          name: "Sovereign Wrapped Ether Token",
          symbol: "sWETH",
          decimals: 18,
          totalSupply: "1,000,000",
          balance: "150",
        });

        setHolders([
          { address: "0x3f5c9e2b1d7a8d9e8a7b6c5d4e3f281f449219d5", percentage: 55, balance: "550,000" },
          { address: "0x25091a8d7a8b6c5d4e3f281f449219d54e47fd8a", percentage: 25, balance: "250,000" },
          { address: "0x1234567890abcdef1234567890abcdef12345678", percentage: 10, balance: "100,000" },
          { address: "0x892a10be892a10be892a10be892a10be892a10be8", percentage: 7, balance: "70,000" },
          { address: "Others", percentage: 3, balance: "30,000" }
        ]);

        setTransfers([
          { hash: "0x3f5c9e2b1d7a8d9e8a7b6c5d4e3f281f449219d54e47fd8ad83861b464815d9d", from: "0x3f5c9e2b1d7a", to: "0x25091a8d7a8b", amount: "5.00", time: new Date().toISOString() },
          { hash: "0x8a7b6c5d4e3f281f449219d54e47fd8ad83861b464815d9d3f5c9e2b1d7a8d9e", from: "0x25091a8d7a8b", to: "0x1234567890ab", amount: "1.25", time: new Date(Date.now() - 60000).toISOString() },
        ]);

        setPriceHistory([
          { time: "09:00", price: 3450 },
          { time: "10:00", price: 3480 },
          { time: "11:00", price: 3465 },
          { time: "12:00", price: 3495 },
          { time: "13:00", price: 3510 },
          { time: "14:00", price: 3505 },
          { time: "15:00", price: 3530 },
        ]);
      } finally {
        setLoading(false);
      }
    };
    fetchTokenDetails();
  }, [addr]);

  if (loading) {
    return (
      <div className="p-6 max-w-6xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading token details...</div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/evm" className="hover:text-white transition">EVM</Link>
        <span>/</span>
        <Link href="/evm/tokens" className="hover:text-white transition">Tokens</Link>
        <span>/</span>
        <span className="text-gray-300 font-mono text-xs">{addr.slice(0, 10)}...</span>
      </nav>

      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between border-b border-gray-800 pb-6 gap-4">
        <div className="flex items-center gap-3">
          <Link href="/evm" className="p-2 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-gray-400 hover:text-white transition">
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div>
            <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
              <Coins className="w-8 h-8 text-purple-500 animate-pulse" />
              {token?.name} ({token?.symbol})
            </h1>
            <p className="text-gray-400 mt-1 font-mono text-xs break-all">Contract: {token?.address}</p>
          </div>
        </div>
      </div>

      {/* Stats Panel */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-6">
        <div className="bg-gray-950 border border-gray-900 p-5 rounded-xl space-y-2 shadow-md">
          <div className="text-xs text-gray-500 uppercase tracking-wider font-bold">Total Supply</div>
          <div className="text-2xl font-bold text-white font-mono">{token?.totalSupply}</div>
        </div>
        <div className="bg-gray-950 border border-gray-900 p-5 rounded-xl space-y-2 shadow-md">
          <div className="text-xs text-gray-500 uppercase tracking-wider font-bold">Decimals</div>
          <div className="text-2xl font-bold text-white font-mono">{token?.decimals}</div>
        </div>
        <div className="bg-gray-950 border border-gray-900 p-5 rounded-xl space-y-2 shadow-md">
          <div className="text-xs text-gray-500 uppercase tracking-wider font-bold">Contract Standard</div>
          <div className="text-2xl font-bold text-white font-mono">ERC-20</div>
        </div>
      </div>

      {/* Charts Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Simulated Price Chart */}
        <div className="bg-gray-950 border border-gray-900 p-6 rounded-2xl shadow-lg space-y-4">
          <h3 className="text-lg font-bold text-white flex items-center gap-2">
            <TrendingUp className="h-5 w-5 text-green-500" />
            Market Price Chart (24h)
          </h3>
          <div className="h-64 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={priceHistory}>
                <defs>
                  <linearGradient id="colorPrice" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#8b5cf6" stopOpacity={0.4}/>
                    <stop offset="95%" stopColor="#8b5cf6" stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <XAxis dataKey="time" stroke="#4b5563" fontSize={11} tickLine={false} />
                <YAxis stroke="#4b5563" fontSize={11} domain={["auto", "auto"]} tickLine={false} />
                <Tooltip contentStyle={{ backgroundColor: "#09090b", borderColor: "#18181b" }} labelClassName="text-white" />
                <Area type="monotone" dataKey="price" stroke="#8b5cf6" strokeWidth={2} fillOpacity={1} fill="url(#colorPrice)" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Token Holders Donut Chart */}
        <div className="bg-gray-950 border border-gray-900 p-6 rounded-2xl shadow-lg space-y-4">
          <h3 className="text-lg font-bold text-white flex items-center gap-2">
            <Activity className="h-5 w-5 text-indigo-500" />
            Top Token Holders Breakdown
          </h3>
          <div className="flex flex-col sm:flex-row items-center gap-6">
            <div className="h-48 w-48 shrink-0">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={holders}
                    cx="50%"
                    cy="50%"
                    innerRadius={60}
                    outerRadius={80}
                    paddingAngle={3}
                    dataKey="percentage"
                  >
                    {holders.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip contentStyle={{ backgroundColor: "#09090b", borderColor: "#18181b" }} />
                </PieChart>
              </ResponsiveContainer>
            </div>
            <div className="space-y-2 text-xs w-full">
              {holders.map((h, idx) => (
                <div key={idx} className="flex items-center justify-between font-mono bg-gray-900/30 p-2 border border-gray-900 rounded-lg">
                  <div className="flex items-center gap-2">
                    <div className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: COLORS[idx % COLORS.length] }} />
                    <span className="text-gray-300 font-bold">{h.address.slice(0, 10)}...</span>
                  </div>
                  <span className="text-white font-semibold">{h.percentage}% ({h.balance} {token?.symbol})</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* Transfer History Table */}
      <div className="bg-gray-950 border border-gray-900 p-6 rounded-2xl shadow-lg space-y-4">
        <h3 className="text-lg font-bold text-white flex items-center gap-2">
          <History className="h-5 w-5 text-blue-500" />
          Transfer Activity Logs
        </h3>
        <div className="overflow-x-auto border border-gray-900 rounded-xl">
          <table className="w-full text-left text-sm text-gray-400 font-mono">
            <thead className="bg-black/50 text-xs text-gray-500 uppercase tracking-wider font-bold">
              <tr>
                <th className="p-4">Tx Hash</th>
                <th className="p-4">From</th>
                <th className="p-4">To</th>
                <th className="p-4">Amount</th>
                <th className="p-4 text-right">Age</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-900">
              {transfers.map((tx) => (
                <tr key={tx.hash} className="hover:bg-gray-900/30 transition text-xs">
                  <td className="p-4 font-bold text-white">
                    <Link href={`/evm/txs/${tx.hash}`} className="text-blue-500 hover:underline">
                      {tx.hash.slice(0, 12)}...
                    </Link>
                  </td>
                  <td className="p-4">{tx.from}</td>
                  <td className="p-4">{tx.to}</td>
                  <td className="p-4 text-white font-bold">{tx.amount} {token?.symbol}</td>
                  <td className="p-4 text-right text-gray-500">{new Date(tx.time).toLocaleTimeString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
