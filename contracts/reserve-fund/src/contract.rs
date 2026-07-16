#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
use cosmwasm_std::{
    to_json_binary, BankMsg, Binary, Coin, Deps, DepsMut, Env, MessageInfo, Response, StdError,
    StdResult, QueryRequest, SubMsg, Reply,
};
use crate::msg::{ConfigResponse, ExecuteMsg, InstantiateMsg, QueryMsg, SovereignQuery, MilestoneResponse};
use crate::state::{Config, CONFIG};

const DISBURSE_REPLY_ID: u64 = 1;

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut<SovereignQuery>,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> StdResult<Response> {
    let config = Config {
        governance_address: deps.api.addr_validate(&msg.governance_address)?,
        cold_multisig_address: deps.api.addr_validate(&msg.cold_multisig_address)?,
        min_balance_threshold: msg.min_balance_threshold,
        is_paused: false,
        reentrancy_lock: false,
    };
    CONFIG.save(deps.storage, &config)?;
    Ok(Response::new()
        .add_attribute("action", "instantiate")
        .add_attribute("cold_multisig", msg.cold_multisig_address)
        .add_attribute("governance_address", msg.governance_address)
        .add_attribute("min_balance_threshold", msg.min_balance_threshold))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut<SovereignQuery>,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, StdError> {
    let mut config = CONFIG.load(deps.storage)?;
    let gov_addr = config.governance_address.clone();

    match msg {
        ExecuteMsg::DisburseMilestone { milestone_id, recipient, amount, denom } => {
            if config.is_paused {
                return Err(StdError::generic_err("Contract is paused"));
            }
            if info.sender != gov_addr {
                return Err(StdError::generic_err("Unauthorized: Only governance can call disburse"));
            }

            // Reentrancy guard
            if config.reentrancy_lock {
                return Err(StdError::generic_err("Reentrancy guard: Operation in progress"));
            }
            config.reentrancy_lock = true;
            CONFIG.save(deps.storage, &config)?;

            // Balance check (circuit-breaker at minimum balance threshold)
            let contract_balance = deps.querier.query_balance(env.contract.address, &denom)?;
            if contract_balance.amount.checked_sub(amount).unwrap_or_default() < config.min_balance_threshold {
                config.reentrancy_lock = false;
                CONFIG.save(deps.storage, &config)?;
                return Err(StdError::generic_err("Disbursement rejected: Contract balance falls below minimum threshold"));
            }

            // Gated on x/milestone state query
            let query_req = QueryRequest::Custom(SovereignQuery::Milestone { id: milestone_id.clone() });
            let milestone_res: MilestoneResponse = deps.querier.query(&query_req)?;

            if !milestone_res.is_achieved {
                config.reentrancy_lock = false;
                CONFIG.save(deps.storage, &config)?;
                return Err(StdError::generic_err("Disbursement rejected: Milestone is not achieved"));
            }

            // Perform transfer
            let send_msg = BankMsg::Send {
                to_address: deps.api.addr_validate(&recipient)?.to_string(),
                amount: vec![Coin { denom, amount }],
            };

            let sub_msg = SubMsg::reply_on_success(send_msg, DISBURSE_REPLY_ID);

            Ok(Response::new()
                .add_submessage(sub_msg)
                .add_attribute("action", "disburse_milestone")
                .add_attribute("milestone_id", milestone_id)
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

            // Query balance
            let balance = deps.querier.query_all_balances(env.contract.address)?;
            if balance.is_empty() {
                return Ok(Response::new().add_attribute("action", "migrate_balance").add_attribute("amount", "0"));
            }

            let send_msg = BankMsg::Send {
                to_address: new_addr.to_string(),
                amount: balance,
            };

            Ok(Response::new()
                .add_message(send_msg)
                .add_attribute("action", "migrate_balance")
                .add_attribute("destination", new_addr))
        }
    }
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps<SovereignQuery>, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::GetConfig {} => {
            let config = CONFIG.load(deps.storage)?;
            to_json_binary(&ConfigResponse {
                governance_address: config.governance_address.into_string(),
                cold_multisig_address: config.cold_multisig_address.into_string(),
                min_balance_threshold: config.min_balance_threshold,
                is_paused: config.is_paused,
            })
        }
    }
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn reply(
    deps: DepsMut<SovereignQuery>,
    _env: Env,
    msg: Reply,
) -> StdResult<Response> {
    if msg.id == DISBURSE_REPLY_ID {
        let mut config = CONFIG.load(deps.storage)?;
        config.reentrancy_lock = false;
        CONFIG.save(deps.storage, &config)?;
        Ok(Response::new().add_attribute("action", "reply_disburse_success"))
    } else {
        Err(StdError::generic_err("Unknown reply ID"))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use cosmwasm_std::testing::{mock_env, mock_info, MockApi, MockStorage, MockQuerier};
    use cosmwasm_std::{from_json, Coin, OwnedDeps, QuerierResult, SystemResult, Uint128};

    // Custom mock querier wrapper to handle custom queries
    fn mock_dependencies_with_custom_querier(
        contract_balance: &[Coin],
        milestone_achieved: bool,
    ) -> OwnedDeps<MockStorage, MockApi, MockQuerier<SovereignQuery>, SovereignQuery> {
        let custom_querier = MockQuerier::new(&[("cosmos2contract", contract_balance)])
            .with_custom_handler(move |query: &SovereignQuery| -> QuerierResult {
                match query {
                    SovereignQuery::Milestone { .. } => {
                        let res = MilestoneResponse { is_achieved: milestone_achieved };
                        SystemResult::Ok(to_json_binary(&res).into())
                    }
                }
            });

        OwnedDeps {
            storage: MockStorage::default(),
            api: MockApi::default(),
            querier: custom_querier,
            custom_query_type: std::marker::PhantomData,
        }
    }

    #[test]
    fn test_initialization() {
        let mut deps = mock_dependencies_with_custom_querier(&[], true);

        let instantiate_msg = InstantiateMsg {
            cold_multisig_address: "multisig_addr".to_string(),
            governance_address: "governance_addr".to_string(),
            min_balance_threshold: Uint128::new(20),
        };
        instantiate(deps.as_mut(), mock_env(), mock_info("creator", &[]), instantiate_msg).unwrap();

        // Check config
        let query_bin = query(deps.as_ref(), mock_env(), QueryMsg::GetConfig {}).unwrap();
        let config_res: ConfigResponse = from_json(&query_bin).unwrap();
        assert_eq!(config_res.governance_address, "governance_addr");
        assert_eq!(config_res.cold_multisig_address, "multisig_addr");
        assert_eq!(config_res.min_balance_threshold, Uint128::new(20));
        assert!(!config_res.is_paused);
    }

    #[test]
    fn test_milestone_disbursement_and_minimum_balance() {
        // Contract balance has 100 tokens, min balance threshold is 20 tokens
        let mut deps = mock_dependencies_with_custom_querier(
            &[Coin { denom: "ucsov".to_string(), amount: Uint128::new(100) }],
            true, // Milestone achieved
        );

        let instantiate_msg = InstantiateMsg {
            cold_multisig_address: "multisig_addr".to_string(),
            governance_address: "governance_addr".to_string(),
            min_balance_threshold: Uint128::new(20),
        };
        instantiate(deps.as_mut(), mock_env(), mock_info("creator", &[]), instantiate_msg).unwrap();

        // Disburse 30 tokens (100 - 30 = 70 > 20) -> Should succeed
        let res = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("governance_addr", &[]),
            ExecuteMsg::DisburseMilestone {
                milestone_id: "milestone_a".to_string(),
                recipient: "recipient_addr".to_string(),
                amount: Uint128::new(30),
                denom: "ucsov".to_string(),
            },
        ).unwrap();
        assert_eq!(res.attributes[0].value, "disburse_milestone");

        // Now run a second test setup where balance is 70 to test threshold failure
        let mut deps_fail = mock_dependencies_with_custom_querier(
            &[Coin { denom: "ucsov".to_string(), amount: Uint128::new(70) }],
            true,
        );
        let instantiate_msg = InstantiateMsg {
            cold_multisig_address: "multisig_addr".to_string(),
            governance_address: "governance_addr".to_string(),
            min_balance_threshold: Uint128::new(20),
        };
        instantiate(deps_fail.as_mut(), mock_env(), mock_info("creator", &[]), instantiate_msg).unwrap();

        // Disburse 60 tokens (70 - 60 = 10 < 20) -> Should fail due to min balance threshold
        let err = execute(
            deps_fail.as_mut(),
            mock_env(),
            mock_info("governance_addr", &[]),
            ExecuteMsg::DisburseMilestone {
                milestone_id: "milestone_b".to_string(),
                recipient: "recipient_addr".to_string(),
                amount: Uint128::new(60),
                denom: "ucsov".to_string(),
            },
        ).unwrap_err();
        assert!(err.to_string().contains("Contract balance falls below minimum threshold"));
    }

    #[test]
    fn test_milestone_unachieved_disbursement() {
        // Milestone is not achieved (false)
        let mut deps = mock_dependencies_with_custom_querier(
            &[Coin { denom: "ucsov".to_string(), amount: Uint128::new(100) }],
            false,
        );

        let instantiate_msg = InstantiateMsg {
            cold_multisig_address: "multisig_addr".to_string(),
            governance_address: "governance_addr".to_string(),
            min_balance_threshold: Uint128::new(20),
        };
        instantiate(deps.as_mut(), mock_env(), mock_info("creator", &[]), instantiate_msg).unwrap();

        // Disburse -> Should fail because milestone is not achieved
        let err = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("governance_addr", &[]),
            ExecuteMsg::DisburseMilestone {
                milestone_id: "milestone_c".to_string(),
                recipient: "recipient_addr".to_string(),
                amount: Uint128::new(30),
                denom: "ucsov".to_string(),
            },
        ).unwrap_err();
        assert!(err.to_string().contains("Milestone is not achieved"));

        // Rotate governance address
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
        let mut deps = mock_dependencies_with_custom_querier(
            &[Coin { denom: "ucsov".to_string(), amount: Uint128::new(100) }],
            true, // Milestone achieved
        );

        let instantiate_msg = InstantiateMsg {
            cold_multisig_address: "multisig_addr".to_string(),
            governance_address: "governance_addr".to_string(),
            min_balance_threshold: Uint128::new(20),
        };
        instantiate(deps.as_mut(), mock_env(), mock_info("creator", &[]), instantiate_msg).unwrap();

        // Disburse 30 tokens (100 - 30 = 70 > 20) -> Should succeed and lock
        execute(
            deps.as_mut(),
            mock_env(),
            mock_info("governance_addr", &[]),
            ExecuteMsg::DisburseMilestone {
                milestone_id: "milestone_a".to_string(),
                recipient: "recipient_addr".to_string(),
                amount: Uint128::new(30),
                denom: "ucsov".to_string(),
            },
        ).unwrap();

        // Verify reentrancy_lock is set to true
        let config = CONFIG.load(deps.as_ref().storage).unwrap();
        assert!(config.reentrancy_lock);

        // Try calling disburse again -> should fail with reentrancy error
        let err = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("governance_addr", &[]),
            ExecuteMsg::DisburseMilestone {
                milestone_id: "milestone_a".to_string(),
                recipient: "another_recipient".to_string(),
                amount: Uint128::new(10),
                denom: "ucsov".to_string(),
            },
        ).unwrap_err();
        assert!(err.to_string().contains("Reentrancy guard: Operation in progress"));

        // Simulate successful reply callback
        use cosmwasm_std::{SubMsgResponse, SubMsgResult};
        let reply_msg = Reply {
            id: DISBURSE_REPLY_ID,
            result: SubMsgResult::Ok(SubMsgResponse {
                events: vec![],
                data: None,
            }),
        };
        reply(deps.as_mut(), mock_env(), reply_msg).unwrap();

        // Verify reentrancy_lock is now false
        let config = CONFIG.load(deps.as_ref().storage).unwrap();
        assert!(!config.reentrancy_lock);
    }
}
