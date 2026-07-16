use serde::{Deserialize, Serialize};
use schemars::JsonSchema;
use cosmwasm_std::Uint128;
use cosmwasm_schema::QueryResponses;

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct InstantiateMsg {
    pub cold_multisig_address: String,
    pub governance_address: String,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum ExecuteMsg {
    Withdraw { recipient: String, amount: Uint128, denom: String },
    EmergencyPause {},
    Unpause {},
    RotateColdMultisig { new_address: String },
    UpdateGovernanceAddress { new_address: String },
    MigrateBalance { new_address: String },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema, QueryResponses)]
#[serde(rename_all = "snake_case")]
pub enum QueryMsg {
    #[returns(ConfigResponse)]
    GetConfig {},
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct ConfigResponse {
    pub governance_address: String,
    pub cold_multisig_address: String,
    pub is_paused: bool,
}
