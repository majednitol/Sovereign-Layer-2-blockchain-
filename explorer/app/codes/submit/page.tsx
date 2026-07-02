"use client";

import React, { useState } from "react";
import Link from "next/link";
import { Upload, FileCode, CheckCircle2, AlertCircle } from "lucide-react";

export default function SubmitCodeSchemaPage() {
  const [codeId, setCodeId] = useState("");
  const [schemaText, setSchemaText] = useState("");
  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!codeId || !schemaText) {
      setError("Please fill in all fields.");
      return;
    }
    setLoading(true);
    setError(null);
    setSuccess(false);

    // Simulate submission
    setTimeout(() => {
      setLoading(false);
      setSuccess(true);
    }, 1000);
  };

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/codes" className="hover:text-white transition">CosmWasm Codes</Link>
        <span>/</span>
        <span className="text-gray-300">Submit Schema</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4">
        <h1 className="text-3xl font-bold tracking-tight text-white flex items-center gap-2">
          <Upload className="w-8 h-8 text-blue-500" />
          Submit CosmWasm Schema
        </h1>
        <p className="text-gray-400 mt-2">Upload JSON schemas to automatically generate query and execution forms.</p>
      </div>

      {success && (
        <div className="p-4 bg-green-950/30 border border-green-800 rounded-lg flex items-start gap-3">
          <CheckCircle2 className="w-5 h-5 text-green-400 mt-0.5" />
          <div>
            <h3 className="font-semibold text-green-400">Schema Registered</h3>
            <p className="text-sm text-green-300 mt-1">CosmWasm Code ID {codeId} schema has been uploaded and compiled successfully.</p>
          </div>
        </div>
      )}

      {error && (
        <div className="p-4 bg-red-950/30 border border-red-800 rounded-lg flex items-start gap-3">
          <AlertCircle className="w-5 h-5 text-red-400 mt-0.5" />
          <p className="text-sm text-red-300">{error}</p>
        </div>
      )}

      <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
        <form onSubmit={handleSubmit} className="space-y-6">
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-300">Code ID</label>
            <input
              type="number"
              placeholder="1"
              value={codeId}
              onChange={(e) => setCodeId(e.target.value)}
              className="w-full bg-gray-950 border border-gray-800 rounded-lg px-4 py-2.5 text-white font-mono text-sm focus:outline-none focus:border-blue-500"
            />
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-300">JSON Schema (ABI Specification)</label>
            <textarea
              rows={12}
              placeholder='{ "contract_name": "cw20_base", "query": { ... }, "execute": { ... } }'
              value={schemaText}
              onChange={(e) => setSchemaText(e.target.value)}
              className="w-full bg-gray-950 border border-gray-800 rounded-lg px-4 py-2.5 text-white font-mono text-sm focus:outline-none focus:border-blue-500"
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full py-3 bg-blue-600 hover:bg-blue-500 disabled:bg-gray-800 text-white rounded-lg font-medium transition"
          >
            {loading ? "Registering Schema..." : "Submit Schema"}
          </button>
        </form>
      </div>
    </div>
  );
}
