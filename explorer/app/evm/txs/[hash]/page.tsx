"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { ArrowLeft, Clock, ShieldCheck, CheckCircle2, ChevronRight, FileCode2, Terminal, AlertOctagon, ArrowLeftRight } from "lucide-react";

interface EvmTx {
  hash: string;
  height: number;
  time: string;
  from: string;
  to: string;
  value: string;
  gasUsed: string;
  gasLimit: string;
  gasPrice: string;
  status: "success" | "failed";
  input: string;
  revertReason?: string;
}

interface DecodedInput {
  method: string;
  params: { name: string; type: string; value: string }[];
}

interface CallTrace {
  type: "CALL" | "DELEGATECALL" | "STATICCALL" | "CREATE";
  from: string;
  to: string;
  value: string;
  gas: number;
  depth: number;
}

interface TokenTransfer {
  tokenAddress: string;
  tokenSymbol: string;
  from: string;
  to: string;
  amount: string;
}

export default function EvmTxDetailPage() {
  const params = useParams();
  const hash = params?.hash ? String(params.hash) : "";

  const [tx, setTx] = useState<EvmTx | null>(null);
  const [decodedInput, setDecodedInput] = useState<DecodedInput | null>(null);
  const [traces, setTraces] = useState<CallTrace[]>([]);
  const [transfers, setTransfers] = useState<TokenTransfer[]>([]);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    if (!hash) return;
    const fetchTx = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/evm/txs/${hash}`);
        if (resp.ok) {
          const data = await resp.json();
          setTx({
            hash: data.hash || hash,
            height: Number(data.height || 0),
            time: data.time || new Date().toISOString(),
            from: data.sender || "0x0000000000000000000000000000000000000000",
            to: data.to || "0xcontractaddress",
            value: data.value || "0.00",
            gasUsed: data.gasUsed || "21,000",
            gasLimit: data.gasLimit || "100,000",
            gasPrice: data.gasPrice || "10 Gwei",
            status: data.status === 0 ? "success" : "failed",
            input: data.memo || "0x",
            revertReason: data.revertReason
          });
          if (data.decodedInput) setDecodedInput(data.decodedInput);
          if (data.traces) setTraces(data.traces);
          if (data.transfers) setTransfers(data.transfers);
        } else {
          throw new Error("Transaction not found");
        }
      } catch (err) {
        console.warn("Using simulated EVM tx details", err);
        setTx({
          hash: hash,
          height: 120530,
          time: new Date().toISOString(),
          from: "0x3f5c9e2b1d7a8d9e8a7b6c5d4e3f281f449219d54e47fd8ad83861b464815d9d",
          to: "0x25091a8d7a8b6c5d4e3f281f449219d54e47fd8a",
          value: "1.50",
          gasUsed: "84,320",
          gasLimit: "150,000",
          gasPrice: "18 Gwei",
          status: hash.endsWith("ff") ? "failed" : "success",
          input: "0xa9059cbb0000000000000000000000001234567890abcdef1234567890abcdef123456780000000000000000000000000000000000000000000000000de0b6b3a7640000",
          revertReason: hash.endsWith("ff") ? "ERC20: transfer amount exceeds balance" : undefined
        });

        // Set simulated decoded inputs
        setDecodedInput({
          method: "transfer(address to, uint256 value)",
          params: [
            { name: "to", type: "address", value: "0x1234567890abcdef1234567890abcdef12345678" },
            { name: "value", type: "uint256", value: "1,000,000,000,000,000,000 (1.0 SLT)" }
          ]
        });

        // Set simulated traces
        setTraces([
          { type: "CALL", from: "0x3f5c9e2b1d7a8d9e8a7b6c5d4e3f281f449219d54e47fd8ad83861b464815d9d", to: "0x25091a8d7a8b6c5d4e3f281f449219d54e47fd8a", value: "1.50 SLT", gas: 84320, depth: 0 },
          { type: "DELEGATECALL", from: "0x25091a8d7a8b6c5d4e3f281f449219d54e47fd8a", to: "0x892a10be892a10be892a10be892a10be892a10be8", value: "0 SLT", gas: 72150, depth: 1 }
        ]);

        // Set simulated transfers
        setTransfers([
          { tokenAddress: "0x25091a8d7a8b6c5d4e3f281f449219d54e47fd8a", tokenSymbol: "sUSDT", from: "0x3f5c9e2b1d7a8d9e", to: "0x1234567890abcdef", amount: "500.00" }
        ]);
      } finally {
        setLoading(false);
      }
    };
    fetchTx();
  }, [hash]);

  if (loading) {
    return (
      <div className="p-6 max-w-6xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading transaction details...</div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/evm" className="hover:text-white transition">EVM</Link>
        <span>/</span>
        <Link href="/evm/txs" className="hover:text-white transition">Transactions</Link>
        <span>/</span>
        <span className="text-gray-300 font-mono text-xs">{hash.slice(0, 10)}...</span>
      </nav>

      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between border-b border-gray-800 pb-6 gap-4">
        <div className="flex items-center gap-3">
          <Link href="/evm/txs" className="p-2 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-gray-400 hover:text-white transition">
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <h1 className="text-3xl font-extrabold tracking-tight text-white font-mono text-sm md:text-xl break-all">
            {tx?.hash}
          </h1>
        </div>
        <span className={`px-2.5 py-1 text-xs rounded font-semibold uppercase ${
          tx?.status === "success" ? "bg-green-950 text-green-400 border border-green-900" : "bg-red-950 text-red-400 border border-red-900"
        }`}>
          {tx?.status}
        </span>
      </div>

      {/* Revert Reason Banner */}
      {tx?.status === "failed" && tx.revertReason && (
        <div className="bg-red-950/20 border border-red-900/50 p-4 rounded-xl flex items-start gap-3 text-red-400">
          <AlertOctagon className="h-5 w-5 shrink-0 mt-0.5" />
          <div>
            <span className="font-bold block text-sm">Execution Reverted</span>
            <p className="text-xs text-gray-300 font-mono mt-1">{tx.revertReason}</p>
          </div>
        </div>
      )}

      {/* Details Grid */}
      <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-6 shadow-lg">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 text-sm">
          <div className="space-y-4">
            <div>
              <div className="text-gray-500 text-xs uppercase font-bold">From</div>
              <div className="font-mono text-xs text-gray-200 mt-1 break-all bg-gray-900/50 border border-gray-850 p-2 rounded-lg select-all">
                {tx?.from}
              </div>
            </div>
            <div>
              <div className="text-gray-500 text-xs uppercase font-bold">To / Contract</div>
              <div className="font-mono text-xs text-gray-200 mt-1 break-all bg-gray-900/50 border border-gray-850 p-2 rounded-lg select-all">
                {tx?.to}
              </div>
            </div>
            <div>
              <div className="text-gray-500 text-xs uppercase font-bold">Value</div>
              <div className="font-mono text-sm text-white mt-1">{tx?.value} SLT</div>
            </div>
          </div>
          <div className="space-y-4">
            <div>
              <div className="text-gray-500 text-xs uppercase font-bold">Gas Limit</div>
              <div className="font-mono text-sm text-gray-200 mt-1">{tx?.gasLimit}</div>
            </div>
            <div>
              <div className="text-gray-500 text-xs uppercase font-bold">Gas Used</div>
              <div className="font-mono text-sm text-gray-200 mt-1">{tx?.gasUsed}</div>
            </div>
            <div>
              <div className="text-gray-500 text-xs uppercase font-bold">Gas Price</div>
              <div className="font-mono text-sm text-gray-200 mt-1">{tx?.gasPrice}</div>
            </div>
          </div>
        </div>
      </div>

      {/* ABI Decoded Inputs */}
      {decodedInput && (
        <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-lg space-y-4">
          <h3 className="text-lg font-bold text-white flex items-center gap-2 border-b border-gray-900 pb-3">
            <FileCode2 className="text-blue-500 h-5 w-5" /> Decoded Contract Call ABI
          </h3>
          <div className="space-y-3">
            <div>
              <span className="text-xs text-gray-500 font-bold uppercase">Method called</span>
              <div className="font-mono text-sm text-green-400 font-semibold mt-1">{decodedInput.method}</div>
            </div>
            <div className="space-y-2">
              <span className="text-xs text-gray-500 font-bold uppercase block">Parameters</span>
              <div className="grid grid-cols-1 gap-2">
                {decodedInput.params.map((p, idx) => (
                  <div key={idx} className="bg-gray-900/50 border border-gray-850 p-3 rounded-lg text-xs flex justify-between font-mono">
                    <span className="text-gray-400 font-semibold">{p.name} ({p.type})</span>
                    <span className="text-white select-all break-all text-right max-w-md">{p.value}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Token Transfers Log */}
      {transfers.length > 0 && (
        <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-lg space-y-4">
          <h3 className="text-lg font-bold text-white flex items-center gap-2">
            <ArrowLeftRight className="text-indigo-500 h-5 w-5" /> Token Transfers (Log Events)
          </h3>
          <div className="space-y-2">
            {transfers.map((t, index) => (
              <div key={index} className="bg-gray-900/50 border border-gray-850 p-4 rounded-xl text-xs flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="font-mono text-gray-400">From</span>
                  <span className="font-mono font-bold text-white">{t.from.slice(0, 8)}...</span>
                  <span className="font-mono text-gray-400">To</span>
                  <span className="font-mono font-bold text-white">{t.to.slice(0, 8)}...</span>
                </div>
                <div className="text-right font-mono">
                  <span className="font-extrabold text-white">{t.amount}</span>{" "}
                  <span className="text-blue-400 font-semibold">{t.tokenSymbol}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Call Traces Tree */}
      {traces.length > 0 && (
        <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-lg space-y-4">
          <h3 className="text-lg font-bold text-white flex items-center gap-2 border-b border-gray-900 pb-3">
            <Terminal className="text-purple-500 h-5 w-5" /> Internal Transactions Call Tree
          </h3>
          <div className="space-y-3 font-mono text-xs">
            {traces.map((tr, idx) => (
              <div 
                key={idx} 
                className="border-l-2 border-gray-800 pl-4 py-2 space-y-1"
                style={{ marginLeft: `${tr.depth * 16}px` }}
              >
                <div className="flex items-center justify-between flex-wrap gap-2">
                  <span className={`px-2 py-0.5 rounded text-[10px] font-extrabold border ${
                    tr.type === "CALL" ? "bg-blue-950 border-blue-900 text-blue-400" :
                    tr.type === "DELEGATECALL" ? "bg-purple-950 border-purple-900 text-purple-400" :
                    "bg-gray-900 border-gray-800 text-gray-400"
                  }`}>
                    {tr.type}
                  </span>
                  <span className="text-gray-500 font-semibold">{tr.gas} gas</span>
                </div>
                <div className="text-gray-300">
                  {tr.from.slice(0, 8)}... &rarr; {tr.to.slice(0, 8)}...
                </div>
                {tr.value !== "0 SLT" && (
                  <div className="text-green-400 font-semibold">Value: {tr.value}</div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
