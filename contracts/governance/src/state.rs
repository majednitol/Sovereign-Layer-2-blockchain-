use serde::{Deserialize, Serialize};
use schemars::JsonSchema;
use cosmwasm_std::Addr;
use cw_storage_plus::{Item, Map};
use crate::msg::ProposalLog;

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Config {
    pub constitution_address: Addr,
    pub treasury_address: Addr,
    pub reserve_fund_address: Addr,
}

pub const CONFIG: Item<Config> = Item::new("config");
pub const LOG_COUNT: Item<u64> = Item::new("log_count");
pub const AUDIT_LOGS: Map<u64, ProposalLog> = Map::new("audit_logs");
