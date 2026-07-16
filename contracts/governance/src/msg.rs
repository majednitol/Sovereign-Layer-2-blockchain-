use serde::{Deserialize, Serialize};
use schemars::JsonSchema;
use cosmwasm_std::CosmosMsg;
use cosmwasm_schema::QueryResponses;
use crate::state::ProposalStatus;

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct InstantiateMsg {
    pub constitution_address: String,
    pub treasury_address: String,
    pub reserve_fund_address: String,
    pub proposers: Vec<String>,
    pub approval_threshold: u64,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum ExecuteMsg {
    SubmitProposal {
        title: String,
        description: String,
        actions: Vec<CosmosMsg>,
    },
    ApproveProposal {
        proposal_id: u64,
    },
    ExecuteProposal {
        proposal_id: u64,
    },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema, QueryResponses)]
#[serde(rename_all = "snake_case")]
pub enum QueryMsg {
    #[returns(ConfigResponse)]
    GetConfig {},
    #[returns(AuditLogsResponse)]
    GetAuditLogs {},
    #[returns(ProposalResponse)]
    GetProposal { id: u64 },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct ConfigResponse {
    pub constitution_address: String,
    pub treasury_address: String,
    pub reserve_fund_address: String,
    pub proposers: Vec<String>,
    pub approval_threshold: u64,
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

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct ProposalResponse {
    pub id: u64,
    pub title: String,
    pub description: String,
    pub actions: Vec<CosmosMsg>,
    pub status: ProposalStatus,
    pub approvals: Vec<String>,
}
