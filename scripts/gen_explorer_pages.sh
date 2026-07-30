#!/bin/bash
cd "$(dirname "$0")/../explorer" || exit 1

gen_data_page() {
  local filepath="$1" title="$2" apiPath="$3" schema="$4" icon="$5"
  mkdir -p "$(dirname "$filepath")"
  cat > "$filepath" << EOF
"use client";

import React from "react";
import Link from "next/link";
import { ${icon} } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { ${schema} } from "@/lib/schemas";

export default function Page() {
  const { data, isLoading } = useQuery({
    queryKey: ["${title}"],
    queryFn: () => apiClient.get("${apiPath}", ${schema}),
  });

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white">${title}</span>
      </nav>
      <div>
        <h1 className="text-3xl font-bold text-white flex items-center gap-3">
          <${icon} className="h-8 w-8 text-cyan-400" />
          ${title}
        </h1>
      </div>
      {isLoading ? (
        <div className="py-20 text-center text-gray-400">Loading...</div>
      ) : !data ? (
        <div className="py-20 text-center text-gray-500">No data available</div>
      ) : (
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-6">
          <pre className="text-xs text-gray-300 font-mono overflow-auto">{JSON.stringify(data, null, 2)}</pre>
        </div>
      )}
    </div>
  );
}
EOF
}

gen_static_page() {
  local filepath="$1" title="$2" icon="$3" desc="$4"
  mkdir -p "$(dirname "$filepath")"
  cat > "$filepath" << EOF
"use client";

import React from "react";
import Link from "next/link";
import { ${icon} } from "lucide-react";

export default function Page() {
  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white">${title}</span>
      </nav>
      <div>
        <h1 className="text-3xl font-bold text-white flex items-center gap-3">
          <${icon} className="h-8 w-8 text-cyan-400" />
          ${title}
        </h1>
        <p className="text-gray-400 mt-2">${desc}</p>
      </div>
      <div className="bg-gray-950 border border-gray-900 rounded-xl p-8 text-center">
        <p className="text-gray-500 text-sm">This page is under construction. Full functionality will be available soon.</p>
      </div>
    </div>
  );
}
EOF
}

gen_dynamic_page() {
  local filepath="$1" title="$2" icon="$3" desc="$4"
  mkdir -p "$(dirname "$filepath")"
  cat > "$filepath" << EOF
"use client";

import React from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { ${icon} } from "lucide-react";

export default function Page() {
  const params = useParams();
  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white">${title}</span>
      </nav>
      <div>
        <h1 className="text-3xl font-bold text-white flex items-center gap-3">
          <${icon} className="h-8 w-8 text-cyan-400" />
          ${title}
        </h1>
        <p className="text-gray-400 mt-2">${desc}</p>
      </div>
      <div className="bg-gray-950 border border-gray-900 rounded-xl p-6">
        <p className="text-gray-400 text-sm">Viewing: <span className="font-mono text-white">{JSON.stringify(params)}</span></p>
      </div>
    </div>
  );
}
EOF
}

# Data-driven pages
gen_data_page "app/bridge/relayers/page.tsx" "Bridge Relayers" "/api/rest/v1/explorer/bridge/relayers" "RelayerListSchema" "Users"
gen_data_page "app/bridge/nonces/page.tsx" "Nonce Registry" "/api/rest/v1/explorer/bridge/nonces" "NonceRegistrySchema" "Hash"
gen_data_page "app/txs/withdrawals/page.tsx" "Staking Withdrawals" "/api/rest/v1/explorer/txs/withdrawals" "TxListSchema" "ArrowDownLeft"
gen_data_page "app/blocks/reorgs/page.tsx" "Forked Blocks" "/api/rest/v1/explorer/blocks/reorgs" "BlockListSchema" "GitBranch"
gen_data_page "app/txs/erc20/page.tsx" "ERC-20 Transfers" "/api/rest/v1/explorer/tokens/evm/transfers" "TokenTransferListSchema" "Coins"
gen_data_page "app/txs/erc721/page.tsx" "ERC-721 Transfers" "/api/rest/v1/explorer/nfts/evm/transfers" "TokenTransferListSchema" "Image"
gen_data_page "app/txs/erc1155/page.tsx" "ERC-1155 Transfers" "/api/rest/v1/explorer/nfts/evm/transfers-1155" "TokenTransferListSchema" "Layers"
gen_data_page "app/evm/tokens/multi/page.tsx" "ERC-1155 Collections" "/api/rest/v1/explorer/evm/tokens/multi" "EvmTokenListSchema" "Layers"
gen_data_page "app/nfts/top-mints/page.tsx" "Top NFT Mints" "/api/rest/v1/explorer/nfts/top-mints" "EvmTokenListSchema" "TrendingUp"
gen_data_page "app/nfts/latest-trades/page.tsx" "Latest NFT Trades" "/api/rest/v1/explorer/nfts/latest-trades" "TokenTransferListSchema" "ArrowRightLeft"
gen_data_page "app/nfts/latest-transfers/page.tsx" "Latest NFT Transfers" "/api/rest/v1/explorer/nfts/latest-transfers" "TokenTransferListSchema" "ArrowRight"
gen_data_page "app/nfts/latest-mints/page.tsx" "Latest NFT Mints" "/api/rest/v1/explorer/nfts/latest-mints" "TokenTransferListSchema" "Sparkles"
gen_data_page "app/labelcloud/page.tsx" "Label Cloud" "/api/rest/v1/explorer/labels" "SearchResultsSchema" "Tags"
gen_data_page "app/gas/guzzlers/page.tsx" "Gas Guzzlers" "/api/rest/v1/explorer/gas/guzzlers" "TopAccountListSchema" "Flame"
gen_data_page "app/gas/spenders/page.tsx" "Gas Spenders" "/api/rest/v1/explorer/gas/spenders" "TopAccountListSchema" "Wallet"
gen_data_page "app/txs/internal/page.tsx" "Internal Transactions" "/api/rest/v1/explorer/txs/internal" "TxListSchema" "ArrowRightLeft"

