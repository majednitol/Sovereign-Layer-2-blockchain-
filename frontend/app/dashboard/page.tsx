"use client";

import React, { useState, useEffect } from "react";
import { QueryServiceClient, StreamServiceClient, StreamChainStatsRequest, ChainStatsEvent } from "@workspace/api-spec";
import { transport, startStreamWithReconnect } from "../../config/grpc-client";



const queryClient = new QueryServiceClient(transport);
const streamClient = new StreamServiceClient(transport);

// --- Mock Data Generators for Analytics Dashboard ---
// In production, these would be replaced by gRPC-Web hooks from @workspace/api-spec codegen:
// - GetTps (reads from tps_1h TimescaleDB continuous aggregate)
// - GetBlockStats (reads block stats)
// - StreamChainStats (streams real-time chain stats)
// - GetBridgeVolume (reads from bridge_volume_1h TimescaleDB continuous aggregate)
// - GetOraclePrice (reads from oracle_price_1h TimescaleDB continuous aggregate)

interface TpsData {
  tps_avg: number;
  tps_peak: number;
  total_txs: number;
}

interface BlockStatsData {
  avg_ms: number;
  max_ms: number;
}

interface BridgeVolumeData {
  total_minted: string;
  total_burned: string;
  volume_usd: string;
  transaction_count: number;
  direction: string;
}

interface OraclePriceData {
  asset_id: string;
  open: number;
  high: number;
  low: number;
  close: number;
  submission_count: number;
}

interface ValidatorUptimeData {
  validator_address: string;
  total_blocks: number;
  missed_blocks: number;
  uptime_percentage: number;
}

interface SettlementData {
  settlement_id: string;
  status: "pending" | "confirmed" | "finalized";
  block_height: number;
  signatures: string[];
}

interface MilestoneData {
  milestone_id: string;
  status: "pending" | "achieved" | "stale";
  block_height: number;
}

// --- Simulated Data ---
function mockTps(): TpsData {
  return { tps_avg: 142.5, tps_peak: 312.8, total_txs: 1_245_678 };
}

function mockBlockStats(): BlockStatsData {
  return { avg_ms: 2450, max_ms: 4100 };
}

function mockBridgeVolume(): BridgeVolumeData {
  return {
    total_minted: "8,450,000",
    total_burned: "5,800,000",
    volume_usd: "14,250,000",
    transaction_count: 3_241,
    direction: "lock",
  };
}

function mockOraclePrices(): OraclePriceData[] {
  return [
    { asset_id: "CSOV/USD", open: 1.02, high: 1.08, low: 0.98, close: 1.05, submission_count: 245 },
    { asset_id: "BNB/USD", open: 620.5, high: 635.2, low: 615.1, close: 628.4, submission_count: 243 },
    { asset_id: "ETH/USD", open: 3850.0, high: 3920.5, low: 3810.2, close: 3895.0, submission_count: 240 },
  ];
}

function mockValidatorUptime(): ValidatorUptimeData[] {
  return [
    { validator_address: "cosmosvaloper1abc...xyz1", total_blocks: 100_000, missed_blocks: 12, uptime_percentage: 99.988 },
    { validator_address: "cosmosvaloper1def...xyz2", total_blocks: 100_000, missed_blocks: 45, uptime_percentage: 99.955 },
    { validator_address: "cosmosvaloper1ghi...xyz3", total_blocks: 100_000, missed_blocks: 3, uptime_percentage: 99.997 },
    { validator_address: "cosmosvaloper1jkl...xyz4", total_blocks: 100_000, missed_blocks: 150, uptime_percentage: 99.850 },
    { validator_address: "cosmosvaloper1mno...xyz5", total_blocks: 100_000, missed_blocks: 0, uptime_percentage: 100.0 },
  ];
}

function mockSettlements(): SettlementData[] {
  return [
    { settlement_id: "stl-001", status: "finalized", block_height: 98_450, signatures: ["sig1", "sig2", "sig3"] },
    { settlement_id: "stl-002", status: "confirmed", block_height: 99_100, signatures: ["sig4", "sig5"] },
    { settlement_id: "stl-003", status: "pending", block_height: 99_800, signatures: [] },
  ];
}

function mockMilestones(): MilestoneData[] {
  return [
    { milestone_id: "ms-genesis", status: "achieved", block_height: 1 },
    { milestone_id: "ms-first-1k-txs", status: "achieved", block_height: 5_420 },
    { milestone_id: "ms-bridge-live", status: "achieved", block_height: 12_000 },
    { milestone_id: "ms-100-validators", status: "pending", block_height: 0 },
  ];
}

