# EVM Layer & Precompile Stubs

This directory is reserved for custom Solidity contracts and EVM integration configurations.

## Planned Role
* **Solidity Contracts**: Housing for custom Solidity smart contracts deployed on the sovereign chain post-launch.
* **Precompile Stubs (Phase 2.9)**: Solidity interfaces and precompile contract stubs providing access to native Cosmos SDK modules from EVM Solidity smart contracts:
  - `x/oracle` Precompile: Access to current aggregated oracle prices.
  - `x/milestone` Precompile: Access to the state of milestone completions.
* **Tooling**: Standard EVM development environments (such as Foundry or Hardhat) will be integrated here for compilation and contract verification.
