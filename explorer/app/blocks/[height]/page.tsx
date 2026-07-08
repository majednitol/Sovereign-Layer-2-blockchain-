"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { ArrowLeft, Clock, Server, FileText, Activity, ShieldAlert, Cpu } from "lucide-react";

interface BlockDetail {
  height: number;
  time: string;
  proposer: string;
  txCount: number;
  gasUsed: number;
  gasLimit: number;
  appHash: string;
}

interface Tx {
  hash: string;
  height: number;
  time: string;
  type: string;
  msgTypes: string[];
  status: number;
  fee: number;
}

import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/Tabs";

interface CommitSignature {
  validatorAddress: string;
  timestamp: string;
  signature: string;
  blockIdFlag: number;
}

interface Props {
  params: Promise<{ height: string }>;
}

export default function BlockDetailPage({ params }: Props) {
  const { height } = React.use(params);
  const [block, setBlock] = useState<BlockDetail | null>(null);
  const [txs, setTxs] = useState<Tx[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [beginEvents, setBeginEvents] = useState<{ type: string; attributes: { key: string; value: string }[] }[]>([]);
  const [endEvents, setEndEvents] = useState<{ type: string; attributes: { key: string; value: string }[] }[]>([]);
  const [signatures, setSignatures] = useState<CommitSignature[]>([]);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      setError(null);
      try {
        // Fetch block details
        const blockResp = await fetch(`${API_BASE}/api/rest/v1/explorer/blocks/${height}`);
        if (!blockResp.ok) {
          throw new Error(`Block not found at height ${height}`);
        }
        const blockData = await blockResp.json();
        setBlock({
          height: Number(blockData.height),
          time: blockData.time,
          proposer: blockData.proposer,
          txCount: Number(blockData.txCount || 0),
          gasUsed: Number(blockData.gasUsed || 0),
          gasLimit: Number(blockData.gasLimit || 0),
          appHash: blockData.appHash,
        });

        // Fetch transactions for this block height
        const txsResp = await fetch(`${API_BASE}/api/rest/v1/explorer/txs?height=${height}`);
        if (txsResp.ok) {
          const txsData = await txsResp.json();
          if (txsData.txs) {
            const mappedTxs = txsData.txs.map((t: any) => ({
              hash: t.hash,
              height: Number(t.height),
              time: t.time,
              type: t.type,
              msgTypes: t.msgTypes || [],
              status: Number(t.status || 0),
              fee: Number(t.fee || 0),
            }));
            setTxs(mappedTxs);
          }
        }

        // Fetch block results for ABCI events
        const COMET_RPC = process.env.NEXT_PUBLIC_RPC_URL || "http://localhost:26657";
        const resultsResp = await fetch(`${COMET_RPC}/block_results?height=${height}`);
        if (resultsResp.ok) {
          const resultsData = await resultsResp.json();
          const result = resultsData.result || {};
          
          const mapEvents = (events: any[]): { type: string; attributes: { key: string; value: string }[] }[] => {
            return (events || []).map((e: any) => ({
              type: e.type,
              attributes: (e.attributes || []).map((a: any) => {
                let key = a.key || "";
                let value = a.value || "";

                return { key, value };
              }),
            }));
          };

          setBeginEvents(mapEvents(result.begin_block_events));
          setEndEvents(mapEvents(result.end_block_events));
        }

        // Fetch block commit signatures
        const commitResp = await fetch(`${COMET_RPC}/commit?height=${height}`);
        if (commitResp.ok) {
          const commitData = await commitResp.json();
          const commitObj = commitData.result?.signed_header?.commit || {};
          const sigs = (commitObj.signatures || []).map((s: any) => ({
            validatorAddress: s.validator_address || "",
            timestamp: s.timestamp || "",
            signature: s.signature || "",
            blockIdFlag: s.block_id_flag || 0,
          }));
          setSignatures(sigs);
        }
      } catch (err: any) {
        setError(err.message || "Failed to load block data");
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [height, API_BASE]);

  if (loading) {
    return (
      <div className="flex justify-center items-center py-40">
        <Activity className="h-8 w-8 text-blue-500 animate-spin" />
      </div>
    );
  }

  if (error || !block) {
    return (
      <div className="p-6 max-w-4xl mx-auto text-center space-y-4">
        <ShieldAlert className="h-16 w-16 text-red-500 mx-auto" />
        <h2 className="text-2xl font-bold text-white">Error Loading Block</h2>
        <p className="text-gray-400">{error || "Block not found"}</p>
        <Link href="/blocks" className="inline-block px-4 py-2 bg-gray-900 border border-gray-800 rounded-lg text-white hover:bg-gray-800 transition">
          Back to Blocks
        </Link>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/blocks" className="hover:text-white transition">Blocks</Link>
        <span>/</span>
        <span className="text-white">#{block.height}</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4 flex items-center space-x-4">
        <Link href="/blocks" className="p-2 bg-gray-900 border border-gray-800 hover:bg-gray-800 rounded-lg text-gray-400 hover:text-white transition">
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-white">Block #{block.height}</h1>
          <p className="text-gray-400 mt-1">Block details and transactions</p>
        </div>
      </div>

      {/* Tabs Layout */}
      <Tabs defaultValue="overview" className="space-y-6">
        <TabsList>
          <TabsTrigger value="overview">Overview & Transactions</TabsTrigger>
          <TabsTrigger value="consensus">Consensus Info ({signatures.length})</TabsTrigger>
          <TabsTrigger value="events">ABCI Events ({beginEvents.length + endEvents.length})</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-6">
          {/* Details Grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 space-y-4 shadow-xl">
              <h3 className="text-lg font-bold text-white border-b border-gray-900 pb-2">Overview</h3>
              
              <div className="grid grid-cols-3 gap-2 text-sm">
                <div className="text-gray-500 flex items-center space-x-2">
                  <Clock className="h-4 w-4 text-gray-400" />
                  <span>Time</span>
                </div>
                <div className="col-span-2 text-white font-medium">
                  {new Date(block.time).toLocaleString()}
                </div>

                <div className="text-gray-500 flex items-center space-x-2">
                  <Server className="h-4 w-4 text-gray-400" />
                  <span>Proposer</span>
                </div>
                <div className="col-span-2 font-mono text-xs text-gray-400 truncate" title={block.proposer}>
                  {block.proposer}
                </div>

                <div className="text-gray-500 flex items-center space-x-2">
                  <Cpu className="h-4 w-4 text-gray-400" />
                  <span>App Hash</span>
                </div>
                <div className="col-span-2 font-mono text-xs text-gray-400 truncate" title={block.appHash}>
                  {block.appHash}
                </div>
              </div>
            </div>

            <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 space-y-4 shadow-xl">
              <h3 className="text-lg font-bold text-white border-b border-gray-900 pb-2">Resources</h3>
              
              <div className="grid grid-cols-3 gap-2 text-sm">
                <div className="text-gray-500 flex items-center space-x-2">
                  <FileText className="h-4 w-4 text-gray-400" />
                  <span>Transactions</span>
                </div>
                <div className="col-span-2 text-white font-medium">
                  {block.txCount}
                </div>

                <div className="text-gray-500">Gas Limit</div>
                <div className="col-span-2 text-white font-mono">
                  {block.gasLimit.toLocaleString()}
                </div>

                <div className="text-gray-500">Gas Used</div>
                <div className="col-span-2 text-white font-mono">
                  {block.gasUsed.toLocaleString()} ({((block.gasUsed / (block.gasLimit || 1)) * 100).toFixed(2)}%)
                </div>
              </div>
            </div>
          </div>

          {/* Transactions in Block */}
          <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 space-y-4 shadow-xl">
            <h3 className="text-lg font-bold text-white border-b border-gray-900 pb-2">
              Transactions in Block ({txs.length})
            </h3>

            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="bg-gray-900/50 text-gray-400 text-xs font-bold uppercase border-b border-gray-900">
                    <th className="py-3 px-4">Tx Hash</th>
                    <th className="py-3 px-4">Type</th>
                    <th className="py-3 px-4">Messages</th>
                    <th className="py-3 px-4 text-right">Fee</th>
                    <th className="py-3 px-4 text-right">Status</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-900/30 text-sm text-gray-300">
                  {txs.length === 0 ? (
                    <tr>
                      <td colSpan={5} className="py-6 text-center text-gray-500 font-mono">
                        No transactions indexed for this block.
                      </td>
                    </tr>
                  ) : (
                    txs.map((tx) => (
                      <tr key={tx.hash} className="hover:bg-gray-900/10 transition">
                        <td className="py-3 px-4 font-mono text-xs text-blue-500">
                          <Link href={`/txs/${tx.hash}`} className="hover:underline">
                            {tx.hash.slice(0, 24)}...
                          </Link>
                        </td>
                        <td className="py-3 px-4">
                          <span className="capitalize px-2 py-0.5 bg-gray-900 rounded border border-gray-800 text-xs text-gray-400">
                            {tx.type}
                          </span>
                        </td>
                        <td className="py-3 px-4 text-xs font-mono text-gray-400">
                          {tx.msgTypes[0] || "Msg"}
                        </td>
                        <td className="py-3 px-4 text-right font-mono text-xs text-gray-500">
                          {tx.fee} uSLT
                        </td>
                        <td className="py-3 px-4 text-right">
                          <span className={`inline-block px-2 py-0.5 rounded text-xs font-bold ${tx.status === 0 ? "bg-green-950 text-green-400 border border-green-900" : "bg-red-950 text-red-400 border border-red-900"} border`}>
                            {tx.status === 0 ? "Success" : "Failed"}
                          </span>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </TabsContent>

        <TabsContent value="consensus">
          {/* Commit Signatures */}
          <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 space-y-4 shadow-xl">
            <h3 className="text-lg font-bold text-white border-b border-gray-900 pb-2">
              CometBFT Commit Signatures
            </h3>
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="bg-gray-900/50 text-gray-400 text-xs font-bold uppercase border-b border-gray-900">
                    <th className="py-3 px-4">Validator Hex Address</th>
                    <th className="py-3 px-4">Timestamp</th>
                    <th className="py-3 px-4">Signature Value</th>
                    <th className="py-3 px-4 text-right">Status</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-900/30 text-sm text-gray-300">
                  {signatures.length === 0 ? (
                    <tr>
                      <td colSpan={4} className="py-6 text-center text-gray-500 font-mono">
                        No commit signatures found.
                      </td>
                    </tr>
                  ) : (
                    signatures.map((sig, idx) => (
                      <tr key={idx} className="hover:bg-gray-900/10 transition font-mono">
                        <td className="py-3 px-4 text-xs text-gray-400">
                          {sig.validatorAddress ? `0x${sig.validatorAddress.toLowerCase()}` : "Absent"}
                        </td>
                        <td className="py-3 px-4 text-xs text-gray-500">
                          {sig.timestamp ? new Date(sig.timestamp).toLocaleString() : "-"}
                        </td>
                        <td className="py-3 px-4 text-xs text-gray-600 truncate max-w-[200px]" title={sig.signature}>
                          {sig.signature || "nil"}
                        </td>
                        <td className="py-3 px-4 text-right">
                          <span className={`inline-block px-2 py-0.5 rounded text-xs font-bold ${sig.blockIdFlag === 2 ? "bg-green-950 text-green-400 border border-green-900" : "bg-amber-950 text-amber-400 border border-amber-900"} border`}>
                            {sig.blockIdFlag === 2 ? "Commit" : sig.blockIdFlag === 1 ? "Absent" : "Nil"}
                          </span>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </TabsContent>

        <TabsContent value="events" className="space-y-6">
          {/* ABCI Events Log */}
          <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 space-y-6 shadow-xl">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* Begin Block Events */}
              <div className="space-y-4">
                <h4 className="text-sm font-bold text-gray-400 uppercase tracking-wider">Begin Block Events ({beginEvents.length})</h4>
                <div className="space-y-3 max-h-[400px] overflow-y-auto pr-2 divide-y divide-gray-900">
                  {beginEvents.length === 0 ? (
                    <div className="text-xs text-gray-500 py-2">No BeginBlock events.</div>
                  ) : (
                    beginEvents.map((ev, idx) => (
                      <div key={idx} className="pt-3 first:pt-0 space-y-1">
                        <span className="text-xs font-bold text-blue-400 font-mono">{ev.type}</span>
                        <div className="grid grid-cols-1 gap-1 pl-3">
                          {ev.attributes.map((attr, attrIdx) => (
                            <div key={attrIdx} className="text-xs font-mono flex items-start space-x-2">
                              <span className="text-gray-500 font-medium">{attr.key}:</span>
                              <span className="text-gray-300 break-all">{attr.value}</span>
                            </div>
                          ))}
                        </div>
                      </div>
                    ))
                  )}
                </div>
              </div>

              {/* End Block Events */}
              <div className="space-y-4">
                <h4 className="text-sm font-bold text-gray-400 uppercase tracking-wider">End Block Events ({endEvents.length})</h4>
                <div className="space-y-3 max-h-[400px] overflow-y-auto pr-2 divide-y divide-gray-900">
                  {endEvents.length === 0 ? (
                    <div className="text-xs text-gray-500 py-2">No EndBlock events.</div>
                  ) : (
                    endEvents.map((ev, idx) => (
                      <div key={idx} className="pt-3 first:pt-0 space-y-1">
                        <span className="text-xs font-bold text-green-400 font-mono">{ev.type}</span>
                        <div className="grid grid-cols-1 gap-1 pl-3">
                          {ev.attributes.map((attr, attrIdx) => (
                            <div key={attrIdx} className="text-xs font-mono flex items-start space-x-2">
                              <span className="text-gray-500 font-medium">{attr.key}:</span>
                              <span className="text-gray-300 break-all">{attr.value}</span>
                            </div>
                          ))}
                        </div>
                      </div>
                    ))
                  )}
                </div>
              </div>
            </div>
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
