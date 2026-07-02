// Deep Test Suite for EVM Verification Integrity
// Covers all 5 requested test cases to ensure no invalid data can bypass verification.
// Usage: node test_verification_deep.js

const fs = require("fs");
const path = require("path");

const API_BASE = process.env.API_BASE || "http://localhost:8082";

async function runTests() {
  console.log("============================================================");
  console.log("  EVM Verification Deep Test Suite");
  console.log("============================================================");

  // Load deployment details
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

  console.log(`Target Contract Address: ${address}`);

  // Helper to sleep
  const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

  // Helper to send POST request to verify endpoint
  async function verify(payload, testName) {
    // Wait to avoid rate limiting (1 req/sec)
    await sleep(1200);
    try {
      const response = await fetch(`${API_BASE}/api/rest/v1/explorer/verify/evm`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      const body = await response.text();
      if (response.status === 400) {
        console.log(`✅ [PASS] ${testName}: Correctly rejected (400). Error: ${body.trim()}`);
        return true;
      } else if (response.ok) {
        console.error(`❌ [FAIL] ${testName}: Unexpectedly accepted (200)! Response: ${body}`);
        return false;
      } else {
        console.log(`⚠️ [WARN] ${testName}: Returned status ${response.status}. Response: ${body}`);
        return false;
      }
    } catch (err) {
      console.error(`❌ [ERROR] ${testName}: Request failed:`, err.message);
      return false;
    }
  }

  // --- TEST CASE 1: Use another smart contract's source code ---
  const anotherSource = `
    pragma solidity ^0.8.20;
    contract SimpleStore {
        uint256 public value;
        function set(uint256 _value) public { value = _value; }
    }
  `;
  const payload1 = {
    address: address,
    sourceCode: anotherSource,
    abi: abi,
    compilerVersion: "v0.8.24+commit.e11b9ed9",
    optimizerEnabled: true,
    optimizerRuns: 200,
    constructorArgs: ctorArgs,
    compiledBytecode: bytecode
  };
  await verify(payload1, "Test Case 1: Mismatched Smart Contract");

  // --- TEST CASE 2: Modify the smart contract logic ---
  const modifiedSource = correctSource.replace("count += 1", "count += 5");
  const payload2 = {
    address: address,
    sourceCode: modifiedSource,
    abi: abi,
    compilerVersion: "v0.8.24+commit.e11b9ed9",
    optimizerEnabled: true,
    optimizerRuns: 200,
    constructorArgs: ctorArgs,
    compiledBytecode: bytecode
  };
  await verify(payload2, "Test Case 2: Modified Smart Contract Logic");

  // --- TEST CASE 3: Change the ABI and/or the compiled bytecode ---
  const payload3 = {
    address: address,
    sourceCode: correctSource,
    abi: [{ "inputs": [], "name": "fakeFunction", "outputs": [], "stateMutability": "nonpayable", "type": "function" }],
    compilerVersion: "v0.8.24+commit.e11b9ed9",
    optimizerEnabled: true,
    optimizerRuns: 200,
    constructorArgs: ctorArgs,
    compiledBytecode: "0xdeadbeef" // Mismatched bytecode
  };
  // Wait, if the client sends correct source, the server compiles it. If compiled matches on-chain,
  // the server uses the compiled ABI, NOT the client-provided ABI/bytecode!
  // Let's verify that a mismatched client-provided bytecode/ABI is ignored as long as the source matches.
  // Actually, if clientCompiledBytecode doesn't match, does the server still compile and verify?
  // Yes, because the server compiles the SOURCE code, which produces the correct bytecode matching on-chain.
  // So this test case should succeed (since the source code is correct and produces the correct bytecode),
  // but the final stored ABI in DB must be the CORRECT compiled ABI, not the "fakeFunction" ABI.
  console.log("\n⏳ Running Test Case 3: Changed ABI/Bytecode in payload...");
  await sleep(1200);
  try {
    const response = await fetch(`${API_BASE}/api/rest/v1/explorer/verify/evm`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload3),
    });
    if (response.ok) {
      // Fetch verified contract from DB to check if ABI was overwritten by the correct compiled ABI
      const checkResp = await fetch(`${API_BASE}/api/rest/v1/explorer/evm/contracts/${address}`);
      const contractDetails = await checkResp.json();
      const hasFakeFunction = JSON.stringify(contractDetails.abi).includes("fakeFunction");
      if (!hasFakeFunction) {
        console.log("✅ [PASS] Test Case 3: Correctly ignored client-side fake ABI and saved the real compiled ABI.");
      } else {
        console.error("❌ [FAIL] Test Case 3: Saved the fake client-side ABI into the database!");
      }
    } else {
      console.error(`❌ [FAIL] Test Case 3: Failed with status ${response.status}`);
    }
  } catch (err) {
    console.error("❌ [ERROR] Test Case 3 failed:", err.message);
  }

  // --- TEST CASE 4: Change the constructor arguments (invalid/mismatched) ---
  const payload4 = {
    address: address,
    sourceCode: correctSource,
    abi: abi,
    compilerVersion: "v0.8.24+commit.e11b9ed9",
    optimizerEnabled: true,
    optimizerRuns: 200,
    constructorArgs: "0x0000000000000000000000000000000000000000000000000000000000000000", // Mismatched / too short (32 bytes instead of 96)
    compiledBytecode: bytecode
  };
  await verify(payload4, "Test Case 4: Mismatched Constructor Arguments");

  // --- TEST CASE 5: Change the JSON Artifact ---
  // In the UI, pasting a modified JSON artifact auto-populates the ABI and Source.
  // If the user modifies the JSON artifact to have wrong source/bytecode, the server-side compilation
  // will catch it. Let's send an invalid JSON artifact structure or mismatch to the API.
  const payload5 = {
    address: address,
    sourceCode: "contract Invalid { }", // from modified artifact
    abi: abi,
    compilerVersion: "v0.8.24+commit.e11b9ed9",
    optimizerEnabled: true,
    optimizerRuns: 200,
    constructorArgs: ctorArgs,
    compiledBytecode: bytecode
  };
  await verify(payload5, "Test Case 5: Modified JSON Artifact Source");

  // --- TEST CASE 6: Valid verification (Happy Path) ---
  console.log("\n⏳ Running Test Case 6: Happy Path (Valid Inputs)...");
  await sleep(1200);
  const payload6 = {
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
      body: JSON.stringify(payload6),
    });
    if (response.ok) {
      const res = await response.json();
      console.log(`✅ [PASS] Happy Path: Verified successfully! Match type: ${res.matchType}`);
    } else {
      console.error(`❌ [FAIL] Happy Path: Rejected with status ${response.status}`);
    }
  } catch (err) {
    console.error("❌ [ERROR] Happy Path failed:", err.message);
  }
}

runTests();
