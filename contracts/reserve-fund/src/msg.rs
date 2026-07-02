use serde::{Deserialize, Serialize};
use schemars::JsonSchema;
use cosmwasm_std::{Uint128, CustomQuery};

use cosmwasm_schema::QueryResponses;

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct InstantiateMsg {
    pub cold_multisig_address: String,
    pub min_balance_threshold: Uint128,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum ExecuteMsg {
    SetupGovernanceAddress { address: String },
    DisburseMilestone { milestone_id: String, recipient: String, amount: Uint128, denom: String },
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
    pub governance_address: Option<String>,
    pub cold_multisig_address: String,
    pub min_balance_threshold: Uint128,
    pub is_paused: bool,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum SovereignQuery {
    Milestone { id: String },
}

impl CustomQuery for SovereignQuery {}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct MilestoneResponse {
    pub is_achieved: bool,
}
