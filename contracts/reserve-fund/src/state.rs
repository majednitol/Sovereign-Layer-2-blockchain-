use serde::{Deserialize, Serialize};
use schemars::JsonSchema;
use cosmwasm_std::{Addr, Uint128};
use cw_storage_plus::Item;

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Config {
    pub governance_address: Addr,
    pub cold_multisig_address: Addr,
    pub min_balance_threshold: Uint128,
    pub is_paused: bool,
    pub reentrancy_lock: bool,
}

pub const CONFIG: Item<Config> = Item::new("config");
