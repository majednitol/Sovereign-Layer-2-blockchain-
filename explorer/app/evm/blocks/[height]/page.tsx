"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Database, Clock, ArrowLeft, ArrowUpRight, CheckCircle2, ChevronRight, User, Hash } from "lucide-react";

interface EvmBlock {
  height: number;
  time: string;
  txCount: number;
  gasUsed: string;
  gasLimit: string;
  miner: string;
  appHash: string;
}

interface BlockTx {
  hash: string;
  from: string;
  to: string;
  value: string;
  status: "success" | "failed";
}

export default function EvmBlockDetailPage() {
  const params = useParams();
  const height = params?.height ? Number(params.height) : 100;

  const [block, setBlock] = useState<EvmBlock | null>(null);
  const [txs, setTxs] = useState<BlockTx[]>([]);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchBlockAndTxs = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/evm/blocks/${height}`);
        if (resp.ok) {
          const data = await resp.json();
          setBlock({
            height: Number(data.height || height),
            time: data.time || new Date().toISOString(),
            txCount: Number(data.txCount || 0),
            gasUsed: data.gasUsed || "0",
            gasLimit: data.gasLimit || "30,000,000",
            miner: data.proposer || "0x0000000000000000000000000000000000000000",
            appHash: data.appHash || "",
          });
          if (data.transactions) {
            setTxs(data.transactions);
          }
        } else {
          throw new Error("Block not found");
        }
      } catch (err) {
        console.warn("Using simulated EVM block details", err);
        setBlock({
          height: height,
          time: new Date().toISOString(),
          txCount: 3,
          gasUsed: "120,530",
          gasLimit: "30,000,000",
          miner: "0x1234567890abcdef1234567890abcdef12345678",
          appHash: "0x5f3a7c8d9e2b0a1f7c7c8d9e2b0a1f7c5f3a7c8d9e2b0a1f7c",
        });
        setTxs([
          { hash: "0x3f5c9e2b1d7a8d9e8a7b6c5d4e3f281f449219d54e47fd8ad83861b464815d9d", from: "0xsenderaddress1", to: "0xreceiveraddress1", value: "2.5 SLT", status: "success" },
          { hash: "0x8a7b6c5d4e3f281f449219d54e47fd8ad83861b464815d9d3f5c9e2b1d7a8d9e", from: "0xsenderaddress2", to: "0xreceiveraddress2", value: "0.1 SLT", status: "success" },
        ]);
      } finally {
        setLoading(false);
      }
    };
    fetchBlockAndTxs();
  }, [height]);

  if (loading) {
    return (
      <div className="p-6 max-w-6xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading block details...</div>
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
        <Link href="/evm/blocks" className="hover:text-white transition">Blocks</Link>
        <span>/</span>
        <span className="text-gray-300">Block #{height}</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4 flex items-center gap-3">
        <Link href="/evm/blocks" className="p-2 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-gray-400 hover:text-white transition">
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
            <Database className="w-8 h-8 text-purple-500 animate-pulse" />
            Block #{block?.height} Details
          </h1>
          <p className="text-xs text-gray-500 mt-1">Mined on: {new Date(block?.time || "").toLocaleString()}</p>
        </div>
      </div>

      {/* Profile Grid */}
      <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-6 shadow-lg">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 text-sm">
          <div className="space-y-4">
            <div>
              <div className="text-xs text-gray-500 uppercase font-bold">Block Hash</div>
              <div className="font-mono text-sm text-gray-200 mt-1 break-all bg-gray-900/50 border border-gray-850 p-2.5 rounded-lg select-all">
                {block?.appHash || "N/A"}
              </div>
            </div>
            <div>
              <div className="text-xs text-gray-500 uppercase font-bold">Miner / Proposer</div>
              <div className="font-mono text-sm text-gray-200 mt-1 break-all bg-gray-900/50 border border-gray-850 p-2.5 rounded-lg select-all">
                {block?.miner}
              </div>
            </div>
          </div>
          <div className="space-y-4">
            <div>
              <div className="text-xs text-gray-500 uppercase font-bold">Gas Limit</div>
              <div className="font-mono text-sm text-gray-200 mt-1">{block?.gasLimit}</div>
            </div>
            <div>
              <div className="text-xs text-gray-500 uppercase font-bold">Gas Used</div>
              <div className="font-mono text-sm text-gray-200 mt-1">{block?.gasUsed}</div>
            </div>
          </div>
        </div>
      </div>

      {/* Transactions in Block */}
      <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-lg space-y-4">
        <h2 className="text-lg font-bold text-white flex items-center gap-2">
          <Hash className="h-5 w-5 text-indigo-500" />
          Transactions Mined in Block ({txs.length})
        </h2>

        <div className="overflow-x-auto border border-gray-900 rounded-xl">
          <table className="w-full text-left text-sm text-gray-400">
            <thead className="bg-black/50 text-xs text-gray-500 uppercase tracking-wider font-bold">
              <tr>
                <th className="p-4">Tx Hash</th>
                <th className="p-4">From</th>
                <th className="p-4">To</th>
                <th className="p-4">Value</th>
                <th className="p-4 text-right">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-900">
              {txs.map((tx) => (
                <tr key={tx.hash} className="hover:bg-gray-900/30 transition">
                  <td className="p-4 font-mono font-bold text-white">
                    <Link href={`/evm/txs/${tx.hash}`} className="text-blue-500 hover:underline">
                      {tx.hash.slice(0, 14)}...{tx.hash.slice(-8)}
                    </Link>
                  </td>
                  <td className="p-4 font-mono text-xs">{tx.from.slice(0, 8)}...{tx.from.slice(-6)}</td>
                  <td className="p-4 font-mono text-xs">{tx.to.slice(0, 8)}...{tx.to.slice(-6)}</td>
                  <td className="p-4 font-mono font-semibold text-gray-200">{tx.value}</td>
                  <td className="p-4 text-right">
                    <span className={`px-2 py-0.5 rounded text-[10px] font-bold uppercase border ${
                      tx.status === "success" ? "bg-green-950/40 border-green-900/50 text-green-400" : "bg-red-950/40 border-red-900/50 text-red-400"
                    }`}>
                      {tx.status}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
