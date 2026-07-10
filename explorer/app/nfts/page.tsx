"use client";

import React, { useState } from "react";
import Link from "next/link";
import { ArrowLeft, Image as ImageIcon, Flame, ShoppingBag, ArrowUpRight } from "lucide-react";

interface NFTCollection {
  id: string;
  name: string;
  symbol: string;
  floorPrice: string;
  volume24h: string;
  totalMints: number;
}

export default function NFTsPage() {
  const [collections] = useState<NFTCollection[]>([
    { id: "0x1a2b3c4d5e6f7g8h9i0j", name: "Sovereign Punks", symbol: "SPUNK", floorPrice: "1.25 ESOV", volume24h: "450.50 ESOV", totalMints: 10000 },
    { id: "0x9a8b7c6d5e4f3g2h1i0j", name: "Cosmic Realms", symbol: "REALM", floorPrice: "0.85 ESOV", volume24h: "210.00 ESOV", totalMints: 5000 },
    { id: "0x5f3e2d1c0b9a8a7b6c5d", name: "Wasm Wizards", symbol: "WIZ", floorPrice: "2.40 ESOV", volume24h: "185.20 ESOV", totalMints: 3333 }
  ]);

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white font-medium">NFTs</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-900 pb-4 flex items-center space-x-3">
        <Link href="/" className="p-2 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-gray-400 hover:text-white transition">
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
            <ImageIcon className="text-pink-500 w-8 h-8 animate-pulse" />
            NFT Leaderboard Catalog
          </h1>
          <p className="text-gray-400 mt-1">Registry of ERC-721 and ERC-1155 NFT collections on Sovereign L1.</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Leaderboard collections */}
        <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-xl lg:col-span-2 space-y-4">
          <h3 className="text-lg font-bold text-white flex items-center gap-2 border-b border-gray-900 pb-3">
            <Flame className="text-orange-500 h-5 w-5" />
            Top NFT Collections (24H Volume)
          </h3>

          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm text-gray-400">
              <thead className="bg-black/50 text-xs text-gray-500 uppercase tracking-wider font-bold">
                <tr>
                  <th className="p-3">Rank</th>
                  <th className="p-3">Collection</th>
                  <th className="p-3">Floor Price</th>
                  <th className="p-3">24H Volume</th>
                  <th className="p-3 text-right">Total Mints</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-900">
                {collections.map((c, index) => (
                  <tr key={c.id} className="hover:bg-gray-900/30 transition">
                    <td className="p-3 font-bold text-gray-500">#{index + 1}</td>
                    <td className="p-3">
                      <div className="font-semibold text-white">{c.name}</div>
                      <div className="text-[10px] text-gray-500 font-mono truncate max-w-[150px]">{c.id}</div>
                    </td>
                    <td className="p-3 font-mono font-bold text-white">{c.floorPrice}</td>
                    <td className="p-3 font-mono text-green-400">{c.volume24h}</td>
                    <td className="p-3 text-right font-mono">{c.totalMints.toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Latest trades */}
        <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-xl space-y-4">
          <h3 className="text-lg font-bold text-white flex items-center gap-2 border-b border-gray-900 pb-3">
            <ShoppingBag className="text-pink-500 h-5 w-5" />
            Recent Trades activity
          </h3>

          <div className="space-y-4">
            {[
              { token: "SPUNK #482", price: "1.30 ESOV", time: "2 mins ago" },
              { token: "REALM #1024", price: "0.90 ESOV", time: "12 mins ago" },
              { token: "WIZ #42", price: "2.55 ESOV", time: "25 mins ago" }
            ].map((t, idx) => (
              <div key={idx} className="flex justify-between items-center bg-gray-900/40 p-3 rounded-xl border border-gray-850">
                <div>
                  <div className="text-xs font-bold text-white">{t.token}</div>
                  <div className="text-[10px] text-gray-500">{t.time}</div>
                </div>
                <div className="text-right">
                  <div className="text-xs font-mono font-bold text-green-400">{t.price}</div>
                  <span className="text-[9px] text-gray-500 flex items-center gap-0.5 justify-end">
                    Tx <ArrowUpRight className="h-3 w-3" />
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
