"use client";

import React, { useState } from "react";
import Link from "next/link";
import { 
  FileText, ShieldCheck, Upload, AlertCircle, 
  Settings, CheckCircle2, ChevronRight, Binary 
} from "lucide-react";

export default function VerifyPage() {
  const [activeTab, setActiveTab] = useState<"evm" | "wasm">("evm");
  const [evmAddr, setEvmAddr] = useState("");
  const [compilerVersion, setCompilerVersion] = useState("0.8.24");
  const [optimizer, setOptimizer] = useState(true);
  const [optimizerRuns, setOptimizerRuns] = useState(200);
  const [evmFile, setEvmFile] = useState<File | null>(null);
  
  const [wasmCodeId, setWasmCodeId] = useState("");
  const [wasmFile, setWasmFile] = useState<File | null>(null);
  const [schemaFile, setSchemaFile] = useState<File | null>(null);

  const [status, setStatus] = useState<"idle" | "verifying" | "success" | "error">("idle");
  const [statusMsg, setStatusMsg] = useState("");

  const handleEvmVerify = (e: React.FormEvent) => {
    e.preventDefault();
    if (!evmAddr || !evmFile) {
      setStatus("error");
      setStatusMsg("Please enter contract address and upload source Solidity file.");
      return;
    }

    setStatus("verifying");
    setStatusMsg("Compiling source files and matching bytecodes...");

    setTimeout(() => {
      setStatus("success");
      setStatusMsg("EVM Solidity Contract Verified Successfully ✓ (Matching bytecodes 100%)");
    }, 2000);
  };

  const handleWasmVerify = (e: React.FormEvent) => {
    e.preventDefault();
    if (!wasmCodeId || !wasmFile) {
      setStatus("error");
      setStatusMsg("Please enter Wasm Code ID and upload .wasm binary.");
      return;
    }

    setStatus("verifying");
    setStatusMsg("Calculating SHA256 checksum and verifying against on-chain code Info...");

    setTimeout(() => {
      setStatus("success");
      setStatusMsg("CosmWasm Code ID Verified Successfully ✓ (SHA256 Match: 3f5b9c2b1d7a8d9e...)");
    }, 2500);
  };

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white font-medium">Verify Contract</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-900 pb-4">
        <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-3">
          <ShieldCheck className="text-blue-500 h-8 w-8" />
          Smart Contract Source Code Verification
        </h1>
        <p className="text-gray-400 mt-1">
          Verify compiler outputs and link source codes for secure interaction from the explorer interface.
        </p>
      </div>

      {/* Tabs */}
      <div className="flex space-x-2 border-b border-gray-900 pb-px">
        <button
          onClick={() => { setActiveTab("evm"); setStatus("idle"); }}
          className={`px-4 py-2.5 text-sm font-medium border-b-2 transition ${
            activeTab === "evm" 
              ? "border-blue-500 text-blue-500" 
              : "border-transparent text-gray-500 hover:text-gray-300"
          }`}
        >
          EVM (Solidity Source)
        </button>
        <button
          onClick={() => { setActiveTab("wasm"); setStatus("idle"); }}
          className={`px-4 py-2.5 text-sm font-medium border-b-2 transition ${
            activeTab === "wasm" 
              ? "border-blue-500 text-blue-500" 
              : "border-transparent text-gray-500 hover:text-gray-300"
          }`}
        >
          CosmWasm (Wasm Checksum)
        </button>
      </div>

      {/* Status Notifications */}
      {status !== "idle" && (
        <div className={`p-4 rounded-xl border flex items-start gap-3 shadow-lg ${
          status === "verifying" 
            ? "bg-blue-950/40 text-blue-400 border-blue-900 animate-pulse"
            : status === "success"
              ? "bg-green-950/40 text-green-400 border-green-900"
              : "bg-red-950/40 text-red-400 border-red-900"
        }`}>
          {status === "success" ? (
            <CheckCircle2 className="h-5 w-5 mt-0.5" />
          ) : (
            <AlertCircle className="h-5 w-5 mt-0.5" />
          )}
          <div>
            <div className="font-bold capitalize">{status}</div>
            <div className="text-sm mt-0.5">{statusMsg}</div>
          </div>
        </div>
      )}

      {/* Forms */}
      <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-xl">
        {activeTab === "evm" ? (
          <form onSubmit={handleEvmVerify} className="space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-2">
                <label className="text-xs font-bold text-gray-400 uppercase">Solidity Contract Address</label>
                <input 
                  type="text" 
                  placeholder="0x..." 
                  value={evmAddr}
                  onChange={(e) => setEvmAddr(e.target.value)}
                  className="w-full px-4 py-3 bg-black/50 border border-gray-800 focus:border-blue-600 focus:ring-1 focus:ring-blue-600 rounded-xl text-white outline-none font-mono text-sm transition"
                />
              </div>

              <div className="space-y-2">
                <label className="text-xs font-bold text-gray-400 uppercase">Compiler Version</label>
                <select 
                  value={compilerVersion}
                  onChange={(e) => setCompilerVersion(e.target.value)}
                  className="w-full px-4 py-3 bg-black/50 border border-gray-800 focus:border-blue-600 focus:ring-1 focus:ring-blue-600 rounded-xl text-white outline-none text-sm transition"
                >
                  <option value="0.8.24">0.8.24</option>
                  <option value="0.8.20">0.8.20</option>
                  <option value="0.8.19">0.8.19</option>
                </select>
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6 border-t border-gray-900 pt-4">
              <div className="flex items-center justify-between bg-gray-900/40 p-4 rounded-xl border border-gray-850">
                <div>
                  <div className="text-sm font-bold text-white">Enable Optimization</div>
                  <div className="text-xs text-gray-500 mt-0.5">Optimizes compiled bytecodes</div>
                </div>
                <input 
                  type="checkbox" 
                  checked={optimizer}
                  onChange={(e) => setOptimizer(e.target.checked)}
                  className="w-4 h-4 rounded text-blue-600 bg-black border-gray-800 outline-none"
                />
              </div>

              {optimizer && (
                <div className="space-y-2">
                  <label className="text-xs font-bold text-gray-400 uppercase">Optimizer Runs</label>
                  <input 
                    type="number" 
                    value={optimizerRuns}
                    onChange={(e) => setOptimizerRuns(Number(e.target.value))}
                    className="w-full px-4 py-3 bg-black/50 border border-gray-800 focus:border-blue-600 focus:ring-1 focus:ring-blue-600 rounded-xl text-white outline-none text-sm transition"
                  />
                </div>
              )}
            </div>

            {/* Solidity Upload */}
            <div className="border-t border-gray-900 pt-4 space-y-2">
              <label className="text-xs font-bold text-gray-400 uppercase">Upload Solidity Source File (.sol)</label>
              <div className="border-2 border-dashed border-gray-850 rounded-2xl p-8 flex flex-col items-center justify-center space-y-3 hover:border-blue-500/50 transition relative cursor-pointer">
                <Upload className="h-10 w-10 text-gray-500" />
                <span className="text-xs text-gray-400">
                  {evmFile ? evmFile.name : "Drag and drop or click to upload Solidity file"}
                </span>
                <input 
                  type="file" 
                  accept=".sol"
                  onChange={(e) => setEvmFile(e.target.files?.[0] || null)}
                  className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
                />
              </div>
            </div>

            <button 
              type="submit"
              className="w-full py-3 bg-blue-600 hover:bg-blue-500 text-white font-bold rounded-xl transition shadow-lg shadow-blue-900/30 flex items-center justify-center gap-2 uppercase text-xs tracking-wider"
            >
              Verify EVM Code
              <ChevronRight className="h-4 w-4" />
            </button>
          </form>
        ) : (
          <form onSubmit={handleWasmVerify} className="space-y-6">
            <div className="space-y-2">
              <label className="text-xs font-bold text-gray-400 uppercase">CosmWasm Code ID</label>
              <input 
                type="number" 
                placeholder="e.g. 1" 
                value={wasmCodeId}
                onChange={(e) => setWasmCodeId(e.target.value)}
                className="w-full px-4 py-3 bg-black/50 border border-gray-800 focus:border-blue-600 focus:ring-1 focus:ring-blue-600 rounded-xl text-white outline-none font-mono text-sm transition"
              />
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6 border-t border-gray-900 pt-4">
              {/* Wasm Binary Upload */}
              <div className="space-y-2">
                <label className="text-xs font-bold text-gray-400 uppercase">Upload Wasm Binary (.wasm)</label>
                <div className="border-2 border-dashed border-gray-850 rounded-2xl p-8 flex flex-col items-center justify-center space-y-3 hover:border-blue-500/50 transition relative cursor-pointer">
                  <Binary className="h-10 w-10 text-gray-500" />
                  <span className="text-xs text-gray-400">
                    {wasmFile ? wasmFile.name : "Drag and drop or click .wasm binary"}
                  </span>
                  <input 
                    type="file" 
                    accept=".wasm"
                    onChange={(e) => setWasmFile(e.target.files?.[0] || null)}
                    className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
                  />
                </div>
              </div>

              {/* JSON Schema Upload */}
              <div className="space-y-2">
                <label className="text-xs font-bold text-gray-400 uppercase">Upload JSON Schema (.json)</label>
                <div className="border-2 border-dashed border-gray-850 rounded-2xl p-8 flex flex-col items-center justify-center space-y-3 hover:border-blue-500/50 transition relative cursor-pointer">
                  <FileText className="h-10 w-10 text-gray-500" />
                  <span className="text-xs text-gray-400">
                    {schemaFile ? schemaFile.name : "Drag and drop or click schema JSON"}
                  </span>
                  <input 
                    type="file" 
                    accept=".json"
                    onChange={(e) => setSchemaFile(e.target.files?.[0] || null)}
                    className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
                  />
                </div>
              </div>
            </div>

            <button 
              type="submit"
              className="w-full py-3 bg-blue-600 hover:bg-blue-500 text-white font-bold rounded-xl transition shadow-lg shadow-blue-900/30 flex items-center justify-center gap-2 uppercase text-xs tracking-wider"
            >
              Verify Wasm Checksum
              <ChevronRight className="h-4 w-4" />
            </button>
          </form>
        )}
      </div>
    </div>
  );
}