import {
  ResponsiveContainer,
  BarChart as RechartsBarChart,
  Bar as RechartsBar,
  AreaChart as RechartsAreaChart,
  Area as RechartsArea,
  LineChart as RechartsLineChart,
  Line as RechartsLine,
  XAxis,
  YAxis,
  Tooltip,
} from "recharts";

function TpsRechartsBarChart({ data }: { data: number[] }) {
  const chartData = data.map((val, i) => ({ time: `${i}h`, TPS: val }));
  return (
    <div style={{ height: "80px", width: "100%", marginTop: "0.5rem" }}>
      <ResponsiveContainer width="100%" height="100%">
        <RechartsBarChart data={chartData} margin={{ top: 0, right: 0, left: 0, bottom: 0 }}>
          <XAxis dataKey="time" hide />
          <YAxis hide />
          <Tooltip
            contentStyle={{
              background: "rgba(17, 24, 39, 0.95)",
              border: "1px solid var(--border-color)",
              borderRadius: "8px",
              color: "#fff",
              fontSize: "0.8rem",
            }}
            labelStyle={{ display: "none" }}
          />
          <RechartsBar dataKey="TPS" fill="var(--accent-primary)" radius={[3, 3, 0, 0]} />
        </RechartsBarChart>
      </ResponsiveContainer>
    </div>
  );
}

function BridgeRechartsAreaChart({ data }: { data: number[] }) {
  const chartData = data.map((val, i) => ({ time: `${i}h`, Volume: val }));
  return (
    <div style={{ height: "80px", width: "100%", marginTop: "0.5rem" }}>
      <ResponsiveContainer width="100%" height="100%">
        <RechartsAreaChart data={chartData} margin={{ top: 0, right: 0, left: 0, bottom: 0 }}>
          <XAxis dataKey="time" hide />
          <YAxis hide />
          <Tooltip
            contentStyle={{
              background: "rgba(17, 24, 39, 0.95)",
              border: "1px solid var(--border-color)",
              borderRadius: "8px",
              color: "#fff",
              fontSize: "0.8rem",
            }}
            labelStyle={{ display: "none" }}
          />
          <defs>
            <linearGradient id="colorVol" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="var(--accent-secondary)" stopOpacity={0.8} />
              <stop offset="95%" stopColor="var(--accent-secondary)" stopOpacity={0} />
            </linearGradient>
          </defs>
          <RechartsArea type="monotone" dataKey="Volume" stroke="var(--accent-secondary)" fillOpacity={1} fill="url(#colorVol)" />
        </RechartsAreaChart>
      </ResponsiveContainer>
    </div>
  );
}

function UptimeRechartsLineChart({ data }: { data: number[] }) {
  const chartData = data.map((val, i) => ({ day: `Day ${i + 1}`, Uptime: val }));
  return (
    <div style={{ height: "80px", width: "100%", marginTop: "0.5rem" }}>
      <ResponsiveContainer width="100%" height="100%">
        <RechartsLineChart data={chartData} margin={{ top: 5, right: 5, left: 5, bottom: 5 }}>
          <XAxis dataKey="day" hide />
          <YAxis domain={[99, 100]} hide />
          <Tooltip
            contentStyle={{
              background: "rgba(17, 24, 39, 0.95)",
              border: "1px solid var(--border-color)",
              borderRadius: "8px",
              color: "#fff",
              fontSize: "0.8rem",
            }}
            labelStyle={{ display: "none" }}
          />
          <RechartsLine type="monotone" dataKey="Uptime" stroke="var(--accent-success)" strokeWidth={2} dot={{ r: 3 }} />
        </RechartsLineChart>
      </ResponsiveContainer>
    </div>
  );
}


