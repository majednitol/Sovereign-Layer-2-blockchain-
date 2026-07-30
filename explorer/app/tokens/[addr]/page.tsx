"use client";

import React from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Coins } from "lucide-react";

export default function Page() {
  const params = useParams();
  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white">Token Detail</span>
      </nav>
      <div>
        <h1 className="text-3xl font-bold text-white flex items-center gap-3">
          <Coins className="h-8 w-8 text-cyan-400" />
          Token Detail
        </h1>
        <p className="text-gray-400 mt-2">Token information, holders, transfers, and price data.</p>
      </div>
      <div className="bg-gray-950 border border-gray-900 rounded-xl p-6">
        <p className="text-gray-400 text-sm">Viewing: <span className="font-mono text-white">{JSON.stringify(params)}</span></p>
      </div>
    </div>
  );
}