# Static pages
gen_static_page "app/tools/unit-converter/page.tsx" "Unit Converter" "Calculator" "Convert between SOV, uSOV, Wei, Gwei, and USD."
gen_static_page "app/tools/broadcast/page.tsx" "Broadcast Transaction" "Send" "Submit a signed raw transaction to the chain."
gen_static_page "app/tools/abi-encoder/page.tsx" "ABI Encoder" "FileCode" "Encode and decode ABI function calls."
gen_static_page "app/tools/disassembler/page.tsx" "Bytecode Disassembler" "Cpu" "Disassemble EVM bytecode to opcodes."
gen_static_page "app/tools/similar-contracts/page.tsx" "Similar Contracts" "Copy" "Find contracts with matching bytecode."
gen_static_page "app/tools/contract-diff/page.tsx" "Contract Diff Checker" "GitCompare" "Compare two contracts side by side."
gen_static_page "app/tools/constructor-args/page.tsx" "Constructor Args Decoder" "Braces" "Decode ABI-encoded constructor arguments."
gen_static_page "app/tools/verify-signature/page.tsx" "Verify Signature" "ShieldCheck" "Verify EIP-191 or Cosmos ADR-036 signatures."
gen_static_page "app/tools/compiler/page.tsx" "Online Compiler" "Terminal" "Browser-based Solidity compiler."
gen_static_page "app/tools/code-reader/page.tsx" "AI Code Reader" "Bot" "AI-powered contract explanation."
gen_static_page "app/api-plans/page.tsx" "API Plans" "Key" "API rate tier plans and key management."
gen_static_page "app/myaccount/page.tsx" "My Account" "User" "Watchlist, private notes, API keys, and notifications."
gen_static_page "app/directory/page.tsx" "Project Directory" "FolderTree" "Curated list of projects on Sovereign L1."
gen_static_page "app/domains/page.tsx" "Domain Names" "Globe" "Name service registry and lookup."

# Dynamic pages
gen_dynamic_page "app/tokens/[addr]/page.tsx" "Token Detail" "Coins" "Token information, holders, transfers, and price data."
gen_dynamic_page "app/evm/tokens/[addr]/multi/page.tsx" "ERC-1155 Token Detail" "Layers" "Multi-token detail with token IDs and balances."
# Note: /evm/nfts/[addr]/[tokenId] already exists — do not create [id] variant (causes ambiguous route error)
gen_dynamic_page "app/contracts/verified/[type]/page.tsx" "Verified Contracts" "BadgeCheck" "Browse verified contracts by type."
gen_dynamic_page "app/address/[any]/tokencheck/page.tsx" "Token Approval Checker" "ShieldAlert" "Check ERC-20 token approvals for this address."
gen_dynamic_page "app/label/[slug]/page.tsx" "Addresses by Label" "Tag" "All addresses with this label tag."
gen_dynamic_page "app/tokens/[addr]/submit/page.tsx" "Submit Token Info" "Upload" "Submit token logo and project information."

# Chart sub-pages (22 routes)
for chart in tx tx-fee active-addresses new-addresses unique-addresses gas-price gas-used block-size block-time tps block-count burnt-fees price market-cap validator-count staking-ratio contracts-deployed contracts-verified token-transfers nft-transfers bridge-volume ibc-volume; do
  title=$(echo "$chart" | sed 's/-/ /g' | sed 's/\b./\U&/g')
  gen_data_page "app/charts/${chart}/page.tsx" "${title} Chart" "/api/rest/v1/explorer/charts/${chart}" "ChartDataSchema" "BarChart3"
done

echo "All pages generated"