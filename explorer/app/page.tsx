"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { Search, Server, Cpu, Database, Activity, RefreshCw } from "lucide-react";
import BlockTicker from "@/components/BlockTicker";

interface Block {
  height: number;
  time: string;
  proposer: string;
  txCount: number;
  gasUsed: number;
}

interface Tx {
  hash: string;
  height: number;
  time: string;
  type: string;
  msgTypes: string[];
  status: number;
  fee: number;
}

export default function HomePage() {
  const [blocks, setBlocks] = useState<Block[]>([]);
  const [txs, setTxs] = useState<Tx[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState("");
  const [tps, setTps] = useState(12.4);
  const [validatorsCount, setValidatorsCount] = useState(30);
  const [error, setError] = useState<string | null>(null);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  const fetchDashboardData = async () => {
    try {
      const blocksResp = await fetch(`${API_BASE}/api/rest/v1/explorer/blocks`);
      if (blocksResp.ok) {
        const data = await blocksResp.json();
        if (data.blocks) {
          setBlocks(data.blocks.map((b: any) => ({
            height: Number(b.height),
            time: b.time,
            proposer: b.proposer,
            txCount: Number(b.txCount || 0),
            gasUsed: Number(b.gasUsed || 0),
          })));
        }
      }

      const txsResp = await fetch(`${API_BASE}/api/rest/v1/explorer/txs`);
      if (txsResp.ok) {
        const data = await txsResp.json();
        if (data.txs) {
          setTxs(data.txs.map((t: any) => ({
            hash: t.hash,
            height: Number(t.height),
            time: t.time,
            type: t.type,
            msgTypes: t.msgTypes || [],
            status: Number(t.status || 0),
            fee: Number(t.fee || 0),
          })));
        }
      }

      // Fetch real TPS
      const tpsResp = await fetch(`${API_BASE}/api/rest/v1/explorer/tps`);
      if (tpsResp.ok) {
        const data = await tpsResp.json();
        if (data.points && data.points.length > 0) {
          setTps(Number(data.points[data.points.length - 1].tps || 0));
        }
      }

      // Fetch real validator count
      const valResp = await fetch(`${API_BASE}/api/rest/v1/explorer/validators`);
      if (valResp.ok) {
        const data = await valResp.json();
        if (data.validators) {
          setValidatorsCount(data.validators.length);
        }
      }
    } catch (err) {
      console.warn("Failed to fetch dashboard data.", err);
      setError("Unable to reach explorer API. Please check the API service.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchDashboardData();
    const interval = setInterval(fetchDashboardData, 5000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    let ws: WebSocket | null = null;
    let isMounted = true;

    const connectWS = () => {
      try {
        const WS_BASE = process.env.NEXT_PUBLIC_COSMOS_RPC_WS || "ws://localhost:26657/websocket";
        ws = new WebSocket(WS_BASE);

        ws.onopen = () => {
          if (!isMounted) return;
          console.log("Homepage WebSocket connected.");
          ws?.send(JSON.stringify({
            jsonrpc: "2.0",
            method: "subscribe",
            id: 1,
            params: { query: "tm.event='NewBlock'" }
          }));
        };

        ws.onmessage = (event) => {
          if (!isMounted) return;
          try {
            const data = JSON.parse(event.data);
            if (data.result?.query === "tm.event='NewBlock'") {
              console.log("New block event received via WS. Refreshing dashboard...");
              fetchDashboardData();
            }
          } catch (e) {
            console.error("Error parsing WS message", e);
          }
        };

        ws.onclose = () => {
          console.log("Homepage WebSocket closed, reconnecting in 5s...");
          if (isMounted) {
            setTimeout(connectWS, 5000);
          }
        };

        ws.onerror = (err) => {
          console.warn("Homepage WebSocket error", err);
        };
      } catch (err) {
        console.warn("Failed to instantiate homepage WebSocket", err);
      }
    };

    connectWS();

    const interval = setInterval(() => {
      fetchDashboardData();
    }, 5000);

    return () => {
      isMounted = false;
      if (ws) ws.close();
      clearInterval(interval);
    };
  }, []);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    if (!searchQuery) return;
    const query = searchQuery.trim();
    if (query.startsWith("0x") && query.length === 66) {
      window.location.href = `/txs/${query}`;
    } else if (query.length === 64) {
      window.location.href = `/txs/${query}`;
    } else if (/^\d+$/.test(query)) {
      window.location.href = `/blocks/${query}`;
    } else if (
      query.startsWith("cosmos1") ||
      query.startsWith("sovereign1") ||
      query.startsWith("sov1") ||
      (query.startsWith("0x") && query.length === 42)
    ) {
      window.location.href = `/address/${query}`;
    } else {
      window.location.href = `/search?q=${encodeURIComponent(query)}`;
    }
  };

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-8">
      {/* Search Section */}
      <div className="bg-gradient-to-r from-blue-950 via-gray-900 to-indigo-950 border border-gray-800 rounded-2xl p-8 text-center space-y-6 shadow-xl">
        <h2 className="text-3xl font-extrabold tracking-tight text-white md:text-4xl">
          Sovereign L1 Explorer
        </h2>
        <p className="text-gray-400 max-w-2xl mx-auto text-sm md:text-base">
          Search block height, transaction hash, smart contract address, or account.
        </p>

        <form onSubmit={handleSearch} className="max-w-2xl mx-auto flex items-center space-x-2">
          <div className="relative flex-grow">
            <Search className="absolute left-4 top-3.5 h-5 w-5 text-gray-500" />
            <input
              type="text"
              placeholder="Search height, hash, address..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-12 pr-4 py-3 bg-black/50 border border-gray-800 focus:border-blue-600 focus:ring-1 focus:ring-blue-600 rounded-xl text-white outline-none transition"
            />
          </div>
          <button
            type="submit"
            className="px-6 py-3 bg-blue-600 hover:bg-blue-500 text-white font-medium rounded-xl transition shadow-lg shadow-blue-900/30"
          >
            Search
          </button>
        </form>
      </div>

      {/* Network Stats Card */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 flex items-center space-x-4">
          <Activity className="h-8 w-8 text-blue-500" />
          <div>
            <div className="text-xs text-gray-500 uppercase font-bold">Live TPS</div>
            <div className="text-2xl font-semibold text-white">{tps} tx/s</div>
          </div>
        </div>

        <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 flex items-center space-x-4">
          <Server className="h-8 w-8 text-green-500" />
          <div>
            <div className="text-xs text-gray-500 uppercase font-bold">Validators</div>
            <div className="text-2xl font-semibold text-white">{validatorsCount} Active</div>
          </div>
        </div>

        <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 flex items-center space-x-4">
          <Cpu className="h-8 w-8 text-purple-500" />
          <div>
            <div className="text-xs text-gray-500 uppercase font-bold">Avg Block Time</div>
            <div className="text-2xl font-semibold text-white">3.00s</div>
          </div>
        </div>

        <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 flex items-center space-x-4">
          <Database className="h-8 w-8 text-indigo-500" />
          <div>
            <div className="text-xs text-gray-500 uppercase font-bold">Latest Height</div>
            <div className="text-2xl font-semibold text-white">
              {blocks.length > 0 ? blocks[0].height : "120532"}
            </div>
          </div>
        </div>
      </div>

      {/* Tables Feed */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Blocks Feed */}
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 space-y-4">
          <div className="flex justify-between items-center">
            <h3 className="text-lg font-bold text-white">Recent Blocks</h3>
            <Link href="/blocks" className="text-xs text-blue-500 hover:text-blue-400">View All</Link>
          </div>

          <div className="divide-y divide-gray-900">
            {blocks.map((block) => (
              <div key={block.height} className="py-4 flex justify-between items-center text-sm">
                <div>
                  <Link href={`/blocks/${block.height}`} className="text-blue-500 hover:underline font-bold">
                    #{block.height}
                  </Link>
                  <div className="text-xs text-gray-500 mt-0.5">
                    Proposer: {block.proposer}
                  </div>
                </div>
                <div className="text-right">
                  <div className="text-white font-medium">{block.txCount} txs</div>
                  <div className="text-xs text-gray-500 mt-0.5">
                    {new Date(block.time).toLocaleTimeString()}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Transactions Feed */}
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 space-y-4">
          <div className="flex justify-between items-center">
            <h3 className="text-lg font-bold text-white">Recent Transactions</h3>
            <Link href="/txs" className="text-xs text-blue-500 hover:text-blue-400">View All</Link>
          </div>

          <div className="divide-y divide-gray-900">
            {txs.map((tx) => (
              <div key={tx.hash} className="py-4 flex justify-between items-center text-sm">
                <div>
                  <Link href={`/txs/${tx.hash}`} className="text-blue-500 hover:underline font-mono font-medium">
                    {tx.hash.slice(0, 16)}...
                  </Link>
                  <div className="flex items-center space-x-2 text-xs text-gray-500 mt-0.5">
                    <span className="capitalize px-1.5 py-0.5 bg-gray-900 rounded text-gray-400 border border-gray-800">
                      {tx.type}
                    </span>
                    <span>{tx.msgTypes[0] || "Msg"}</span>
                  </div>
                </div>
                <div className="text-right">
                  <div className="text-white font-medium">Fee: {tx.fee} uSLT</div>
                  <span className={`inline-block w-2 h-2 rounded-full mt-1.5 ${tx.status === 0 ? "bg-green-500" : "bg-red-500"}`}></span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
