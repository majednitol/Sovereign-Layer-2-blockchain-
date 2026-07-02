const { ethers } = require("ethers");

const EVM_RPC_URL = "http://localhost:8545";

async function main() {
  const provider = new ethers.JsonRpcProvider(EVM_RPC_URL);
  const latestBlock = await provider.getBlockNumber();
  console.log(`Latest block: ${latestBlock}`);

  for (let i = 1; i <= latestBlock; i++) {
    const block = await provider.getBlock(i, true);
    if (block && block.prefetchedTransactions && block.prefetchedTransactions.length > 0) {
      console.log(`Block ${i} has ${block.prefetchedTransactions.length} transactions:`);
      for (const tx of block.prefetchedTransactions) {
        console.log(`  - Hash: ${tx.hash}, Sender: ${tx.from}, Recipient: ${tx.to}, Nonce: ${tx.nonce}`);
      }
    } else if (block && block.transactions && block.transactions.length > 0) {
      console.log(`Block ${i} has ${block.transactions.length} transactions:`);
      for (const txHash of block.transactions) {
        const tx = await provider.getTransaction(txHash);
        console.log(`  - Hash: ${tx.hash}, Sender: ${tx.from}, Recipient: ${tx.to}, Nonce: ${tx.nonce}`);
      }
    }
  }
  console.log("Done checking blocks.");
}

main();
