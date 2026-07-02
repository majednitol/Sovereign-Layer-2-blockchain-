"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { 
  Cpu, AlertTriangle, CheckCircle, Clock, ArrowLeft, TrendingUp, 
  ShieldAlert, Activity, CheckCircle2, ShieldCheck, Flame, RefreshCw 
} from "lucide-react";
import { ComposedChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, Line } from "recharts";

interface FeedDetail {
  feedId: string;
  title: string;
  latestPrice: string;
  status: "fresh" | "stale" | "stale-blocked";
  lastUpdated: string;
  stalenessThresholdSec: number;
}

interface PricePoint {
  time: string;
  open: number;
  high: number;
  low: number;
  close: number;
}

interface OperatorSubmission {
  operator: string;
  roundId: number;
  price: string;
  status: "submitted" | "missed" | "slashed";
  timestamp: string;
}

// Custom Candlestick shape for Recharts ComposedChart
const CandlestickShape = (props: any) => {
  const { x, y, width, height, open, close, high, low } = props;
  if (x === undefined || y === undefined || width === undefined || height === undefined) return null;
  const isUp = close >= open;
  const color = isUp ? "#22c55e" : "#ef4444";
  
  // Wick line coordinates
  const cx = x + width / 2;
  const yHigh = y - (high - Math.max(open, close)) * (height / Math.max(0.00001, Math.abs(open - close)));
  const yLow = y + height + (Math.min(open, close) - low) * (height / Math.max(0.00001, Math.abs(open - close)));

  return (
    <g>
      {/* Wick */}
      <line x1={cx} y1={yHigh || y} x2={cx} y2={yLow || (y + height)} stroke={color} strokeWidth={1.5} />
      {/* Body */}
      <rect x={x} y={y} width={width} height={Math.max(2, height)} fill={color} rx={1} />
    </g>
  );
};

