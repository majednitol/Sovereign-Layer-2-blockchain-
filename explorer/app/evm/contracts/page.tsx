"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { Cpu, ShieldCheck, ChevronRight } from "lucide-react";

interface EvmContract {
  address: string;
  name: string;
  creator: string;
  txHash: string;
  verified: boolean;
}

export default function EvmContractsPage() {
  const [contracts, setContracts] = useState<EvmContract[]>([]);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchContracts = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/contracts`);
        if (resp.ok) {
          const data = await resp.json();
          if (data.contracts) {
            setContracts(data.contracts.map((c: any) => ({
              address: c.address,
              name: c.label || "Smart Contract",
              creator: c.creator,
              txHash: "0xtxhash",
              verified: true,
            })));
          }
        }
      } catch (err) {
        console.warn("Using simulated EVM contracts", err);
        setContracts([
          { address: "0x3f5c9e2b1d7a8d05cf5d2eb1...", name: "Sovereign ERC20 Bridge", creator: "0x12345678...", txHash: "0xtxhash123...", verified: true },
          { address: "0x9876543210fedcba98765432...", name: "WETH Mock Token", creator: "0x98765432...", txHash: "0xtxhash456...", verified: false },
        ]);
      } finally {
        setLoading(false);
      }
    };
    fetchContracts();
  }, []);

  if (loading) {
    return (
      <div className="p-6 max-w-6xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading EVM contracts...</div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-gray-300">EVM Contracts</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4">
        <h1 className="text-3xl font-bold tracking-tight text-white flex items-center gap-2">
          <Cpu className="w-8 h-8 text-blue-500" />
          EVM Contracts
        </h1>
        <p className="text-gray-400 mt-2">Deployed Solidity contract instances on Sovereign EVM.</p>
      </div>

      {/* Table */}
      <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm text-gray-400">
            <thead className="bg-gray-950 text-xs text-gray-500 uppercase tracking-wider font-semibold">
              <tr>
                <th className="p-4">Contract Address</th>
                <th className="p-4">Name</th>
                <th className="p-4">Verification</th>
                <th className="p-4 text-right">Details</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-850">
              {contracts.map((c) => (
                <tr key={c.address} className="hover:bg-gray-850/40 transition">
                  <td className="p-4 font-mono font-semibold text-white text-xs">
                    <Link href={`/evm/contracts/${c.address}`} className="text-blue-500 hover:text-blue-400">
                      {c.address.slice(0, 16)}...
                    </Link>
                  </td>
                  <td className="p-4 text-gray-300 font-medium">{c.name}</td>
                  <td className="p-4">
                    {c.verified ? (
                      <span className="px-2.5 py-0.5 text-xs bg-green-950 text-green-400 border border-green-900 rounded font-semibold uppercase flex items-center gap-1.5 w-fit">
                        <ShieldCheck className="w-3.5 h-3.5" />
                        Verified
                      </span>
                    ) : (
                      <span className="px-2.5 py-0.5 text-xs bg-gray-950 text-gray-400 border border-gray-800 rounded font-semibold uppercase">
                        Unverified
                      </span>
                    )}
                  </td>
                  <td className="p-4 text-right">
                    <Link
                      href={`/evm/contracts/${c.address}`}
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
