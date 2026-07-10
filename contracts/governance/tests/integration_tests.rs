use cosmwasm_std::{
    to_json_binary, Addr, Api, Binary, BlockInfo, Coin, CosmosMsg, CustomQuery, Empty,
    Querier, QuerierResult, QuerierWrapper, Storage, Uint128, Deps, DepsMut,
};
use cw_multi_test::{
    AppBuilder, AppResponse, CosmosRouter, Contract, ContractWrapper, Executor, Module,
};

use constitution::msg::{
    ConstitutionResponse, ExecuteMsg as ConstitutionExecuteMsg,
    InstantiateMsg as ConstitutionInstantiateMsg, QueryMsg as ConstitutionQueryMsg,
};
use governance::msg::{
    AuditLogsResponse, ExecuteMsg as GovernanceExecuteMsg,
    InstantiateMsg as GovernanceInstantiateMsg, QueryMsg as GovernanceQueryMsg,
};
use reserve_fund::msg::{
    ConfigResponse as ReserveConfigResponse, ExecuteMsg as ReserveExecuteMsg,
    InstantiateMsg as ReserveInstantiateMsg, QueryMsg as ReserveQueryMsg,
    SovereignQuery, MilestoneResponse,
};
use treasury::msg::{
    ConfigResponse as TreasuryConfigResponse, ExecuteMsg as TreasuryExecuteMsg,
    InstantiateMsg as TreasuryInstantiateMsg, QueryMsg as TreasuryQueryMsg,
};

struct QuerierForwarder<'a, Q: CustomQuery> {
    querier: &'a QuerierWrapper<'a, Q>,
}

impl<'a, Q: CustomQuery> Querier for QuerierForwarder<'a, Q> {
    fn raw_query(&self, bin_request: &[u8]) -> QuerierResult {
        self.querier.raw_query(bin_request)
    }
}

fn constitution_contract() -> Box<dyn Contract<Empty, SovereignQuery>> {
    let contract = ContractWrapper::new(
        |deps, env, info, msg| {
            let DepsMut { storage, api, querier } = deps;
            let forwarder = QuerierForwarder { querier: &querier };
            let empty_deps = DepsMut {
                storage,
                api,
                querier: QuerierWrapper::new(&forwarder),
            };
            constitution::contract::execute(empty_deps, env, info, msg)
        },
        |deps, env, info, msg| {
            let DepsMut { storage, api, querier } = deps;
            let forwarder = QuerierForwarder { querier: &querier };
            let empty_deps = DepsMut {
                storage,
                api,
                querier: QuerierWrapper::new(&forwarder),
            };
            constitution::contract::instantiate(empty_deps, env, info, msg)
        },
        |deps, env, msg| {
            let Deps { storage, api, querier } = deps;
            let forwarder = QuerierForwarder { querier: &querier };
            let empty_deps = Deps {
                storage,
                api,
                querier: QuerierWrapper::new(&forwarder),
            };
            constitution::contract::query(empty_deps, env, msg)
        },
    );
    Box::new(contract)
}

fn treasury_contract() -> Box<dyn Contract<Empty, SovereignQuery>> {
    let contract = ContractWrapper::new(
        |deps, env, info, msg| {
            let DepsMut { storage, api, querier } = deps;
            let forwarder = QuerierForwarder { querier: &querier };
            let empty_deps = DepsMut {
                storage,
                api,
                querier: QuerierWrapper::new(&forwarder),
            };
            treasury::contract::execute(empty_deps, env, info, msg)
        },
        |deps, env, info, msg| {
            let DepsMut { storage, api, querier } = deps;
            let forwarder = QuerierForwarder { querier: &querier };
            let empty_deps = DepsMut {
                storage,
                api,
                querier: QuerierWrapper::new(&forwarder),
            };
            treasury::contract::instantiate(empty_deps, env, info, msg)
        },
        |deps, env, msg| {
            let Deps { storage, api, querier } = deps;
            let forwarder = QuerierForwarder { querier: &querier };
            let empty_deps = Deps {
                storage,
                api,
                querier: QuerierWrapper::new(&forwarder),
            };
            treasury::contract::query(empty_deps, env, msg)
        },
    );
    Box::new(contract)
}

