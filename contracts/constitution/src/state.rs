use serde::{Deserialize, Serialize};
use schemars::JsonSchema;
use cosmwasm_std::Addr;
use cw_storage_plus::Item;

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Config {
    pub governance_address: Addr,
    pub cold_multisig_address: Addr,
    pub rules: String,
    pub is_paused: bool,
}

pub const CONFIG: Item<Config> = Item::new("config");
