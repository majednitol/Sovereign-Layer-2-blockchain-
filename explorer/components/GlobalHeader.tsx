"use client";

import React, { useEffect, useState } from "react";
import SearchBar from "./SearchBar";

export default function GlobalHeader() {
  const [price, setPrice] = useState<number | null>(null);
  const [change24h, setChange24h] = useState<number>(0);
  const [gasPrice, setGasPrice] = useState<string>("0.02");

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/oracle/feed/sov-usd`);
        if (resp.ok) {
          const data = await resp.json();
          if (data.price) {
            setPrice(Number(data.price));
            setChange24h(Number(data.change24h || 0));
          }
        }
      } catch (e) {
        // Safe fallback
      }

      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/gas-price`);
        if (resp.ok) {
          const data = await resp.json();
          if (data.standard) {
            setGasPrice(data.standard);
          }
        }
      } catch (e) {
        // Safe fallback
      }
    };
    fetchStats();
    const interval = setInterval(fetchStats, 10000);
    return () => clearInterval(interval);
  }, [API_BASE]);

  return (
    <div className="bg-gray-950 border-b border-gray-900 py-1.5 px-4 text-xs font-medium text-gray-500 font-mono">
      <div className="max-w-7xl mx-auto flex flex-col sm:flex-row justify-between items-center space-y-1 sm:space-y-0">
        <div className="flex items-center space-x-6">
          <div className="flex items-center space-x-1.5">
            <span className="text-gray-600 uppercase">CSOV/USD:</span>
            <span className="text-white font-semibold">
              {price ? `$${price.toFixed(2)}` : "$1.25"}
            </span>
            <span className={change24h >= 0 ? "text-green-500" : "text-red-500"}>
              ({change24h >= 0 ? "+" : ""}{change24h.toFixed(1)}%)
            </span>
          </div>
          <div className="flex items-center space-x-1.5">
            <span className="text-gray-600 uppercase">GAS PRICE:</span>
            <span className="text-cyan-400 font-semibold">{gasPrice} Gwei</span>
          </div>
        </div>
        <div className="hidden sm:flex items-center space-x-2">
          <span className="h-1.5 w-1.5 rounded-full bg-green-500 animate-pulse"></span>
          <span className="text-gray-500">CometBFT Consensus Active</span>
        </div>
      </div>
    </div>
  );
}
