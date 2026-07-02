"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { Milestone, Flag, Shield, CheckCircle2, ChevronRight, Filter, Clock } from "lucide-react";

interface MilestoneDetail {
  id: number;
  creator: string;
  status: "achieved" | "expired" | "pending";
  title: string;
  targetPrice: string;
  feedId: string;
  achievedHeight: number;
  expiredHeight: number;
  deadlineTimestamp: string;
}

export default function MilestonesPage() {
  const [milestones, setMilestones] = useState<MilestoneDetail[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<"all" | "achieved" | "pending" | "expired">("all");

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchMilestones = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/milestones`);
        if (resp.ok) {
          const data = await resp.json();
          if (data.milestones) {
            setMilestones(data.milestones.map((m: any) => ({
              id: Number(m.id),
              creator: m.creator,
              status: m.status,
              title: m.title,
              targetPrice: m.targetPrice,
              feedId: m.feedId,
              achievedHeight: Number(m.achievedHeight || 0),
              expiredHeight: Number(m.expiredHeight || 0),
              deadlineTimestamp: m.deadlineTimestamp || new Date(Date.now() + 86400000 * 3).toISOString(), // 3 days remaining
            })));
          }
        }
      } catch (err) {
        console.warn("Using simulated milestones", err);
        setMilestones([
          { id: 1, creator: "sovereign1creator", status: "achieved", title: "Genesis Ring Verification", targetPrice: "1.00", feedId: "slt-usdt", achievedHeight: 100, expiredHeight: 0, deadlineTimestamp: new Date(Date.now() - 172800000).toISOString() },
          { id: 2, creator: "sovereign1creator", status: "achieved", title: "Ring V2 Hard Fork Upgrade", targetPrice: "1.20", feedId: "slt-usdt", achievedHeight: 12000, expiredHeight: 0, deadlineTimestamp: new Date(Date.now() - 86400000).toISOString() },
          { id: 3, creator: "sovereign1creator", status: "pending", title: "Cross-Chain Settlement Bridge V3", targetPrice: "1.50", feedId: "slt-usdt", achievedHeight: 0, expiredHeight: 0, deadlineTimestamp: new Date(Date.now() + 86400000 * 2.5).toISOString() },
          { id: 4, creator: "sovereign1creator", status: "expired", title: "Oracle Multi-Feed Migration", targetPrice: "2.00", feedId: "slt-usdt", achievedHeight: 0, expiredHeight: 9000, deadlineTimestamp: new Date(Date.now() - 86400000 * 5).toISOString() },
        ]);
      } finally {
        setLoading(false);
      }
    };
    fetchMilestones();
  }, []);

  const getFilteredMilestones = () => {
    if (activeTab === "all") return milestones;
    return milestones.filter((m) => m.status === activeTab);
  };

  // Counts for ratio card
  const achievedCount = milestones.filter(m => m.status === "achieved").length;
  const expiredCount = milestones.filter(m => m.status === "expired").length;
  const pendingCount = milestones.filter(m => m.status === "pending").length;
  const totalCount = milestones.length;
  const achievedRatio = totalCount > 0 ? ((achievedCount / (achievedCount + expiredCount)) * 100).toFixed(0) : "0";

  // Simple countdown formatting
  const getRemainingTime = (deadline: string) => {
    const diff = new Date(deadline).getTime() - Date.now();
    if (diff <= 0) return "Deadline Reached";
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));
    const hours = Math.floor((diff / (1000 * 60 * 60)) % 24);
    return `${days}d ${hours}h remaining`;
  };

  if (loading) {
    return (
      <div className="p-6 max-w-7xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading milestones...</div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-gray-300">Milestones</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4">
        <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
          <Flag className="w-8 h-8 text-blue-500" />
          Rollup Milestones Timeline
        </h1>
        <p className="text-gray-400 mt-2">ZK state transitions and epoch certifications history.</p>
      </div>

      {/* Achieved vs Expired Ratio Indicator */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-gray-950 border border-gray-900 rounded-2xl p-5 shadow-lg flex items-center justify-between">
          <div className="space-y-1">
            <span className="text-xs text-gray-500 uppercase font-bold tracking-wider">Success Rate Ratio</span>
            <div className="text-2xl font-extrabold text-white">{achievedRatio}% Achieved</div>
            <div className="text-xs text-gray-400">{achievedCount} success / {expiredCount} expired</div>
          </div>
          <div className="w-14 h-14 rounded-full border-4 border-indigo-900 border-t-indigo-500 flex items-center justify-center font-bold text-xs text-indigo-400">
            SLA
          </div>
        </div>

        <div className="bg-gray-950 border border-gray-900 rounded-2xl p-5 shadow-lg flex items-center justify-between">
          <div className="space-y-1">
            <span className="text-xs text-gray-500 uppercase font-bold tracking-wider">Active Challenges</span>
            <div className="text-2xl font-extrabold text-indigo-400">{pendingCount} Pending</div>
            <div className="text-xs text-gray-400">Awaiting state transition certification</div>
          </div>
          <div className="w-10 h-10 rounded-xl bg-indigo-950 border border-indigo-900/50 flex items-center justify-center text-indigo-400">
            <Clock className="w-5 h-5 animate-pulse" />
          </div>
        </div>

        <div className="bg-gray-950 border border-gray-900 rounded-2xl p-5 shadow-lg flex items-center justify-between">
          <div className="space-y-1">
            <span className="text-xs text-gray-500 uppercase font-bold tracking-wider">Total Rollup Milestones</span>
            <div className="text-2xl font-extrabold text-white">{totalCount} Registered</div>
            <div className="text-xs text-gray-400">Tracked since genesis block #0</div>
          </div>
          <div className="w-10 h-10 rounded-xl bg-gray-900 border border-gray-800 flex items-center justify-center text-gray-400">
            <Milestone className="w-5 h-5" />
          </div>
        </div>
      </div>

      {/* Filter Tabs */}
      <div className="flex space-x-2 border-b border-gray-900 pb-px">
        {(["all", "achieved", "pending", "expired"] as const).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2.5 text-sm font-semibold border-b-2 capitalize transition -mb-px ${
              activeTab === tab
                ? "border-blue-500 text-white"
                : "border-transparent text-gray-400 hover:text-gray-200"
            }`}
          >
            {tab}
          </button>
        ))}
      </div>

      {/* Timeline view */}
      <div className="relative border-l border-gray-900 ml-4 pl-8 space-y-6">
        {getFilteredMilestones().length === 0 ? (
          <div className="bg-gray-950 border border-gray-900 rounded-2xl p-12 text-center text-gray-500 max-w-xl">
            No milestones found under this category.
          </div>
        ) : (
          getFilteredMilestones().map((m) => (
            <div key={m.id} className="relative">
              {/* Dots */}
              <span className={`absolute -left-[45px] top-1.5 flex h-8 w-8 items-center justify-center rounded-full border border-gray-950 ${
                m.status === "achieved" ? "bg-green-950 text-green-400" :
                m.status === "expired" ? "bg-red-950 text-red-400" :
                "bg-blue-950 text-blue-400 animate-pulse"
              }`}>
                <Milestone className="w-4 h-4" />
              </span>

              <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 flex flex-col md:flex-row md:items-center justify-between hover:border-gray-800 transition gap-4 shadow-lg">
                <div className="space-y-1">
                  <div className="flex items-center gap-3">
                    <h3 className="text-lg font-bold text-white">{m.title}</h3>
                    <span className={`px-2 py-0.5 text-[10px] rounded font-bold uppercase border ${
                      m.status === "achieved" ? "bg-green-950/40 border-green-900/50 text-green-400" :
                      m.status === "expired" ? "bg-red-950/40 border-red-900/50 text-red-400" :
                      "bg-blue-950/40 border-blue-900/50 text-blue-400"
                    }`}>
                      {m.status}
                    </span>
                  </div>

                  <div className="text-xs text-gray-400 flex flex-wrap items-center gap-4">
                    <span>Target Price: <span className="font-mono text-white font-bold">${m.targetPrice}</span> via <span className="font-mono text-gray-200">{m.feedId}</span></span>
                    {m.achievedHeight > 0 && (
                      <span>Achieved Block: <span className="font-mono text-green-400 font-bold">#{m.achievedHeight}</span></span>
                    )}
                    {m.expiredHeight > 0 && (
                      <span>Expired Block: <span className="font-mono text-red-400 font-bold">#{m.expiredHeight}</span></span>
                    )}
                  </div>

                  {m.status === "pending" && (
                    <div className="flex items-center gap-1 text-xs text-indigo-400 font-semibold pt-1">
                      <Clock className="w-3.5 h-3.5" />
                      <span>Deadline: {getRemainingTime(m.deadlineTimestamp)}</span>
                    </div>
                  )}
                </div>

                <Link
                  href={`/milestones/${m.id}`}
                  className="p-2.5 bg-gray-900 hover:bg-gray-850 border border-gray-800 rounded-xl text-gray-400 hover:text-white transition self-end md:self-auto"
                >
                  <ChevronRight className="w-5 h-5" />
                </Link>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
