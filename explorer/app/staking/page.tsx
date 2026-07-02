"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { Coins, Flame, Award, HelpCircle, ArrowRight } from "lucide-react";

interface StakingStats {
  totalBonded: string;
  bondedRatio: string;
  inflation: string;
  communityPool: string;
  apr: string;
}

export default function StakingPage() {
  const [stats, setStats] = useState<StakingStats>({
    totalBonded: "450,000,000 uSLT",
    bondedRatio: "45.0%",
    inflation: "7.0%",
    communityPool: "10,000,000 uSLT",
    apr: "12.5%",
  });
  const [loading, setLoading] = useState(true);
  const [stakeAmount, setStakeAmount] = useState("1000");
  const [calculatedYield, setCalculatedYield] = useState(125);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchStakingStats = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/staking/stats`);
        if (resp.ok) {
          const data = await resp.json();
          setStats({
            totalBonded: data.totalBonded,
            bondedRatio: data.bondedRatio,
            inflation: data.inflation,
            communityPool: data.communityPool,
            apr: data.apr,
          });
        }
      } catch (err) {
        console.warn("Using simulated staking stats", err);
      } finally {
        setLoading(false);
      }
    };
    fetchStakingStats();
  }, []);

  useEffect(() => {
    const amt = parseFloat(stakeAmount);
    if (!isNaN(amt)) {
      const aprVal = parseFloat(stats.apr.replace("%", "")) / 100;
      setCalculatedYield(+(amt * aprVal).toFixed(2));
    } else {
      setCalculatedYield(0);
    }
  }, [stakeAmount, stats.apr]);

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-8">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white">Staking</span>
      </nav>

      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight text-white">Staking Dashboard</h1>
        <p className="text-gray-400 mt-1 font-normal">
          Manage your delegations, view chain inflation, and estimate validation rewards.
        </p>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 flex items-center space-x-4">
          <Coins className="h-8 w-8 text-blue-500" />
          <div>
            <div className="text-xs text-gray-500 uppercase font-bold">Total Bonded</div>
            <div className="text-lg font-semibold text-white">{stats.totalBonded}</div>
            <div className="text-xs text-gray-500 mt-0.5">{stats.bondedRatio} of supply</div>
          </div>
        </div>

        <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 flex items-center space-x-4">
          <Flame className="h-8 w-8 text-orange-500" />
          <div>
            <div className="text-xs text-gray-500 uppercase font-bold">Inflation</div>
            <div className="text-lg font-semibold text-white">{stats.inflation}</div>
            <div className="text-xs text-gray-500 mt-0.5">Dynamic recalculation</div>
          </div>
        </div>

        <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 flex items-center space-x-4">
          <Award className="h-8 w-8 text-green-500" />
          <div>
            <div className="text-xs text-gray-500 uppercase font-bold">Staking APR</div>
            <div className="text-lg font-semibold text-green-400">{stats.apr}</div>
            <div className="text-xs text-gray-500 mt-0.5">Estimated reward rate</div>
          </div>
        </div>

        <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 flex items-center space-x-4">
          <Coins className="h-8 w-8 text-purple-500" />
          <div>
            <div className="text-xs text-gray-500 uppercase font-bold">Community Pool</div>
            <div className="text-lg font-semibold text-white">{stats.communityPool}</div>
            <div className="text-xs text-gray-500 mt-0.5">Governance treasury</div>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* APR Calculator */}
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 space-y-6">
          <h3 className="text-lg font-bold text-white">Staking APR Calculator</h3>
          <div className="space-y-4">
            <div>
              <label className="block text-xs font-bold text-gray-400 uppercase mb-2">
                Staking Amount (SLT)
              </label>
              <input
                type="number"
                value={stakeAmount}
                onChange={(e) => setStakeAmount(e.target.value)}
                className="w-full px-4 py-2.5 bg-black border border-gray-800 focus:border-blue-600 focus:ring-1 focus:ring-blue-600 rounded-lg text-white outline-none font-mono"
              />
            </div>

            <div className="p-4 bg-gray-900/50 border border-gray-850 rounded-xl space-y-2">
              <div className="flex justify-between items-center text-sm">
                <span className="text-gray-400">Yield Rate</span>
                <span className="font-semibold text-white">{stats.apr}</span>
              </div>
              <div className="flex justify-between items-center border-t border-gray-800 pt-2 text-base font-medium">
                <span className="text-gray-300">Estimated Annual Rewards</span>
                <span className="font-bold text-green-400">{calculatedYield} SLT</span>
              </div>
            </div>
          </div>
        </div>

        {/* Quick Delegation Actions Card */}
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 flex flex-col justify-between space-y-6">
          <div className="space-y-2">
            <h3 className="text-lg font-bold text-white">Delegate Tokens</h3>
            <p className="text-gray-400 text-sm leading-relaxed">
              Bond your tokens directly to any of the 30 active validator slots to help secure the network and earn rewards. Staking involves a 21-day unbonding period.
            </p>
          </div>
          <div>
            <Link
              href="/address/sovereign1qyqszqszqszqszqszqszqszqszqszqyqszq/stake"
              className="inline-flex items-center space-x-2 text-sm px-4 py-2.5 bg-blue-600 hover:bg-blue-500 text-white rounded-lg font-medium transition shadow-lg shadow-blue-900/20"
            >
              <span>Go to Delegation Form</span>
              <ArrowRight className="h-4 w-4" />
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
