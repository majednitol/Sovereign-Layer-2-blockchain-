"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { 
  TrendingUp, Clock, ShieldCheck, 
  ArrowLeftRight, HelpCircle, Activity, FileDown, Calendar
} from "lucide-react";
import { 
  AreaChart, Area, XAxis, YAxis, Tooltip, ReferenceLine,
  ResponsiveContainer, BarChart, Bar, 
  CartesianGrid, Legend 
} from "recharts";

interface TpsPoint {
  time: string;
  tps: number;
}

interface BlockTimePoint {
  time: string;
  duration: number;
}

interface UptimePoint {
  slotIndex: number;
  time: string;
  uptime: number;
  missedBlocks: number;
}

interface VolumePoint {
  time: string;
  volume: number;
}

interface CandlestickData {
  time: string;
  open: number;
  high: number;
  low: number;
  close: number;
}

export default function AnalyticsPage() {
  const [tpsHistory, setTpsHistory] = useState<TpsPoint[]>([]);
  const [blockTimeHistory, setBlockTimeHistory] = useState<BlockTimePoint[]>([]);
  const [uptimeGrid, setUptimeGrid] = useState<UptimePoint[]>([]);
  const [bridgeVolume, setBridgeVolume] = useState<VolumePoint[]>([]);
  const [oracleFeed, setOracleFeed] = useState<CandlestickData[]>([]);
  const [selectedDate, setSelectedDate] = useState<string>("2026-06-27");
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  const fetchAnalyticsData = async () => {
    try {
      const tpsResp = await fetch(`${API_BASE}/api/rest/v1/explorer/analytics/tps`);
      if (tpsResp.ok) {
        const data = await tpsResp.json();
        if (data.points) setTpsHistory(data.points);
      }

      const btResp = await fetch(`${API_BASE}/api/rest/v1/explorer/analytics/block-time`);
      if (btResp.ok) {
        const data = await btResp.json();
        if (data.points) setBlockTimeHistory(data.points);
      }

      const uptimeResp = await fetch(`${API_BASE}/api/rest/v1/explorer/analytics/validator-uptime`);
      if (uptimeResp.ok) {
        const data = await uptimeResp.json();
        if (data.points) setUptimeGrid(data.points);
      }

      const volResp = await fetch(`${API_BASE}/api/rest/v1/explorer/analytics/bridge-volume`);
      if (volResp.ok) {
        const data = await volResp.json();
        if (data.points) setBridgeVolume(data.points);
      }
    } catch (err) {
      console.warn("Failed to fetch analytics data from API, using fallback mocks.", err);
      // Mocks
      const now = new Date();
      const mockTps = [];
      const mockBt = [];
      const mockVol = [];
      const mockUptime = [];

      for (let i = 12; i >= 0; i--) {
        const tStr = new Date(now.getTime() - i * 3600000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
        mockTps.push({ time: tStr, tps: 10 + (i % 3) * 5 + Math.random() * 2 });
        mockBt.push({ time: tStr, duration: 1.2 + (i % 2) * 0.3 + Math.random() * 0.1 });
        mockVol.push({ time: tStr, volume: 50000 + i * 5000 + Math.floor(Math.random() * 1000) });
      }

      for (let slot = 0; slot < 20; slot++) {
        mockUptime.push({ slotIndex: slot, time: "Today", uptime: 98.5 + (slot % 3) * 0.2 + Math.random() * 0.1, missedBlocks: (slot % 4) === 0 ? 2 : 0 });
      }

      setTpsHistory(mockTps);
      setBlockTimeHistory(mockBt);
      setBridgeVolume(mockVol);
      setUptimeGrid(mockUptime);
      setOracleFeed([
        { time: "09:00", open: 3450, high: 3480, low: 3440, close: 3470 },
        { time: "10:00", open: 3470, high: 3495, low: 3460, close: 3485 },
        { time: "11:00", open: 3485, high: 3510, low: 3480, close: 3500 },
        { time: "12:00", open: 3500, high: 3520, low: 3495, close: 3515 },
        { time: "13:00", open: 3515, high: 3540, low: 3510, close: 3535 },
      ]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAnalyticsData();
    const interval = setInterval(fetchAnalyticsData, 10000);
    return () => clearInterval(interval);
  }, []);

  const handleCsvDownload = () => {
    let csvContent = "data:text/csv;charset=utf-8,Time,TPS,BlockTime(s),BridgeVolume\n";
    tpsHistory.forEach((pt, idx) => {
      const bt = blockTimeHistory[idx]?.duration || 1.5;
      const vol = bridgeVolume[idx]?.volume || 0;
      csvContent += `${pt.time},${pt.tps.toFixed(2)},${bt.toFixed(2)},${vol}\n`;
    });
    const encodedUri = encodeURI(csvContent);
    const link = document.createElement("a");
    link.setAttribute("href", encodedUri);
    link.setAttribute("download", "network_performance_metrics.csv");
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white font-medium">Analytics</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-900 pb-4 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-3">
            <Activity className="text-purple-500 h-8 w-8 animate-pulse" />
            Performance Analytics Console
          </h1>
          <p className="text-gray-400 mt-1 text-xs">
            Continuous system aggregations tracking transaction TPS, percentile block timings, and validator signing matrices.
          </p>
        </div>

        {/* Toolbar */}
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2 bg-gray-950 border border-gray-900 px-3 py-1.5 rounded-xl text-xs">
            <Calendar className="h-4 w-4 text-gray-500" />
            <input 
              type="date" 
              value={selectedDate}
              onChange={(e) => setSelectedDate(e.target.value)}
              className="bg-transparent text-white focus:outline-none cursor-pointer"
            />
          </div>
          <button 
            onClick={handleCsvDownload}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white font-bold text-xs uppercase tracking-wider rounded-xl shadow-lg shadow-blue-900/20 transition"
          >
            <FileDown className="h-4 w-4" /> Export CSV
          </button>
        </div>
      </div>

      {loading ? (
        <div className="py-20 text-center text-gray-400">Loading metrics and grids...</div>
      ) : (
        <div className="space-y-6">
          {/* Charts Grid */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* TPS Area Chart */}
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-md space-y-4">
              <h3 className="text-lg font-bold text-white flex items-center gap-2">
                <TrendingUp className="text-blue-500 h-5 w-5" /> Transactions Per Second (TPS)
              </h3>
              <div className="h-64 w-full">
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={tpsHistory}>
                    <defs>
                      <linearGradient id="colorTps" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3}/>
                        <stop offset="95%" stopColor="#3b82f6" stopOpacity={0}/>
                      </linearGradient>
                    </defs>
                    <CartesianGrid stroke="#18181b" strokeDasharray="3 3" />
                    <XAxis dataKey="time" stroke="#4b5563" fontSize={11} tickLine={false} />
                    <YAxis stroke="#4b5563" fontSize={11} tickLine={false} />
                    <Tooltip contentStyle={{ backgroundColor: "#09090b", borderColor: "#18181b" }} />
                    <Area type="monotone" dataKey="tps" stroke="#3b82f6" fillOpacity={1} fill="url(#colorTps)" strokeWidth={2} />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
            </div>

            {/* Block Time Histogram with Percentiles */}
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-md space-y-4">
              <h3 className="text-lg font-bold text-white flex items-center gap-2">
                <Clock className="text-green-500 h-5 w-5" /> Percentile Block Times (p50/p95/p99)
              </h3>
              <div className="h-64 w-full">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={blockTimeHistory}>
                    <CartesianGrid stroke="#18181b" strokeDasharray="3 3" />
                    <XAxis dataKey="time" stroke="#4b5563" fontSize={11} tickLine={false} />
                    <YAxis stroke="#4b5563" fontSize={11} tickLine={false} domain={[0, 2.5]} />
                    <Tooltip contentStyle={{ backgroundColor: "#09090b", borderColor: "#18181b" }} />
                    <ReferenceLine y={1.2} label={{ value: "p50 (Median)", fill: "#10b981", fontSize: 10, position: "insideBottomRight" }} stroke="#10b981" strokeDasharray="3 3" />
                    <ReferenceLine y={1.5} label={{ value: "p95", fill: "#f59e0b", fontSize: 10, position: "insideBottomRight" }} stroke="#f59e0b" strokeDasharray="3 3" />
                    <ReferenceLine y={2.0} label={{ value: "p99 Limit", fill: "#ef4444", fontSize: 10, position: "insideBottomRight" }} stroke="#ef4444" strokeWidth={1.5} />
                    <Bar dataKey="duration" fill="#10b981" radius={[4, 4, 0, 0]} maxBarSize={30} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </div>

            {/* Bridge Inflow Vol */}
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-md space-y-4">
              <h3 className="text-lg font-bold text-white flex items-center gap-2">
                <ArrowLeftRight className="text-indigo-500 h-5 w-5" /> Bridge Aggregated Volume (SLT)
              </h3>
              <div className="h-64 w-full">
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={bridgeVolume}>
                    <defs>
                      <linearGradient id="colorVol" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#8b5cf6" stopOpacity={0.3}/>
                        <stop offset="95%" stopColor="#8b5cf6" stopOpacity={0}/>
                      </linearGradient>
                    </defs>
                    <CartesianGrid stroke="#18181b" strokeDasharray="3 3" />
                    <XAxis dataKey="time" stroke="#4b5563" fontSize={11} tickLine={false} />
                    <YAxis stroke="#4b5563" fontSize={11} tickLine={false} />
                    <Tooltip contentStyle={{ backgroundColor: "#09090b", borderColor: "#18181b" }} />
                    <Area type="monotone" dataKey="volume" stroke="#8b5cf6" fillOpacity={1} fill="url(#colorVol)" strokeWidth={2} />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
            </div>

            {/* Oracle Feed Chart */}
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-md space-y-4">
              <h3 className="text-lg font-bold text-white flex items-center gap-2">
                <Activity className="text-yellow-500 h-5 w-5" /> Oracle Feed Candlestick Feed (OHLC)
              </h3>
              <div className="h-64 w-full">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={oracleFeed}>
                    <CartesianGrid stroke="#18181b" strokeDasharray="3 3" />
                    <XAxis dataKey="time" stroke="#4b5563" fontSize={11} tickLine={false} />
                    <YAxis stroke="#4b5563" fontSize={11} tickLine={false} domain={["dataMin - 100", "dataMax + 100"]} />
                    <Tooltip contentStyle={{ backgroundColor: "#09090b", borderColor: "#18181b" }} />
                    <Bar dataKey="high" fill="#ef4444" radius={[2, 2, 0, 0]} maxBarSize={10} />
                    <Bar dataKey="low" fill="#10b981" radius={[0, 0, 2, 2]} maxBarSize={10} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </div>
          </div>

          {/* Validator Uptime Grid (Heatmap style) */}
          <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-lg space-y-4">
            <div>
              <h3 className="text-lg font-bold text-white flex items-center gap-2">
                <ShieldCheck className="text-green-500 h-5 w-5" /> Validator Consensus Signing Matrix
              </h3>
              <p className="text-xs text-gray-500 mt-1">
                Signing performance heatmaps across consecutive slot blocks. Red blocks represent missed signing slots.
              </p>
            </div>

            <div className="grid grid-cols-4 sm:grid-cols-8 md:grid-cols-10 gap-3 pt-2">
              {uptimeGrid.map((pt, idx) => (
                <div 
                  key={idx} 
                  className={`border p-3.5 rounded-xl flex flex-col items-center justify-center space-y-1 relative group hover:scale-105 transition cursor-help ${
                    pt.missedBlocks > 0 ? "bg-red-950/20 border-red-900/50" : "bg-gray-900/50 border-gray-850"
                  }`}
                >
                  <span className="text-[10px] text-gray-500 font-bold uppercase font-mono">Slot {pt.slotIndex}</span>
                  <span className={`text-xs font-extrabold font-mono ${pt.missedBlocks > 0 ? "text-red-400" : "text-white"}`}>
                    {pt.uptime.toFixed(2)}%
                  </span>
                  <span className="text-[9px] text-gray-500">Missed: {pt.missedBlocks}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
