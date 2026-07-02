"use client";

import React, { useEffect, useState } from "react";

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

interface BridgeTrackerProps {
  activeTxs: BridgeTx[];
  onTick: (txs: BridgeTx[]) => void;
}

export default function BridgeTracker({
  activeTxs,
  onTick,
}: BridgeTrackerProps) {
  const [wsStatus, setWsStatus] = useState<"connected" | "connecting" | "disconnected">("disconnected");

  useEffect(() => {
    setWsStatus("connecting");
    let abortController: AbortController;
    let reconnectTimeout: NodeJS.Timeout;
    let retryCount = 0;
    const MAX_RETRY_DELAY = 30000;

    const connect = async () => {
      abortController = new AbortController();
      try {
        // gRPC server-streaming via Envoy gRPC-Web proxy (replaces removed WebSocket endpoint)
        // All real-time data flows through gRPC server-streaming only — no WebSocket
        const grpcWebUrl = process.env.NEXT_PUBLIC_GRPC_WEB_URL || "http://localhost:8080/api/grpcweb";
        const response = await fetch(`${grpcWebUrl}/backend.v1.StreamService/StreamBridgeEvents`, {
          method: "POST",
          headers: {
            "Content-Type": "application/grpc-web-text",
            "X-Grpc-Web": "1",
          },
          body: btoa(""), // Empty StreamBridgeEventsRequest
          signal: abortController.signal,
        });

        if (response.ok && response.body) {
          setWsStatus("connected");
          retryCount = 0;
          const reader = response.body.getReader();
          while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            // Handle incoming gRPC stream frames (stub for streaming bridge records)
            console.log("gRPC stream frame received:", value);
          }
        }
        // Stream ended normally — reconnect
        setWsStatus("disconnected");
      } catch (err: unknown) {
        if (err instanceof Error && err.name === "AbortError") return;
        setWsStatus("disconnected");
      }

      // Exponential backoff auto-reconnect
      const delay = Math.min(1000 * Math.pow(2, retryCount), MAX_RETRY_DELAY);
      retryCount++;
      reconnectTimeout = setTimeout(() => {
        connect();
      }, delay);
    };

    connect();

    return () => {
      if (abortController) abortController.abort();
      clearTimeout(reconnectTimeout);
    };
  }, []);

  // Countdown countdown tick interval (every 1s)
  useEffect(() => {
    const interval = setInterval(() => {
      if (activeTxs.length === 0) return;

      const updated = activeTxs.map((tx) => {
        if (tx.status === "confirmed") return tx;

        const newSecRemaining = Math.max(0, tx.secondsRemaining - 1);
        // Estimate block progress: 5 seconds per block
        const elapsedSec = (tx.blocksTotal * 5) - newSecRemaining;
        const currentBlockProgress = Math.min(tx.blocksTotal, Math.floor(elapsedSec / 5));
        const newBlocksRemaining = Math.max(0, tx.blocksTotal - currentBlockProgress);

        let newStatus: BridgeTx["status"] = tx.status;
        if (newSecRemaining === 0) {
          newStatus = "confirmed";
        }

        return {
          ...tx,
          secondsRemaining: newSecRemaining,
          blocksRemaining: newBlocksRemaining,
          status: newStatus,
        };
      });

      onTick(updated);
    }, 1000);

    return () => clearInterval(interval);
  }, [activeTxs, onTick]);

  const truncate = (hash: string) => {
    if (!hash) return "";
    return hash.slice(0, 10) + "..." + hash.slice(-8);
  };

  return (
    <div className="card col-12" style={{ marginTop: "1.5rem" }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "1.5rem" }}>
        <h3 style={{ fontFamily: "var(--font-title)" }}>Active Transfers & Confirmation Tiers</h3>
        <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
          <span style={{
            display: "inline-block",
            width: "8px",
            height: "8px",
            borderRadius: "50%",
            backgroundColor: wsStatus === "connected" ? "var(--accent-success)" : wsStatus === "connecting" ? "var(--accent-warning)" : "var(--text-secondary)"
          }} className={wsStatus === "connecting" ? "pulse" : ""}></span>
          <span style={{ fontSize: "0.85rem", color: "var(--text-secondary)" }}>
            {wsStatus === "disconnected" ? "Stream: Offline (Polling Fallback)" : `Stream: ${wsStatus.toUpperCase()}`}
          </span>
        </div>
      </div>

      {activeTxs.length === 0 ? (
        <p style={{ color: "var(--text-secondary)", textAlign: "center", padding: "2rem 0" }}>
          No active cross-chain bridge transfers in flight. Send tokens from BSC to trigger confirmations.
        </p>
      ) : (
        <div className="table-container">
          <table>
            <thead>
              <tr>
                <th>Tx Hash</th>
                <th>Sender (BSC)</th>
                <th>Receiver (L1)</th>
                <th>Amount</th>
                <th>Confirmations</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {activeTxs.map((tx) => (
                <tr key={tx.txHash}>
                  <td style={{ fontFamily: "monospace" }}>{truncate(tx.txHash)}</td>
                  <td style={{ fontFamily: "monospace" }}>{truncate(tx.sender)}</td>
                  <td style={{ fontFamily: "monospace" }}>{truncate(tx.recipient)}</td>
                  <td style={{ fontWeight: 600 }}>{tx.amount}</td>
                  <td>
                    {tx.status === "confirmed" ? (
                      <span className="badge badge-success">Completed</span>
                    ) : (
                      <div style={{ display: "flex", flexDirection: "column", gap: "0.25rem" }}>
                        <span style={{ fontSize: "0.85rem", fontWeight: 500 }}>
                          Blocks: {tx.blocksTotal - tx.blocksRemaining} / {tx.blocksTotal}
                        </span>
                        <div style={{
                          width: "120px",
                          height: "6px",
                          backgroundColor: "rgba(255,255,255,0.05)",
                          borderRadius: "3px",
                          overflow: "hidden"
                        }}>
                          <div style={{
                            width: `${((tx.blocksTotal - tx.blocksRemaining) / tx.blocksTotal) * 100}%`,
                            height: "100%",
                            backgroundColor: tx.blocksTotal > 15 ? "var(--accent-secondary)" : "var(--accent-primary)",
                            transition: "width 0.5s ease"
                          }}></div>
                        </div>
                        <span style={{ fontSize: "0.75rem", color: "var(--text-secondary)" }}>
                          Est. Time: {tx.secondsRemaining}s
                        </span>
                      </div>
                    )}
                  </td>
                  <td>
                    <span className={`badge ${
                      tx.status === "confirmed" ? "badge-success" : tx.status === "pending" ? "badge-warning" : "badge-secondary"
                    }`}>
                      {tx.status}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
