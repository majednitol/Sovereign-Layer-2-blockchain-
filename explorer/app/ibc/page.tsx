"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { ArrowLeftRight, Cpu, Layers, Activity, CheckCircle2, ChevronRight } from "lucide-react";

interface IbcChannel {
  channelId: string;
  portId: string;
  counterpartyChannelId: string;
  counterpartyPortId: string;
  state: string;
  ordering: string;
  packetCount: number;
}

export default function IBCPage() {
  const [channels, setChannels] = useState<IbcChannel[]>([]);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchChannels = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/ibc/channels`);
        if (resp.ok) {
          const data = await resp.json();
          if (data.channels) {
            setChannels(data.channels.map((c: any) => ({
              channelId: c.channelId,
              portId: c.portId,
              counterpartyChannelId: c.counterpartyChannelId,
              counterpartyPortId: c.counterpartyPortId,
              state: c.state,
              ordering: c.ordering,
              packetCount: Number(c.packetCount || 0),
            })));
          }
        }
      } catch (err) {
        console.warn("Using simulated IBC channels", err);
        setChannels([
          { channelId: "channel-0", portId: "transfer", counterpartyChannelId: "channel-14", counterpartyPortId: "transfer", state: "STATE_OPEN", ordering: "ORDER_UNORDERED", packetCount: 150 },
        ]);
      } finally {
        setLoading(false);
      }
    };
    fetchChannels();
  }, []);

  if (loading) {
    return (
      <div className="p-6 max-w-6xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading IBC dashboard...</div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-gray-300">IBC Portals</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4">
        <h1 className="text-3xl font-bold tracking-tight text-white flex items-center gap-2">
          <ArrowLeftRight className="w-8 h-8 text-blue-500 animate-pulse" />
          Inter-Blockchain Communication (IBC)
        </h1>
        <p className="text-gray-400 mt-2">Active client states, channel connections, and packet flow trackers.</p>
      </div>

      {/* Navigation shortcuts */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
        <Link href="/ibc/channels" className="bg-gray-900 border border-gray-850 p-5 rounded-xl hover:border-gray-700 transition space-y-2">
          <div className="text-xs text-gray-400 uppercase tracking-wider font-semibold">Channels</div>
          <div className="text-2xl font-bold text-white flex items-center justify-between">
            <span>{channels.length} Open Channels</span>
            <ChevronRight className="w-5 h-5 text-gray-500" />
          </div>
        </Link>
        <Link href="/ibc/assets" className="bg-gray-900 border border-gray-850 p-5 rounded-xl hover:border-gray-700 transition space-y-2">
          <div className="text-xs text-gray-400 uppercase tracking-wider font-semibold">IBC Denoms</div>
          <div className="text-2xl font-bold text-white flex items-center justify-between">
            <span>Asset Directory</span>
            <ChevronRight className="w-5 h-5 text-gray-500" />
          </div>
        </Link>
      </div>
    </div>
  );
}
