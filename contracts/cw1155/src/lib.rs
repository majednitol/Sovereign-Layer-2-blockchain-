use cosmwasm_std::{
    entry_point, to_json_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response, StdError,
    StdResult, Uint128,
};
use cw_storage_plus::Map;
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct InstantiateMsg {}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum ExecuteMsg {
    Mint {
        to: String,
        id: String,
        value: Uint128,
    },
    SendFrom {
        from: String,
        to: String,
        id: String,
        value: Uint128,
        msg: Option<Binary>,
    },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum QueryMsg {
    Balance { owner: String, id: String },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct BalanceResponse {
    pub balance: Uint128,
}

// Map: (owner, token_id) -> balance
pub const BALANCES: Map<(&str, &str), Uint128> = Map::new("balances");

#[entry_point]
pub fn instantiate(
    _deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    _msg: InstantiateMsg,
) -> StdResult<Response> {
    Ok(Response::default())
}

#[entry_point]
pub fn execute(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: ExecuteMsg,
) -> StdResult<Response> {
    match msg {
        ExecuteMsg::Mint { to, id, value } => {
            BALANCES.update(deps.storage, (&to, &id), |old| -> StdResult<_> {
                Ok(old.unwrap_or_default() + value)
            })?;
            Ok(Response::new().add_attribute("action", "mint"))
        }
        ExecuteMsg::SendFrom { from, to, id, value, .. } => {
            BALANCES.update(deps.storage, (&from, &id), |old| -> StdResult<_> {
                let bal = old.unwrap_or_default();
                if bal < value {
                    return Err(StdError::generic_err("Insufficient balance"));
                }
                Ok(bal - value)
            })?;
            BALANCES.update(deps.storage, (&to, &id), |old| -> StdResult<_> {
                Ok(old.unwrap_or_default() + value)
            })?;
            Ok(Response::new().add_attribute("action", "transfer"))
        }
    }
}

#[entry_point]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::Balance { owner, id } => {
            let balance = BALANCES.may_load(deps.storage, (&owner, &id))?.unwrap_or_default();
            to_json_binary(&BalanceResponse { balance })
        }
    }
}
