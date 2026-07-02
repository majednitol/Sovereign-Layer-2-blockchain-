"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { Code, Users, Cpu, FileJson } from "lucide-react";

interface Contract {
  address: string;
  codeId: number;
  label: string;
  creator: string;
  admin: string;
  typeBadge: string;
}

export default function ContractsIndexPage() {
  const [contracts, setContracts] = useState<Contract[]>([]);
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
              codeId: Number(c.codeId),
              label: c.label,
              creator: c.creator,
              admin: c.admin,
              typeBadge: c.typeBadge,
            })));
          }
        }
      } catch (err) {
        console.warn("Using simulated contracts list", err);
        setContracts([
          {
            address: "sovereign1contract120530",
            codeId: 1,
            label: "Sovereign L1 Governance Token",
            creator: "sovereign1address0",
            admin: "",
            typeBadge: "CW-20",
          },
          {
            address: "sovereign1contract120540",
            codeId: 2,
            label: "Sovereign L1 Founders Badge",
            creator: "sovereign1address0",
            admin: "sovereign1address0",
            typeBadge: "CW-721",
          },
        ]);
      } finally {
        setLoading(false);
      }
    };
    fetchContracts();
  }, []);

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white">Contracts</span>
      </nav>

      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight text-white">Smart Contracts</h1>
        <p className="text-gray-400 mt-1">
          Instantiated CosmWasm smart contracts. Token standards are auto-detected.
        </p>
      </div>

      {/* Contracts Table */}
      <div className="bg-gray-950 border border-gray-900 rounded-xl overflow-hidden shadow-lg">
        <div className="px-6 py-4 border-b border-gray-900 flex justify-between items-center">
          <h3 className="text-lg font-bold text-white flex items-center space-x-2">
            <Code className="text-blue-500 h-5 w-5" />
            <span>Instantiated Contracts</span>
          </h3>
          <span className="text-xs px-2.5 py-1 bg-gray-900 border border-gray-800 rounded font-semibold text-gray-400 font-mono">
            {contracts.length} Total
          </span>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="bg-black/50 text-gray-400 uppercase text-xs">
              <tr>
                <th className="px-6 py-3">Contract Address</th>
                <th className="px-6 py-3">Label</th>
                <th className="px-6 py-3">Code ID</th>
                <th className="px-6 py-3">Creator</th>
                <th className="px-6 py-3">Type Standard</th>
                <th className="px-6 py-3">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-900">
              {loading ? (
                <tr>
                  <td colSpan={6} className="px-6 py-12 text-center text-gray-500">Loading contracts...</td>
                </tr>
              ) : (
                contracts.map((c) => (
                  <tr key={c.address} className="hover:bg-gray-900/30 transition">
                    <td className="px-6 py-4 font-mono text-xs text-blue-400">
                      <Link href={`/contracts/${c.address}`} className="hover:underline">
                        {c.address}
                      </Link>
                    </td>
                    <td className="px-6 py-4 font-medium text-white">{c.label}</td>
                    <td className="px-6 py-4 font-mono text-xs text-gray-400">{c.codeId}</td>
                    <td className="px-6 py-4 font-mono text-xs text-gray-400">
                      {c.creator.slice(0, 10)}...{c.creator.slice(-6)}
                    </td>
                    <td className="px-6 py-4">
                      <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold uppercase border ${
                        c.typeBadge === "CW-20" 
                          ? "bg-purple-950/50 text-purple-400 border-purple-900" 
                          : c.typeBadge === "CW-721"
                          ? "bg-green-950/50 text-green-400 border-green-900"
                          : "bg-blue-950/50 text-blue-400 border-blue-900"
                      }`}>
                        {c.typeBadge}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <Link
                        href={`/contracts/${c.address}`}
                        className="text-xs text-blue-500 hover:text-blue-400 font-semibold"
                      >
                        Interact
                      </Link>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
