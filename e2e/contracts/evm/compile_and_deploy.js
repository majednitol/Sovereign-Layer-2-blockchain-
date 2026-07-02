// Hardhat compilation and deployment helper for Counter.sol
// Usage: node compile_and_deploy.js
//
// Prerequisites:
//   npm install ethers@6 solc@0.8.24
//
// Environment Variables:
//   EVM_RPC_URL  - JSON-RPC endpoint (default: http://localhost:8545)
//   PRIVATE_KEY  - Deployer wallet private key

const solc = require("solc");
const { ethers } = require("ethers");
const fs = require("fs");
const path = require("path");

// ─── Configuration ───────────────────────────────────────────────────────────
const EVM_RPC_URL = process.env.EVM_RPC_URL || "http://localhost:8545";
const PRIVATE_KEY =
  process.env.PRIVATE_KEY ||
  "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"; // Hardhat default #0

// Constructor args for our Counter contract
const INITIAL_COUNT = 42;
const INITIAL_LABEL = "SovereignTestCounter";

// ─── Compile ─────────────────────────────────────────────────────────────────
function compile() {
  const sourcePath = path.join(__dirname, "Counter.sol");
  const source = fs.readFileSync(sourcePath, "utf8");

  const input = {
    language: "Solidity",
    sources: {
      "Counter.sol": { content: source },
    },
    settings: {
      optimizer: { enabled: true, runs: 200 },
      outputSelection: {
        "*": {
          "*": ["abi", "evm.bytecode", "evm.deployedBytecode"],
        },
      },
    },
  };

  console.log("⏳ Compiling Counter.sol with solc 0.8.24 (optimizer=200)...");
  const output = JSON.parse(solc.compile(JSON.stringify(input)));

  if (output.errors) {
    const fatals = output.errors.filter((e) => e.severity === "error");
    if (fatals.length > 0) {
      console.error("❌ Compilation errors:");
      fatals.forEach((e) => console.error(e.formattedMessage));
      process.exit(1);
    }
    // Print warnings
    output.errors
      .filter((e) => e.severity === "warning")
      .forEach((w) => console.warn("⚠️", w.formattedMessage));
  }

  const contract = output.contracts["Counter.sol"]["Counter"];
  const abi = contract.abi;
  const bytecode = "0x" + contract.evm.bytecode.object;
  const deployedBytecode = "0x" + contract.evm.deployedBytecode.object;

  console.log("✅ Compilation successful");
  console.log(`   ABI functions: ${abi.filter((a) => a.type === "function").length}`);
  console.log(`   Bytecode size: ${bytecode.length / 2 - 1} bytes`);

  // Write artifact for the verification step
  const artifact = {
    contractName: "Counter",
    sourceName: "Counter.sol",
    abi,
    bytecode,
    deployedBytecode,
    compiler: {
      version: "0.8.24",
      optimizer: { enabled: true, runs: 200 },
    },
  };

  const artifactPath = path.join(__dirname, "Counter.artifact.json");
  fs.writeFileSync(artifactPath, JSON.stringify(artifact, null, 2));
  console.log(`📄 Artifact written to: ${artifactPath}`);

  return { abi, bytecode, deployedBytecode };
}

