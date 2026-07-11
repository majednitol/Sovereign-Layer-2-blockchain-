"use client";

import React, { useState, useEffect } from "react";
import Link from "next/link";
import { Server, Cpu, Database, Activity, ArrowRight, CheckCircle2, Coins } from "lucide-react";

interface NodeStatus {
  chainId: string;
  latestHeight: string;
  latestBlockTime: string;
  online: boolean;
}

export default function HomePage() {
  const [status, setStatus] = useState<NodeStatus>({
    chainId: "sovereign-1",
    latestHeight: "...",
    latestBlockTime: "",
    online: false,
  });
  const [loading, setLoading] = useState(true);

  const RPC_URL = process.env.NEXT_PUBLIC_RPC_URL || "http://localhost:26657";

  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const res = await fetch(`${RPC_URL}/status`);
        if (!res.ok) throw new Error("RPC offline");
        const data = await res.json();
        
        const chainId = data.result?.node_info?.network || "sovereign-1";
        const height = data.result?.sync_info?.latest_block_height || "0";
        const blockTime = data.result?.sync_info?.latest_block_time || "";

        setStatus({
          chainId,
          latestHeight: height,
          latestBlockTime: blockTime ? new Date(blockTime).toLocaleTimeString() : "",
          online: true,
        });
      } catch (err) {
        console.error("Failed to fetch node status:", err);
        setStatus(prev => ({ ...prev, online: false }));
      } finally {
        setLoading(false);
      }
    };

    fetchStatus();
    const interval = setInterval(fetchStatus, 3000);
    return () => clearInterval(interval);
  }, [RPC_URL]);

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-8 font-sans">
      {/* Hero Banner */}
      <div className="relative bg-gradient-to-tr from-gray-950 via-slate-900 to-blue-950/80 border border-gray-900 rounded-2xl p-8 md:p-12 overflow-hidden shadow-2xl">
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,_var(--tw-gradient-stops))] from-blue-950/20 via-transparent to-transparent pointer-events-none" />
        <div className="relative max-w-2xl mx-auto text-center space-y-6">
          <Badge text="Devnet Active" />
          <h1 className="text-4xl md:text-5xl font-extrabold tracking-tight text-white">
            Sovereign L1 Dashboard
          </h1>
          <p className="text-gray-400 text-sm md:text-base font-medium max-w-lg mx-auto">
            Interact with the Sovereign L1 Devnet. Request testnet tokens to compile, test, and deploy smart contracts.
          </p>

          <div className="pt-4 flex justify-center">
            <Link 
              href="/faucet" 
              className="px-6 py-3 bg-blue-600 hover:bg-blue-500 text-white rounded-xl font-bold flex items-center space-x-2 transition shadow-lg shadow-blue-900/30"
            >
              <span>Get Devnet Tokens (SOV)</span>
              <ArrowRight className="h-4 w-4" />
            </Link>
          </div>
        </div>
      </div>

      {/* Node Stats Row */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* Status */}
        <div className="bg-gray-950 border border-gray-900 p-6 rounded-xl space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-xs text-gray-500 font-bold uppercase tracking-wider">Node Status</span>
            <Activity className="h-5 w-5 text-blue-500" />
          </div>
          <div className="flex items-center space-x-2 mt-1">
            <span className={`h-2.5 w-2.5 rounded-full ${status.online ? 'bg-green-500 animate-pulse' : 'bg-red-500'}`}></span>
            <span className="text-xl font-bold text-white font-mono">
              {status.online ? "Online" : "Offline"}
            </span>
          </div>
          <p className="text-[10px] text-gray-500 font-medium">
            Directly connected to CometBFT consensus engine.
          </p>
        </div>

        {/* Height */}
        <div className="bg-gray-950 border border-gray-900 p-6 rounded-xl space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-xs text-gray-500 font-bold uppercase tracking-wider">Latest Block</span>
            <Database className="h-5 w-5 text-blue-500" />
          </div>
          <div className="text-2xl font-bold text-white font-mono mt-1">
            {loading ? "..." : `#${status.latestHeight}`}
          </div>
          <p className="text-[10px] text-gray-500 font-medium">
            {status.latestBlockTime ? `Last updated at ${status.latestBlockTime}` : "Connecting to network..."}
          </p>
        </div>

        {/* Chain ID */}
        <div className="bg-gray-950 border border-gray-900 p-6 rounded-xl space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-xs text-gray-500 font-bold uppercase tracking-wider">Chain Identifier</span>
            <Cpu className="h-5 w-5 text-blue-500" />
          </div>
          <div className="text-2xl font-bold text-white font-mono mt-1">
            {status.chainId}
          </div>
          <p className="text-[10px] text-gray-500 font-medium">
            L1 Devnet sovereign network parameter.
          </p>
        </div>
      </div>

      {/* Network parameters panel */}
      <div className="bg-gray-950/40 border border-gray-900 rounded-xl p-8 space-y-6">
        <h3 className="text-lg font-bold text-white tracking-wide uppercase border-b border-gray-900 pb-3 flex items-center space-x-2">
          <Server className="h-5 w-5 text-blue-500" />
          <span>Devnet Chain Specifications</span>
        </h3>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-8 text-sm">
          <div className="space-y-4">
            <div className="flex justify-between border-b border-gray-900/50 pb-2">
              <span className="text-gray-400 font-medium">Native Token Denomination</span>
              <span className="font-mono text-white font-bold">SOV (ucsov)</span>
            </div>
            <div className="flex justify-between border-b border-gray-900/50 pb-2">
              <span className="text-gray-400 font-medium">Genesis Allocation (Cosmos)</span>
              <span className="font-mono text-white">700,000,000 SOV</span>
            </div>
            <div className="flex justify-between border-b border-gray-900/50 pb-2">
              <span className="text-gray-400 font-medium">Initial Escrowed (BSC)</span>
              <span className="font-mono text-white">300,000,000 SOV</span>
            </div>
            <div className="flex justify-between border-b border-gray-900/50 pb-2">
              <span className="text-gray-400 font-medium">Total Supply (S)</span>
              <span className="font-mono text-white font-bold">1,000,000,000 SOV</span>
            </div>
          </div>

          <div className="space-y-4">
            <div className="flex justify-between border-b border-gray-900/50 pb-2">
              <span className="text-gray-400 font-medium">Validator Seats</span>
              <span className="font-mono text-white">60 active slots</span>
            </div>
            <div className="flex justify-between border-b border-gray-900/50 pb-2">
              <span className="text-gray-400 font-medium">Block Rewards Allocation</span>
              <span className="font-mono text-white">Equal split among validator set</span>
            </div>
            <div className="flex justify-between border-b border-gray-900/50 pb-2">
              <span className="text-gray-400 font-medium">EIP-1559 Fee Market</span>
              <span className="font-mono text-white text-green-400 font-bold">Enabled (1 Gwei Base Fee)</span>
            </div>
            <div className="flex justify-between border-b border-gray-900/50 pb-2">
              <span className="text-gray-400 font-medium">Wasm Constitution</span>
              <span className="font-mono text-white text-green-400 font-bold">Authorized</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function Badge({ text }: { text: string }) {
  return (
    <span className="inline-flex items-center px-3 py-1 bg-blue-950/60 text-blue-400 border border-blue-900/40 rounded-full text-xs font-semibold uppercase tracking-wider">
      <CheckCircle2 className="h-3 w-3 mr-1.5" />
      {text}
    </span>
  );
}
