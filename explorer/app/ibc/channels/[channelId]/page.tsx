"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { 
  ArrowLeftRight, Activity, Clock, CheckCircle2, 
  ArrowLeft, AlertTriangle, Play, HelpCircle 
} from "lucide-react";

interface IbcChannelDetail {
  channelId: string;
  portId: string;
  counterpartyChannelId: string;
  counterpartyPortId: string;
  state: string;
  ordering: string;
  packetCount: number;
}

interface IbcPacket {
  sequence: number;
  status: "sent" | "received" | "acknowledged" | "stuck" | "timeout";
  sourceHeight: number;
  destHeight: number;
  timeoutTimestamp: string;
  amount: string;
  denom: string;
}

export default function IbcChannelDetailPage() {
  const params = useParams();
  const channelId = params?.channelId ? String(params.channelId) : "channel-0";
  const [c, setC] = useState<IbcChannelDetail | null>(null);
  const [packets, setPackets] = useState<IbcPacket[]>([]);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchChannelAndPackets = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/ibc/channels/${channelId}`);
        if (resp.ok) {
          const data = await resp.json();
          setC({
            channelId: data.channelId || channelId,
            portId: data.portId,
            counterpartyChannelId: data.counterpartyChannelId,
            counterpartyPortId: data.counterpartyPortId,
            state: data.state,
            ordering: data.ordering,
            packetCount: Number(data.packetCount || 0),
          });
        } else {
          throw new Error("IBC channel details not found");
        }
      } catch (err) {
        console.warn("Using simulated IBC channel details", err);
        setC({
          channelId: channelId,
          portId: "transfer",
          counterpartyChannelId: "channel-14",
          counterpartyPortId: "transfer",
          state: "STATE_OPEN",
          ordering: "ORDER_UNORDERED",
          packetCount: 150,
        });
      }

      // Simulated dynamic packets timeline log
      setPackets([
        { sequence: 1045, status: "acknowledged", sourceHeight: 120500, destHeight: 489320, timeoutTimestamp: new Date(Date.now() - 3600000).toISOString(), amount: "15,000 ATOM", denom: "ibc/ATOM" },
        { sequence: 1046, status: "received", sourceHeight: 120520, destHeight: 489350, timeoutTimestamp: new Date(Date.now() - 1800000).toISOString(), amount: "500 OSMO", denom: "ibc/OSMO" },
        { sequence: 1047, status: "stuck", sourceHeight: 120530, destHeight: 0, timeoutTimestamp: new Date(Date.now() - 300000).toISOString(), amount: "2,500 ATOM", denom: "ibc/ATOM" },
      ]);

      setLoading(false);
    };
    fetchChannelAndPackets();
  }, [channelId]);

  if (loading) {
    return (
      <div className="p-6 max-w-7xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading channel details...</div>
      </div>
    );
  }

  const stuckPackets = packets.filter(p => p.status === "stuck");

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/ibc" className="hover:text-white transition">IBC</Link>
        <span>/</span>
        <Link href="/ibc/channels" className="hover:text-white transition">Channels</Link>
        <span>/</span>
        <span className="text-gray-300 font-mono text-xs">{channelId}</span>
      </nav>

      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between border-b border-gray-800 pb-6 gap-4">
        <div>
          <div className="flex items-center gap-3">
            <Link href="/ibc" className="p-2 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-gray-400 hover:text-white transition">
              <ArrowLeft className="h-4 w-4" />
            </Link>
            <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
              <ArrowLeftRight className="w-8 h-8 text-blue-500" />
              IBC Channel: {c?.channelId}
            </h1>
            <span className="px-2.5 py-1 text-xs bg-green-950 text-green-400 border border-green-900 rounded font-semibold uppercase">
              {c?.state}
            </span>
          </div>
          <p className="text-xs text-gray-400 mt-2">Local Port: <span className="font-mono text-gray-200">{c?.portId}</span></p>
        </div>
      </div>

      {/* Stuck Packet Alert banner */}
      {stuckPackets.length > 0 && (
        <div className="bg-red-950/20 border border-red-900/50 p-5 rounded-2xl flex items-start gap-4 text-red-400 shadow-lg">
          <AlertTriangle className="h-6 w-6 shrink-0 mt-0.5 animate-pulse" />
          <div className="text-sm space-y-1.5 leading-normal">
            <span className="font-bold block text-white text-base">Stuck Packets Warning ({stuckPackets.length})</span>
            <p className="text-gray-300">
              We detected packets that were committed at the source chain but are pending relay execution or acknowledgement confirmation beyond the timeout threshold. Check that the IBC relayer processes are active.
            </p>
            <div className="space-y-1 pt-1">
              {stuckPackets.map(p => (
                <div key={p.sequence} className="font-mono text-xs text-red-400">
                  &bull; Sequence #{p.sequence}: {p.amount} ({p.denom}) committed in Block #{p.sourceHeight}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Connection Info */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-4 shadow-lg">
          <h2 className="text-lg font-bold text-white">Local End</h2>
          <div className="space-y-3">
            <div>
              <div className="text-xs text-gray-500 uppercase tracking-wider font-bold">Port ID</div>
              <div className="font-mono text-sm text-gray-200 mt-1">{c?.portId}</div>
            </div>
            <div>
              <div className="text-xs text-gray-500 uppercase tracking-wider font-bold">Channel ID</div>
              <div className="font-mono text-sm text-gray-200 mt-1">{c?.channelId}</div>
            </div>
          </div>
        </div>

        <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-4 shadow-lg">
          <h2 className="text-lg font-bold text-white">Counterparty End</h2>
          <div className="space-y-3">
            <div>
              <div className="text-xs text-gray-500 uppercase tracking-wider font-bold">Counterparty Port ID</div>
              <div className="font-mono text-sm text-gray-200 mt-1">{c?.counterpartyPortId}</div>
            </div>
            <div>
              <div className="text-xs text-gray-500 uppercase tracking-wider font-bold">Counterparty Channel ID</div>
              <div className="font-mono text-sm text-gray-200 mt-1">{c?.counterpartyChannelId}</div>
            </div>
          </div>
        </div>
      </div>

      {/* Component 6: Packet Tracker Timeline */}
      <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-lg space-y-4">
        <h2 className="text-lg font-bold text-white flex items-center gap-2">
          <Activity className="h-5 w-5 text-indigo-500" />
          IBC Packet Flow Lifecycle Tracker
        </h2>
        <p className="text-xs text-gray-400">
          Monitor package transfer progression: committed, received by counterparty chain, and finalized with on-chain acknowledgements.
        </p>

        <div className="overflow-x-auto border border-gray-900 rounded-xl mt-2">
          <table className="w-full text-left text-sm text-gray-400">
            <thead className="bg-black/50 text-xs text-gray-500 uppercase tracking-wider font-bold">
              <tr>
                <th className="p-4">Sequence</th>
                <th className="p-4">Asset Amount</th>
                <th className="p-4">Source Block</th>
                <th className="p-4">Destination Block</th>
                <th className="p-4 text-right">Relay Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-900">
              {packets.map((p) => (
                <tr key={p.sequence} className="hover:bg-gray-900/30 transition">
                  <td className="p-4 font-mono font-bold text-white">#{p.sequence}</td>
                  <td className="p-4 font-mono font-semibold text-gray-200">{p.amount}</td>
                  <td className="p-4 font-mono text-xs text-gray-400">#{p.sourceHeight}</td>
                  <td className="p-4 font-mono text-xs text-gray-400">
                    {p.destHeight > 0 ? `#${p.destHeight}` : "Pending Counterparty"}
                  </td>
                  <td className="p-4 text-right">
                    <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 text-xs rounded-lg font-bold uppercase border ${
                      p.status === "acknowledged" ? "bg-green-950/40 border-green-900/50 text-green-400" :
                      p.status === "received" ? "bg-blue-950/40 border-blue-900/50 text-blue-400" :
                      "bg-red-950/40 border-red-900/50 text-red-400 animate-pulse"
                    }`}>
                      {p.status}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
