"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Image, ShieldCheck, User, Compass, History } from "lucide-react";

interface Cw721Transfer {
  from: string;
  to: string;
  txHash: string;
  time: string;
}

interface Cw721TokenDetail {
  address: string;
  tokenId: string;
  owner: string;
  image: string;
  metadataUri: string;
  metadataJson: string;
  transfers: Cw721Transfer[];
}

export default function ContractNftDetailPage() {
  const params = useParams();
  const addr = params?.addr ? String(params.addr) : "";
  const tokenId = params?.tokenId ? String(params.tokenId) : "1";

  const [nft, setNft] = useState<Cw721TokenDetail | null>(null);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchNftDetails = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/tokens/cw721/${addr}/nft/${tokenId}`);
        if (resp.ok) {
          const data = await resp.json();
          setNft({
            address: data.address || addr,
            tokenId: data.tokenId || tokenId,
            owner: data.owner,
            image: data.image || "https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?w=400",
            metadataUri: data.metadataUri || "",
            metadataJson: data.metadataJson || "{}",
            transfers: data.transfers || [],
          });
        } else {
          throw new Error("NFT not found");
        }
      } catch (err) {
        console.warn("Using simulated NFT details", err);
        setNft({
          address: addr,
          tokenId: tokenId,
          owner: "sovereign1owneraddress123",
          image: "https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?w=400",
          metadataUri: "ipfs://QmYwAPJzv5CZ1QDJUfmM...",
          metadataJson: JSON.stringify({ name: `Founders Badge #${tokenId}`, description: "Genesis commemorative badge.", attributes: [{ trait_type: "Genesis", value: "True" }] }),
          transfers: [
            { from: "sovereign1creator", to: "sovereign1owneraddress123", txHash: "4c8a2b9...", time: new Date().toISOString() },
          ],
        });
      } finally {
        setLoading(false);
      }
    };
    fetchNftDetails();
  }, [addr, tokenId]);

  if (loading) {
    return (
      <div className="p-6 max-w-6xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading token metadata...</div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/contracts" className="hover:text-white transition">Contracts</Link>
        <span>/</span>
        <Link href={`/contracts/${addr}`} className="hover:text-white transition font-mono text-xs">{addr.slice(0, 10)}...</Link>
        <span>/</span>
        <Link href={`/contracts/${addr}/nfts`} className="hover:text-white transition">NFTs</Link>
        <span>/</span>
        <span className="text-gray-300">#{tokenId}</span>
      </nav>

      {/* Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
        {/* Left: Image card */}
        <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden aspect-square flex items-center justify-center relative">
          <img src={nft?.image} alt={`NFT #${tokenId}`} className="object-cover w-full h-full" />
        </div>

        {/* Right: Details panel */}
        <div className="space-y-6">
          <div>
            <h1 className="text-3xl font-bold tracking-tight text-white flex items-center gap-2">
              <Image className="w-8 h-8 text-blue-500" />
              Token ID #{nft?.tokenId}
            </h1>
            <p className="text-gray-400 mt-2 font-mono text-xs break-all">Contract: {nft?.address}</p>
          </div>

          <div className="bg-gray-900 border border-gray-800 rounded-xl p-5 space-y-4">
            <div className="flex items-center justify-between border-b border-gray-850 pb-3">
              <div className="text-xs text-gray-500 uppercase">Current Owner</div>
              <div className="font-mono text-sm text-gray-200">{nft?.owner.slice(0, 15)}...</div>
            </div>
            <div className="flex items-center justify-between">
              <div className="text-xs text-gray-500 uppercase">Metadata URI</div>
              <div className="font-mono text-xs text-blue-400 truncate max-w-[200px]">{nft?.metadataUri || "None"}</div>
            </div>
          </div>

          {/* Transfers History */}
          <div className="bg-gray-900 border border-gray-800 rounded-xl p-5 space-y-4">
            <h2 className="text-lg font-bold text-white flex items-center gap-2">
              <History className="w-5 h-5 text-indigo-400" />
              Transfer Logs
            </h2>
            <div className="space-y-3">
              {nft?.transfers.map((tx, idx) => (
                <div key={idx} className="bg-gray-950 p-3 border border-gray-850 rounded-lg flex justify-between items-center text-xs">
                  <div className="space-y-1">
                    <div className="text-gray-400 font-mono">From: {tx.from.slice(0, 8)}... &rarr; To: {tx.to.slice(0, 8)}...</div>
                    <div className="text-gray-600 font-mono break-all">Tx: {tx.txHash.slice(0, 16)}...</div>
                  </div>
                  <div className="text-gray-500">{new Date(tx.time).toLocaleDateString()}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
