const { ethers } = require("ethers");
const fs = require("fs");
const path = require("path");

const EVM_RPC_URL = "http://localhost:8545";
const PRIVATE_KEY = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80";

async function main() {
  const provider = new ethers.JsonRpcProvider(EVM_RPC_URL);
  const wallet = new ethers.Wallet(PRIVATE_KEY, provider);

  const balance = await provider.getBalance(wallet.address);
  console.log(`Address: ${wallet.address}, Balance: ${ethers.formatEther(balance)} ETH`);

  const artifactPath = path.join(__dirname, "Counter.artifact.json");
  const artifact = JSON.parse(fs.readFileSync(artifactPath, "utf8"));
  const { abi, bytecode } = artifact;

  const factory = new ethers.ContractFactory(abi, bytecode, wallet);
  
  console.log("Estimating gas...");
  try {
    const deployTx = await factory.getDeployTransaction(42, "SovereignTestCounter");
    const gasLimit = await provider.estimateGas(deployTx);
    console.log(`Estimated gas: ${gasLimit.toString()}`);

    console.log("Populating transaction...");
    const tx = await wallet.populateTransaction(deployTx);
    console.log("Populated transaction details:", {
      to: tx.to,
      nonce: tx.nonce,
      gasLimit: tx.gasLimit.toString(),
      gasPrice: tx.gasPrice ? tx.gasPrice.toString() : null,
      maxFeePerGas: tx.maxFeePerGas ? tx.maxFeePerGas.toString() : null,
      maxPriorityFeePerGas: tx.maxPriorityFeePerGas ? tx.maxPriorityFeePerGas.toString() : null,
      chainId: tx.chainId.toString(),
    });

    console.log("Signing transaction...");
    const signedTx = await wallet.signTransaction(tx);
    console.log("Sending raw transaction...");
    const txHash = await provider.send("eth_sendRawTransaction", [signedTx]);
    console.log(`Transaction sent successfully! Hash: ${txHash}`);

    console.log("Waiting for receipt...");
    const receipt = await provider.waitForTransaction(txHash, 1, 10000); // 10s timeout
    console.log("Receipt received:", receipt);
  } catch (err) {
    console.error("❌ Error encountered:", err);
  }
}

main();