// ─── Deploy ──────────────────────────────────────────────────────────────────
async function deploy(abi, bytecode) {
  console.log(`\n🌐 Connecting to EVM RPC: ${EVM_RPC_URL}`);
  const provider = new ethers.JsonRpcProvider(EVM_RPC_URL);
  const wallet = new ethers.Wallet(PRIVATE_KEY, provider);

  const network = await provider.getNetwork();
  const balance = await provider.getBalance(wallet.address);

  console.log(`   Chain ID: ${network.chainId}`);
  console.log(`   Deployer: ${wallet.address}`);
  console.log(`   Balance:  ${ethers.formatEther(balance)} ETH`);

  if (balance === 0n) {
    console.error("❌ Deployer has zero balance. Fund the account first.");
    process.exit(1);
  }

  console.log(`\n🚀 Deploying Counter(initialCount=${INITIAL_COUNT}, label="${INITIAL_LABEL}")...`);

  const factory = new ethers.ContractFactory(abi, bytecode, wallet);
  const contract = await factory.deploy(INITIAL_COUNT, INITIAL_LABEL);
  const receipt = await contract.deploymentTransaction().wait(1);

  const deployedAddress = await contract.getAddress();

  console.log("✅ Contract deployed!");
  console.log(`   Address:     ${deployedAddress}`);
  console.log(`   Tx Hash:     ${receipt.hash}`);
  console.log(`   Block:       ${receipt.blockNumber}`);
  console.log(`   Gas Used:    ${receipt.gasUsed.toString()}`);

  // Encode constructor args for verification
  const iface = new ethers.Interface(abi);
  const encodedArgs = iface.encodeDeploy([INITIAL_COUNT, INITIAL_LABEL]);
  console.log(`   Ctor Args:   ${encodedArgs}`);

  // Write deployment record
  const deployInfo = {
    address: deployedAddress,
    transactionHash: receipt.hash,
    blockNumber: receipt.blockNumber,
    gasUsed: receipt.gasUsed.toString(),
    constructorArgs: encodedArgs,
    constructorValues: {
      _initialCount: INITIAL_COUNT,
      _label: INITIAL_LABEL,
    },
    deployer: wallet.address,
    chainId: Number(network.chainId),
    timestamp: new Date().toISOString(),
  };

  const deployPath = path.join(__dirname, "Counter.deploy.json");
  fs.writeFileSync(deployPath, JSON.stringify(deployInfo, null, 2));
  console.log(`📄 Deploy info written to: ${deployPath}`);

  // Quick read test
  console.log("\n📖 Read test (count)...");
  const currentCount = await contract.count();
  console.log(`   count() = ${currentCount} (expected: ${INITIAL_COUNT})`);

  const summaryResult = await contract.summary();
  console.log(`   summary() = { count: ${summaryResult[0]}, owner: ${summaryResult[1]}, label: "${summaryResult[2]}", paused: ${summaryResult[3]} }`);

  // Quick write test
  console.log("\n✏️  Write test (increment)...");
  const tx = await contract.increment();
  const txReceipt = await tx.wait(1);
  const newCount = await contract.count();
  console.log(`   increment() → tx: ${txReceipt.hash}`);
  console.log(`   count() = ${newCount} (expected: ${INITIAL_COUNT + 1})`);

  console.log("\n" + "═".repeat(60));
  console.log("📋 NEXT STEPS FOR MANUAL VERIFICATION:");
  console.log("═".repeat(60));
  console.log(`1. Open explorer: http://localhost:3000/verify`);
  console.log(`2. Select "EVM (Solidity)" tab`);
  console.log(`3. Enter address: ${deployedAddress}`);
  console.log(`4. Paste source code from: Counter.sol`);
  console.log(`5. Paste artifact JSON from: Counter.artifact.json`);
  console.log(`6. Constructor Args: ${encodedArgs}`);
  console.log(`7. Click "Verify & Publish"`);
  console.log(`8. Navigate to: http://localhost:3000/evm/contracts/${deployedAddress}`);
  console.log(`9. Test read functions (count, owner, summary)`);
  console.log(`10. Test write functions (increment, decrement, setLabel)`);
  console.log("═".repeat(60));
}

// ─── Main ────────────────────────────────────────────────────────────────────
async function main() {
  console.log("═".repeat(60));
  console.log("  Sovereign L1 — EVM Contract Test Suite");
  console.log("═".repeat(60));

  const { abi, bytecode } = compile();
  await deploy(abi, bytecode);
}

main().catch((err) => {
  console.error("❌ Fatal error:", err);
  process.exit(1);
});
