"use client";

import React, { useState, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { 
  Landmark, 
  Vote, 
  FileText, 
  CheckCircle2, 
  XCircle, 
  AlertCircle, 
  Loader2, 
  Coins, 
  Clock, 
  ArrowLeft, 
  ShieldCheck, 
  Scale,
  Sparkles
} from "lucide-react";
import { useWalletStore } from "@/store/wallet";

export default function SubmitProposalPage() {
  const router = useRouter();
  const { walletType, connected, address, connectWallet, disconnectWallet } = useWalletStore();

  // Form State
  const [title, setTitle] = useState("");
  const [propType, setPropType] = useState("Text Proposal");
  const [description, setDescription] = useState("");
  const [deposit, setDeposit] = useState("1000");
  const [votingDays, setVotingDays] = useState("7");

  // Pre-check Status State
  const [runCheck, setRunCheck] = useState(false);
  const [checkLoading, setCheckLoading] = useState(false);
  const [checksPassed, setChecksPassed] = useState(false);

  // Submission State
  const [submitting, setSubmitting] = useState(false);
  const [submitStep, setSubmitStep] = useState("");
  const [submitResult, setSubmitResult] = useState<string | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);

  // Constitution Guard Checks
  const hasTitle = title.trim().length > 0;
  const isLengthOk = description.trim().length >= 30;
  const isDepositOk = Number(deposit) >= 1000;
  const hasKeywords = /benefit|improve|upgrade|fix|secure|param|community|governance|propose/i.test(description);

  useEffect(() => {
    if (runCheck) {
      // Re-evaluate checks
      const allPassed = hasTitle && isLengthOk && isDepositOk && hasKeywords;
      setChecksPassed(allPassed);
    }
  }, [title, description, deposit, runCheck, hasTitle, isLengthOk, isDepositOk, hasKeywords]);

  const handleRunPreCheck = () => {
    setCheckLoading(true);
    setTimeout(() => {
      setCheckLoading(false);
      setRunCheck(true);
      const allPassed = hasTitle && isLengthOk && isDepositOk && hasKeywords;
      setChecksPassed(allPassed);
    }, 800);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitError(null);

    if (!connected) {
      setSubmitError("Wallet connection required to submit proposal.");
      return;
    }

    if (!hasTitle || !isLengthOk || !isDepositOk) {
      setSubmitError("Please ensure all critical Constitution Guard checks pass before submission.");
      return;
    }

    setSubmitting(true);
    setSubmitStep("Initiating static Constitution Guard check...");
    
    setTimeout(() => {
      setSubmitStep(`Requesting transaction signature via ${walletType}...`);
      setTimeout(() => {
        setSubmitStep("Broadcasting governance proposal to Sovereign L1 node...");
        setTimeout(() => {
          const mockPropId = Math.floor(Math.random() * 100) + 10;
          setSubmitResult(`Proposal #${mockPropId} successfully instantiated!`);
          setSubmitting(false);
          setSubmitStep("");

          // Redirect to governance page after 2 seconds
          setTimeout(() => {
            router.push("/governance");
          }, 2000);
        }, 1200);
      }, 1000);
    }, 700);
  };

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/governance" className="hover:text-white transition">Governance</Link>
        <span>/</span>
        <span className="text-white">Submit Proposal</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-900 pb-4 flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center space-x-2">
            <Scale className="text-blue-500 h-8 w-8" />
            <span>Submit Governance Proposal</span>
          </h1>
          <p className="text-gray-400 mt-1">
            Propose devnet upgrades, community spends, or parameter updates.
          </p>
        </div>

        {/* Wallet Connect Panel */}
        <div className="flex items-center space-x-4 bg-gray-950 border border-gray-900 p-3 rounded-xl shadow-lg w-full md:w-auto justify-between">
          {connected ? (
            <div className="flex items-center space-x-3">
              <span className="text-xs px-2 py-1 bg-green-950 text-green-400 border border-green-900 rounded font-semibold uppercase">
                {walletType}
              </span>
              <span className="text-sm text-gray-300 font-mono">
                {address ? `${address.slice(0, 8)}...${address.slice(-6)}` : ""}
              </span>
              <button 
                onClick={disconnectWallet}
                className="text-xs text-red-400 hover:text-red-300 transition"
              >
                Disconnect
              </button>
            </div>
          ) : (
            <div className="flex space-x-2">
              <button 
                onClick={() => connectWallet("keplr")}
                className="text-xs px-3 py-1.5 bg-blue-600 hover:bg-blue-500 text-white rounded-lg font-medium transition"
              >
                Connect Keplr
              </button>
              <button 
                onClick={() => connectWallet("metamask")}
                className="text-xs px-3 py-1.5 bg-amber-600 hover:bg-amber-500 text-white rounded-lg font-medium transition"
              >
                Connect MetaMask
              </button>
            </div>
          )}
        </div>
      </div>

      {submitting ? (
        <div className="bg-gray-950 border border-gray-900 rounded-2xl p-12 text-center space-y-6 max-w-lg mx-auto shadow-2xl">
          <Loader2 className="animate-spin h-12 w-12 text-blue-500 mx-auto" />
          <h3 className="text-lg font-bold text-white uppercase tracking-wider">{submitStep}</h3>
          <p className="text-sm text-gray-400">Please do not refresh or close this tab.</p>
        </div>
      ) : submitResult ? (
        <div className="bg-gray-950 border border-green-900/50 rounded-2xl p-12 text-center space-y-6 max-w-lg mx-auto shadow-2xl">
          <CheckCircle2 className="h-16 w-16 text-green-500 mx-auto" />
          <h3 className="text-xl font-bold text-white">Proposal Submitted Successfully</h3>
          <p className="text-sm text-green-400 font-medium">{submitResult}</p>
          <p className="text-xs text-gray-400 animate-pulse">Redirecting you to the Governance proposals list...</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Proposal form */}
          <div className="lg:col-span-2 bg-gray-950 border border-gray-900 p-6 rounded-2xl shadow-xl space-y-6">
            <h3 className="text-lg font-bold text-white flex items-center space-x-2 border-b border-gray-900 pb-3">
              <FileText className="text-blue-500 h-5 w-5" />
              <span>Proposal Details</span>
            </h3>

            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-1">
                <label className="text-xs text-gray-400 font-semibold uppercase">Proposal Title</label>
                <input
                  type="text"
                  required
                  placeholder="e.g. Upgrade staking min deposit parameter"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  className="w-full bg-gray-900 border border-gray-850 text-white rounded-lg px-4 py-2.5 text-sm focus:outline-none focus:border-blue-500"
                />
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div className="space-y-1">
                  <label className="text-xs text-gray-400 font-semibold uppercase">Proposal Type</label>
                  <select
                    value={propType}
                    onChange={(e) => setPropType(e.target.value)}
                    className="w-full bg-gray-900 border border-gray-850 text-white rounded-lg px-4 py-2.5 text-sm focus:outline-none focus:border-blue-500 cursor-pointer"
                  >
                    <option>Text Proposal</option>
                    <option>Parameter Change</option>
                    <option>Community Pool Spend</option>
                    <option>Software Upgrade</option>
                  </select>
                </div>

                <div className="space-y-1">
                  <label className="text-xs text-gray-400 font-semibold uppercase">Initial Deposit (SLT)</label>
                  <div className="relative">
                    <input
                      type="number"
                      required
                      placeholder="Min 1000"
                      value={deposit}
                      onChange={(e) => setDeposit(e.target.value)}
                      className="w-full bg-gray-900 border border-gray-850 text-white rounded-lg pl-4 pr-12 py-2.5 text-sm focus:outline-none focus:border-blue-500 font-mono"
                    />
                    <span className="absolute right-3 top-3 text-xs text-gray-500 font-semibold font-mono">SLT</span>
                  </div>
                </div>
              </div>

              <div className="space-y-1">
                <label className="text-xs text-gray-400 font-semibold uppercase">Proposal Description</label>
                <textarea
                  required
                  rows={8}
                  placeholder="Provide a detailed description of the governance proposal, goals, and implementations. Mention the specific benefits for the Sovereign L1 Devnet."
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="w-full bg-gray-900 border border-gray-850 text-white rounded-lg p-4 text-sm focus:outline-none focus:border-blue-500 resize-y"
                />
                <span className="text-[10px] text-gray-500 block text-right">
                  {description.trim().length} / min 30 characters
                </span>
              </div>

              <div className="space-y-1">
                <label className="text-xs text-gray-400 font-semibold uppercase">Voting Period (Days)</label>
                <div className="relative max-w-[200px]">
                  <input
                    type="number"
                    required
                    min={1}
                    max={14}
                    value={votingDays}
                    onChange={(e) => setVotingDays(e.target.value)}
                    className="w-full bg-gray-900 border border-gray-850 text-white rounded-lg pl-4 pr-12 py-2.5 text-sm focus:outline-none focus:border-blue-500 font-mono"
                  />
                  <span className="absolute right-3 top-3 text-xs text-gray-500 font-semibold font-mono">DAYS</span>
                </div>
              </div>

              {submitError && (
                <div className="text-xs text-red-400 bg-red-950/20 border border-red-900/50 p-3 rounded-lg flex items-center space-x-2">
                  <AlertCircle className="h-4 w-4 shrink-0" />
                  <span>{submitError}</span>
                </div>
              )}

              <div className="pt-4">
                <button
                  type="submit"
                  disabled={!connected || !checksPassed}
                  className="w-full bg-blue-600 hover:bg-blue-500 disabled:bg-gray-900 disabled:border disabled:border-gray-850 disabled:text-gray-500 text-white rounded-xl py-3 font-semibold transition flex items-center justify-center space-x-2 shadow-lg shadow-blue-900/10"
                >
                  <Vote className="h-5 w-5" />
                  <span>Sign and Submit Proposal</span>
                </button>
                {!connected && (
                  <span className="text-[10px] text-gray-500 text-center block mt-2">
                    Please connect Keplr or MetaMask to unlock submission.
                  </span>
                )}
              </div>
            </form>
          </div>

          {/* Constitution guard dashboard */}
          <div className="bg-gray-950 border border-gray-900 p-6 rounded-2xl shadow-xl h-fit space-y-6">
            <h3 className="text-lg font-bold text-white flex items-center space-x-2 border-b border-gray-900 pb-3">
              <ShieldCheck className="text-purple-500 h-5 w-5" />
              <span>Constitution Guard</span>
            </h3>

            <p className="text-xs text-gray-400 leading-relaxed">
              Static validation ensures proposals conform to the Sovereign L1 legal and consensus frameworks before being committed to state storage.
            </p>

            <button
              onClick={handleRunPreCheck}
              disabled={checkLoading}
              className="w-full bg-gray-900 hover:bg-gray-850 border border-gray-800 text-white rounded-xl py-2.5 text-xs font-semibold transition flex items-center justify-center space-x-2"
            >
              {checkLoading ? (
                <>
                  <Loader2 className="animate-spin h-3.5 w-3.5" />
                  <span>Scanning text...</span>
                </>
              ) : (
                <>
                  <Sparkles className="h-3.5 w-3.5 text-purple-400" />
                  <span>Run Constitution Scan</span>
                </>
              )}
            </button>

            {runCheck && (
              <div className="space-y-4 pt-2">
                <div className={`p-4 rounded-xl border text-xs flex items-center space-x-2.5 ${
                  checksPassed 
                    ? "bg-green-950/20 border-green-900/50 text-green-400" 
                    : "bg-red-950/20 border-red-900/50 text-red-400"
                }`}>
                  {checksPassed ? (
                    <>
                      <CheckCircle2 className="h-5 w-5 shrink-0" />
                      <span><strong>PASSED:</strong> Ready for on-chain submission.</span>
                    </>
                  ) : (
                    <>
                      <XCircle className="h-5 w-5 shrink-0" />
                      <span><strong>FAILED:</strong> Address constitutional errors below.</span>
                    </>
                  )}
                </div>

                <div className="space-y-3 pt-2 text-xs">
                  {/* Check 1 */}
                  <div className="flex items-center justify-between">
                    <span className="text-gray-400">1. Proposal Title Defined</span>
                    {hasTitle ? (
                      <CheckCircle2 className="h-4.5 w-4.5 text-green-500" />
                    ) : (
                      <XCircle className="h-4.5 w-4.5 text-red-500" />
                    )}
                  </div>
                  
                  {/* Check 2 */}
                  <div className="flex items-center justify-between">
                    <span className="text-gray-400">2. Text Detail Length (&ge;30)</span>
                    {isLengthOk ? (
                      <CheckCircle2 className="h-4.5 w-4.5 text-green-500" />
                    ) : (
                      <XCircle className="h-4.5 w-4.5 text-red-500" />
                    )}
                  </div>

                  {/* Check 3 */}
                  <div className="flex items-center justify-between">
                    <span className="text-gray-400">3. Deposit Requirement (&ge;1k SLT)</span>
                    {isDepositOk ? (
                      <CheckCircle2 className="h-4.5 w-4.5 text-green-500" />
                    ) : (
                      <XCircle className="h-4.5 w-4.5 text-red-500" />
                    )}
                  </div>

                  {/* Check 4 */}
                  <div className="flex items-center justify-between">
                    <span className="text-gray-400">4. Alignment Keywords Scan</span>
                    {hasKeywords ? (
                      <CheckCircle2 className="h-4.5 w-4.5 text-green-500" />
                    ) : (
                      <XCircle className="h-4.5 w-4.5 text-red-500" />
                    )}
                  </div>
                </div>

                {!checksPassed && (
                  <div className="text-[10px] text-gray-500 bg-black/35 border border-gray-900 p-3 rounded-lg leading-relaxed space-y-1">
                    <span className="font-bold text-gray-400 uppercase tracking-wide block mb-1">How to resolve:</span>
                    {!hasTitle && <div>&bull; Provide a title describing the change.</div>}
                    {!isLengthOk && <div>&bull; Elaborate the description text to be 30 characters or more.</div>}
                    {!isDepositOk && <div>&bull; Increase deposit value to 1,000 SLT or more.</div>}
                    {!hasKeywords && (
                      <div>
                        &bull; Include one or more alignment keywords in your description (e.g. <i>benefit, improve, upgrade, governance, community</i>).
                      </div>
                    )}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
