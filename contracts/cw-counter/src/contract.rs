#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
use cosmwasm_std::{to_json_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response, StdResult};
use crate::error::ContractError;
use crate::msg::{CountResponse, ExecuteMsg, InstantiateMsg, QueryMsg, SummaryResponse};
use crate::state::{Config, CONFIG};

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    let config = Config {
        count: msg.initial_count,
        owner: info.sender,
        label: msg.label,
        paused: false,
    };
    CONFIG.save(deps.storage, &config)?;

    Ok(Response::new()
        .add_attribute("method", "instantiate")
        .add_attribute("owner", config.owner.to_string())
        .add_attribute("initial_count", msg.initial_count.to_string()))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    match msg {
        ExecuteMsg::Increment {} => try_increment(deps, info),
        ExecuteMsg::Decrement {} => try_decrement(deps, info),
        ExecuteMsg::SetLabel { label } => try_set_label(deps, info, label),
        ExecuteMsg::Pause {} => try_pause(deps, info),
        ExecuteMsg::Unpause {} => try_unpause(deps, info),
    }
}

pub fn try_increment(deps: DepsMut, _info: MessageInfo) -> Result<Response, ContractError> {
    let mut config = CONFIG.load(deps.storage)?;
    if config.paused {
        return Err(ContractError::ContractPaused {});
    }
    config.count += 1;
    CONFIG.save(deps.storage, &config)?;

    Ok(Response::new()
        .add_attribute("action", "increment")
        .add_attribute("new_count", config.count.to_string()))
}

pub fn try_decrement(deps: DepsMut, _info: MessageInfo) -> Result<Response, ContractError> {
    let mut config = CONFIG.load(deps.storage)?;
    if config.paused {
        return Err(ContractError::ContractPaused {});
    }
    if config.count == 0 {
        return Err(ContractError::Underflow {});
    }
    config.count -= 1;
    CONFIG.save(deps.storage, &config)?;

    Ok(Response::new()
        .add_attribute("action", "decrement")
        .add_attribute("new_count", config.count.to_string()))
}

pub fn try_set_label(
    deps: DepsMut,
    info: MessageInfo,
    label: String,
) -> Result<Response, ContractError> {
    let mut config = CONFIG.load(deps.storage)?;
    if info.sender != config.owner {
        return Err(ContractError::Unauthorized {});
    }
    config.label = label.clone();
    CONFIG.save(deps.storage, &config)?;

    Ok(Response::new()
        .add_attribute("action", "set_label")
        .add_attribute("new_label", label))
}

pub fn try_pause(deps: DepsMut, info: MessageInfo) -> Result<Response, ContractError> {
    let mut config = CONFIG.load(deps.storage)?;
    if info.sender != config.owner {
        return Err(ContractError::Unauthorized {});
    }
    config.paused = true;
    CONFIG.save(deps.storage, &config)?;

    Ok(Response::new().add_attribute("action", "pause"))
}

pub fn try_unpause(deps: DepsMut, info: MessageInfo) -> Result<Response, ContractError> {
    let mut config = CONFIG.load(deps.storage)?;
    if info.sender != config.owner {
        return Err(ContractError::Unauthorized {});
    }
    config.paused = false;
    CONFIG.save(deps.storage, &config)?;

    Ok(Response::new().add_attribute("action", "unpause"))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::GetCount {} => to_json_binary(&query_count(deps)?),
        QueryMsg::GetSummary {} => to_json_binary(&query_summary(deps)?),
    }
}

fn query_count(deps: Deps) -> StdResult<CountResponse> {
    let config = CONFIG.load(deps.storage)?;
    Ok(CountResponse { count: config.count })
}

fn query_summary(deps: Deps) -> StdResult<SummaryResponse> {
    let config = CONFIG.load(deps.storage)?;
    Ok(SummaryResponse {
        count: config.count,
        owner: config.owner.to_string(),
        label: config.label,
        paused: config.paused,
    })
}
