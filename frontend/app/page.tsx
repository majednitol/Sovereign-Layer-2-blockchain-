"use client";

import React, { useState, useEffect } from "react";
import WalletConnect from "../components/WalletConnect";
import BridgeTracker from "../components/BridgeTracker";
import DataStatusIndicator from "../components/DataStatusIndicator";
import { QueryServiceClient } from "@workspace/api-spec";
import { transport } from "../config/grpc-client";


const queryClient = new QueryServiceClient(transport);

interface BridgeTx {
  txHash: string;
  sender: string;
  recipient: string;
  amount: string;
  timestamp: string;
  status: "pending" | "confirmed" | "failed";
  blocksRemaining: number;
  blocksTotal: number;
  secondsRemaining: number;
}

// Encode LockBox.lock(uint256, string) manually to avoid dependencies
function encodeLockData(amountWSOV: number, recipient: string): string {
  const selector = "f643509c"; // selector for lock(uint256,string)
  const amountBig = BigInt(Math.floor(amountWSOV)) * BigInt(10 ** 18);
  const amountHex = amountBig.toString(16).padStart(64, "0");
  const offsetHex = "0000000000000000000000000000000000000000000000000000000000000040";
  const utf8 = new TextEncoder().encode(recipient);
  const lenHex = utf8.length.toString(16).padStart(64, "0");

  let contentHex = "";
  for (let i = 0; i < utf8.length; i++) {
    contentHex += utf8[i].toString(16).padStart(2, "0");
  }
  const paddedLen = Math.ceil(utf8.length / 32) * 32;
  contentHex = contentHex.padEnd(paddedLen * 2, "0");

  return "0x" + selector + amountHex + offsetHex + lenHex + contentHex;
}

