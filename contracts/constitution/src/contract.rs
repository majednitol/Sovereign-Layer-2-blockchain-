#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
use cosmwasm_std::{
    to_json_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response, StdError, StdResult,
};
use crate::msg::{ConstitutionResponse, ExecuteMsg, InstantiateMsg, QueryMsg, CheckProposalResponse};
use crate::state::{Config, CONFIG};

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
        rules: msg.rules,
        is_paused: false,
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
    _env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, StdError> {
    let mut config = CONFIG.load(deps.storage)?;
    let gov_addr = config.governance_address.clone();

    match msg {
        ExecuteMsg::UpdateConstitution { rules } => {
            if config.is_paused {
                return Err(StdError::generic_err("Contract is paused"));
            }
            if info.sender != gov_addr {
                return Err(StdError::generic_err("Unauthorized: Only governance can update constitution"));
            }
            config.rules = rules;
            CONFIG.save(deps.storage, &config)?;
            Ok(Response::new().add_attribute("action", "update_constitution"))
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
        ExecuteMsg::CheckProposal { proposal_type, summary } => {
            if config.is_paused {
                return Err(StdError::generic_err("Contract is paused"));
            }
            if summary.contains("VIOLATION") {
                return Err(StdError::generic_err("Proposal violates constitution"));
            }
            Ok(Response::new()
                .add_attribute("action", "check_proposal")
                .add_attribute("proposal_type", proposal_type))
        }
    }
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::GetConstitution {} => {
            let config = CONFIG.load(deps.storage)?;
            to_json_binary(&ConstitutionResponse {
                rules: config.rules,
                is_paused: config.is_paused,
                governance_address: config.governance_address.into_string(),
                cold_multisig_address: config.cold_multisig_address.into_string(),
            })
        }
        QueryMsg::CheckProposal { proposal_type: _, summary } => {
            let config = CONFIG.load(deps.storage)?;
            if config.is_paused {
                return to_json_binary(&CheckProposalResponse {
                    is_valid: false,
                    reason: "Contract is paused".to_string(),
                });
            }
            if summary.contains("VIOLATION") {
                return to_json_binary(&CheckProposalResponse {
                    is_valid: false,
                    reason: "Proposal violates constitution".to_string(),
                });
            }
            to_json_binary(&CheckProposalResponse {
                is_valid: true,
                reason: "".to_string(),
            })
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use cosmwasm_std::testing::{mock_dependencies, mock_env, mock_info};
    use cosmwasm_std::from_json;

    #[test]
    fn test_initialization() {
        let mut deps = mock_dependencies();

        let instantiate_msg = InstantiateMsg {
            rules: "My Constitution Rules".to_string(),
            cold_multisig_address: "multisig_addr".to_string(),
            governance_address: "governance_addr".to_string(),
        };

        // Instantiate
        let info = mock_info("creator", &[]);
        let res = instantiate(deps.as_mut(), mock_env(), info, instantiate_msg).unwrap();
        assert_eq!(res.attributes[0].value, "instantiate");

        // Try standard execute from random sender - should fail
        let info = mock_info("multisig_addr", &[]);
        let err = execute(
            deps.as_mut(),
            mock_env(),
            info,
            ExecuteMsg::UpdateConstitution { rules: "New rules".to_string() },
        ).unwrap_err();
        assert!(err.to_string().contains("Unauthorized: Only governance can update constitution"));
    }

    #[test]
    fn test_execution_pausing_and_bypass() {
        let mut deps = mock_dependencies();

        // Setup
        let instantiate_msg = InstantiateMsg {
            rules: "Rules 1".to_string(),
            cold_multisig_address: "multisig_addr".to_string(),
            governance_address: "governance_addr".to_string(),
        };
        instantiate(deps.as_mut(), mock_env(), mock_info("creator", &[]), instantiate_msg).unwrap();

        // Check proposal initially passes
        let check_res = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("any_caller", &[]),
            ExecuteMsg::CheckProposal {
                proposal_type: "test".to_string(),
                summary: "Clean summary".to_string(),
            },
        ).unwrap();
        assert_eq!(check_res.attributes[0].value, "check_proposal");

        // Pause via cold multi-sig
        let pause_res = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("multisig_addr", &[]),
            ExecuteMsg::EmergencyPause {},
        ).unwrap();
        assert_eq!(pause_res.attributes[0].value, "emergency_pause");

        // Check proposal now fails because it's paused
        let check_err = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("any_caller", &[]),
            ExecuteMsg::CheckProposal {
                proposal_type: "test".to_string(),
                summary: "Clean summary".to_string(),
            },
        ).unwrap_err();
        assert!(check_err.to_string().contains("Contract is paused"));

        // Query checks still succeed even when paused
        let query_bin = query(deps.as_ref(), mock_env(), QueryMsg::GetConstitution {}).unwrap();
        let query_res: ConstitutionResponse = from_json(&query_bin).unwrap();
        assert_eq!(query_res.rules, "Rules 1");
        assert!(query_res.is_paused);

        // Try updating constitution while paused (should fail)
        let update_err = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("governance_addr", &[]),
            ExecuteMsg::UpdateConstitution { rules: "Rules 2".to_string() },
        ).unwrap_err();
        assert!(update_err.to_string().contains("Contract is paused"));

        // Try unpausing via cold multisig (should fail: governance only)
        let unpause_err = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("multisig_addr", &[]),
            ExecuteMsg::Unpause {},
        ).unwrap_err();
        assert!(unpause_err.to_string().contains("Unauthorized: Only governance can unpause"));

        // Unpause via governance
        let unpause_res = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("governance_addr", &[]),
            ExecuteMsg::Unpause {},
        ).unwrap();
        assert_eq!(unpause_res.attributes[0].value, "unpause");

        // Now update constitution succeeds
        let update_res = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("governance_addr", &[]),
            ExecuteMsg::UpdateConstitution { rules: "Rules 2".to_string() },
        ).unwrap();
        assert_eq!(update_res.attributes[0].value, "update_constitution");

        // Rotate cold multisig
        let rotate_res = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("governance_addr", &[]),
            ExecuteMsg::RotateColdMultisig { new_address: "new_multisig_addr".to_string() },
        ).unwrap();
        assert_eq!(rotate_res.attributes[0].value, "rotate_cold_multisig");

        // Rotate governance address
        let rotate_gov_res = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("governance_addr", &[]),
            ExecuteMsg::UpdateGovernanceAddress { new_address: "new_governance_addr".to_string() },
        ).unwrap();
        assert_eq!(rotate_gov_res.attributes[0].value, "update_governance_address");

        // Verify updated governance address
        let query_bin = query(deps.as_ref(), mock_env(), QueryMsg::GetConstitution {}).unwrap();
        let query_res: ConstitutionResponse = from_json(&query_bin).unwrap();
        assert_eq!(query_res.governance_address, "new_governance_addr");
    }
}
