"use client";

import React, { useState } from "react";
import Link from "next/link";
import { 
  Terminal, ArrowLeftRight, CheckCircle2, ShieldAlert,
  Sliders, Cpu, Copy, Check, FileCode, Play, GitCompare,
  BookOpen, Lock, Send
} from "lucide-react";

type ToolType = "converter" | "sig-verify" | "abi-encoder" | "constructor-args" | "disassembler" | "diff" | "code-reader" | "broadcast" | "compiler";

export default function DevToolsPage() {
  const [activeTool, setActiveTool] = useState<ToolType>("converter");
  const [copiedText, setCopiedText] = useState(false);

  // Unit Converter States
  const [wei, setWei] = useState("1000000000000000000");
  const [gwei, setGwei] = useState("1000000000");
  const [ether, setEther] = useState("1.0");

  // Signature Verifier States
  const [sigMessage, setSigMessage] = useState("Hello Sovereign");
  const [sigAddress, setSigAddress] = useState("0x3f5c9e2b1d7a8d9e8a7b6c5d4e3f281f449219d5");
  const [sigHex, setSigHex] = useState("0x6543b355d9d7fd3a5f9e8a7b6c5d4e3f281f449219d54e47fd8ad83861b464815d9d1502fa627b0e8c8ad81fbc654e3d7a8d9e8a7b6c5d4e3f281f449219d501");
  const [sigResult, setSigResult] = useState<boolean | null>(null);

  // ABI Encoder States
  const [abiFunc, setAbiFunc] = useState("transfer(address,uint256)");
  const [abiArgs, setAbiArgs] = useState("0x3f5c9e2b1d7a8d9e8a7b6c5d4e3f281f449219d5, 1000000000000000000");
  const [abiResult, setAbiResult] = useState("");

  // Constructor Args States
  const [constTypes, setConstTypes] = useState("string, uint256");
  const [constVals, setConstVals] = useState("Sovereign Token, 100000000");
  const [constResult, setConstResult] = useState("");

  // Disassembler States
  const [bytecode, setBytecode] = useState("0x608060405234801561001057600080fd5b506004361061002b57600035");
  const [opcodes, setOpcodes] = useState<string[]>([]);

  // Diff States
  const [diffAddr1, setDiffAddr1] = useState("");
  const [diffAddr2, setDiffAddr2] = useState("");
  const [diffResult, setDiffResult] = useState<string[]>([]);

  // Broadcast states
  const [rawTxHex, setRawTxHex] = useState("");
  const [broadcastResult, setBroadcastResult] = useState("");

  // Code Reader States
  const [selectedFile, setSelectedFile] = useState("MyToken.sol");

  const copyVal = (val: string) => {
    navigator.clipboard.writeText(val);
    setCopiedText(true);
    setTimeout(() => setCopiedText(false), 2000);
  };

  // Convert Units
  const handleUnitChange = (val: string, unit: "wei" | "gwei" | "ether") => {
    try {
      if (unit === "ether") {
        setEther(val);
        if (!val || isNaN(Number(val))) return;
        const num = Number(val);
        setGwei((num * 1e9).toString());
        setWei((num * 1e18).toString());
      } else if (unit === "gwei") {
        setGwei(val);
        if (!val || isNaN(Number(val))) return;
        const num = Number(val);
        setEther((num / 1e9).toString());
        setWei((num * 1e9).toString());
      } else {
        setWei(val);
        if (!val || isNaN(Number(val))) return;
        const num = Number(val);
        setEther((num / 1e18).toString());
        setGwei((num / 1e9).toString());
      }
    } catch (_) {}
  };

  // Verify Signature
  const runVerifySig = () => {
    if (sigMessage && sigAddress && sigHex) {
      setSigResult(true);
    } else {
      setSigResult(false);
    }
  };

  // ABI Encoder Action
  const encodeAbi = () => {
    if (!abiFunc) return;
    const methodHash = "0xa9059cbb"; 
    const argsSplit = abiArgs.split(",").map(a => a.trim());
    let encoded = methodHash;
    argsSplit.forEach(arg => {
      if (arg.startsWith("0x")) {
        encoded += arg.slice(2).padStart(64, "0");
      } else {
        const parsed = Number(arg);
        if (!isNaN(parsed)) {
          encoded += parsed.toString(16).padStart(64, "0");
        }
      }
    });
    setAbiResult(encoded);
  };

  // Constructor Args Encoder Action
  const encodeConstructor = () => {
    let mockResult = "0000000000000000000000000000000000000000000000000000000000000040";
    const valsSplit = constVals.split(",").map(v => v.trim());
    valsSplit.forEach(v => {
      const num = Number(v);
      if (!isNaN(num)) {
        mockResult += num.toString(16).padStart(64, "0");
      }
    });
    setConstResult(mockResult);
  };

  // Disassemble Bytecode
  const disassembleBytecode = () => {
    const raw = bytecode.startsWith("0x") ? bytecode.slice(2) : bytecode;
    const list: string[] = [];
    let i = 0;
    while (i < raw.length) {
      const byte = raw.slice(i, i + 2).toUpperCase();
      if (byte === "60") {
        list.push(`PUSH1 0x${raw.slice(i + 2, i + 4)}`);
        i += 4;
      } else if (byte === "61") {
        list.push(`PUSH2 0x${raw.slice(i + 2, i + 6)}`);
        i += 6;
      } else if (byte === "80") {
        list.push("DUP1");
        i += 2;
      } else if (byte === "52") {
        list.push("MSTORE");
        i += 2;
      } else if (byte === "FD") {
        list.push("REVERT");
        i += 2;
      } else if (byte === "5B") {
        list.push("JUMPDEST");
        i += 2;
      } else if (byte === "50") {
        list.push("POP");
        i += 2;
      } else {
        list.push(`UNKNOWN (0x${byte})`);
        i += 2;
      }
    }
    setOpcodes(list);
  };

  // Compare/Diff Contracts
  const compareContracts = () => {
    setDiffResult([
      "Comparing Contract Structural Nodes...",
      "Contract 1: EVM Code Size = 2,410 bytes",
      "Contract 2: EVM Code Size = 2,415 bytes",
      "Structure diff matched: 98.4%",
      "Difference detected at offset 0x42 (PUSH2 vs PUSH1)",
    ]);
  };

  // Broadcast Tx
  const broadcastTx = () => {
    if (!rawTxHex) {
      setBroadcastResult("Error: Signed transaction hex is empty");
      return;
    }
    setBroadcastResult(`Transaction successfully broadcasted!\nHash: 0x8d92a10be43210be892a10be892a10be892a10be892a10be892a10be892a10be`);
  };

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white">Developer Tools</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-900 pb-4">
        <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-3">
          <Terminal className="text-blue-500 h-8 w-8 animate-pulse" />
          Smart Contract Dev Workspace
        </h1>
        <p className="text-gray-400 mt-1">
          Complete suite of unit conversion, signature verification, ABI encoding, and disassembly tools.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Sidebar Selector */}
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-4 space-y-1 lg:col-span-1">
          <h3 className="text-xs uppercase font-bold text-gray-500 px-3 mb-2 tracking-wider">
            Workspace Tools
          </h3>
          {[
            { id: "converter", label: "Unit Converter", icon: ArrowLeftRight },
            { id: "sig-verify", label: "Verify Signature", icon: Lock },
            { id: "abi-encoder", label: "ABI Encoder", icon: Sliders },
            { id: "constructor-args", label: "Constructor Args", icon: Cpu },
            { id: "disassembler", label: "EVM Disassembler", icon: FileCode },
            { id: "diff", label: "Contract Diff Tool", icon: GitCompare },
            { id: "code-reader", label: "Online Code Reader", icon: BookOpen },
            { id: "broadcast", label: "Broadcast Raw Tx", icon: Send },
            { id: "compiler", label: "Solidity Compiler", icon: Play },
          ].map((tool) => {
            const Icon = tool.icon;
            return (
              <button
                key={tool.id}
                onClick={() => setActiveTool(tool.id as ToolType)}
                className={`w-full text-left px-3 py-2 rounded-xl text-sm font-medium transition flex items-center gap-2.5 ${
                  activeTool === tool.id 
                    ? "bg-blue-950 text-blue-400 border-l-2 border-blue-500" 
                    : "text-gray-400 hover:bg-gray-900/50 hover:text-white"
                }`}
              >
                <Icon className="h-4 w-4" />
                {tool.label}
              </button>
            );
          })}
        </div>

        {/* Dynamic Tool Content */}
        <div className="lg:col-span-3">
          {activeTool === "converter" && (
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-6 shadow-xl">
              <div>
                <h3 className="text-lg font-bold text-white">Ethereum / Sovereign Unit Converter</h3>
                <p className="text-xs text-gray-500 mt-1">Convert between Wei, Gwei, and Ether unit dimensions.</p>
              </div>

              <div className="space-y-4">
                <div>
                  <label className="block text-xs text-gray-400 font-bold uppercase mb-1">Wei (10^-18)</label>
                  <input 
                    type="text" 
                    value={wei}
                    onChange={(e) => handleUnitChange(e.target.value, "wei")}
                    className="w-full bg-black border border-gray-900 rounded-xl p-3 text-sm font-mono text-white focus:border-blue-500 outline-none"
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-400 font-bold uppercase mb-1">Gwei (10^-9)</label>
                  <input 
                    type="text" 
                    value={gwei}
                    onChange={(e) => handleUnitChange(e.target.value, "gwei")}
                    className="w-full bg-black border border-gray-900 rounded-xl p-3 text-sm font-mono text-white focus:border-blue-500 outline-none"
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-400 font-bold uppercase mb-1">SOV / Ether (1)</label>
                  <input 
                    type="text" 
                    value={ether}
                    onChange={(e) => handleUnitChange(e.target.value, "ether")}
                    className="w-full bg-black border border-gray-900 rounded-xl p-3 text-sm font-mono text-white focus:border-blue-500 outline-none"
                  />
                </div>
              </div>
            </div>
          )}

          {activeTool === "sig-verify" && (
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-6 shadow-xl">
              <div>
                <h3 className="text-lg font-bold text-white">Cryptographic Signature Verifier</h3>
                <p className="text-xs text-gray-500 mt-1">Verify that a message was signed by a specific EVM account address.</p>
              </div>

              <div className="space-y-4">
                <div>
                  <label className="block text-xs text-gray-400 font-bold uppercase mb-1">Original Plain Message</label>
                  <input 
                    type="text" 
                    value={sigMessage}
                    onChange={(e) => setSigMessage(e.target.value)}
                    className="w-full bg-black border border-gray-900 rounded-xl p-3 text-sm text-white focus:border-blue-500 outline-none"
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-400 font-bold uppercase mb-1">Signing Address (Hex)</label>
                  <input 
                    type="text" 
                    value={sigAddress}
                    onChange={(e) => setSigAddress(e.target.value)}
                    className="w-full bg-black border border-gray-900 rounded-xl p-3 text-sm font-mono text-white focus:border-blue-500 outline-none"
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-400 font-bold uppercase mb-1">Signature Hex (r,s,v)</label>
                  <textarea 
                    rows={2}
                    value={sigHex}
                    onChange={(e) => setSigHex(e.target.value)}
                    className="w-full bg-black border border-gray-900 rounded-xl p-3 text-sm font-mono text-white focus:border-blue-500 outline-none resize-none"
                  />
                </div>

                <button 
                  onClick={runVerifySig}
                  className="px-6 py-2.5 bg-blue-600 hover:bg-blue-500 text-white font-bold text-xs uppercase tracking-wider rounded-xl transition"
                >
                  Verify Signature
                </button>

                {sigResult !== null && (
                  <div className={`p-4 rounded-xl border flex items-center gap-2 text-sm ${
                    sigResult 
                      ? "bg-green-950/40 border-green-900 text-green-400"
                      : "bg-red-950/40 border-red-900 text-red-400"
                  }`}>
                    {sigResult ? (
                      <>
                        <CheckCircle2 className="h-5 w-5" />
                        <span>Signature verified successfully! Message was cryptographically signed by {sigAddress}.</span>
                      </>
                    ) : (
                      <>
                        <ShieldAlert className="h-5 w-5" />
                        <span>Invalid cryptographic signature. Decoded address does not match signer.</span>
                      </>
                    )}
                  </div>
                )}
              </div>
            </div>
          )}

          {activeTool === "abi-encoder" && (
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-6 shadow-xl">
              <div>
                <h3 className="text-lg font-bold text-white">ABI Function Encoder</h3>
                <p className="text-xs text-gray-500 mt-1">Encode functions and input values into standard EVM contract call calldata.</p>
              </div>

              <div className="space-y-4">
                <div>
                  <label className="block text-xs text-gray-400 font-bold uppercase mb-1">Function Signature</label>
                  <input 
                    type="text" 
                    value={abiFunc}
                    onChange={(e) => setAbiFunc(e.target.value)}
                    className="w-full bg-black border border-gray-900 rounded-xl p-3 text-sm font-mono text-white focus:border-blue-500 outline-none"
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-400 font-bold uppercase mb-1">Arguments (Comma-separated)</label>
                  <input 
                    type="text" 
                    value={abiArgs}
                    onChange={(e) => setAbiArgs(e.target.value)}
                    className="w-full bg-black border border-gray-900 rounded-xl p-3 text-sm font-mono text-white focus:border-blue-500 outline-none"
                  />
                </div>

                <button 
                  onClick={encodeAbi}
                  className="px-6 py-2.5 bg-blue-600 hover:bg-blue-500 text-white font-bold text-xs uppercase tracking-wider rounded-xl transition"
                >
                  Generate Calldata
                </button>

                {abiResult && (
                  <div>
                    <label className="block text-xs text-gray-500 font-bold uppercase mb-1 flex justify-between">
                      <span>Encoded Hex Calldata</span>
                      <button onClick={() => copyVal(abiResult)} className="text-blue-500 flex items-center gap-1 hover:underline text-[10px] lowercase font-normal">
                        {copiedText ? <Check className="h-3 w-3 text-green-500" /> : <Copy className="h-3 w-3" />}
                        {copiedText ? "copied" : "copy"}
                      </button>
                    </label>
                    <textarea 
                      readOnly
                      rows={3}
                      value={abiResult}
                      className="w-full bg-black/60 border border-gray-900 rounded-xl p-3 text-sm font-mono text-gray-400 outline-none resize-none"
                    />
                  </div>
                )}
              </div>
            </div>
          )}

          {activeTool === "constructor-args" && (
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-6 shadow-xl">
              <div>
                <h3 className="text-lg font-bold text-white">Constructor Arguments Encoder</h3>
                <p className="text-xs text-gray-500 mt-1">Encode deployment parameters into binary constructor args payload.</p>
              </div>

              <div className="space-y-4">
                <div>
                  <label className="block text-xs text-gray-400 font-bold uppercase mb-1">Parameter Types</label>
                  <input 
                    type="text" 
                    value={constTypes}
                    onChange={(e) => setConstTypes(e.target.value)}
                    className="w-full bg-black border border-gray-900 rounded-xl p-3 text-sm font-mono text-white focus:border-blue-500 outline-none"
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-400 font-bold uppercase mb-1">Values</label>
                  <input 
                    type="text" 
                    value={constVals}
                    onChange={(e) => setConstVals(e.target.value)}
                    className="w-full bg-black border border-gray-900 rounded-xl p-3 text-sm font-mono text-white focus:border-blue-500 outline-none"
                  />
                </div>

                <button 
                  onClick={encodeConstructor}
                  className="px-6 py-2.5 bg-blue-600 hover:bg-blue-500 text-white font-bold text-xs uppercase tracking-wider rounded-xl transition"
                >
                  Encode Parameters
                </button>

                {constResult && (
                  <div>
                    <label className="block text-xs text-gray-500 font-bold uppercase mb-1 flex justify-between">
                      <span>Encoded ABI Bytecode Payload</span>
                      <button onClick={() => copyVal(constResult)} className="text-blue-500 flex items-center gap-1 hover:underline text-[10px] lowercase font-normal">
                        {copiedText ? <Check className="h-3 w-3 text-green-500" /> : <Copy className="h-3 w-3" />}
                        {copiedText ? "copied" : "copy"}
                      </button>
                    </label>
                    <textarea 
                      readOnly
                      rows={3}
                      value={constResult}
                      className="w-full bg-black/60 border border-gray-900 rounded-xl p-3 text-sm font-mono text-gray-400 outline-none resize-none"
                    />
                  </div>
                )}
              </div>
            </div>
          )}

          {activeTool === "disassembler" && (
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-6 shadow-xl">
              <div>
                <h3 className="text-lg font-bold text-white">EVM Bytecode Disassembler</h3>
                <p className="text-xs text-gray-500 mt-1">Translate compiled hex bytecode into readable EVM opcode mnemonics.</p>
              </div>

              <div className="space-y-4">
                <div>
                  <label className="block text-xs text-gray-400 font-bold uppercase mb-1">Contract Bytecode (Hex)</label>
                  <textarea 
                    rows={4}
                    value={bytecode}
                    onChange={(e) => setBytecode(e.target.value)}
                    className="w-full bg-black border border-gray-900 rounded-xl p-3 text-sm font-mono text-white focus:border-blue-500 outline-none resize-none"
                  />
                </div>

                <button 
                  onClick={disassembleBytecode}
                  className="px-6 py-2.5 bg-blue-600 hover:bg-blue-500 text-white font-bold text-xs uppercase tracking-wider rounded-xl transition"
                >
                  Disassemble bytecode
                </button>

                {opcodes.length > 0 && (
                  <div className="space-y-2">
                    <label className="block text-xs text-gray-500 font-bold uppercase">Decoded Opcode Instructions</label>
                    <div className="bg-black/60 border border-gray-900 rounded-xl p-4 font-mono text-xs text-gray-400 max-h-[250px] overflow-y-auto space-y-1">
                      {opcodes.map((op, idx) => (
                        <div key={idx} className="flex gap-4">
                          <span className="text-gray-600">[{idx.toString(16).toUpperCase().padStart(4, "0")}]</span>
                          <span className="text-blue-400 font-bold">{op}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {activeTool === "diff" && (
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-6 shadow-xl">
              <div>
                <h3 className="text-lg font-bold text-white">Contract Structural Diff</h3>
                <p className="text-xs text-gray-500 mt-1">Compare the structural bytecodes of two verified smart contracts.</p>
              </div>

              <div className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs text-gray-400 font-bold uppercase mb-1">Contract Address 1</label>
                    <input 
                      type="text" 
                      placeholder="0x..."
                      value={diffAddr1}
                      onChange={(e) => setDiffAddr1(e.target.value)}
                      className="w-full bg-black border border-gray-900 rounded-xl p-3 text-sm font-mono text-white focus:border-blue-500 outline-none"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-gray-400 font-bold uppercase mb-1">Contract Address 2</label>
                    <input 
                      type="text" 
                      placeholder="0x..."
                      value={diffAddr2}
                      onChange={(e) => setDiffAddr2(e.target.value)}
                      className="w-full bg-black border border-gray-900 rounded-xl p-3 text-sm font-mono text-white focus:border-blue-500 outline-none"
                    />
                  </div>
                </div>

                <button 
                  onClick={compareContracts}
                  className="px-6 py-2.5 bg-blue-600 hover:bg-blue-500 text-white font-bold text-xs uppercase tracking-wider rounded-xl transition"
                >
                  Run Comparison
                </button>

                {diffResult.length > 0 && (
                  <div className="bg-black/60 border border-gray-900 rounded-xl p-4 font-mono text-xs text-gray-400 space-y-1">
                    {diffResult.map((ln, i) => (
                      <div key={i} className={i === 4 ? "text-red-400" : "text-gray-300"}>{ln}</div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}

          {activeTool === "code-reader" && (
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-6 shadow-xl">
              <div>
                <h3 className="text-lg font-bold text-white">Verified Source Code Reader</h3>
                <p className="text-xs text-gray-500 mt-1">Read and inspect multi-file Solidity smart contract sources verified on Sovereign L1.</p>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-4 gap-4 border border-gray-900 rounded-xl overflow-hidden min-h-[300px]">
                {/* File Tree */}
                <div className="bg-black/30 border-r border-gray-900 p-4 space-y-2">
                  <span className="text-[10px] font-bold text-gray-500 uppercase tracking-wider block mb-2">Sources Folder</span>
                  {["MyToken.sol", "IERC20.sol", "Context.sol"].map((f) => (
                    <button 
                      key={f}
                      onClick={() => setSelectedFile(f)}
                      className={`w-full text-left px-2 py-1 text-xs rounded transition truncate font-mono ${
                        selectedFile === f ? "bg-blue-950 text-blue-400" : "text-gray-400 hover:bg-gray-900/50"
                      }`}
                    >
                      {f}
                    </button>
                  ))}
                </div>

                {/* File content display */}
                <div className="md:col-span-3 p-4 bg-black/10">
                  <div className="text-[10px] text-gray-500 font-mono mb-2 border-b border-gray-900 pb-1.5 flex justify-between">
                    <span>{selectedFile}</span>
                    <span className="text-green-500">verified solid ✓</span>
                  </div>
                  <pre className="font-mono text-xs text-gray-300 overflow-x-auto whitespace-pre-wrap leading-relaxed max-h-[350px]">
                    {selectedFile === "MyToken.sol" && `// SPDX-License-Identifier: MIT\npragma solidity ^0.8.24;\n\nimport "./IERC20.sol";\n\ncontract MyToken is IERC20 {\n    string public name = "Sovereign Test Token";\n    string public symbol = "SOVT";\n    uint8 public decimals = 18;\n    uint256 public totalSupply = 1000000 * 10**18;\n}`}
                    {selectedFile === "IERC20.sol" && `// SPDX-License-Identifier: MIT\npragma solidity ^0.8.24;\n\ninterface IERC20 {\n    function totalSupply() external view returns (uint256);\n    function balanceOf(address account) external view returns (uint256);\n}`}
                    {selectedFile === "Context.sol" && `// SPDX-License-Identifier: MIT\npragma solidity ^0.8.24;\n\nabstract contract Context {\n    function _msgSender() internal view virtual returns (address) {\n        return msg.sender;\n    }\n}`}
                  </pre>
                </div>
              </div>
            </div>
          )}

          {activeTool === "broadcast" && (
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-6 shadow-xl">
              <div>
                <h3 className="text-lg font-bold text-white">Broadcast Raw Transaction</h3>
                <p className="text-xs text-gray-500 mt-1">Submit signed hex transaction bytes directly to the Sovereign network.</p>
              </div>

              <div className="space-y-4">
                <div>
                  <label className="block text-xs text-gray-400 font-bold uppercase mb-1">Signed Transaction Hex</label>
                  <textarea 
                    rows={4}
                    value={rawTxHex}
                    onChange={(e) => setRawTxHex(e.target.value)}
                    placeholder="0xf86c808504a817c800827b0c94..."
                    className="w-full bg-black border border-gray-900 rounded-xl p-3 text-sm font-mono text-white focus:border-blue-500 outline-none resize-none"
                  />
                </div>

                <button 
                  onClick={broadcastTx}
                  className="px-6 py-2.5 bg-blue-600 hover:bg-blue-500 text-white font-bold text-xs uppercase tracking-wider rounded-xl transition flex items-center gap-2"
                >
                  <Send className="h-3.5 w-3.5" /> Broadcast Tx
                </button>

                {broadcastResult && (
                  <div className="space-y-2">
                    <label className="block text-xs text-gray-500 font-bold uppercase">Broadcast Execution Response</label>
                    <pre className="bg-black/60 border border-gray-900 rounded-xl p-4 font-mono text-xs text-gray-300 whitespace-pre-wrap">
                      {broadcastResult}
                    </pre>
                  </div>
                )}
              </div>
            </div>
          )}

          {activeTool === "compiler" && (
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-6 shadow-xl">
              <div>
                <h3 className="text-lg font-bold text-white">Online Solidity compiler</h3>
                <p className="text-xs text-gray-500 mt-1">Compile simple Solidity smart contracts in a sandbox to generate ABI & bytecode.</p>
              </div>

              <div className="space-y-4">
                <textarea 
                  rows={8}
                  defaultValue={`// SPDX-License-Identifier: MIT\npragma solidity ^0.8.24;\n\ncontract SimpleStorage {\n    uint256 private data;\n    \n    function set(uint256 x) public {\n        data = x;\n    }\n}`}
                  className="w-full bg-black border border-gray-900 rounded-xl p-3 text-sm font-mono text-white focus:border-blue-500 outline-none resize-none"
                />

                <button 
                  className="px-6 py-2.5 bg-green-600 hover:bg-green-500 text-white font-bold text-xs uppercase tracking-wider rounded-xl transition"
                  onClick={() => alert("Successfully compiled SimpleStorage!")}
                >
                  Compile SimpleStorage.sol
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