export default function Home() {
  const [l1Address, setL1Address] = useState<string>("");
  const [l2Address, setL2Address] = useState<string>("");
  const [amount, setAmount] = useState<string>("");
  const [activeTxs, setActiveTxs] = useState<BridgeTx[]>([]);
  const [connStatus, setConnStatus] = useState<"live" | "degraded" | "offline">("live");
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const [metrics, setMetrics] = useState({
    totalVolume: "0 WSOV",
    uptime: "0.00%",
    validatorCount: "0 / 0",
    rewardsRunway: "—",
  });

  const fetchRealMetrics = async () => {
    let volumeVal = "0 WSOV";
    let uptimeVal = "0.00%";
    let validatorVal = "0 / 0";
    let failedCount = 0;

    // 1. Get total volume from bridge volume API
    try {
      const bridgeCall = await queryClient.getBridgeVolume({
        tokenAddress: "uwsov",
        chainId: "sovereign-1",
        timeframe: "all",
      });
      const volumeStr = bridgeCall.response.totalMinted || "0";
      const parsedVolume = parseFloat(volumeStr);
      volumeVal = `${parsedVolume.toLocaleString()} WSOV`;
    } catch (e) {
      volumeVal = "Unavailable (API Error)";
      failedCount++;
    }

    // 2. Get validator uptime (average of primary validators)
    try {
      const res = await queryClient.getValidatorUptime({ validatorAddress: "FC77EDB49C1CA633E23F6D59E0C51DC86ED1C61C" });
      if (res.response && res.response.uptimePercentage > 0) {
        uptimeVal = `${res.response.uptimePercentage.toFixed(2)}%`;
      }
    } catch (e) {
      uptimeVal = "Unavailable (API Error)";
      failedCount++;
    }

    // 3. Get active validator count by fetching cosmos staking endpoint
    try {
      const res = await fetch("http://localhost:8080/api/rest/cosmos/staking/v1beta1/validators");
      if (res.ok) {
        const data = await res.json();
        if (data && data.validators && data.validators.length > 0) {
          const total = data.validators.length;
          const active = data.validators.filter((v: any) => v.status === "BOND_STATUS_BONDED").length;
          validatorVal = `${active} / ${total}`;
        }
      } else {
        validatorVal = "Unavailable (API Error)";
        failedCount++;
      }
    } catch (e) {
      validatorVal = "Unavailable (API Error)";
      failedCount++;
    }

    setMetrics({
      totalVolume: volumeVal,
      uptime: uptimeVal,
      validatorCount: validatorVal,
      rewardsRunway: "—",
    });

    if (failedCount === 3) {
      setConnStatus("offline");
    } else if (failedCount > 0) {
      setConnStatus("degraded");
    } else {
      setConnStatus("live");
    }
    setLastUpdated(new Date());
  };

  // Poll real bridge metrics (total volume, uptime, validator count) from backend APIs
  useEffect(() => {
    fetchRealMetrics();
    const interval = setInterval(fetchRealMetrics, 10000);
    return () => clearInterval(interval);
  }, []);

  // Poll real bridge tx status from backend API
  useEffect(() => {
    const pendingTxs = activeTxs.filter(tx => tx.status === "pending");
    if (pendingTxs.length === 0) return;

    const interval = setInterval(async () => {
      const updatedTxs = await Promise.all(
        activeTxs.map(async (tx) => {
          if (tx.status !== "pending") return tx;
          try {
            const res = await queryClient.getBridgeTx({ txHash: tx.txHash });
            if (res.response && res.response.status) {
              const apiStatus = res.response.status.toLowerCase();
              let status: BridgeTx["status"] = "pending";
              if (apiStatus === "confirmed" || apiStatus === "success" || apiStatus === "completed") {
                status = "confirmed";
              } else if (apiStatus === "failed") {
                status = "failed";
              }
              return {
                ...tx,
                status,
                blocksRemaining: status === "confirmed" ? 0 : tx.blocksRemaining,
                secondsRemaining: status === "confirmed" ? 0 : tx.secondsRemaining,
              };
            }
          } catch (e) {
            // Ignore error and retry next time
          }
          return tx;
        })
      );

      const hasChanges = updatedTxs.some((tx, idx) => tx.status !== activeTxs[idx].status);
      if (hasChanges) {
        setActiveTxs(updatedTxs);
      }
    }, 5000);

    return () => clearInterval(interval);
  }, [activeTxs]);

  const handleSendBridge = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!amount || parseFloat(amount) <= 0) {
      alert("Please enter a valid amount to bridge.");
      return;
    }
    if (!l1Address) {
      alert("Please connect your L1 Keplr Wallet first to define the recipient.");
      return;
    }

    const value = parseFloat(amount);
    const isLarge = value >= 10000;
    const blocks = isLarge ? 100 : 15;
    const timeInSec = blocks * 5; // 5 seconds per block estimate

    const win = window as any;
    if (!win.ethereum) {
      alert("MetaMask extension not found. Please connect MetaMask.");
      return;
    }

    try {
      const accounts = await win.ethereum.request({ method: "eth_accounts" });
      if (!accounts || accounts.length === 0) {
        alert("Please connect MetaMask wallet first using the Connect button.");
        return;
      }

      const from = accounts[0];
      const data = encodeLockData(value, l1Address);
      const lockBoxAddress = process.env.NEXT_PUBLIC_LOCKBOX_ADDRESS || "0x1234567890123456789012345678901234567890";

      const txHash = await win.ethereum.request({
        method: "eth_sendTransaction",
        params: [{
          from,
          to: lockBoxAddress,
          data,
          value: "0x0"
        }]
      });

      const newTx: BridgeTx = {
        txHash,
        sender: from,
        recipient: l1Address,
        amount: `${value.toLocaleString()} WSOV`,
        timestamp: new Date().toLocaleTimeString(),
        status: "pending",
        blocksTotal: blocks,
        blocksRemaining: blocks,
        secondsRemaining: timeInSec,
      };

      setActiveTxs([newTx, ...activeTxs]);
      setAmount("");

      // Update total volume metric
      const parsedVol = parseFloat(metrics.totalVolume.replace(/,/g, ""));
      const currentVol = isNaN(parsedVol) ? 0 : parsedVol;
      const updatedVol = (currentVol + value).toLocaleString() + " WSOV";
      setMetrics(prev => ({ ...prev, totalVolume: updatedVol }));
    } catch (err: any) {
      console.error("MetaMask lock transaction error:", err);
      alert("Transaction failed: " + err.message);
    }
  };

  const handleTick = (updated: BridgeTx[]) => {
    setActiveTxs(updated);
  };

  return (
    <div>
      <h1 className="title-gradient" style={{ fontSize: "2.5rem", marginBottom: "0.5rem", fontFamily: "var(--font-title)" }}>
        Sovereign Bridge Dashboard
      </h1>
      <p style={{ color: "var(--text-secondary)", marginBottom: "2rem", fontSize: "1.05rem" }}>
        Initiate cross-chain deposits from Binance Smart Chain and track real-time confirmations on the Sovereign L1.
      </p>

      <DataStatusIndicator
        status={connStatus}
        lastUpdated={lastUpdated}
        onRefresh={fetchRealMetrics}
      />

      {/* Wallet connection panel */}
      <WalletConnect 
        onConnectL1={setL1Address}
        onConnectL2={setL2Address}
        onDisconnect={() => {
          setL1Address("");
          setL2Address("");
        }}
      />

      {/* Bento Grid */}
      <div className="bento-grid" style={{ marginTop: "1.5rem" }}>
        
        {/* Bridge Submission Form */}
        <div className="card col-6">
          <h3 style={{ marginBottom: "1rem", fontFamily: "var(--font-title)", display: "flex", alignItems: "center", gap: "0.5rem" }}>
            <span style={{ display: "inline-block", width: "12px", height: "12px", borderRadius: "50%", backgroundColor: "var(--accent-primary)" }}></span>
            Initiate Bridge-In Deposit
          </h3>
          
          <form onSubmit={handleSendBridge}>
            <div className="form-group">
              <label>Destination AccAddress (Sovereign L1)</label>
              <input 
                type="text" 
                className="form-control" 
                placeholder="Connect Keplr Wallet to autofill address..." 
                value={l1Address}
                onChange={(e) => setL1Address(e.target.value)}
                required
              />
            </div>
            
            <div className="form-group">
              <label>Amount to Bridge (WSOV)</label>
              <input 
                type="number" 
                className="form-control" 
                placeholder="e.g. 5000 (Standard) or 15000 (Large)" 
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
                required
              />
            </div>

            <div style={{ display: "flex", gap: "1rem", alignItems: "center", background: "rgba(255,255,255,0.02)", padding: "0.75rem 1rem", borderRadius: "8px", border: "1px solid var(--border-color)", marginBottom: "1.5rem" }}>
              <span style={{ fontSize: "0.85rem", color: "var(--text-secondary)" }}>Confirmation Tier:</span>
              {amount && parseFloat(amount) >= 10000 ? (
                <span className="badge badge-secondary">Large Transfer (100 Blocks)</span>
              ) : (
                <span className="badge badge-primary">Standard Transfer (15 Blocks)</span>
              )}
            </div>

            <button 
              type="submit" 
              className="btn btn-primary" 
              style={{ width: "100%" }}
              disabled={!l1Address || connStatus === "offline"}
            >
              {connStatus === "offline" ? "Bridge Offline (API Connection Failed)" : "Bridge Tokens Inbound"}
            </button>
          </form>
        </div>

        {/* Read-side Projection Metrics Card */}
        <div className="card col-6" style={{ display: "grid", gridTemplateColumns: "repeat(2, 1fr)", gap: "1.5rem" }}>
          
          <div style={{ gridColumn: "span 2" }}>
            <h3 style={{ fontFamily: "var(--font-title)", display: "flex", alignItems: "center", gap: "0.5rem" }}>
              <span style={{ display: "inline-block", width: "12px", height: "12px", borderRadius: "50%", backgroundColor: "var(--accent-secondary)" }}></span>
              Network Metrics (CQRS Read Projection)
            </h3>
          </div>

          <div style={{ background: "rgba(0,0,0,0.15)", padding: "1.25rem", borderRadius: "12px", border: "1px solid var(--border-color)" }}>
            <span style={{ fontSize: "0.8rem", color: "var(--text-secondary)", textTransform: "uppercase" }}>Total Volume</span>
            <div style={{ fontSize: "1.35rem", fontWeight: 700, marginTop: "0.25rem", fontFamily: "var(--font-title)", color: "var(--text-primary)" }}>
              {metrics.totalVolume}
            </div>
          </div>

          <div style={{ background: "rgba(0,0,0,0.15)", padding: "1.25rem", borderRadius: "12px", border: "1px solid var(--border-color)" }}>
            <span style={{ fontSize: "0.8rem", color: "var(--text-secondary)", textTransform: "uppercase" }}>Validator Uptime</span>
            <div style={{ fontSize: "1.35rem", fontWeight: 700, marginTop: "0.25rem", fontFamily: "var(--font-title)", color: "var(--accent-success)" }}>
              {metrics.uptime}
            </div>
          </div>

          <div style={{ background: "rgba(0,0,0,0.15)", padding: "1.25rem", borderRadius: "12px", border: "1px solid var(--border-color)" }}>
            <span style={{ fontSize: "0.8rem", color: "var(--text-secondary)", textTransform: "uppercase" }}>Active Validators</span>
            <div style={{ fontSize: "1.35rem", fontWeight: 700, marginTop: "0.25rem", fontFamily: "var(--font-title)", color: "var(--text-primary)" }}>
              {metrics.validatorCount}
            </div>
          </div>

          <div style={{ background: "rgba(0,0,0,0.15)", padding: "1.25rem", borderRadius: "12px", border: "1px solid var(--border-color)" }}>
            <span style={{ fontSize: "0.8rem", color: "var(--text-secondary)", textTransform: "uppercase" }}>Rewards Runway</span>
            <div style={{ fontSize: "1.35rem", fontWeight: 700, marginTop: "0.25rem", fontFamily: "var(--font-title)", color: "var(--accent-warning)" }}>
              {metrics.rewardsRunway}
            </div>
          </div>

        </div>

      </div>

      {/* Bridge Activity Real-time Monitor */}
      <BridgeTracker 
        activeTxs={activeTxs}
        onTick={handleTick}
      />
    </div>
  );
}
