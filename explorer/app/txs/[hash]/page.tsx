"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { ArrowLeft, Clock, Activity, ShieldAlert, FileText, CheckCircle2, XCircle } from "lucide-react";

interface TxDetail {
  hash: string;
  height: number;
  time: string;
  type: string;
  msgTypes: string[];
  decoded: any; // Parsed from JSON string
  fee: number;
  gasUsed: number;
  status: number;
}

interface Props {
  params: Promise<{ hash: string }>;
}

function parseMemoFromTxBytes(base64Tx: string): string {
  try {
    const binary = atob(base64Tx);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) {
      bytes[i] = binary.charCodeAt(i);
    }

    let pos = 0;
    const readVarint = () => {
      let value = 0;
      let shift = 0;
      while (pos < bytes.length) {
        const b = bytes[pos++];
        value |= (b & 0x7f) << shift;
        if ((b & 0x80) === 0) break;
        shift += 7;
      }
      return value;
    };

    if (pos < bytes.length) {
      const tag = readVarint();
      const fieldNum = tag >> 3;
      const wireType = tag & 0x07;
      if (fieldNum === 1 && wireType === 2) {
        const bodyLen = readVarint();
        const bodyEnd = pos + bodyLen;
        
        while (pos < bodyEnd && pos < bytes.length) {
          const bodyTag = readVarint();
          const bodyFieldNum = bodyTag >> 3;
          const bodyWireType = bodyTag & 0x07;
          
          if (bodyFieldNum === 1 && bodyWireType === 2) {
            const msgLen = readVarint();
            pos += msgLen;
          } else if (bodyFieldNum === 2 && bodyWireType === 2) {
            const memoLen = readVarint();
            const memoBytes = bytes.slice(pos, pos + memoLen);
            pos += memoLen;
            return new TextDecoder().decode(memoBytes);
          } else {
            if (bodyWireType === 0) readVarint();
            else if (bodyWireType === 2) pos += readVarint();
            else if (bodyWireType === 1) pos += 8;
            else if (bodyWireType === 5) pos += 4;
            else break;
          }
        }
      }
    }
  } catch (e) {
    console.error("Failed to parse memo from protobuf", e);
  }
  return "";
}

