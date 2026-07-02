use serde::{Deserialize, Serialize};
use schemars::JsonSchema;
use cosmwasm_std::CosmosMsg;

use cosmwasm_schema::QueryResponses;

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct InstantiateMsg {
    pub constitution_address: String,
    pub treasury_address: String,
    pub reserve_fund_address: String,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum ExecuteMsg {
    SubmitProposal {
        title: String,
        description: String,
        actions: Vec<CosmosMsg>,
    },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema, QueryResponses)]
#[serde(rename_all = "snake_case")]
pub enum QueryMsg {
    #[returns(ConfigResponse)]
    GetConfig {},
    #[returns(AuditLogsResponse)]
    GetAuditLogs {},
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct ConfigResponse {
    pub constitution_address: String,
    pub treasury_address: String,
    pub reserve_fund_address: String,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct ProposalLog {
    pub id: u64,
    pub title: String,
    pub description: String,
    pub passed: bool,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct AuditLogsResponse {
    pub logs: Vec<ProposalLog>,
}
