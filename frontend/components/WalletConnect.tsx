"use client";

import React, { useState, useEffect } from "react";
import walletsConfig from "../config/wallets.json";

interface WalletConnectProps {
  onConnectL1: (address: string) => void;
  onConnectL2: (address: string) => void;
  onDisconnect: () => void;
}

export default function WalletConnect({
  onConnectL1,
  onConnectL2,
  onDisconnect,
}: WalletConnectProps) {
  const [l1Address, setL1Address] = useState<string>("");
  const [l2Address, setL2Address] = useState<string>("");
  const [loading, setLoading] = useState<string | null>(null);

  // Initialize wallets from localStorage and sync live active account on mount
  useEffect(() => {
    // Graceful check for WalletConnect projectId from configuration
    const wcProjectId = walletsConfig.walletConnect?.projectId;
    if (!wcProjectId || wcProjectId.includes("OWNER_ACTION_REQUIRED") || wcProjectId.trim() === "") {
      console.warn(
        "WalletConnect Project ID is not configured. Gracefully falling back to direct browser extension providers (Keplr, MetaMask)."
      );
    }

    const storedL1 = localStorage.getItem("l1_address");
    if (storedL1) {
      setL1Address(storedL1);
      onConnectL1(storedL1);
    }

    const win = window as any;
    if (win.ethereum) {
      // Sync currently active account from MetaMask if authorized
      win.ethereum.request({ method: "eth_accounts" }).then((accounts: string[]) => {
        if (accounts && accounts.length > 0) {
          setL2Address(accounts[0]);
          localStorage.setItem("l2_address", accounts[0]);
          onConnectL2(accounts[0]);
        } else {
          setL2Address("");
          localStorage.removeItem("l2_address");
        }
      }).catch(console.error);

      // Listen for live MetaMask account switches
      const handleAccountsChanged = (accounts: string[]) => {
        if (accounts && accounts.length > 0) {
          setL2Address(accounts[0]);
          localStorage.setItem("l2_address", accounts[0]);
          onConnectL2(accounts[0]);
        } else {
          setL2Address("");
          localStorage.removeItem("l2_address");
          onDisconnect();
        }
      };

      if (win.ethereum.on) {
        win.ethereum.on("accountsChanged", handleAccountsChanged);
      }

      return () => {
        if (win.ethereum.removeListener) {
          win.ethereum.removeListener("accountsChanged", handleAccountsChanged);
        }
      };
    } else {
      const storedL2 = localStorage.getItem("l2_address");
      if (storedL2) {
        setL2Address(storedL2);
        onConnectL2(storedL2);
      }
    }
  }, []);

  const connectKeplr = async () => {
    setLoading("l1");
    await new Promise((resolve) => setTimeout(resolve, 800));

    const win = window as any;
    if (!win.keplr) {
      alert("Keplr Wallet extension not found. Please install Keplr wallet.");
      setLoading(null);
      return;
    }

    try {
      // Suggest the chain to Keplr if it is not already registered
      if (win.keplr.experimentalSuggestChain) {
        try {
          await win.keplr.experimentalSuggestChain(walletsConfig.keplr);
        } catch (e) {
          console.warn("Failed to suggest Keplr chain:", e);
        }
      }
      await win.keplr.enable("sovereign-1");
      const offlineSigner = win.keplr.getOfflineSigner("sovereign-1");
      const accounts = await offlineSigner.getAccounts();
      setL1Address(accounts[0].address);
      localStorage.setItem("l1_address", accounts[0].address);
      onConnectL1(accounts[0].address);
    } catch (err: any) {
      console.error("Keplr connection error:", err);
      alert("Failed to connect Keplr: " + err.message);
    }
    setLoading(null);
  };

  const connectOkx = async () => {
    setLoading("okx");
    await new Promise((resolve) => setTimeout(resolve, 800));

    const win = window as any;
    const okxKeplr = win.okxwallet?.keplr || win.okxwallet;
    if (!okxKeplr) {
      alert("OKX Wallet extension not found. Please install OKX Wallet.");
      setLoading(null);
      return;
    }

    try {
      if (okxKeplr.experimentalSuggestChain) {
        try {
          await okxKeplr.experimentalSuggestChain(walletsConfig.keplr);
        } catch (e) {
          console.warn("Failed to suggest OKX chain:", e);
        }
      }
      await okxKeplr.enable("sovereign-1");
      const offlineSigner = okxKeplr.getOfflineSigner("sovereign-1");
      const accounts = await offlineSigner.getAccounts();
      setL1Address(accounts[0].address);
      localStorage.setItem("l1_address", accounts[0].address);
      onConnectL1(accounts[0].address);
    } catch (err: any) {
      console.error("OKX connection error:", err);
      alert("Failed to connect OKX Wallet: " + err.message);
    }
    setLoading(null);
  };

  const connectCosmostation = async () => {
    setLoading("cosmostation");
    await new Promise((resolve) => setTimeout(resolve, 800));

    const win = window as any;
    if (!win.cosmostation) {
      alert("Cosmostation extension not found. Please install Cosmostation wallet.");
      setLoading(null);
      return;
    }

    try {
      // 1. Try native Cosmostation cos_addChain API
      if (win.cosmostation.cosmos?.request) {
        try {
          await win.cosmostation.cosmos.request({
            method: "cos_addChain",
            params: {
              chainId: "sovereign-1",
              chainName: "Sovereign L1",
              addressPrefix: "cosmos",
              baseDenom: "ucsov",
              displayDenom: "CSOV",
              restURL: "http://localhost:1317",
              coinType: "60",
              decimals: 6,
              gasRate: {
                low: "0.01",
                average: "0.025",
                tiny: "0.001",
              },
            },
          });
        } catch (e) {
          console.warn("Native cos_addChain skipped/failed, falling back to keplr provider:", e);
        }
      }

      // 2. Fallback / supplementary Keplr provider integration
      const cosmostationProvider = win.cosmostation.providers?.keplr || win.keplr;
      if (cosmostationProvider?.experimentalSuggestChain) {
        try {
          await cosmostationProvider.experimentalSuggestChain(walletsConfig.keplr);
        } catch (e) {
          console.warn("Failed to suggest Cosmostation chain:", e);
        }
      }
      if (cosmostationProvider?.enable) {
        await cosmostationProvider.enable("sovereign-1");
        const offlineSigner = cosmostationProvider.getOfflineSigner("sovereign-1");
        const accounts = await offlineSigner.getAccounts();
        setL1Address(accounts[0].address);
        localStorage.setItem("l1_address", accounts[0].address);
        onConnectL1(accounts[0].address);
      }
    } catch (err: any) {
      console.error("Cosmostation connection error:", err);
      alert("Failed to connect Cosmostation: " + err.message);
    }
    setLoading(null);
  };

  const connectMetaMask = async () => {
    setLoading("l2");
    await new Promise((resolve) => setTimeout(resolve, 800));

    const win = window as any;
    if (!win.ethereum) {
      alert("MetaMask extension not found. Please install MetaMask wallet.");
      setLoading(null);
      return;
    }

    try {
      const accounts = await win.ethereum.request({ method: "eth_requestAccounts" });
      setL2Address(accounts[0]);
      localStorage.setItem("l2_address", accounts[0]);
      onConnectL2(accounts[0]);
    } catch (err: any) {
      console.error("MetaMask connection error:", err);
      alert("Failed to connect MetaMask: " + err.message);
    }
    setLoading(null);
  };

  const addSovereignEVMNetwork = async () => {
    const win = window as any;
    if (!win.ethereum) {
      alert("MetaMask extension not found. Please install it first.");
      return;
    }
    try {
      await win.ethereum.request({
        method: "wallet_addEthereumChain",
        params: [walletsConfig.metamaskSovereignEvm],
      });
      alert("Sovereign EVM network added successfully!");
    } catch (err: any) {
      console.error("Failed to add Sovereign EVM Network:", err);
      if (err.message && err.message.includes("HTTPS")) {
        alert("MetaMask API requires HTTPS for programmatic add.\n\nPlease add manually in MetaMask:\n• Network Name: Sovereign L1 EVM\n• RPC URL: http://localhost:8545\n• Chain ID: 7777\n• Symbol: CSOV");
      } else {
        alert("Failed to add network: " + err.message);
      }
    }
  };

  const handleDisconnect = () => {
    setL1Address("");
    setL2Address("");
    localStorage.removeItem("l1_address");
    localStorage.removeItem("l2_address");
    onDisconnect();
  };

  const truncate = (addr: string) => {
    if (!addr) return "";
    return addr.slice(0, 8) + "..." + addr.slice(-6);
  };

  return (
    <div className="card col-12" style={{ display: "flex", justifyContent: "space-between", alignItems: "center", flexWrap: "wrap", gap: "1rem" }}>
      <div>
        <h3 style={{ marginBottom: "0.25rem", fontFamily: "var(--font-title)" }}>Connect Wallets</h3>
        <p style={{ color: "var(--text-secondary)", fontSize: "0.875rem" }}>
          Connect Keplr, Cosmostation, or OKX for L1 Cosmos and MetaMask for L2 EVM.
        </p>
      </div>

      <div style={{ display: "flex", gap: "0.75rem", flexWrap: "wrap", alignItems: "center" }}>
        {l1Address ? (
          <div className="badge badge-primary" style={{ padding: "0.5rem 1rem", fontSize: "0.85rem", borderRadius: "10px" }}>
            Cosmos: {truncate(l1Address)}
          </div>
        ) : (
          <>
            <button className="btn btn-primary" onClick={connectKeplr} disabled={loading !== null}>
              {loading === "l1" ? "Connecting..." : "Keplr"}
            </button>
            <button className="btn btn-primary" style={{ background: "linear-gradient(135deg, #6B73FF 0%, #000DFF 100%)" }} onClick={connectCosmostation} disabled={loading !== null}>
              {loading === "cosmostation" ? "Connecting..." : "Cosmostation"}
            </button>
            <button className="btn btn-primary" style={{ background: "linear-gradient(135deg, #111 0%, #333 100%)", border: "1px solid rgba(255,255,255,0.2)" }} onClick={connectOkx} disabled={loading !== null}>
              {loading === "okx" ? "Connecting..." : "OKX Wallet"}
            </button>
          </>
        )}

        {l2Address ? (
          <div style={{ display: "flex", gap: "0.5rem", alignItems: "center" }}>
            <div className="badge badge-secondary" style={{ padding: "0.5rem 1rem", fontSize: "0.85rem", borderRadius: "10px" }}>
              BSC: {truncate(l2Address)}
            </div>
            <button 
              className="btn btn-secondary" 
              style={{ padding: "0.4rem 0.8rem", fontSize: "0.75rem", background: "rgba(255,255,255,0.05)" }} 
              onClick={addSovereignEVMNetwork}
            >
              Add Sovereign EVM
            </button>
          </div>
        ) : (
          <button className="btn btn-primary" style={{ background: "linear-gradient(135deg, var(--accent-secondary) 0%, #db2777 100%)", boxShadow: "0 4px 15px rgba(236, 72, 153, 0.3)" }} onClick={connectMetaMask} disabled={loading !== null}>
            {loading === "l2" ? "Connecting..." : "MetaMask"}
          </button>
        )}

        {(l1Address || l2Address) && (
          <button className="btn btn-secondary" onClick={handleDisconnect}>
            Disconnect
          </button>
        )}
      </div>
    </div>
  );
}
