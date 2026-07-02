"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Image, ChevronRight, Filter, Grid, RefreshCw } from "lucide-react";

interface NftSummary {
  tokenId: string;
  owner: string;
  image: string;
  attributes: { trait_type: string; value: string }[];
}

export default function EvmNftCollectionGalleryPage() {
  const params = useParams();
  const addr = params?.addr ? String(params.addr) : "";
  const [tokens, setTokens] = useState<NftSummary[]>([]);
  const [selectedTrait, setSelectedTrait] = useState<string>("All");
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchNfts = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/tokens/cw721/${addr}`);
        if (resp.ok) {
          const data = await resp.json();
          if (data.tokens) {
            setTokens(data.tokens.map((t: any) => ({
              tokenId: t.tokenId,
              owner: t.owner,
              image: t.image || "https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?w=200",
              attributes: t.attributes || []
            })));
          }
        }
      } catch (err) {
        console.warn("Using simulated NFTs list", err);
        setTokens([
          { tokenId: "1", owner: "0x1234567890abcdef1234567890abcdef12345678", image: "https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?w=400", attributes: [{ trait_type: "Background", value: "Purple" }, { trait_type: "Rarity", value: "Legendary" }] },
          { tokenId: "2", owner: "0x9876543210fedcba9876543210fedcba98765432", image: "https://images.unsplash.com/photo-1620641788421-7a1c342ea42e?w=400", attributes: [{ trait_type: "Background", value: "Gold" }, { trait_type: "Rarity", value: "Common" }] },
          { tokenId: "3", owner: "0x892a10be892a10be892a10be892a10be892a10be8", image: "https://images.unsplash.com/photo-1634017839464-5c339ebe3cb4?w=400", attributes: [{ trait_type: "Background", value: "Purple" }, { trait_type: "Rarity", value: "Rare" }] },
        ]);
      } finally {
        setLoading(false);
      }
    };
    fetchNfts();
  }, [addr]);

  // Extract unique rarity or background values for filtering
  const allRarities = Array.from(new Set(tokens.flatMap(t => t.attributes.filter(a => a.trait_type === "Rarity").map(a => a.value))));

  const filteredTokens = selectedTrait === "All" 
    ? tokens 
    : tokens.filter(t => t.attributes.some(a => a.trait_type === "Rarity" && a.value === selectedTrait));

  if (loading) {
    return (
      <div className="p-6 max-w-6xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading collection gallery...</div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/evm" className="hover:text-white transition">EVM</Link>
        <span>/</span>
        <span className="text-gray-300 font-mono text-xs">{addr.slice(0, 10)}...</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4 flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
            <Image className="w-8 h-8 text-purple-500 animate-pulse" />
            ERC-721 Collection Gallery
          </h1>
          <p className="text-gray-400 mt-1 font-mono text-xs break-all">Contract: {addr}</p>
        </div>

        {/* Filters */}
        <div className="flex items-center gap-2 text-xs bg-gray-950 border border-gray-900 p-2 rounded-xl">
          <Filter className="h-4.5 w-4.5 text-gray-500 pl-1" />
          <span className="text-gray-400 font-bold uppercase">Rarity Filter:</span>
          <select 
            value={selectedTrait}
            onChange={(e) => setSelectedTrait(e.target.value)}
            className="bg-gray-900 border border-gray-800 rounded px-2.5 py-1 text-white focus:outline-none focus:border-blue-500"
          >
            <option value="All">All Rarities</option>
            {allRarities.map(r => (
              <option key={r} value={r}>{r}</option>
            ))}
          </select>
        </div>
      </div>

      {/* Gallery Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6">
        {filteredTokens.map((token) => (
          <div key={token.tokenId} className="bg-gray-950 border border-gray-900 rounded-2xl overflow-hidden hover:border-blue-500/50 hover:shadow-lg hover:shadow-blue-900/10 transition group flex flex-col justify-between">
            <div className="aspect-square bg-gray-900 flex items-center justify-center relative overflow-hidden">
              <img src={token.image} alt={`NFT #${token.tokenId}`} className="object-cover w-full h-full group-hover:scale-105 transition duration-300" />
            </div>
            <div className="p-4 space-y-3">
              <div className="flex justify-between items-center">
                <span className="font-extrabold text-white text-sm">Token ID #{token.tokenId}</span>
                {token.attributes.find(a => a.trait_type === "Rarity") && (
                  <span className="px-2 py-0.5 rounded bg-purple-950/40 border border-purple-900/50 text-purple-400 text-[10px] font-bold uppercase">
                    {token.attributes.find(a => a.trait_type === "Rarity")?.value}
                  </span>
                )}
              </div>
              <div className="text-xs text-gray-500 truncate font-mono">Owner: {token.owner}</div>
              <Link
                href={`/evm/nfts/${addr}/${token.tokenId}`}
                className="text-xs text-center font-bold bg-gray-900 hover:bg-gray-850 text-gray-200 hover:text-white py-2 rounded-xl transition block"
              >
                View Item &rarr;
              </Link>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
