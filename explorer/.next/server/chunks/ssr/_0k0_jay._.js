module.exports=[5781,a=>{"use strict";let b=(0,a.i(66238).default)("Cpu",[["rect",{width:"16",height:"16",x:"4",y:"4",rx:"2",key:"14l7u7"}],["rect",{width:"6",height:"6",x:"9",y:"9",rx:"1",key:"5aljv4"}],["path",{d:"M15 2v2",key:"13l42r"}],["path",{d:"M15 20v2",key:"15mkzm"}],["path",{d:"M2 15h2",key:"1gxd5l"}],["path",{d:"M2 9h2",key:"1bbxkp"}],["path",{d:"M20 15h2",key:"19e6y8"}],["path",{d:"M20 9h2",key:"19tzq7"}],["path",{d:"M9 2v2",key:"165o2o"}],["path",{d:"M9 20v2",key:"i2bqo8"}]]);a.s(["Cpu",0,b],5781)},1586,a=>{"use strict";let b=(0,a.i(66238).default)("Code",[["polyline",{points:"16 18 22 12 16 6",key:"z7tu5w"}],["polyline",{points:"8 6 2 12 8 18",key:"1eg1df"}]]);a.s(["Code",0,b],1586)},22644,a=>{"use strict";let b=(0,a.i(66238).default)("Terminal",[["polyline",{points:"4 17 10 11 4 5",key:"akl6gq"}],["line",{x1:"12",x2:"20",y1:"19",y2:"19",key:"q2wloq"}]]);a.s(["Terminal",0,b],22644)},58060,a=>{"use strict";let b=(0,a.i(66238).default)("Settings",[["path",{d:"M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z",key:"1qme2f"}],["circle",{cx:"12",cy:"12",r:"3",key:"1v7zrd"}]]);a.s(["Settings",0,b],58060)},75862,a=>{"use strict";let b=(0,a.i(66238).default)("Info",[["circle",{cx:"12",cy:"12",r:"10",key:"1mglay"}],["path",{d:"M12 16v-4",key:"1dtifu"}],["path",{d:"M12 8h.01",key:"e9boi3"}]]);a.s(["Info",0,b],75862)},54426,a=>{"use strict";var b=a.i(53914),c=a.i(35100),d=a.i(42458),e=a.i(22644),f=a.i(1586),g=a.i(58060);let h=(0,a.i(66238).default)("Copy",[["rect",{width:"14",height:"14",x:"8",y:"8",rx:"2",ry:"2",key:"17jyea"}],["path",{d:"M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2",key:"zix9uf"}]]);var i=a.i(21712),j=a.i(75862),k=a.i(5781),l=a.i(10021),m=a.i(23356);a.s(["default",0,function(){let{walletType:a,connected:n,address:o,connectWallet:p,disconnectWallet:q}=(0,m.useWalletStore)(),[r,s]=(0,c.useState)("wallet"),[t,u]=(0,c.useState)(null),v=[{title:"MetaMask — Add Network (via window.ethereum)",lang:"javascript",description:"Run this script inside your web application to prompt users to add Sovereign L1 to MetaMask.",code:`await window.ethereum.request({
  method: 'wallet_addEthereumChain',
  params: [{
    chainId: '0x1E61',           // 7777 in hex
    chainName: 'Sovereign L1',
    nativeCurrency: { name: 'SLT', symbol: 'SLT', decimals: 18 },
    rpcUrls: ['http://localhost:8545'],
    blockExplorerUrls: ['http://localhost:3000/evm']
  }]
});`}],w=[{title:"Hardhat — hardhat.config.ts",lang:"typescript",description:"Define Sovereign L1 inside your hardhat configuration file.",code:`import { HardhatUserConfig } from "hardhat/config";
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

export default config;`},{title:"Foundry — foundry.toml",lang:"toml",description:"Configure the RPC endpoint in foundry.toml.",code:`[rpc_endpoints]
sovereign = "http://localhost:8545"`},{title:"Foundry — Deploy & Verify Commands",lang:"bash",description:"Deploy a smart contract via Forge and verify the source code.",code:`# Deploy contract
forge create --rpc-url sovereign --private-key $PRIVATE_KEY src/MyContract.sol:MyContract

# Verify contract
forge verify-contract --chain-id 7777 --etherscan-api-url http://localhost:3000/api \\
  <CONTRACT_ADDR> src/MyContract.sol:MyContract`}],x=[{title:"ethers.js v6",lang:"typescript",description:"Initialize provider and sign transactions using ethers.js.",code:`import { JsonRpcProvider, Wallet } from "ethers";

const provider = new JsonRpcProvider("http://localhost:8545");
const signer = new Wallet(process.env.PRIVATE_KEY!, provider);`},{title:"viem",lang:"typescript",description:"Interact with the EVM runtime using viem clients.",code:`import { createPublicClient, createWalletClient, http } from "viem";
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
});`},{title:"wagmi v2 config",lang:"typescript",description:"Set up wagmi config with defineChain helper.",code:`import { createConfig, http } from "wagmi";
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
});`}],y=[{title:"CosmJS — Cosmos SDK (StargateClient)",lang:"typescript",description:"Establish a connection and query details using CosmJS.",code:`import { StargateClient } from "@cosmjs/stargate";

const client = await StargateClient.connect("http://localhost:26657");

// or with signing:
import { SigningStargateClient, DirectSecp256k1HdWallet } from "@cosmjs/stargate";

const signer = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, { prefix: "sov" });
const signingClient = await SigningStargateClient.connectWithSigner("http://localhost:26657", signer);`},{title:"CosmWasm — Upload & Execute",lang:"typescript",description:"Upload and interact with CosmWasm smart contracts.",code:`import { SigningCosmWasmClient } from "@cosmjs/cosmwasm-stargate";

const client = await SigningCosmWasmClient.connectWithSigner("http://localhost:26657", signer);

// Upload wasm bytecode:
const { codeId } = await client.upload(senderAddr, wasmBytes, "auto");

// Instantiate contract:
const { contractAddress } = await client.instantiate(senderAddr, codeId, initMsg, "label", "auto");

// Execute execution message:
const result = await client.execute(senderAddr, contractAddress, executeMsg, "auto");`},{title:"wasmd CLI — CosmWasm Deploy Commands",lang:"bash",description:"Manage Wasm contracts using the native wasmd client CLI.",code:`# Upload wasm binary
wasmd tx wasm store contract.wasm --from mykey --chain-id sovereign-1 --node http://localhost:26657 --gas auto

# Instantiate contract
wasmd tx wasm instantiate <code_id> '{"key":"value"}' --from mykey --label "MyContract" \\
  --chain-id sovereign-1 --node http://localhost:26657 --no-admin

# Execute contract message
wasmd tx wasm execute <contract_addr> '{"action":{}}' --from mykey \\
  --chain-id sovereign-1 --node http://localhost:26657`}];return(0,b.jsxs)("div",{className:"p-6 max-w-7xl mx-auto space-y-6",children:[(0,b.jsxs)("nav",{className:"text-sm text-gray-400 flex items-center space-x-2",children:[(0,b.jsx)(d.default,{href:"/",className:"hover:text-white transition",children:"Home"}),(0,b.jsx)("span",{children:"/"}),(0,b.jsx)("span",{className:"text-white",children:"Developers"})]}),(0,b.jsx)("div",{className:"border-b border-gray-800 pb-4 flex justify-between items-center",children:(0,b.jsxs)("div",{className:"flex items-center space-x-3",children:[(0,b.jsx)(e.Terminal,{className:"text-blue-500 h-8 w-8 animate-pulse"}),(0,b.jsxs)("div",{children:[(0,b.jsx)("h1",{className:"text-3xl font-bold tracking-tight text-white",children:"Developer Hub"}),(0,b.jsx)("p",{className:"text-gray-400 mt-1",children:"Integration guides, configs, and commands for building on Sovereign L1"})]})]})}),(0,b.jsxs)("div",{className:"grid grid-cols-1 lg:grid-cols-4 gap-6",children:[(0,b.jsxs)("div",{className:"space-y-2 lg:col-span-1",children:[(0,b.jsx)("h3",{className:"text-xs uppercase font-bold text-gray-500 px-3 mb-2 tracking-wider",children:"Integration Guides"}),(0,b.jsx)("button",{onClick:()=>s("wallet"),className:`w-full text-left px-3 py-2 rounded-xl text-sm font-medium transition ${"wallet"===r?"bg-blue-950 text-blue-400 border-l-2 border-blue-500":"text-gray-400 hover:bg-gray-900/50 hover:text-white"}`,children:"MetaMask / Wallets"}),(0,b.jsx)("button",{onClick:()=>s("hardhat"),className:`w-full text-left px-3 py-2 rounded-xl text-sm font-medium transition ${"hardhat"===r?"bg-blue-950 text-blue-400 border-l-2 border-blue-500":"text-gray-400 hover:bg-gray-900/50 hover:text-white"}`,children:"Hardhat & Foundry"}),(0,b.jsx)("button",{onClick:()=>s("jssdk"),className:`w-full text-left px-3 py-2 rounded-xl text-sm font-medium transition ${"jssdk"===r?"bg-blue-950 text-blue-400 border-l-2 border-blue-500":"text-gray-400 hover:bg-gray-900/50 hover:text-white"}`,children:"JS SDKs (ethers, viem)"}),(0,b.jsx)("button",{onClick:()=>s("cosmos"),className:`w-full text-left px-3 py-2 rounded-xl text-sm font-medium transition ${"cosmos"===r?"bg-blue-950 text-blue-400 border-l-2 border-blue-500":"text-gray-400 hover:bg-gray-900/50 hover:text-white"}`,children:"Cosmos & CosmWasm"}),(0,b.jsxs)("div",{className:"pt-4 border-t border-gray-900 mt-4 px-3 space-y-3",children:[(0,b.jsx)("h4",{className:"text-xs uppercase font-bold text-gray-500 tracking-wider",children:"Verification"}),(0,b.jsxs)(d.default,{href:"/verify",className:"flex items-center space-x-1 text-xs text-blue-500 hover:underline font-medium",children:[(0,b.jsx)("span",{children:"Verify contract source"}),(0,b.jsx)(l.ExternalLink,{className:"h-3 w-3"})]})]})]}),(0,b.jsx)("div",{className:"lg:col-span-2 space-y-6",children:(()=>{switch(r){case"wallet":return v;case"hardhat":return w;case"jssdk":return x;case"cosmos":return y}})().map((a,c)=>(0,b.jsxs)("div",{className:"bg-gray-950 border border-gray-900 rounded-xl p-6 space-y-3 shadow-lg",children:[(0,b.jsxs)("div",{children:[(0,b.jsxs)("h3",{className:"text-base font-bold text-white flex items-center space-x-2",children:[(0,b.jsx)(f.Code,{className:"h-4 w-4 text-blue-500"}),(0,b.jsx)("span",{children:a.title})]}),(0,b.jsx)("p",{className:"text-xs text-gray-500 mt-1",children:a.description})]}),(0,b.jsxs)("div",{className:"relative group",children:[(0,b.jsx)("button",{onClick:()=>{var b;return b=a.code,void(navigator.clipboard.writeText(b),u(c),setTimeout(()=>u(null),2e3))},className:"absolute right-3 top-3 p-1.5 bg-gray-900/80 border border-gray-800 rounded-lg text-gray-400 hover:text-white hover:bg-gray-800 transition opacity-0 group-hover:opacity-100 focus:opacity-100",title:"Copy code",children:t===c?(0,b.jsx)(i.Check,{className:"h-3.5 w-3.5 text-green-500"}):(0,b.jsx)(h,{className:"h-3.5 w-3.5"})}),(0,b.jsx)("pre",{className:"font-mono text-xs text-gray-300 bg-black/40 border border-gray-900 rounded-lg p-4 overflow-x-auto leading-relaxed max-h-[350px]",children:(0,b.jsx)("code",{children:a.code})})]})]},c))}),(0,b.jsx)("div",{className:"space-y-4 lg:col-span-1",children:(0,b.jsxs)("div",{className:"bg-gray-950 border border-gray-900 rounded-xl p-5 shadow-lg space-y-4",children:[(0,b.jsxs)("h3",{className:"text-sm font-extrabold text-white flex items-center space-x-2 border-b border-gray-900 pb-2",children:[(0,b.jsx)(g.Settings,{className:"h-4 w-4 text-blue-500"}),(0,b.jsx)("span",{children:"Network Parameters"})]}),(0,b.jsxs)("div",{className:"space-y-3 text-xs",children:[(0,b.jsxs)("div",{children:[(0,b.jsx)("span",{className:"block text-gray-500 uppercase font-bold tracking-wider mb-0.5",children:"EVM Chain ID"}),(0,b.jsx)("span",{className:"font-mono font-semibold text-white",children:"7777 (0x1E61)"})]}),(0,b.jsxs)("div",{children:[(0,b.jsx)("span",{className:"block text-gray-500 uppercase font-bold tracking-wider mb-0.5",children:"Cosmos Chain ID"}),(0,b.jsx)("span",{className:"font-mono font-semibold text-white",children:"sovereign-1"})]}),(0,b.jsxs)("div",{children:[(0,b.jsx)("span",{className:"block text-gray-500 uppercase font-bold tracking-wider mb-0.5",children:"EVM RPC (HTTP)"}),(0,b.jsx)("span",{className:"font-mono font-semibold text-gray-300 break-all select-all",children:"http://localhost:8545"})]}),(0,b.jsxs)("div",{children:[(0,b.jsx)("span",{className:"block text-gray-500 uppercase font-bold tracking-wider mb-0.5",children:"Cosmos RPC (CometBFT)"}),(0,b.jsx)("span",{className:"font-mono font-semibold text-gray-300 break-all select-all",children:"http://localhost:26657"})]}),(0,b.jsxs)("div",{children:[(0,b.jsx)("span",{className:"block text-gray-500 uppercase font-bold tracking-wider mb-0.5",children:"Cosmos gRPC"}),(0,b.jsx)("span",{className:"font-mono font-semibold text-gray-300 break-all select-all",children:"localhost:9090"})]}),(0,b.jsxs)("div",{children:[(0,b.jsx)("span",{className:"block text-gray-500 uppercase font-bold tracking-wider mb-0.5",children:"Native Token"}),(0,b.jsx)("span",{className:"font-semibold text-white",children:"SOV (uSLT)"})]})]}),(0,b.jsxs)("div",{className:"p-3 bg-blue-950/20 border border-blue-900/50 rounded-lg text-[11px] text-blue-400 flex items-start space-x-2 leading-relaxed",children:[(0,b.jsx)(j.Info,{className:"h-4 w-4 mt-0.5 flex-shrink-0"}),(0,b.jsx)("span",{children:"Local Devnet RPC endpoints. Use these configs in your scripts or configuration files to connect and sign."})]})]})})]}),(0,b.jsxs)("div",{className:"bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-xl space-y-6",children:[(0,b.jsxs)("h2",{className:"text-xl font-bold text-white flex items-center gap-2",children:[(0,b.jsx)(k.Cpu,{className:"text-purple-500 h-5 w-5 animate-pulse"}),"Sovereign Smart Contract Deployment & Verification Pipeline"]}),(0,b.jsx)("div",{className:"flex justify-center p-4 bg-black/40 rounded-xl border border-gray-900",children:(0,b.jsxs)("svg",{viewBox:"0 0 800 150",className:"w-full max-w-4xl text-xs font-mono",children:[(0,b.jsx)("defs",{children:(0,b.jsx)("marker",{id:"arrow",viewBox:"0 0 10 10",refX:"6",refY:"5",markerWidth:"6",markerHeight:"6",orient:"auto-start-reverse",children:(0,b.jsx)("path",{d:"M 0 2 L 8 5 L 0 8 z",fill:"#4b5563"})})}),(0,b.jsx)("rect",{x:"10",y:"30",width:"160",height:"70",rx:"10",fill:"#1e1b4b",stroke:"#3730a3",strokeWidth:"1.5"}),(0,b.jsx)("text",{x:"90",y:"60",fill:"#fff",fontWeight:"bold",textAnchor:"middle",children:"1. COMPILE"}),(0,b.jsx)("text",{x:"90",y:"80",fill:"#a5b4fc",fontSize:"10",textAnchor:"middle",children:"Solidity / Wasm bytecode"}),(0,b.jsx)("line",{x1:"170",y1:"65",x2:"210",y2:"65",stroke:"#4b5563",strokeWidth:"2",markerEnd:"url(#arrow)"}),(0,b.jsx)("rect",{x:"220",y:"30",width:"160",height:"70",rx:"10",fill:"#14532d",stroke:"#166534",strokeWidth:"1.5"}),(0,b.jsx)("text",{x:"300",y:"60",fill:"#fff",fontWeight:"bold",textAnchor:"middle",children:"2. AUTHENTICATE"}),(0,b.jsx)("text",{x:"300",y:"80",fill:"#86efac",fontSize:"10",textAnchor:"middle",children:"Connect MetaMask/Keplr"}),(0,b.jsx)("line",{x1:"380",y1:"65",x2:"420",y2:"65",stroke:"#4b5563",strokeWidth:"2",markerEnd:"url(#arrow)"}),(0,b.jsx)("rect",{x:"430",y:"30",width:"160",height:"70",rx:"10",fill:"#701a75",stroke:"#86198f",strokeWidth:"1.5"}),(0,b.jsx)("text",{x:"510",y:"60",fill:"#fff",fontWeight:"bold",textAnchor:"middle",children:"3. BROADCAST"}),(0,b.jsx)("text",{x:"510",y:"80",fill:"#f5d0fe",fontSize:"10",textAnchor:"middle",children:"Sign & transmit to RPC"}),(0,b.jsx)("line",{x1:"590",y1:"65",x2:"630",y2:"65",stroke:"#4b5563",strokeWidth:"2",markerEnd:"url(#arrow)"}),(0,b.jsx)("rect",{x:"640",y:"30",width:"150",height:"70",rx:"10",fill:"#065f46",stroke:"#047857",strokeWidth:"1.5"}),(0,b.jsx)("text",{x:"715",y:"60",fill:"#fff",fontWeight:"bold",textAnchor:"middle",children:"4. VERIFY"}),(0,b.jsx)("text",{x:"715",y:"80",fill:"#a7f3d0",fontSize:"10",textAnchor:"middle",children:"Sourcify code upload"})]})})]})]})}],54426)}];

//# sourceMappingURL=_0k0_jay._.js.map