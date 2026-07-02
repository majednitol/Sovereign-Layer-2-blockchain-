use cosmwasm_schema::{cw_serde, QueryResponses};

#[cw_serde]
pub struct InstantiateMsg {
    pub initial_count: u64,
    pub label: String,
}

#[cw_serde]
pub enum ExecuteMsg {
    Increment {},
    Decrement {},
    SetLabel { label: String },
    Pause {},
    Unpause {},
}

#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    #[returns(CountResponse)]
    GetCount {},
    #[returns(SummaryResponse)]
    GetSummary {},
}

#[cw_serde]
pub struct CountResponse {
    pub count: u64,
}

#[cw_serde]
pub struct SummaryResponse {
    pub count: u64,
    pub owner: String,
    pub label: String,
    pub paused: bool,
}
