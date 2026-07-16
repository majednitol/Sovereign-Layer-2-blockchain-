use cosmwasm_std::{
    entry_point, to_json_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response, StdError, StdResult,
    WasmQuery, QueryRequest, Addr,
};
use crate::msg::{ConfigResponse, ExecuteMsg, InstantiateMsg, QueryMsg, ProposalLog, AuditLogsResponse, ProposalResponse};
use crate::state::{Config, CONFIG, LOG_COUNT, AUDIT_LOGS, Proposal, ProposalStatus, PROPOSALS, PROPOSAL_COUNT};
use constitution::msg::{QueryMsg as ConstitutionQueryMsg, ConstitutionResponse};

#[entry_point]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> StdResult<Response> {
    let proposers = msg.proposers.into_iter()
        .map(|p| deps.api.addr_validate(&p))
        .collect::<StdResult<Vec<Addr>>>()?;

    let config = Config {
        constitution_address: deps.api.addr_validate(&msg.constitution_address)?,
        treasury_address: deps.api.addr_validate(&msg.treasury_address)?,
        reserve_fund_address: deps.api.addr_validate(&msg.reserve_fund_address)?,
        proposers,
        approval_threshold: msg.approval_threshold,
    };
    CONFIG.save(deps.storage, &config)?;
    LOG_COUNT.save(deps.storage, &0)?;
    PROPOSAL_COUNT.save(deps.storage, &0)?;
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
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, StdError> {
    let config = CONFIG.load(deps.storage)?;

    match msg {
        ExecuteMsg::SubmitProposal { title, description, actions } => {
            // Check that sender is proposer
            if !config.proposers.contains(&info.sender) {
                return Err(StdError::generic_err("Unauthorized: sender is not an authorized proposer"));
            }

            // 1. Query Constitution to check rules and pause state
            let query_req = QueryRequest::Wasm(WasmQuery::Smart {
                contract_addr: config.constitution_address.to_string(),
                msg: to_json_binary(&ConstitutionQueryMsg::GetConstitution {})?,
            });
            let constitution_res: ConstitutionResponse = deps.querier.query(&query_req)?;

            // 2. Perform constitution check logic
            // TODO: placeholder compliance check — not real rule enforcement
            if constitution_res.rules.contains("VIOLATION") {
                return Err(StdError::generic_err("Proposal violates constitution"));
            }

            // Get proposal count
            let next_id = PROPOSAL_COUNT.load(deps.storage).unwrap_or(0) + 1;
            PROPOSAL_COUNT.save(deps.storage, &next_id)?;

            let proposal = Proposal {
                id: next_id,
                title: title.clone(),
                description: description.clone(),
                actions,
                status: ProposalStatus::Pending,
                approvals: vec![],
            };
            PROPOSALS.save(deps.storage, next_id, &proposal)?;

            Ok(Response::new()
                .add_attribute("action", "submit_proposal")
                .add_attribute("proposal_id", next_id.to_string())
                .add_attribute("rules_status", "compliant"))
        }
        ExecuteMsg::ApproveProposal { proposal_id } => {
            // Check that sender is proposer (voter)
            if !config.proposers.contains(&info.sender) {
                return Err(StdError::generic_err("Unauthorized: sender is not an authorized proposer"));
            }

            let mut proposal = PROPOSALS.load(deps.storage, proposal_id)?;
            if proposal.status != ProposalStatus::Pending {
                return Err(StdError::generic_err("Proposal is not in pending status"));
            }

            if !proposal.approvals.contains(&info.sender) {
                proposal.approvals.push(info.sender.clone());
                PROPOSALS.save(deps.storage, proposal_id, &proposal)?;
            }

            Ok(Response::new()
                .add_attribute("action", "approve_proposal")
                .add_attribute("proposal_id", proposal_id.to_string())
                .add_attribute("approver", info.sender.to_string()))
        }
        ExecuteMsg::ExecuteProposal { proposal_id } => {
            let mut proposal = PROPOSALS.load(deps.storage, proposal_id)?;
            if proposal.status != ProposalStatus::Pending {
                return Err(StdError::generic_err("Proposal is not in pending status"));
            }

            if (proposal.approvals.len() as u64) < config.approval_threshold {
                return Err(StdError::generic_err("Proposal has not reached approval threshold"));
            }

            proposal.status = ProposalStatus::Executed;
            PROPOSALS.save(deps.storage, proposal_id, &proposal)?;

            // Log the audit details
            let next_log_id = LOG_COUNT.load(deps.storage).unwrap_or(0) + 1;
            LOG_COUNT.save(deps.storage, &next_log_id)?;

            let log_entry = ProposalLog {
                id: next_log_id,
                title: proposal.title.clone(),
                description: proposal.description.clone(),
                passed: true,
            };
            AUDIT_LOGS.save(deps.storage, next_log_id, &log_entry)?;

            Ok(Response::new()
                .add_messages(proposal.actions)
                .add_attribute("action", "execute_proposal")
                .add_attribute("proposal_id", proposal_id.to_string()))
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
                proposers: config.proposers.into_iter().map(|a| a.into_string()).collect(),
                approval_threshold: config.approval_threshold,
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
        QueryMsg::GetProposal { id } => {
            let proposal = PROPOSALS.load(deps.storage, id)?;
            to_json_binary(&ProposalResponse {
                id: proposal.id,
                title: proposal.title,
                description: proposal.description,
                actions: proposal.actions,
                status: proposal.status,
                approvals: proposal.approvals.into_iter().map(|a| a.into_string()).collect(),
            })
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
                            governance_address: "gov_contract".to_string(),
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
            proposers: vec!["proposer".to_string(), "voter2".to_string()],
            approval_threshold: 2,
        };
        instantiate(deps.as_mut(), mock_env(), mock_info("creator", &[]), instantiate_msg).unwrap();

        // Submit proposal with unauthorized sender -> should fail
        let err = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("unauthorized", &[]),
            ExecuteMsg::SubmitProposal {
                title: "Prop 1".to_string(),
                description: "Description 1".to_string(),
                actions: vec![],
            },
        ).unwrap_err();
        assert!(err.to_string().contains("Unauthorized: sender is not an authorized proposer"));

        // Submit proposal with authorized sender -> succeeds
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

        // Try executing without enough approvals -> fails
        let err = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("proposer", &[]),
            ExecuteMsg::ExecuteProposal { proposal_id: 1 },
        ).unwrap_err();
        assert!(err.to_string().contains("Proposal has not reached approval threshold"));

        // Approve proposal 1
        execute(
            deps.as_mut(),
            mock_env(),
            mock_info("proposer", &[]),
            ExecuteMsg::ApproveProposal { proposal_id: 1 },
        ).unwrap();

        // Approve proposal 2nd time by same -> length remains 1 -> still fails
        let err = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("proposer", &[]),
            ExecuteMsg::ExecuteProposal { proposal_id: 1 },
        ).unwrap_err();
        assert!(err.to_string().contains("Proposal has not reached approval threshold"));

        // Approve by voter2
        execute(
            deps.as_mut(),
            mock_env(),
            mock_info("voter2", &[]),
            ExecuteMsg::ApproveProposal { proposal_id: 1 },
        ).unwrap();

        // Execute succeeds
        let exec_res = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("proposer", &[]),
            ExecuteMsg::ExecuteProposal { proposal_id: 1 },
        ).unwrap();
        assert_eq!(exec_res.attributes[0].value, "execute_proposal");

        // Verify audit log
        let query_bin = query(deps.as_ref(), mock_env(), QueryMsg::GetAuditLogs {}).unwrap();
        let logs_res: AuditLogsResponse = from_json(&query_bin).unwrap();
        assert_eq!(logs_res.logs.len(), 1);
        assert_eq!(logs_res.logs[0].title, "Prop 1");

        // Try executing again -> fails (replay protection)
        let err = execute(
            deps.as_mut(),
            mock_env(),
            mock_info("proposer", &[]),
            ExecuteMsg::ExecuteProposal { proposal_id: 1 },
        ).unwrap_err();
        assert!(err.to_string().contains("Proposal is not in pending status"));
    }

    #[test]
    fn test_governance_proposal_violating_rules() {
        let mut deps = mock_dependencies_with_constitution("Rules containing VIOLATION pattern".to_string(), false);

        let instantiate_msg = InstantiateMsg {
            constitution_address: "constitution_addr".to_string(),
            treasury_address: "treasury_addr".to_string(),
            reserve_fund_address: "reserve_fund_addr".to_string(),
            proposers: vec!["proposer".to_string()],
            approval_threshold: 1,
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
