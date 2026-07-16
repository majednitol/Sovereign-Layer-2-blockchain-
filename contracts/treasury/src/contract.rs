#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
use cosmwasm_std::{
    to_json_binary, BankMsg, Binary, Coin, Deps, DepsMut, Env, MessageInfo, Response, StdError,
    StdResult, SubMsg, Reply,
};
use crate::msg::{ConfigResponse, ExecuteMsg, InstantiateMsg, QueryMsg};
use crate::state::{Config, CONFIG};

const WITHDRAW_REPLY_ID: u64 = 1;

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> StdResult<Response> {
    let config = Config {
        governance_address: deps.api.addr_validate(&msg.governance_address)?,
        cold_multisig_address: deps.api.addr_validate(&msg.cold_multisig_address)?,
        is_paused: false,
        reentrancy_lock: false,
    };
    CONFIG.save(deps.storage, &config)?;
    Ok(Response::new()
        .add_attribute("action", "instantiate")
        .add_attribute("cold_multisig", msg.cold_multisig_address)
        .add_attribute("governance_address", msg.governance_address))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, StdError> {
    let mut config = CONFIG.load(deps.storage)?;
    let gov_addr = config.governance_address.clone();

    match msg {
        ExecuteMsg::Withdraw { recipient, amount, denom } => {
            if config.is_paused {
                return Err(StdError::generic_err("Contract is paused"));
            }
            if info.sender != gov_addr {
                return Err(StdError::generic_err("Unauthorized: Only governance can withdraw"));
            }
            // Strict reentrancy guard
            if config.reentrancy_lock {
                return Err(StdError::generic_err("Reentrancy guard: Operation already in progress"));
            }
            config.reentrancy_lock = true;
            CONFIG.save(deps.storage, &config)?;

            let send_msg = BankMsg::Send {
                to_address: deps.api.addr_validate(&recipient)?.to_string(),
                amount: vec![Coin { denom, amount }],
            };

            let sub_msg = SubMsg::reply_on_success(send_msg, WITHDRAW_REPLY_ID);

            Ok(Response::new()
                .add_submessage(sub_msg)
                .add_attribute("action", "withdraw")
                .add_attribute("recipient", recipient)
                .add_attribute("amount", amount))
        }
        ExecuteMsg::EmergencyPause {} => {
            if info.sender != gov_addr && info.sender != config.cold_multisig_address {
                return Err(StdError::generic_err("Unauthorized: Only governance or cold multi-sig can pause"));
            }
            config.is_paused = true;
            CONFIG.save(deps.storage, &config)?;
            Ok(Response::new().add_attribute("action", "emergency_pause"))
        }
        ExecuteMsg::Unpause {} => {
            if info.sender != gov_addr {
                return Err(StdError::generic_err("Unauthorized: Only governance can unpause"));
            }
            config.is_paused = false;
            CONFIG.save(deps.storage, &config)?;
            Ok(Response::new().add_attribute("action", "unpause"))
        }
        ExecuteMsg::RotateColdMultisig { new_address } => {
            if info.sender != gov_addr {
                return Err(StdError::generic_err("Unauthorized: Only governance can rotate cold multi-sig"));
            }
            let new_addr = deps.api.addr_validate(&new_address)?;
            config.cold_multisig_address = new_addr;
            CONFIG.save(deps.storage, &config)?;
            Ok(Response::new().add_attribute("action", "rotate_cold_multisig"))
        }
        ExecuteMsg::UpdateGovernanceAddress { new_address } => {
            if info.sender != gov_addr && info.sender != config.cold_multisig_address {
                return Err(StdError::generic_err("Unauthorized: Only governance or cold multi-sig can update governance address"));
            }
            let new_addr = deps.api.addr_validate(&new_address)?;
            config.governance_address = new_addr;
            CONFIG.save(deps.storage, &config)?;
            Ok(Response::new().add_attribute("action", "update_governance_address"))
        }
        ExecuteMsg::MigrateBalance { new_address } => {
            if info.sender != gov_addr && info.sender != config.cold_multisig_address {
                return Err(StdError::generic_err("Unauthorized: Only governance or cold multi-sig can migrate balance"));
            }
            let new_addr = deps.api.addr_validate(&new_address)?;

            // Query all contract balance
            let balance = deps.querier.query_all_balances(env.contract.address)?;
            if balance.is_empty() {
                return Ok(Response::new().add_attribute("action", "migrate_balance").add_attribute("amount", "0"));
            }

            let send_msg = BankMsg::Send {
                to_address: new_addr.to_string(),
                amount: balance.clone(),
            };

            Ok(Response::new()
                .add_message(send_msg)
                .add_attribute("action", "migrate_balance")
                .add_attribute("destination", new_addr))
        }
    }
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::GetConfig {} => {
            let config = CONFIG.load(deps.storage)?;
            to_json_binary(&ConfigResponse {
                governance_address: config.governance_address.into_string(),
                cold_multisig_address: config.cold_multisig_address.into_string(),
                is_paused: config.is_paused,
            })
        }
    }
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn reply(deps: DepsMut, _env: Env, msg: Reply) -> StdResult<Response> {
    if msg.id == WITHDRAW_REPLY_ID {
        let mut config = CONFIG.load(deps.storage)?;
        config.reentrancy_lock = false;
        CONFIG.save(deps.storage, &config)?;
        Ok(Response::new().add_attribute("action", "reply_withdraw_success"))
    } else {
        Err(StdError::generic_err("Unknown reply ID"))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use cosmwasm_std::testing::{mock_dependencies, mock_dependencies_with_balance, mock_env, mock_info};
    use cosmwasm_std::{from_json, Uint128};

    #[test]
    fn test_initialization() {
        let mut deps = mock_dependencies();

        let instantiate_msg = InstantiateMsg {
            cold_multisig_address: "multisig_addr".to_string(),
            governance_address: "governance_addr".to_string(),
        };

        // Instantiate
        instantiate(deps.as_mut(), mock_env(), mock_info("creator", &[]), instantiate_msg).unwrap();

        // Check config
        let query_bin = query(deps.as_ref(), mock_env(), QueryMsg::GetConfig {}).unwrap();
        let config_res: ConfigResponse = from_json(&query_bin).unwrap();
        assert_eq!(config_res.governance_address, "governance_addr");
        assert_eq!(config_res.cold_multisig_address, "multisig_addr");
        assert!(!config_res.is_paused);
    }

    #[test]
    fn test_withdrawals_and_pausing() {
        let mut deps = mock_dependencies_with_balance(&[Coin {
            denom: "ucsov".to_string(),
            amount: Uint128::new(100000000),
        }]);

        // Setup
        let instantiate_msg = InstantiateMsg {
            cold_multisig_address: "multisig_addr".to_string(),
            governance_address: "governance_addr".to_string(),
        };
        instantiate(deps.as_mut(), mock_env(), mock_info("creator", &[]), instantiate_msg).unwrap();

        // Withdraw by non-governance should fail
        let withdraw_err = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("any_caller", &[]),
            ExecuteMsg::Withdraw {
                recipient: "recipient_addr".to_string(),
                amount: Uint128::new(5000000),
                denom: "ucsov".to_string(),
            },
        ).unwrap_err();
        assert!(withdraw_err.to_string().contains("Unauthorized: Only governance can withdraw"));

        // Withdraw by governance succeeds
        let withdraw_res = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("governance_addr", &[]),
            ExecuteMsg::Withdraw {
                recipient: "recipient_addr".to_string(),
                amount: Uint128::new(5000000),
                denom: "ucsov".to_string(),
            },
        ).unwrap();
        assert_eq!(withdraw_res.attributes[0].value, "withdraw");

        // Emergency Pause by cold multisig
        execute(
            deps.as_mut(),
            mock_env(),
            mock_info("multisig_addr", &[]),
            ExecuteMsg::EmergencyPause {},
        ).unwrap();

        // Withdraw by governance fails when paused
        let withdraw_err_paused = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("governance_addr", &[]),
            ExecuteMsg::Withdraw {
                recipient: "recipient_addr".to_string(),
                amount: Uint128::new(5000000),
                denom: "ucsov".to_string(),
            },
        ).unwrap_err();
        assert!(withdraw_err_paused.to_string().contains("Contract is paused"));

        // Migrate balance bypasses pause
        let migrate_res = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("multisig_addr", &[]),
            ExecuteMsg::MigrateBalance { new_address: "migration_addr".to_string() },
        ).unwrap();
        assert_eq!(migrate_res.attributes[0].value, "migrate_balance");

        // Rotate governance address (bypasses pause)
        let rotate_gov_res = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("governance_addr", &[]),
            ExecuteMsg::UpdateGovernanceAddress { new_address: "new_governance_addr".to_string() },
        ).unwrap();
        assert_eq!(rotate_gov_res.attributes[0].value, "update_governance_address");

        // Verify updated governance address
        let query_bin = query(deps.as_ref(), mock_env(), QueryMsg::GetConfig {}).unwrap();
        let config_res: ConfigResponse = from_json(&query_bin).unwrap();
        assert_eq!(config_res.governance_address, "new_governance_addr");
    }

    #[test]
    fn test_reentrancy_protection() {
        let mut deps = mock_dependencies_with_balance(&[Coin {
            denom: "ucsov".to_string(),
            amount: Uint128::new(100000000),
        }]);

        // Setup
        let instantiate_msg = InstantiateMsg {
            cold_multisig_address: "multisig_addr".to_string(),
            governance_address: "governance_addr".to_string(),
        };
        instantiate(deps.as_mut(), mock_env(), mock_info("creator", &[]), instantiate_msg).unwrap();

        // Withdraw by governance should succeed first time and lock it
        execute(
            deps.as_mut(),
            mock_env(),
            mock_info("governance_addr", &[]),
            ExecuteMsg::Withdraw {
                recipient: "recipient_addr".to_string(),
                amount: Uint128::new(5000000),
                denom: "ucsov".to_string(),
            },
        ).unwrap();

        // Verify reentrancy_lock is set to true
        let config = CONFIG.load(deps.as_ref().storage).unwrap();
        assert!(config.reentrancy_lock);

        // Try calling withdraw again -> should fail with reentrancy error
        let err = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("governance_addr", &[]),
            ExecuteMsg::Withdraw {
                recipient: "another_recipient".to_string(),
                amount: Uint128::new(5000000),
                denom: "ucsov".to_string(),
            },
        ).unwrap_err();
        assert!(err.to_string().contains("Reentrancy guard: Operation already in progress"));

        // Simulate successful reply callback
        use cosmwasm_std::{SubMsgResponse, SubMsgResult};
        let reply_msg = Reply {
            id: WITHDRAW_REPLY_ID,
            result: SubMsgResult::Ok(SubMsgResponse {
                events: vec![],
                data: None,
            }),
        };
        reply(deps.as_mut(), mock_env(), reply_msg).unwrap();

        // Verify reentrancy_lock is now false
        let config = CONFIG.load(deps.as_ref().storage).unwrap();
        assert!(!config.reentrancy_lock);

        // Try calling withdraw third time -> succeeds
        execute(
            deps.as_mut(),
            mock_env(),
            mock_info("governance_addr", &[]),
            ExecuteMsg::Withdraw {
                recipient: "recipient_addr".to_string(),
                amount: Uint128::new(5000000),
                denom: "ucsov".to_string(),
            },
        ).unwrap();
    }
}
