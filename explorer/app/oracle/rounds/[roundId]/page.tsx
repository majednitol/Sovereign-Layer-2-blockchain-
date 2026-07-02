"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { 
  Scale, Lock, Eye, ArrowLeft, CheckCircle2, 
  XCircle, AlertTriangle, ShieldCheck, HelpCircle 
} from "lucide-react";

interface OracleParticipant {
  validator: string;
  moniker: string;
  commitHash: string;
  revealValue: string;
  salt: string;
  isVerified: boolean;
  timestamp: string;
}

interface RoundDetail {
  roundId: number;
  feedId: string;
  height: number;
  time: string;
  aggregatedMedian: string;
  status: string;
  participants: OracleParticipant[];
}

export default function OracleRoundDetailPage() {
  const params = useParams();
  const roundId = params?.roundId ? Number(params.roundId) : 105;
  const [round, setRound] = useState<RoundDetail | null>(null);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchRound = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/oracle/feeds/slt-usdt/rounds/${roundId}`);
        if (resp.ok) {
          const data = await resp.json();
          setRound({
            roundId: Number(data.roundId || roundId),
            feedId: data.feedId || "slt-usdt",
            height: Number(data.height || 120530),
            time: data.time || new Date().toISOString(),
            aggregatedMedian: data.aggregatedMedian || "1.25",
            status: data.status || "finalized",
            participants: data.participants || [],
          });
        } else {
          throw new Error("Round details not found");
        }
      } catch (err) {
        console.warn("Using simulated round details", err);
        setRound({
          roundId: roundId,
          feedId: "slt-usdt",
          height: 120530,
          time: new Date().toISOString(),
          aggregatedMedian: "1.25",
          status: "finalized",
          participants: [
            {
              validator: "sovereignvaloper1address1",
              moniker: "Sovereign Validator #1",
              commitHash: "4f8a2b9c7d8e1f0a3b2c6d5e9f8a7b6c5d4e3f2a1b0c9d8e7f6a5b4c3d2e1f0a",
              revealValue: "1.24",
              salt: "99283f",
              isVerified: true,
              timestamp: new Date().toISOString(),
            },
            {
              validator: "sovereignvaloper1address2",
              moniker: "Genesis Validator L1",
              commitHash: "d8e3f9a1b0c9d8e7f6a5b4c3d2e1f0a9f8e7d6c5b4a3b2c1d0e9f8a7b6c5d4e3",
              revealValue: "1.26",
              salt: "28fa72",
              isVerified: true,
              timestamp: new Date().toISOString(),
            },
            {
              validator: "sovereignvaloper1address3",
              moniker: "Faulty Validator Node",
              commitHash: "e3a8c2f10b9d8a7c6e5b4d3c2a10f9e8d7c6b5a4a3b2c1d0e9f8a7b6c5d4e3f2",
              revealValue: "0.00",
              salt: "000000",
              isVerified: false,
              timestamp: new Date().toISOString(),
            }
          ],
        });
      } finally {
        setLoading(false);
      }
    };
    fetchRound();
  }, [roundId]);

  if (loading) {
    return (
      <div className="p-6 max-w-7xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading round details...</div>
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
        <span className="text-gray-300">Round #{roundId}</span>
      </nav>

      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between border-b border-gray-800 pb-6 gap-4">
        <div className="space-y-1">
          <div className="flex items-center gap-3">
            <Link href={`/oracle/slt-usdt`} className="p-2 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-gray-400 hover:text-white transition">
              <ArrowLeft className="h-4 w-4" />
            </Link>
            <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
              <Scale className="w-8 h-8 text-blue-500" />
              Consensus Round #{roundId}
            </h1>
          </div>
          <p className="text-xs text-gray-400">Feed ID: <span className="font-mono text-gray-300">{round?.feedId}</span> | Block Height: <span className="font-mono text-gray-350">#{round?.height}</span></p>
        </div>

        <div className="flex items-center gap-3 bg-gray-950 border border-gray-900 p-4 rounded-2xl shadow-lg">
          <div className="text-right">
            <div className="text-xs text-gray-500 uppercase tracking-wider font-bold">Consensus Price</div>
            <div className="text-2xl font-extrabold text-white font-mono">${round?.aggregatedMedian}</div>
          </div>
        </div>
      </div>

      {/* Cryptographic Pre-image Check Log */}
      <div className="bg-gray-950 border border-gray-900 rounded-2xl shadow-xl overflow-hidden">
        <div className="p-5 border-b border-gray-900 flex justify-between items-center bg-gray-900/10">
          <h2 className="text-lg font-bold text-white flex items-center gap-2">
            <ShieldCheck className="h-5 w-5 text-blue-500" />
            Commitment Pre-image Verification Log
          </h2>
          <div className="text-xs text-gray-400">
            Hash check logic: <code className="bg-gray-900 px-1.5 py-0.5 rounded text-blue-400">SHA256(Price + Salt) === CommitHash</code>
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm text-gray-400">
            <thead className="bg-black/50 text-xs text-gray-500 uppercase tracking-wider font-bold">
              <tr>
                <th className="p-4">Validator</th>
                <th className="p-4">Commit Hash</th>
                <th className="p-4">Revealed Value</th>
                <th className="p-4">Salt</th>
                <th className="p-4">Pre-image Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-900">
              {round?.participants.map((p, i) => (
                <tr key={i} className="hover:bg-gray-900/30 transition">
                  <td className="p-4">
                    <div className="space-y-0.5">
                      <div className="font-bold text-white text-xs">{p.moniker}</div>
                      <Link href={`/address/${p.validator}`} className="font-mono text-[10px] text-gray-500 hover:text-blue-400 transition">
                        {p.validator.slice(0, 15)}...{p.validator.slice(-8)}
                      </Link>
                    </div>
                  </td>
                  <td className="p-4 font-mono text-xs text-gray-400">
                    <span title={p.commitHash}>
                      {p.commitHash.slice(0, 16)}...{p.commitHash.slice(-8)}
                    </span>
                  </td>
                  <td className="p-4 font-mono text-sm text-white font-semibold">
                    {p.revealValue !== "0.00" ? `$${p.revealValue}` : "N/A"}
                  </td>
                  <td className="p-4 font-mono text-xs text-gray-500">
                    {p.salt}
                  </td>
                  <td className="p-4">
                    {p.isVerified ? (
                      <span className="inline-flex items-center gap-1 px-2.5 py-1 text-xs bg-green-950 text-green-400 border border-green-900 rounded-lg font-bold">
                        <CheckCircle2 className="h-3.5 w-3.5" /> Verified Match
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1 px-2.5 py-1 text-xs bg-red-950 text-red-400 border border-red-900 rounded-lg font-bold">
                        <XCircle className="h-3.5 w-3.5" /> Hash Mismatch
                      </span>
                    )}
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
