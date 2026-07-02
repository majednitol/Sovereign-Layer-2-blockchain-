import React from "react";
import "./globals.css";
import ThemeProvider from "@/components/ThemeProvider";
import MultiWalletButton from "@/components/MultiWalletButton";

export const metadata = {
  title: "Sovereign L1 Explorer",
  description: "Unified Enterprise Blockchain Explorer for Sovereign L1",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <ThemeProvider attribute="class" defaultTheme="dark" enableSystem>
      <html lang="en" className="dark">
        <body className="bg-black text-gray-100 min-h-screen flex flex-col font-sans">
        {/* Navigation Bar */}
        <header className="border-b border-gray-800 bg-gray-950/80 backdrop-blur sticky top-0 z-50">
          <div className="max-w-7xl mx-auto px-4 h-16 flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className="w-8 h-8 rounded bg-gradient-to-tr from-blue-600 to-indigo-600 flex items-center justify-center font-bold text-white shadow-lg">
                S
              </div>
              <span className="text-xl font-bold tracking-tight text-white bg-clip-text bg-gradient-to-r from-white to-gray-400">
                Sovereign L1
              </span>
            </div>

            <nav className="hidden md:flex space-x-6 text-sm font-medium text-gray-400">
              <a href="/" className="hover:text-white transition">Dashboard</a>
              <a href="/blocks" className="hover:text-white transition">Blocks</a>
              <a href="/txs" className="hover:text-white transition">Transactions</a>
              <a href="/consensus" className="hover:text-white transition">Consensus</a>
              <a href="/validators" className="hover:text-white transition">Validators</a>
              <a href="/staking" className="hover:text-white transition">Staking</a>
              <a href="/governance" className="hover:text-white transition">Governance</a>
              <a href="/faucet" className="hover:text-white transition">Faucet</a>
              <a href="/verify" className="hover:text-white transition">Verify</a>
              <a href="/network" className="hover:text-white transition text-indigo-400 font-semibold">Network Config</a>
            </nav>

            <div className="flex items-center space-x-3">
              <MultiWalletButton />
              <span className="h-2 w-2 rounded-full bg-green-500 animate-pulse"></span>
              <span className="text-xs text-gray-400 font-medium hidden sm:inline">Devnet Connected</span>
            </div>
          </div>
        </header>

        {/* Content wrapper */}
        <main className="flex-grow">
          {children}
        </main>

        {/* Footer */}
        <footer className="border-t border-gray-900 bg-black py-8 mt-12">
          <div className="max-w-7xl mx-auto px-4 flex flex-col md:flex-row justify-between items-center text-xs text-gray-500 space-y-4 md:space-y-0">
            <div>
              &copy; {new Date().getFullYear()} Sovereign L1 Blockchain. All rights reserved.
            </div>
            <div className="flex space-x-6">
              <a href="/developers" className="hover:text-gray-300">Developers</a>
              <a href="/docs" className="hover:text-gray-300">API Docs</a>
              <a href="/status" className="hover:text-gray-300">System Status</a>
              <a href="/params" className="hover:text-gray-300">Parameters</a>
              <a href="/analytics" className="hover:text-gray-300">Analytics</a>
            </div>
          </div>
        </footer>
      </body>
    </html>
    </ThemeProvider>
  );
}
