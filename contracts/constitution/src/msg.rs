use serde::{Deserialize, Serialize};
use schemars::JsonSchema;
use cosmwasm_schema::QueryResponses;

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct InstantiateMsg {
    pub rules: String,
    pub cold_multisig_address: String,
    pub governance_address: String,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum ExecuteMsg {
    UpdateConstitution { rules: String },
    EmergencyPause {},
    Unpause {},
    RotateColdMultisig { new_address: String },
    UpdateGovernanceAddress { new_address: String },
    CheckProposal { proposal_type: String, summary: String },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema, QueryResponses)]
#[serde(rename_all = "snake_case")]
pub enum QueryMsg {
    #[returns(ConstitutionResponse)]
    GetConstitution {},
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct ConstitutionResponse {
    pub rules: String,
    pub is_paused: bool,
    pub governance_address: String,
    pub cold_multisig_address: String,
}