fn reserve_contract() -> Box<dyn Contract<Empty, SovereignQuery>> {
    let contract = ContractWrapper::new(
        reserve_fund::contract::execute,
        reserve_fund::contract::instantiate,
        reserve_fund::contract::query,
    );
    Box::new(contract)
}

fn governance_contract() -> Box<dyn Contract<Empty, SovereignQuery>> {
    let contract = ContractWrapper::new(
        |deps, env, info, msg| {
            let DepsMut { storage, api, querier } = deps;
            let forwarder = QuerierForwarder { querier: &querier };
            let empty_deps = DepsMut {
                storage,
                api,
                querier: QuerierWrapper::new(&forwarder),
            };
            governance::contract::execute(empty_deps, env, info, msg)
        },
        |deps, env, info, msg| {
            let DepsMut { storage, api, querier } = deps;
            let forwarder = QuerierForwarder { querier: &querier };
            let empty_deps = DepsMut {
                storage,
                api,
                querier: QuerierWrapper::new(&forwarder),
            };
            governance::contract::instantiate(empty_deps, env, info, msg)
        },
        |deps, env, msg| {
            let Deps { storage, api, querier } = deps;
            let forwarder = QuerierForwarder { querier: &querier };
            let empty_deps = Deps {
                storage,
                api,
                querier: QuerierWrapper::new(&forwarder),
            };
            governance::contract::query(empty_deps, env, msg)
        },
    );
    Box::new(contract)
}

struct CustomSovereignModule;

impl Module for CustomSovereignModule {
    type ExecT = Empty;
    type QueryT = SovereignQuery;
    type SudoT = Empty;

    fn execute<ExecC, QueryC>(
        &self,
        _api: &dyn Api,
        _storage: &mut dyn Storage,
        _router: &dyn CosmosRouter<ExecC = ExecC, QueryC = QueryC>,
        _block: &BlockInfo,
        _sender: Addr,
        _msg: Self::ExecT,
    ) -> Result<AppResponse, anyhow::Error> {
        Err(anyhow::anyhow!("Custom message not supported"))
    }

    fn query(
        &self,
        _api: &dyn Api,
        _storage: &dyn Storage,
        _querier: &dyn Querier,
        _block: &BlockInfo,
        request: Self::QueryT,
    ) -> Result<Binary, anyhow::Error> {
        match request {
            SovereignQuery::Milestone { id } => {
                let achieved = id == "achieved_milestone";
                Ok(to_json_binary(&MilestoneResponse { is_achieved: achieved })?)
            }
        }
    }

    fn sudo<ExecC, QueryC>(
        &self,
        _api: &dyn Api,
        _storage: &mut dyn Storage,
        _router: &dyn CosmosRouter<ExecC = ExecC, QueryC = QueryC>,
        _block: &BlockInfo,
        _msg: Self::SudoT,
    ) -> Result<AppResponse, anyhow::Error> {
        Err(anyhow::anyhow!("Sudo not supported"))
    }
}

