"use client";

import React, { useState } from "react";
import Link from "next/link";
import { Terminal, Code, Settings, Copy, Check, Info, Shield, Cpu, ExternalLink } from "lucide-react";
import { useWalletStore } from "@/store/wallet";

interface CodeSnippet {
  title: string;
  lang: string;
  code: string;
  description: string;
}

export default function DEVELOPERSPage() {
  const { walletType, connected, address, connectWallet, disconnectWallet } = useWalletStore();
  const [activeTab, setActiveTab] = useState<"wallet" | "hardhat" | "jssdk" | "cosmos">("wallet");
  const [copiedIndex, setCopiedIndex] = useState<number | null>(null);

  const copyToClipboard = (text: string, index: number) => {
    navigator.clipboard.writeText(text);
    setCopiedIndex(index);
    setTimeout(() => setCopiedIndex(null), 2000);
  };

  const walletSnippets: CodeSnippet[] = [
    {
      title: "MetaMask — Add Network (via window.ethereum)",
      lang: "javascript",
      description: "Run this script inside your web application to prompt users to add Sovereign L1 to MetaMask.",
      code: `await window.ethereum.request({
  method: 'wallet_addEthereumChain',
  params: [{
    chainId: '0x1E61',           // 7777 in hex
    chainName: 'Sovereign L1',
    nativeCurrency: { name: 'SLT', symbol: 'SLT', decimals: 18 },
    rpcUrls: ['http://localhost:8545'],
    blockExplorerUrls: ['http://localhost:3000/evm']
  }]
});`
    }
  ];

  const hardhatSnippets: CodeSnippet[] = [
    {
      title: "Hardhat — hardhat.config.ts",
      lang: "typescript",
      description: "Define Sovereign L1 inside your hardhat configuration file.",
      code: `import { HardhatUserConfig } from "hardhat/config";
import "@nomicfoundation/hardhat-toolbox";

const config: HardhatUserConfig = {
  solidity: "0.8.24",
  networks: {
    sovereign: {
      url: "http://localhost:8545",
      chainId: 7777,
      accounts: [process.env.PRIVATE_KEY!],
    },
  },
};

export default config;`
    },
    {
      title: "Foundry — foundry.toml",
      lang: "toml",
      description: "Configure the RPC endpoint in foundry.toml.",
      code: `[rpc_endpoints]
sovereign = "http://localhost:8545"`
    },
    {
      title: "Foundry — Deploy & Verify Commands",
      lang: "bash",
      description: "Deploy a smart contract via Forge and verify the source code.",
      code: `# Deploy contract
forge create --rpc-url sovereign --private-key $PRIVATE_KEY src/MyContract.sol:MyContract

# Verify contract
forge verify-contract --chain-id 7777 --etherscan-api-url http://localhost:3000/api \\
  <CONTRACT_ADDR> src/MyContract.sol:MyContract`
    }
  ];

  const jssdkSnippets: CodeSnippet[] = [
    {
      title: "ethers.js v6",
      lang: "typescript",
      description: "Initialize provider and sign transactions using ethers.js.",
      code: `import { JsonRpcProvider, Wallet } from "ethers";

const provider = new JsonRpcProvider("http://localhost:8545");
const signer = new Wallet(process.env.PRIVATE_KEY!, provider);`
    },
    {
      title: "viem",
      lang: "typescript",
      description: "Interact with the EVM runtime using viem clients.",
      code: `import { createPublicClient, createWalletClient, http } from "viem";
import { privateKeyToAccount } from "viem/accounts";

const sovereignChain = {
  id: 7777,
  name: "Sovereign L1",
  nativeCurrency: { name: "SLT", symbol: "SLT", decimals: 18 },
  rpcUrls: { default: { http: ["http://localhost:8545"] } }
};

const publicClient = createPublicClient({ 
  chain: sovereignChain, 
  transport: http() 
});

const walletClient = createWalletClient({ 
  account: privateKeyToAccount(process.env.PRIVATE_KEY!), 
  chain: sovereignChain, 
  transport: http() 
});`
    },
    {
      title: "wagmi v2 config",
      lang: "typescript",
      description: "Set up wagmi config with defineChain helper.",
      code: `import { createConfig, http } from "wagmi";
import { defineChain } from "viem";

const sovereign = defineChain({
  id: 7777,
  name: "Sovereign L1",
  nativeCurrency: { name: "SLT", symbol: "SLT", decimals: 18 },
  rpcUrls: { default: { http: ["http://localhost:8545"] } }
});

export const config = createConfig({
  chains: [sovereign],
  transports: {
    [sovereign.id]: http()
  }
});`
    }
  ];

  const cosmosSnippets: CodeSnippet[] = [
    {
      title: "CosmJS — Cosmos SDK (StargateClient)",
      lang: "typescript",
      description: "Establish a connection and query details using CosmJS.",
      code: `import { StargateClient } from "@cosmjs/stargate";

const client = await StargateClient.connect("http://localhost:26657");

// or with signing:
import { SigningStargateClient, DirectSecp256k1HdWallet } from "@cosmjs/stargate";

const signer = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, { prefix: "sov" });
const signingClient = await SigningStargateClient.connectWithSigner("http://localhost:26657", signer);`
    },
    {
      title: "CosmWasm — Upload & Execute",
      lang: "typescript",
      description: "Upload and interact with CosmWasm smart contracts.",
      code: `import { SigningCosmWasmClient } from "@cosmjs/cosmwasm-stargate";

const client = await SigningCosmWasmClient.connectWithSigner("http://localhost:26657", signer);

// Upload wasm bytecode:
const { codeId } = await client.upload(senderAddr, wasmBytes, "auto");

// Instantiate contract:
const { contractAddress } = await client.instantiate(senderAddr, codeId, initMsg, "label", "auto");

// Execute execution message:
const result = await client.execute(senderAddr, contractAddress, executeMsg, "auto");`
    },
    {
      title: "wasmd CLI — CosmWasm Deploy Commands",
      lang: "bash",
      description: "Manage Wasm contracts using the native wasmd client CLI.",
      code: `# Upload wasm binary
wasmd tx wasm store contract.wasm --from mykey --chain-id sovereign-1 --node http://localhost:26657 --gas auto

# Instantiate contract
wasmd tx wasm instantiate <code_id> '{"key":"value"}' --from mykey --label "MyContract" \\
  --chain-id sovereign-1 --node http://localhost:26657 --no-admin

# Execute contract message
wasmd tx wasm execute <contract_addr> '{"action":{}}' --from mykey \\
  --chain-id sovereign-1 --node http://localhost:26657`
    }
  ];

  const getActiveSnippets = () => {
    switch (activeTab) {
      case "wallet": return walletSnippets;
      case "hardhat": return hardhatSnippets;
      case "jssdk": return jssdkSnippets;
      case "cosmos": return cosmosSnippets;
    }
  };

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white">Developers</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4 flex justify-between items-center">
        <div className="flex items-center space-x-3">
          <Terminal className="text-blue-500 h-8 w-8 animate-pulse" />
          <div>
            <h1 className="text-3xl font-bold tracking-tight text-white">Developer Hub</h1>
            <p className="text-gray-400 mt-1">Integration guides, configs, and commands for building on Sovereign L1</p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Sidebar Nav */}
        <div className="space-y-2 lg:col-span-1">
          <h3 className="text-xs uppercase font-bold text-gray-500 px-3 mb-2 tracking-wider">
            Integration Guides
          </h3>
          <button
            onClick={() => setActiveTab("wallet")}
            className={`w-full text-left px-3 py-2 rounded-xl text-sm font-medium transition ${
              activeTab === "wallet" ? "bg-blue-950 text-blue-400 border-l-2 border-blue-500" : "text-gray-400 hover:bg-gray-900/50 hover:text-white"
            }`}
          >
            MetaMask / Wallets
          </button>
          <button
            onClick={() => setActiveTab("hardhat")}
            className={`w-full text-left px-3 py-2 rounded-xl text-sm font-medium transition ${
              activeTab === "hardhat" ? "bg-blue-950 text-blue-400 border-l-2 border-blue-500" : "text-gray-400 hover:bg-gray-900/50 hover:text-white"
            }`}
          >
            Hardhat & Foundry
          </button>
          <button
            onClick={() => setActiveTab("jssdk")}
            className={`w-full text-left px-3 py-2 rounded-xl text-sm font-medium transition ${
              activeTab === "jssdk" ? "bg-blue-950 text-blue-400 border-l-2 border-blue-500" : "text-gray-400 hover:bg-gray-900/50 hover:text-white"
            }`}
          >
            JS SDKs (ethers, viem)
          </button>
          <button
            onClick={() => setActiveTab("cosmos")}
            className={`w-full text-left px-3 py-2 rounded-xl text-sm font-medium transition ${
              activeTab === "cosmos" ? "bg-blue-950 text-blue-400 border-l-2 border-blue-500" : "text-gray-400 hover:bg-gray-900/50 hover:text-white"
            }`}
          >
            Cosmos & CosmWasm
          </button>

          {/* Verification link */}
          <div className="pt-4 border-t border-gray-900 mt-4 px-3 space-y-3">
            <h4 className="text-xs uppercase font-bold text-gray-500 tracking-wider">Verification</h4>
            <Link 
              href="/verify" 
              className="flex items-center space-x-1 text-xs text-blue-500 hover:underline font-medium"
            >
              <span>Verify contract source</span>
              <ExternalLink className="h-3 w-3" />
            </Link>
          </div>
        </div>

        {/* Snippets Area */}
        <div className="lg:col-span-2 space-y-6">
          {getActiveSnippets().map((snippet, idx) => (
            <div key={idx} className="bg-gray-950 border border-gray-900 rounded-xl p-6 space-y-3 shadow-lg">
              <div>
                <h3 className="text-base font-bold text-white flex items-center space-x-2">
                  <Code className="h-4 w-4 text-blue-500" />
                  <span>{snippet.title}</span>
                </h3>
                <p className="text-xs text-gray-500 mt-1">{snippet.description}</p>
              </div>

              {/* Fenced code block with copy action */}
              <div className="relative group">
                <button
                  onClick={() => copyToClipboard(snippet.code, idx)}
                  className="absolute right-3 top-3 p-1.5 bg-gray-900/80 border border-gray-800 rounded-lg text-gray-400 hover:text-white hover:bg-gray-800 transition opacity-0 group-hover:opacity-100 focus:opacity-100"
                  title="Copy code"
                >
                  {copiedIndex === idx ? <Check className="h-3.5 w-3.5 text-green-500" /> : <Copy className="h-3.5 w-3.5" />}
                </button>
                <pre className="font-mono text-xs text-gray-300 bg-black/40 border border-gray-900 rounded-lg p-4 overflow-x-auto leading-relaxed max-h-[350px]">
                  <code>{snippet.code}</code>
                </pre>
              </div>
            </div>
          ))}
        </div>

        {/* Network Config Reference Widget */}
        <div className="space-y-4 lg:col-span-1">
          <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 shadow-lg space-y-4">
            <h3 className="text-sm font-extrabold text-white flex items-center space-x-2 border-b border-gray-900 pb-2">
              <Settings className="h-4 w-4 text-blue-500" />
              <span>Network Parameters</span>
            </h3>

            <div className="space-y-3 text-xs">
              <div>
                <span className="block text-gray-500 uppercase font-bold tracking-wider mb-0.5">EVM Chain ID</span>
                <span className="font-mono font-semibold text-white">7777 (0x1E61)</span>
              </div>
              <div>
                <span className="block text-gray-500 uppercase font-bold tracking-wider mb-0.5">Cosmos Chain ID</span>
                <span className="font-mono font-semibold text-white">sovereign-1</span>
              </div>
              <div>
                <span className="block text-gray-500 uppercase font-bold tracking-wider mb-0.5">EVM RPC (HTTP)</span>
                <span className="font-mono font-semibold text-gray-300 break-all select-all">http://localhost:8545</span>
              </div>
              <div>
                <span className="block text-gray-500 uppercase font-bold tracking-wider mb-0.5">Cosmos RPC (CometBFT)</span>
                <span className="font-mono font-semibold text-gray-300 break-all select-all">http://localhost:26657</span>
              </div>
              <div>
                <span className="block text-gray-500 uppercase font-bold tracking-wider mb-0.5">Cosmos gRPC</span>
                <span className="font-mono font-semibold text-gray-300 break-all select-all">localhost:9090</span>
              </div>
              <div>
                <span className="block text-gray-500 uppercase font-bold tracking-wider mb-0.5">Native Token</span>
                <span className="font-semibold text-white">CSOV (ucsov)</span>
              </div>
            </div>

            <div className="p-3 bg-blue-950/20 border border-blue-900/50 rounded-lg text-[11px] text-blue-400 flex items-start space-x-2 leading-relaxed">
              <Info className="h-4 w-4 mt-0.5 flex-shrink-0" />
              <span>Local Devnet RPC endpoints. Use these configs in your scripts or configuration files to connect and sign.</span>
            </div>
          </div>
        </div>
      </div>

      {/* SVG Deployment Pipeline Flowchart */}
      <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-xl space-y-6">
        <h2 className="text-xl font-bold text-white flex items-center gap-2">
          <Cpu className="text-purple-500 h-5 w-5 animate-pulse" />
          Sovereign Smart Contract Deployment & Verification Pipeline
        </h2>
        <div className="flex justify-center p-4 bg-black/40 rounded-xl border border-gray-900">
          <svg viewBox="0 0 800 150" className="w-full max-w-4xl text-xs font-mono">
            {/* Definitions for markers/gradients */}
            <defs>
              <marker id="arrow" viewBox="0 0 10 10" refX="6" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
                <path d="M 0 2 L 8 5 L 0 8 z" fill="#4b5563" />
              </marker>
            </defs>

            {/* Step 1: Compile */}
            <rect x="10" y="30" width="160" height="70" rx="10" fill="#1e1b4b" stroke="#3730a3" strokeWidth="1.5" />
            <text x="90" y="60" fill="#fff" fontWeight="bold" textAnchor="middle">1. COMPILE</text>
            <text x="90" y="80" fill="#a5b4fc" fontSize="10" textAnchor="middle">Solidity / Wasm bytecode</text>

            <line x1="170" y1="65" x2="210" y2="65" stroke="#4b5563" strokeWidth="2" markerEnd="url(#arrow)" />

            {/* Step 2: Connect Wallet */}
            <rect x="220" y="30" width="160" height="70" rx="10" fill="#14532d" stroke="#166534" strokeWidth="1.5" />
            <text x="300" y="60" fill="#fff" fontWeight="bold" textAnchor="middle">2. AUTHENTICATE</text>
            <text x="300" y="80" fill="#86efac" fontSize="10" textAnchor="middle">Connect MetaMask/Keplr</text>

            <line x1="380" y1="65" x2="420" y2="65" stroke="#4b5563" strokeWidth="2" markerEnd="url(#arrow)" />

            {/* Step 3: Broadcast */}
            <rect x="430" y="30" width="160" height="70" rx="10" fill="#701a75" stroke="#86198f" strokeWidth="1.5" />
            <text x="510" y="60" fill="#fff" fontWeight="bold" textAnchor="middle">3. BROADCAST</text>
            <text x="510" y="80" fill="#f5d0fe" fontSize="10" textAnchor="middle">Sign & transmit to RPC</text>

            <line x1="590" y1="65" x2="630" y2="65" stroke="#4b5563" strokeWidth="2" markerEnd="url(#arrow)" />

            {/* Step 4: Verify */}
            <rect x="640" y="30" width="150" height="70" rx="10" fill="#065f46" stroke="#047857" strokeWidth="1.5" />
            <text x="715" y="60" fill="#fff" fontWeight="bold" textAnchor="middle">4. VERIFY</text>
            <text x="715" y="80" fill="#a7f3d0" fontSize="10" textAnchor="middle">Sourcify code upload</text>
          </svg>
        </div>
      </div>
    </div>
  );
}
