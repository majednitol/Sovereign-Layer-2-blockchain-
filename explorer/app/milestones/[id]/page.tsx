"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { 
  Milestone, Flag, Shield, CheckCircle2, Calendar, Clock, 
  ArrowLeft, ShieldAlert, Award, Play, Pause 
} from "lucide-react";

interface MilestoneEvent {
  id: number;
  height: number;
  eventType: string;
  value: string;
  time: string;
}

interface MilestoneDetail {
  id: number;
  creator: string;
  status: "achieved" | "pending" | "expired";
  title: string;
  targetPrice: string;
  feedId: string;
  achievedHeight: number;
  expiredHeight: number;
  startHeight: number;
  deadlineHeight: number;
  currentHeight: number;
  totalPausedDuration: number;
  events: MilestoneEvent[];
}

// Component 1: Animated SVG State Machine Visualizer
function StateMachineViz({ status }: { status: "achieved" | "pending" | "expired" }) {
  return (
    <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-lg flex flex-col items-center justify-center space-y-4">
      <h3 className="text-xs font-bold text-gray-500 uppercase tracking-wider self-start">State Transition Engine</h3>
      
      <svg className="w-full max-w-md h-32" viewBox="0 0 400 100">
        <defs>
          <marker id="arrow" viewBox="0 0 10 10" refX="6" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
            <path d="M 0 2 L 10 5 L 0 8 z" fill="#374151" />
          </marker>
          <marker id="arrow-active" viewBox="0 0 10 10" refX="6" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
            <path d="M 0 2 L 10 5 L 0 8 z" fill="#3b82f6" />
          </marker>
        </defs>

        {/* Node 1: Pending */}
        <circle cx="60" cy="50" r="22" fill="#030712" stroke={status === "pending" ? "#3b82f6" : "#374151"} strokeWidth="2.5" className={status === "pending" ? "animate-pulse" : ""} />
        <text x="60" y="54" fill={status === "pending" ? "#3b82f6" : "#9ca3af"} fontSize="9" fontWeight="bold" textAnchor="middle">PENDING</text>

        {/* Connection 1 */}
        <line x1="86" y1="50" x2="164" y2="50" stroke={status === "pending" ? "#3b82f6" : "#374151"} strokeWidth="2" strokeDasharray={status === "pending" ? "4,4" : ""} markerEnd={`url(#${status === "pending" ? "arrow-active" : "arrow"})`} />

        {/* Node 2: Blocked / Processing */}
        <circle cx="200" cy="50" r="22" fill="#030712" stroke="#374151" strokeWidth="2.5" />
        <text x="200" y="54" fill="#9ca3af" fontSize="9" fontWeight="bold" textAnchor="middle">BLOCKED</text>

        {/* Connection 2 */}
        <line x1="226" y1="50" x2="304" y2="50" stroke={status === "achieved" || status === "expired" ? "#22c55e" : "#374151"} strokeWidth="2" markerEnd={`url(#arrow)`} />

        {/* Node 3: Final State (Achieved or Expired) */}
        {status === "achieved" ? (
          <>
            <circle cx="340" cy="50" r="22" fill="#030712" stroke="#22c55e" strokeWidth="2.5" />
            <text x="340" y="54" fill="#22c55e" fontSize="9" fontWeight="bold" textAnchor="middle">ACHIEVED</text>
          </>
        ) : status === "expired" ? (
          <>
            <circle cx="340" cy="50" r="22" fill="#030712" stroke="#ef4444" strokeWidth="2.5" />
            <text x="340" y="54" fill="#ef4444" fontSize="9" fontWeight="bold" textAnchor="middle">EXPIRED</text>
          </>
        ) : (
          <>
            <circle cx="340" cy="50" r="22" fill="#030712" stroke="#374151" strokeWidth="2.5" />
            <text x="340" y="54" fill="#9ca3af" fontSize="9" fontWeight="bold" textAnchor="middle">RESOLVED</text>
          </>
        )}
      </svg>
    </div>
  );
}

