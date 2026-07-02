"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { Server, Activity, ShieldCheck, HelpCircle } from "lucide-react";

interface Validator {
  address: string;
  slotIndex: number;
  power: number;
  status: string;
  missedBlocks: number;
  certificationScore: number;
}

export default function ValidatorsPage() {
  const [validators, setValidators] = useState<Validator[]>([]);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchValidators = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/validators`);
        if (resp.ok) {
          const data = await resp.json();
          if (data.validators) {
            setValidators(data.validators.map((v: any) => ({
              address: v.address,
              slotIndex: Number(v.slotIndex),
              power: Number(v.power),
              status: v.status,
              missedBlocks: Number(v.missedBlocks),
              certificationScore: Number(v.certificationScore),
            })));
          }
        }
      } catch (err) {
        console.warn("Using empty validator slot grid", err);
        setValidators([]);
      } finally {
        setLoading(false);
      }
    };
    fetchValidators();
  }, []);

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white">Validators</span>
      </nav>

      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight text-white">Validator Slot Grid</h1>
        <p className="text-gray-400 mt-1">
          Sovereign L1 validator sets. 30 active slots with equal consensus power.
        </p>
      </div>

      {/* Staking Summary cards */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 flex items-center space-x-4">
          <Server className="h-8 w-8 text-blue-500" />
          <div>
            <div className="text-xs text-gray-500 uppercase font-bold">Total Active Slots</div>
            <div className="text-2xl font-semibold text-white">30 / 30</div>
          </div>
        </div>

        <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 flex items-center space-x-4">
          <ShieldCheck className="h-8 w-8 text-green-500" />
          <div>
            <div className="text-xs text-gray-500 uppercase font-bold">Avg Attestation Score</div>
            <div className="text-2xl font-semibold text-white">97.8%</div>
          </div>
        </div>

        <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 flex items-center space-x-4">
          <Activity className="h-8 w-8 text-indigo-500" />
          <div>
            <div className="text-xs text-gray-500 uppercase font-bold">Consensus Mode</div>
            <div className="text-2xl font-semibold text-white">Equal Power</div>
          </div>
        </div>
      </div>

      {/* Grid */}
      <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 lg:grid-cols-6 gap-4">
        {loading ? (
          <div className="col-span-full py-12 text-center text-gray-500">Loading validator grid...</div>
        ) : (
          validators.map((val) => (
            <Link
              key={val.slotIndex}
              href={`/validators/${val.address}`}
              className="group block bg-gray-950 border border-gray-900 hover:border-blue-600 rounded-xl p-4 transition-all duration-300 transform hover:-translate-y-1 shadow-md hover:shadow-lg hover:shadow-blue-900/10"
            >
              <div className="flex justify-between items-start mb-2">
                <span className="text-xs font-bold text-gray-500 font-mono group-hover:text-blue-400">
                  SLOT #{val.slotIndex.toString().padStart(2, "0")}
                </span>
                <span className="h-2 w-2 rounded-full bg-green-500 animate-pulse"></span>
              </div>

              <div className="space-y-1">
                <div className="text-sm font-medium text-white font-mono truncate">
                  {val.address.slice(0, 12)}...
                </div>
                <div className="flex justify-between items-center text-xs mt-3">
                  <span className="text-gray-400">Attest Score</span>
                  <span className="font-semibold text-green-400">{val.certificationScore}%</span>
                </div>
                <div className="flex justify-between items-center text-xs">
                  <span className="text-gray-400">Missed</span>
                  <span className="font-mono text-gray-300">{val.missedBlocks} blocks</span>
                </div>
              </div>
            </Link>
          ))
        )}
      </div>
    </div>
  );
}
