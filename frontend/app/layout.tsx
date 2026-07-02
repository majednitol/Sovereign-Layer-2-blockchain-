import type { Metadata } from "next";
import "./globals.css";
import Link from "next/link";

export const metadata: Metadata = {
  title: "Sovereign Portal — Bridge, Governance & Analytics",
  description: "Cross-chain bridge tracker, governance constitution checker, analytics dashboard, and EVM explorer for the Sovereign L1 Blockchain.",
  keywords: "blockchain, cosmos, bsc, bridge, cross-chain, governance, constitution, web3, analytics, evm, wagmi, viem",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <head>
        <link rel="icon" href="/favicon.ico" />
      </head>
      <body>
        <header>
          <div className="nav-container">
            <Link href="/" className="logo">
              <span className="logo-icon"></span>
              Sovereign Portal
            </Link>
            <nav>
              <ul>
                <li>
                  <Link href="/">Bridge & Activity</Link>
                </li>
                <li>
                  <Link href="/governance">Governance Invariants</Link>
                </li>
                <li>
                  <Link href="/dashboard">Analytics Dashboard</Link>
                </li>
              </ul>
            </nav>
          </div>
        </header>
        <main className="container">{children}</main>
        <footer>
          <div className="footer-container">
            <div className="footer-left">
              <span>© {new Date().getFullYear()} Sovereign L1 Blockchain. All rights reserved.</span>
            </div>
            <div className="footer-right">
              <a href="http://localhost:3001" target="_blank" rel="noopener noreferrer" className="footer-link">Explorer</a>
              <span className="footer-separator">•</span>
              <a href="https://github.com/majednitol/Sovereign-L1-Blockchain" target="_blank" rel="noopener noreferrer" className="footer-link">GitHub</a>
              <span className="footer-separator">•</span>
              <a href="#" className="footer-link">Documentation</a>
            </div>
          </div>
        </footer>
      </body>
    </html>
  );
}
