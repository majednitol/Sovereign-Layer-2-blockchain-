// Test script to verify the integrity of EVM contract verification
// Usage: node test_verification_integrity.js

const fs = require("fs");
const path = require("path");

const API_BASE = process.env.API_BASE || "http://localhost:8082";

async function runTest() {
  console.log("════════════════════════════════════════════════════════════");
  console.log("  EVM Verification Integrity Test");
  console.log("════════════════════════════════════════════════════════════");

  // 1. Read deployment details
  const deployPath = path.join(__dirname, "Counter.deploy.json");
  const artifactPath = path.join(__dirname, "Counter.artifact.json");
  const sourcePath = path.join(__dirname, "Counter.sol");

  if (!fs.existsSync(deployPath) || !fs.existsSync(artifactPath) || !fs.existsSync(sourcePath)) {
    console.error("❌ Missing required files. Run compile_and_deploy.js first.");
    process.exit(1);
  }

  const deployInfo = JSON.parse(fs.readFileSync(deployPath, "utf8"));
  const artifact = JSON.parse(fs.readFileSync(artifactPath, "utf8"));
  const correctSource = fs.readFileSync(sourcePath, "utf8");

  const address = deployInfo.address;
  const ctorArgs = deployInfo.constructorArgs;
  const bytecode = artifact.deployedBytecode;
  const abi = artifact.abi;

  console.log(`Testing with contract address: ${address}`);

  // 2. Test Case 1: Verification with invalid/mismatched contract logic
  console.log("\n⏳ Test Case 1: Verification with mismatched contract logic...");
  const invalidSource = correctSource.replace("count += 1", "count += 2");
  
  const payload1 = {
    address: address,
    sourceCode: invalidSource,
    abi: abi,
    compilerVersion: "v0.8.24+commit.e11b9ed9",
    optimizerEnabled: true,
    optimizerRuns: 200,
    constructorArgs: ctorArgs,
    compiledBytecode: bytecode
  };

  try {
    const response = await fetch(`${API_BASE}/api/rest/v1/explorer/verify/evm`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload1)
    });

    if (response.status === 400) {
      const body = await response.text();
      console.log(`✅ Passed: Mismatched source code was correctly rejected.`);
      console.log(`   Response status: 400, Error: ${body.trim()}`);
    } else {
      console.error(`❌ Failed: Mismatched source code was NOT rejected. Status: ${response.status}`);
      const body = await response.text();
      console.error(`   Response: ${body}`);
    }
  } catch (err) {
    console.error("❌ Failed to perform request:", err);
  }

  // 3. Test Case 2: Verification with correct source code
  console.log("\n⏳ Test Case 2: Verification with correct source code...");
  const payload2 = {
    address: address,
    sourceCode: correctSource,
    abi: abi,
    compilerVersion: "v0.8.24+commit.e11b9ed9",
    optimizerEnabled: true,
    optimizerRuns: 200,
    constructorArgs: ctorArgs,
    compiledBytecode: bytecode
  };

  try {
    const response = await fetch(`${API_BASE}/api/rest/v1/explorer/verify/evm`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload2)
    });

    if (response.ok) {
      const result = await response.json();
      if (result.success) {
        console.log(`✅ Passed: Correct source code verified successfully! Match type: ${result.matchType}`);
      } else {
        console.error(`❌ Failed: Server responded with success=false`, result);
      }
    } else {
      const body = await response.text();
      console.error(`❌ Failed: Correct source code was rejected. Status: ${response.status}`);
      console.error(`   Response: ${body}`);
    }
  } catch (err) {
    console.error("❌ Failed to perform request:", err);
  }
}

runTest();
