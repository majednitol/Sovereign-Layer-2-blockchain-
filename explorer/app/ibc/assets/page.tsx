"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { ArrowLeftRight, Coins, ShieldCheck } from "lucide-react";

interface IbcAsset {
  denom: string;
  originChain: string;
  path: string;
  amount: string;
  traceHash: string;
}

export default function IbcAssetsPage() {
  const [assets, setAssets] = useState<IbcAsset[]>([]);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchAssets = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/ibc/assets`);
        if (resp.ok) {
          const data = await resp.json();
          if (data.assets) {
            setAssets(data.assets.map((a: any) => ({
              denom: a.denom,
              originChain: a.originChain,
              path: a.path,
              amount: a.amount,
              traceHash: a.traceHash,
            })));
          }
        }
      } catch (err) {
        console.warn("Using simulated IBC assets", err);
        setAssets([
          { denom: "ibc/ATOM", originChain: "cosmoshub-4", path: "transfer/channel-0", amount: "50,000 ATOM", traceHash: "27394E9A..." },
          { denom: "ibc/OSMO", originChain: "osmosis-1", path: "transfer/channel-1", amount: "250,000 OSMO", traceHash: "A9E2847B..." },
        ]);
      } finally {
        setLoading(false);
      }
    };
    fetchAssets();
  }, []);

  if (loading) {
    return (
      <div className="p-6 max-w-6xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading IBC assets...</div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/ibc" className="hover:text-white transition">IBC</Link>
        <span>/</span>
        <span className="text-gray-300">Assets</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4">
        <h1 className="text-3xl font-bold tracking-tight text-white flex items-center gap-2">
          <Coins className="w-8 h-8 text-blue-500" />
          Cross-Chain Assets Directory
        </h1>
        <p className="text-gray-400 mt-2">Active denominations, origin hubs, and supply details.</p>
      </div>

      {/* Assets Table */}
      <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm text-gray-400">
            <thead className="bg-gray-950 text-xs text-gray-500 uppercase tracking-wider font-semibold">
              <tr>
                <th className="p-4">Asset Denom</th>
                <th className="p-4">Origin Chain</th>
                <th className="p-4">IBC Path</th>
                <th className="p-4">Trace Hash</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-850">
              {assets.map((asset, i) => (
                <tr key={i} className="hover:bg-gray-850/40 transition">
                  <td className="p-4 font-semibold text-white font-mono text-xs">{asset.denom}</td>
                  <td className="p-4 text-gray-300">{asset.originChain}</td>
                  <td className="p-4 font-mono text-xs text-gray-400">{asset.path}</td>
                  <td className="p-4 font-mono text-xs text-gray-500">{asset.traceHash.slice(0, 16)}...</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
