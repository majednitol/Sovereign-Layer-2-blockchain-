"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { ArrowLeft, CheckCircle2, ShieldAlert, Cpu, Layers } from "lucide-react";

interface VerifiedContract {
  address: string;
  name: string;
  type: "EVM" | "CosmWasm";
  compiler: string;
  dateVerified: string;
  isProxy: boolean;
  isVault: boolean;
}

export default function VerifiedContractsPage() {
  const [contracts, setContracts] = useState<VerifiedContract[]>([]);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchVerifiedContracts = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/contracts/verified`);
        if (resp.ok) {
          const data = await resp.json();
          if (data.contracts) {
            setContracts(data.contracts);
          }
        }
      } catch (err) {
        console.warn("Failed to fetch verified contracts, using fallback mock.", err);
        setContracts([
          { address: "0x1234567890123456789012345678901234567890", name: "SovereignBridgeBox", type: "EVM", compiler: "v0.8.24", dateVerified: new Date().toISOString(), isProxy: true, isVault: false },
          { address: "sov13f5c9e2b1d7a8d9e8a7b6c5d4e3f281f449219d5", name: "cw20_base_token", type: "CosmWasm", compiler: "cosmwasm-v1.4", dateVerified: new Date(Date.now() - 86400000).toISOString(), isProxy: false, isVault: false },
          { address: "0x5a109a25b2a0c7cfd21c0e3a6c57f722971239aa", name: "YieldVaultSOV", type: "EVM", compiler: "v0.8.24", dateVerified: new Date(Date.now() - 172800000).toISOString(), isProxy: false, isVault: true }
        ]);
      } finally {
        setLoading(false);
      }
    };
    fetchVerifiedContracts();
  }, []);

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white font-medium">Verified Contracts</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-900 pb-4 flex items-center space-x-3">
        <Link href="/" className="p-2 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-gray-400 hover:text-white transition">
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
            <CheckCircle2 className="text-green-500 w-8 h-8 animate-pulse" />
            Verified Smart Contracts
          </h1>
          <p className="text-gray-400 mt-1">Directory of source code verified EVM & CosmWasm smart contracts.</p>
        </div>
      </div>

      {loading ? (
        <div className="py-20 text-center text-gray-400">Loading verified contracts catalog...</div>
      ) : (
        <div className="bg-gray-950 border border-gray-900 rounded-2xl overflow-hidden shadow-lg">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm text-gray-400">
              <thead className="bg-black/50 text-xs text-gray-500 uppercase tracking-wider font-bold">
                <tr>
                  <th className="p-4">Contract Address</th>
                  <th className="p-4">Name / ID</th>
                  <th className="p-4">Runtime Engine</th>
                  <th className="p-4">Compiler</th>
                  <th className="p-4">Features</th>
                  <th className="p-4 text-right">Verification Date</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-900">
                {contracts.map((c) => (
                  <tr key={c.address} className="hover:bg-gray-900/30 transition">
                    <td className="p-4 font-mono text-xs text-blue-500 hover:underline">
                      <Link href={`/contracts/${c.address}`}>{c.address}</Link>
                    </td>
                    <td className="p-4 font-semibold text-white">{c.name}</td>
                    <td className="p-4">
                      <span className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                        c.type === "EVM" ? "bg-orange-950 text-orange-400 border border-orange-900" : "bg-purple-950 text-purple-400 border border-purple-900"
                      }`}>
                        {c.type}
                      </span>
                    </td>
                    <td className="p-4 font-mono text-xs">{c.compiler}</td>
                    <td className="p-4 space-x-1">
                      {c.isProxy && (
                        <span className="px-2 py-0.5 bg-blue-950/40 border border-blue-900 text-blue-400 rounded text-[9px] font-bold uppercase">
                          Proxy
                        </span>
                      )}
                      {c.isVault && (
                        <span className="px-2 py-0.5 bg-green-950/40 border border-green-900 text-green-400 rounded text-[9px] font-bold uppercase">
                          ERC-4626 Vault
                        </span>
                      )}
                      {!c.isProxy && !c.isVault && <span className="text-gray-600 font-mono text-[10px]">—</span>}
                    </td>
                    <td className="p-4 text-right text-xs text-gray-500">
                      {new Date(c.dateVerified).toLocaleDateString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
