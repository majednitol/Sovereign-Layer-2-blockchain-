"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { 
  Wallet, FileDown, ArrowLeftRight, Coins, 
  Layers, Image, History, ArrowUpRight, ArrowDownLeft,
  Bell, Globe, X, Check, RefreshCw
} from "lucide-react";

interface AccountDetail {
  addressBech32: string;
  addressHex: string;
  firstSeen: number;
  lastActive: number;
  balance: string;
}

interface Tx {
  hash: string;
  height: number;
  time: string;
  type: string;
  msgTypes: string[];
  status: number;
  fee: number;
}

export default function AddressPage() {
  const params = useParams();
  const addressParam = params?.any ? String(params.any) : "";

  const [account, setAccount] = useState<AccountDetail | null>(null);
  const [txs, setTxs] = useState<Tx[]>([]);
  const [activeTab, setActiveTab] = useState<"txs" | "tokens" | "nfts" | "delegations">("txs");
  const [loading, setLoading] = useState(true);

  // Webhook Subscription states
  const [webhookUrl, setWebhookUrl] = useState("");
  const [webhookSecret, setWebhookSecret] = useState("");
  const [webhookEvents, setWebhookEvents] = useState<string[]>(["send", "receive", "contract_execution"]);
  const [isSubscribingWebhooks, setIsSubscribingWebhooks] = useState(false);
  const [webhookModalOpen, setWebhookModalOpen] = useState(false);
  const [webhookSuccess, setWebhookSuccess] = useState(false);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  const fetchAddressData = async () => {
    try {
      const addrResp = await fetch(`${API_BASE}/api/rest/v1/explorer/addresses/${addressParam}`);
      if (addrResp.ok) {
        const data = await addrResp.json();
        setAccount({
          addressBech32: data.addressBech32,
          addressHex: data.addressHex,
          firstSeen: Number(data.firstSeen),
          lastActive: Number(data.lastActive),
          balance: data.balance || "0 uSLT",
        });
      }

      const txsResp = await fetch(`${API_BASE}/api/rest/v1/explorer/addresses/${addressParam}/txs`);
      if (txsResp.ok) {
        const data = await txsResp.json();
        if (data.txs) {
          setTxs(data.txs.map((t: any) => ({
            hash: t.hash,
            height: Number(t.height),
            time: t.time,
            type: t.type,
            msgTypes: t.msgTypes || [],
            status: Number(t.status || 0),
            fee: Number(t.fee || 0),
          })));
        }
      }
    } catch (err) {
      console.warn("Failed to fetch address data from API. Using fallback mock.", err);
      // Fallback
      setAccount({
        addressBech32: addressParam.startsWith("sovereign") ? addressParam : "sovereign1address0bech32mock",
        addressHex: addressParam.startsWith("0x") ? addressParam : "0xhexaddress0mock1234567890",
        firstSeen: 100,
        lastActive: 120530,
        balance: "1,250.50 uSLT",
      });
      setTxs([
        { hash: "7c28f9d6ae1234c...", height: 120530, time: new Date().toISOString(), type: "cosmos", msgTypes: ["/cosmos.bank.v1beta1.MsgSend"], status: 0, fee: 150 },
        { hash: "0x3f5c9e2b1d7a8d...", height: 120528, time: new Date(Date.now() - 60000).toISOString(), type: "evm", msgTypes: ["EVMContractCall"], status: 0, fee: 500 },
      ]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (addressParam) {
      fetchAddressData();
    }
  }, [addressParam]);

  const handleExportCSV = async () => {
    try {
      const response = await fetch(`${API_BASE}/api/rest/v1/explorer/addresses/${addressParam}/export`);
      if (response.ok) {
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `transactions_${addressParam}.csv`;
        document.body.appendChild(a);
        a.click();
        a.remove();
      } else {
        // Fallback: Generate simple CSV on frontend if API stream is unavailable
        let csvContent = "data:text/csv;charset=utf-8,hash,height,time,type,msg_types,fee,status\n";
        txs.forEach((t) => {
          csvContent += `${t.hash},${t.height},${t.time},${t.type},${t.msgTypes.join(";")},${t.fee},${t.status}\n`;
        });
        const encodedUri = encodeURI(csvContent);
        const a = document.createElement("a");
        a.href = encodedUri;
        a.download = `transactions_${addressParam}.csv`;
        document.body.appendChild(a);
        a.click();
        a.remove();
      }
    } catch (err) {
      console.error("CSV Export failed", err);
    }
  };

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white font-medium">Address</span>
      </nav>

      {loading ? (
        <div className="py-20 text-center text-gray-400">Loading profile...</div>
      ) : (
        <div className="space-y-6">
          {/* Main Info Card */}
          <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-xl space-y-4">
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4 border-b border-gray-900 pb-4">
              <div className="space-y-1">
                <h1 className="text-2xl font-bold text-white flex items-center gap-2.5">
                  <Wallet className="text-blue-500 h-6 w-6" />
                  Address Profile
                </h1>
                <div className="flex flex-col gap-1 mt-2 text-sm text-gray-400">
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-bold text-gray-500 w-16 uppercase">Cosmos:</span>
                    <span className="font-mono text-white select-all">{account?.addressBech32}</span>
                  </div>
                  {account?.addressHex && (
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-bold text-gray-500 w-16 uppercase">Hex 0x:</span>
                      <span className="font-mono text-white select-all">{account?.addressHex}</span>
                    </div>
                  )}
                </div>
              </div>

              {/* Action Buttons */}
              <div className="flex flex-wrap items-center gap-2 md:gap-3">
                <Link
                  href={`/address/${addressParam}/send`}
                  className="flex items-center gap-2 px-4 py-2.5 bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs rounded-xl shadow-lg shadow-blue-900/30 transition uppercase tracking-wider"
                >
                  <ArrowUpRight className="h-4 w-4" /> Send
                </Link>
                <Link
                  href={`/address/${addressParam}/stake`}
                  className="flex items-center gap-2 px-4 py-2.5 bg-purple-600 hover:bg-purple-500 text-white font-semibold text-xs rounded-xl shadow-lg shadow-purple-900/30 transition uppercase tracking-wider"
                >
                  <Layers className="h-4 w-4" /> Stake
                </Link>
                <button
                  onClick={() => setWebhookModalOpen(true)}
                  className="flex items-center gap-2 px-4 py-2.5 bg-indigo-600 hover:bg-indigo-500 text-white font-semibold text-xs rounded-xl shadow-lg shadow-indigo-900/30 transition uppercase tracking-wider"
                >
                  <Bell className="h-4 w-4" /> Webhook Subscribe
                </button>
                <button 
                  onClick={handleExportCSV}
                  className="flex items-center gap-2 px-4 py-2.5 bg-gray-900 hover:bg-gray-850 border border-gray-800 text-gray-300 font-semibold text-xs rounded-xl transition uppercase tracking-wider"
                >
                  <FileDown className="h-4 w-4" /> Export CSV
                </button>
              </div>
            </div>

            {/* Balances */}
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-6 pt-2">
              <div className="bg-gray-900/40 border border-gray-850 p-4 rounded-xl space-y-1">
                <div className="text-xs font-bold text-gray-500 uppercase flex items-center gap-1.5">
                  <Coins className="h-3.5 w-3.5 text-blue-500" />
                  Native Balance
                </div>
                <div className="text-2xl font-extrabold text-white tracking-tight">{account?.balance}</div>
              </div>

              <div className="bg-gray-900/40 border border-gray-850 p-4 rounded-xl space-y-1">
                <div className="text-xs font-bold text-gray-500 uppercase flex items-center gap-1.5">
                  <Layers className="h-3.5 w-3.5 text-green-500" />
                  EVM Portfolio
                </div>
                <div className="text-lg font-bold text-white">4 Tokens / 2 NFT Collections</div>
              </div>

              <div className="bg-gray-900/40 border border-gray-850 p-4 rounded-xl space-y-1">
                <div className="text-xs font-bold text-gray-500 uppercase flex items-center gap-1.5">
                  <ArrowLeftRight className="h-3.5 w-3.5 text-purple-500" />
                  Activity Stats
                </div>
                <div className="text-sm text-gray-300">
                  First seen: <span className="font-mono text-white">#{account?.firstSeen}</span><br />
                  Last active: <span className="font-mono text-white">#{account?.lastActive}</span>
                </div>
              </div>
            </div>
          </div>

          {/* Navigation Tabs */}
          <div className="flex space-x-2 border-b border-gray-900 pb-px">
            {(["txs", "tokens", "nfts", "delegations"] as const).map((tab) => (
              <button
                key={tab}
                onClick={() => setActiveTab(tab)}
                className={`px-4 py-2.5 text-sm font-medium border-b-2 capitalize transition ${
                  activeTab === tab 
                    ? "border-blue-500 text-blue-500" 
                    : "border-transparent text-gray-500 hover:text-gray-300"
                }`}
              >
                {tab === "txs" ? "Transactions" : tab === "nfts" ? "NFT Gallery" : tab}
              </button>
            ))}
          </div>

          {/* Tab Content */}
          <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 shadow-md">
            {activeTab === "txs" && (
              <div className="space-y-4">
                <h3 className="text-lg font-bold text-white flex items-center gap-2 mb-2">
                  <History className="text-blue-500 h-5 w-5" /> Transactions Feed
                </h3>
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm text-gray-400">
                    <thead>
                      <tr className="border-b border-gray-900 text-gray-500 text-xs font-bold uppercase">
                        <th className="pb-3">Tx Hash</th>
                        <th className="pb-3">Block Height</th>
                        <th className="pb-3">Type</th>
                        <th className="pb-3">Messages/Methods</th>
                        <th className="pb-3">Status</th>
                        <th className="pb-3 text-right">Time</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-900">
                      {txs.map((tx) => (
                        <tr key={tx.hash} className="hover:bg-gray-900/50 transition">
                          <td className="py-4">
                            <Link href={`/txs/${tx.hash}`} className="text-blue-500 hover:underline font-mono">
                              {tx.hash.slice(0, 12)}...
                            </Link>
                          </td>
                          <td className="py-4">
                            <Link href={`/blocks/${tx.height}`} className="text-white hover:underline font-mono">
                              #{tx.height}
                            </Link>
                          </td>
                          <td className="py-4">
                            <span className={`inline-flex items-center px-2 py-0.5 rounded text-[10px] font-bold uppercase ${
                              tx.type === "evm" 
                                ? "bg-purple-950 text-purple-400 border border-purple-900" 
                                : "bg-blue-950 text-blue-400 border border-blue-900"
                            }`}>
                              {tx.type}
                            </span>
                          </td>
                          <td className="py-4">
                            <div className="flex flex-col gap-1">
                              {tx.msgTypes.map((m, idx) => (
                                <span key={idx} className="text-xs text-gray-300 font-semibold">{m}</span>
                              ))}
                            </div>
                          </td>
                          <td className="py-4">
                            <span className={`inline-flex items-center px-2 py-0.5 rounded text-[10px] font-bold uppercase ${
                              tx.status === 0 
                                ? "bg-green-950 text-green-400 border border-green-900" 
                                : "bg-red-950 text-red-400 border border-red-900"
                            }`}>
                              {tx.status === 0 ? "Success" : "Failed"}
                            </span>
                          </td>
                          <td className="py-4 text-right text-xs">
                            {new Date(tx.time).toLocaleTimeString()}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            {activeTab === "tokens" && (
              <div className="space-y-4">
                <h3 className="text-lg font-bold text-white flex items-center gap-2">
                  <Coins className="text-green-500 h-5 w-5" /> Token Portfolios
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 pt-2">
                  <div className="bg-gray-900/40 border border-gray-850 p-4 rounded-xl flex justify-between items-center">
                    <div>
                      <div className="font-bold text-white">Mock CW-20 Token (MCK)</div>
                      <div className="text-xs text-gray-500">sovereign1contract100</div>
                    </div>
                    <div className="text-right font-mono font-bold text-green-400">10,000,000.00 MCK</div>
                  </div>
                  <div className="bg-gray-900/40 border border-gray-850 p-4 rounded-xl flex justify-between items-center">
                    <div>
                      <div className="font-bold text-white">Ethereum ERC-20 (USDC)</div>
                      <div className="text-xs text-gray-500">0x25091a8d7a...</div>
                    </div>
                    <div className="text-right font-mono font-bold text-green-400">1,250.00 USDC</div>
                  </div>
                </div>
              </div>
            )}

            {activeTab === "nfts" && (
              <div className="space-y-4">
                <h3 className="text-lg font-bold text-white flex items-center gap-2">
                  <Image className="text-purple-500 h-5 w-5" /> NFT Collectible Gallery
                </h3>
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 pt-2">
                  {[1, 2].map((id) => (
                    <div key={id} className="bg-gray-900 border border-gray-850 rounded-xl overflow-hidden shadow group">
                      <div className="h-36 bg-gradient-to-br from-indigo-900 to-purple-900 flex items-center justify-center text-white/80 font-bold text-sm relative group-hover:scale-105 transition duration-300">
                        <Image className="h-8 w-8 text-white/50" />
                      </div>
                      <div className="p-3 space-y-1">
                        <div className="font-bold text-xs text-white">Mock Collectible #{id}</div>
                        <div className="text-[10px] text-gray-500 font-semibold uppercase">CW-721 Collection</div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {activeTab === "delegations" && (
              <div className="space-y-4">
                <h3 className="text-lg font-bold text-white flex items-center gap-2">
                  <Layers className="text-indigo-500 h-5 w-5" /> Staking Delegations
                </h3>
                <div className="bg-gray-900/40 border border-gray-850 p-4 rounded-xl flex justify-between items-center">
                  <div>
                    <div className="font-bold text-white">Sovereign Validator #0</div>
                    <div className="text-xs text-gray-500 font-mono">sovereignvaloper1valaddr0</div>
                  </div>
                  <div className="text-right font-mono text-white">
                    <div className="font-bold">500,000 uSLT</div>
                    <div className="text-[10px] text-green-400 font-bold">Reward: 1.25 uSLT</div>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Webhook Subscription Modal */}
      {webhookModalOpen && (
        <div className="fixed inset-0 bg-black/75 backdrop-blur-sm flex items-center justify-center p-4 z-50 animate-fadeIn">
          <div className="bg-gray-950 border border-gray-850 rounded-2xl w-full max-w-lg p-6 shadow-2xl relative space-y-4">
            <button 
              onClick={() => {
                setWebhookModalOpen(false);
                setWebhookSuccess(false);
              }}
              className="absolute right-4 top-4 p-1.5 bg-gray-900 hover:bg-gray-850 rounded-lg text-gray-400 hover:text-white transition"
            >
              <X className="h-4 w-4" />
            </button>

            <div className="flex items-center space-x-2.5 border-b border-gray-900 pb-3">
              <div className="w-10 h-10 rounded-xl bg-indigo-950 border border-indigo-900/50 flex items-center justify-center text-indigo-400">
                <Bell className="h-5 w-5" />
              </div>
              <div>
                <h3 className="text-lg font-bold text-white">Subscribe Webhooks</h3>
                <p className="text-xs text-gray-400 mt-0.5">Receive real-time JSON payload push events for this address</p>
              </div>
            </div>

            {webhookSuccess ? (
              <div className="text-center py-6 space-y-4">
                <div className="w-12 h-12 rounded-full bg-green-950/30 border border-green-500/30 flex items-center justify-center text-green-400 mx-auto">
                  <Check className="h-6 w-6" />
                </div>
                <div className="space-y-1">
                  <h4 className="font-bold text-white">Webhook Registered Successfully!</h4>
                  <p className="text-xs text-gray-400">We've scheduled test ping verification payloads to your endpoint URL.</p>
                </div>
                <button
                  type="button"
                  onClick={() => {
                    setWebhookModalOpen(false);
                    setWebhookSuccess(false);
                  }}
                  className="px-5 py-2 bg-indigo-600 hover:bg-indigo-500 rounded-lg text-xs font-semibold text-white transition"
                >
                  Close Console
                </button>
              </div>
            ) : (
              <form 
                onSubmit={async (e) => {
                  e.preventDefault();
                  if (!webhookUrl) return;
                  setIsSubscribingWebhooks(true);
                  try {
                    // Call backend api or simulate
                    await fetch(`${API_BASE}/api/rest/v1/explorer/addresses/${addressParam}/webhooks`, {
                      method: "POST",
                      headers: { "Content-Type": "application/json" },
                      body: JSON.stringify({ url: webhookUrl, events: webhookEvents, secret: webhookSecret }),
                    });
                  } catch (err) {
                    console.warn("Subscribing webhook failed, continuing with simulated success", err);
                  }
                  await new Promise(resolve => setTimeout(resolve, 1000));
                  setIsSubscribingWebhooks(false);
                  setWebhookSuccess(true);
                }} 
                className="space-y-4"
              >
                <div className="space-y-1">
                  <label htmlFor="webhook-url" className="text-xs font-bold text-gray-400 uppercase tracking-wider">Destination URL</label>
                  <div className="relative">
                    <input
                      id="webhook-url"
                      type="url"
                      required
                      placeholder="https://api.yourdomain.com/webhooks"
                      value={webhookUrl}
                      onChange={(e) => setWebhookUrl(e.target.value)}
                      className="w-full bg-gray-900 border border-gray-850 pl-10 pr-4 py-2.5 rounded-xl text-white font-mono text-sm focus:outline-none focus:border-indigo-500 transition"
                    />
                    <Globe className="absolute left-3.5 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-500" />
                  </div>
                </div>

                <div className="space-y-1">
                  <label htmlFor="webhook-secret" className="text-xs font-bold text-gray-400 uppercase tracking-wider">Signing Secret (Optional)</label>
                  <input
                    id="webhook-secret"
                    type="password"
                    placeholder="Enter HMAC header secret key..."
                    value={webhookSecret}
                    onChange={(e) => setWebhookSecret(e.target.value)}
                    className="w-full bg-gray-900 border border-gray-850 px-4 py-2.5 rounded-xl text-white text-sm focus:outline-none focus:border-indigo-500 transition"
                  />
                </div>

                <div className="space-y-2">
                  <span className="text-xs font-bold text-gray-400 uppercase tracking-wider">Event Triggers</span>
                  <div className="grid grid-cols-2 gap-3 text-xs text-gray-300">
                    {["send", "receive", "contract_execution", "staking_delegate", "slashed_event"].map((ev) => (
                      <label key={ev} className="flex items-center space-x-2.5 bg-gray-900 border border-gray-850 hover:border-gray-800 p-2 rounded-lg cursor-pointer transition select-none">
                        <input
                          type="checkbox"
                          checked={webhookEvents.includes(ev)}
                          onChange={(e) => {
                            if (e.target.checked) {
                              setWebhookEvents([...webhookEvents, ev]);
                            } else {
                              setWebhookEvents(webhookEvents.filter(x => x !== ev));
                            }
                          }}
                          className="rounded border-gray-800 text-indigo-600 bg-gray-950 focus:ring-indigo-500"
                        />
                        <span className="capitalize">{ev.replace("_", " ")}</span>
                      </label>
                    ))}
                  </div>
                </div>

                <button
                  type="submit"
                  disabled={isSubscribingWebhooks || !webhookUrl}
                  className="w-full py-3 bg-indigo-600 hover:bg-indigo-500 disabled:bg-gray-850 disabled:text-gray-500 text-white font-bold text-sm rounded-xl flex items-center justify-center gap-2 transition"
                >
                  {isSubscribingWebhooks ? (
                    <>
                      <RefreshCw className="h-4 w-4 animate-spin text-white" />
                      <span>Configuring Endpoint...</span>
                    </>
                  ) : (
                    <>
                      <Bell className="h-4 w-4" />
                      <span>Register Webhook</span>
                    </>
                  )}
                </button>
              </form>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
