"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { Cpu, ShieldAlert, Award, Star, Activity, AlertTriangle, ShieldCheck } from "lucide-react";

interface Operator {
  address: string;
  moniker: string;
  reputationScore: number;
  participationRate: string;
  slashCount: number;
  lastActive: string;
  trend: "up" | "down" | "stable";
}

interface SlashEvent {
  height: number;
  operator: string;
  reason: string;
  amount: string;
  time: string;
}

export default function OracleOperatorsPage() {
  const [operators, setOperators] = useState<Operator[]>([]);
  const [slashes, setSlashes] = useState<SlashEvent[]>([]);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchOperatorsAndSlashes = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/oracle/operators`);
        if (resp.ok) {
          const data = await resp.json();
          if (data.operators) {
            setOperators(data.operators);
          }
        }
      } catch (err) {
        console.warn("Using simulated operators list", err);
        setOperators([
          { address: "sovereignvaloper1address1", moniker: "Sovereign Validator #1", reputationScore: 99, participationRate: "99.8%", slashCount: 0, lastActive: new Date().toISOString(), trend: "up" },
          { address: "sovereignvaloper1address2", moniker: "Genesis Validator L1", reputationScore: 95, participationRate: "97.5%", slashCount: 1, lastActive: new Date().toISOString(), trend: "stable" },
          { address: "sovereignvaloper1address3", moniker: "Faulty Validator Node", reputationScore: 72, participationRate: "82.4%", slashCount: 3, lastActive: new Date().toISOString(), trend: "down" },
        ]);
      }

      // Simulated recent slashing events
      setSlashes([
        { height: 118400, operator: "sovereignvaloper1address3", reason: "Oracle report timeout cycle exceed", amount: "500 uSLT", time: new Date(Date.now() - 3600000).toISOString() },
        { height: 116200, operator: "sovereignvaloper1address3", reason: "Submitted price value out of median bound check", amount: "1,500 uSLT", time: new Date(Date.now() - 7200000).toISOString() },
        { height: 95000, operator: "sovereignvaloper1address2", reason: "Double signing slot block consensus", amount: "10,000 uSLT", time: new Date(Date.now() - 172800000).toISOString() },
      ]);

      setLoading(false);
    };
    fetchOperatorsAndSlashes();
  }, []);

  if (loading) {
    return (
      <div className="p-6 max-w-7xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading operators...</div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/oracle" className="hover:text-white transition">Oracle</Link>
        <span>/</span>
        <span className="text-gray-300">Operators</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4">
        <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
          <Star className="w-8 h-8 text-blue-500" />
          Oracle Operators Directory
        </h1>
        <p className="text-gray-400 mt-2">Active price reporting nodes, reputation metrics, and SLA status.</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Operators list */}
        <div className="lg:col-span-2 bg-gray-950 border border-gray-900 rounded-2xl shadow-xl overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm text-gray-400">
              <thead className="bg-gray-900/40 text-xs text-gray-500 uppercase tracking-wider font-bold">
                <tr>
                  <th className="p-4">Operator Info</th>
                  <th className="p-4">Reputation Score</th>
                  <th className="p-4">Participation Rate</th>
                  <th className="p-4">Slashes</th>
                  <th className="p-4 text-right">Trend</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-900">
                {operators.map((op) => (
                  <tr key={op.address} className="hover:bg-gray-900/30 transition">
                    <td className="p-4">
                      <div className="space-y-0.5">
                        <div className="font-bold text-white text-xs">{op.moniker}</div>
                        <span className="font-mono text-[10px] text-gray-500 break-all">{op.address}</span>
                      </div>
                    </td>
                    <td className="p-4 font-semibold text-green-400">{op.reputationScore}/100</td>
                    <td className="p-4 text-gray-300 font-mono">{op.participationRate}</td>
                    <td className="p-4">
                      <span className={`px-2 py-0.5 text-xs rounded font-bold uppercase border ${
                        op.slashCount > 0 
                          ? "bg-red-950/40 border-red-900/50 text-red-400" 
                          : "bg-green-950/40 border-green-900/50 text-green-400"
                      }`}>
                        {op.slashCount} Slashes
                      </span>
                    </td>
                    <td className="p-4 text-right">
                      <span className={`inline-flex items-center px-2 py-0.5 rounded text-[10px] font-bold uppercase ${
                        op.trend === "up" ? "bg-green-950 text-green-400" :
                        op.trend === "down" ? "bg-red-950 text-red-400" : "bg-gray-900 text-gray-400"
                      }`}>
                        {op.trend}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Slashing Feed */}
        <div className="bg-gray-950 border border-gray-900 rounded-2xl shadow-xl p-5 space-y-4 h-fit">
          <h3 className="text-lg font-bold text-white flex items-center gap-2">
            <AlertTriangle className="h-5 w-5 text-red-500 animate-pulse" />
            Recent Slashing Logs
          </h3>
          <div className="space-y-3">
            {slashes.map((s, i) => (
              <div key={i} className="bg-gray-900/40 border border-gray-850 p-3 rounded-xl space-y-2 text-xs">
                <div className="flex justify-between items-center">
                  <span className="font-mono font-bold text-gray-400">Block #{s.height}</span>
                  <span className="text-[10px] text-gray-500">{new Date(s.time).toLocaleTimeString()}</span>
                </div>
                <div className="font-mono text-blue-400 break-all">{s.operator.slice(0, 15)}...{s.operator.slice(-8)}</div>
                <p className="text-gray-300 leading-normal">{s.reason}</p>
                <div className="flex justify-between items-center border-t border-gray-850 pt-1.5 text-[10px]">
                  <span className="text-gray-500 font-bold uppercase">Penalty:</span>
                  <span className="text-red-400 font-bold font-mono">{s.amount}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
