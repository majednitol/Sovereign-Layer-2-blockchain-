"use client";

import { useState, useRef, useEffect } from "react";
import { useWalletStore } from "@/store/wallet";
import { Wallet, ChevronDown, LogOut, Check, ExternalLink } from "lucide-react";

const WALLET_OPTIONS = [
  { id: "keplr" as const, label: "Keplr", color: "bg-blue-600 hover:bg-blue-500" },
  { id: "leap" as const, label: "Leap", color: "bg-green-600 hover:bg-green-500" },
  { id: "cosmostation" as const, label: "Cosmostation", color: "bg-purple-600 hover:bg-purple-500" },
  { id: "metamask" as const, label: "MetaMask", color: "bg-orange-500 hover:bg-orange-400" },
];

export default function MultiWalletButton() {
  const { walletType, connected, address, connectWallet, disconnectWallet } =
    useWalletStore();
  const [open, setOpen] = useState(false);
  const [connecting, setConnecting] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const handleConnect = async (type: "keplr" | "metamask" | "leap" | "cosmostation") => {
    setConnecting(type);
    setError(null);
    try {
      await connectWallet(type);
      setOpen(false);
    } catch (err: any) {
      setError(err.message || "Failed to connect");
    } finally {
      setConnecting(null);
    }
  };

  const handleDisconnect = () => {
    disconnectWallet();
    setOpen(false);
    setError(null);
  };

  const truncate = (addr: string, chars = 8) =>
    `${addr.slice(0, chars)}...${addr.slice(-4)}`;

  const activeWallet = WALLET_OPTIONS.find((w) => w.id === walletType);

  return (
    <div className="relative" ref={dropdownRef}>
      {/* Connected state */}
      {connected && address ? (
        <button
          onClick={() => setOpen(!open)}
          className="flex items-center space-x-2 bg-gray-900 border border-gray-800 hover:border-gray-700 rounded-lg px-3 py-1.5 text-sm transition"
        >
          <span className="h-2 w-2 rounded-full bg-green-500" />
          <span className="text-gray-300 font-mono text-xs">
            {truncate(address)}
          </span>
          <ChevronDown className="h-3.5 w-3.5 text-gray-500" />
        </button>
      ) : (
        <button
          onClick={() => setOpen(!open)}
          className="flex items-center space-x-2 bg-blue-600 hover:bg-blue-500 text-white rounded-lg px-4 py-1.5 text-sm font-medium transition shadow-lg shadow-blue-900/20"
        >
          <Wallet className="h-4 w-4" />
          <span>Connect Wallet</span>
        </button>
      )}

      {/* Dropdown */}
      {open && (
        <div className="absolute right-0 mt-2 w-56 bg-gray-950 border border-gray-800 rounded-xl shadow-2xl z-50 overflow-hidden">
          {/* Wallet options */}
          <div className="p-1.5 space-y-0.5">
            {connected ? (
              <>
                {/* Connected info */}
                <div className="px-3 py-2 border-b border-gray-800 mb-1">
                  <div className="text-xs text-gray-500">Connected with</div>
                  <div className="flex items-center space-x-1.5 mt-0.5">
                    <span className="text-sm font-semibold text-white">
                      {activeWallet?.label || walletType}
                    </span>
                    {activeWallet && (
                      <Check className="h-3.5 w-3.5 text-green-400" />
                    )}
                  </div>
                  <div className="text-xs font-mono text-gray-400 mt-0.5 break-all">
                    {address}
                  </div>
                </div>

                {/* Disconnect */}
                <button
                  onClick={handleDisconnect}
                  className="w-full flex items-center space-x-2 px-3 py-2 text-sm text-red-400 hover:bg-red-950/30 rounded-lg transition"
                >
                  <LogOut className="h-4 w-4" />
                  <span>Disconnect</span>
                </button>
              </>
            ) : (
              WALLET_OPTIONS.map((w) => (
                <button
                  key={w.id}
                  onClick={() => handleConnect(w.id)}
                  disabled={connecting === w.id}
                  className={`w-full flex items-center justify-between px-3 py-2.5 text-sm text-white rounded-lg transition ${
                    connecting === w.id
                      ? "opacity-50 cursor-wait"
                      : "hover:bg-gray-900"
                  }`}
                >
                  <div className="flex items-center space-x-2.5">
                    <span
                      className={`w-2.5 h-2.5 rounded-full ${w.color.replace("hover:", "")}`}
                    />
                    <span>{w.label}</span>
                  </div>
                  {connecting === w.id ? (
                    <span className="h-4 w-4 rounded-full border-2 border-t-transparent border-white animate-spin" />
                  ) : (
                    <ExternalLink className="h-3.5 w-3.5 text-gray-600" />
                  )}
                </button>
              ))
            )}
          </div>

          {/* Error message */}
          {error && (
            <div className="px-3 py-2 bg-red-950/20 border-t border-red-900/50">
              <p className="text-xs text-red-400 break-words">{error}</p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
