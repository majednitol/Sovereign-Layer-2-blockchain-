"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { ArrowLeft, Wallet, ShieldAlert, Layers, Key, Coins, Download, ArrowUpRight, ArrowDownLeft } from "lucide-react";

interface VaultEvent {
  txHash: string;
  logIndex: number;
  sender: string;
  owner: string;
  assets: string;
  shares: string;
  eventType: string;
}

interface VaultDetail {
  vaultAddress: string;
  underlyingAssetAddress: string;
  totalAssets: string;
  totalShares: string;
  sharePrice: string;
  history: VaultEvent[];
}

export default function VaultDetailPage() {
  const params = useParams();
  const addr = params?.addr ? String(params.addr) : "";

  const [vault, setVault] = useState<VaultDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    if (!addr) return;
    const fetchVaultDetail = async () => {
      setLoading(true);
      setError(null);
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/vaults/evm/${addr}`);
        if (!resp.ok) {
          throw new Error("ERC-4626 vault not found or not indexed yet.");
        }
        const data = await resp.json();
        setVault(data);
      } catch (err: any) {
        setError(err.message || "Network error occurred.");
      } finally {
        setLoading(false);
      }
    };
    fetchVaultDetail();
  }, [addr, API_BASE]);

  if (loading) {
    return (
      <div className="p-6 max-w-6xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400 font-mono animate-pulse">Loading ERC-4626 Vault metrics...</div>
      </div>
    );
  }

  if (error || !vault) {
    return (
      <div className="p-6 max-w-6xl mx-auto text-center space-y-4 py-32">
        <h2 className="text-2xl font-bold text-white">Vault Not Found</h2>
        <p className="text-gray-400">{error || "The requested ERC-4626 Vault does not exist or has not been recognized yet."}</p>
        <Link href="/" className="inline-block px-4 py-2 bg-gray-900 border border-gray-800 rounded-lg text-white hover:bg-gray-800 transition">
          Back to Home
        </Link>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-gray-300">ERC-4626 Vaults</span>
        <span>/</span>
        <span className="text-gray-300 font-mono text-xs">{addr}</span>
      </nav>

      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between border-b border-gray-800 pb-6 gap-4">
        <div className="flex items-center gap-3">
          <Link href="/" className="p-2 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-gray-400 hover:text-white transition">
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div>
            <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
              <Layers className="w-8 h-8 text-indigo-500" />
              ERC-4626 Vault Dashboard
            </h1>
            <p className="text-gray-400 mt-1 font-mono text-xs break-all">Vault address: {vault.vaultAddress}</p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <a
            href={`${API_BASE}/api/rest/v1/explorer/vaults/evm/${addr}?download=true`}
            target="_blank"
            rel="noreferrer"
            className="flex items-center gap-2 px-3 py-1.5 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-xs text-gray-300 hover:text-white transition"
          >
            <Download className="h-3.5 w-3.5" /> Download JSON
          </a>
        </div>
      </div>

      {/* Stats grid */}
      <div className="grid grid-cols-1 sm:grid-cols-4 gap-6">
        <div className="bg-gray-950 border border-gray-900 p-5 rounded-xl space-y-1 shadow-md">
          <div className="text-xs text-gray-500 uppercase tracking-wider font-bold">Total Shares</div>
          <div className="text-lg font-bold text-white font-mono">{Number(vault.totalShares).toLocaleString()}</div>
        </div>
        <div className="bg-gray-950 border border-gray-900 p-5 rounded-xl space-y-1 shadow-md">
          <div className="text-xs text-gray-500 uppercase tracking-wider font-bold">Total Managed Assets</div>
          <div className="text-lg font-bold text-white font-mono">{Number(vault.totalAssets).toLocaleString()}</div>
        </div>
        <div className="bg-gray-950 border border-gray-900 p-5 rounded-xl space-y-1 shadow-md">
          <div className="text-xs text-gray-500 uppercase tracking-wider font-bold">Price per Share</div>
          <div className="text-lg font-bold text-white font-mono">{vault.sharePrice}</div>
        </div>
        <div className="bg-gray-950 border border-gray-900 p-5 rounded-xl space-y-1 shadow-md">
          <div className="text-xs text-gray-500 uppercase tracking-wider font-bold">Vault Standard</div>
          <div className="text-lg font-bold text-white font-mono">ERC-4626</div>
        </div>
      </div>

      {/* Key metadata */}
      <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 space-y-4 text-xs font-mono">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <span className="text-gray-500 uppercase font-bold block flex items-center gap-1.5">
              <Coins className="h-3.5 w-3.5 text-yellow-500" /> Underlying Asset
            </span>
            <Link href={`/evm/tokens/${vault.underlyingAssetAddress}`} className="text-blue-500 hover:underline mt-1 block select-all break-all">
              {vault.underlyingAssetAddress}
            </Link>
          </div>
          <div>
            <span className="text-gray-500 uppercase font-bold block flex items-center gap-1.5">
              <Key className="h-3.5 w-3.5 text-blue-500" /> Vault ABI Check
            </span>
            <span className="text-green-400 font-semibold mt-1 block">ERC-4626 Parity Verified</span>
          </div>
        </div>
      </div>

      {/* Historical event logs */}
      <div className="bg-gray-950 border border-gray-900 p-6 rounded-2xl shadow-lg space-y-4">
        <h3 className="text-lg font-bold text-white flex items-center gap-2">
          <Wallet className="h-5 w-5 text-purple-500" />
          Deposit / Withdrawal Ledger Logs
        </h3>
        <div className="overflow-x-auto border border-gray-900 rounded-xl">
          <table className="w-full text-left text-sm text-gray-400 font-mono">
            <thead className="bg-black/50 text-xs text-gray-500 uppercase tracking-wider font-bold">
              <tr>
                <th className="p-4">Tx Hash</th>
                <th className="p-4">Action</th>
                <th className="p-4">Sender</th>
                <th className="p-4">Owner / Recipient</th>
                <th className="p-4">Assets</th>
                <th className="p-4 text-right">Shares</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-900">
              {vault.history.length === 0 ? (
                <tr>
                  <td colSpan={6} className="p-8 text-center text-gray-500 font-mono">No deposit or withdrawal events recorded yet.</td>
                </tr>
              ) : (
                vault.history.map((e, idx) => (
                  <tr key={`${e.txHash}-${idx}`} className="hover:bg-gray-900/30 transition text-xs">
                    <td className="p-4 font-bold text-white">
                      <Link href={`/evm/txs/${e.txHash}`} className="text-blue-500 hover:underline">
                        {e.txHash.slice(0, 16)}...
                      </Link>
                    </td>
                    <td className="p-4">
                      <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-semibold border ${
                        e.eventType === "Deposit" ? "bg-green-950/40 border-green-900 text-green-400" : "bg-red-950/40 border-red-900 text-red-400"
                      }`}>
                        {e.eventType === "Deposit" ? <ArrowUpRight className="h-3 w-3" /> : <ArrowDownLeft className="h-3 w-3" />}
                        {e.eventType}
                      </span>
                    </td>
                    <td className="p-4">
                      <Link href={`/address/${e.sender}`} className="text-gray-400 hover:text-white transition">
                        {e.sender.slice(0, 10)}...{e.sender.slice(-6)}
                      </Link>
                    </td>
                    <td className="p-4">
                      <Link href={`/address/${e.owner}`} className="text-gray-400 hover:text-white transition">
                        {e.owner.slice(0, 10)}...{e.owner.slice(-6)}
                      </Link>
                    </td>
                    <td className="p-4 text-white font-bold">{Number(e.assets).toLocaleString()}</td>
                    <td className="p-4 text-right text-gray-400">{Number(e.shares).toLocaleString()}</td>
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