export default function OracleFeedDetailPage() {
  const params = useParams();
  const feedId = params?.feedId ? String(params.any || params.feedId) : "";

  const [feed, setFeed] = useState<FeedDetail | null>(null);
  const [history, setHistory] = useState<PricePoint[]>([]);
  const [submissions, setSubmissions] = useState<OperatorSubmission[]>([]);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    if (!feedId) return;
    const fetchFeedDetails = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/oracle/feeds/${feedId}`);
        if (resp.ok) {
          const data = await resp.json();
          setFeed({
            feedId: data.feedId || feedId,
            title: data.title || `Price Feed ${feedId.toUpperCase()}`,
            latestPrice: data.latestPrice || "0.00",
            status: data.status || "fresh",
            lastUpdated: data.lastUpdated || new Date().toISOString(),
            stalenessThresholdSec: 60,
          });
        }
      } catch (err) {
        console.warn("Using simulated feed details", err);
        setFeed({
          feedId: feedId,
          title: `Sovereign L1 ${feedId.toUpperCase()} Price Feed`,
          latestPrice: feedId.includes("btc") ? "97350.00" : "1.25",
          status: "fresh",
          lastUpdated: new Date().toISOString(),
          stalenessThresholdSec: 60,
        });
      }

      // Populate candlestick data
      const basePrice = feedId.includes("btc") ? 97300 : 1.2;
      const points: PricePoint[] = Array.from({ length: 15 }).map((_, i) => {
        const open = basePrice + Math.random() * (feedId.includes("btc") ? 50 : 0.05) - (feedId.includes("btc") ? 25 : 0.025);
        const close = open + Math.random() * (feedId.includes("btc") ? 40 : 0.04) - (feedId.includes("btc") ? 20 : 0.02);
        const high = Math.max(open, close) + Math.random() * (feedId.includes("btc") ? 15 : 0.01);
        const low = Math.min(open, close) - Math.random() * (feedId.includes("btc") ? 15 : 0.01);
        return {
          time: new Date(Date.now() - (15 - i) * 60000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }),
          open,
          high,
          low,
          close,
        };
      });
      setHistory(points);

      // Populate operator submissions
      setSubmissions([
        { operator: "sovereign1operator0x82f9", roundId: 105, price: feedId.includes("btc") ? "97350.00" : "1.25", status: "submitted", timestamp: new Date(Date.now() - 10000).toISOString() },
        { operator: "sovereign1operator0x12a5", roundId: 105, price: feedId.includes("btc") ? "97349.50" : "1.24", status: "submitted", timestamp: new Date(Date.now() - 15000).toISOString() },
        { operator: "sovereign1operator0x9c3f", roundId: 105, price: "0.00", status: "slashed", timestamp: new Date(Date.now() - 8000).toISOString() },
      ]);

      setLoading(false);
    };

    fetchFeedDetails();
  }, [feedId]);

  if (loading) {
    return (
      <div className="p-6 max-w-7xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading feed details...</div>
      </div>
    );
  }

  // 3-State Staleness Indicator Badge
  const renderStalenessBadge = (status: FeedDetail["status"]) => {
    switch (status) {
      case "fresh":
        return (
          <span className="flex items-center gap-1.5 px-3 py-1 text-xs bg-green-950/60 text-green-400 border border-green-900 rounded-lg font-bold uppercase tracking-wider">
            <CheckCircle2 className="h-3.5 w-3.5 animate-pulse" /> Fresh
          </span>
        );
      case "stale":
        return (
          <span className="flex items-center gap-1.5 px-3 py-1 text-xs bg-yellow-950/60 text-yellow-400 border border-yellow-900 rounded-lg font-bold uppercase tracking-wider animate-pulse">
            <Clock className="h-3.5 w-3.5" /> Stale
          </span>
        );
      case "stale-blocked":
        return (
          <span className="flex items-center gap-1.5 px-3 py-1 text-xs bg-red-950/60 text-red-400 border border-red-900 rounded-lg font-bold uppercase tracking-wider">
            <Flame className="h-3.5 w-3.5 animate-bounce" /> Stale Blocked
          </span>
        );
    }
  };

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/oracle" className="hover:text-white transition">Oracle</Link>
        <span>/</span>
        <span className="text-gray-300 font-mono text-xs">{feedId}</span>
      </nav>

      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between border-b border-gray-800 pb-6 gap-4">
        <div className="space-y-1">
          <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
            <Cpu className="w-8 h-8 text-blue-500" />
            {feed?.title}
          </h1>
          <p className="text-xs text-gray-400">Unique Feed Identifier: <span className="font-mono text-gray-300 font-bold">{feed?.feedId}</span></p>
        </div>

        <div className="flex items-center gap-4 bg-gray-950 border border-gray-900 p-4 rounded-2xl shadow-lg">
          <div className="text-right">
            <div className="text-xs text-gray-500 uppercase font-bold tracking-wider">Latest Price</div>
            <div className="text-2xl font-extrabold text-white font-mono">${feed?.latestPrice}</div>
          </div>
          {feed && renderStalenessBadge(feed.status)}
        </div>
      </div>

      {/* Candlestick Composed Chart */}
      <div className="bg-gray-950 border border-gray-900 p-6 rounded-2xl space-y-4 shadow-xl">
        <div className="flex justify-between items-center">
          <h2 className="text-lg font-bold text-white flex items-center gap-2">
            <TrendingUp className="w-5 h-5 text-blue-500" />
            Oracle OHLC Performance (Candlesticks)
          </h2>
          <span className="text-xs px-2.5 py-1 bg-gray-900 border border-gray-800 text-gray-400 font-mono rounded">
            Interval: 1m
          </span>
        </div>
        <div className="h-72 w-full">
          <ResponsiveContainer width="100%" height="100%">
            <ComposedChart data={history} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
              <XAxis dataKey="time" stroke="#6b7280" fontSize={10} tickLine={false} />
              <YAxis stroke="#6b7280" fontSize={10} tickLine={false} domain={['auto', 'auto']} />
              <Tooltip 
                contentStyle={{ backgroundColor: '#030712', borderColor: '#1f2937', color: '#fff', borderRadius: '12px' }}
                labelClassName="font-bold text-xs"
              />
              <Bar 
                dataKey="close" 
                shape={<CandlestickShape />} 
                tooltipType="none"
              />
            </ComposedChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Consensus Rounds */}
        <div className="lg:col-span-2 bg-gray-950 border border-gray-900 rounded-2xl shadow-xl overflow-hidden">
          <div className="p-5 border-b border-gray-900 flex justify-between items-center">
            <h3 className="text-lg font-bold text-white flex items-center gap-2">
              <Activity className="h-5 w-5 text-indigo-500" /> Recent Consensus Rounds
            </h3>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm text-gray-400">
              <thead className="bg-black/40 text-xs text-gray-500 uppercase tracking-wider font-semibold">
                <tr>
                  <th className="p-4">Round ID</th>
                  <th className="p-4">Timestamp</th>
                  <th className="p-4">Median Price</th>
                  <th className="p-4">Status</th>
                  <th className="p-4 text-right">Details</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-900">
                {[105, 104, 103, 102].map((round) => (
                  <tr key={round} className="hover:bg-gray-900/30 transition">
                    <td className="p-4 font-mono font-bold text-white">#{round}</td>
                    <td className="p-4 text-xs">{new Date(Date.now() - (105 - round) * 60000).toLocaleTimeString()}</td>
                    <td className="p-4 font-mono text-white font-bold">${feed?.latestPrice}</td>
                    <td className="p-4">
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 text-[10px] bg-green-950 text-green-400 border border-green-900 rounded font-bold uppercase">
                        <CheckCircle className="h-3 w-3" /> Finalized
                      </span>
                    </td>
                    <td className="p-4 text-right">
                      <Link
                        href={`/oracle/rounds/${round}`}
                        className="text-xs text-blue-500 hover:text-blue-400 font-semibold"
                      >
                        Inspect &rarr;
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Live Operator submissions list */}
        <div className="bg-gray-950 border border-gray-900 rounded-2xl shadow-xl p-5 space-y-4">
          <h3 className="text-lg font-bold text-white flex items-center gap-2">
            <ShieldCheck className="h-5 w-5 text-green-500" /> Active Round Operators
          </h3>
          <div className="space-y-3">
            {submissions.map((sub, i) => (
              <div key={i} className="bg-gray-900/40 border border-gray-850 p-3.5 rounded-xl space-y-2">
                <div className="flex justify-between items-start">
                  <div className="space-y-0.5">
                    <span className="font-mono text-xs text-blue-400 break-all">{sub.operator}</span>
                    <div className="text-[10px] text-gray-500">Round #{sub.roundId}</div>
                  </div>
                  <span className={`px-2 py-0.5 text-[9px] font-bold uppercase rounded border ${
                    sub.status === "submitted" 
                      ? "bg-green-950 text-green-400 border-green-900" 
                      : "bg-red-950 text-red-400 border-red-900"
                  }`}>
                    {sub.status}
                  </span>
                </div>
                {sub.status === "submitted" ? (
                  <div className="flex justify-between items-center text-xs font-mono">
                    <span className="text-gray-500">Reported Price:</span>
                    <span className="text-white font-bold">${sub.price}</span>
                  </div>
                ) : (
                  <div className="text-xs text-red-400 font-medium">
                    Slashed for submission timeout or faulty median deviation.
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
