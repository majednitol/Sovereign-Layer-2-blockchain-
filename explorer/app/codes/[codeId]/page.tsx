"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { FileCode, ShieldCheck, Cpu, Code2, Calendar, ExternalLink, GitBranch, Check, AlertCircle } from "lucide-react";

interface CodeDetail {
  codeId: number;
  uploader: string;
  height: number;
  checksum: string;
  instantiationCount: number;
  txHash: string;
}

interface VerifiedInfo {
  verified: boolean;
  checksum: string;
  instantiateMsg: any;
  executeMsg: any;
  queryMsg: any;
  gitRepo: string;
  gitCommit: string;
  optimizerVersion: string;
}

export default function CodeDetailPage() {
  const params = useParams();
  const codeId = params?.codeId ? Number(params.codeId) : 1;
  const [c, setC] = useState<CodeDetail | null>(null);
  const [verified, setVerified] = useState<VerifiedInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<string>("overview");

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchCode = async () => {
      try {
        // Fetch basic code info from gRPC-gateway
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/codes/${codeId}`);
        if (resp.ok) {
          const data = await resp.json();
          setC({
            codeId: Number(data.codeId || codeId),
            uploader: data.uploader || "",
            height: Number(data.height || 0),
            checksum: data.checksum || "",
            instantiationCount: Number(data.instantiationCount || 0),
            txHash: data.txHash || "",
          });
        }
      } catch (err) {
        console.warn("Failed to load code details from gRPC", err);
      }

      // Fetch verification status from the custom REST endpoint
      try {
        const verifyResp = await fetch(`${API_BASE}/api/rest/v1/explorer/cosmwasm/codes/${codeId}`);
        if (verifyResp.ok) {
          const verifyData = await verifyResp.json();
          setVerified(verifyData);
          // Use checksum from verified data if available and base info missing
          if (verifyData.checksum && (!c || !c.checksum)) {
            setC(prev => prev ? { ...prev, checksum: verifyData.checksum } : {
              codeId,
              uploader: "",
              height: 0,
              checksum: verifyData.checksum,
              instantiationCount: 0,
              txHash: "",
            });
          }
        }
      } catch (err) {
        console.warn("Failed to load verification status", err);
      }

      setLoading(false);
    };
    fetchCode();
  }, [codeId]);

  if (loading) {
    return (
      <div className="p-6 max-w-6xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading code details...</div>
      </div>
    );
  }

  const isVerified = verified?.verified === true;
  const tabs = ["overview"];
  if (isVerified) {
    tabs.push("instantiate", "execute", "query");
  }

  const renderSchema = (schema: any, title: string) => {
    if (!schema || Object.keys(schema).length === 0) {
      return <div className="text-gray-500 text-sm italic">No schema available</div>;
    }

    const renderMethods = (s: any) => {
      const methods: { name: string; desc: string; fields: any }[] = [];
      if (s.oneOf) {
        for (const variant of s.oneOf) {
          if (variant.properties) {
            const key = Object.keys(variant.properties)[0];
            const inner = variant.properties[key];
            methods.push({
              name: key,
              desc: variant.description || "",
              fields: inner?.properties || {},
            });
          }
        }
      } else if (s.properties) {
        methods.push({
          name: s.title || "Message",
          desc: s.description || "",
          fields: s.properties,
        });
      }
      return methods;
    };

    const methods = renderMethods(schema);

    return (
      <div className="space-y-3">
        <h3 className="text-lg font-bold text-white">{title}</h3>
        {schema.description && (
          <p className="text-sm text-gray-400">{schema.description}</p>
        )}
        <div className="space-y-2">
          {methods.map((m, i) => (
            <div key={i} className="bg-gray-950 border border-gray-850 rounded-lg p-4">
              <div className="flex items-center gap-2 mb-1">
                <code className="text-blue-400 font-mono text-sm font-semibold">{m.name}</code>
                {m.desc && <span className="text-gray-500 text-xs">— {m.desc}</span>}
              </div>
              {Object.keys(m.fields).length > 0 ? (
                <div className="mt-2 space-y-1">
                  {Object.entries(m.fields).map(([fname, fval]: [string, any]) => (
                    <div key={fname} className="flex items-center gap-2 text-xs font-mono">
                      <span className="text-gray-300">{fname}</span>
                      <span className="text-gray-600">:</span>
                      <span className="text-emerald-400">{fval?.type || "any"}</span>
                      {fval?.description && (
                        <span className="text-gray-600 font-sans">({fval.description})</span>
                      )}
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-xs text-gray-600 mt-1 font-mono">{"{ }"} — no arguments</div>
              )}
            </div>
          ))}
        </div>
        <details className="mt-2">
          <summary className="text-xs text-gray-500 cursor-pointer hover:text-gray-300 transition">
            View Raw JSON Schema
          </summary>
          <pre className="mt-2 bg-gray-950 border border-gray-850 rounded-lg p-3 text-xs text-gray-300 overflow-x-auto max-h-[300px]">
            {JSON.stringify(schema, null, 2)}
          </pre>
        </details>
      </div>
    );
  };

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/codes" className="hover:text-white transition">CosmWasm Codes</Link>
        <span>/</span>
        <span className="text-gray-300">Code #{codeId}</span>
      </nav>

      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between border-b border-gray-800 pb-6 gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-white flex items-center gap-3">
            <FileCode className="w-8 h-8 text-blue-500" />
            WASM Code ID #{c?.codeId || codeId}
            {isVerified && (
              <span className="inline-flex items-center gap-1 px-3 py-1 bg-emerald-900/40 border border-emerald-700/50 rounded-full text-emerald-400 text-sm font-semibold">
                <ShieldCheck className="w-4 h-4" />
                Verified
              </span>
            )}
          </h1>
          {c?.uploader && (
            <p className="text-gray-400 mt-2">
              Creator: <span className="font-mono text-gray-200">{c.uploader}</span>
            </p>
          )}
        </div>
        {!isVerified && (
          <Link
            href="/verify"
            className="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white rounded-lg font-medium transition text-sm inline-flex items-center gap-2"
          >
            <ShieldCheck className="w-4 h-4" /> Verify This Code
          </Link>
        )}
      </div>

      {/* Stats Panel */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-6">
        {c?.height ? (
          <div className="bg-gray-900 border border-gray-850 p-5 rounded-xl space-y-2">
            <div className="text-xs text-gray-400 uppercase tracking-wider font-semibold">Store Height</div>
            <div className="text-2xl font-bold text-white font-mono">#{c.height}</div>
          </div>
        ) : null}
        <div className="bg-gray-900 border border-gray-850 p-5 rounded-xl space-y-2">
          <div className="text-xs text-gray-400 uppercase tracking-wider font-semibold">Active Instantiations</div>
          <div className="text-2xl font-bold text-white">{c?.instantiationCount || 0} child contracts</div>
        </div>
        {isVerified && verified?.optimizerVersion && (
          <div className="bg-gray-900 border border-gray-850 p-5 rounded-xl space-y-2">
            <div className="text-xs text-gray-400 uppercase tracking-wider font-semibold">Optimizer</div>
            <div className="text-sm font-mono text-gray-300 mt-1.5">{verified.optimizerVersion}</div>
          </div>
        )}
      </div>

      {/* Tabs */}
      <div className="flex border-b border-gray-850 gap-1">
        {tabs.map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2.5 text-sm font-medium border-b-2 transition capitalize ${
              activeTab === tab
                ? "border-blue-500 text-blue-400"
                : "border-transparent text-gray-500 hover:text-gray-300"
            }`}
          >
            {tab === "instantiate" ? "Instantiate Schema" :
             tab === "execute" ? "Execute Schema" :
             tab === "query" ? "Query Schema" : "Overview"}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      {activeTab === "overview" && (
        <div className="space-y-6">
          {/* Checksum */}
          <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 space-y-4">
            <h2 className="text-xl font-bold text-white flex items-center gap-2">
              <Cpu className="w-5 h-5 text-blue-500" />
              SHA-256 Checksum
            </h2>
            <div className="bg-gray-950 p-4 border border-gray-850 rounded-lg">
              <div className="text-xs text-gray-500 uppercase">On-chain WASM SHA-256 Checksum</div>
              <div className="font-mono text-sm text-gray-200 mt-1 break-all">
                {c?.checksum || verified?.checksum || "N/A"}
              </div>
            </div>
          </div>

          {/* Verification Details */}
          {isVerified && (
            <div className="bg-gray-900 border border-emerald-900/30 rounded-xl p-6 space-y-4">
              <h2 className="text-xl font-bold text-white flex items-center gap-2">
                <ShieldCheck className="w-5 h-5 text-emerald-400" />
                Verification Details
              </h2>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {verified?.gitRepo && (
                  <div className="bg-gray-950 p-4 border border-gray-850 rounded-lg">
                    <div className="text-xs text-gray-500 uppercase flex items-center gap-1">
                      <GitBranch className="w-3 h-3" /> Git Repository
                    </div>
                    <a
                      href={verified.gitRepo}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="font-mono text-sm text-blue-400 hover:text-blue-300 mt-1 break-all flex items-center gap-1"
                    >
                      {verified.gitRepo}
                      <ExternalLink className="w-3 h-3 flex-shrink-0" />
                    </a>
                  </div>
                )}
                {verified?.gitCommit && (
                  <div className="bg-gray-950 p-4 border border-gray-850 rounded-lg">
                    <div className="text-xs text-gray-500 uppercase">Git Commit</div>
                    <div className="font-mono text-sm text-gray-200 mt-1">{verified.gitCommit}</div>
                  </div>
                )}
                {verified?.optimizerVersion && (
                  <div className="bg-gray-950 p-4 border border-gray-850 rounded-lg">
                    <div className="text-xs text-gray-500 uppercase">Optimizer Version</div>
                    <div className="font-mono text-sm text-gray-200 mt-1">{verified.optimizerVersion}</div>
                  </div>
                )}
              </div>

              {/* Schema Summary */}
              <div className="border-t border-gray-850 pt-4">
                <div className="text-sm text-gray-400 mb-3">Available Schemas:</div>
                <div className="flex flex-wrap gap-2">
                  {verified?.instantiateMsg && Object.keys(verified.instantiateMsg).length > 0 && (
                    <button
                      onClick={() => setActiveTab("instantiate")}
                      className="px-3 py-1.5 bg-blue-900/30 border border-blue-800/40 rounded-lg text-blue-400 text-xs font-medium hover:bg-blue-900/50 transition"
                    >
                      Instantiate Schema →
                    </button>
                  )}
                  {verified?.executeMsg && Object.keys(verified.executeMsg).length > 0 && (
                    <button
                      onClick={() => setActiveTab("execute")}
                      className="px-3 py-1.5 bg-purple-900/30 border border-purple-800/40 rounded-lg text-purple-400 text-xs font-medium hover:bg-purple-900/50 transition"
                    >
                      Execute Schema →
                    </button>
                  )}
                  {verified?.queryMsg && Object.keys(verified.queryMsg).length > 0 && (
                    <button
                      onClick={() => setActiveTab("query")}
                      className="px-3 py-1.5 bg-emerald-900/30 border border-emerald-800/40 rounded-lg text-emerald-400 text-xs font-medium hover:bg-emerald-900/50 transition"
                    >
                      Query Schema →
                    </button>
                  )}
                </div>
              </div>
            </div>
          )}

          {/* Not Verified Notice */}
          {!isVerified && (
            <div className="bg-yellow-950/30 border border-yellow-800/30 rounded-xl p-6 flex items-start gap-3">
              <AlertCircle className="w-5 h-5 text-yellow-500 mt-0.5 flex-shrink-0" />
              <div>
                <div className="text-yellow-400 font-semibold">Not Yet Verified</div>
                <p className="text-sm text-gray-400 mt-1">
                  This code has not been verified yet. Submit a verification request with your compiled .wasm binary
                  and JSON schemas to enable interactive contract exploration.
                </p>
                <Link
                  href="/verify"
                  className="inline-flex items-center gap-1 mt-3 text-sm text-blue-400 hover:text-blue-300 transition"
                >
                  Go to Verification Hub <ExternalLink className="w-3 h-3" />
                </Link>
              </div>
            </div>
          )}
        </div>
      )}

      {activeTab === "instantiate" && isVerified && (
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
          {renderSchema(verified?.instantiateMsg, "Instantiate Message Schema")}
        </div>
      )}

      {activeTab === "execute" && isVerified && (
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
          {renderSchema(verified?.executeMsg, "Execute Message Schema")}
        </div>
      )}

      {activeTab === "query" && isVerified && (
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
          {renderSchema(verified?.queryMsg, "Query Message Schema")}
        </div>
      )}
    </div>
  );
}
