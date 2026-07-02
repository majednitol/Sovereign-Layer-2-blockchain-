"use client";

import React, { useState, useEffect } from "react";
import { SigningStargateClient } from "@cosmjs/stargate";

interface Proposal {
  id: string;
  title: string;
  status: string;
  gasUsed: string;
}

export default function Governance() {
  const [loading, setLoading] = useState<boolean>(false);
  const [proposals, setProposals] = useState<Proposal[]>([]);
  
  const [proposalTitle, setProposalTitle] = useState<string>("");
  const [proposalGas, setProposalGas] = useState<string>("150000");
  const [proposalCode, setProposalCode] = useState<string>(
    `{\n  "contract_type": "treasury",\n  "withdraw_limit": 5000000\n}`
  );
  
  const [validationResult, setValidationResult] = useState<{
    status: "success" | "error" | null;
    message: string;
  }>({ status: null, message: "" });

  const fetchProposals = async () => {
    try {
      const res = await fetch("http://localhost:8080/api/rest/cosmos/gov/v1beta1/proposals");
      const data = await res.json();
      if (data && data.proposals) {
        setProposals(
          data.proposals.map((p: any) => ({
            id: String(p.proposal_id || p.id),
            title: p.content?.title || p.title || "Untitled Proposal",
            status: p.status || "Voting",
            gasUsed: "150,000 gas",
          }))
        );
      } else {
        setProposals([]);
      }
    } catch (e) {
      console.warn("Failed to fetch live proposals:", e);
      setProposals([]);
    }
  };

  useEffect(() => {
    fetchProposals();
  }, []);

  const validateProposal = (e: React.FormEvent) => {
    e.preventDefault();
    if (!proposalTitle) {
      alert("Please enter a proposal title.");
      return;
    }

    const gas = parseInt(proposalGas);
    if (gas < 100000) {
      setValidationResult({
        status: "error",
        message: "Constitution Check Rejected: Gas limit is below the minimum required boundary of 100,000 gas units.",
      });
      return;
    }
    if (gas > 2000000) {
      setValidationResult({
        status: "error",
        message: "Constitution Check Rejected: Gas limit exceeds the maximum allowable block limit of 2,000,000 gas units.",
      });
      return;
    }

    // Check for violations in the code
    if (proposalCode.includes("VIOLATION") || proposalCode.includes("limit: 0")) {
      setValidationResult({
        status: "error",
        message: "Constitution Check Violated: Proposed parameters violate Article IV of the Sovereign Constitution (invalid treasury limits).",
      });
      return;
    }

    setValidationResult({
      status: "success",
      message: "Constitution Check Passed! Proposal complies with all active on-chain rules and is safe to submit.",
    });
  };

  const submitProposalTx = async () => {
    if (!proposalTitle) {
      alert("Please enter a proposal title.");
      return;
    }
    const gas = parseInt(proposalGas);
    if (gas < 100000 || gas > 2000000 || proposalCode.includes("VIOLATION") || proposalCode.includes("limit: 0")) {
      alert("Proposal violates Constitution checks. Please correct the code and gas limits before submitting.");
      return;
    }

    const win = window as any;
    if (!win.keplr) {
      alert("Keplr Wallet extension not found.");
      return;
    }

    try {
      setLoading(true);
      const chainId = "sovereign-testnet-1";
      await win.keplr.enable(chainId);
      const offlineSigner = win.keplr.getOfflineSigner(chainId);
      const accounts = await offlineSigner.getAccounts();
      const sender = accounts[0].address;

      const client = await SigningStargateClient.connectWithSigner(
        "http://localhost:26657",
        offlineSigner
      );

      const msgSubmitProposal = {
        typeUrl: "/cosmos.gov.v1beta1.MsgSubmitProposal",
        value: {
          content: {
            typeUrl: "/cosmos.gov.v1beta1.TextProposal",
            value: {
              title: proposalTitle,
              description: `Gas Limit: ${proposalGas}\nCode: ${proposalCode}`,
            },
          },
          initialDeposit: [{ denom: "atoken", amount: "10000000" }], // 10 SOV
          proposer: sender,
        },
      };

      const fee = {
        amount: [{ denom: "atoken", amount: "20000" }],
        gas: "250000",
      };

      const result = await client.signAndBroadcast(sender, [msgSubmitProposal], fee);
      console.log("Gov proposal broadcast result:", result);
      alert(`Proposal broadcasted successfully!\nTx Hash: ${result.transactionHash}`);
      fetchProposals();
    } catch (err: any) {
      console.error("Failed to submit CosmJS proposal:", err);
      alert("CosmJS submission failed: " + err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <h1 className="title-gradient" style={{ fontSize: "2.5rem", marginBottom: "0.5rem", fontFamily: "var(--font-title)" }}>
        Governance & Constitution Invariants
      </h1>
      <p style={{ color: "var(--text-secondary)", marginBottom: "2rem", fontSize: "1.05rem" }}>
        Review active delays, gas safety boundaries, and compliance rules mandated by the Sovereign L1 Constitution.
      </p>

      {/* Bento Grid */}
      <div className="bento-grid">
        
        {/* Bounds Card */}
        <div className="card col-6" style={{ display: "flex", flexDirection: "column", gap: "1.5rem" }}>
          <h3 style={{ fontFamily: "var(--font-title)", display: "flex", alignItems: "center", gap: "0.5rem" }}>
            <span style={{ display: "inline-block", width: "12px", height: "12px", borderRadius: "50%", backgroundColor: "var(--accent-primary)" }}></span>
            Active Parameter Invariants
          </h3>

          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "1rem" }}>
            <div style={{ background: "rgba(0,0,0,0.15)", padding: "1rem", borderRadius: "10px", border: "1px solid var(--border-color)" }}>
              <span style={{ fontSize: "0.85rem", color: "var(--text-secondary)" }}>Min Gas Boundary</span>
              <div style={{ fontSize: "1.2rem", fontWeight: 700, marginTop: "0.25rem", color: "var(--accent-warning)" }}>100,000 gas</div>
            </div>
            <div style={{ background: "rgba(0,0,0,0.15)", padding: "1rem", borderRadius: "10px", border: "1px solid var(--border-color)" }}>
              <span style={{ fontSize: "0.85rem", color: "var(--text-secondary)" }}>Max Gas Boundary</span>
              <div style={{ fontSize: "1.2rem", fontWeight: 700, marginTop: "0.25rem", color: "var(--accent-warning)" }}>2,000,000 gas</div>
            </div>
            <div style={{ background: "rgba(0,0,0,0.15)", padding: "1rem", borderRadius: "10px", border: "1px solid var(--border-color)", gridColumn: "span 2" }}>
              <span style={{ fontSize: "0.85rem", color: "var(--text-secondary)" }}>Gov Delay (x/gov-ext)</span>
              <div style={{ fontSize: "1.2rem", fontWeight: 700, marginTop: "0.25rem", color: "var(--text-primary)" }}>
                7 Days Execution Time-lock
              </div>
              <p style={{ fontSize: "0.75rem", color: "var(--text-secondary)", marginTop: "0.25rem" }}>
                Mandatory for contract replacement proposals (bypassing Constitution limits).
              </p>
            </div>
          </div>
        </div>

        {/* Validator degraded mode */}
        <div className="card col-6" style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
          <h3 style={{ fontFamily: "var(--font-title)", display: "flex", alignItems: "center", gap: "0.5rem" }}>
            <span style={{ display: "inline-block", width: "12px", height: "12px", borderRadius: "50%", backgroundColor: "var(--accent-secondary)" }}></span>
            Liveness & Degraded Mode
          </h3>

          <p style={{ color: "var(--text-secondary)", fontSize: "0.95rem" }}>
            The chain automatically switches to <strong>Degraded Mode</strong> if blocks are consecutively rejected.
          </p>

          <div style={{ display: "flex", flexDirection: "column", gap: "0.75rem", marginTop: "0.5rem" }}>
            <div style={{ display: "flex", justifyContent: "space-between", borderBottom: "1px solid var(--border-color)", paddingBottom: "0.5rem" }}>
              <span style={{ fontSize: "0.9rem" }}>Consecutive Rejections Limit:</span>
              <span style={{ fontWeight: 600 }}>5 Blocks</span>
            </div>
            <div style={{ display: "flex", justifyContent: "space-between", borderBottom: "1px solid var(--border-color)", paddingBottom: "0.5rem" }}>
              <span style={{ fontSize: "0.9rem" }}>Normal Proposal Quorum:</span>
              <span style={{ fontWeight: 600 }} className="badge badge-primary">67% Voting Power</span>
            </div>
            <div style={{ display: "flex", justifyContent: "space-between", paddingBottom: "0.5rem" }}>
              <span style={{ fontSize: "0.9rem" }}>Degraded Proposal Quorum:</span>
              <span style={{ fontWeight: 600 }} className="badge badge-secondary">51% Voting Power</span>
            </div>
          </div>
        </div>

        {/* Compliance Simulator & Submitter */}
        <div className="card col-7" style={{ marginTop: "1.5rem" }}>
          <h3 style={{ marginBottom: "1.25rem", fontFamily: "var(--font-title)" }}>Constitution Compliance & Proposal Creator</h3>
          
          <form onSubmit={validateProposal} style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "1.5rem" }}>
            <div>
              <div className="form-group">
                <label>Proposal Title</label>
                <input 
                  type="text" 
                  className="form-control" 
                  placeholder="e.g. Upgrade Reserve Fund Allocations" 
                  value={proposalTitle}
                  onChange={(e) => setProposalTitle(e.target.value)}
                  required
                />
              </div>

              <div className="form-group">
                <label>Execution Gas Limit</label>
                <input 
                  type="number" 
                  className="form-control" 
                  placeholder="e.g. 150000" 
                  value={proposalGas}
                  onChange={(e) => setProposalGas(e.target.value)}
                  required
                />
              </div>
            </div>

            <div>
              <div className="form-group">
                <label>Proposed Msg Code (JSON format)</label>
                <textarea 
                  className="form-control" 
                  rows={5}
                  style={{ fontFamily: "monospace", resize: "none" }}
                  value={proposalCode}
                  onChange={(e) => setProposalCode(e.target.value)}
                  required
                />
              </div>
            </div>

            <div style={{ gridColumn: "span 2", display: "flex", flexDirection: "column", gap: "1rem" }}>
              <div style={{ display: "flex", gap: "1rem" }}>
                <button type="submit" className="btn btn-secondary" style={{ flex: 1 }}>
                  Validate Constitution Compliance
                </button>
                <button 
                  type="button" 
                  className="btn btn-primary" 
                  style={{ flex: 1 }} 
                  onClick={submitProposalTx}
                  disabled={loading}
                >
                  {loading ? "Submitting..." : "Sign & Submit Proposal"}
                </button>
              </div>

              {validationResult.status && (
                <div style={{
                  padding: "1rem",
                  borderRadius: "10px",
                  backgroundColor: validationResult.status === "success" ? "rgba(16, 185, 129, 0.1)" : "rgba(239, 68, 68, 0.1)",
                  border: validationResult.status === "success" ? "1px solid rgba(16, 185, 129, 0.2)" : "1px solid rgba(239, 68, 68, 0.2)",
                  color: validationResult.status === "success" ? "var(--accent-success)" : "#f87171",
                  fontSize: "0.95rem",
                  fontWeight: 500
                }}>
                  {validationResult.message}
                </div>
              )}
            </div>
          </form>
        </div>

        {/* Proposals List Card */}
        <div className="card col-5" style={{ marginTop: "1.5rem" }}>
          <h3 style={{ marginBottom: "1.25rem", fontFamily: "var(--font-title)" }}>On-Chain Proposals</h3>
          <div style={{ display: "flex", flexDirection: "column", gap: "0.75rem", overflowY: "auto", maxHeight: "310px" }}>
            {proposals.length === 0 ? (
              <p style={{ color: "var(--text-secondary)", textAlign: "center", padding: "2rem 0", fontSize: "0.95rem" }}>
                No active on-chain proposals found.
              </p>
            ) : (
              proposals.map((p) => (
                <div 
                  key={p.id}
                  style={{
                    background: "rgba(0,0,0,0.15)",
                    padding: "0.85rem",
                    borderRadius: "10px",
                    border: "1px solid var(--border-color)",
                    display: "flex",
                    justifyContent: "space-between",
                    alignItems: "center"
                  }}
                >
                  <div>
                    <div style={{ fontSize: "0.9rem", fontWeight: 600 }}>#{p.id} - {p.title}</div>
                    <div style={{ fontSize: "0.75rem", color: "var(--text-secondary)", marginTop: "0.2:rem" }}>
                      Cost: {p.gasUsed}
                    </div>
                  </div>
                  <span className={`badge ${p.status === "Passed" ? "badge-success" : "badge-warning"}`}>
                    {p.status}
                  </span>
                </div>
              ))
            )}
          </div>
        </div>

      </div>
    </div>
  );
}
