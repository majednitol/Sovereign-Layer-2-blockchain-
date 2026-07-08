"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { Vote, FileText, CheckCircle2, XCircle, AlertCircle, RefreshCw, Plus, Users, Landmark } from "lucide-react";
import { useWalletStore } from "@/store/wallet";

interface Proposal {
  id: string;
  status: string;
  title: string;
  typeBadge: string;
  description: string;
  submitTime: string;
  depositEndTime: string;
  votingStartTime: string;
  votingEndTime: string;
  tallyResult: string; // JSON string
  constitutionCheckPassed: boolean;
}

export default function GOVERNANCEPage() {
  const { walletType, connected, address, connectWallet, disconnectWallet } = useWalletStore();
  const [proposals, setProposals] = useState<Proposal[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<"all" | "voting" | "passed" | "rejected">("all");

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  const fetchProposals = async () => {
    setLoading(true);
    try {
      const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/governance/proposals`);
      if (resp.ok) {
        const data = await resp.json();
        if (data.proposals) {
          setProposals(data.proposals);
        }
      }
    } catch (err) {
      console.error("Failed to fetch proposals", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProposals();
  }, []);

  const getFilteredProposals = () => {
    let filtered = proposals;
    if (activeTab !== "all") {
      filtered = proposals.filter((p) => p.status.toLowerCase() === activeTab);
    }
    return filtered;
  };

  const parseTally = (tallyResultStr: string) => {
    try {
      const tally = JSON.parse(tallyResultStr || "{}");
      const yes = Number(tally.yes || 0);
      const no = Number(tally.no || 0);
      const abstain = Number(tally.abstain || 0);
      const total = yes + no + abstain;

      if (total === 0) return { yes: 0, no: 0, abstain: 0, yesPct: 0, noPct: 0, abstainPct: 0, total: 0 };

      return {
        yes,
        no,
        abstain,
        yesPct: (yes / total) * 100,
        noPct: (no / total) * 100,
        abstainPct: (abstain / total) * 100,
        total,
      };
    } catch (e) {
      return { yes: 0, no: 0, abstain: 0, yesPct: 0, noPct: 0, abstainPct: 0, total: 0 };
    }
  };

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white">Governance</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4 flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
        <div className="flex items-center space-x-3">
          <Landmark className="text-blue-500 h-8 w-8" />
          <div>
            <h1 className="text-3xl font-bold tracking-tight text-white">Governance proposals</h1>
            <p className="text-gray-400 mt-1">Vote on proposals, parameters, software upgrades and review constitutionality</p>
          </div>
        </div>

        <div className="flex items-center space-x-3 self-stretch md:self-auto justify-between">
          <button 
            onClick={fetchProposals}
            className="p-2 bg-gray-900 hover:bg-gray-800 border border-gray-800 rounded-lg text-gray-400 hover:text-white transition"
            title="Reload"
          >
            <RefreshCw className="h-4 w-4" />
          </button>
          
          <Link 
            href="/governance/submit"
            className="flex items-center space-x-2 px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white rounded-lg font-medium transition shadow-lg shadow-blue-900/20 text-sm"
          >
            <Plus className="h-4 w-4" />
            <span>New proposal</span>
          </Link>

          {/* Wallet Connect Panel */}
          <div className="flex items-center space-x-2 bg-gray-950 border border-gray-900 p-1.5 rounded-lg">
            {connected ? (
              <div className="flex items-center space-x-2 px-2">
                <span className="text-[10px] px-1.5 py-0.5 bg-green-950 text-green-400 border border-green-900 rounded font-semibold uppercase">
                  {walletType}
                </span>
                <span className="text-xs text-gray-300 font-mono">
                  {address ? `${address.slice(0, 6)}...${address.slice(-4)}` : ""}
                </span>
                <button 
                  onClick={disconnectWallet}
                  className="text-xs text-red-400 hover:text-red-300 transition pl-1 border-l border-gray-850"
                >
                  Disconnect
                </button>
              </div>
            ) : (
              <div className="flex space-x-1">
                <button 
                  onClick={() => connectWallet("keplr")}
                  className="text-[10px] px-2 py-1 bg-blue-600 hover:bg-blue-500 text-white rounded font-medium transition"
                >
                  Keplr
                </button>
                <button 
                  onClick={() => connectWallet("metamask")}
                  className="text-[10px] px-2 py-1 bg-yellow-600 hover:bg-yellow-500 text-white rounded font-medium transition"
                >
                  MetaMask
                </button>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex space-x-2 border-b border-gray-900 pb-px">
        {(["all", "voting", "passed", "rejected"] as const).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2.5 text-sm font-medium capitalize border-b-2 transition -mb-px ${
              activeTab === tab
                ? "border-blue-500 text-white"
                : "border-transparent text-gray-400 hover:text-gray-200"
            }`}
          >
            {tab}
          </button>
        ))}
      </div>

      {/* Main Content Area */}
      {loading ? (
        <div className="flex justify-center items-center py-20">
          <RefreshCw className="h-8 w-8 text-blue-500 animate-spin" />
        </div>
      ) : (
        <div className="space-y-4">
          {getFilteredProposals().length === 0 ? (
            <div className="bg-gray-950 border border-gray-900 rounded-xl p-12 text-center text-gray-500">
              <Vote className="h-12 w-12 mx-auto text-gray-700 mb-3" />
              <p>No proposals found in this category.</p>
            </div>
          ) : (
            getFilteredProposals().map((proposal) => {
              const tally = parseTally(proposal.tallyResult);
              return (
                <div key={proposal.id} className="bg-gray-950 border border-gray-900 rounded-xl p-6 hover:border-gray-800 transition space-y-4 shadow-lg">
                  <div className="flex flex-wrap items-start justify-between gap-4">
                    <div className="space-y-1">
                      <div className="flex items-center space-x-2">
                        <span className="text-xs font-mono font-bold text-gray-500">#{proposal.id}</span>
                        <span className={`px-2 py-0.5 text-xs font-semibold rounded uppercase ${
                          proposal.status.toLowerCase() === "voting" ? "bg-yellow-950 text-yellow-400 border border-yellow-900" :
                          proposal.status.toLowerCase() === "passed" ? "bg-green-950 text-green-400 border border-green-900" :
                          "bg-red-950 text-red-400 border border-red-900"
                        }`}>
                          {proposal.status}
                        </span>
                        <span className="px-2 py-0.5 text-xs font-medium bg-gray-900 text-gray-400 rounded-full">
                          {proposal.typeBadge}
                        </span>
                      </div>
                      <h3 className="text-lg font-bold text-white hover:text-blue-400 transition">
                        <Link href={`/governance/${proposal.id}`}>
                          {proposal.title}
                        </Link>
                      </h3>
                    </div>

                    <div className="flex items-center space-x-3 text-xs">
                      {proposal.constitutionCheckPassed ? (
                        <div className="flex items-center space-x-1.5 px-2.5 py-1 bg-green-950/30 border border-green-900/50 rounded-lg text-green-400">
                          <CheckCircle2 className="h-4 w-4" />
                          <span>Constitution check Passed</span>
                        </div>
                      ) : (
                        <div className="flex items-center space-x-1.5 px-2.5 py-1 bg-red-950/30 border border-red-900/50 rounded-lg text-red-400">
                          <XCircle className="h-4 w-4" />
                          <span>Constitution Check Failed</span>
                        </div>
                      )}
                    </div>
                  </div>

                  <p className="text-sm text-gray-400 line-clamp-2">
                    {proposal.description}
                  </p>

                  {/* Vote Tally Progress */}
                  {tally.total > 0 && (
                    <div className="space-y-2">
                      <div className="flex justify-between items-center text-xs">
                        <div className="flex items-center space-x-4">
                          <span className="text-green-500 font-semibold">Yes: {tally.yesPct.toFixed(1)}%</span>
                          <span className="text-red-500 font-semibold">No: {tally.noPct.toFixed(1)}%</span>
                          <span className="text-gray-400 font-semibold">Abstain: {tally.abstainPct.toFixed(1)}%</span>
                        </div>
                        <span className="text-gray-500 font-medium">Total: {tally.total.toLocaleString()} votes</span>
                      </div>
                      <div className="w-full bg-gray-900 h-2.5 rounded-full overflow-hidden flex border border-gray-800">
                        <div className="bg-green-500 h-full" style={{ width: `${tally.yesPct}%` }} title="Yes" />
                        <div className="bg-red-500 h-full" style={{ width: `${tally.noPct}%` }} title="No" />
                        <div className="bg-gray-600 h-full" style={{ width: `${tally.abstainPct}%` }} title="Abstain" />
                      </div>
                    </div>
                  )}

                  {/* Timestamps info */}
                  <div className="flex flex-wrap items-center justify-between text-xs text-gray-500 pt-2 border-t border-gray-900 gap-2">
                    <div>
                      Submitted: <span className="text-gray-400">{new Date(proposal.submitTime).toLocaleString()}</span>
                    </div>
                    {proposal.status.toLowerCase() === "deposit" ? (
                      <div>
                        Deposit Ends: <span className="text-gray-400">{new Date(proposal.depositEndTime).toLocaleString()}</span>
                      </div>
                    ) : (
                      <div>
                        Voting Ends: <span className="text-gray-400">{new Date(proposal.votingEndTime).toLocaleString()}</span>
                      </div>
                    )}
                  </div>
                </div>
              );
            })
          )}
        </div>
      )}
    </div>
  );
}
