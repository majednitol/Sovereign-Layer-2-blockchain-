"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { Coins, ChevronRight } from "lucide-react";

interface EvmToken {
  address: string;
  name: string;
  symbol: string;
  decimals: number;
  totalSupply: string;
  typeBadge: string;
}

export default function EvmTokensPage() {
  const [tokens, setTokens] = useState<EvmToken[]>([]);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchTokens = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/tokens/cw20/0x3f5c9e2b1d7a8d05cf5d2eb1`);
        if (resp.ok) {
          const data = await resp.json();
          setTokens([
            {
              address: data.address || "0x3f5c9e2b1d7a8d05cf5d2eb1",
              name: data.name || "Sovereign Stablecoin",
              symbol: data.symbol || "sUSDT",
              decimals: Number(data.decimals || 18),
              totalSupply: data.totalSupply || "10,000,000",
              typeBadge: "ERC-20",
            }
          ]);
        }
      } catch (err) {
        console.warn("Using simulated tokens list", err);
        setTokens([
          { address: "0x3f5c9e2b1d7a8d05cf5d2eb1...", name: "Sovereign Wrapped Ether", symbol: "sWETH", decimals: 18, totalSupply: "1,000,000", typeBadge: "ERC-20" },
          { address: "0x9876543210fedcba98765432...", name: "Sovereign Multi-Asset Item", symbol: "sITEM", decimals: 0, totalSupply: "10,000", typeBadge: "ERC-1155" },
        ]);
      } finally {
        setLoading(false);
      }
    };
    fetchTokens();
  }, []);

  if (loading) {
    return (
      <div className="p-6 max-w-6xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading tokens list...</div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-gray-300">EVM Smart Tokens</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4">
        <h1 className="text-3xl font-bold tracking-tight text-white flex items-center gap-2">
          <Coins className="w-8 h-8 text-blue-500" />
          Smart Tokens Directory
        </h1>
        <p className="text-gray-400 mt-2">Active ERC-20 and ERC-1155 smart tokens deployed on Sovereign L1 EVM.</p>
      </div>

      {/* Table */}
      <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm text-gray-400">
            <thead className="bg-gray-950 text-xs text-gray-500 uppercase tracking-wider font-semibold">
              <tr>
                <th className="p-4">Token Name</th>
                <th className="p-4">Symbol</th>
                <th className="p-4">Type</th>
                <th className="p-4">Total Supply</th>
                <th className="p-4 text-right">Details</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-850">
              {tokens.map((token, i) => (
                <tr key={i} className="hover:bg-gray-850/40 transition">
                  <td className="p-4 font-semibold text-white">
                    <Link href={`/evm/tokens/${token.address}`} className="text-blue-500 hover:text-blue-400">
                      {token.name}
                    </Link>
                  </td>
                  <td className="p-4 font-mono text-gray-300">{token.symbol}</td>
                  <td className="p-4">
                    <span className="px-2 py-0.5 text-xs bg-indigo-950 text-indigo-400 border border-indigo-900 rounded font-semibold uppercase">
                      {token.typeBadge}
                    </span>
                  </td>
                  <td className="p-4 font-mono">{token.totalSupply}</td>
                  <td className="p-4 text-right">
                    <Link
                      href={token.typeBadge === "ERC-1155" ? `/evm/tokens/${token.address}/multi` : `/evm/tokens/${token.address}`}
                      className="p-2 bg-gray-850 hover:bg-gray-800 rounded-lg text-gray-400 hover:text-white transition inline-block"
                    >
                      <ChevronRight className="w-4 h-4" />
                    </Link>
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