export default function TxDetailPage({ params }: Props) {
  const { hash } = React.use(params);
  const [tx, setTx] = useState<TxDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [memo, setMemo] = useState<string>("");
  const [events, setEvents] = useState<{ type: string; attributes: { key: string; value: string }[] }[]>([]);

  const [retrying, setRetrying] = useState(false);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    let cancelled = false;
    const MAX_RETRIES = 10;
    const RETRY_DELAY_MS = 2000;

    const fetchTxDetail = async (attempt: number): Promise<boolean> => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/txs/${hash}`);
        if (!resp.ok) {
          // Transaction not indexed yet — eligible for retry
          return false;
        }
        const data = await resp.json();
        
        let decodedObj = {};
        if (data.decoded) {
          try {
            decodedObj = JSON.parse(data.decoded);
          } catch (e) {
            decodedObj = { raw: data.decoded };
          }
        }

        if (!cancelled) {
          setTx({
            hash: data.hash,
            height: Number(data.height),
            time: data.time,
            type: data.type,
            msgTypes: data.msgTypes || [],
            decoded: decodedObj,
            fee: Number(data.fee || 0),
            gasUsed: Number(data.gasUsed || 0),
            status: Number(data.status || 0),
          });

          // Fetch events and memo from CometBFT
          const COMET_RPC = process.env.NEXT_PUBLIC_RPC_URL || "http://localhost:26657";
          const cometResp = await fetch(`${COMET_RPC}/tx?hash=0x${hash}`);
          if (cometResp.ok) {
            const cometData = await cometResp.json();
            const result = cometData.result || {};
            
            if (result.tx) {
              const parsedMemo = parseMemoFromTxBytes(result.tx);
              setMemo(parsedMemo);
            }

            if (result.tx_result?.events) {
              const mappedEvents = result.tx_result.events.map((e: any) => ({
                type: e.type,
                attributes: (e.attributes || []).map((a: any) => {
                  let key = a.key || "";
                  let value = a.value || "";

                  return { key, value };
                }),
              }));
              setEvents(mappedEvents);
            }
          }
        }
        return true; // success
      } catch (err: any) {
        return false;
      }
    };

    const run = async () => {
      setLoading(true);
      setError(null);

      // First attempt
      const found = await fetchTxDetail(0);
      if (found || cancelled) {
        if (!cancelled) setLoading(false);
        return;
      }

      // Not found yet — start retrying with a "waiting" UI
      if (!cancelled) {
        setLoading(false);
        setRetrying(true);
      }

      for (let attempt = 1; attempt <= MAX_RETRIES; attempt++) {
        if (cancelled) return;
        await new Promise((resolve) => setTimeout(resolve, RETRY_DELAY_MS));
        if (cancelled) return;

        const found = await fetchTxDetail(attempt);
        if (found || cancelled) {
          if (!cancelled) setRetrying(false);
          return;
        }
      }

      // All retries exhausted
      if (!cancelled) {
        setRetrying(false);
        setError(`Transaction not found: ${hash}`);
      }
    };

    run();

    return () => {
      cancelled = true;
    };
  }, [hash, API_BASE]);

  if (loading) {
    return (
      <div className="flex justify-center items-center py-40">
        <Activity className="h-8 w-8 text-blue-500 animate-spin" />
      </div>
    );
  }

  if (retrying) {
    return (
      <div className="p-6 max-w-4xl mx-auto text-center space-y-4 py-32">
        <Clock className="h-16 w-16 text-yellow-500 mx-auto animate-pulse" />
        <h2 className="text-2xl font-bold text-white">Waiting for Confirmation</h2>
        <p className="text-gray-400">
          Transaction <span className="font-mono text-sm text-gray-300">{hash.slice(0, 16)}...</span> has been broadcast and is awaiting block confirmation.
        </p>
        <div className="flex justify-center pt-2">
          <Activity className="h-5 w-5 text-blue-500 animate-spin" />
          <span className="ml-2 text-gray-500 text-sm">Checking for confirmation...</span>
        </div>
      </div>
    );
  }

  if (error || !tx) {
    return (
      <div className="p-6 max-w-4xl mx-auto text-center space-y-4">
        <ShieldAlert className="h-16 w-16 text-red-500 mx-auto" />
        <h2 className="text-2xl font-bold text-white">Error Loading Transaction</h2>
        <p className="text-gray-400">{error || "Transaction not found"}</p>
        <Link href="/txs" className="inline-block px-4 py-2 bg-gray-900 border border-gray-800 rounded-lg text-white hover:bg-gray-800 transition">
          Back to Transactions
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
        <Link href="/txs" className="hover:text-white transition">Transactions</Link>
        <span>/</span>
        <span className="text-white truncate max-w-xs">{tx.hash.slice(0, 16)}...</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4 flex items-center space-x-4">
        <Link href="/txs" className="p-2 bg-gray-900 border border-gray-800 hover:bg-gray-800 rounded-lg text-gray-400 hover:text-white transition">
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-white flex items-center space-x-2">
            <span>Transaction Details</span>
          </h1>
          <p className="text-gray-400 mt-1 font-mono text-sm break-all">{tx.hash}</p>
        </div>
      </div>

      {/* Details Card */}
      <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 space-y-6 shadow-xl">
        <div className="flex justify-between items-center border-b border-gray-900 pb-4">
          <span className="text-sm font-bold text-white uppercase tracking-wider">Parameters</span>
          <span className={`inline-flex items-center space-x-1.5 px-3 py-1 rounded-full text-xs font-bold ${tx.status === 0 ? "bg-green-950 text-green-400 border border-green-900" : "bg-red-950 text-red-400 border border-red-900"} border`}>
            {tx.status === 0 ? (
              <>
                <CheckCircle2 className="h-3.5 w-3.5" />
                <span>Success</span>
              </>
            ) : (
              <>
                <XCircle className="h-3.5 w-3.5" />
                <span>Failed</span>
              </>
            )}
          </span>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 text-sm">
          <div className="space-y-4">
            <div className="grid grid-cols-3 gap-2">
              <span className="text-gray-500">Block Height</span>
              <span className="col-span-2 text-blue-500 font-bold">
                <Link href={`/blocks/${tx.height}`} className="hover:underline">
                  #{tx.height}
                </Link>
              </span>
            </div>

            <div className="grid grid-cols-3 gap-2">
              <span className="text-gray-500">Timestamp</span>
              <span className="col-span-2 text-white font-medium">
                {new Date(tx.time).toLocaleString()}
              </span>
            </div>

            <div className="grid grid-cols-3 gap-2">
              <span className="text-gray-500">Transaction Type</span>
              <span className="col-span-2">
                <span className="capitalize px-2 py-0.5 bg-gray-900 border border-gray-800 text-xs rounded text-gray-400 font-mono font-medium">
                  {tx.type}
                </span>
              </span>
            </div>

            <div className="grid grid-cols-3 gap-2">
              <span className="text-gray-500">Memo</span>
              <span className="col-span-2 text-white italic">
                {memo || <span className="text-gray-600">No memo</span>}
              </span>
            </div>
          </div>

          <div className="space-y-4">
            <div className="grid grid-cols-3 gap-2">
              <span className="text-gray-500">Transaction Fee</span>
              <span className="col-span-2 text-white font-mono font-semibold">
                {tx.fee.toLocaleString()} uSLT
              </span>
            </div>

            <div className="grid grid-cols-3 gap-2">
              <span className="text-gray-500">Gas Used</span>
              <span className="col-span-2 text-white font-mono">
                {tx.gasUsed.toLocaleString()}
              </span>
            </div>

            <div className="grid grid-cols-3 gap-2">
              <span className="text-gray-500">Message Schema</span>
              <span className="col-span-2 text-gray-400 font-mono text-xs truncate">
                {tx.msgTypes.join(", ") || "N/A"}
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Decoded Payload */}
      <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 space-y-4 shadow-xl">
        <h3 className="text-lg font-bold text-white border-b border-gray-900 pb-2">
          Decoded Payload
        </h3>

        {tx.decoded && Object.keys(tx.decoded).length > 0 ? (
          <div className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 bg-black/30 border border-gray-900 rounded-xl p-4">
              {Object.entries(tx.decoded).map(([key, val]) => (
                <div key={key} className="space-y-1">
                  <span className="text-xs font-bold text-gray-500 uppercase tracking-wider">{key}</span>
                  <div className="text-sm font-mono text-gray-300 break-all">
                    {typeof val === "object" ? JSON.stringify(val) : String(val)}
                  </div>
                </div>
              ))}
            </div>
            
            <details className="group">
              <summary className="text-xs text-gray-500 hover:text-white cursor-pointer transition select-none">
                View Raw JSON
              </summary>
              <pre className="mt-2 bg-black/50 border border-gray-900 rounded-xl p-4 overflow-x-auto text-xs font-mono text-green-400 leading-relaxed max-h-[300px]">
                {JSON.stringify(tx.decoded, null, 2)}
              </pre>
            </details>
          </div>
        ) : (
          <div className="py-6 text-center text-gray-500 text-sm">
            No message payload data parsed for this transaction.
          </div>
        )}
      </div>

      {/* Transaction Events */}
      <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 space-y-4 shadow-xl">
        <h3 className="text-lg font-bold text-white border-b border-gray-900 pb-2">
          Transaction Events ({events.length})
        </h3>

        <div className="space-y-4 divide-y divide-gray-900 max-h-[500px] overflow-y-auto pr-2">
          {events.length === 0 ? (
            <div className="py-6 text-center text-gray-500 text-sm">
              No events emitted by this transaction.
            </div>
          ) : (
            events.map((ev, idx) => (
              <div key={idx} className="pt-4 first:pt-0 space-y-2">
                <div className="flex items-center space-x-2">
                  <span className="text-xs font-bold px-2 py-0.5 rounded bg-blue-950 text-blue-400 border border-blue-900 font-mono">
                    {ev.type}
                  </span>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-2 pl-2">
                  {ev.attributes.map((attr, attrIdx) => (
                    <div key={attrIdx} className="text-xs font-mono flex items-start space-x-2 bg-gray-900/20 p-1.5 rounded border border-gray-900/30">
                      <span className="text-gray-500 font-medium shrink-0">{attr.key}:</span>
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
  );
}
