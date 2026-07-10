"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { ArrowLeft, Users, Trophy, DollarSign, Activity } from "lucide-react";

interface AccountItem {
  addressBech32: string;
  addressHex: string;
  balance: string;
  txCount: number;
}

export default function AccountsPage() {
  const [accounts, setAccounts] = useState<AccountItem[]>([]);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchTopAccounts = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/top-accounts`);
        if (resp.ok) {
          const data = await resp.json();
          if (data.accounts) {
            setAccounts(data.accounts);
          }
        }
      } catch (err) {
        console.warn("Failed to fetch top accounts, using fallback mock.", err);
        setAccounts([
          { addressBech32: "cosmos13f5c9e2b1d7a8d9e8a7b6c5d4e3f281f449219d5", addressHex: "0x3f5c9e2b1d7a8d9e8a7b6c5d4e3f281f449219d5", balance: "1,200,450.00 CSOV", txCount: 154 },
          { addressBech32: "cosmos18a7b6c5d4e3f281f449219d54e47fd8ad83861b", addressHex: "0x8a7b6c5d4e3f281f449219d54e47fd8ad83861b46", balance: "945,100.22 CSOV", txCount: 89 },
          { addressBech32: "cosmos10bech32addressmock1234567890abcdefghijk", addressHex: "0x0bech32addressmock1234567890abcdefghijk", balance: "420,000.00 CSOV", txCount: 42 }
        ]);
      } finally {
        setLoading(false);
      }
    };
    fetchTopAccounts();
  }, []);

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white font-medium">Top Accounts</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-900 pb-4 flex items-center space-x-3">
        <Link href="/" className="p-2 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-gray-400 hover:text-white transition">
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
            <Trophy className="text-yellow-500 w-8 h-8 animate-pulse" />
            Top Rich List Accounts
          </h1>
          <p className="text-gray-400 mt-1">Registry of highest balance accounts on the Sovereign L1 chain.</p>
        </div>
      </div>

      {loading ? (
        <div className="py-20 text-center text-gray-400">Loading accounts richness leaderboard...</div>
      ) : (
        <div className="bg-gray-950 border border-gray-900 rounded-2xl overflow-hidden shadow-lg">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm text-gray-400">
              <thead className="bg-black/50 text-xs text-gray-500 uppercase tracking-wider font-bold">
                <tr>
                  <th className="p-4 w-16">Rank</th>
                  <th className="p-4">Account Address</th>
                  <th className="p-4">EVM Mirror Address</th>
                  <th className="p-4">Available Balance</th>
                  <th className="p-4 text-right">Transactions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-900">
                {accounts.map((acc, index) => (
                  <tr key={acc.addressBech32} className="hover:bg-gray-900/30 transition">
                    <td className="p-4 font-bold text-gray-500">#{index + 1}</td>
                    <td className="p-4 font-mono text-xs text-blue-500 hover:underline">
                      <Link href={`/address/${acc.addressBech32}`}>{acc.addressBech32}</Link>
                    </td>
                    <td className="p-4 font-mono text-xs text-gray-400">
                      {acc.addressHex ? acc.addressHex.slice(0, 16) + "..." : "No EVM address"}
                    </td>
                    <td className="p-4 font-mono font-bold text-green-400">{acc.balance}</td>
                    <td className="p-4 text-right font-mono font-semibold text-gray-300">{acc.txCount}</td>
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
