package e2e

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Helper to run a command on the host
func runCmd(t *testing.T, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return stdout.String(), fmt.Errorf("command failed: %v\nstderr: %s\nstdout: %s", err, stderr.String(), stdout.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

// Helper to run a command inside the chain-node container
func dockerExec(t *testing.T, args ...string) (string, error) {
	fullArgs := append([]string{"exec", "chain-node"}, args...)
	return runCmd(t, "docker", fullArgs...)
}

func TestCosmWasmCounter(t *testing.T) {
	// Skip test if docker daemon is not available or if the chain-node container is not running
	if _, err := exec.Command("docker", "info").Output(); err != nil {
		t.Skip("Docker daemon is not running, skipping CosmWasm Counter integration test.")
	}
	containerStatus, err := exec.Command("docker", "ps", "--filter", "name=chain-node", "--filter", "status=running", "--quiet").Output()
	if err != nil || strings.TrimSpace(string(containerStatus)) == "" {
		t.Skip("Docker container 'chain-node' is not running, skipping CosmWasm Counter integration test.")
	}

	t.Log("Starting CosmWasm Counter Integration Test")

	// 1. Copy the Wasm file into the container
	t.Log("Copying cw_counter.wasm to chain-node container")
	_, err = runCmd(t, "docker", "cp", "contracts/cosmwasm/cw_counter.wasm", "chain-node:/tmp/cw_counter.wasm")
	if err != nil {
		t.Fatalf("Failed to copy Wasm binary to container: %v", err)
	}

	// 2. Store the Wasm contract on-chain
	t.Log("Storing Wasm contract on-chain")
	storeRes, err := dockerExec(t,
		"chaind", "tx", "wasm", "store", "/tmp/cw_counter.wasm",
		"--from", "validator",
		"--chain-id", "sovereign-1",
		"--gas-prices", "0aesov",
		"--gas", "auto",
		"--gas-adjustment", "1.3",
		"--keyring-backend", "test",
		"--home", "/root/.sovereign",
		"-y", "--output", "json",
	)
	if err != nil {
		t.Fatalf("Failed to store Wasm contract: %v", err)
	}

	t.Logf("Store Tx response: %s", storeRes)
	// Give the block time to commit
	time.Sleep(6 * time.Second)

	// Query latest Code ID
	t.Log("Querying list of codes to find the latest Code ID")
	listCodesRes, err := dockerExec(t, "chaind", "q", "wasm", "list-code", "--output", "json")
	if err != nil {
		t.Fatalf("Failed to query list of codes: %v", err)
	}

	var listCodes struct {
		CodeInfos []struct {
			CodeID string `json:"code_id"`
		} `json:"code_infos"`
	}
	if err := json.Unmarshal([]byte(listCodesRes), &listCodes); err != nil {
		t.Fatalf("Failed to parse list-code JSON: %v", err)
	}

	if len(listCodes.CodeInfos) == 0 {
		t.Fatal("No code infos found on-chain")
	}
	codeID := listCodes.CodeInfos[len(listCodes.CodeInfos)-1].CodeID
	t.Logf("Latest stored Code ID is: %s", codeID)

	// 3. Instantiate the contract
	t.Logf("Instantiating contract from Code ID %s", codeID)
	instMsg := `{"initial_count":10,"label":"TestCounter"}`
	instRes, err := dockerExec(t,
		"chaind", "tx", "wasm", "instantiate", codeID, instMsg,
		"--from", "validator",
		"--chain-id", "sovereign-1",
		"--label", "counter_instance",
		"--no-admin",
		"--gas-prices", "0aesov",
		"--gas", "auto",
		"--gas-adjustment", "1.3",
		"--keyring-backend", "test",
		"--home", "/root/.sovereign",
		"-y", "--output", "json",
	)
	if err != nil {
		t.Fatalf("Failed to instantiate contract: %v", err)
	}

	t.Logf("Instantiate Tx response: %s", instRes)
	time.Sleep(6 * time.Second)

	// Get contract address
	t.Logf("Querying contract address for Code ID %s", codeID)
	contractsRes, err := dockerExec(t, "chaind", "q", "wasm", "list-contract-by-code", codeID, "--output", "json")
	if err != nil {
		t.Fatalf("Failed to query contracts by code: %v", err)
	}

	var listContracts struct {
		Contracts []string `json:"contracts"`
	}
	if err := json.Unmarshal([]byte(contractsRes), &listContracts); err != nil {
		t.Fatalf("Failed to parse contracts JSON: %v", err)
	}

	if len(listContracts.Contracts) == 0 {
		t.Fatalf("No contracts found instantiated for Code ID %s", codeID)
	}
	contractAddr := listContracts.Contracts[0]
	t.Logf("Instantiated contract address: %s", contractAddr)

	// 4. Query initial state
	t.Log("Querying count state")
	countQueryRes, err := dockerExec(t, "chaind", "q", "wasm", "contract-state", "smart", contractAddr, `{"get_count":{}}`, "--output", "json")
	if err != nil {
		t.Fatalf("Failed to query count: %v", err)
	}
	t.Logf("Count query response: %s", countQueryRes)


	var countMap map[string]interface{}
	if err := json.Unmarshal([]byte(countQueryRes), &countMap); err != nil {
		t.Fatalf("Failed to parse count query response: %v", err)
	}
	dataObj, ok := countMap["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data object in count response, got: %s", countQueryRes)
	}
	countVal, ok := dataObj["count"].(float64)
	if !ok || countVal != 10 {
		t.Fatalf("Expected initial count to be 10, got: %v", dataObj["count"])
	}

	t.Log("Querying summary state")
	summaryQueryRes, err := dockerExec(t, "chaind", "q", "wasm", "contract-state", "smart", contractAddr, `{"get_summary":{}}`, "--output", "json")
	if err != nil {
		t.Fatalf("Failed to query summary: %v", err)
	}
	t.Logf("Summary query response: %s", summaryQueryRes)

	var summaryMap map[string]interface{}
	if err := json.Unmarshal([]byte(summaryQueryRes), &summaryMap); err != nil {
		t.Fatalf("Failed to parse summary query response: %v", err)
	}
	summaryData, ok := summaryMap["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data object in summary response, got: %s", summaryQueryRes)
	}
	if summaryData["label"] != "TestCounter" {
		t.Fatalf("Expected label to be 'TestCounter', got: %v", summaryData["label"])
	}
	if summaryData["paused"] != false {
		t.Fatalf("Expected paused to be false, got: %v", summaryData["paused"])
	}

	// 5. Execute Write: Increment
	t.Log("Executing Increment transaction")
	_, err = dockerExec(t,
		"chaind", "tx", "wasm", "execute", contractAddr, `{"increment":{}}`,
		"--from", "validator",
		"--chain-id", "sovereign-1",
		"--gas-prices", "0aesov",
		"--gas", "auto",
		"--gas-adjustment", "1.3",
		"--keyring-backend", "test",
		"--home", "/root/.sovereign",
		"-y", "--output", "json",
	)
	if err != nil {
		t.Fatalf("Failed to execute increment: %v", err)
	}
	time.Sleep(6 * time.Second)

	// Verify incremented count
	countQueryRes, err = dockerExec(t, "chaind", "q", "wasm", "contract-state", "smart", contractAddr, `{"get_count":{}}`, "--output", "json")
	if err != nil {
		t.Fatalf("Failed to query count after increment: %v", err)
	}
	json.Unmarshal([]byte(countQueryRes), &countMap)
	dataObj, ok = countMap["data"].(map[string]interface{})
	if !ok || dataObj["count"].(float64) != 11 {
		t.Fatalf("Expected count to be 11 after increment, got: %v", countMap)
	}
	t.Log("Increment successful! Count is now 11")

	// 6. Execute Write: Decrement
	t.Log("Executing Decrement transaction")
	_, err = dockerExec(t,
		"chaind", "tx", "wasm", "execute", contractAddr, `{"decrement":{}}`,
		"--from", "validator",
		"--chain-id", "sovereign-1",
		"--gas-prices", "0aesov",
		"--gas", "auto",
		"--gas-adjustment", "1.3",
		"--keyring-backend", "test",
		"--home", "/root/.sovereign",
		"-y", "--output", "json",
	)
	if err != nil {
		t.Fatalf("Failed to execute decrement: %v", err)
	}
	time.Sleep(6 * time.Second)

	// Verify decremented count
	countQueryRes, err = dockerExec(t, "chaind", "q", "wasm", "contract-state", "smart", contractAddr, `{"get_count":{}}`, "--output", "json")
	if err != nil {
		t.Fatalf("Failed to query count after decrement: %v", err)
	}
	json.Unmarshal([]byte(countQueryRes), &countMap)
	dataObj, ok = countMap["data"].(map[string]interface{})
	if !ok || dataObj["count"].(float64) != 10 {
		t.Fatalf("Expected count to be 10 after decrement, got: %v", countMap)
	}
	t.Log("Decrement successful! Count is now 10")

	// 7. Execute Write: SetLabel
	t.Log("Executing SetLabel transaction")
	_, err = dockerExec(t,
		"chaind", "tx", "wasm", "execute", contractAddr, `{"set_label":{"label":"UpdatedCounterLabel"}}`,
		"--from", "validator",
		"--chain-id", "sovereign-1",
		"--gas-prices", "0aesov",
		"--gas", "auto",
		"--gas-adjustment", "1.3",
		"--keyring-backend", "test",
		"--home", "/root/.sovereign",
		"-y", "--output", "json",
	)
	if err != nil {
		t.Fatalf("Failed to execute set_label: %v", err)
	}
	time.Sleep(6 * time.Second)

	// Verify updated label
	summaryQueryRes, err = dockerExec(t, "chaind", "q", "wasm", "contract-state", "smart", contractAddr, `{"get_summary":{}}`, "--output", "json")
	if err != nil {
		t.Fatalf("Failed to query summary after set_label: %v", err)
	}
	json.Unmarshal([]byte(summaryQueryRes), &summaryMap)
	summaryData, ok = summaryMap["data"].(map[string]interface{})
	if !ok || summaryData["label"] != "UpdatedCounterLabel" {
		t.Fatalf("Expected label to be 'UpdatedCounterLabel', got: %v", countMap)
	}
	t.Log("SetLabel successful! Label is now 'UpdatedCounterLabel'")

	// 8. Execute Write: Pause (reverts execution)
	t.Log("Executing Pause transaction")
	_, err = dockerExec(t,
		"chaind", "tx", "wasm", "execute", contractAddr, `{"pause":{}}`,
		"--from", "validator",
		"--chain-id", "sovereign-1",
		"--gas-prices", "0aesov",
		"--gas", "auto",
		"--gas-adjustment", "1.3",
		"--keyring-backend", "test",
		"--home", "/root/.sovereign",
		"-y", "--output", "json",
	)
	if err != nil {
		t.Fatalf("Failed to execute pause: %v", err)
	}
	time.Sleep(6 * time.Second)

	// Verify paused status
	summaryQueryRes, err = dockerExec(t, "chaind", "q", "wasm", "contract-state", "smart", contractAddr, `{"get_summary":{}}`, "--output", "json")
	json.Unmarshal([]byte(summaryQueryRes), &summaryMap)
	summaryData, ok = summaryMap["data"].(map[string]interface{})
	if !ok || summaryData["paused"] != true {
		t.Fatalf("Expected paused to be true, got: %v", countMap)
	}
	t.Log("Pause successful! Contract is now paused")

	// Try executing increment while paused (should fail)
	t.Log("Testing that Increment fails while contract is paused")
	failRes, err := dockerExec(t,
		"chaind", "tx", "wasm", "execute", contractAddr, `{"increment":{}}`,
		"--from", "validator",
		"--chain-id", "sovereign-1",
		"--gas-prices", "0aesov",
		"--gas", "auto",
		"--gas-adjustment", "1.3",
		"--keyring-backend", "test",
		"--home", "/root/.sovereign",
		"-y", "--output", "json",
	)
	if err == nil {
		t.Fatalf("Expected increment to fail while paused, but it succeeded: %s", failRes)
	}
	if !strings.Contains(err.Error(), "Contract is paused") {
		t.Fatalf("Expected pause error message, got: %v", err)
	}
	t.Log("Increment correctly failed with 'Contract is paused'")

	// 9. Execute Write: Unpause
	t.Log("Executing Unpause transaction")
	_, err = dockerExec(t,
		"chaind", "tx", "wasm", "execute", contractAddr, `{"unpause":{}}`,
		"--from", "validator",
		"--chain-id", "sovereign-1",
		"--gas-prices", "0aesov",
		"--gas", "auto",
		"--gas-adjustment", "1.3",
		"--keyring-backend", "test",
		"--home", "/root/.sovereign",
		"-y", "--output", "json",
	)
	if err != nil {
		t.Fatalf("Failed to execute unpause: %v", err)
	}
	time.Sleep(6 * time.Second)

	// Verify unpaused status
	summaryQueryRes, err = dockerExec(t, "chaind", "q", "wasm", "contract-state", "smart", contractAddr, `{"get_summary":{}}`, "--output", "json")
	json.Unmarshal([]byte(summaryQueryRes), &summaryMap)
	summaryData, ok = summaryMap["data"].(map[string]interface{})
	if !ok || summaryData["paused"] != false {
		t.Fatalf("Expected paused to be false, got: %v", countMap)
	}
	t.Log("Unpause successful! Contract is now unpaused")

	// Execute increment again (should succeed)
	t.Log("Executing Increment transaction again after unpausing")
	_, err = dockerExec(t,
		"chaind", "tx", "wasm", "execute", contractAddr, `{"increment":{}}`,
		"--from", "validator",
		"--chain-id", "sovereign-1",
		"--gas-prices", "0aesov",
		"--gas", "auto",
		"--gas-adjustment", "1.3",
		"--keyring-backend", "test",
		"--home", "/root/.sovereign",
		"-y", "--output", "json",
	)
	if err != nil {
		t.Fatalf("Failed to execute increment after unpausing: %v", err)
	}
	time.Sleep(6 * time.Second)

	// Verify incremented count is 11
	countQueryRes, err = dockerExec(t, "chaind", "q", "wasm", "contract-state", "smart", contractAddr, `{"get_count":{}}`, "--output", "json")
	json.Unmarshal([]byte(countQueryRes), &countMap)
	dataObj, ok = countMap["data"].(map[string]interface{})
	if !ok || dataObj["count"].(float64) != 11 {
		t.Fatalf("Expected count to be 11 after post-unpause increment, got: %v", countMap)
	}

	t.Log("COSMWASM COUNTER INTEGRATION TEST PASSED SUCCESSFULLY!")
}
