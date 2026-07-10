#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
use cosmwasm_std::{
    to_json_binary, BankMsg, Binary, Coin, Deps, DepsMut, Env, MessageInfo, Response, StdError,
    StdResult,
};
use crate::msg::{ConfigResponse, ExecuteMsg, InstantiateMsg, QueryMsg};
use crate::state::{Config, CONFIG};

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> StdResult<Response> {
    let config = Config {
        governance_address: None,
        cold_multisig_address: deps.api.addr_validate(&msg.cold_multisig_address)?,
        is_paused: false,
        reentrancy_lock: false,
    };
    CONFIG.save(deps.storage, &config)?;
    Ok(Response::new()
        .add_attribute("action", "instantiate")
        .add_attribute("cold_multisig", msg.cold_multisig_address))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, StdError> {
    let mut config = CONFIG.load(deps.storage)?;

    // 1. One-time setup of governance address
    if let ExecuteMsg::SetupGovernanceAddress { address } = msg {
        if config.governance_address.is_some() {
            return Err(StdError::generic_err("Governance address is already setup"));
        }
        let gov_addr = deps.api.addr_validate(&address)?;
        config.governance_address = Some(gov_addr.clone());
        CONFIG.save(deps.storage, &config)?;
        return Ok(Response::new()
            .add_attribute("action", "setup_governance")
            .add_attribute("governance_address", gov_addr));
    }

    // 2. Load and validate standard permissions
    let gov_addr = config.governance_address.clone().ok_or_else(|| {
        StdError::generic_err("Governance address not set. SetupGovernanceAddress must be called first.")
    })?;

    match msg {
        ExecuteMsg::SetupGovernanceAddress { .. } => unreachable!(),
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

            // Release reentrancy guard before finishing execution
            config.reentrancy_lock = false;
            CONFIG.save(deps.storage, &config)?;

            Ok(Response::new()
                .add_message(send_msg)
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
            config.governance_address = Some(new_addr);
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
                governance_address: config.governance_address.map(|a| a.into_string()),
                cold_multisig_address: config.cold_multisig_address.into_string(),
                is_paused: config.is_paused,
            })
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use cosmwasm_std::testing::{mock_dependencies, mock_dependencies_with_balance, mock_env, mock_info};
    use cosmwasm_std::{from_json, Uint128};

    #[test]
    fn test_initialization_and_governance_setup() {
        let mut deps = mock_dependencies();

        let instantiate_msg = InstantiateMsg {
            cold_multisig_address: "multisig_addr".to_string(),
        };

        // Instantiate
        instantiate(deps.as_mut(), mock_env(), mock_info("creator", &[]), instantiate_msg).unwrap();

        // Setup governance
        let setup_msg = ExecuteMsg::SetupGovernanceAddress { address: "governance_addr".to_string() };
        execute(deps.as_mut(), mock_env(), mock_info("any", &[]), setup_msg).unwrap();

        // Check config
        let query_bin = query(deps.as_ref(), mock_env(), QueryMsg::GetConfig {}).unwrap();
        let config_res: ConfigResponse = from_json(&query_bin).unwrap();
        assert_eq!(config_res.governance_address.unwrap(), "governance_addr");
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
        };
        instantiate(deps.as_mut(), mock_env(), mock_info("creator", &[]), instantiate_msg).unwrap();
        execute(
            deps.as_mut(),
            mock_env(),
            mock_info("any", &[]),
            ExecuteMsg::SetupGovernanceAddress { address: "governance_addr".to_string() },
        ).unwrap();

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
        assert_eq!(config_res.governance_address.unwrap(), "new_governance_addr");
    }
}