// Component 2: Horizontal Deadline Timeline with Pause segments
function DeadlineTimeline({ m }: { m: MilestoneDetail }) {
  const totalDuration = m.deadlineHeight - m.startHeight;
  const elapsed = Math.min(totalDuration, Math.max(0, m.currentHeight - m.startHeight));
  const progressPct = totalDuration > 0 ? (elapsed / totalDuration) * 100 : 0;

  return (
    <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-xl space-y-4">
      <h3 className="text-xs font-bold text-gray-500 uppercase tracking-wider">Deadline Block Timeline</h3>
      
      <div className="space-y-4">
        {/* Timeline Bar */}
        <div className="w-full bg-gray-900 h-4 rounded-full overflow-hidden relative border border-gray-800 flex">
          {/* Progress Segment */}
          <div className="bg-blue-600 h-full transition-all duration-500" style={{ width: `${progressPct}%` }} />
          
          {/* Pause Segment representation */}
          {m.totalPausedDuration > 0 && (
            <div 
              className="absolute bg-yellow-500/80 h-full border-x border-yellow-600" 
              style={{ left: "40%", width: "15%" }} 
              title={`SLA Execution Paused: ${m.totalPausedDuration}s`} 
            />
          )}
        </div>

        {/* Milestones labels */}
        <div className="flex justify-between items-center text-xs font-mono text-gray-500">
          <div className="space-y-0.5">
            <div>Start Block</div>
            <div className="text-white font-bold">#{m.startHeight}</div>
          </div>
          <div className="space-y-0.5 text-center">
            <div>Current Block</div>
            <div className="text-blue-400 font-bold">#{m.currentHeight}</div>
          </div>
          {m.achievedHeight > 0 && (
            <div className="space-y-0.5 text-center">
              <div>Achieved Block</div>
              <div className="text-green-400 font-bold">#{m.achievedHeight}</div>
            </div>
          )}
          <div className="space-y-0.5 text-right">
            <div>Deadline Block</div>
            <div className="text-red-400 font-bold">#{m.deadlineHeight}</div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default function MilestoneDetailPage() {
  const params = useParams();
  const id = params?.id ? Number(params.id) : 1;
  const [m, setM] = useState<MilestoneDetail | null>(null);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchMilestone = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/milestones/${id}`);
        if (resp.ok) {
          const data = await resp.json();
          setM({
            id: Number(data.id || id),
            creator: data.creator,
            status: data.status,
            title: data.title,
            targetPrice: data.targetPrice,
            feedId: data.feedId,
            achievedHeight: Number(data.achievedHeight || 0),
            expiredHeight: Number(data.expiredHeight || 0),
            startHeight: Number(data.startHeight || 100),
            deadlineHeight: Number(data.deadlineHeight || 10000),
            currentHeight: Number(data.currentHeight || 5500),
            totalPausedDuration: Number(data.totalPausedDuration || 0),
            events: data.events || [],
          });
        } else {
          throw new Error("Milestone details not found");
        }
      } catch (err) {
        console.warn("Using simulated milestone details", err);
        setM({
          id: id,
          creator: "sovereign1creatoraddress",
          status: id === 3 ? "pending" : "achieved",
          title: id === 3 ? "Cross-Chain Settlement Bridge V3" : "Genesis Ring Verification Milestone",
          targetPrice: id === 3 ? "1.50" : "1.00",
          feedId: "slt-usdt",
          achievedHeight: id === 3 ? 0 : 120,
          expiredHeight: 0,
          startHeight: 100,
          deadlineHeight: id === 3 ? 20000 : 1000,
          currentHeight: 650,
          totalPausedDuration: id === 3 ? 120 : 0,
          events: [
            { id: 1, height: 100, eventType: "create", value: "init", time: new Date(Date.now() - 36000000).toISOString() },
            { id: 2, height: 120, eventType: "achieved", value: "success", time: new Date().toISOString() },
          ],
        });
      } finally {
        setLoading(false);
      }
    };
    fetchMilestone();
  }, [id]);

  if (loading) {
    return (
      <div className="p-6 max-w-7xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading milestone details...</div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/milestones" className="hover:text-white transition">Milestones</Link>
        <span>/</span>
        <span className="text-gray-300 font-mono text-xs">Milestone #{id}</span>
      </nav>

      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between border-b border-gray-800 pb-6 gap-4">
        <div className="space-y-1">
          <div className="flex items-center gap-3">
            <Link href="/milestones" className="p-2 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-gray-400 hover:text-white transition">
              <ArrowLeft className="h-4 w-4" />
            </Link>
            <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
              <Milestone className="w-8 h-8 text-blue-500" />
              {m?.title}
            </h1>
            <span className={`px-2.5 py-0.5 text-xs font-semibold rounded uppercase border ${
              m?.status === "achieved" ? "bg-green-950/50 text-green-400 border-green-900" :
              m?.status === "expired" ? "bg-red-950/50 text-red-400 border-red-900" :
              "bg-blue-950/50 text-blue-400 border-blue-900"
            }`}>
              {m?.status}
            </span>
          </div>
          <p className="text-xs text-gray-400">Milestone Creator: <span className="font-mono text-gray-300">{m?.creator}</span></p>
        </div>
      </div>

      {/* State visualizers */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {m && <StateMachineViz status={m.status} />}
        {m && <DeadlineTimeline m={m} />}
      </div>

      {/* Stats Panel */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-6">
        <div className="bg-gray-950 border border-gray-900 p-5 rounded-2xl space-y-1 shadow-lg">
          <div className="text-xs text-gray-500 uppercase tracking-wider font-bold">Achieved Block Height</div>
          <div className="text-2xl font-extrabold text-white font-mono">
            {m?.achievedHeight && m.achievedHeight > 0 ? `#${m.achievedHeight}` : "N/A"}
          </div>
        </div>
        <div className="bg-gray-950 border border-gray-900 p-5 rounded-2xl space-y-1 shadow-lg">
          <div className="text-xs text-gray-500 uppercase tracking-wider font-bold">Oracle Target</div>
          <div className="text-2xl font-extrabold text-white">${m?.targetPrice} ({m?.feedId})</div>
        </div>
        <div className="bg-gray-950 border border-gray-900 p-5 rounded-2xl space-y-1 shadow-lg">
          <div className="text-xs text-gray-500 uppercase tracking-wider font-bold">Total Paused Duration</div>
          <div className="text-2xl font-extrabold text-white">{m?.totalPausedDuration}s</div>
        </div>
      </div>

      {/* Events Timeline */}
      <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-4 shadow-lg">
        <h2 className="text-lg font-bold text-white">Verification Progression Log</h2>
        <div className="space-y-4">
          {m?.events.map((e, index) => (
            <div key={index} className="flex items-start gap-4 border-l border-gray-900 pl-4 py-2 relative">
              <span className="absolute -left-[7px] top-3.5 h-3 w-3 rounded-full bg-blue-600 border border-gray-950"></span>
              <div>
                <div className="font-semibold text-gray-200 uppercase text-xs">{e.eventType}</div>
                <div className="text-[10px] text-gray-500 font-mono mt-0.5">
                  Block #{e.height} &bull; {new Date(e.time).toLocaleString()}
                </div>
                <div className="text-xs text-gray-400 mt-1 font-mono bg-gray-900/60 p-2 rounded border border-gray-850">
                  Value Payload: {e.value}
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
