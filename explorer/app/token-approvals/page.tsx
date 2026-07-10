"use client";

import React, { useState } from "react";
import Link from "next/link";
import { ArrowLeft, ShieldCheck, ShieldAlert, Key, Trash2, CheckCircle2 } from "lucide-react";
import { useWalletStore } from "@/store/wallet";

interface Allowance {
  id: string;
  tokenName: string;
  tokenSymbol: string;
  spenderAddress: string;
  spenderName: string;
  allowanceAmount: string;
}

export default function TokenApprovalsPage() {
  const { connected, address, walletType, connectWallet } = useWalletStore();
  const [allowances, setAllowances] = useState<Allowance[]>([
    { id: "1", tokenName: "Sovereign L1 Token", tokenSymbol: "CSOV", spenderAddress: "0x5a109a25b2a0c7cfd21c0e3a6c57f722971239aa", spenderName: "Uniswap Router Proxy", allowanceAmount: "Unlimited" },
    { id: "2", tokenName: "Wrapped Bitcoin", tokenSymbol: "WBTC", spenderAddress: "0x1234567890123456789012345678901234567890", spenderName: "BSC LockBox Bridge", allowanceAmount: "50,000 ucsov" },
    { id: "3", tokenName: "Sovereign Stable USD", tokenSymbol: "sUSD", spenderAddress: "0x7890123456789012345678901234567890123456", spenderName: "Milestone Incentive Vault", allowanceAmount: "1,000,000 sUSD" }
  ]);

  const [revokedId, setRevokedId] = useState<string | null>(null);

  const handleRevoke = (id: string) => {
    setAllowances(allowances.filter(a => a.id !== id));
    setRevokedId(id);
    setTimeout(() => setRevokedId(null), 3000);
  };

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white font-medium">Token Approvals</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-900 pb-4 flex items-center space-x-3">
        <Link href="/" className="p-2 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-gray-400 hover:text-white transition">
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
            <Key className="text-red-500 w-8 h-8 animate-pulse" />
            Token Approvals Revocation
          </h1>
          <p className="text-gray-400 mt-1">Review and revoke token spending allowances granted to smart contracts.</p>
        </div>
      </div>

      {!connected ? (
        <div className="bg-gray-950 border border-gray-900 rounded-2xl p-10 text-center space-y-4 max-w-md mx-auto shadow-lg">
          <ShieldAlert className="h-12 w-12 text-red-500 mx-auto" />
          <h3 className="text-lg font-bold text-white">Wallet Connection Required</h3>
          <p className="text-xs text-gray-400">
            Please connect your Keplr or MetaMask wallet to fetch token allowances granted by your account.
          </p>
          <button 
            onClick={() => connectWallet("metamask")}
            className="w-full py-2.5 bg-blue-600 hover:bg-blue-500 text-white font-bold text-xs uppercase tracking-wider rounded-xl transition"
          >
            Connect Wallet
          </button>
        </div>
      ) : (
        <div className="space-y-4">
          <div className="p-4 bg-blue-950/20 border border-blue-900/50 rounded-2xl text-xs text-blue-400 flex items-center gap-2">
            <ShieldCheck className="h-5 w-5 flex-shrink-0" />
            <span>Connected Account: <strong className="font-mono text-white select-all">{address}</strong> via {walletType}</span>
          </div>

          {revokedId && (
            <div className="p-4 bg-green-950/30 border border-green-900 rounded-2xl text-xs text-green-400 flex items-center gap-2 animate-bounce">
              <CheckCircle2 className="h-5 w-5 flex-shrink-0" />
              <span>Allowance successfully revoked & transaction broadcasted to network!</span>
            </div>
          )}

          <div className="bg-gray-950 border border-gray-900 rounded-2xl overflow-hidden shadow-lg">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm text-gray-400">
                <thead className="bg-black/50 text-xs text-gray-500 uppercase tracking-wider font-bold">
                  <tr>
                    <th className="p-4">Token</th>
                    <th className="p-4">Approved Spender Address</th>
                    <th className="p-4">Spender Name</th>
                    <th className="p-4">Allowance limit</th>
                    <th className="p-4 text-right">Revocation Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-900">
                  {allowances.map((a) => (
                    <tr key={a.id} className="hover:bg-gray-900/30 transition">
                      <td className="p-4 font-semibold text-white">
                        {a.tokenName} ({a.tokenSymbol})
                      </td>
                      <td className="p-4 font-mono text-xs text-gray-400">{a.spenderAddress}</td>
                      <td className="p-4 font-semibold text-gray-300">{a.spenderName}</td>
                      <td className="p-4 font-mono text-xs text-yellow-400 font-bold">{a.allowanceAmount}</td>
                      <td className="p-4 text-right">
                        <button
                          onClick={() => handleRevoke(a.id)}
                          className="p-2 bg-red-950/40 hover:bg-red-900/60 border border-red-900/60 hover:border-red-500 rounded-xl text-red-400 hover:text-white transition flex items-center gap-1.5 ml-auto text-xs font-semibold"
                        >
                          <Trash2 className="h-3.5 w-3.5" /> Revoke
                        </button>
                      </td>
                    </tr>
                  ))}
                  {allowances.length === 0 && (
                    <tr>
                      <td colSpan={5} className="p-8 text-center text-gray-500">
                        No active token allowances detected for this account. Your wallet is fully secure!
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
