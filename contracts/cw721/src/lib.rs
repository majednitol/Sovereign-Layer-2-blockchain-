use cosmwasm_std::{entry_point, Binary, Deps, DepsMut, Env, MessageInfo, Response, StdResult};
use cw721_base::{ExecuteMsg, Extension, InstantiateMsg, QueryMsg};

type Cw721Contract<'a> = cw721_base::Cw721Contract<'a, Extension, cosmwasm_std::Empty, cosmwasm_std::Empty, cosmwasm_std::Empty>;

#[entry_point]
pub fn instantiate(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: InstantiateMsg,
) -> StdResult<Response> {
    Cw721Contract::default().instantiate(deps, env, info, msg)
}

#[entry_point]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg<Extension, cosmwasm_std::Empty>,
) -> Result<Response, cw721_base::ContractError> {
    Cw721Contract::default().execute(deps, env, info, msg)
}

#[entry_point]
pub fn query(
    deps: Deps,
    env: Env,
    msg: QueryMsg<cosmwasm_std::Empty>,
) -> StdResult<Binary> {
    Cw721Contract::default().query(deps, env, msg)
}
