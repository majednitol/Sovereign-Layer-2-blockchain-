"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { Code2, ChevronRight, FileCode } from "lucide-react";

interface CodeDetail {
  codeId: number;
  uploader: string;
  height: number;
  checksum: string;
  instantiationCount: number;
  txHash: string;
}

export default function CodesPage() {
  const [codes, setCodes] = useState<CodeDetail[]>([]);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchCodes = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/codes`);
        if (resp.ok) {
          const data = await resp.json();
          if (data.codes) {
            setCodes(data.codes.map((c: any) => ({
              codeId: Number(c.codeId),
              uploader: c.uploader,
              height: Number(c.height),
              checksum: c.checksum,
              instantiationCount: Number(c.instantiationCount || 0),
              txHash: c.txHash,
            })));
          }
        }
      } catch (err) {
        console.warn("Failed to load codes list", err);
        setCodes([]);
      } finally {
        setLoading(false);
      }
    };
    fetchCodes();
  }, []);

  if (loading) {
    return (
      <div className="p-6 max-w-6xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading CosmWasm codes...</div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-gray-300">CosmWasm Codes</span>
      </nav>

      {/* Header */}
      <div className="flex justify-between items-center border-b border-gray-800 pb-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-white flex items-center gap-2">
            <FileCode className="w-8 h-8 text-blue-500" />
            CosmWasm Codes Registry
          </h1>
          <p className="text-gray-400 mt-2">Registered WebAssembly codes uploaded to Sovereign L1.</p>
        </div>
        <Link
          href="/codes/submit"
          className="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white rounded-lg font-medium transition text-sm"
        >
          Submit Code Schema
        </Link>
      </div>

      {/* Table */}
      <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm text-gray-400">
            <thead className="bg-gray-950 text-xs text-gray-500 uppercase tracking-wider">
              <tr>
                <th className="p-4">Code ID</th>
                <th className="p-4">Uploader</th>
                <th className="p-4">Height</th>
                <th className="p-4">Instances</th>
                <th className="p-4 text-right">Details</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-850">
              {codes.map((c) => (
                <tr key={c.codeId} className="hover:bg-gray-850/40 transition">
                  <td className="p-4 font-mono font-semibold text-white">#{c.codeId}</td>
                  <td className="p-4 font-mono text-xs text-gray-500">{c.uploader.slice(0, 15)}...</td>
                  <td className="p-4 font-mono">#{c.height}</td>
                  <td className="p-4 font-semibold text-blue-400">{c.instantiationCount} active</td>
                  <td className="p-4 text-right">
                    <Link
                      href={`/codes/${c.codeId}`}
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
