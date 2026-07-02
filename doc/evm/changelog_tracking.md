# Weekly Changelog Tracking Process for `cosmos/evm`

This document defines the weekly tracking process for upstream updates, breaking changes, and dependency tracking of pre-v1 releases of the `github.com/cosmos/evm` repository.

## Overview
Because `cosmos/evm` is a pre-v1 (`v0.x`) dependency undergoing active development, new minor and patch releases may introduce API updates, transaction structure refactors, or state machine alterations. This tracking process ensures the Sovereign L1 blockchain maintains compatibility.

## Weekly Verification Checklist
Every week, a validator developer or automated agent must execute the following protocol:

1. **Check Upstream Releases**:
   - Query github releases: `gh release list --repo cosmos/evm --limit 5`
   - Review release logs for EIP additions, changes to precompiles, or state-transition updates.

2. **Verify Version Alignment in `go.mod`**:
   - Inspect the current pinned version: `grep "github.com/cosmos/evm" chain/go.mod`
   - Compare with the latest published version on GitHub: `gh release list --repo cosmos/evm --limit 5`

3. **Validate Invariants & Upgrades**:
   - Run compilation checks: `go build ./...`
   - Run simulation suite: `go test -v ./app -run TestAppSimulation -NumBlocks 5000`
   - Verify E2E compatibility: `go test -v ./e2e/...`

4. **Document Upstream Changes**:
   - Record updates, deprecated features, and needed upgrades in `doc/evm/upstream_updates_log.md`.