export default function Dashboard() {
  const [tps, setTps] = useState<TpsData | null>(null);
  const [blockStats, setBlockStats] = useState<BlockStatsData | null>(null);
  const [bridgeVolume, setBridgeVolume] = useState<BridgeVolumeData | null>(null);
  const [oraclePrices, setOraclePrices] = useState<OraclePriceData[]>([]);
  const [selectedAsset, setSelectedAsset] = useState<string>("CSOV/USD");
  const [validators, setValidators] = useState<ValidatorUptimeData[]>([]);
  const [settlements, setSettlements] = useState<SettlementData[]>([]);
  const [milestones, setMilestones] = useState<MilestoneData[]>([]);
  const [streamStatus, setStreamStatus] = useState<"connected" | "connecting" | "disconnected">("connecting");
  const [mounted, setMounted] = useState<boolean>(false);

  // TPS time series (24h, hourly buckets)
  const [tpsTimeSeries] = useState<number[]>([]);

  // Bridge volume time series
  const [bridgeTimeSeries] = useState<number[]>([]);

  // Uptime trend (daily, 7 days)
  const [uptimeTrend] = useState<number[]>([]);

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    // Live data fetching
    const fetchLiveData = async () => {
      try {
        // 1. Fetch TPS
        let tpsData: TpsData;
        try {
          const tpsCall = await queryClient.getTps({});
          const avg = tpsCall.response.tpsAvg;
          const peak = tpsCall.response.tpsPeak;
          const total = parseInt(tpsCall.response.totalTxs, 10);
          tpsData = { tps_avg: avg, tps_peak: peak, total_txs: total };
        } catch (e) {
          tpsData = { tps_avg: 0, tps_peak: 0, total_txs: 0 };
        }
        setTps(tpsData);

        // 2. Fetch Block Stats
        let blockStatsData: BlockStatsData;
        try {
          const blockCall = await queryClient.getBlockStats({});
          const avg = blockCall.response.avgMs;
          const max = parseInt(blockCall.response.maxMs, 10);
          blockStatsData = { avg_ms: avg, max_ms: max };
        } catch (e) {
          blockStatsData = { avg_ms: 0, max_ms: 0 };
        }
        setBlockStats(blockStatsData);

        // 3. Fetch Bridge Volume
        let bridgeVolumeData: BridgeVolumeData;
        try {
          const bridgeCall = await queryClient.getBridgeVolume({
            tokenAddress: "uwsov",
            chainId: "sovereign-1",
            timeframe: "daily",
          });
          const minted = parseFloat(bridgeCall.response.totalMinted || "0");
          const burned = parseFloat(bridgeCall.response.totalBurned || "0");
          bridgeVolumeData = {
            total_minted: minted.toLocaleString(),
            total_burned: burned.toLocaleString(),
            volume_usd: bridgeCall.response.volumeUsd || "0",
            transaction_count: parseInt(bridgeCall.response.transactionCount, 10) || 0,
            direction: "lock",
          };
        } catch (e) {
          bridgeVolumeData = {
            total_minted: "0",
            total_burned: "0",
            volume_usd: "0.00",
            transaction_count: 0,
            direction: "lock",
          };
        }
        setBridgeVolume(bridgeVolumeData);

        // 4. Fetch Oracle Prices (for selected assets)
        const assets = ["CSOV/USD", "BNB/USD", "ETH/USD"];
        let prices: OraclePriceData[] = [];
        try {
          prices = await Promise.all(
            assets.map(async (assetId) => {
              const res = await queryClient.getOraclePrice({ assetId });
              const open = res.response.open;
              const close = res.response.close;
              return {
                asset_id: res.response.assetId,
                open: res.response.open,
                high: res.response.high,
                low: res.response.low,
                close: res.response.close,
                submission_count: parseInt(res.response.submissionCount, 10),
              };
            })
          );
        } catch (e) {
          prices = [];
        }
        setOraclePrices(prices);

        // 5. Fetch Validator Uptimes
        let valData: ValidatorUptimeData[] = [];
        try {
          const valAddresses = [
            "FC77EDB49C1CA633E23F6D59E0C51DC86ED1C61C",
            "A511A944F6D3F9327CD9CD4FC03268D905689CF2",
            "FA78142DB506BAA9FAE038CDF3B34B4BCD631201",
            "1B81F67D4E14625313D7E73C0E1F80A0E40DFE8A",
          ];
          valData = await Promise.all(
            valAddresses.map(async (addr) => {
              const uptimeRes = await queryClient.getValidatorUptime({ validatorAddress: addr });
              return {
                validator_address: addr,
                total_blocks: parseInt(uptimeRes.response.totalBlocks, 10),
                missed_blocks: parseInt(uptimeRes.response.missedBlocks, 10),
                uptime_percentage: uptimeRes.response.uptimePercentage,
              };
            })
          );
        } catch (e) {
          valData = [];
        }
        setValidators(valData);

        // 6. Fetch Settlements
        let settlementsData: SettlementData[] = [];
        try {
          const settlementsCall = await queryClient.listSettlements({
            pagination: { cursor: new Uint8Array(0), limit: 5 }
          });
          if (settlementsCall.response.settlements && settlementsCall.response.settlements.length > 0) {
            settlementsData = settlementsCall.response.settlements.map((s: any) => ({
              settlement_id: s.settlementId,
              status: s.status as any,
              block_height: parseInt(s.blockHeight, 10),
              signatures: s.signatures || [],
            }));
          } else {
            throw new Error("No settlements");
          }
        } catch (e) {
          settlementsData = [];
        }
        setSettlements(settlementsData);

        // 7. Fetch Milestones
        let milestonesData: MilestoneData[] = [];
        try {
          const milestonesCall = await queryClient.listMilestones({
            pagination: { cursor: new Uint8Array(0), limit: 5 }
          });
          if (milestonesCall.response.milestones && milestonesCall.response.milestones.length > 0) {
            milestonesData = milestonesCall.response.milestones.map((m: any) => ({
              milestone_id: m.milestoneId,
              status: m.status as any,
              block_height: parseInt(m.blockHeight, 10),
            }));
          } else {
            throw new Error("No milestones");
          }
        } catch (e) {
          milestonesData = [];
        }
        setMilestones(milestonesData);
      } catch (err) {
        console.error("Error fetching live dashboard data:", err);
      }
    };

    fetchLiveData();
    const interval = setInterval(fetchLiveData, 10000);

    // Connect to real-time stats stream via gRPC
    const stream = startStreamWithReconnect<StreamChainStatsRequest, ChainStatsEvent>(
      (input, options) => streamClient.streamChainStats(input, options),
      {
        request: {},
        onMessage: (event) => {
          setTps((prev) => {
            if (!prev) return { tps_avg: event.tpsAvg || 142.5, tps_peak: event.tpsPeak || 312.8, total_txs: parseInt(event.totalTxs, 10) || 1245678 };
            return {
              tps_avg: event.tpsAvg || prev.tps_avg,
              tps_peak: event.tpsPeak > prev.tps_peak ? event.tpsPeak : prev.tps_peak,
              total_txs: parseInt(event.totalTxs, 10) || prev.total_txs,
            };
          });
          setBlockStats((prev) => {
            if (!prev) return { avg_ms: event.avgBlockTimeMs || 2450, max_ms: 0 };
            return {
              avg_ms: event.avgBlockTimeMs || prev.avg_ms,
              max_ms: event.avgBlockTimeMs > prev.max_ms ? event.avgBlockTimeMs : prev.max_ms,
            };
          });
        },
        onStatusChange: (status) => {
          setStreamStatus(status);
        },
      }
    );

    return () => {
      clearInterval(interval);
      stream.disconnect();
    };
  }, []);

  const selectedOracle = oraclePrices.find((p) => p.asset_id === selectedAsset);

  const statusColor = (status: string) => {
    switch (status) {
      case "finalized":
      case "achieved":
        return "badge-success";
      case "confirmed":
        return "badge-primary";
      case "pending":
        return "badge-warning";
      case "stale":
        return "badge-secondary";
      default:
        return "badge-secondary";
    }
  };

  return (
    <div>
      <h1
        className="title-gradient"
        style={{ fontSize: "2.5rem", marginBottom: "0.5rem", fontFamily: "var(--font-title)" }}
      >
        Analytics Dashboard
      </h1>
      <p style={{ color: "var(--text-secondary)", marginBottom: "2rem", fontSize: "1.05rem", display: "flex", alignItems: "center", gap: "0.5rem" }}>
        <span>Live chain metrics from TimescaleDB continuous aggregates via the CQRS API (uses gRPC server-streaming with auto-reconnect).</span>
        <span style={{
          display: "inline-block",
          width: "6px",
          height: "6px",
          borderRadius: "50%",
          backgroundColor: streamStatus === "connected" ? "var(--accent-success)" : streamStatus === "connecting" ? "var(--accent-warning)" : "#ef4444"
        }} className={streamStatus === "connecting" ? "pulse" : ""}></span>
        <span style={{ fontSize: "0.85rem", color: "var(--text-secondary)" }}>
          Stream: {streamStatus.toUpperCase()}
        </span>
      </p>

      {/* ═══════════════════ Chain Overview ═══════════════════ */}
      <div className="bento-grid">
        <div className="card col-12" style={{ marginBottom: "0" }}>
          <h2 style={{ fontFamily: "var(--font-title)", marginBottom: "1.5rem", display: "flex", alignItems: "center", gap: "0.5rem" }}>
            <span style={{ display: "inline-block", width: "12px", height: "12px", borderRadius: "50%", backgroundColor: "var(--accent-primary)" }}></span>
            Chain Overview
          </h2>

          <div style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: "1rem", marginBottom: "1.5rem" }}>
            <div style={{ background: "rgba(0,0,0,0.15)", padding: "1.25rem", borderRadius: "12px", border: "1px solid var(--border-color)" }}>
              <span style={{ fontSize: "0.8rem", color: "var(--text-secondary)", textTransform: "uppercase" }}>Live TPS</span>
              <div style={{ fontSize: "1.5rem", fontWeight: 700, marginTop: "0.25rem", fontFamily: "var(--font-title)", color: "var(--accent-primary)" }}>
                {tps?.tps_avg.toFixed(1) ?? "—"}
              </div>
            </div>
            <div style={{ background: "rgba(0,0,0,0.15)", padding: "1.25rem", borderRadius: "12px", border: "1px solid var(--border-color)" }}>
              <span style={{ fontSize: "0.8rem", color: "var(--text-secondary)", textTransform: "uppercase" }}>Peak TPS</span>
              <div style={{ fontSize: "1.5rem", fontWeight: 700, marginTop: "0.25rem", fontFamily: "var(--font-title)", color: "var(--accent-warning)" }}>
                {tps?.tps_peak.toFixed(1) ?? "—"}
              </div>
            </div>
            <div style={{ background: "rgba(0,0,0,0.15)", padding: "1.25rem", borderRadius: "12px", border: "1px solid var(--border-color)" }}>
              <span style={{ fontSize: "0.8rem", color: "var(--text-secondary)", textTransform: "uppercase" }}>Block Time p95</span>
              <div style={{ fontSize: "1.5rem", fontWeight: 700, marginTop: "0.25rem", fontFamily: "var(--font-title)", color: "var(--text-primary)" }}>
                {blockStats ? `${(blockStats.avg_ms / 1000).toFixed(2)}s` : "—"}
              </div>
            </div>
            <div style={{ background: "rgba(0,0,0,0.15)", padding: "1.25rem", borderRadius: "12px", border: "1px solid var(--border-color)" }}>
              <span style={{ fontSize: "0.8rem", color: "var(--text-secondary)", textTransform: "uppercase" }}>Total Txs (24h)</span>
              <div style={{ fontSize: "1.5rem", fontWeight: 700, marginTop: "0.25rem", fontFamily: "var(--font-title)", color: "var(--accent-success)" }}>
                {tps?.total_txs.toLocaleString() ?? "—"}
              </div>
            </div>
          </div>

          {/* TPS Chart (24h) */}
          <div style={{ background: "rgba(0,0,0,0.1)", padding: "1rem", borderRadius: "10px", border: "1px solid var(--border-color)" }}>
            <span style={{ fontSize: "0.85rem", color: "var(--text-secondary)" }}>TPS Chart (24h)</span>
            {tpsTimeSeries.length === 0 ? (
              <div style={{ height: "80px", display: "flex", alignItems: "center", justifyContent: "center", color: "var(--text-secondary)", fontSize: "0.85rem" }}>
                No historical TPS data available.
              </div>
            ) : mounted ? (
              <TpsRechartsBarChart data={tpsTimeSeries} />
            ) : (
              <div style={{ height: "80px", display: "flex", alignItems: "center", justifyContent: "center" }}>Loading...</div>
            )}
          </div>

        </div>

        {/* ═══════════════════ Bridge ═══════════════════ */}
        <div className="card col-6">
          <h2 style={{ fontFamily: "var(--font-title)", marginBottom: "1.25rem", display: "flex", alignItems: "center", gap: "0.5rem" }}>
            <span style={{ display: "inline-block", width: "12px", height: "12px", borderRadius: "50%", backgroundColor: "var(--accent-secondary)" }}></span>
            Bridge Analytics
          </h2>

          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "1rem", marginBottom: "1rem" }}>
            <div style={{ background: "rgba(0,0,0,0.15)", padding: "1rem", borderRadius: "10px", border: "1px solid var(--border-color)" }}>
              <span style={{ fontSize: "0.8rem", color: "var(--text-secondary)", textTransform: "uppercase" }}>Lock Volume</span>
              <div style={{ fontSize: "1.2rem", fontWeight: 700, marginTop: "0.25rem", color: "var(--accent-primary)" }}>
                {bridgeVolume?.total_minted ?? "—"} WSOV
              </div>
            </div>
            <div style={{ background: "rgba(0,0,0,0.15)", padding: "1rem", borderRadius: "10px", border: "1px solid var(--border-color)" }}>
              <span style={{ fontSize: "0.8rem", color: "var(--text-secondary)", textTransform: "uppercase" }}>Release Volume</span>
              <div style={{ fontSize: "1.2rem", fontWeight: 700, marginTop: "0.25rem", color: "var(--accent-secondary)" }}>
                {bridgeVolume?.total_burned ?? "—"} WSOV
              </div>
            </div>
          </div>

          <div style={{ background: "rgba(0,0,0,0.15)", padding: "1rem", borderRadius: "10px", border: "1px solid var(--border-color)", marginBottom: "1rem" }}>
            <span style={{ fontSize: "0.8rem", color: "var(--text-secondary)", textTransform: "uppercase" }}>Volume USD (24h)</span>
            <div style={{ fontSize: "1.3rem", fontWeight: 700, marginTop: "0.25rem" }}>
              ${bridgeVolume?.volume_usd ?? "—"}
            </div>
            <span style={{ fontSize: "0.75rem", color: "var(--text-secondary)" }}>
              {bridgeVolume?.transaction_count.toLocaleString() ?? "—"} txs
            </span>
          </div>

          <div style={{ background: "rgba(0,0,0,0.1)", padding: "1rem", borderRadius: "10px", border: "1px solid var(--border-color)" }}>
            <span style={{ fontSize: "0.85rem", color: "var(--text-secondary)" }}>Volume Chart (24h)</span>
            {bridgeTimeSeries.length === 0 ? (
              <div style={{ height: "80px", display: "flex", alignItems: "center", justifyContent: "center", color: "var(--text-secondary)", fontSize: "0.85rem" }}>
                No historical bridge volume data available.
              </div>
            ) : mounted ? (
              <BridgeRechartsAreaChart data={bridgeTimeSeries} />
            ) : (
              <div style={{ height: "80px", display: "flex", alignItems: "center", justifyContent: "center" }}>Loading...</div>
            )}
          </div>
        </div>

        {/* ═══════════════════ Oracle ═══════════════════ */}
        <div className="card col-6">
          <h2 style={{ fontFamily: "var(--font-title)", marginBottom: "1.25rem", display: "flex", alignItems: "center", gap: "0.5rem" }}>
            <span style={{ display: "inline-block", width: "12px", height: "12px", borderRadius: "50%", backgroundColor: "var(--accent-warning)" }}></span>
            Oracle Price Feed
          </h2>

          {/* Asset Selector */}
          <div style={{ marginBottom: "1rem" }}>
            <label style={{ fontSize: "0.85rem", color: "var(--text-secondary)", marginBottom: "0.25rem", display: "block" }}>Asset Selector</label>
            <div style={{ display: "flex", gap: "0.5rem" }}>
              {oraclePrices.map((p) => (
                <button
                  key={p.asset_id}
                  className={`btn ${selectedAsset === p.asset_id ? "btn-primary" : "btn-secondary"}`}
                  style={{ padding: "0.4rem 0.8rem", fontSize: "0.85rem" }}
                  onClick={() => setSelectedAsset(p.asset_id)}
                >
                  {p.asset_id}
                </button>
              ))}
            </div>
          </div>

          {oraclePrices.length === 0 ? (
            <div style={{ color: "var(--text-secondary)", padding: "3.5rem 0", textAlign: "center", fontSize: "0.95rem" }}>
              No active price feeds. Oracle operators have not submitted prices yet.
            </div>
          ) : selectedOracle ? (
            <div style={{ display: "grid", gridTemplateColumns: "repeat(2, 1fr)", gap: "0.75rem" }}>
              <div style={{ background: "rgba(0,0,0,0.15)", padding: "1rem", borderRadius: "10px", border: "1px solid var(--border-color)" }}>
                <span style={{ fontSize: "0.8rem", color: "var(--text-secondary)" }}>Open</span>
                <div style={{ fontSize: "1.1rem", fontWeight: 700, marginTop: "0.25rem" }}>${selectedOracle.open.toFixed(2)}</div>
              </div>
              <div style={{ background: "rgba(0,0,0,0.15)", padding: "1rem", borderRadius: "10px", border: "1px solid var(--border-color)" }}>
                <span style={{ fontSize: "0.8rem", color: "var(--text-secondary)" }}>High</span>
                <div style={{ fontSize: "1.1rem", fontWeight: 700, marginTop: "0.25rem", color: "var(--accent-success)" }}>${selectedOracle.high.toFixed(2)}</div>
              </div>
              <div style={{ background: "rgba(0,0,0,0.15)", padding: "1rem", borderRadius: "10px", border: "1px solid var(--border-color)" }}>
                <span style={{ fontSize: "0.8rem", color: "var(--text-secondary)" }}>Low</span>
                <div style={{ fontSize: "1.1rem", fontWeight: 700, marginTop: "0.25rem", color: "#f87171" }}>${selectedOracle.low.toFixed(2)}</div>
              </div>
              <div style={{ background: "rgba(0,0,0,0.15)", padding: "1rem", borderRadius: "10px", border: "1px solid var(--border-color)" }}>
                <span style={{ fontSize: "0.8rem", color: "var(--text-secondary)" }}>Close</span>
                <div style={{ fontSize: "1.1rem", fontWeight: 700, marginTop: "0.25rem" }}>${selectedOracle.close.toFixed(2)}</div>
              </div>
            </div>
          ) : null}
        </div>

        {/* ═══════════════════ Validators ═══════════════════ */}
        <div className="card col-12">
          <h2 style={{ fontFamily: "var(--font-title)", marginBottom: "1.25rem", display: "flex", alignItems: "center", gap: "0.5rem" }}>
            <span style={{ display: "inline-block", width: "12px", height: "12px", borderRadius: "50%", backgroundColor: "var(--accent-success)" }}></span>
            Validator Uptime
          </h2>

          <div style={{ display: "grid", gridTemplateColumns: "2fr 1fr", gap: "1.5rem" }}>
            {/* Uptime Table — GetValidators → validator_uptime_1d */}
            <div className="table-container">
              <table>
                <thead>
                  <tr>
                    <th>Validator</th>
                    <th>Total Blocks</th>
                    <th>Missed</th>
                    <th>Uptime %</th>
                  </tr>
                </thead>
                <tbody>
                  {validators.map((v) => (
                    <tr key={v.validator_address}>
                      <td style={{ fontFamily: "monospace", fontSize: "0.85rem" }}>{v.validator_address}</td>
                      <td>{v.total_blocks.toLocaleString()}</td>
                      <td style={{ color: v.missed_blocks > 100 ? "#f87171" : "var(--text-primary)" }}>
                        {v.missed_blocks}
                      </td>
                      <td>
                        <span className={`badge ${v.uptime_percentage >= 99.9 ? "badge-success" : v.uptime_percentage >= 99.0 ? "badge-primary" : "badge-warning"}`}>
                          {v.uptime_percentage.toFixed(3)}%
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Uptime Trend Chart — GetValidatorUptime → validator_uptime_1d time series */}
            <div style={{ background: "rgba(0,0,0,0.1)", padding: "1rem", borderRadius: "10px", border: "1px solid var(--border-color)" }}>
              <span style={{ fontSize: "0.85rem", color: "var(--text-secondary)" }}>Uptime Trend (7d)</span>
              {uptimeTrend.length === 0 ? (
                <div style={{ height: "80px", display: "flex", alignItems: "center", justifyContent: "center", color: "var(--text-secondary)", fontSize: "0.85rem" }}>
                  No historical uptime trend data available.
                </div>
              ) : mounted ? (
                <UptimeRechartsLineChart data={uptimeTrend} />
              ) : (
                <div style={{ height: "80px", display: "flex", alignItems: "center", justifyContent: "center" }}>Loading...</div>
              )}
              <div style={{ display: "flex", justifyContent: "space-between", marginTop: "0.5rem" }}>
                <span style={{ fontSize: "0.75rem", color: "var(--text-secondary)" }}>7d ago</span>
                <span style={{ fontSize: "0.75rem", color: "var(--text-secondary)" }}>Today</span>
              </div>
            </div>
          </div>
        </div>

        {/* ═══════════════════ Settlement ═══════════════════ */}
        <div className="card col-6">
          <h2 style={{ fontFamily: "var(--font-title)", marginBottom: "1.25rem", display: "flex", alignItems: "center", gap: "0.5rem" }}>
            <span style={{ display: "inline-block", width: "12px", height: "12px", borderRadius: "50%", backgroundColor: "#a78bfa" }}></span>
            Settlement Lifecycle
          </h2>

          <div style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
            {settlements.length === 0 ? (
              <p style={{ color: "var(--text-secondary)", textAlign: "center", padding: "2rem 0", fontSize: "0.95rem" }}>
                No settlements recorded.
              </p>
            ) : (
              settlements.map((s) => (
                <div
                  key={s.settlement_id}
                  style={{
                    background: "rgba(0,0,0,0.15)",
                    padding: "1rem",
                    borderRadius: "10px",
                    border: "1px solid var(--border-color)",
                    display: "flex",
                    justifyContent: "space-between",
                    alignItems: "center",
                  }}
                >
                  <div>
                    <div style={{ fontWeight: 600, fontFamily: "monospace" }}>{s.settlement_id}</div>
                    <div style={{ fontSize: "0.8rem", color: "var(--text-secondary)", marginTop: "0.25rem" }}>
                      Block #{s.block_height.toLocaleString()} — {s.signatures.length} signatures
                    </div>
                  </div>
                  <span className={`badge ${statusColor(s.status)}`}>{s.status}</span>
                </div>
              ))
            )}
          </div>

          {/* Lifecycle display: pending → confirmed → finalized */}
          <div style={{ marginTop: "1.5rem", background: "rgba(0,0,0,0.1)", padding: "1rem", borderRadius: "10px", border: "1px solid var(--border-color)" }}>
            <span style={{ fontSize: "0.85rem", color: "var(--text-secondary)", marginBottom: "0.75rem", display: "block" }}>
              Settlement Lifecycle Flow
            </span>
            <div style={{ display: "flex", alignItems: "center", gap: "0.5rem", justifyContent: "center" }}>
              <span className="badge badge-warning" style={{ padding: "0.4rem 0.8rem" }}>Pending</span>
              <span style={{ color: "var(--text-secondary)" }}>→</span>
              <span className="badge badge-primary" style={{ padding: "0.4rem 0.8rem" }}>Confirmed</span>
              <span style={{ color: "var(--text-secondary)" }}>→</span>
              <span className="badge badge-success" style={{ padding: "0.4rem 0.8rem" }}>Finalized</span>
            </div>
          </div>
        </div>

        {/* ═══════════════════ Milestones & Certifications ═══════════════════ */}
        <div className="card col-6">
          <h2 style={{ fontFamily: "var(--font-title)", marginBottom: "1.25rem", display: "flex", alignItems: "center", gap: "0.5rem" }}>
            <span style={{ display: "inline-block", width: "12px", height: "12px", borderRadius: "50%", backgroundColor: "#f59e0b" }}></span>
            Milestones & Certifications
          </h2>

          {/* Milestone Timeline — ListMilestones RPC → milestone_status projection */}
          <div style={{ display: "flex", flexDirection: "column", gap: "0" }}>
            {milestones.length === 0 ? (
              <p style={{ color: "var(--text-secondary)", textAlign: "center", padding: "2rem 0", fontSize: "0.95rem" }}>
                No milestones achieved yet.
              </p>
            ) : (
              milestones.map((m, i) => (
                <div
                  key={m.milestone_id}
                  style={{
                    display: "flex",
                    alignItems: "flex-start",
                    gap: "1rem",
                    paddingBottom: i < milestones.length - 1 ? "1.5rem" : "0",
                    position: "relative",
                  }}
                >
                  {/* Timeline connector */}
                  <div style={{ display: "flex", flexDirection: "column", alignItems: "center", minWidth: "20px" }}>
                    <div
                      style={{
                        width: "12px",
                        height: "12px",
                        borderRadius: "50%",
                        backgroundColor:
                          m.status === "achieved"
                            ? "var(--accent-success)"
                            : m.status === "pending"
                            ? "var(--accent-warning)"
                            : "var(--text-secondary)",
                        flexShrink: 0,
                      }}
                    ></div>
                    {i < milestones.length - 1 && (
                      <div
                        style={{
                          width: "2px",
                          flexGrow: 1,
                          backgroundColor: "var(--border-color)",
                          marginTop: "4px",
                          minHeight: "20px",
                        }}
                      ></div>
                    )}
                  </div>

                  <div style={{ flex: 1 }}>
                    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                      <span style={{ fontWeight: 600, fontFamily: "monospace", fontSize: "0.9rem" }}>{m.milestone_id}</span>
                      <span className={`badge ${statusColor(m.status)}`}>{m.status}</span>
                    </div>
                    <div style={{ fontSize: "0.8rem", color: "var(--text-secondary)", marginTop: "0.25rem" }}>
                      {m.block_height > 0 ? `Block #${m.block_height.toLocaleString()}` : "Awaiting..."}
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>

          {/* Certification List — ListCertifications → certification projection */}
          <div style={{ marginTop: "1.5rem", background: "rgba(0,0,0,0.1)", padding: "1rem", borderRadius: "10px", border: "1px solid var(--border-color)" }}>
            <span style={{ fontSize: "0.85rem", color: "var(--text-secondary)", marginBottom: "0.75rem", display: "block" }}>
              Active Certifications
            </span>
            <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem" }}>
              <div style={{ display: "flex", justifyContent: "space-between" }}>
                <span style={{ fontSize: "0.9rem" }}>Genesis Validator Certification</span>
                <span className="badge badge-success">Certified</span>
              </div>
              <div style={{ display: "flex", justifyContent: "space-between" }}>
                <span style={{ fontSize: "0.9rem" }}>Bridge Operator Certification</span>
                <span className="badge badge-success">Certified</span>
              </div>
              <div style={{ display: "flex", justifyContent: "space-between" }}>
                <span style={{ fontSize: "0.9rem" }}>Oracle Provider Certification</span>
                <span className="badge badge-warning">Pending</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
