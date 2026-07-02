"use client";

import React, { useState } from "react";
import Link from "next/link";
import { FileText, Code, Shield, HelpCircle, Layers, ArrowRight, Check } from "lucide-react";

export default function DocsPage() {
  const [activeTab, setActiveTab] = useState<"etherscan" | "webhooks" | "grpc">("etherscan");

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white">API Docs</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-950 pb-4">
        <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
          <FileText className="h-8 w-8 text-blue-500" />
          Developer API Documentation
        </h1>
        <p className="text-gray-400 mt-1">Guide to public Etherscan-compatible REST, gRPC gateway and Webhook notification systems</p>
      </div>

      {/* Navigation tabs */}
      <div className="flex border-b border-gray-900 space-x-4">
        <button
          onClick={() => setActiveTab("etherscan")}
          className={`pb-3 text-sm font-semibold border-b-2 transition ${activeTab === "etherscan" ? "border-blue-500 text-white" : "border-transparent text-gray-400 hover:text-white"}`}
        >
          Etherscan REST API
        </button>
        <button
          onClick={() => setActiveTab("webhooks")}
          className={`pb-3 text-sm font-semibold border-b-2 transition ${activeTab === "webhooks" ? "border-blue-500 text-white" : "border-transparent text-gray-400 hover:text-white"}`}
        >
          HMAC-SHA256 Webhooks
        </button>
        <button
          onClick={() => setActiveTab("grpc")}
          className={`pb-3 text-sm font-semibold border-b-2 transition ${activeTab === "grpc" ? "border-blue-500 text-white" : "border-transparent text-gray-400 hover:text-white"}`}
        >
          gRPC REST Gateway
        </button>
      </div>

      {/* Tab Contents */}
      {activeTab === "etherscan" && (
        <div className="space-y-6">
          <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 space-y-4">
            <h3 className="text-lg font-bold text-white">Overview</h3>
            <p className="text-sm text-gray-400 leading-relaxed">
              For complete Ethereum compatibility, Sovereign L1 hosts an Etherscan-compatible API endpoint at <code className="bg-gray-900 px-1.5 py-0.5 rounded text-blue-400 font-mono">/api</code>.
              This allows existing Etherscan client SDKs, scripts, and analytical systems to query Sovereign balance, block, and transactions out-of-the-box.
            </p>
          </div>

          <div className="space-y-4">
            <h4 className="text-md font-bold text-white">Supported Modules & Actions</h4>
            
            {/* Account Balance */}
            <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 space-y-3">
              <div className="flex justify-between items-center">
                <span className="text-sm font-semibold text-white font-mono">GET /api?module=account&action=balance&address=&#123;address&#125;</span>
                <span className="text-xs px-2 py-0.5 bg-blue-950 text-blue-400 rounded font-semibold uppercase">Account</span>
              </div>
              <p className="text-xs text-gray-400">Fetch the native token balance of a Bech32 or Hex address.</p>
              <pre className="bg-black/60 p-4 rounded-lg text-xs text-green-400 font-mono overflow-x-auto border border-gray-900">
{`{
  "status": "1",
  "message": "OK",
  "result": "1000000000000000000"
}`}
              </pre>
            </div>

            {/* Account Tx List */}
            <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 space-y-3">
              <div className="flex justify-between items-center">
                <span className="text-sm font-semibold text-white font-mono">GET /api?module=account&action=txlist&address=&#123;address&#125;</span>
                <span className="text-xs px-2 py-0.5 bg-blue-950 text-blue-400 rounded font-semibold uppercase">Account</span>
              </div>
              <p className="text-xs text-gray-400">List transactions associated with the given address.</p>
              <pre className="bg-black/60 p-4 rounded-lg text-xs text-green-400 font-mono overflow-x-auto border border-gray-900">
{`{
  "status": "1",
  "message": "OK",
  "result": [
    {
      "blockNumber": "100",
      "timeStamp": "1719233689",
      "hash": "0x7c28f9d6ae1234c...",
      "from": "sovereign1address0",
      "to": "0xcontractaddress",
      "value": "1000000000000000000",
      "gas": "21000",
      "gasUsed": "21000",
      "txreceipt_status": "1",
      "isError": "0"
    }
  ]
}`}
              </pre>
            </div>

            {/* Stats Supply */}
            <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 space-y-3">
              <div className="flex justify-between items-center">
                <span className="text-sm font-semibold text-white font-mono">GET /api?module=stats&action=ethsupply</span>
                <span className="text-xs px-2 py-0.5 bg-blue-950 text-blue-400 rounded font-semibold uppercase">Stats</span>
              </div>
              <p className="text-xs text-gray-400">Get the total supply of the native token on the chain.</p>
              <pre className="bg-black/60 p-4 rounded-lg text-xs text-green-400 font-mono overflow-x-auto border border-gray-900">
{`{
  "status": "1",
  "message": "OK",
  "result": "2500000000000000000000000"
}`}
              </pre>
            </div>
          </div>
        </div>
      )}

      {activeTab === "webhooks" && (
        <div className="space-y-6">
          <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 space-y-4">
            <h3 className="text-lg font-bold text-white flex items-center gap-2">
              <Shield className="h-5 w-5 text-indigo-400" />
              Secure HMAC-SHA256 Webhook Callbacks
            </h3>
            <p className="text-sm text-gray-400 leading-relaxed">
              Sovereign L1 Explorer supports active real-time webhook subscriptions. When a transaction involving your registered address is committed, the Indexer sends a POST callback containing the payload.
            </p>
            <p className="text-sm text-gray-400 leading-relaxed">
              To verify that the request originated from the Sovereign Explorer Indexer, the body is signed using <code className="bg-gray-900 px-1.5 py-0.5 rounded text-indigo-400 font-mono">HMAC-SHA256</code> with the webhook secret.
              The signature is included in the header as:
            </p>
            <div className="bg-black p-3 rounded-lg border border-gray-900 text-xs font-mono text-white">
              X-Sovereign-Signature: &lt;hex-encoded-signature&gt;
            </div>
          </div>

          <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 space-y-4">
            <h4 className="text-md font-bold text-white">Webhook Payload Schema</h4>
            <pre className="bg-black/60 p-4 rounded-lg text-xs text-green-400 font-mono overflow-x-auto border border-gray-900">
{`{
  "event": "tx_activity",
  "address": "sovereign1address0",
  "height": 120532,
  "timestamp": "2026-06-24T12:00:00Z"
}`}
            </pre>
          </div>

          <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 space-y-4">
            <h4 className="text-md font-bold text-white">Managing Webhooks via API</h4>
            <ul className="text-sm text-gray-400 space-y-2">
              <li className="flex items-center gap-2">
                <ArrowRight className="h-4 w-4 text-indigo-400" />
                <span className="font-mono text-white">POST /api/rest/v1/explorer/webhooks</span> — Register new URL
              </li>
              <li className="flex items-center gap-2">
                <ArrowRight className="h-4 w-4 text-indigo-400" />
                <span className="font-mono text-white">GET /api/rest/v1/explorer/webhooks</span> — List active webhooks
              </li>
              <li className="flex items-center gap-2">
                <ArrowRight className="h-4 w-4 text-indigo-400" />
                <span className="font-mono text-white">DELETE /api/rest/v1/explorer/webhooks/&#123;id&#125;</span> — Decommission webhook
              </li>
            </ul>
          </div>
        </div>
      )}

      {activeTab === "grpc" && (
        <div className="space-y-6">
          <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 space-y-4">
            <h3 className="text-lg font-bold text-white">gRPC REST Gateway (OpenAPI)</h3>
            <p className="text-sm text-gray-400 leading-relaxed">
              The primary developer API is served via gRPC and exposed as JSON REST endpoints on port <code className="bg-gray-900 px-1.5 py-0.5 rounded text-blue-400 font-mono">8081</code>.
              These endpoints support fully typed payload structures mapping directly to internal Go protocol buffers.
            </p>
          </div>

          <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 space-y-4">
            <h4 className="text-md font-bold text-white">Core Endpoints</h4>
            <div className="space-y-3">
              <div className="flex justify-between items-center text-sm font-mono border-b border-gray-900 pb-2">
                <span className="text-emerald-400">GET /api/rest/v1/explorer/blocks</span>
                <span className="text-gray-500">List block headers</span>
              </div>
              <div className="flex justify-between items-center text-sm font-mono border-b border-gray-900 pb-2">
                <span className="text-emerald-400">GET /api/rest/v1/explorer/txs</span>
                <span className="text-gray-500">List transactions</span>
              </div>
              <div className="flex justify-between items-center text-sm font-mono border-b border-gray-900 pb-2">
                <span className="text-emerald-400">GET /api/rest/v1/explorer/search?query=&#123;q&#125;</span>
                <span className="text-gray-500">Global trigram lookup</span>
              </div>
              <div className="flex justify-between items-center text-sm font-mono border-b border-gray-900 pb-2">
                <span className="text-emerald-400">GET /api/rest/v1/explorer/status</span>
                <span className="text-gray-500">Service health lag info</span>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
