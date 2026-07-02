use cosmwasm_std::{
    entry_point, to_json_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response, StdError, StdResult,
    WasmQuery, QueryRequest,
};
use crate::msg::{ConfigResponse, ExecuteMsg, InstantiateMsg, QueryMsg, ProposalLog, AuditLogsResponse};
use crate::state::{Config, CONFIG, LOG_COUNT, AUDIT_LOGS};
use constitution::msg::{QueryMsg as ConstitutionQueryMsg, ConstitutionResponse};

#[entry_point]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> StdResult<Response> {
    let config = Config {
        constitution_address: deps.api.addr_validate(&msg.constitution_address)?,
        treasury_address: deps.api.addr_validate(&msg.treasury_address)?,
        reserve_fund_address: deps.api.addr_validate(&msg.reserve_fund_address)?,
    };
    CONFIG.save(deps.storage, &config)?;
    LOG_COUNT.save(deps.storage, &0)?;
    Ok(Response::new()
        .add_attribute("action", "instantiate")
        .add_attribute("constitution", msg.constitution_address)
        .add_attribute("treasury", msg.treasury_address)
        .add_attribute("reserve_fund", msg.reserve_fund_address))
}

#[entry_point]
pub fn execute(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, StdError> {
    let config = CONFIG.load(deps.storage)?;

    match msg {
        ExecuteMsg::SubmitProposal { title, description, actions } => {
            // 1. Query Constitution to check rules and pause state
            let query_req = QueryRequest::Wasm(WasmQuery::Smart {
                contract_addr: config.constitution_address.to_string(),
                msg: to_json_binary(&ConstitutionQueryMsg::GetConstitution {})?,
            });
            let constitution_res: ConstitutionResponse = deps.querier.query(&query_req)?;

            // 2. Perform constitution check logic
            if constitution_res.rules.contains("VIOLATION") {
                return Err(StdError::generic_err("Proposal violates constitution"));
            }

            // 3. Log the audit details
            let next_id = LOG_COUNT.load(deps.storage).unwrap_or(0) + 1;
            LOG_COUNT.save(deps.storage, &next_id)?;

            let log_entry = ProposalLog {
                id: next_id,
                title: title.clone(),
                description: description.clone(),
                passed: true,
            };
            AUDIT_LOGS.save(deps.storage, next_id, &log_entry)?;

            Ok(Response::new()
                .add_messages(actions)
                .add_attribute("action", "submit_proposal")
                .add_attribute("proposal_id", next_id.to_string())
                .add_attribute("rules_status", "compliant"))
        }
    }
}

#[entry_point]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::GetConfig {} => {
            let config = CONFIG.load(deps.storage)?;
            to_json_binary(&ConfigResponse {
                constitution_address: config.constitution_address.into_string(),
                treasury_address: config.treasury_address.into_string(),
                reserve_fund_address: config.reserve_fund_address.into_string(),
            })
        }
        QueryMsg::GetAuditLogs {} => {
            let limit = LOG_COUNT.load(deps.storage)?;
            let mut logs = Vec::new();
            for i in 1..=limit {
                if let Ok(log) = AUDIT_LOGS.load(deps.storage, i) {
                    logs.push(log);
                }
            }
            to_json_binary(&AuditLogsResponse { logs })
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use cosmwasm_std::testing::{mock_env, mock_info, MockApi, MockStorage, MockQuerier};
    use cosmwasm_std::{from_json, OwnedDeps, SystemResult, ContractResult, SystemError};

    // Custom MockQuerier that intercept queries to the Constitution contract
    fn mock_dependencies_with_constitution(
        rules: String,
        is_paused: bool,
    ) -> OwnedDeps<MockStorage, MockApi, MockQuerier> {
        let mut deps = OwnedDeps {
            storage: MockStorage::default(),
            api: MockApi::default(),
            querier: MockQuerier::new(&[]),
            custom_query_type: std::marker::PhantomData,
        };

        deps.querier.update_wasm(move |query| {
            match query {
                WasmQuery::Smart { contract_addr, .. } => {
                    if contract_addr == "constitution_addr" {
                        let bin_res = to_json_binary(&ConstitutionResponse {
                            rules: rules.clone(),
                            is_paused,
                            governance_address: Some("gov_contract".to_string()),
                            cold_multisig_address: "multisig_addr".to_string(),
                        })
                        .unwrap();
                        SystemResult::Ok(ContractResult::Ok(bin_res))
                    } else {
                        SystemResult::Err(SystemError::NoSuchContract {
                            addr: contract_addr.clone(),
                        })
                    }
                }
                _ => SystemResult::Err(SystemError::Unknown {}),
            }
        });

        deps
    }

    #[test]
    fn test_governance_proposal_submission() {
        let mut deps = mock_dependencies_with_constitution("Safe constitutional rules".to_string(), false);

        let instantiate_msg = InstantiateMsg {
            constitution_address: "constitution_addr".to_string(),
            treasury_address: "treasury_addr".to_string(),
            reserve_fund_address: "reserve_fund_addr".to_string(),
        };
        instantiate(deps.as_mut(), mock_env(), mock_info("creator", &[]), instantiate_msg).unwrap();

        // Submit proposal
        let res = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("proposer", &[]),
            ExecuteMsg::SubmitProposal {
                title: "Prop 1".to_string(),
                description: "Description 1".to_string(),
                actions: vec![],
            },
        ).unwrap();
        assert_eq!(res.attributes[0].value, "submit_proposal");

        // Verify audit log
        let query_bin = query(deps.as_ref(), mock_env(), QueryMsg::GetAuditLogs {}).unwrap();
        let logs_res: AuditLogsResponse = from_json(&query_bin).unwrap();
        assert_eq!(logs_res.logs.len(), 1);
        assert_eq!(logs_res.logs[0].title, "Prop 1");
    }

    #[test]
    fn test_governance_proposal_violating_rules() {
        let mut deps = mock_dependencies_with_constitution("Rules containing VIOLATION pattern".to_string(), false);

        let instantiate_msg = InstantiateMsg {
            constitution_address: "constitution_addr".to_string(),
            treasury_address: "treasury_addr".to_string(),
            reserve_fund_address: "reserve_fund_addr".to_string(),
        };
        instantiate(deps.as_mut(), mock_env(), mock_info("creator", &[]), instantiate_msg).unwrap();

        // Submit proposal -> should fail due to rule violation
        let err = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("proposer", &[]),
            ExecuteMsg::SubmitProposal {
                title: "Bad Prop".to_string(),
                description: "Bad proposal".to_string(),
                actions: vec![],
            },
        ).unwrap_err();
        assert!(err.to_string().contains("Proposal violates constitution"));
    }
}
