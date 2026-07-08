"use client";

import React, { useState } from "react";
import Link from "next/link";
import { ArrowLeft, Download, FileText, Calendar, Info } from "lucide-react";

export default function ExportDataPage() {
  const [exportType, setExportType] = useState("txs");
  const [address, setAddress] = useState("");
  const [loading, setLoading] = useState(false);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  const handleExport = () => {
    setLoading(true);
    let downloadUrl = "";
    if (exportType === "txs") {
      downloadUrl = `${API_BASE}/api/rest/v1/explorer/charts/tx?format=csv`;
    } else if (exportType === "gas") {
      downloadUrl = `${API_BASE}/api/rest/v1/explorer/charts/gas-used?format=csv`;
    } else {
      downloadUrl = `${API_BASE}/api/rest/v1/explorer/charts/bridge-volume?format=csv`;
    }
    
    // Trigger download
    const link = document.createElement("a");
    link.href = downloadUrl;
    link.download = `${exportType}_export_data.csv`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    
    setTimeout(() => setLoading(false), 1000);
  };

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white font-medium">Export Data Hub</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-900 pb-4 flex items-center space-x-3">
        <Link href="/" className="p-2 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-gray-400 hover:text-white transition">
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
            <Download className="text-blue-500 w-8 h-8" />
            Download & Export Data Hub
          </h1>
          <p className="text-gray-400 mt-1">Export transaction logs, blocks, and gas charts into raw CSV spreadsheet files.</p>
        </div>
      </div>

      <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-xl space-y-6 max-w-lg mx-auto">
        <div className="space-y-4 text-xs">
          <div>
            <label className="block text-gray-400 font-bold uppercase mb-1">Export Database Category</label>
            <select
              value={exportType}
              onChange={(e) => setExportType(e.target.value)}
              className="w-full bg-black border border-gray-900 rounded-xl p-3 text-white focus:border-blue-500 outline-none cursor-pointer"
            >
              <option value="txs">Transactions volume statistics (CSV)</option>
              <option value="gas">Gas used history stats (CSV)</option>
              <option value="bridge">Bridge deposits & withdrawals volume (CSV)</option>
            </select>
          </div>

          {exportType === "txs" && (
            <div>
              <label className="block text-gray-400 font-bold uppercase mb-1">Filter by Wallet Address (Optional)</label>
              <input 
                type="text" 
                placeholder="sov1... or 0x..." 
                value={address}
                onChange={(e) => setAddress(e.target.value)}
                className="w-full bg-black border border-gray-900 rounded-xl p-3 text-white focus:border-blue-500 outline-none font-mono"
              />
            </div>
          )}

          <button 
            onClick={handleExport}
            disabled={loading}
            className="w-full py-3 bg-blue-600 hover:bg-blue-500 disabled:bg-gray-800 text-white font-bold text-xs uppercase tracking-wider rounded-xl transition flex items-center justify-center gap-1.5"
          >
            <FileText className="h-4 w-4" />
            {loading ? "Generating CSV..." : "Download Export Spreadsheet"}
          </button>
        </div>

        <div className="p-3.5 bg-blue-950/20 border border-blue-900/50 rounded-xl text-[11px] text-blue-400 flex items-start space-x-2 leading-relaxed">
          <Info className="h-4 w-4 mt-0.5 flex-shrink-0" />
          <span>CSVs are streamed live from the TimescaleDB ledger hypertable logs. For queries exceeding 10,000 rows, consider using RPC exports.</span>
        </div>
      </div>
    </div>
  );
}
