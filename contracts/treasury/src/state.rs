use serde::{Deserialize, Serialize};
use schemars::JsonSchema;
use cosmwasm_std::Addr;
use cw_storage_plus::Item;

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Config {
    pub governance_address: Option<Addr>,
    pub cold_multisig_address: Addr,
    pub is_paused: bool,
    pub reentrancy_lock: bool,
}

pub const CONFIG: Item<Config> = Item::new("config");
