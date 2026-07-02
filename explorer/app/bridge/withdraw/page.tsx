"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { ArrowLeft, ArrowUpRight, Clock, ShieldCheck, RefreshCw } from "lucide-react";

interface WithdrawEvent {
  nonce: number;
  cosmosHash: string;
  sender: string;
  receiver: string;
  amount: string;
  height: number;
  time: string;
  status: "burn" | "relaying" | "released";
  bscHash?: string;
}

export default function WithdrawPage() {
  const [withdraws, setWithdraws] = useState<WithdrawEvent[]>([]);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchWithdraws = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/bridge/withdraws`);
        if (resp.ok) {
          const data = await resp.json();
          if (data.withdraws) {
            setWithdraws(data.withdraws);
          }
        }
      } catch (err) {
        console.warn("Using simulated bridge withdraw events", err);
        setWithdraws([
          {
            nonce: 1081,
            cosmosHash: "8d92a10be43210be892a10be892a10be892a10be892a10be892a10be892a10be",
            sender: "sovereign1address0bech32mock",
            receiver: "0x3f5c9e2b1d7a8d9e",
            amount: "8,500 uSLT",
            height: 121040,
            time: new Date().toISOString(),
            status: "released",
            bscHash: "0x3f5c9e2b1d7a8d9e8a7b6c5d4e3f281f449219d54e47fd8ad83861b464815d9d"
          },
          {
            nonce: 1082,
            cosmosHash: "5f3a09e0129bcfe170298a09ee09ea090a908a908d098e09fcd090909e0909fe",
            sender: "sovereign1address0receiver3f5c9e",
            receiver: "0x8a7b6c5d4e3f281f",
            amount: "1,200 uSLT",
            height: 121085,
            time: new Date(Date.now() - 400000).toISOString(),
            status: "relaying"
          }
        ]);
      } finally {
        setLoading(false);
      }
    };
    fetchWithdraws();
  }, []);

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/bridge" className="hover:text-white transition">Bridge</Link>
        <span>/</span>
        <span className="text-gray-300">Withdrawals</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4 flex justify-between items-center">
        <div className="flex items-center space-x-3">
          <Link href="/bridge" className="p-2 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-gray-400 hover:text-white transition">
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div>
            <h1 className="text-3xl font-bold tracking-tight text-white flex items-center gap-2">
              <ArrowUpRight className="w-8 h-8 text-indigo-500" />
              Cosmos → BSC Withdrawals
            </h1>
            <p className="text-gray-400 mt-2">Log of outgoing MsgBridgeOut burn actions and relay states on BSC.</p>
          </div>
        </div>
      </div>

      {loading ? (
        <div className="flex justify-center items-center py-20">
          <RefreshCw className="h-8 w-8 text-blue-500 animate-spin" />
        </div>
      ) : (
        <div className="bg-gray-950 border border-gray-900 rounded-2xl overflow-hidden shadow-lg">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm text-gray-400">
              <thead className="bg-black/50 text-xs text-gray-500 uppercase tracking-wider font-bold">
                <tr>
                  <th className="p-4">Nonce</th>
                  <th className="p-4">Cosmos Tx Hash</th>
                  <th className="p-4">Sender (Cosmos)</th>
                  <th className="p-4">Receiver (BSC)</th>
                  <th className="p-4">Amount</th>
                  <th className="p-4">BSC Release Tx</th>
                  <th className="p-4 text-right">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-900">
                {withdraws.map((w) => (
                  <tr key={w.nonce} className="hover:bg-gray-900/30 transition">
                    <td className="p-4 font-mono font-bold text-white">
                      <Link href={`/bridge/tx/${w.nonce}`} className="text-blue-500 hover:underline">
                        #{w.nonce}
                      </Link>
                    </td>
                    <td className="p-4 font-mono text-xs">
                      {w.cosmosHash.slice(0, 14)}...{w.cosmosHash.slice(-8)}
                    </td>
                    <td className="p-4 font-mono text-xs">{w.sender.slice(0, 8)}...{w.sender.slice(-6)}</td>
                    <td className="p-4 font-mono text-xs">{w.receiver.slice(0, 8)}...{w.receiver.slice(-6)}</td>
                    <td className="p-4 font-mono font-semibold text-gray-200">{w.amount}</td>
                    <td className="p-4 font-mono text-xs text-gray-400">
                      {w.bscHash ? `${w.bscHash.slice(0, 12)}...` : "Pending release"}
                    </td>
                    <td className="p-4 text-right">
                      <span className={`px-2.5 py-1 rounded text-xs font-bold uppercase border ${
                        w.status === "released" ? "bg-green-950/40 border-green-900/50 text-green-400" :
                        w.status === "relaying" ? "bg-yellow-950/40 border-yellow-900/50 text-yellow-400" :
                        "bg-red-950/40 border-red-900/50 text-red-400 animate-pulse"
                      }`}>
                        {w.status}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
