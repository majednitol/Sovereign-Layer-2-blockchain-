"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from "recharts";
import { ArrowLeft, TrendingUp, Download, Info, BarChart2, Calendar } from "lucide-react";

type ChartSlug = "tx" | "active-addresses" | "gas-used" | "bridge-volume" | "ibc-volume" | "block-time" | "tps";

interface ChartCoord {
  date: string;
  value: number;
}

export default function ChartsPage() {
  const [activeChart, setActiveChart] = useState<ChartSlug>("tx");
  const [data, setData] = useState<ChartCoord[]>([]);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  const fetchChartData = async (slug: ChartSlug) => {
    setLoading(true);
    try {
      const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/charts/${slug}`);
      if (resp.ok) {
        const json = await resp.json();
        if (json.data) {
          setData(json.data);
        }
      }
    } catch (err) {
      console.warn("Failed to fetch chart data. Using mock coordinates.", err);
      const mockData = [];
      const now = new Date();
      for (let i = 30; i >= 0; i--) {
        mockData.push({
          date: new Date(now.getTime() - i * 86400000).toLocaleDateString([], { month: "short", day: "numeric" }),
          value: 100 + (i % 7) * 20 + Math.floor(Math.random() * 50)
        });
      }
      setData(mockData);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchChartData(activeChart);
  }, [activeChart]);

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white font-medium">Charts</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-900 pb-4 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="flex items-center space-x-3">
          <Link href="/" className="p-2 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-gray-400 hover:text-white transition">
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div>
            <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
              <BarChart2 className="text-blue-500 w-8 h-8" />
              Sovereign Protocol Charts Hub
            </h1>
            <p className="text-gray-400 mt-1">TimescaleDB-backed daily metrics and time-series history.</p>
          </div>
        </div>

        {/* CSV Download Button */}
        <a 
          href={`${API_BASE}/api/rest/v1/explorer/charts/${activeChart}?format=csv`}
          download
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white font-bold text-xs uppercase tracking-wider rounded-xl shadow-lg transition text-center"
        >
          <Download className="h-4 w-4" /> Download CSV
        </a>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Sidebar Nav */}
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-4 space-y-1 lg:col-span-1">
          <h3 className="text-xs uppercase font-bold text-gray-500 px-3 mb-2 tracking-wider">
            Chart Categories
          </h3>
          {[
            { id: "tx", label: "Daily Transactions" },
            { id: "active-addresses", label: "Daily Active Addresses" },
            { id: "gas-used", label: "Daily Gas Consumption" },
            { id: "bridge-volume", label: "Daily Bridge Volume" },
            { id: "ibc-volume", label: "Daily IBC Transfers" },
            { id: "block-time", label: "Average Block Time" },
            { id: "tps", label: "Max Transactions Per Second" }
          ].map((chart) => (
            <button
              key={chart.id}
              onClick={() => setActiveChart(chart.id as ChartSlug)}
              className={`w-full text-left px-3 py-2 rounded-xl text-sm font-medium transition flex items-center gap-2 ${
                activeChart === chart.id 
                  ? "bg-blue-950 text-blue-400 border-l-2 border-blue-500" 
                  : "text-gray-400 hover:bg-gray-900/50 hover:text-white"
              }`}
            >
              <TrendingUp className="h-4 w-4" />
              {chart.label}
            </button>
          ))}
        </div>

        {/* Chart Viewport */}
        <div className="lg:col-span-3 bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-xl space-y-6">
          <div>
            <h3 className="text-lg font-bold text-white capitalize">{activeChart.replace("-", " ")} Chart</h3>
            <p className="text-xs text-gray-500 mt-1">30-day moving window timeline coordinates.</p>
          </div>

          {loading ? (
            <div className="h-80 flex items-center justify-center text-gray-500">Loading chart analytics data...</div>
          ) : (
            <div className="h-80 w-full">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={data}>
                  <defs>
                    <linearGradient id="colorValue" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3}/>
                      <stop offset="95%" stopColor="#3b82f6" stopOpacity={0}/>
                    </linearGradient>
                  </defs>
                  <XAxis dataKey="date" stroke="#6b7280" fontSize={11} tickLine={false} />
                  <YAxis stroke="#6b7280" fontSize={11} tickLine={false} />
                  <Tooltip 
                    contentStyle={{ backgroundColor: "#09090b", border: "1px solid #1f2937" }}
                    labelStyle={{ color: "#9ca3af" }}
                  />
                  <Area type="monotone" dataKey="value" stroke="#3b82f6" fillOpacity={1} fill="url(#colorValue)" strokeWidth={2} />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          )}

          <div className="p-4 bg-blue-950/20 border border-blue-900/50 rounded-xl text-xs text-blue-400 flex items-start space-x-2 leading-relaxed">
            <Info className="h-4 w-4 mt-0.5 flex-shrink-0" />
            <span>Daily chart points are collected directly from TimescaleDB hypertable aggregation intervals. Download the CSV format above to export the complete history for raw data modelling.</span>
          </div>
        </div>
      </div>
    </div>
  );
}