#[test]
fn test_multi_contract_flow() {
    let cold_multisig = Addr::unchecked("cold_multisig_addr");
    let initial_treasury_balance = vec![Coin::new(1_000_000, "ucsov")];
    let initial_reserve_balance = vec![Coin::new(500_000, "ucsov")];

    let mut app = AppBuilder::new_custom()
        .with_custom(CustomSovereignModule)
        .build(|router, _api, storage| {
            // Initialize balances for the creator address
            router
                .bank
                .init_balance(
                    storage,
                    &Addr::unchecked("creator"),
                    vec![Coin::new(10_000_000, "ucsov")],
                )
                .unwrap();
        });

    let constitution_code_id = app.store_code(constitution_contract());
    let treasury_code_id = app.store_code(treasury_contract());
    let reserve_code_id = app.store_code(reserve_contract());
    let governance_code_id = app.store_code(governance_contract());

    // 1. Instantiate Constitution
    let constitution_addr = app
        .instantiate_contract(
            constitution_code_id,
            Addr::unchecked("creator"),
            &ConstitutionInstantiateMsg {
                rules: "Safe rules".to_string(),
                cold_multisig_address: cold_multisig.to_string(),
            },
            &[],
            "Constitution",
            None,
        )
        .unwrap();

    // 2. Instantiate Treasury
    let treasury_addr = app
        .instantiate_contract(
            treasury_code_id,
            Addr::unchecked("creator"),
            &TreasuryInstantiateMsg {
                cold_multisig_address: cold_multisig.to_string(),
            },
            &[],
            "Treasury",
            None,
        )
        .unwrap();

    // 3. Instantiate Reserve Fund
    let reserve_addr = app
        .instantiate_contract(
            reserve_code_id,
            Addr::unchecked("creator"),
            &ReserveInstantiateMsg {
                cold_multisig_address: cold_multisig.to_string(),
                min_balance_threshold: Uint128::new(100_000),
            },
            &[],
            "Reserve Fund",
            None,
        )
        .unwrap();

    // Send initial balances to treasury and reserve fund
    app.send_tokens(
        Addr::unchecked("creator"),
        treasury_addr.clone(),
        &initial_treasury_balance,
    )
    .unwrap();

    app.send_tokens(
        Addr::unchecked("creator"),
        reserve_addr.clone(),
        &initial_reserve_balance,
    )
    .unwrap();

    // 4. Instantiate Governance
    let governance_addr = app
        .instantiate_contract(
            governance_code_id,
            Addr::unchecked("creator"),
            &GovernanceInstantiateMsg {
                constitution_address: constitution_addr.to_string(),
                treasury_address: treasury_addr.to_string(),
                reserve_fund_address: reserve_addr.to_string(),
            },
            &[],
            "Governance",
            None,
        )
        .unwrap();

    // 5. Setup Governance addresses inside target contracts (one-time setup)
    app.execute_contract(
        Addr::unchecked("any_caller"),
        constitution_addr.clone(),
        &ConstitutionExecuteMsg::SetupGovernanceAddress {
            address: governance_addr.to_string(),
        },
        &[],
    )
    .unwrap();

    app.execute_contract(
        Addr::unchecked("any_caller"),
        treasury_addr.clone(),
        &TreasuryExecuteMsg::SetupGovernanceAddress {
            address: governance_addr.to_string(),
        },
        &[],
    )
    .unwrap();

    app.execute_contract(
        Addr::unchecked("any_caller"),
        reserve_addr.clone(),
        &ReserveExecuteMsg::SetupGovernanceAddress {
            address: governance_addr.to_string(),
        },
        &[],
    )
    .unwrap();

    // Second call to SetupGovernanceAddress must fail
    let err = app
        .execute_contract(
            Addr::unchecked("any_caller"),
            constitution_addr.clone(),
            &ConstitutionExecuteMsg::SetupGovernanceAddress {
                address: governance_addr.to_string(),
            },
            &[],
        )
        .unwrap_err();
    assert!(
        format!("{:#}", err).contains("already setup"),
        "Expected 'already setup' in error, got: {}",
        err
    );

    // --- CONSTITUTION TESTS ---
    // Update constitution rules by non-governance (should fail)
    let err = app
        .execute_contract(
            Addr::unchecked("any_caller"),
            constitution_addr.clone(),
            &ConstitutionExecuteMsg::UpdateConstitution {
                rules: "New Violating Rules".to_string(),
            },
            &[],
        )
        .unwrap_err();
    assert!(format!("{:#}", err).contains("Unauthorized"));

    // Pause via cold multi-sig
    app.execute_contract(
        cold_multisig.clone(),
        constitution_addr.clone(),
        &ConstitutionExecuteMsg::EmergencyPause {},
        &[],
    )
    .unwrap();

    // Update constitution when paused (should fail)
    let err = app
        .execute_contract(
            governance_addr.clone(),
            constitution_addr.clone(),
            &ConstitutionExecuteMsg::UpdateConstitution {
                rules: "New rules".to_string(),
            },
            &[],
        )
        .unwrap_err();
    assert!(format!("{:#}", err).contains("Contract is paused"));

    // Query rules when paused (should succeed)
    let config: ConstitutionResponse = app
        .wrap()
        .query_wasm_smart(constitution_addr.clone(), &ConstitutionQueryMsg::GetConstitution {})
        .unwrap();
    assert_eq!(config.rules, "Safe rules");
    assert!(config.is_paused);

    // Unpause via governance
    app.execute_contract(
        governance_addr.clone(),
        constitution_addr.clone(),
        &ConstitutionExecuteMsg::Unpause {},
        &[],
    )
    .unwrap();

    // --- TREASURY TESTS ---
    // Withdrawal by hacker (fails)
    let err = app
        .execute_contract(
            Addr::unchecked("hacker"),
            treasury_addr.clone(),
            &TreasuryExecuteMsg::Withdraw {
                recipient: "hacker".to_string(),
                amount: Uint128::new(100),
                denom: "ucsov".to_string(),
            },
            &[],
        )
        .unwrap_err();
    assert!(format!("{:#}", err).contains("Unauthorized"));

    // Pause treasury
    app.execute_contract(
        cold_multisig.clone(),
        treasury_addr.clone(),
        &TreasuryExecuteMsg::EmergencyPause {},
        &[],
    )
    .unwrap();

    // Withdrawal when paused (fails)
    let err = app
        .execute_contract(
            governance_addr.clone(),
            treasury_addr.clone(),
            &TreasuryExecuteMsg::Withdraw {
                recipient: "recipient".to_string(),
                amount: Uint128::new(100),
                denom: "ucsov".to_string(),
            },
            &[],
        )
        .unwrap_err();
    assert!(format!("{:#}", err).contains("Contract is paused"));

    // Unpause treasury
    app.execute_contract(
        governance_addr.clone(),
        treasury_addr.clone(),
        &TreasuryExecuteMsg::Unpause {},
        &[],
    )
    .unwrap();

    // --- RESERVE FUND TESTS ---
    // Disburse for unachieved milestone (fails)
    let err = app
        .execute_contract(
            governance_addr.clone(),
            reserve_addr.clone(),
            &ReserveExecuteMsg::DisburseMilestone {
                milestone_id: "unachieved_milestone".to_string(),
                recipient: "recipient".to_string(),
                amount: Uint128::new(50_000),
                denom: "ucsov".to_string(),
            },
            &[],
        )
        .unwrap_err();
    assert!(format!("{:#}", err).contains("Milestone is not achieved"));

    // Disburse exceeding min threshold (fails)
    // Initial balance: 500_000. Try to disburse 450_000 (remains 50_000 < 100_000)
    let err = app
        .execute_contract(
            governance_addr.clone(),
            reserve_addr.clone(),
            &ReserveExecuteMsg::DisburseMilestone {
                milestone_id: "achieved_milestone".to_string(),
                recipient: "recipient".to_string(),
                amount: Uint128::new(450_000),
                denom: "ucsov".to_string(),
            },
            &[],
        )
        .unwrap_err();
    assert!(format!("{:#}", err).contains("falls below minimum threshold"));

    // --- GOVERNANCE PROPOSALS & COMPLIANCE ---
    // Submit compliant proposal (succeeds)
    let actions: Vec<CosmosMsg<Empty>> = vec![];
    app.execute_contract(
        Addr::unchecked("proposer"),
        governance_addr.clone(),
        &GovernanceExecuteMsg::SubmitProposal {
            title: "Compliant Proposal".to_string(),
            description: "Some description".to_string(),
            actions,
        },
        &[],
    )
    .unwrap();

    // Verify audit log has 1 entry
    let audit_logs: AuditLogsResponse = app
        .wrap()
        .query_wasm_smart(governance_addr.clone(), &GovernanceQueryMsg::GetAuditLogs {})
        .unwrap();
    assert_eq!(audit_logs.logs.len(), 1);
    assert_eq!(audit_logs.logs[0].title, "Compliant Proposal");

    // Update constitution rules to contain "VIOLATION" pattern
    app.execute_contract(
        governance_addr.clone(),
        constitution_addr.clone(),
        &ConstitutionExecuteMsg::UpdateConstitution {
            rules: "Rules containing VIOLATION pattern".to_string(),
        },
        &[],
    )
    .unwrap();

    // Submit proposal violating rules (fails)
    let err = app
        .execute_contract(
            Addr::unchecked("proposer"),
            governance_addr.clone(),
            &GovernanceExecuteMsg::SubmitProposal {
                title: "Violating Proposal".to_string(),
                description: "This proposal violates constitution".to_string(),
                actions: vec![],
            },
            &[],
        )
        .unwrap_err();
    assert!(format!("{:#}", err).contains("Proposal violates constitution"));

    // --- CONTRACT REPLACEMENT PROCEDURE ---
    // Step A: Pause Treasury and Reserve Fund
    app.execute_contract(
        cold_multisig.clone(),
        treasury_addr.clone(),
        &TreasuryExecuteMsg::EmergencyPause {},
        &[],
    )
    .unwrap();
    app.execute_contract(
        cold_multisig.clone(),
        reserve_addr.clone(),
        &ReserveExecuteMsg::EmergencyPause {},
        &[],
    )
    .unwrap();

    // Step B: Instantiate new Governance contract
    let new_governance_addr = app
        .instantiate_contract(
            governance_code_id,
            Addr::unchecked("creator"),
            &GovernanceInstantiateMsg {
                constitution_address: constitution_addr.to_string(),
                treasury_address: treasury_addr.to_string(),
                reserve_fund_address: reserve_addr.to_string(),
            },
            &[],
            "New Governance",
            None,
        )
        .unwrap();

    // Step C: Update governance address on all 3 target contracts
    app.execute_contract(
        cold_multisig.clone(),
        constitution_addr.clone(),
        &ConstitutionExecuteMsg::UpdateGovernanceAddress {
            new_address: new_governance_addr.to_string(),
        },
        &[],
    )
    .unwrap();
    app.execute_contract(
        cold_multisig.clone(),
        treasury_addr.clone(),
        &TreasuryExecuteMsg::UpdateGovernanceAddress {
            new_address: new_governance_addr.to_string(),
        },
        &[],
    )
    .unwrap();
    app.execute_contract(
        cold_multisig.clone(),
        reserve_addr.clone(),
        &ReserveExecuteMsg::UpdateGovernanceAddress {
            new_address: new_governance_addr.to_string(),
        },
        &[],
    )
    .unwrap();

    // Step D: Unpause contracts using the new governance authority
    app.execute_contract(
        new_governance_addr.clone(),
        treasury_addr.clone(),
        &TreasuryExecuteMsg::Unpause {},
        &[],
    )
    .unwrap();
    app.execute_contract(
        new_governance_addr.clone(),
        reserve_addr.clone(),
        &ReserveExecuteMsg::Unpause {},
        &[],
    )
    .unwrap();

    // Verify treasury and reserve-fund are no longer paused
    let treasury_config: TreasuryConfigResponse = app
        .wrap()
        .query_wasm_smart(treasury_addr.clone(), &TreasuryQueryMsg::GetConfig {})
        .unwrap();
    assert!(!treasury_config.is_paused);

    let reserve_config: ReserveConfigResponse = app
        .wrap()
        .query_wasm_smart(reserve_addr.clone(), &ReserveQueryMsg::GetConfig {})
        .unwrap();
    assert!(!reserve_config.is_paused);
}
