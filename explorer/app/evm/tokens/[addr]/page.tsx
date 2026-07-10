"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Coins, Activity, Users, ArrowLeft, History, Download, Award, ChevronLeft, ChevronRight } from "lucide-react";
import { ResponsiveContainer, PieChart, Pie, Cell, Tooltip } from "recharts";

interface EvmToken {
  address: string;
  name: string;
  symbol: string;
  decimals: number;
  totalSupply: string;
  balance: string;
  minterAddress?: string;
  ownerAddress?: string;
  verified?: boolean;
  typeBadge?: string;
  holderCount?: number;
  transferCount?: number;
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
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Pagination states
  const [tCursor, setTCursor] = useState<string>("");
  const [tHasMore, setTHasMore] = useState<boolean>(false);
  const [tPrevCursors, setTPrevCursors] = useState<string[]>([]);

  const [hCursor, setHCursor] = useState<string>("");
  const [hHasMore, setHHasMore] = useState<boolean>(false);
  const [hPrevCursors, setHPrevCursors] = useState<string[]>([]);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  // Fetch token detail
  useEffect(() => {
    if (!addr) return;
    const fetchTokenDetails = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/tokens/evm/${addr}`);
        if (resp.ok) {
          const data = await resp.json();
          setToken({
            address: data.address,
            name: data.name,
            symbol: data.symbol,
            decimals: Number(data.decimals),
            totalSupply: data.totalSupply,
            balance: "0",
            minterAddress: data.minterAddress,
            ownerAddress: data.ownerAddress,
            verified: data.verified,
            typeBadge: data.typeBadge,
            holderCount: Number(data.holderCount || 0),
            transferCount: Number(data.transferCount || 0)
          });
        } else {
          setError("Token not found");
        }
      } catch (err) {
        console.error("Failed to fetch token details", err);
        setError("Network error");
      }
    };
    fetchTokenDetails();
  }, [addr, API_BASE]);

  // Fetch transfers
  useEffect(() => {
    if (!addr) return;
    const fetchTransfers = async () => {
      try {
        const url = `${API_BASE}/api/rest/v1/explorer/tokens/evm/${addr}/transfers?limit=10${tCursor ? `&cursor=${tCursor}` : ""}`;
        const resp = await fetch(url);
        if (resp.ok) {
          const data = await resp.json();
          setTransfers(data.transfers.map((tx: any) => ({
            hash: tx.txHash,
            from: tx.fromAddress,
            to: tx.toAddress,
            amount: tx.value,
            time: tx.blockTime
          })));
          setTHasMore(data.hasMore);
        }
      } catch (err) {
        console.error("Failed to fetch transfers", err);
      }
    };
    fetchTransfers();
  }, [addr, tCursor, API_BASE]);

  // Fetch holders
  useEffect(() => {
    if (!addr) return;
    const fetchHolders = async () => {
      try {
        const url = `${API_BASE}/api/rest/v1/explorer/tokens/evm/${addr}/holders?limit=5${hCursor ? `&cursor=${hCursor}` : ""}`;
        const resp = await fetch(url);
        if (resp.ok) {
          const data = await resp.json();
          setHolders(data.holders.map((h: any) => ({
            address: h.address,
            percentage: Number(Number(h.share || 0).toFixed(2)),
            balance: h.balance
          })));
          setHHasMore(data.hasMore);
        }
      } catch (err) {
        console.error("Failed to fetch holders", err);
      } finally {
        setLoading(false);
      }
    };
    fetchHolders();
  }, [addr, hCursor, API_BASE]);

  const handleNextTransfers = () => {
    if (transfers.length > 0 && tHasMore) {
      const last = transfers[transfers.length - 1];
      const nextCursorStr = btoa(`${last.hash}`); // simplified cursor representation
      setTPrevCursors([...tPrevCursors, tCursor]);
      setTCursor(nextCursorStr);
    }
  };

  const handlePrevTransfers = () => {
    if (tPrevCursors.length > 0) {
      const prev = tPrevCursors[tPrevCursors.length - 1];
      setTPrevCursors(tPrevCursors.slice(0, -1));
      setTCursor(prev);
    }
  };

  const handleNextHolders = () => {
    if (holders.length > 0 && hHasMore) {
      const last = holders[holders.length - 1];
      const nextCursorStr = btoa(`${last.balance},${last.address}`);
      setHPrevCursors([...hPrevCursors, hCursor]);
      setHCursor(nextCursorStr);
    }
  };

  const handlePrevHolders = () => {
    if (hPrevCursors.length > 0) {
      const prev = hPrevCursors[hPrevCursors.length - 1];
      setHPrevCursors(hPrevCursors.slice(0, -1));
      setHCursor(prev);
    }
  };

  if (loading) {
    return (
      <div className="p-6 max-w-6xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading token details...</div>
      </div>
    );
  }

  if (error || !token) {
    return (
      <div className="p-6 max-w-6xl mx-auto text-center space-y-4 py-32">
        <h2 className="text-2xl font-bold text-white">Token Not Found</h2>
        <p className="text-gray-400">{error || "The requested EVM token does not exist in the registry."}</p>
        <Link href="/evm" className="inline-block px-4 py-2 bg-gray-900 border border-gray-800 rounded-lg text-white hover:bg-gray-800 transition">
          Back to EVM Tokens
        </Link>
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
        <span className="text-gray-300 font-mono text-xs">{addr}</span>
      </nav>

      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between border-b border-gray-800 pb-6 gap-4">
        <div className="flex items-center gap-3">
          <Link href="/evm" className="p-2 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-gray-400 hover:text-white transition">
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div>
            <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
              <Coins className="w-8 h-8 text-purple-500" />
              {token.name} ({token.symbol})
              {token.verified && (
                <span className="inline-flex items-center gap-1 px-2 py-0.5 text-[10px] bg-green-950 border border-green-900 rounded text-green-400 font-semibold uppercase">
                  <Award className="h-3 w-3" /> Verified
                </span>
              )}
            </h1>
            <p className="text-gray-400 mt-1 font-mono text-xs break-all">Contract: {token.address}</p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <a
            href={`${API_BASE}/api/rest/v1/explorer/tokens/evm/${addr}?download=true`}
            target="_blank"
            rel="noreferrer"
            className="flex items-center gap-2 px-3 py-1.5 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-xs text-gray-300 hover:text-white transition"
          >
            <Download className="h-3.5 w-3.5" /> Download JSON
          </a>
        </div>
      </div>

      {/* Stats Panel */}
      <div className="grid grid-cols-1 sm:grid-cols-4 gap-6">
        <div className="bg-gray-950 border border-gray-900 p-5 rounded-xl space-y-1 shadow-md">
          <div className="text-xs text-gray-500 uppercase tracking-wider font-bold">Total Supply</div>
          <div className="text-lg font-bold text-white font-mono">{token.totalSupply}</div>
        </div>
        <div className="bg-gray-950 border border-gray-900 p-5 rounded-xl space-y-1 shadow-md">
          <div className="text-xs text-gray-500 uppercase tracking-wider font-bold">Decimals</div>
          <div className="text-lg font-bold text-white font-mono">{token.decimals}</div>
        </div>
        <div className="bg-gray-950 border border-gray-900 p-5 rounded-xl space-y-1 shadow-md">
          <div className="text-xs text-gray-500 uppercase tracking-wider font-bold">Token Holders</div>
          <div className="text-lg font-bold text-white font-mono">{token.holderCount}</div>
        </div>
        <div className="bg-gray-950 border border-gray-900 p-5 rounded-xl space-y-1 shadow-md">
          <div className="text-xs text-gray-500 uppercase tracking-wider font-bold">Contract Standard</div>
          <div className="text-lg font-bold text-white font-mono">{token.typeBadge || "EVM"}</div>
        </div>
      </div>

      {/* Owner & Minter info */}
      <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 space-y-4 text-xs font-mono">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <span className="text-gray-500 uppercase font-bold block">Owner / Admin</span>
            <span className="text-gray-300 mt-1 block select-all break-all">{token.ownerAddress || "None"}</span>
          </div>
          <div>
            <span className="text-gray-500 uppercase font-bold block">Minter Address</span>
            <span className="text-gray-300 mt-1 block select-all break-all">{token.minterAddress || "None"}</span>
          </div>
        </div>
      </div>

      {/* Chart & Holders */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Token Holders Donut Chart */}
        <div className="bg-gray-950 border border-gray-900 p-6 rounded-2xl shadow-lg space-y-4 flex flex-col justify-between">
          <h3 className="text-lg font-bold text-white flex items-center gap-2">
            <Activity className="h-5 w-5 text-indigo-500" />
            Holders Distribution
          </h3>
          <div className="flex items-center justify-center py-4">
            {holders.length > 0 ? (
              <div className="h-44 w-44 shrink-0">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={holders}
                      cx="50%"
                      cy="50%"
                      innerRadius={50}
                      outerRadius={70}
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
            ) : (
              <span className="text-gray-600">No holder metrics available</span>
            )}
          </div>
        </div>

        {/* Holders List Table */}
        <div className="lg:col-span-2 bg-gray-950 border border-gray-900 p-6 rounded-2xl shadow-lg space-y-4">
          <div className="flex items-center justify-between border-b border-gray-900 pb-3">
            <h3 className="text-lg font-bold text-white flex items-center gap-2">
              <Users className="h-5 w-5 text-purple-500" />
              Token Holders
            </h3>
            <div className="flex items-center space-x-2">
              <button
                onClick={handlePrevHolders}
                disabled={hPrevCursors.length === 0}
                className="p-1.5 bg-gray-900 border border-gray-800 rounded-lg text-gray-400 hover:text-white disabled:opacity-30 disabled:hover:text-gray-400 transition"
              >
                <ChevronLeft className="h-4 w-4" />
              </button>
              <button
                onClick={handleNextHolders}
                disabled={!hHasMore}
                className="p-1.5 bg-gray-900 border border-gray-800 rounded-lg text-gray-400 hover:text-white disabled:opacity-30 disabled:hover:text-gray-400 transition"
              >
                <ChevronRight className="h-4 w-4" />
              </button>
            </div>
          </div>
          <div className="space-y-2 text-xs">
            {holders.length === 0 ? (
              <div className="py-8 text-center text-gray-500 font-mono">No token holders listed</div>
            ) : (
              holders.map((h, idx) => (
                <div key={idx} className="flex items-center justify-between font-mono bg-gray-900/30 p-2 border border-gray-900 rounded-lg">
                  <div className="flex items-center gap-2">
                    <div className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: COLORS[idx % COLORS.length] }} />
                    <Link href={`/address/${h.address}`} className="text-blue-500 hover:underline">
                      {h.address}
                    </Link>
                  </div>
                  <span className="text-white font-semibold">{h.percentage}% ({h.balance} {token.symbol})</span>
                </div>
              ))
            )}
          </div>
        </div>
      </div>

      {/* Transfer History Table */}
      <div className="bg-gray-950 border border-gray-900 p-6 rounded-2xl shadow-lg space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-bold text-white flex items-center gap-2">
            <History className="h-5 w-5 text-blue-500" />
            Transfer Logs
          </h3>
          <div className="flex items-center space-x-2">
            <button
              onClick={handlePrevTransfers}
              disabled={tPrevCursors.length === 0}
              className="p-1.5 bg-gray-900 border border-gray-800 rounded-lg text-gray-400 hover:text-white disabled:opacity-30 disabled:hover:text-gray-400 transition"
            >
              <ChevronLeft className="h-4 w-4" />
            </button>
            <button
              onClick={handleNextTransfers}
              disabled={!tHasMore}
              className="p-1.5 bg-gray-900 border border-gray-800 rounded-lg text-gray-400 hover:text-white disabled:opacity-30 disabled:hover:text-gray-400 transition"
            >
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
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
              {transfers.length === 0 ? (
                <tr>
                  <td colSpan={5} className="p-8 text-center text-gray-500 font-mono">No transfers found</td>
                </tr>
              ) : (
                transfers.map((tx) => (
                  <tr key={tx.hash} className="hover:bg-gray-900/30 transition text-xs">
                    <td className="p-4 font-bold text-white">
                      <Link href={`/evm/txs/${tx.hash}`} className="text-blue-500 hover:underline">
                        {tx.hash.slice(0, 16)}...
                      </Link>
                    </td>
                    <td className="p-4">
                      <Link href={`/address/${tx.from}`} className="text-blue-500 hover:underline">
                        {tx.from.slice(0, 10)}...{tx.from.slice(-6)}
                      </Link>
                    </td>
                    <td className="p-4">
                      <Link href={`/address/${tx.to}`} className="text-blue-500 hover:underline">
                        {tx.to.slice(0, 10)}...{tx.to.slice(-6)}
                      </Link>
                    </td>
                    <td className="p-4 text-white font-bold">{tx.amount} {token.symbol}</td>
                    <td className="p-4 text-right text-gray-500">{new Date(tx.time).toLocaleString()}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
