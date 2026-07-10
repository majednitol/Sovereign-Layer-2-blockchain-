"use client";

import React, { useState, useEffect } from "react";
import Link from "next/link";
import { Coins, CheckCircle, AlertCircle, RefreshCw, ArrowRight, Loader2 } from "lucide-react";
import { useWalletStore } from "@/store/wallet";

const formatSuccessMessage = (msg: string) => {
  if (!msg) return "";
  // Match any 64-character hex string, capturing the first 10 and last 10 characters
  const hexRegex = /\b([a-fA-F0-9]{10})[a-fA-F0-9]{44}([a-fA-F0-9]{10})\b/;
  return msg.replace(hexRegex, "$1...$2");
};

export default function FAUCETPage() {
  const { walletType, connected, address: walletAddress, connectWallet, disconnectWallet } = useWalletStore();
  
  const [targetAddress, setTargetAddress] = useState("");
  const [loading, setLoading] = useState(false);
  const [successMsg, setSuccessMsg] = useState("");
  const [errorMsg, setErrorMsg] = useState("");
  const [txHash, setTxHash] = useState("");
  const [cooldownRemaining, setCooldownRemaining] = useState<number>(0);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  // Check local storage for active cooldown
  useEffect(() => {
    if (!targetAddress) return;
    const checkCooldown = () => {
      const claimTime = localStorage.getItem(`faucet_next_claim_${targetAddress}`);
      if (claimTime) {
        const remaining = Math.ceil((Number(claimTime) - Date.now()) / 1000);
        if (remaining > 0) {
          setCooldownRemaining(remaining);
        } else {
          setCooldownRemaining(0);
          localStorage.removeItem(`faucet_next_claim_${targetAddress}`);
        }
      } else {
        setCooldownRemaining(0);
      }
    };

    checkCooldown();
    const interval = setInterval(checkCooldown, 1000);
    return () => clearInterval(interval);
  }, [targetAddress]);

  // Auto-fill address if wallet is connected
  useEffect(() => {
    if (connected && walletAddress) {
      setTargetAddress(walletAddress);
    }
  }, [connected, walletAddress]);

  const formatCooldown = (sec: number) => {
    const h = Math.floor(sec / 3600);
    const m = Math.floor((sec % 3600) / 60);
    const s = sec % 60;
    return `${h.toString().padStart(2, "0")}:${m.toString().padStart(2, "0")}:${s.toString().padStart(2, "0")}`;
  };

  const handleRequestTokens = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!targetAddress.trim()) {
      setErrorMsg("Please enter a valid address.");
      return;
    }

    setLoading(true);
    setErrorMsg("");
    setSuccessMsg("");
    setTxHash("");

    try {
      const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/faucet`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ address: targetAddress.trim() }),
      });

      let data: any = {};
      const contentType = resp.headers.get("content-type");
      if (contentType && contentType.includes("application/json")) {
        data = await resp.json();
      } else {
        const text = await resp.text();
        data = { error: text };
      }

      if (resp.ok && data.success) {
        setSuccessMsg(data.message || "Tokens successfully requested!");
        const cooldown = Number(data.cooldownSeconds || 86400);
        localStorage.setItem(`faucet_next_claim_${targetAddress.trim()}`, String(Date.now() + cooldown * 1000));
        setCooldownRemaining(cooldown);
        if (data.tx_hash) {
          setTxHash(data.tx_hash);
        }
      } else if (resp.status === 429) {
        const cooldown = Number(data.cooldownSeconds || 86400);
        localStorage.setItem(`faucet_next_claim_${targetAddress.trim()}`, String(Date.now() + cooldown * 1000));
        setCooldownRemaining(cooldown);
        setErrorMsg(data.error || "Rate limit: 1 request per 24 hours.");
      } else {
        setErrorMsg(data.error || "Failed to claim tokens. Please try again.");
      }
    } catch (err) {
      console.error(err);
      setErrorMsg("Network error. Could not connect to the faucet service.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white">Faucet</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4 flex justify-between items-center">
        <div className="flex items-center space-x-3">
          <Coins className="text-blue-500 h-8 w-8" />
          <div>
            <h1 className="text-3xl font-bold tracking-tight text-white">Devnet Faucet</h1>
            <p className="text-gray-400 mt-1">Get free test tokens (USOV) to develop and test smart contracts</p>
          </div>
        </div>

        {/* Wallet Connect Panel */}
        <div className="flex items-center space-x-4 bg-gray-900 border border-gray-850 p-3 rounded-lg">
          {connected ? (
            <div className="flex items-center space-x-3">
              <span className="text-xs px-2 py-1 bg-green-950 text-green-400 border border-green-900 rounded font-semibold uppercase">
                {walletType}
              </span>
              <span className="text-sm text-gray-300 font-mono">
                {walletAddress ? `${walletAddress.slice(0, 8)}...${walletAddress.slice(-6)}` : ""}
              </span>
              <button 
                onClick={disconnectWallet}
                className="text-xs text-red-400 hover:text-red-300 transition"
              >
                Disconnect
              </button>
            </div>
          ) : (
            <div className="flex space-x-2">
              <button 
                onClick={() => connectWallet("keplr")}
                className="text-xs px-3 py-1.5 bg-blue-600 hover:bg-blue-500 text-white rounded font-medium transition"
              >
                Connect Keplr
              </button>
              <button 
                onClick={() => connectWallet("metamask")}
                className="text-xs px-3 py-1.5 bg-yellow-600 hover:bg-yellow-500 text-white rounded font-medium transition"
              >
                Connect MetaMask
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Faucet Card */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="md:col-span-2 bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-xl space-y-6">
          <h2 className="text-xl font-bold text-white">Request Devnet Funds</h2>
          
          <form onSubmit={handleRequestTokens} className="space-y-4">
            <div className="space-y-2">
              <label htmlFor="address" className="text-xs text-gray-400 uppercase font-bold tracking-wider">
                Wallet Address
              </label>
              <input
                id="address"
                type="text"
                value={targetAddress}
                onChange={(e) => setTargetAddress(e.target.value)}
                placeholder="Enter your sov... or cosmos... address"
                className="w-full px-4 py-3 bg-black/40 border border-gray-800 focus:border-blue-600 focus:ring-1 focus:ring-blue-600 rounded-xl text-white font-mono text-sm outline-none transition"
                disabled={loading}
              />
              {connected && (
                <button
                  type="button"
                  onClick={() => setTargetAddress(walletAddress || "")}
                  className="text-xs text-blue-500 hover:text-blue-400 transition"
                >
                  Use Connected Wallet Address
                </button>
              )}
            </div>

            <button
              type="submit"
              disabled={loading || !targetAddress.trim() || cooldownRemaining > 0}
              className="w-full py-3 bg-blue-600 hover:bg-blue-500 disabled:bg-gray-850 disabled:text-gray-500 text-white font-medium rounded-xl transition shadow-lg shadow-blue-900/20 flex justify-center items-center space-x-2"
            >
              {loading ? (
                <>
                  <Loader2 className="h-5 w-5 animate-spin" />
                  <span>Requesting...</span>
                </>
              ) : cooldownRemaining > 0 ? (
                <span>Cooldown: {formatCooldown(cooldownRemaining)}</span>
              ) : (
                <>
                  <span>Request 10 CSOV</span>
                  <ArrowRight className="h-4 w-4" />
                </>
              )}
            </button>
          </form>

          {/* Success Panel */}
          {successMsg && (
            <div className="p-4 bg-green-950/20 border border-green-900 rounded-xl space-y-2 text-green-400">
              <div className="flex items-center space-x-2 font-bold">
                <CheckCircle className="h-5 w-5" />
                <span>{formatSuccessMessage(successMsg)}</span>
              </div>
              {txHash && (
                <div className="pt-2">
                  <Link 
                    href={`/txs/${txHash}`}
                    className="inline-flex items-center gap-1.5 px-4 py-2 bg-green-900/40 hover:bg-green-900/60 border border-green-800 text-green-300 hover:text-green-200 text-xs font-bold uppercase tracking-wider rounded-lg transition"
                  >
                    <span>View Transaction</span>
                    <ArrowRight className="h-3.5 w-3.5" />
                  </Link>
                </div>
              )}
            </div>
          )}

          {/* Error Panel */}
          {errorMsg && (
            <div className="p-4 bg-red-950/20 border border-red-900 rounded-xl flex items-start space-x-2 text-red-400">
              <AlertCircle className="h-5 w-5 mt-0.5 flex-shrink-0" />
              <div className="text-sm">
                <span className="font-bold block">Error</span>
                <span className="text-xs leading-normal">{errorMsg}</span>
              </div>
            </div>
          )}
        </div>

        {/* Sidebar details */}
        <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-xl space-y-4 text-sm text-gray-400">
          <h3 className="text-base font-bold text-white">Faucet Info</h3>
          
          <div className="space-y-3">
            <div className="pb-3 border-b border-gray-900">
              <span className="block text-xs uppercase font-bold text-gray-500">Distribution Amount</span>
              <span className="text-white font-medium">10,000,000 ucsov (10 CSOV)</span>
            </div>

            <div className="pb-3 border-b border-gray-900">
              <span className="block text-xs uppercase font-bold text-gray-500">Rate Limit</span>
              <span className="text-white font-medium">1 request per address / IP every 24 hours</span>
            </div>

            <div>
              <span className="block text-xs uppercase font-bold text-gray-500">Supported Formats</span>
              <ul className="list-disc pl-4 space-y-1 mt-1 text-xs">
                <li>Cosmos Addresses (<code className="text-gray-300 font-mono">cosmos...</code>)</li>
              </ul>
            </div>
          </div>

          <div className="pt-4 border-t border-gray-900 text-xs text-gray-500 leading-normal">
            Please note: This faucet is strictly for development and testing purposes on the Sovereign Devnet. The tokens distributed here have no real monetary value.
          </div>
        </div>
      </div>
    </div>
  );
}
