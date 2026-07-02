"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Image, ShieldCheck, History, ArrowLeft, Layers, Tag } from "lucide-react";

interface EvmNftTransfer {
  from: string;
  to: string;
  txHash: string;
  time: string;
}

interface EvmNftDetail {
  address: string;
  tokenId: string;
  owner: string;
  image: string;
  metadataUri: string;
  metadataJson: string;
  transfers: EvmNftTransfer[];
}

export default function EvmNftDetailPage() {
  const params = useParams();
  const addr = params?.addr ? String(params.addr) : "";
  const tokenId = params?.tokenId ? String(params.tokenId) : "1";

  const [nft, setNft] = useState<EvmNftDetail | null>(null);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    if (!addr) return;
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
          owner: "0x1234567890abcdef1234567890abcdef12345678",
          image: "https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?w=400",
          metadataUri: "ipfs://QmYwAPJzv5CZ1QDJUfmM...",
          metadataJson: JSON.stringify({
            name: `Genesis Badge #${tokenId}`,
            description: "Genesis commemorative badge issued to early validators and operators of Sovereign network.",
            attributes: [
              { trait_type: "Background", value: "Purple" },
              { trait_type: "Rarity", value: "Legendary" },
              { trait_type: "Level", value: "Tier 5" },
              { trait_type: "Issuer", value: "Sovereign Labs" }
            ]
          }),
          transfers: [
            { from: "0x0000000000000000000000000000000000000000", to: "0x1234567890abcdef1234567890abcdef12345678", txHash: "0x3f5c9e2b1d7a8d9e8a7b6c5d4e3f281f449219d54e47fd8ad83861b464815d9d", time: new Date().toISOString() },
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
        <div className="text-gray-400">Loading NFT details...</div>
      </div>
    );
  }

  // Parse Metadata Attributes
  let parsedMeta: { name?: string; description?: string; attributes?: { trait_type: string; value: string }[] } = {};
  try {
    parsedMeta = nft?.metadataJson ? JSON.parse(nft.metadataJson) : {};
  } catch (e) {
    console.error("Failed to parse metadata JSON", e);
  }

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/evm" className="hover:text-white transition">EVM</Link>
        <span>/</span>
        <Link href={`/evm/nfts/${addr}`} className="hover:text-white transition font-mono text-xs">{addr.slice(0, 10)}...</Link>
        <span>/</span>
        <span className="text-gray-300">#{tokenId}</span>
      </nav>

      {/* Header */}
      <div className="flex items-center gap-3 border-b border-gray-800 pb-6">
        <Link href={`/evm/nfts/${addr}`} className="p-2 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-gray-400 hover:text-white transition">
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
            <Image className="w-8 h-8 text-purple-500 animate-pulse" />
            {parsedMeta.name || `Token ID #${tokenId}`}
          </h1>
          <p className="text-gray-400 mt-1 font-mono text-xs break-all">Contract: {addr}</p>
        </div>
      </div>

      {/* Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
        {/* Left: Image card */}
        <div className="bg-gray-950 border border-gray-900 rounded-2xl overflow-hidden aspect-square flex items-center justify-center relative shadow-lg">
          <img src={nft?.image} alt={`NFT #${tokenId}`} className="object-cover w-full h-full" />
        </div>

        {/* Right: Details panel */}
        <div className="space-y-6">
          <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-4 shadow-md">
            <h3 className="text-lg font-bold text-white flex items-center gap-2 border-b border-gray-900 pb-3">
              <ShieldCheck className="h-5 w-5 text-green-500" />
              Ownership & Source
            </h3>
            <div className="space-y-3 text-xs">
              <div>
                <span className="text-gray-500 font-bold uppercase">Current Owner</span>
                <span className="font-mono text-gray-200 mt-1 block select-all break-all bg-gray-900/50 border border-gray-850 p-2 rounded-lg">{nft?.owner}</span>
              </div>
              <div className="flex justify-between items-center pt-2">
                <span className="text-gray-500 font-bold uppercase">Metadata URI</span>
                <span className="font-mono text-blue-400 truncate max-w-[250px]">{nft?.metadataUri || "None"}</span>
              </div>
            </div>
          </div>

          {/* Description & Attributes */}
          <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-4 shadow-md">
            <h3 className="text-lg font-bold text-white flex items-center gap-2 border-b border-gray-900 pb-3">
              <Tag className="h-5 w-5 text-indigo-500" />
              Traits & Attributes
            </h3>
            {parsedMeta.description && (
              <p className="text-xs text-gray-400 leading-relaxed bg-gray-900/30 p-3 border border-gray-900 rounded-xl">{parsedMeta.description}</p>
            )}

            {parsedMeta.attributes && parsedMeta.attributes.length > 0 ? (
              <div className="grid grid-cols-2 gap-4">
                {parsedMeta.attributes.map((attr, idx) => (
                  <div key={idx} className="bg-gray-900/50 border border-gray-850 p-3 rounded-xl text-center space-y-1">
                    <span className="text-[10px] text-gray-500 uppercase font-bold">{attr.trait_type}</span>
                    <span className="text-sm font-extrabold text-white block">{attr.value}</span>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-xs text-gray-500">No properties parsed in metadata JSON.</div>
            )}
          </div>

          {/* Transfers History */}
          <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-4 shadow-md">
            <h3 className="text-lg font-bold text-white flex items-center gap-2 border-b border-gray-900 pb-3">
              <History className="w-5 h-5 text-blue-500" />
              Transfer Logs
            </h3>
            <div className="space-y-3 font-mono text-xs">
              {nft?.transfers.map((tx, idx) => (
                <div key={idx} className="bg-gray-900/30 p-3 border border-gray-900 rounded-xl flex justify-between items-center gap-4">
                  <div className="space-y-1">
                    <div className="text-gray-300">From: {tx.from.slice(0, 8)}... &rarr; To: {tx.to.slice(0, 8)}...</div>
                    <div className="text-gray-500 text-[10px] break-all">
                      Tx:{" "}
                      <Link href={`/evm/txs/${tx.txHash}`} className="text-blue-500 hover:underline">
                        {tx.txHash.slice(0, 16)}...
                      </Link>
                    </div>
                  </div>
                  <div className="text-gray-500 text-[10px]">{new Date(tx.time).toLocaleDateString()}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
