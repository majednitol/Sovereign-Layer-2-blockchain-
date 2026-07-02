"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useWalletStore } from "@/store/wallet";
import { Coins, Layers, ArrowLeft, History, Users, Send, CheckCircle2, Loader2 } from "lucide-react";

interface MultiTokenItem {
  tokenId: string;
  name: string;
  supply: string;
  uri: string;
}

interface HolderItem {
  address: string;
  tokenId: string;
  balance: string;
}

interface BatchTransferLog {
  hash: string;
  from: string;
  to: string;
  tokenIds: string[];
  amounts: string[];
  time: string;
}

export default function EvmMultiTokenDetailPage() {
  const params = useParams();
  const addr = params?.addr ? String(params.addr) : "";
  const { connected, address, walletType, connectWallet } = useWalletStore();

  const [items, setItems] = useState<MultiTokenItem[]>([]);
  const [holders, setHolders] = useState<HolderItem[]>([]);
  const [transfers, setTransfers] = useState<BatchTransferLog[]>([]);
  const [activeTab, setActiveTab] = useState<"items" | "holders" | "transfers" | "send">("items");
  const [loading, setLoading] = useState(true);

  // Form states
  const [recipient, setRecipient] = useState("");
  const [selectedIds, setSelectedIds] = useState("");
  const [selectedAmounts, setSelectedAmounts] = useState("");
  const [sending, setSending] = useState(false);
  const [sendSuccess, setSendSuccess] = useState<string | null>(null);

  useEffect(() => {
    // Populate simulated items for ERC-1155
    setTimeout(() => {
      setItems([
        { tokenId: "1", name: "Sovereign Gold Medal", supply: "100", uri: "https://ipfs.io/ipfs/QmGoldMedal" },
        { tokenId: "2", name: "Sovereign Silver Badge", supply: "500", uri: "https://ipfs.io/ipfs/QmSilverBadge" },
        { tokenId: "3", name: "Sovereign Participant Ticket", supply: "1200", uri: "https://ipfs.io/ipfs/QmParticipant" },
      ]);
      setHolders([
        { address: "0x3f5c9e2b1d7a8d9e8a7b6c5d4e3f281f449219d5", tokenId: "1", balance: "10" },
        { address: "0x25091a8d7a8b6c5d4e3f281f449219d54e47fd8a", tokenId: "1", balance: "5" },
        { address: "0x1234567890abcdef1234567890abcdef12345678", tokenId: "2", balance: "50" },
        { address: "0x892a10be892a10be892a10be892a10be892a10be8", tokenId: "3", balance: "120" },
      ]);
      setTransfers([
        { hash: "0x3f5c9e2b1d7a8d9e8a7b6c5d4e3f281f449219d54e47fd8ad83861b464815d9d", from: "0x3f5c9e2b1d7a", to: "0x25091a8d7a8b", tokenIds: ["1", "2"], amounts: ["2", "5"], time: new Date().toISOString() },
        { hash: "0x8a7b6c5d4e3f281f449219d54e47fd8ad83861b464815d9d3f5c9e2b1d7a8d9e", from: "0x25091a8d7a8b", to: "0x1234567890ab", tokenIds: ["3"], amounts: ["10"], time: new Date(Date.now() - 60000).toISOString() }
      ]);
      setLoading(false);
    }, 500);
  }, [addr]);

  const handleBatchTransfer = (e: React.FormEvent) => {
    e.preventDefault();
    if (!connected) return;
    setSending(true);
    setSendSuccess(null);
    setTimeout(() => {
      setSending(false);
      setSendSuccess("0x3f5c9e2b1d7a8d9e8a7b6c5d4e3f281f449219d54e47fd8ad83861b464815d9d");
    }, 1200);
  };

  if (loading) {
    return (
      <div className="p-6 max-w-6xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading collection details...</div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/evm" className="hover:text-white transition">EVM</Link>
        <span>/</span>
        <Link href="/evm/tokens" className="hover:text-white transition">Tokens</Link>
        <span>/</span>
        <span className="text-gray-300 font-mono text-xs">{addr.slice(0, 10)}...</span>
      </nav>

      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between border-b border-gray-800 pb-6 gap-4">
        <div className="flex items-center gap-3">
          <Link href="/evm" className="p-2 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-gray-400 hover:text-white transition">
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div>
            <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
              <Layers className="w-8 h-8 text-purple-500 animate-pulse" />
              ERC-1155 Multi-Token Collection
            </h1>
            <p className="text-gray-400 mt-2 font-mono text-xs break-all">Contract: {addr}</p>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex space-x-2 border-b border-gray-900 pb-px">
        <button onClick={() => setActiveTab("items")} className={`px-4 py-2.5 text-sm font-medium border-b-2 transition ${activeTab === "items" ? "border-blue-500 text-blue-500" : "border-transparent text-gray-500 hover:text-gray-300"}`}>
          Items Registry ({items.length})
        </button>
        <button onClick={() => setActiveTab("holders")} className={`px-4 py-2.5 text-sm font-medium border-b-2 transition ${activeTab === "holders" ? "border-blue-500 text-blue-500" : "border-transparent text-gray-500 hover:text-gray-300"}`}>
          Balances & Holders
        </button>
        <button onClick={() => setActiveTab("transfers")} className={`px-4 py-2.5 text-sm font-medium border-b-2 transition ${activeTab === "transfers" ? "border-blue-500 text-blue-500" : "border-transparent text-gray-500 hover:text-gray-300"}`}>
          Batch Transfer Activity
        </button>
        <button onClick={() => setActiveTab("send")} className={`px-4 py-2.5 text-sm font-medium border-b-2 transition ${activeTab === "send" ? "border-blue-500 text-blue-500" : "border-transparent text-gray-500 hover:text-gray-300"}`}>
          Batch Transfer Tool
        </button>
      </div>

      {/* Tab Panels */}
      {activeTab === "items" && (
        <div className="bg-gray-950 border border-gray-900 rounded-2xl overflow-hidden shadow-lg">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm text-gray-400">
              <thead className="bg-black/50 text-xs text-gray-500 uppercase tracking-wider font-bold">
                <tr>
                  <th className="p-4">Token ID</th>
                  <th className="p-4">Item Name</th>
                  <th className="p-4">Circulating Supply</th>
                  <th className="p-4 text-right">Metadata URI</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-900 font-mono">
                {items.map((item) => (
                  <tr key={item.tokenId} className="hover:bg-gray-900/30 transition">
                    <td className="p-4 font-semibold text-white">#{item.tokenId}</td>
                    <td className="p-4 text-gray-300 font-sans font-semibold">{item.name}</td>
                    <td className="p-4">{item.supply} items</td>
                    <td className="p-4 text-right text-xs text-blue-400 hover:underline">
                      <a href={item.uri} target="_blank" rel="noreferrer">{item.uri.slice(0, 30)}...</a>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {activeTab === "holders" && (
        <div className="bg-gray-950 border border-gray-900 rounded-2xl overflow-hidden shadow-lg">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm text-gray-400 font-mono">
              <thead className="bg-black/50 text-xs text-gray-500 uppercase tracking-wider font-bold">
                <tr>
                  <th className="p-4">Holder Address</th>
                  <th className="p-4">Token ID</th>
                  <th className="p-4 text-right">Balance</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-900">
                {holders.map((h, idx) => (
                  <tr key={idx} className="hover:bg-gray-900/30 transition">
                    <td className="p-4 text-white font-semibold">{h.address}</td>
                    <td className="p-4">#{h.tokenId}</td>
                    <td className="p-4 text-right font-extrabold text-gray-200">{h.balance} items</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {activeTab === "transfers" && (
        <div className="bg-gray-950 border border-gray-900 rounded-2xl overflow-hidden shadow-lg">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm text-gray-400 font-mono">
              <thead className="bg-black/50 text-xs text-gray-500 uppercase tracking-wider font-bold">
                <tr>
                  <th className="p-4">Batch Tx Hash</th>
                  <th className="p-4">From</th>
                  <th className="p-4">To</th>
                  <th className="p-4">Token IDs</th>
                  <th className="p-4">Amounts</th>
                  <th className="p-4 text-right">Age</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-900">
                {transfers.map((tx) => (
                  <tr key={tx.hash} className="hover:bg-gray-900/30 transition text-xs">
                    <td className="p-4 font-bold text-white">
                      <Link href={`/evm/txs/${tx.hash}`} className="text-blue-500 hover:underline">
                        {tx.hash.slice(0, 10)}...
                      </Link>
                    </td>
                    <td className="p-4">{tx.from}</td>
                    <td className="p-4">{tx.to}</td>
                    <td className="p-4 text-yellow-400 font-bold">[{tx.tokenIds.join(", ")}]</td>
                    <td className="p-4 text-green-400 font-bold">[{tx.amounts.join(", ")}]</td>
                    <td className="p-4 text-right text-gray-500">{new Date(tx.time).toLocaleTimeString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {activeTab === "send" && (
        <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-lg space-y-6">
          <h3 className="text-lg font-bold text-white flex items-center gap-2 border-b border-gray-900 pb-3">
            <Send className="h-5 w-5 text-indigo-500" /> ERC-1155 safeBatchTransferFrom
          </h3>

          {!connected ? (
            <div className="text-center p-6 bg-gray-900/30 border border-gray-850 rounded-2xl text-xs text-gray-500 space-y-3">
              <p>Connect your MetaMask wallet to send batch transactions.</p>
              <button onClick={() => connectWallet("metamask")} className="px-4 py-2 bg-yellow-600 hover:bg-yellow-500 text-white rounded font-medium transition">
                Connect Wallet
              </button>
            </div>
          ) : sendSuccess ? (
            <div className="p-4 bg-green-950/20 border border-green-900/50 rounded-xl text-green-400 text-xs space-y-2">
              <span className="font-bold block text-sm">Batch Transfer Transmitted!</span>
              <span className="font-mono break-all mt-1 block">Tx Hash: {sendSuccess}</span>
              <button onClick={() => setSendSuccess(null)} className="text-xs text-blue-400 hover:underline font-bold mt-1">Send another batch</button>
            </div>
          ) : (
            <form onSubmit={handleBatchTransfer} className="space-y-4 max-w-xl text-xs">
              <div className="space-y-1">
                <label className="text-[10px] text-gray-500 font-bold uppercase block">Recipient Address (to)</label>
                <input 
                  type="text" 
                  value={recipient}
                  onChange={(e) => setRecipient(e.target.value)}
                  placeholder="0x..." 
                  className="w-full bg-gray-900 border border-gray-800 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-blue-500"
                  required
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <label className="text-[10px] text-gray-500 font-bold uppercase block">Token IDs (comma separated)</label>
                  <input 
                    type="text" 
                    value={selectedIds}
                    onChange={(e) => setSelectedIds(e.target.value)}
                    placeholder="e.g. 1, 2" 
                    className="w-full bg-gray-900 border border-gray-800 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-blue-500"
                    required
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-[10px] text-gray-500 font-bold uppercase block">Amounts (comma separated)</label>
                  <input 
                    type="text" 
                    value={selectedAmounts}
                    onChange={(e) => setSelectedAmounts(e.target.value)}
                    placeholder="e.g. 10, 5" 
                    className="w-full bg-gray-900 border border-gray-800 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-blue-500"
                    required
                  />
                </div>
              </div>

              <button 
                type="submit"
                disabled={sending}
                className="py-2.5 px-6 bg-blue-600 hover:bg-blue-500 disabled:bg-gray-850 text-white rounded-xl font-bold text-xs uppercase tracking-wider flex items-center justify-center gap-2 shadow-lg shadow-blue-900/20"
              >
                {sending ? <Loader2 className="h-4 w-4 animate-spin" /> : "Broadcast safeBatchTransferFrom"}
              </button>
            </form>
          )}
        </div>
      )}
    </div>
  );
}
