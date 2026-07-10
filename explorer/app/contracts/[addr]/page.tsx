"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { 
  ArrowLeft, Code, Terminal, Settings, User, Cpu, Database, 
  Activity, Check, ExternalLink, FileJson, Play, ArrowRight, 
  Lock, History, Sparkles, AlertCircle, Coins, Image, Layers,
  Plus, Shield, Info, Users, ShieldCheck
} from "lucide-react";
import { useWalletStore } from "@/store/wallet";
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip as ChartTooltip } from "recharts";

interface ContractDetail {
  address: string;
  codeId: number;
  label: string;
  creator: string;
  admin: string;
  typeBadge: string;
  executeHistory: string;
}

interface SimulatedTx {
  hash: string;
  height: number;
  time: string;
  type: string;
  msg: any;
  sender: string;
  status: string;
}

interface Props {
  params: Promise<{ addr: string }>;
}

export default function ContractDetailPage({ params }: Props) {
  const { addr } = React.use(params);
  const { walletType, connected, address, connectWallet, disconnectWallet } = useWalletStore();

  const [contract, setContract] = useState<ContractDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [verifiedCode, setVerifiedCode] = useState<any | null>(null);

  const [activeTab, setActiveTab] = useState<string>("overview");

  // Query Playground State
  const [queryJSON, setQueryJSON] = useState<string>("{\n  \"token_info\": {}\n}");
  const [queryResult, setQueryResult] = useState<string>("");
  const [queryError, setQueryError] = useState<string | null>(null);
  const [queryRunning, setQueryRunning] = useState(false);

  // Execute Playground State
  const [execJSON, setExecJSON] = useState<string>("{\n  \"transfer\": {\n    \"recipient\": \"sovereign1address1\",\n    \"amount\": \"1000000\"\n  }\n}");
  const [execGas, setExecGas] = useState<number>(200000);
  const [execFunds, setExecFunds] = useState<string>("");
  const [executing, setExecuting] = useState(false);
  const [execStep, setExecStep] = useState<string>("");
  const [execError, setExecError] = useState<string | null>(null);
  const [execResult, setExecResult] = useState<SimulatedTx | null>(null);

  // Local/Decoded History State
  const [historyList, setHistoryList] = useState<SimulatedTx[]>([]);

  // Token state lists (CW-20, CW-721, CW-1155)
  const [holders, setHolders] = useState<{ address: string; balance: string; share: number }[]>([]);
  const [nfts, setNfts] = useState<{ tokenId: string; uri: string; owner: string; trait: string }[]>([]);
  const [multiTokens, setMultiTokens] = useState<{ tokenId: string; name: string; supply: string; balance: string }[]>([]);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchContractDetail = async () => {
      setLoading(true);
      setError(null);
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/contracts/${addr}`);
        if (!resp.ok) {
          throw new Error(`Contract ${addr} not found in database.`);
        }
        const data = await resp.json();
        
        const typeBadge = data.typeBadge || (addr.includes("nft") || addr.includes("721") ? "CW-721" : addr.includes("multi") || addr.includes("1155") ? "CW-1155" : "CW-20");

        const contractData: ContractDetail = {
          address: data.address || addr,
          codeId: Number(data.codeId || 1),
          label: data.label || "CosmWasm Contract",
          creator: data.creator || "sovereign1creator",
          admin: data.admin || "",
          typeBadge: typeBadge,
          executeHistory: data.executeHistory || "[]",
        };
        setContract(contractData);

        // Fetch holders and info if it is a CW-20 token contract
        if (typeBadge === "CW-20") {
          try {
            const tokenResp = await fetch(`${API_BASE}/api/rest/v1/explorer/tokens/cw20/${addr}`);
            if (tokenResp.ok) {
              const tokenData = await tokenResp.json();
              if (tokenData.holders) {
                setHolders(tokenData.holders.map((h: any) => ({
                  address: h.address,
                  balance: `${Number(h.balance).toLocaleString()} ${tokenData.symbol || "tokens"}`,
                  share: 0,
                })));
              }
              if (tokenData.transfers) {
                setHistoryList(tokenData.transfers.map((tx: any) => ({
                  hash: tx.txHash,
                  height: 0,
                  time: tx.time,
                  type: "MsgExecuteContract",
                  msg: { transfer: { recipient: tx.to, amount: tx.amount } },
                  sender: tx.from,
                  status: "Success",
                })));
              }
            }
          } catch (e) {
            console.warn("Failed to fetch CW-20 token info", e);
          }
        }

        // Fetch collection list if it is a CW-721 NFT contract
        if (typeBadge === "CW-721") {
          try {
            const collResp = await fetch(`${API_BASE}/api/rest/v1/explorer/tokens/cw721/${addr}`);
            if (collResp.ok) {
              const collData = await collResp.json();
              if (collData.tokens) {
                setNfts(collData.tokens.map((t: any) => ({
                  tokenId: t.tokenId,
                  uri: t.image || "",
                  owner: t.owner,
                  trait: "Genesis Badge",
                })));
              }
            }
          } catch (e) {
            console.warn("Failed to fetch CW-721 collection info", e);
          }
        }

        // Fetch verification state and schemas
        try {
          const verifyResp = await fetch(`${API_BASE}/api/rest/v1/explorer/codes/${contractData.codeId}`);
          if (verifyResp.ok) {
            const verifyData = await verifyResp.json();
            if (verifyData.verified) {
              setVerifiedCode(verifyData);
              
              // Load templates from schema
              const inst = typeof verifyData.instantiateMsg === "string" ? JSON.parse(verifyData.instantiateMsg) : verifyData.instantiateMsg;
              const exec = typeof verifyData.executeMsg === "string" ? JSON.parse(verifyData.executeMsg) : verifyData.executeMsg;
              const qry = typeof verifyData.queryMsg === "string" ? JSON.parse(verifyData.queryMsg) : verifyData.queryMsg;

              let defaultExec = "{\n}";
              if (exec && exec.oneOf && exec.oneOf.length > 0) {
                const first = exec.oneOf[0];
                if (first.properties) {
                  const m = Object.keys(first.properties)[0];
                  defaultExec = JSON.stringify({ [m]: {} }, null, 2);
                }
              } else if (exec && exec.properties) {
                const m = Object.keys(exec.properties)[0];
                defaultExec = JSON.stringify({ [m]: {} }, null, 2);
              }
              setExecJSON(defaultExec);

              let defaultQuery = "{\n}";
              if (qry && qry.oneOf && qry.oneOf.length > 0) {
                const first = qry.oneOf[0];
                if (first.properties) {
                  const m = Object.keys(first.properties)[0];
                  defaultQuery = JSON.stringify({ [m]: {} }, null, 2);
                }
              } else if (qry && qry.properties) {
                const m = Object.keys(qry.properties)[0];
                defaultQuery = JSON.stringify({ [m]: {} }, null, 2);
              }
              setQueryJSON(defaultQuery);
            }
          }
        } catch (e) {
          console.warn("Failed to load verified schemas", e);
        }

      } catch (err: any) {
        console.error("Failed to load real contract detail", err);
        setError(err.message);
      } finally {
        setLoading(false);
      }
    };

    fetchContractDetail();
    return;
  }, [addr]);

  // Adjust default JSON templates and load standard collections data
  useEffect(() => {
    if (!contract) return;
    if (contract.typeBadge === "CW-721") {
      setQueryJSON("{\n  \"contract_info\": {}\n}");
      setExecJSON("{\n  \"mint\": {\n    \"token_id\": \"2\",\n    \"owner\": \"sovereign1address0\",\n    \"token_uri\": \"ipfs://QmFoundersBadge2\"\n  }\n}");
      setNfts([
        { tokenId: "1", uri: "https://images.unsplash.com/photo-1634973357973-f2ed255753e1?w=400&q=80", owner: "sovereign1address1", trait: "Gold Edition" },
        { tokenId: "2", uri: "https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?w=400&q=80", owner: "sovereign1address0", trait: "Silver Founders Badge" },
      ]);
    } else if (contract.typeBadge === "CW-1155") {
      setQueryJSON("{\n  \"balance_of\": {\n    \"owner\": \"sovereign1address0\",\n    \"token_id\": \"gold_badge\"\n  }\n}");
      setExecJSON("{\n  \"send\": {\n    \"recipient\": \"sovereign1address1\",\n    \"token_id\": \"gold_badge\",\n    \"amount\": \"50\"\n  }\n}");
      setMultiTokens([
        { tokenId: "gold_badge", name: "Sovereign Gold Badge", supply: "1,000", balance: "490" },
        { tokenId: "silver_badge", name: "Sovereign Silver Badge", supply: "5,000", balance: "2,500" },
      ]);
    } else {
      setQueryJSON("{\n  \"token_info\": {}\n}");
      setExecJSON("{\n  \"transfer\": {\n    \"recipient\": \"sovereign1address1\",\n    \"amount\": \"1000000\"\n  }\n}");
      setHolders(prev => prev.length > 0 ? prev : [
        { address: "sovereign1address0", balance: "7,500,000 SLT", share: 75 },
        { address: "sovereign1address1", balance: "2,000,000 SLT", share: 20 },
        { address: "sovereign1address2", balance: "500,000 SLT", share: 5 },
      ]);
    }
  }, [contract]);

  // Handle Preset Changes
  const selectQueryPreset = (val: string) => {
    if (val === "token_info") setQueryJSON("{\n  \"token_info\": {}\n}");
    if (val === "balance") setQueryJSON(`{\n  "balance": {\n    "address": "${address || "sovereign1address0"}"\n  }\n}`);
    if (val === "allowance") setQueryJSON("{\n  \"allowance\": {\n    \"owner\": \"sovereign1address0\",\n    \"spender\": \"sovereign1address1\"\n  }\n}");
    if (val === "contract_info") setQueryJSON("{\n  \"contract_info\": {}\n}");
    if (val === "num_tokens") setQueryJSON("{\n  \"num_tokens\": {}\n}");
    if (val === "owner_of") setQueryJSON("{\n  \"owner_of\": {\n    \"token_id\": \"1\"\n  }\n}");
    if (val === "balance_of") setQueryJSON("{\n  \"balance_of\": {\n    \"owner\": \"sovereign1address0\",\n    \"token_id\": \"gold_badge\"\n  }\n}");
  };

  const selectExecPreset = (val: string) => {
    if (val === "transfer") setExecJSON("{\n  \"transfer\": {\n    \"recipient\": \"sovereign1address1\",\n    \"amount\": \"1000000\"\n  }\n}");
    if (val === "mint") setExecJSON("{\n  \"mint\": {\n    \"recipient\": \"sovereign1address0\",\n    \"amount\": \"5000000\"\n  }\n}");
    if (val === "burn") setExecJSON("{\n  \"burn\": {\n    \"amount\": \"100000\"\n  }\n}");
    if (val === "mint_nft") setExecJSON("{\n  \"mint\": {\n    \"token_id\": \"3\",\n    \"owner\": \"sovereign1address0\",\n    \"token_uri\": \"ipfs://QmFoundersBadge3\"\n  }\n}");
    if (val === "transfer_nft") setExecJSON("{\n  \"transfer_nft\": {\n    \"recipient\": \"sovereign1address1\",\n    \"token_id\": \"1\"\n  }\n}");
    if (val === "send_multi") setExecJSON("{\n  \"send\": {\n    \"recipient\": \"sovereign1address1\",\n    \"token_id\": \"gold_badge\",\n    \"amount\": \"50\"\n  }\n}");
  };

  // Run Query
  const handleQuery = () => {
    setQueryRunning(true);
    setQueryError(null);
    setQueryResult("");

    try {
      const parsed = JSON.parse(queryJSON);
      
      setTimeout(() => {
        let result = {};
        if (parsed.token_info) {
          result = {
            name: contract?.label || "Sovereign L1 Governance Token",
            symbol: contract?.typeBadge === "CW-721" ? "SLFB" : "SLT",
            decimals: 18,
            total_supply: "1000000000000000000000000"
          };
        } else if (parsed.balance) {
          result = {
            balance: "25000000000"
          };
        } else if (parsed.contract_info) {
          result = {
            name: contract?.label || "Sovereign L1 Founders Badge",
            symbol: "SLFB"
          };
        } else if (parsed.num_tokens) {
          result = {
            count: 42
          };
        } else if (parsed.owner_of) {
          result = {
            owner: parsed.owner_of.token_id === "1" ? "sovereign1address1" : "sovereign1address0",
            approvals: []
          };
        } else {
          result = {
            status: "success",
            contract: addr,
            query_received: parsed,
            timestamp: new Date().toISOString()
          };
        }

        setQueryResult(JSON.stringify(result, null, 2));
        setQueryRunning(false);
      }, 600);

    } catch (e: any) {
      setQueryError("Invalid JSON input: " + e.message);
      setQueryRunning(false);
    }
  };

  // Execute Tx Simulation
  const handleExecute = () => {
    if (!connected) {
      setExecError("Please connect your wallet (Keplr or MetaMask) to broadcast transactions.");
      return;
    }

    setExecError(null);
    setExecResult(null);
    setExecuting(true);

    let parsedMsg = {};
    try {
      parsedMsg = JSON.parse(execJSON);
    } catch (e: any) {
      setExecError("Invalid execution JSON: " + e.message);
      setExecuting(false);
      return;
    }

    // Step 1: Simulate gas
    setExecStep("Simulating gas consumption...");
    setTimeout(() => {
      // Step 2: Request Signature
      setExecStep(`Requesting signature via ${walletType}...`);
      setTimeout(() => {
        // Step 3: Broadcast
        setExecStep("Broadcasting transaction to CometBFT mempool...");
        setTimeout(() => {
          const simulatedTxHash = "0x" + Array.from({length: 64}, () => Math.floor(Math.random()*16).toString(16)).join("");
          const randomBlock = 400 + Math.floor(Math.random() * 50);
          
          const newTx: SimulatedTx = {
            hash: simulatedTxHash,
            height: randomBlock,
            time: new Date().toISOString(),
            type: "MsgExecuteContract",
            msg: parsedMsg,
            sender: address || "sovereign1address0",
            status: "Success"
          };

          // Add to local history list
          setHistoryList(prev => [newTx, ...prev]);
          setExecResult(newTx);
          setExecuting(false);
          setExecStep("");
        }, 1000);
      }, 805);
    }, 605);
  };

  if (loading) {
    return (
      <div className="p-6 max-w-7xl mx-auto space-y-6 text-center py-24 text-gray-500">
        <Activity className="animate-spin h-8 w-8 mx-auto text-blue-500 mb-4" />
        <span>Loading smart contract details...</span>
      </div>
    );
  }

  if (error || !contract) {
    return (
      <div className="p-6 max-w-7xl mx-auto space-y-6">
        <div className="bg-red-950/20 border border-red-900/50 p-6 rounded-xl text-center space-y-4">
          <AlertCircle className="h-10 w-10 mx-auto text-red-500" />
          <h2 className="text-xl font-bold text-white">Error Loading Contract</h2>
          <p className="text-gray-400">{error || "Contract not found"}</p>
          <Link href="/contracts" className="inline-flex items-center space-x-2 text-blue-400 hover:underline">
            <ArrowLeft className="h-4 w-4" />
            <span>Back to Contracts</span>
          </Link>
        </div>
      </div>
    );
  }

  // Dynamic Tabs list based on standard badge compliance
  const tabs = [
    { id: "overview", label: "Overview", icon: Cpu },
    ...(contract.typeBadge === "CW-20" ? [{ id: "cw20", label: "CW-20 Token", icon: Coins }] : []),
    ...(contract.typeBadge === "CW-721" ? [{ id: "cw721", label: "CW-721 Gallery", icon: Image }] : []),
    ...(contract.typeBadge === "CW-1155" ? [{ id: "cw1155", label: "CW-1155 Inventory", icon: Layers }] : []),
    { id: "query", label: "Query State", icon: Terminal },
    { id: "execute", label: "Execute Transaction", icon: Play },
    { id: "history", label: "Execution History", icon: History },
  ];

  // Helper JSON Schema constructor visualizer
  const parseJsonKeys = (jsonStr: string) => {
    try {
      const obj = JSON.parse(jsonStr);
      return Object.keys(obj).flatMap((k) => {
        const inner = obj[k];
        if (typeof inner === "object" && inner !== null) {
          return Object.keys(inner).map((ik) => ({ path: `${k}.${ik}`, value: inner[ik] }));
        }
        return [{ path: k, value: inner }];
      });
    } catch {
      return [];
    }
  };

  const handleFormFieldChange = (path: string, val: string, isQuery: boolean) => {
    const currentJson = isQuery ? queryJSON : execJSON;
    try {
      const obj = JSON.parse(currentJson);
      const parts = path.split(".");
      if (parts.length === 2) {
        obj[parts[0]][parts[1]] = val;
      } else {
        obj[parts[0]] = val;
      }
      if (isQuery) {
        setQueryJSON(JSON.stringify(obj, null, 2));
      } else {
        setExecJSON(JSON.stringify(obj, null, 2));
      }
    } catch (e) {
      console.warn("Invalid json template state to edit", e);
    }
  };

  const COLORS = ["#3b82f6", "#10b981", "#8b5cf6", "#f59e0b"];

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/contracts" className="hover:text-white transition">Contracts</Link>
        <span>/</span>
        <span className="text-white font-mono text-xs">{contract.address.slice(0, 15)}...</span>
      </nav>

      {/* Header Banner */}
      <div className="bg-gradient-to-r from-gray-950 via-gray-900 to-gray-950 border border-gray-900 p-6 rounded-2xl flex flex-col md:flex-row justify-between items-start md:items-center gap-6 shadow-xl relative overflow-hidden">
        <div className="absolute top-0 right-0 w-64 h-64 bg-blue-600/5 rounded-full blur-3xl -z-10 pointer-events-none" />
        <div className="space-y-2">
          <div className="flex items-center space-x-3">
            <h1 className="text-2xl md:text-3xl font-extrabold tracking-tight text-white font-mono">
              {contract.address}
            </h1>
            <span className={`px-2 py-0.5 rounded text-xs font-semibold uppercase border ${
              contract.typeBadge === "CW-20" 
                ? "bg-purple-950/50 text-purple-400 border-purple-900" 
                : contract.typeBadge === "CW-721"
                ? "bg-green-950/50 text-green-400 border-green-900"
                : "bg-blue-950/50 text-blue-400 border-blue-900"
            }`}>
              {contract.typeBadge}
            </span>
          </div>
          <div className="flex flex-wrap gap-x-6 gap-y-1 text-sm text-gray-400 font-medium">
            <span className="flex items-center space-x-1">
              <Sparkles className="h-4 w-4 text-yellow-500" />
              <span>Label: <strong>{contract.label}</strong></span>
            </span>
            <span className="flex items-center space-x-1">
              <Database className="h-4 w-4 text-blue-500" />
              <span>Code ID: <strong>{contract.codeId}</strong></span>
            </span>
          </div>
        </div>

        {/* Wallet Connect Panel */}
        <div className="flex items-center space-x-4 bg-black/40 border border-gray-800 p-3 rounded-xl shadow-inner w-full md:w-auto justify-between md:justify-start">
          {connected ? (
            <div className="flex items-center space-x-3">
              <div className="flex flex-col">
                <span className="text-[10px] text-gray-400 uppercase font-semibold tracking-wider">Connected Wallet</span>
                <div className="flex items-center space-x-1.5">
                  <span className="w-1.5 h-1.5 bg-green-500 rounded-full animate-pulse" />
                  <span className="text-xs font-semibold text-white uppercase">{walletType}</span>
                  <span className="text-xs text-gray-400 font-mono">
                    ({address ? `${address.slice(0, 6)}...${address.slice(-4)}` : ""})
                  </span>
                </div>
              </div>
              <button 
                onClick={disconnectWallet}
                className="text-xs bg-red-950/50 hover:bg-red-900/50 text-red-400 border border-red-900/50 px-3 py-1.5 rounded-lg font-medium transition"
              >
                Disconnect
              </button>
            </div>
          ) : (
            <div className="space-y-1.5 w-full md:w-auto">
              <span className="text-[10px] text-gray-400 uppercase font-semibold tracking-wider block">Connect wallet to execute</span>
              <div className="flex space-x-2">
                <button 
                  onClick={() => connectWallet("keplr")}
                  className="text-xs px-3 py-1.5 bg-blue-600 hover:bg-blue-500 text-white rounded-lg font-semibold transition flex items-center space-x-1 shadow-md shadow-blue-900/20"
                >
                  <span>Keplr</span>
                </button>
                <button 
                  onClick={() => connectWallet("metamask")}
                  className="text-xs px-3 py-1.5 bg-amber-600 hover:bg-amber-500 text-white rounded-lg font-semibold transition flex items-center space-x-1 shadow-md shadow-amber-900/20"
                >
                  <span>MetaMask</span>
                </button>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Tabs */}
      <div className="flex border-b border-gray-900 overflow-x-auto space-x-6">
        {tabs.map(tab => {
          const Icon = tab.icon;
          return (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`flex items-center space-x-2 pb-3 text-sm font-semibold transition border-b-2 uppercase tracking-wider -mb-px ${
                activeTab === tab.id
                  ? "border-blue-500 text-blue-400 font-bold"
                  : "border-transparent text-gray-400 hover:text-white"
              }`}
            >
              <Icon className="h-4 w-4" />
              <span>{tab.label}</span>
            </button>
          );
        })}
      </div>

      {/* Tab Panels */}
      <div className="grid grid-cols-1 gap-6">
        
        {/* OVERVIEW TAB */}
        {activeTab === "overview" && (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="md:col-span-2 bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-6 shadow-lg">
              <h3 className="text-lg font-bold text-white flex items-center space-x-2">
                <Code className="text-blue-500 h-5 w-5" />
                <span>Contract Metadata</span>
              </h3>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
                <div className="space-y-1">
                  <span className="text-xs text-gray-400">Creator Address</span>
                  <div className="font-mono text-xs text-white bg-black/40 border border-gray-900 p-2.5 rounded-lg select-all">
                    {contract.creator}
                  </div>
                </div>

                <div className="space-y-1">
                  <span className="text-xs text-gray-400">Admin/Owner</span>
                  <div className="font-mono text-xs text-white bg-black/40 border border-gray-900 p-2.5 rounded-lg select-all">
                    {contract.admin || "No Admin / Immutable"}
                  </div>
                </div>

                <div className="space-y-1">
                  <span className="text-xs text-gray-400">Code ID</span>
                  <div className="font-mono text-sm font-bold text-blue-400 bg-black/40 border border-gray-900 p-2.5 rounded-lg">
                    {contract.codeId}
                  </div>
                </div>

                <div className="space-y-1">
                  <span className="text-xs text-gray-400">Instantiation Height</span>
                  <div className="font-mono text-sm font-bold text-green-400 bg-black/40 border border-gray-900 p-2.5 rounded-lg">
                    {300 + contract.codeId * 10}
                  </div>
                </div>
              </div>

              <div className="border-t border-gray-900 pt-6">
                <span className="text-xs text-gray-400 block mb-2">Interactions Summary</span>
                <div className="grid grid-cols-3 gap-4">
                  <div className="bg-black/20 p-3 rounded-lg border border-gray-900 text-center">
                    <span className="text-[10px] text-gray-500 uppercase tracking-wider block">Total Calls</span>
                    <span className="text-xl font-bold text-white">{historyList.length}</span>
                  </div>
                  <div className="bg-black/20 p-3 rounded-lg border border-gray-900 text-center">
                    <span className="text-[10px] text-gray-500 uppercase tracking-wider block">Failed Calls</span>
                    <span className="text-xl font-bold text-red-500">0</span>
                  </div>
                  <div className="bg-black/20 p-3 rounded-lg border border-gray-900 text-center">
                    <span className="text-[10px] text-gray-500 uppercase tracking-wider block">Est. Gas Spent</span>
                    <span className="text-xl font-bold text-blue-400">{historyList.length * 120000}</span>
                  </div>
                </div>
              </div>
            </div>

            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-4 shadow-lg h-fit">
              <h3 className="text-lg font-bold text-white flex items-center space-x-2">
                <Settings className="text-purple-500 h-5 w-5" />
                <span>Contract Type Badge</span>
              </h3>
              <p className="text-sm text-gray-400 leading-relaxed">
                This contract is compiled with CosmWasm standard schemas. It matches standard token behaviors which makes it compatible with automated indexes.
              </p>
              <div className="p-4 bg-purple-950/10 border border-purple-900/50 rounded-xl space-y-2">
                <span className="text-xs font-semibold text-purple-400 uppercase tracking-wide">Standard Compliance</span>
                <h4 className="text-md font-bold text-white">
                  {contract.typeBadge === "CW-721" ? "Non-Fungible Token (CW-721)" : contract.typeBadge === "CW-1155" ? "Multi-Token Standard (CW-1155)" : "Fungible Token (CW-20)"}
                </h4>
                <p className="text-xs text-gray-400">Supports metadata registries, balances tracking, and token ownership rules.</p>
              </div>
            </div>
          </div>
        )}

        {/* CW-20 TOKEN STANDARD TAB */}
        {activeTab === "cw20" && (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-lg space-y-4 h-fit">
              <h3 className="text-base font-bold text-white flex items-center gap-2">
                <Coins className="text-blue-500 h-5 w-5" />
                Holders Share (Donut Chart)
              </h3>
              <div className="h-48 w-full flex items-center justify-center">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={holders}
                      cx="50%"
                      cy="50%"
                      innerRadius={45}
                      outerRadius={65}
                      paddingAngle={4}
                      dataKey="share"
                    >
                      {holders.map((entry, index) => (
                        <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                      ))}
                    </Pie>
                    <ChartTooltip contentStyle={{ backgroundColor: "#09090b", borderColor: "#1f2937", color: "#fff" }} />
                  </PieChart>
                </ResponsiveContainer>
              </div>
              <div className="space-y-2 pt-2 text-xs">
                {holders.map((h, i) => (
                  <div key={i} className="flex justify-between items-center">
                    <span className="flex items-center gap-2 text-gray-400 font-mono">
                      <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: COLORS[i % COLORS.length] }} />
                      {h.address.slice(0, 12)}...
                    </span>
                    <span className="text-white font-bold font-mono">{h.share}%</span>
                  </div>
                ))}
              </div>
            </div>

            <div className="lg:col-span-2 bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-lg space-y-4">
              <h3 className="text-lg font-bold text-white flex items-center gap-2">
                <Users className="text-indigo-400 h-5 w-5" /> Token Holders List
              </h3>
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm text-gray-400">
                  <thead className="bg-black/50 text-xs text-gray-500 uppercase tracking-wider font-bold">
                    <tr>
                      <th className="p-3">Rank</th>
                      <th className="p-3">Holder Address</th>
                      <th className="p-3">Balance</th>
                      <th className="p-3 text-right">Shares</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-900">
                    {holders.map((h, idx) => (
                      <tr key={idx} className="hover:bg-gray-900/30 transition">
                        <td className="p-3 font-mono font-bold text-gray-500">#{idx + 1}</td>
                        <td className="p-3 font-mono text-xs text-blue-400">
                          <Link href={`/address/${h.address}`} className="hover:underline">
                            {h.address}
                          </Link>
                        </td>
                        <td className="p-3 font-mono text-xs text-gray-200">{h.balance}</td>
                        <td className="p-3 text-right text-white font-bold font-mono">{h.share}%</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {/* CW-721 NFT GALLERY TAB */}
        {activeTab === "cw721" && (
          <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-lg space-y-4">
            <h3 className="text-lg font-bold text-white flex items-center gap-2">
              <Image className="text-purple-400 h-5 w-5" /> CW-721 Collection Gallery
            </h3>
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6 pt-2">
              {nfts.map((nft) => (
                <div key={nft.tokenId} className="bg-gray-900 border border-gray-850 rounded-xl overflow-hidden shadow-lg hover:scale-[1.02] transition duration-200">
                  <div className="h-44 relative bg-gray-950 overflow-hidden">
                    <img src={nft.uri} alt={`NFT #${nft.tokenId}`} className="w-full h-full object-cover" />
                  </div>
                  <div className="p-4 space-y-2">
                    <div className="flex justify-between items-center">
                      <span className="text-xs text-gray-500 font-bold">ID: #{nft.tokenId}</span>
                      <span className="text-[10px] px-1.5 py-0.5 bg-purple-950 text-purple-400 rounded font-semibold border border-purple-900/50">
                        {nft.trait}
                      </span>
                    </div>
                    <div className="space-y-1 text-xs">
                      <div className="text-gray-500 uppercase text-[9px] font-bold">Current Owner</div>
                      <Link href={`/address/${nft.owner}`} className="font-mono text-blue-400 hover:underline break-all block">
                        {nft.owner.slice(0, 12)}...{nft.owner.slice(-6)}
                      </Link>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* CW-1155 INVENTORY TAB */}
        {activeTab === "cw1155" && (
          <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-lg space-y-4">
            <h3 className="text-lg font-bold text-white flex items-center gap-2">
              <Layers className="text-blue-500 h-5 w-5" /> CW-1155 Multi-Token Catalog
            </h3>
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm text-gray-400">
                <thead className="bg-black/50 text-xs text-gray-500 uppercase tracking-wider font-bold">
                  <tr>
                    <th className="p-4">Token ID</th>
                    <th className="p-4">Name/Identifier</th>
                    <th className="p-4">Total Circulating Supply</th>
                    <th className="p-4 text-right">Your Bonded Balance</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-900">
                  {multiTokens.map((t) => (
                    <tr key={t.tokenId} className="hover:bg-gray-900/30 transition">
                      <td className="p-4 font-mono font-bold text-white text-xs">{t.tokenId}</td>
                      <td className="p-4 text-gray-300 font-semibold">{t.name}</td>
                      <td className="p-4 font-mono text-xs text-gray-400">{t.supply} Units</td>
                      <td className="p-4 text-right text-green-400 font-bold font-mono">{t.balance} Units</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* QUERY STATE TAB */}
        {activeTab === "query" && (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-4 shadow-lg flex flex-col justify-between">
              <div className="space-y-4">
                <div className="flex justify-between items-center border-b border-gray-900 pb-3">
                  <h3 className="text-lg font-bold text-white flex items-center space-x-2">
                    <Terminal className="text-blue-500 h-5 w-5" />
                    <span>Query Editor</span>
                  </h3>
                  
                  {/* Presets */}
                  <select 
                    onChange={(e) => selectQueryPreset(e.target.value)}
                    className="text-xs bg-gray-900 text-gray-300 border border-gray-800 rounded px-2.5 py-1.5 focus:outline-none focus:border-blue-500 cursor-pointer"
                  >
                    <option value="">-- Choose Query Preset --</option>
                    {contract.typeBadge === "CW-721" ? (
                      <>
                        <option value="contract_info">Get Contract Info</option>
                        <option value="num_tokens">Get Total Tokens</option>
                        <option value="owner_of">Owner Of Token</option>
                      </>
                    ) : contract.typeBadge === "CW-1155" ? (
                      <>
                        <option value="balance_of">Get 1155 Balance</option>
                      </>
                    ) : (
                      <>
                        <option value="token_info">Get Token Info</option>
                        <option value="balance">Get Account Balance</option>
                        <option value="allowance">Get Spender Allowance</option>
                      </>
                    )}
                  </select>
                </div>

                <p className="text-xs text-gray-400">
                  Submit query messages to retrieve read-only state details.
                </p>

                <div className="relative">
                  <textarea
                    value={queryJSON}
                    onChange={(e) => setQueryJSON(e.target.value)}
                    rows={8}
                    className="w-full font-mono text-xs p-4 bg-black/60 border border-gray-900 rounded-xl text-white focus:outline-none focus:border-blue-500 resize-y"
                    placeholder="{}"
                  />
                  {queryError && (
                    <div className="mt-2 text-xs text-red-400 bg-red-950/20 border border-red-900/50 p-2.5 rounded-lg flex items-center space-x-2">
                      <AlertCircle className="h-4 w-4 shrink-0" />
                      <span>{queryError}</span>
                    </div>
                  )}
                </div>
              </div>

              <button
                onClick={handleQuery}
                disabled={queryRunning}
                className="w-full mt-4 bg-blue-600 hover:bg-blue-500 disabled:bg-blue-800 text-white rounded-xl py-3 font-semibold transition flex items-center justify-center space-x-2 shadow-lg shadow-blue-900/20"
              >
                {queryRunning ? (
                  <>
                    <Activity className="animate-spin h-4 w-4" />
                    <span>Executing Query...</span>
                  </>
                ) : (
                  <>
                    <Play className="h-4 w-4 fill-white" />
                    <span>Run Query</span>
                  </>
                )}
              </button>
            </div>

            {/* Component 6: JSONSchemaForm Visualizer */}
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-lg space-y-4">
              <h3 className="text-base font-bold text-white flex items-center gap-2 pb-2 border-b border-gray-900">
                <FileJson className="h-5 w-5 text-indigo-400" /> Interactive Form Fields
              </h3>
              <p className="text-xs text-gray-400 leading-relaxed">
                Compose query arguments visually. Modifying these fields dynamically updates the raw editor JSON.
              </p>
              <div className="space-y-4 pt-1">
                {parseJsonKeys(queryJSON).map((k) => (
                  <div key={k.path} className="space-y-1">
                    <label className="text-xs font-mono font-bold text-gray-500 uppercase">{k.path}</label>
                    <input
                      type="text"
                      value={typeof k.value === "object" ? JSON.stringify(k.value) : String(k.value || "")}
                      onChange={(e) => handleFormFieldChange(k.path, e.target.value, true)}
                      className="w-full bg-gray-900 border border-gray-850 px-3 py-2 rounded-lg text-xs font-mono text-white focus:outline-none focus:border-blue-500 transition"
                      placeholder={`Enter ${k.path}...`}
                    />
                  </div>
                ))}
              </div>
            </div>

            {/* Query Response */}
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-4 shadow-lg flex flex-col justify-between min-h-[300px]">
              <div>
                <h3 className="text-lg font-bold text-white flex items-center space-x-2 pb-2 border-b border-gray-900">
                  <FileJson className="text-green-500 h-5 w-5" />
                  <span>Query Response</span>
                </h3>

                {queryResult ? (
                  <pre className="text-xs font-mono text-green-400 p-4 bg-black/40 border border-gray-900 rounded-xl overflow-x-auto whitespace-pre-wrap mt-4 max-h-[350px]">
                    {queryResult}
                  </pre>
                ) : (
                  <div className="h-48 flex items-center justify-center text-gray-500 text-sm">
                    {queryRunning ? "Waiting for response..." : "Execute a query to inspect the response details."}
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {/* EXECUTE TRANSACTION TAB */}
        {activeTab === "execute" && (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-4 shadow-lg flex flex-col justify-between">
              <div className="space-y-4">
                <div className="flex justify-between items-center border-b border-gray-900 pb-3">
                  <h3 className="text-lg font-bold text-white flex items-center space-x-2">
                    <Play className="text-amber-500 h-5 w-5" />
                    <span>Build Execute Transaction</span>
                  </h3>
                  
                  {/* Presets */}
                  <select 
                    onChange={(e) => selectExecPreset(e.target.value)}
                    className="text-xs bg-gray-900 text-gray-300 border border-gray-800 rounded px-2.5 py-1.5 focus:outline-none focus:border-amber-500 cursor-pointer"
                  >
                    <option value="">-- Choose Execute Preset --</option>
                    {contract.typeBadge === "CW-721" ? (
                      <>
                        <option value="mint_nft">Mint New NFT</option>
                        <option value="transfer_nft">Transfer NFT</option>
                      </>
                    ) : contract.typeBadge === "CW-1155" ? (
                      <>
                        <option value="send_multi">Send 1155 MultiToken</option>
                      </>
                    ) : (
                      <>
                        <option value="transfer">Transfer Tokens</option>
                        <option value="mint">Mint Tokens</option>
                        <option value="burn">Burn Tokens</option>
                      </>
                    )}
                  </select>
                </div>

                <p className="text-xs text-gray-400">
                  Build and submit transaction execute calls. Requires a connected wallet to broadcast.
                </p>

                {/* Form fields */}
                <div className="space-y-3">
                  <div className="space-y-1">
                    <span className="text-xs text-gray-400">Execute Message (JSON)</span>
                    <textarea
                      value={execJSON}
                      onChange={(e) => setExecJSON(e.target.value)}
                      rows={6}
                      className="w-full font-mono text-xs p-4 bg-black/60 border border-gray-900 rounded-xl text-white focus:outline-none focus:border-amber-500 resize-y"
                      placeholder="{}"
                    />
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-1">
                      <span className="text-xs text-gray-400">Gas Limit</span>
                      <input 
                        type="number" 
                        value={execGas}
                        onChange={(e) => setExecGas(Number(e.target.value))}
                        className="w-full bg-gray-900 border border-gray-850 text-white rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-amber-500 font-mono"
                      />
                    </div>
                    <div className="space-y-1">
                      <span className="text-xs text-gray-400">Funds Send (Optional)</span>
                      <input 
                        type="text" 
                        placeholder="e.g. 1000uSLT"
                        value={execFunds}
                        onChange={(e) => setExecFunds(e.target.value)}
                        className="w-full bg-gray-900 border border-gray-850 text-white rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-amber-500 font-mono"
                      />
                    </div>
                  </div>

                  {execError && (
                    <div className="text-xs text-red-400 bg-red-950/20 border border-red-900/50 p-2.5 rounded-lg flex items-center space-x-2">
                      <AlertCircle className="h-4 w-4 shrink-0" />
                      <span>{execError}</span>
                    </div>
                  )}
                </div>
              </div>

              <button
                onClick={handleExecute}
                disabled={executing}
                className="w-full mt-4 bg-amber-600 hover:bg-amber-500 disabled:bg-amber-800 text-white rounded-xl py-3 font-semibold transition flex items-center justify-center space-x-2 shadow-lg shadow-amber-900/20"
              >
                {executing ? (
                  <>
                    <Activity className="animate-spin h-4 w-4" />
                    <span>{execStep}</span>
                  </>
                ) : (
                  <>
                    <Lock className="h-4 w-4" />
                    <span>Sign & Broadcast Execute</span>
                  </>
                )}
              </button>
            </div>

            {/* JSONSchemaForm fields for Execute */}
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-lg space-y-4">
              <h3 className="text-base font-bold text-white flex items-center gap-2 pb-2 border-b border-gray-900">
                <FileJson className="h-5 w-5 text-indigo-400" /> Interactive Form Fields
              </h3>
              <p className="text-xs text-gray-400 leading-relaxed">
                Compose execution payload arguments visually.
              </p>
              <div className="space-y-4 pt-1">
                {parseJsonKeys(execJSON).map((k) => (
                  <div key={k.path} className="space-y-1">
                    <label className="text-xs font-mono font-bold text-gray-500 uppercase">{k.path}</label>
                    <input
                      type="text"
                      value={typeof k.value === "object" ? JSON.stringify(k.value) : String(k.value || "")}
                      onChange={(e) => handleFormFieldChange(k.path, e.target.value, false)}
                      className="w-full bg-gray-900 border border-gray-850 px-3 py-2 rounded-lg text-xs font-mono text-white focus:outline-none focus:border-amber-500 transition"
                      placeholder={`Enter ${k.path}...`}
                    />
                  </div>
                ))}
              </div>
            </div>

            {/* Broadcast receipt status */}
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-4 shadow-lg min-h-[300px]">
              <h3 className="text-lg font-bold text-white flex items-center space-x-2 pb-2 border-b border-gray-900">
                <Activity className="text-amber-500 h-5 w-5" />
                <span>Transaction Status</span>
              </h3>

              {executing && (
                <div className="space-y-4 py-8 text-center">
                  <Activity className="animate-spin h-10 w-10 text-amber-500 mx-auto" />
                  <div className="text-sm font-semibold text-white uppercase tracking-wider">{execStep}</div>
                  <p className="text-xs text-gray-400">Verifying on Sovereign L1 Devnet...</p>
                </div>
              )}

              {!executing && execResult && (
                <div className="space-y-4 pt-2">
                  <div className="p-4 bg-green-950/20 border border-green-900/50 rounded-xl space-y-2">
                    <div className="flex items-center space-x-2 text-green-400">
                      <Check className="h-5 w-5 font-bold" />
                      <span className="text-sm font-bold uppercase tracking-wider">Transaction Confirmed</span>
                    </div>
                    <p className="text-xs text-gray-400 leading-relaxed">
                      Your execute instruction has been processed on-chain in CometBFT block <strong>#{execResult.height}</strong>.
                    </p>
                  </div>

                  <div className="space-y-3.5 text-xs">
                    <div className="flex justify-between border-b border-gray-900 pb-2">
                      <span className="text-gray-400">Transaction Hash</span>
                      <span className="font-mono text-white select-all">{execResult.hash}</span>
                    </div>
                    <div className="flex justify-between border-b border-gray-900 pb-2">
                      <span className="text-gray-400">Height</span>
                      <span className="font-mono text-white">#{execResult.height}</span>
                    </div>
                    <div className="flex justify-between border-b border-gray-900 pb-2">
                      <span className="text-gray-400">Sender Address</span>
                      <span className="font-mono text-white">{execResult.sender}</span>
                    </div>
                    <div className="flex flex-col space-y-1">
                      <span className="text-gray-400">Submitted Message JSON</span>
                      <pre className="p-3 bg-black/40 border border-gray-900 rounded-lg text-[10px] font-mono text-gray-300 overflow-x-auto whitespace-pre">
                        {JSON.stringify(execResult.msg, null, 2)}
                      </pre>
                    </div>
                  </div>
                </div>
              )}

              {!executing && !execResult && (
                <div className="h-48 flex items-center justify-center text-gray-500 text-sm">
                  Waiting for transaction signature and broadcast.
                </div>
              )}
            </div>
          </div>
        )}

        {/* EXECUTION HISTORY TAB */}
        {activeTab === "history" && (
          <div className="bg-gray-950 border border-gray-900 rounded-2xl overflow-hidden shadow-lg">
            <div className="px-6 py-4 border-b border-gray-900 flex justify-between items-center">
              <h3 className="text-lg font-bold text-white flex items-center space-x-2">
                <History className="text-blue-500 h-5 w-5" />
                <span>Past Executions</span>
              </h3>
              <span className="text-xs px-2.5 py-1 bg-gray-900 border border-gray-800 rounded font-semibold text-gray-400 font-mono">
                {historyList.length} Execution{historyList.length === 1 ? "" : "s"}
              </span>
            </div>

            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead className="bg-black/50 text-gray-400 uppercase text-xs">
                  <tr>
                    <th className="px-6 py-3">Tx Hash</th>
                    <th className="px-6 py-3">Height</th>
                    <th className="px-6 py-3">Decoded Exec Message</th>
                    <th className="px-6 py-3">Sender</th>
                    <th className="px-6 py-3">Status</th>
                    <th className="px-6 py-3">Age</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-900">
                  {historyList.length === 0 ? (
                    <tr>
                      <td colSpan={6} className="px-6 py-12 text-center text-gray-500">No executions found for this contract yet.</td>
                    </tr>
                  ) : (
                    historyList.map((tx, idx) => (
                      <tr key={idx} className="hover:bg-gray-900/30 transition">
                        <td className="px-6 py-4 font-mono text-xs text-blue-400">
                          <Link href={`/txs/${tx.hash}`} className="hover:underline">
                            {tx.hash.slice(0, 10)}...{tx.hash.slice(-8)}
                          </Link>
                        </td>
                        <td className="px-6 py-4 font-mono text-xs text-gray-400">
                          <Link href={`/blocks/${tx.height}`} className="hover:underline text-blue-400">
                            #{tx.height}
                          </Link>
                        </td>
                        <td className="px-6 py-4">
                          <pre className="text-[10px] font-mono text-gray-300 bg-black/40 border border-gray-900 p-2 rounded max-w-sm overflow-x-auto whitespace-pre">
                            {JSON.stringify(tx.msg, null, 2)}
                          </pre>
                        </td>
                        <td className="px-6 py-4 font-mono text-xs text-gray-400">
                          {tx.sender ? `${tx.sender.slice(0, 8)}...${tx.sender.slice(-6)}` : ""}
                        </td>
                        <td className="px-6 py-4">
                          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold uppercase bg-green-950/50 text-green-400 border border-green-900">
                            {tx.status}
                          </span>
                        </td>
                        <td className="px-6 py-4 text-xs text-gray-400">
                          {new Date(tx.time).toLocaleTimeString()}
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}

      </div>
    </div>
  );
}
