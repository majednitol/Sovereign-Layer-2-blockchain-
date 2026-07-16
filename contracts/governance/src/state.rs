use serde::{Deserialize, Serialize};
use schemars::JsonSchema;
use cosmwasm_std::{Addr, CosmosMsg};
use cw_storage_plus::{Item, Map};

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Config {
    pub constitution_address: Addr,
    pub treasury_address: Addr,
    pub reserve_fund_address: Addr,
    pub proposers: Vec<Addr>,
    pub approval_threshold: u64,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum ProposalStatus {
    Pending,
    Approved,
    Executed,
    Rejected,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Proposal {
    pub id: u64,
    pub title: String,
    pub description: String,
    pub actions: Vec<CosmosMsg>,
    pub status: ProposalStatus,
    pub approvals: Vec<Addr>,
}

use crate::msg::ProposalLog;
pub const CONFIG: Item<Config> = Item::new("config");
pub const LOG_COUNT: Item<u64> = Item::new("log_count");
pub const AUDIT_LOGS: Map<u64, ProposalLog> = Map::new("audit_logs");
pub const PROPOSALS: Map<u64, Proposal> = Map::new("proposals");
pub const PROPOSAL_COUNT: Item<u64> = Item::new("proposal_count");
