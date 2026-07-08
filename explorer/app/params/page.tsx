"use client";

import React, { useState, useEffect } from "react";
import Link from "next/link";
import { Settings, ShieldAlert, Coins, Landmark, Calendar, RefreshCw, BarChart2, Layers, Activity } from "lucide-react";
import { useWalletStore } from "@/store/wallet";

interface ParamItem {
  name: string;
  value: string;
  type: string;
  description: string;
}

export default function PARAMSPage() {
  const { walletType, connected, address, connectWallet, disconnectWallet } = useWalletStore();
  const [activeTab, setActiveTab] = useState<"staking" | "gov" | "slashing" | "bank" | "bridge" | "oracle" | "milestone" | "feemarket">("staking");
  const [loading, setLoading] = useState(false);

  const initialStaking: ParamItem[] = [
    { name: "Unbonding Time", value: "21 days (1,814,400s)", type: "Duration", description: "Time duration after requesting undelegation before funds are released." },
    { name: "Max Validators", value: "30", type: "Integer", description: "Maximum number of active validators participating in consensus." },
    { name: "Min Self Delegation", value: "1,000,000 uSLT", type: "Coin", description: "Minimum amount of staking tokens a validator must delegate to themselves." },
    { name: "Bond Denom", value: "uSLT", type: "String", description: "The native denomination used for staking and governance voting power." },
    { name: "Historical Entries", value: "10,000", type: "Integer", description: "Number of historical info entries stored in the state history." },
  ];

  const initialGov: ParamItem[] = [
    { name: "Min Deposit", value: "10,000,000 uSLT", type: "Coin", description: "Minimum deposit required to push a proposal into the voting phase." },
    { name: "Max Deposit Period", value: "48 hours (172,800s)", type: "Duration", description: "Maximum time a proposal can remain in deposit phase to meet min deposit." },
    { name: "Voting Period", value: "24 hours (86,400s)", type: "Duration", description: "Length of the voting phase for active proposals." },
    { name: "Quorum", value: "33.40%", type: "Percentage", description: "Minimum percentage of total bonded voting power required to vote for proposal validity." },
    { name: "Threshold", value: "50.00%", type: "Percentage", description: "Minimum ratio of Yes votes (excluding Abstain) required for a proposal to pass." },
    { name: "Veto Threshold", value: "33.40%", type: "Percentage", description: "Ratio of NoWithVeto votes required to unilaterally reject a proposal." },
  ];

  const initialSlashing: ParamItem[] = [
    { name: "Signed Blocks Window", value: "10,000 blocks", type: "Integer", description: "Rolling block window used to measure validator uptime." },
    { name: "Min Signed Per Window", value: "50.00%", type: "Percentage", description: "Minimum ratio of blocks a validator must sign within the rolling window to avoid jail." },
    { name: "Downtime Jail Duration", value: "600 seconds", type: "Duration", description: "Minimum time a validator remains jailed after being jailed for downtime." },
    { name: "Slash Fraction Double Sign", value: "5.00%", type: "Percentage", description: "Percentage of stake slashed if a validator double signs a block." },
    { name: "Slash Fraction Downtime", value: "0.01%", type: "Percentage", description: "Percentage of stake slashed if a validator goes offline beyond the window limit." },
  ];

  const initialBank: ParamItem[] = [
    { name: "Send Enabled", value: "true", type: "Boolean", description: "Global permission enabling bank transfers between accounts." },
    { name: "Default Send Restricted", value: "false", type: "Boolean", description: "If true, transfers require explicit account-level send permissions." },
    { name: "Total Supply Limit", value: "1,000,000,000 SOV", type: "Coin", description: "Hardcap supply limit for native staking token." },
  ];

  const initialBridge: ParamItem[] = [
    { name: "Bridge Fee", value: "0.05%", type: "Percentage", description: "Standard transfer fee applied to bridging transactions." },
    { name: "Escrow Wallet Address", value: "sovereign1bridgeescrowaddress", type: "Address", description: "System account where bridged assets are escrowed." },
    { name: "Daily Limit Cap", value: "10,000,000 SOV", type: "Coin", description: "Maximum value of funds that can be bridged in a single 24-hour cycle." },
  ];

  const initialOracle: ParamItem[] = [
    { name: "Update Threshold", value: "10 blocks", type: "Integer", description: "Maximum number of blocks allowed between price feed updates." },
    { name: "Feeder SLA Score", value: "95.00%", type: "Percentage", description: "Required Service Level Agreement score for registered price feeders." },
    { name: "Price Deviation Limit", value: "2.00%", type: "Percentage", description: "Maximum price fluctuation allowed in a single round before rejection." },
  ];

  const initialMilestone: ParamItem[] = [
    { name: "Active Validator Limit", value: "30", type: "Integer", description: "Number of active validators required to commit checkpoint milestones." },
    { name: "Checkpoint Frequency", value: "500 blocks", type: "Integer", description: "Block interval at which cryptographic consensus checkpoints are finalized." },
    { name: "Upgrade Delay Period", value: "48 hours (172,800s)", type: "Duration", description: "Mandatory time lock before approved software upgrades are activated." },
  ];

  const initialFeemarket: ParamItem[] = [
    { name: "Base Fee Denominator", value: "8", type: "Integer", description: "Divider determining block-by-block base fee volatility speed." },
    { name: "Target Block Gas Limit", value: "15,000,000 gas", type: "Integer", description: "The targeted optimal block size in gas units." },
    { name: "Min Gas Price", value: "0.0025 uSLT/gas", type: "Coin", description: "Minimum price threshold for transaction execution." },
  ];

  const [staking, setStaking] = useState<ParamItem[]>(initialStaking);
  const [gov, setGov] = useState<ParamItem[]>(initialGov);
  const [slashing, setSlashing] = useState<ParamItem[]>(initialSlashing);
  const [bank, setBank] = useState<ParamItem[]>(initialBank);
  const [bridge, setBridge] = useState<ParamItem[]>(initialBridge);
  const [oracle, setOracle] = useState<ParamItem[]>(initialOracle);
  const [milestone, setMilestone] = useState<ParamItem[]>(initialMilestone);
  const [feemarket, setFeemarket] = useState<ParamItem[]>(initialFeemarket);

  const fetchParams = async () => {
    setLoading(true);
    const restBase = process.env.NEXT_PUBLIC_COSMOS_REST || "http://localhost:1317";
    try {
      // Staking
      const stakingResp = await fetch(`${restBase}/cosmos/staking/v1beta1/params`);
      if (stakingResp.ok) {
        const data = await stakingResp.json();
        const p = data.params;
        if (p) {
          setStaking([
            { name: "Unbonding Time", value: p.unbonding_time || "21 days (1,814,400s)", type: "Duration", description: "Time duration after requesting undelegation before funds are released." },
            { name: "Max Validators", value: String(p.max_validators || "30"), type: "Integer", description: "Maximum number of active validators participating in consensus." },
            { name: "Min Self Delegation", value: p.min_self_delegation ? `${p.min_self_delegation} uSLT` : "1,000,000 uSLT", type: "Coin", description: "Minimum amount of staking tokens a validator must delegate to themselves." },
            { name: "Bond Denom", value: p.bond_denom || "uSLT", type: "String", description: "The native denomination used for staking and governance voting power." },
            { name: "Historical Entries", value: String(p.historical_entries || "10,000"), type: "Integer", description: "Number of historical info entries stored in the state history." },
          ]);
        }
      }

      // Gov
      const votingResp = await fetch(`${restBase}/cosmos/gov/v1/params/voting`);
      const depositResp = await fetch(`${restBase}/cosmos/gov/v1/params/deposit`);
      const tallyingResp = await fetch(`${restBase}/cosmos/gov/v1/params/tallying`);
      if (votingResp.ok && depositResp.ok && tallyingResp.ok) {
        const votingData = await votingResp.json();
        const depositData = await depositResp.json();
        const tallyingData = await tallyingResp.json();

        const vP = votingData.params || {};
        const dP = depositData.params || {};
        const tP = tallyingData.params || {};

        setGov([
          { name: "Min Deposit", value: dP.min_deposit && dP.min_deposit.length > 0 ? `${dP.min_deposit[0].amount} ${dP.min_deposit[0].denom}` : "10,000,000 uSLT", type: "Coin", description: "Minimum deposit required to push a proposal into the voting phase." },
          { name: "Max Deposit Period", value: dP.max_deposit_period || "48 hours (172,800s)", type: "Duration", description: "Maximum time a proposal can remain in deposit phase to meet min deposit." },
          { name: "Voting Period", value: vP.voting_period || "24 hours (86,400s)", type: "Duration", description: "Length of the voting phase for active proposals." },
          { name: "Quorum", value: tP.quorum ? `${(parseFloat(tP.quorum) * 100).toFixed(2)}%` : "33.40%", type: "Percentage", description: "Minimum percentage of total bonded voting power required to vote for proposal validity." },
          { name: "Threshold", value: tP.threshold ? `${(parseFloat(tP.threshold) * 100).toFixed(2)}%` : "50.00%", type: "Percentage", description: "Minimum ratio of Yes votes (excluding Abstain) required for a proposal to pass." },
          { name: "Veto Threshold", value: tP.veto_threshold ? `${(parseFloat(tP.veto_threshold) * 100).toFixed(2)}%` : "33.40%", type: "Percentage", description: "Ratio of NoWithVeto votes required to unilaterally reject a proposal." },
        ]);
      }

      // Slashing
      const slashingResp = await fetch(`${restBase}/cosmos/slashing/v1beta1/params`);
      if (slashingResp.ok) {
        const data = await slashingResp.json();
        const p = data.params;
        if (p) {
          setSlashing([
            { name: "Signed Blocks Window", value: p.signed_blocks_window || "10,000 blocks", type: "Integer", description: "Rolling block window used to measure validator uptime." },
            { name: "Min Signed Per Window", value: p.min_signed_per_window ? `${(parseFloat(p.min_signed_per_window) * 100).toFixed(2)}%` : "50.00%", type: "Percentage", description: "Minimum ratio of blocks a validator must sign within the rolling window to avoid jail." },
            { name: "Downtime Jail Duration", value: p.downtime_jail_duration || "600 seconds", type: "Duration", description: "Minimum time a validator remains jailed after being jailed for downtime." },
            { name: "Slash Fraction Double Sign", value: p.slash_fraction_double_sign ? `${(parseFloat(p.slash_fraction_double_sign) * 100).toFixed(2)}%` : "5.00%", type: "Percentage", description: "Percentage of stake slashed if a validator double signs a block." },
            { name: "Slash Fraction Downtime", value: p.slash_fraction_downtime ? `${(parseFloat(p.slash_fraction_downtime) * 100).toFixed(2)}%` : "0.01%", type: "Percentage", description: "Percentage of stake slashed if a validator goes offline beyond the window limit." },
          ]);
        }
      }

      // Bank
      const bankResp = await fetch(`${restBase}/cosmos/bank/v1beta1/params`);
      if (bankResp.ok) {
        const data = await bankResp.json();
        const p = data.params;
        if (p) {
          setBank([
            { name: "Send Enabled", value: String(p.send_enabled !== false), type: "Boolean", description: "Global permission enabling bank transfers between accounts." },
            { name: "Default Send Restricted", value: p.default_send_enabled === false ? "true" : "false", type: "Boolean", description: "If true, transfers require explicit account-level send permissions." },
            { name: "Total Supply Limit", value: "1,000,000,000 SOV", type: "Coin", description: "Hardcap supply limit for native staking token." },
          ]);
        }
      }
    } catch (err) {
      console.warn("Failed to fetch dynamic parameters, using default mocks.", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchParams();
  }, []);

  const handleRefresh = () => {
    fetchParams();
  };

  const getParams = () => {
    switch (activeTab) {
      case "staking": return staking;
      case "gov": return gov;
      case "slashing": return slashing;
      case "bank": return bank;
      case "bridge": return bridge;
      case "oracle": return oracle;
      case "milestone": return milestone;
      case "feemarket": return feemarket;
    }
  };

  const getTabIcon = (tab: typeof activeTab) => {
    switch (tab) {
      case "staking": return <BarChart2 className="h-4 w-4" />;
      case "gov": return <Landmark className="h-4 w-4" />;
      case "slashing": return <ShieldAlert className="h-4 w-4" />;
      case "bank": return <Coins className="h-4 w-4" />;
      case "bridge": return <Layers className="h-4 w-4" />;
      case "oracle": return <Activity className="h-4 w-4" />;
      case "milestone": return <Calendar className="h-4 w-4" />;
      case "feemarket": return <Settings className="h-4 w-4" />;
    }
  };

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white">Params</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4 flex justify-between items-center">
        <div className="flex items-center space-x-3">
          <Settings className="text-blue-500 h-8 w-8" />
          <div>
            <h1 className="text-3xl font-bold tracking-tight text-white">Chain Parameters</h1>
            <p className="text-gray-400 mt-1">Live protocol parameters and settings for Sovereign L1 modules</p>
          </div>
        </div>

        <button 
          onClick={handleRefresh}
          className="p-2 bg-gray-900 hover:bg-gray-800 border border-gray-800 rounded-lg text-gray-400 hover:text-white transition"
          title="Reload parameters"
          disabled={loading}
        >
          <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin text-blue-500" : ""}`} />
        </button>
      </div>

      {/* Tabs */}
      <div className="flex space-x-2 border-b border-gray-900 pb-px overflow-x-auto scrollbar-thin">
        {(["staking", "gov", "slashing", "bank", "bridge", "oracle", "milestone", "feemarket"] as const).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2.5 text-sm font-medium capitalize border-b-2 transition -mb-px flex items-center space-x-2 ${
              activeTab === tab
                ? "border-blue-500 text-white"
                : "border-transparent text-gray-400 hover:text-gray-200"
            }`}
          >
            {getTabIcon(tab)}
            <span>{tab === "gov" ? "Governance" : tab}</span>
          </button>
        ))}
      </div>

      {/* Grid Content */}
      {loading ? (
        <div className="flex justify-center items-center py-20">
          <RefreshCw className="h-8 w-8 text-blue-500 animate-spin" />
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {getParams().map((param, idx) => (
            <div key={idx} className="bg-gray-950 border border-gray-900 rounded-xl p-5 hover:border-gray-800 transition flex flex-col justify-between space-y-4 shadow-lg">
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <h3 className="font-semibold text-white text-sm">{param.name}</h3>
                  <span className="text-[10px] px-1.5 py-0.5 bg-gray-900 border border-gray-850 rounded-full font-mono text-gray-500 uppercase">
                    {param.type}
                  </span>
                </div>
                <p className="text-xs text-gray-400 leading-normal">
                  {param.description}
                </p>
              </div>

              <div className="pt-3 border-t border-gray-900">
                <span className="font-mono text-base font-bold text-blue-400 select-all">
                  {param.value}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
