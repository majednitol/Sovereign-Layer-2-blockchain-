"use client";

import { useState } from "react";
import { Copy, Check } from "lucide-react";

interface AddressBadgeProps {
  bech32?: string | null;
  hex?: string | null;
  truncateBech32?: number;
  truncateHex?: number;
  copyable?: boolean;
  className?: string;
}

export default function AddressBadge({
  bech32,
  hex,
  truncateBech32 = 12,
  truncateHex = 10,
  copyable = true,
  className = "",
}: AddressBadgeProps) {
  const [copied, setCopied] = useState<string | null>(null);

  const copyToClipboard = async (text: string, label: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(label);
      setTimeout(() => setCopied(null), 1500);
    } catch {
      // silent
    }
  };

  const displayBech32 =
    bech32 && truncateBech32
      ? `${bech32.slice(0, truncateBech32)}…${bech32.slice(-4)}`
      : bech32 || "—";
  const displayHex =
    hex && truncateHex
      ? `${hex.slice(0, truncateHex)}…${hex.slice(-4)}`
      : hex || "—";

  return (
    <div className={`inline-flex flex-col gap-1 ${className}`}>
      {bech32 && (
        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-gray-500 uppercase tracking-wider">
            Cosmos
          </span>
          <code className="text-xs font-mono text-gray-300 bg-gray-900/60 border border-gray-800 rounded px-2 py-0.5 break-all">
            {displayBech32}
          </code>
          {copyable && (
            <button
              onClick={() => copyToClipboard(bech32, "bech32")}
              className="text-gray-500 hover:text-white transition"
              title="Copy bech32 address"
            >
              {copied === "bech32" ? (
                <Check className="h-3.5 w-3.5 text-green-400" />
              ) : (
                <Copy className="h-3.5 w-3.5" />
              )}
            </button>
          )}
        </div>
      )}
      {hex && (
        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-gray-500 uppercase tracking-wider">
            EVM
          </span>
          <code className="text-xs font-mono text-gray-300 bg-gray-900/60 border border-gray-800 rounded px-2 py-0.5 break-all">
            {displayHex}
          </code>
          {copyable && (
            <button
              onClick={() => copyToClipboard(hex, "hex")}
              className="text-gray-500 hover:text-white transition"
              title="Copy hex address"
            >
              {copied === "hex" ? (
                <Check className="h-3.5 w-3.5 text-green-400" />
              ) : (
                <Copy className="h-3.5 w-3.5" />
              )}
            </button>
          )}
        </div>
      )}
    </div>
  );
}
