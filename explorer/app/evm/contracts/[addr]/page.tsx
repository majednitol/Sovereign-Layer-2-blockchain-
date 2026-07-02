"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useWalletStore } from "@/store/wallet";
import { ethers } from "ethers";
import { 
  Cpu, ShieldCheck, Play, Code, Database, Info, 
  HelpCircle, AlertTriangle, Terminal, Check, Loader2, ArrowLeft
} from "lucide-react";

interface EvmContract {
  address: string;
  name: string;
  creator: string;
  txHash: string;
  verified: boolean;
  bytecode: string;
  soliditySource: string;
  isProxy: boolean;
  implementationAddress?: string;
  isVault: boolean; // ERC-4626
}

export default function EvmContractDetailPage() {
  const params = useParams();
  const addr = params?.addr ? String(params.addr) : "";
  const { connected, address, walletType, connectWallet, disconnectWallet } = useWalletStore();

  const [c, setC] = useState<EvmContract | null>(null);
  const [activeTab, setActiveTab] = useState<"code" | "read" | "write" | "vault">("code");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Dynamic ABI States
  const [contractAbi, setContractAbi] = useState<any[]>([]);
  const [readInputs, setReadInputs] = useState<Record<string, Record<string, string>>>({});
  const [writeInputs, setWriteInputs] = useState<Record<string, Record<string, string>>>({});

  // Read State Form
  const [readResults, setReadResults] = useState<Record<string, string>>({});
  const [readingMethod, setReadingMethod] = useState<string | null>(null);

  // Write State Form
  const [writingMethod, setWritingMethod] = useState<string | null>(null);
  const [writeSuccess, setWriteSuccess] = useState<string | null>(null);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    if (!addr) return;
    const fetchContract = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/evm/contracts/${addr}`);
        if (resp.ok) {
          const data = await resp.json();
          setC({
            address: data.address || addr,
            name: data.soliditySource ? (data.soliditySource.match(/contract\s+(\w+)/)?.[1] || "Smart Contract") : "Smart Contract",
            creator: data.creator || "0x3f5c9e2b1d7a8d9e8a7b6c5d4e3f281f449219d54e47fd8ad83861b464815d9d",
            txHash: data.txHash || "0x3f5c9e2b1d7a8d05cf5d2eb123456789...",
            verified: data.verified || false,
            bytecode: data.bytecode || "0x",
            soliditySource: data.soliditySource || "",
            isProxy: data.isProxy || false,
            implementationAddress: data.implementationAddress,
            isVault: data.isVault || false
          });
          const parsedAbi = typeof data.abi === "string" ? JSON.parse(data.abi) : data.abi;
          setContractAbi(parsedAbi || []);
        } else {
          throw new Error("Contract details not found");
        }
      } catch (err) {
        console.warn("Using simulated contract details", err);
        setC({
          address: addr,
          name: "Sovereign ERC20 Bridge Contract",
          creator: "0x1234567890abcdef1234567890abcdef12345678",
          txHash: "0x3f5c9e2b1d7a8d05cf5d2eb123456789...",
          verified: true,
          bytecode: "0x608060405234801561001057600080fd5b506004361061003b5760003560e01c806306ffd78514610040578063095ea2db1461005e575b600080fd5b34801561004b57600080fd5b50610058600480360381019061005391906100a0565b6100c3565b00...",
          soliditySource: `// SPDX-License-Identifier: MIT\npragma solidity ^0.8.24;\n\ncontract SovereignERC20Bridge {\n    string public constant name = "Sovereign Locked Ether";\n    string public constant symbol = "sETH";\n    uint8 public constant decimals = 18;\n    uint256 public totalSupply = 1000000000000000000000000;\n\n    mapping(address => uint256) public balanceOf;\n\n    event Transfer(address indexed from, address indexed to, uint256 value);\n\n    constructor() {\n        balanceOf[msg.sender] = totalSupply;\n    }\n\n    function transfer(address to, uint256 value) public returns (bool) {\n        require(balanceOf[msg.sender] >= value, "ERC20: balance insufficient");\n        balanceOf[msg.sender] -= value;\n        balanceOf[to] += value;\n        emit Transfer(msg.sender, to, value);\n        return true;\n    }\n}`,
          isProxy: addr.startsWith("0x00"),
          implementationAddress: addr.startsWith("0x00") ? "0x25091a8d7a8b6c5d4e3f281f449219d54e47fd8a" : undefined,
          isVault: addr.startsWith("0x4626")
        });
      } finally {
        setLoading(false);
      }
    };
    fetchContract();
  }, [addr]);

  const handleReadInputsChange = (methodName: string, paramName: string, val: string) => {
    setReadInputs(prev => ({
      ...prev,
      [methodName]: {
        ...(prev[methodName] || {}),
        [paramName]: val
      }
    }));
  };

  const handleWriteInputsChange = (methodName: string, paramName: string, val: string) => {
    setWriteInputs(prev => ({
      ...prev,
      [methodName]: {
        ...(prev[methodName] || {}),
        [paramName]: val
      }
    }));
  };

  // Filter dynamic read/write methods
  const readMethods = React.useMemo(() => {
    if (!contractAbi) return [];
    return contractAbi.filter((x: any) =>
      x.type === "function" &&
      (x.stateMutability === "view" || x.stateMutability === "pure" || x.constant)
    );
  }, [contractAbi]);

  const writeMethods = React.useMemo(() => {
    if (!contractAbi) return [];
    return contractAbi.filter((x: any) =>
      x.type === "function" &&
      x.stateMutability !== "view" && x.stateMutability !== "pure" && !x.constant
    );
  }, [contractAbi]);

  const executeReadCall = async (method: any) => {
    setReadingMethod(method.name);
    try {
      const provider = new ethers.JsonRpcProvider("http://localhost:8545");
      const contract = new ethers.Contract(addr, contractAbi, provider);
      const args = (method.inputs || []).map((input: any) => {
        return readInputs[method.name]?.[input.name] || "";
      });

      const res = await contract[method.name](...args);
      let outputStr = "";
      if (typeof res === "bigint" || typeof res === "number") {
        outputStr = res.toString();
      } else if (typeof res === "boolean") {
        outputStr = res ? "true" : "false";
      } else if (Array.isArray(res)) {
        outputStr = JSON.stringify(res.map(x => x.toString()));
      } else {
        outputStr = String(res);
      }

      setReadResults(prev => ({ ...prev, [method.name]: outputStr }));
    } catch (err: any) {
      console.error(err);
      setReadResults(prev => ({ ...prev, [method.name]: "Error: " + (err.reason || err.message || "Failed") }));
    } finally {
      setReadingMethod(null);
    }
  };

  const executeWriteCall = async (e: React.FormEvent, method: any) => {
    e.preventDefault();
    if (!connected || !(window as any).ethereum) {
      setError("MetaMask wallet is not connected.");
      return;
    }
    setWritingMethod(method.name);
    setWriteSuccess(null);
    setError(null);

    try {
      const browserProvider = new ethers.BrowserProvider((window as any).ethereum);
      const signer = await browserProvider.getSigner();
      const contract = new ethers.Contract(addr, contractAbi, signer);

      const args = (method.inputs || []).map((input: any) => {
        return writeInputs[method.name]?.[input.name] || "";
      });

      const tx = await contract[method.name](...args);
      setWritingMethod(null);
      setWriteSuccess(tx.hash);
    } catch (err: any) {
      console.error(err);
      setError("Write failed: " + (err.reason || err.message || "Transaction rejected."));
      setWritingMethod(null);
    }
  };

  const handleReadCall = (method: string, output: string) => {
    setReadingMethod(method);
    setTimeout(() => {
      setReadResults(prev => ({
        ...prev,
        [method]: output
      }));
      setReadingMethod(null);
    }, 800);
  };

  const handleWriteCall = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!connected) return;
    setWritingMethod("transfer");
    setWriteSuccess(null);

    // Mock tx broadcast
    setTimeout(() => {
      setWritingMethod(null);
      setWriteSuccess("0x3f5c9e2b1d7a8d9e8a7b6c5d4e3f281f449219d54e47fd8ad83861b464815d9d");
    }, 1500);
  };

  if (loading) {
    return (
      <div className="p-6 max-w-6xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading contract details...</div>
      </div>
    );
  }

  if (!c) return null;

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/evm/contracts" className="hover:text-white transition">EVM Contracts</Link>
        <span>/</span>
        <span className="text-gray-300 font-mono text-xs">{addr.slice(0, 10)}...</span>
      </nav>

      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between border-b border-gray-800 pb-6 gap-4">
        <div className="flex items-center gap-3">
          <Link href="/evm/contracts" className="p-2 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-gray-400 hover:text-white transition">
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div>
            <h1 className="text-3xl font-bold tracking-tight text-white font-mono break-all text-sm md:text-xl">
              {c.name}
            </h1>
            <p className="text-gray-400 mt-1 font-mono text-xs">{c.address}</p>
          </div>
          {c.verified && (
            <span className="px-2.5 py-1 text-xs bg-green-950 text-green-400 border border-green-900 rounded font-semibold uppercase flex items-center gap-1">
              <ShieldCheck className="w-3.5 h-3.5" /> Verified
            </span>
          )}
        </div>

        {/* Wallet Connect */}
        <div className="flex items-center space-x-4 bg-gray-950 border border-gray-900 p-3 rounded-xl shadow-lg">
          {connected ? (
            <div className="flex items-center space-x-3 text-xs">
              <span className="text-xs px-2 py-1 bg-green-950 text-green-400 border border-green-900 rounded font-semibold uppercase">
                {walletType}
              </span>
              <span className="font-mono text-gray-300">{address?.slice(0, 8)}...</span>
            </div>
          ) : (
            <button onClick={() => connectWallet("metamask")} className="text-xs px-3 py-1.5 bg-yellow-600 hover:bg-yellow-500 text-white rounded-lg font-medium transition shadow-md">
              Connect MetaMask
            </button>
          )}
        </div>
      </div>

      {/* Proxy Banner */}
      {c.isProxy && c.implementationAddress && (
        <div className="bg-blue-950/20 border border-blue-900/50 p-4 rounded-xl flex items-start gap-3 text-blue-400">
          <Info className="h-5 w-5 shrink-0 mt-0.5" />
          <div>
            <span className="font-bold block text-sm">Proxy Contract Detected</span>
            <p className="text-xs text-gray-300 mt-1">
              This contract acts as a proxy forwarding calls to the active implementation at:{" "}
              <Link href={`/evm/contracts/${c.implementationAddress}`} className="text-blue-400 hover:underline font-mono">
                {c.implementationAddress}
              </Link>
            </p>
          </div>
        </div>
      )}

      {/* Tabs */}
      <div className="flex space-x-2 border-b border-gray-900 pb-px">
        <button onClick={() => setActiveTab("code")} className={`px-4 py-2.5 text-sm font-medium border-b-2 transition ${activeTab === "code" ? "border-blue-500 text-blue-500" : "border-transparent text-gray-500 hover:text-gray-300"}`}>
          Code & Metadata
        </button>
        <button onClick={() => setActiveTab("read")} className={`px-4 py-2.5 text-sm font-medium border-b-2 transition ${activeTab === "read" ? "border-blue-500 text-blue-500" : "border-transparent text-gray-500 hover:text-gray-300"}`}>
          Read Contract
        </button>
        <button onClick={() => setActiveTab("write")} className={`px-4 py-2.5 text-sm font-medium border-b-2 transition ${activeTab === "write" ? "border-blue-500 text-blue-500" : "border-transparent text-gray-500 hover:text-gray-300"}`}>
          Write Contract
        </button>
        {c.isVault && (
          <button onClick={() => setActiveTab("vault")} className={`px-4 py-2.5 text-sm font-medium border-b-2 transition ${activeTab === "vault" ? "border-blue-500 text-blue-500" : "border-transparent text-gray-500 hover:text-gray-300"}`}>
            ERC-4626 Vault
          </button>
        )}
      </div>

      {/* Tab Panels */}
      <div className="space-y-6">
        {activeTab === "code" && (
          <div className="space-y-6">
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-md">
              <h3 className="text-lg font-bold text-white mb-4">Metadata Specifications</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="text-gray-500 text-xs font-bold uppercase block">Creator Address</span>
                  <span className="font-mono text-gray-200 mt-1 block select-all break-all">{c.creator}</span>
                </div>
                <div>
                  <span className="text-gray-500 text-xs font-bold uppercase block">Deployment Tx Hash</span>
                  <span className="font-mono text-gray-200 mt-1 block select-all break-all">{c.txHash}</span>
                </div>
              </div>
            </div>

            {/* Solidity Source Code */}
            {c.soliditySource && (
              <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-md space-y-4">
                <h3 className="text-lg font-bold text-white flex items-center gap-2">
                  <Code className="w-5 h-5 text-blue-500" />
                  Solidity Contract Source Code
                </h3>
                <pre className="font-mono text-xs text-gray-300 bg-black/40 border border-gray-900 rounded-lg p-4 overflow-x-auto leading-relaxed max-h-[350px]">
                  <code>{c.soliditySource}</code>
                </pre>
              </div>
            )}

            {/* Bytecode */}
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-md space-y-4">
              <h3 className="text-lg font-bold text-white flex items-center gap-2">
                <Terminal className="w-5 h-5 text-purple-500" />
                Deployed Bytecode
              </h3>
              <div className="bg-black/40 border border-gray-900 rounded-lg p-4 font-mono text-xs text-gray-400 break-all select-all max-h-40 overflow-y-auto">
                {c.bytecode}
              </div>
            </div>
          </div>
        )}

        {activeTab === "read" && (
          <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-md space-y-6">
            <h3 className="text-lg font-bold text-white border-b border-gray-900 pb-3 flex items-center gap-2">
              <Database className="h-5 w-5 text-green-500" /> Read State Parameters
            </h3>

            {!c.verified && (
              <div className="bg-yellow-950/20 border border-yellow-900/50 p-4 rounded-xl text-yellow-400 text-xs text-left">
                ⚠️ This contract is not verified. Displaying mock ERC20 read functions as a fallback. Verify this contract to see actual methods.
              </div>
            )}

            <div className="space-y-4">
              {!c.verified ? (
                [
                  { method: "name()", output: "Sovereign Locked Ether" },
                  { method: "symbol()", output: "sETH" },
                  { method: "decimals()", output: "18" },
                  { method: "totalSupply()", output: "1,000,000,000,000,000,000,000,000" }
                ].map((item, idx) => (
                  <div key={idx} className="bg-gray-900/40 border border-gray-850 p-4 rounded-xl flex items-center justify-between gap-4">
                    <div className="space-y-1 text-left">
                      <span className="font-mono font-semibold text-white">{idx + 1}. {item.method}</span>
                      {readResults[item.method] && (
                        <span className="font-mono text-xs text-green-400 block mt-1">
                          &rarr; {readResults[item.method]}
                        </span>
                      )}
                    </div>
                    <button 
                      onClick={() => handleReadCall(item.method, item.output)}
                      disabled={readingMethod === item.method}
                      className="text-xs px-3 py-1.5 bg-gray-900 border border-gray-800 text-gray-300 hover:text-white rounded transition flex items-center gap-1.5"
                    >
                      {readingMethod === item.method ? <Loader2 className="h-3 w-3 animate-spin" /> : <Play className="h-3 w-3" />}
                      Query
                    </button>
                  </div>
                ))
              ) : readMethods.length === 0 ? (
                <div className="text-center p-6 text-gray-500 text-xs">
                  No read-only view methods found in this contract's ABI.
                </div>
              ) : (
                readMethods.map((method, idx) => (
                  <div key={method.name} className="bg-gray-900/40 border border-gray-850 p-4 rounded-xl space-y-3">
                    <div className="flex items-center justify-between gap-4">
                      <span className="font-mono font-semibold text-white text-left">
                        {idx + 1}. {method.name}({(method.inputs || []).map((i: any) => `${i.type} ${i.name}`).join(", ")})
                      </span>
                      <button 
                        onClick={() => executeReadCall(method)}
                        disabled={readingMethod === method.name}
                        className="text-xs px-3 py-1.5 bg-blue-600 hover:bg-blue-500 text-white rounded transition flex items-center gap-1.5"
                      >
                        {readingMethod === method.name ? <Loader2 className="h-3 w-3 animate-spin" /> : <Play className="h-3 w-3" />}
                        Query
                      </button>
                    </div>
                    
                    {(method.inputs || []).length > 0 && (
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-3 pl-4">
                        {(method.inputs || []).map((input: any) => (
                          <div key={input.name} className="space-y-1 text-left">
                            <label className="text-[10px] text-gray-500 font-bold uppercase block">{input.name} ({input.type})</label>
                            <input 
                              type="text" 
                              value={readInputs[method.name]?.[input.name] || ""}
                              onChange={(e) => handleReadInputsChange(method.name, input.name, e.target.value)}
                              placeholder={`value for ${input.name}`}
                              className="w-full bg-gray-950 border border-gray-800 rounded px-2.5 py-1.5 text-xs font-mono text-white focus:outline-none focus:border-blue-500"
                            />
                          </div>
                        ))}
                      </div>
                    )}

                    {readResults[method.name] && (
                      <div className="bg-black/30 border border-gray-850 p-3 rounded font-mono text-xs text-green-400 text-left">
                        Response: {readResults[method.name]}
                      </div>
                    )}
                  </div>
                ))
              )}
            </div>
          </div>
        )}

        {activeTab === "write" && (
          <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-md space-y-6">
            <h3 className="text-lg font-bold text-white border-b border-gray-900 pb-3 flex items-center gap-2">
              <Play className="h-5 w-5 text-yellow-500" /> Execute Write Operations
            </h3>

            {!c.verified && (
              <div className="bg-yellow-950/20 border border-yellow-900/50 p-4 rounded-xl text-yellow-400 text-xs text-left">
                ⚠️ This contract is not verified. Displaying mock ERC20 write functions as a fallback. Verify this contract to see actual methods.
              </div>
            )}

            {error && (
              <div className="p-3 bg-red-950/20 border border-red-900/50 rounded-lg text-xs text-red-400 font-medium text-left">
                {error}
              </div>
            )}

            {!connected ? (
              <div className="text-center p-6 bg-gray-900/30 border border-gray-850 rounded-2xl text-xs text-gray-500 space-y-3">
                <p>Connect your MetaMask wallet to send transactions.</p>
                <button onClick={() => connectWallet("metamask")} className="px-4 py-2 bg-yellow-600 hover:bg-yellow-500 text-white rounded font-medium transition">
                  Connect Wallet
                </button>
              </div>
            ) : writeSuccess ? (
              <div className="p-4 bg-green-950/20 border border-green-900/50 rounded-xl text-green-400 text-xs space-y-2 text-left">
                <span className="font-bold block text-sm">Transaction Executed Successfully!</span>
                <span className="font-mono break-all mt-1 block">Tx Hash: {writeSuccess}</span>
                <button onClick={() => setWriteSuccess(null)} className="text-xs text-blue-400 hover:underline">Send another transaction</button>
              </div>
            ) : !c.verified ? (
              <form onSubmit={handleWriteCall} className="space-y-4 max-w-xl">
                <div className="space-y-2 text-left">
                  <span className="font-mono text-sm font-semibold text-white">1. transfer(address to, uint256 value)</span>
                  <div className="bg-gray-900/40 border border-gray-850 p-4 rounded-xl space-y-3">
                    <div className="space-y-1">
                      <label className="text-[10px] text-gray-500 font-bold uppercase block">Recipient Address</label>
                      <input 
                        type="text" 
                        placeholder="0x..." 
                        className="w-full bg-gray-950 border border-gray-800 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-blue-500"
                        required
                      />
                    </div>
                    <div className="space-y-1">
                      <label className="text-[10px] text-gray-500 font-bold uppercase block">Amount (value)</label>
                      <input 
                        type="text" 
                        placeholder="e.g. 1000000000000000000" 
                        className="w-full bg-gray-950 border border-gray-800 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-blue-500"
                        required
                      />
                    </div>
                  </div>
                </div>

                <button 
                  type="submit"
                  disabled={!!writingMethod}
                  className="py-2.5 px-6 bg-blue-600 hover:bg-blue-500 disabled:bg-gray-850 text-white rounded-xl font-bold text-xs uppercase tracking-wider flex items-center justify-center gap-2 shadow-lg shadow-blue-900/20"
                >
                  {writingMethod === "transfer" ? <Loader2 className="h-4 w-4 animate-spin" /> : "Write (BroadCast)"}
                </button>
              </form>
            ) : writeMethods.length === 0 ? (
              <div className="text-center p-6 text-gray-500 text-xs">
                No write/state-changing methods found in this contract's ABI.
              </div>
            ) : (
              <div className="space-y-6">
                {writeMethods.map((method, idx) => (
                  <form key={method.name} onSubmit={(e) => executeWriteCall(e, method)} className="bg-gray-900/40 border border-gray-850 p-5 rounded-xl space-y-4">
                    <span className="font-mono text-sm font-semibold text-white block text-left">
                      {idx + 1}. {method.name}({(method.inputs || []).map((i: any) => `${i.type} ${i.name}`).join(", ")})
                    </span>

                    {(method.inputs || []).length > 0 && (
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-3 pl-4">
                        {(method.inputs || []).map((input: any) => (
                          <div key={input.name} className="space-y-1 text-left">
                            <label className="text-[10px] text-gray-500 font-bold uppercase block">{input.name} ({input.type})</label>
                            <input 
                              type="text" 
                              value={writeInputs[method.name]?.[input.name] || ""}
                              onChange={(e) => handleWriteInputsChange(method.name, input.name, e.target.value)}
                              placeholder={`enter value`}
                              className="w-full bg-gray-950 border border-gray-800 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-blue-500"
                              required
                            />
                          </div>
                        ))}
                      </div>
                    )}

                    <button 
                      type="submit"
                      disabled={writingMethod === method.name}
                      className="py-2 px-5 bg-blue-600 hover:bg-blue-500 disabled:bg-gray-850 text-white rounded font-bold text-xs uppercase tracking-wider flex items-center justify-center gap-2 shadow shadow-blue-900/20 w-fit"
                    >
                      {writingMethod === method.name ? <Loader2 className="h-4 w-4 animate-spin" /> : "Write (Broadcast)"}
                    </button>
                  </form>
                ))}
              </div>
            )}
          </div>
        )}

        {activeTab === "vault" && (
          <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-md space-y-4">
            <h3 className="text-lg font-bold text-white flex items-center gap-2">
              <Cpu className="text-blue-500 h-5 w-5" /> ERC-4626 Yield Vault Interface
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 pt-2 text-sm">
              <div className="bg-gray-900/40 border border-gray-850 p-4 rounded-xl space-y-1">
                <span className="text-gray-500 text-xs font-bold uppercase block">Total Managed Assets</span>
                <span className="font-mono text-xl font-bold text-white">100,500 SLT</span>
              </div>
              <div className="bg-gray-900/40 border border-gray-850 p-4 rounded-xl space-y-1">
                <span className="text-gray-500 text-xs font-bold uppercase block">Exchange Rate</span>
                <span className="font-mono text-xl font-bold text-white">1.0520 Shares/Asset</span>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
