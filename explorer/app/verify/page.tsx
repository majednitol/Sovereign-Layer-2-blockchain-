"use client";

import React, { useState, useEffect } from "react";
import Link from "next/link";
import { ShieldCheck, Cpu, Code2, AlertCircle, CheckCircle2, Upload, FileCode2, Terminal, ExternalLink } from "lucide-react";
import { useWalletStore } from "@/store/wallet";

export default function VerifyPage() {
  const { connected, address, walletType, connectWallet, disconnectWallet } = useWalletStore();
  const [activeTab, setActiveTab] = useState<"evm" | "cosmwasm">("evm");
  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  // EVM Verify Form
  const [evmAddress, setEvmAddress] = useState("");
  const [compilerVersion, setCompilerVersion] = useState("v0.8.24+commit.e11b9ed9");
  const [contractCode, setContractCode] = useState("");
  const [optimizer, setOptimizer] = useState(true);
  const [evmAbi, setEvmAbi] = useState("");
  const [evmBytecode, setEvmBytecode] = useState("");
  const [constructorArgs, setConstructorArgs] = useState("");
  const [artifactJson, setArtifactJson] = useState("");

  // Wasm Verify Form
  const [codeId, setCodeId] = useState("");
  const [wasmFile, setWasmFile] = useState<File | null>(null);
  const [wasmName, setWasmName] = useState("");
  const [calculatedHash, setCalculatedHash] = useState("");
  const [instantiateSchema, setInstantiateSchema] = useState("");
  const [executeSchema, setExecuteSchema] = useState("");
  const [querySchema, setQuerySchema] = useState("");
  const [gitRepo, setGitRepo] = useState("");
  const [gitCommit, setGitCommit] = useState("");
  const [optimizerVersion, setOptimizerVersion] = useState("cosmwasm/rust-optimizer:0.14.0");

  // Reset success/error state when form inputs are modified
  useEffect(() => {
    setSuccess(false);
    setError(null);
  }, [evmAddress, compilerVersion, contractCode, optimizer, evmAbi, evmBytecode, constructorArgs, artifactJson]);

  const handleArtifactJsonChange = (val: string) => {
    setArtifactJson(val);
    if (!val.trim()) return;
    try {
      const parsed = JSON.parse(val);
      if (parsed.abi) {
        setEvmAbi(JSON.stringify(parsed.abi, null, 2));
      }
      if (parsed.bytecode) {
        if (typeof parsed.bytecode === "string") {
          setEvmBytecode(parsed.bytecode);
        } else if (parsed.bytecode.object) {
          setEvmBytecode(parsed.bytecode.object);
        }
      }
      if (parsed.deployedBytecode) {
        if (typeof parsed.deployedBytecode === "string") {
          setEvmBytecode(parsed.deployedBytecode);
        } else if (parsed.deployedBytecode.object) {
          setEvmBytecode(parsed.deployedBytecode.object);
        }
      }
    } catch (e) {
      // ignore parse errors for partial inputs
    }
  };

  const handleEvmVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!evmAddress) {
      setError("Contract address is required.");
      return;
    }
    if (!contractCode || !evmAbi || !evmBytecode) {
      setError("Solidity Source Code, Compiled ABI, and Bytecode are required.");
      return;
    }

    setLoading(true);
    setError(null);
    setSuccess(false);

    try {
      let parsedAbi;
      try {
        parsedAbi = JSON.parse(evmAbi);
      } catch (err) {
        throw new Error("Invalid ABI JSON format.");
      }

      const res = await fetch(`${API_BASE}/api/rest/v1/explorer/verify/evm`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          address: evmAddress,
          sourceCode: contractCode,
          abi: parsedAbi,
          compilerVersion,
          optimizerEnabled: optimizer,
          optimizerRuns: 200,
          constructorArgs: constructorArgs || "",
          compiledBytecode: evmBytecode,
        }),
      });

      if (!res.ok) {
        const errMsg = await res.text();
        throw new Error(errMsg || "Verification failed");
      }

      setSuccess(true);
    } catch (err: any) {
      setError(err.message || "An unexpected error occurred during EVM verification.");
    } finally {
      setLoading(false);
    }
  };

  const handleWasmFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setWasmFile(file);
    setWasmName(file.name);

    const arrayBuffer = await file.arrayBuffer();
    const hashBuffer = await crypto.subtle.digest("SHA-256", arrayBuffer);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    const hashHex = hashArray.map(b => b.toString(16).padStart(2, "0")).join("");
    setCalculatedHash(hashHex);
  };

  const handleWasmVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!codeId) {
      setError("Code ID is required.");
      return;
    }
    if (!calculatedHash) {
      setError("Please select a compiled WASM file.");
      return;
    }

    setLoading(true);
    setError(null);
    setSuccess(false);

    try {
      const parsedInstantiate = instantiateSchema ? JSON.parse(instantiateSchema) : {};
      const parsedExecute = executeSchema ? JSON.parse(executeSchema) : {};
      const parsedQuery = querySchema ? JSON.parse(querySchema) : {};

      const res = await fetch(`${API_BASE}/api/rest/v1/explorer/verify/cosmwasm`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          codeId: parseInt(codeId, 10),
          checksum: calculatedHash,
          instantiateMsg: parsedInstantiate,
          executeMsg: parsedExecute,
          queryMsg: parsedQuery,
          gitRepo: gitRepo || "",
          gitCommit: gitCommit || "",
          optimizerVersion: optimizerVersion || "",
        }),
      });

      if (!res.ok) {
        const errMsg = await res.text();
        throw new Error(errMsg || "Verification failed");
      }

      setSuccess(true);
    } catch (err: any) {
      setError(err.message || "An unexpected error occurred during CosmWasm verification.");
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
        <span className="text-gray-300">Contract Verification</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4 flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-white flex items-center gap-2">
            <ShieldCheck className="w-8 h-8 text-blue-500 animate-pulse" />
            Contract Verification Hub
          </h1>
          <p className="text-gray-400 mt-1">Verify smart contract source code and metadata on Sovereign L1.</p>
        </div>
      </div>

      {/* Selector Tabs */}
      <div className="flex border-b border-gray-850 gap-2">
        <button
          onClick={() => { setActiveTab("evm"); setSuccess(false); setError(null); }}
          className={`py-3 px-6 font-medium text-sm border-b-2 transition-all flex items-center gap-2 ${
            activeTab === "evm"
              ? "border-blue-500 text-blue-400 bg-blue-950/20"
              : "border-transparent text-gray-400 hover:text-gray-200"
          }`}
        >
          <Cpu className="w-4 h-4" />
          EVM Contract (Sourcify)
        </button>
        <button
          onClick={() => { setActiveTab("cosmwasm"); setSuccess(false); setError(null); }}
          className={`py-3 px-6 font-medium text-sm border-b-2 transition-all flex items-center gap-2 ${
            activeTab === "cosmwasm"
              ? "border-blue-500 text-blue-400 bg-blue-950/20"
              : "border-transparent text-gray-400 hover:text-gray-200"
          }`}
        >
          <Code2 className="w-4 h-4" />
          CosmWasm (SHA-256 Schema)
        </button>
      </div>

      {/* Message feedback */}
      {success && (
        <div className="p-4 bg-green-950/30 border border-green-800 rounded-lg flex items-start gap-3">
          <CheckCircle2 className="w-5 h-5 text-green-400 mt-0.5 flex-shrink-0" />
          <div>
            <h3 className="font-semibold text-green-400">Verification Successful</h3>
            <p className="text-sm text-green-300 mt-1">
              {activeTab === "evm"
                ? `EVM Contract at ${evmAddress} is now fully verified. The verified badge has been applied.`
                : `CosmWasm Code ID ${codeId} verified successfully. SHA-256 match found: ${calculatedHash.slice(0, 16)}...`}
            </p>
          </div>
        </div>
      )}

      {error && (
        <div className="p-4 bg-red-950/30 border border-red-800 rounded-lg flex items-start gap-3">
          <AlertCircle className="w-5 h-5 text-red-400 mt-0.5 flex-shrink-0" />
          <p className="text-sm text-red-300">{error}</p>
        </div>
      )}

      {/* Forms */}
      <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
        {activeTab === "evm" ? (
          <form onSubmit={handleEvmVerify} className="space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-2">
                <label className="block text-sm font-medium text-gray-300">Contract Address</label>
                <input
                  type="text"
                  placeholder="0x..."
                  value={evmAddress}
                  onChange={(e) => setEvmAddress(e.target.value)}
                  className="w-full bg-gray-950 border border-gray-800 rounded-lg px-4 py-2.5 text-white font-mono text-sm focus:outline-none focus:border-blue-500"
                  required
                />
              </div>

              <div className="space-y-2">
                <label className="block text-sm font-medium text-gray-300">Compiler Version</label>
                <select
                  value={compilerVersion}
                  onChange={(e) => setCompilerVersion(e.target.value)}
                  className="w-full bg-gray-950 border border-gray-800 rounded-lg px-4 py-2.5 text-white text-sm focus:outline-none focus:border-blue-500"
                >
                  <option>v0.8.24+commit.e11b9ed9</option>
                  <option>v0.8.20+commit.a1b79de6</option>
                  <option>v0.8.19+commit.7dd6d404</option>
                </select>
              </div>
            </div>

            <div className="space-y-2">
              <label className="block text-sm font-medium text-gray-300">Solidity Source Code</label>
              <textarea
                rows={5}
                placeholder="paste contract code here..."
                value={contractCode}
                onChange={(e) => setContractCode(e.target.value)}
                className="w-full bg-gray-950 border border-gray-800 rounded-lg px-4 py-2.5 text-white font-mono text-sm focus:outline-none focus:border-blue-500"
                required
              />
            </div>

            <div className="bg-gray-950 p-4 border border-gray-850 rounded-xl space-y-4">
              <div className="space-y-1">
                <label className="block text-xs font-bold text-gray-400 uppercase">Option A: Paste Hardhat/Foundry Compilation JSON Artifact</label>
                <textarea
                  rows={2}
                  placeholder='Paste {"abi": [...], "bytecode": "0x..."} artifact JSON'
                  value={artifactJson}
                  onChange={(e) => handleArtifactJsonChange(e.target.value)}
                  className="w-full bg-gray-900 border border-gray-800 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-blue-500"
                />
                <span className="text-[10px] text-gray-500 text-left block">Automatically populates ABI and Bytecode fields below.</span>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-1">
                  <label className="block text-xs font-bold text-gray-400 uppercase">Option B: Compiled ABI JSON</label>
                  <textarea
                    rows={4}
                    placeholder="[ ... ]"
                    value={evmAbi}
                    onChange={(e) => setEvmAbi(e.target.value)}
                    className="w-full bg-gray-900 border border-gray-800 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-blue-500"
                    required
                  />
                </div>
                <div className="space-y-1">
                  <label className="block text-xs font-bold text-gray-400 uppercase">Option B: Deployed Bytecode (Hex)</label>
                  <textarea
                    rows={4}
                    placeholder="0x..."
                    value={evmBytecode}
                    onChange={(e) => setEvmBytecode(e.target.value)}
                    className="w-full bg-gray-900 border border-gray-800 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-blue-500"
                    required
                  />
                </div>
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-2">
                <label className="block text-sm font-medium text-gray-300">Constructor Arguments (ABI Encoded Hex)</label>
                <input
                  type="text"
                  placeholder="e.g. 0000000000000000000000000000000000000000000000000000000000000064"
                  value={constructorArgs}
                  onChange={(e) => setConstructorArgs(e.target.value)}
                  className="w-full bg-gray-950 border border-gray-800 rounded-lg px-4 py-2.5 text-white font-mono text-sm focus:outline-none focus:border-blue-500"
                />
              </div>

              <div className="flex items-center gap-3 bg-gray-950 p-4 border border-gray-850 rounded-lg h-fit mt-7">
                <input
                  type="checkbox"
                  id="optimizer"
                  checked={optimizer}
                  onChange={(e) => setOptimizer(e.target.checked)}
                  className="rounded border-gray-800 bg-gray-900 text-blue-500 focus:ring-0 w-4 h-4"
                />
                <label htmlFor="optimizer" className="text-sm text-gray-300 cursor-pointer">
                  Optimization Enabled (Runs: 200)
                </label>
              </div>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 bg-blue-600 hover:bg-blue-500 disabled:bg-gray-850 text-white rounded-lg font-medium transition flex items-center justify-center gap-2"
            >
              {loading ? "Running Bytecode Verification..." : "Verify EVM Contract"}
            </button>
          </form>
        ) : (
          <form onSubmit={handleWasmVerify} className="space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-2">
                <label className="block text-sm font-medium text-gray-300">Code ID</label>
                <input
                  type="number"
                  placeholder="1"
                  value={codeId}
                  onChange={(e) => setCodeId(e.target.value)}
                  className="w-full bg-gray-950 border border-gray-800 rounded-lg px-4 py-2.5 text-white font-mono text-sm focus:outline-none focus:border-blue-500"
                  required
                />
              </div>

              <div className="space-y-2">
                <label className="block text-sm font-medium text-gray-300">Optimizer Image version</label>
                <input
                  type="text"
                  placeholder="cosmwasm/rust-optimizer:0.14.0"
                  value={optimizerVersion}
                  onChange={(e) => setOptimizerVersion(e.target.value)}
                  className="w-full bg-gray-950 border border-gray-800 rounded-lg px-4 py-2.5 text-white font-mono text-sm focus:outline-none focus:border-blue-500"
                />
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-2">
                <label className="block text-sm font-medium text-gray-300">Git Repository Link (Reproducible Build)</label>
                <input
                  type="text"
                  placeholder="https://github.com/org/repo"
                  value={gitRepo}
                  onChange={(e) => setGitRepo(e.target.value)}
                  className="w-full bg-gray-950 border border-gray-800 rounded-lg px-4 py-2.5 text-white text-sm focus:outline-none focus:border-blue-500"
                />
              </div>

              <div className="space-y-2">
                <label className="block text-sm font-medium text-gray-300">Git Commit Hash</label>
                <input
                  type="text"
                  placeholder="a1b2c3d4..."
                  value={gitCommit}
                  onChange={(e) => setGitCommit(e.target.value)}
                  className="w-full bg-gray-950 border border-gray-800 rounded-lg px-4 py-2.5 text-white font-mono text-sm focus:outline-none focus:border-blue-500"
                />
              </div>
            </div>

            <div className="space-y-2">
              <label className="block text-sm font-medium text-gray-300">Compiled WASM Binary</label>
              <div className="border-2 border-dashed border-gray-800 hover:border-gray-700 transition rounded-xl p-6 text-center cursor-pointer relative bg-gray-950">
                <input
                  type="file"
                  accept=".wasm"
                  onChange={handleWasmFileChange}
                  className="absolute inset-0 opacity-0 cursor-pointer w-full h-full"
                />
                <div className="space-y-2">
                  <Upload className="w-8 h-8 text-gray-500 mx-auto" />
                  <p className="text-sm text-gray-400">
                    {wasmName ? (
                      <span className="text-blue-400 font-mono">{wasmName}</span>
                    ) : (
                      "Click or drag '.wasm' file to upload"
                    )}
                  </p>
                </div>
              </div>
            </div>

            {calculatedHash && (
              <div className="p-4 bg-gray-950 border border-gray-850 rounded-lg space-y-1">
                <div className="text-xs text-gray-500 uppercase tracking-wider font-semibold text-left">Calculated SHA-256 Checksum</div>
                <div className="font-mono text-sm text-gray-300 break-all text-left">{calculatedHash}</div>
              </div>
            )}

            <div className="bg-gray-950 p-4 border border-gray-850 rounded-xl space-y-4">
              <div className="text-sm font-bold text-gray-400 uppercase text-left">JSON Schemas (Instantiate, Execute, Query)</div>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="space-y-1">
                  <label className="block text-[10px] font-bold text-gray-500 uppercase text-left">InstantiateMsg Schema</label>
                  <textarea
                    rows={4}
                    placeholder='{"$schema": ...}'
                    value={instantiateSchema}
                    onChange={(e) => setInstantiateSchema(e.target.value)}
                    className="w-full bg-gray-900 border border-gray-800 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-blue-500"
                  />
                </div>
                <div className="space-y-1">
                  <label className="block text-[10px] font-bold text-gray-500 uppercase text-left">ExecuteMsg Schema</label>
                  <textarea
                    rows={4}
                    placeholder='{"oneOf": [...] || "$schema": ...}'
                    value={executeSchema}
                    onChange={(e) => setExecuteSchema(e.target.value)}
                    className="w-full bg-gray-900 border border-gray-800 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-blue-500"
                  />
                </div>
                <div className="space-y-1">
                  <label className="block text-[10px] font-bold text-gray-500 uppercase text-left">QueryMsg Schema</label>
                  <textarea
                    rows={4}
                    placeholder='{"oneOf": [...] || "$schema": ...}'
                    value={querySchema}
                    onChange={(e) => setQuerySchema(e.target.value)}
                    className="w-full bg-gray-900 border border-gray-800 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-blue-500"
                  />
                </div>
              </div>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 bg-blue-600 hover:bg-blue-500 disabled:bg-gray-850 text-white rounded-lg font-medium transition flex items-center justify-center gap-2"
            >
              {loading ? "Verifying CosmWasm Code..." : "Verify CosmWasm Code"}
            </button>
          </form>
        )}
      </div>
    </div>
  );
}
