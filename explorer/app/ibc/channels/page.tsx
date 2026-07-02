"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { ArrowLeftRight, Activity, ChevronRight } from "lucide-react";

interface IbcChannel {
  channelId: string;
  portId: string;
  counterpartyChannelId: string;
  counterpartyPortId: string;
  state: string;
  ordering: string;
  packetCount: number;
}

export default function IbcChannelsPage() {
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
        <div className="text-gray-400">Loading IBC channels...</div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/ibc" className="hover:text-white transition">IBC</Link>
        <span>/</span>
        <span className="text-gray-300">Channels</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4">
        <h1 className="text-3xl font-bold tracking-tight text-white flex items-center gap-2">
          <ArrowLeftRight className="w-8 h-8 text-blue-500" />
          IBC Channels
        </h1>
      </div>

      {/* Table */}
      <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm text-gray-400">
            <thead className="bg-gray-950 text-xs text-gray-500 uppercase tracking-wider">
              <tr>
                <th className="p-4">Channel ID</th>
                <th className="p-4">Port ID</th>
                <th className="p-4">Counterparty Channel</th>
                <th className="p-4">Counterparty Port</th>
                <th className="p-4">Status</th>
                <th className="p-4 text-right">Details</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-850">
              {channels.map((c) => (
                <tr key={c.channelId} className="hover:bg-gray-850/40 transition">
                  <td className="p-4 font-mono font-semibold text-white">{c.channelId}</td>
                  <td className="p-4 font-mono text-gray-300">{c.portId}</td>
                  <td className="p-4 font-mono text-gray-300">{c.counterpartyChannelId}</td>
                  <td className="p-4 font-mono text-gray-500">{c.counterpartyPortId}</td>
                  <td className="p-4">
                    <span className="px-2.5 py-0.5 text-xs bg-green-950 text-green-400 border border-green-900 rounded font-semibold uppercase">
                      {c.state}
                    </span>
                  </td>
                  <td className="p-4 text-right">
                    <Link
                      href={`/ibc/channels/${c.channelId}`}
                      className="p-2 bg-gray-850 hover:bg-gray-800 rounded-lg text-gray-400 hover:text-white transition inline-block"
                    >
                      <ChevronRight className="w-4 h-4" />
                    </Link>
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
