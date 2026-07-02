"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Image, ShieldCheck, Cpu, ArrowLeft } from "lucide-react";

interface NftSummary {
  tokenId: string;
  owner: string;
  image: string;
}

export default function ContractNftsPage() {
  const params = useParams();
  const addr = params?.addr ? String(params.addr) : "";
  const [tokens, setTokens] = useState<NftSummary[]>([]);
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
            })));
          }
        }
      } catch (err) {
        console.warn("Using simulated NFTs list", err);
        setTokens([
          { tokenId: "1", owner: "sovereign1owner1", image: "https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?w=200" },
          { tokenId: "2", owner: "sovereign1owner2", image: "https://images.unsplash.com/photo-1620641788421-7a1c342ea42e?w=200" },
        ]);
      } finally {
        setLoading(false);
      }
    };
    fetchNfts();
  }, [addr]);

  if (loading) {
    return (
      <div className="p-6 max-w-6xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading collection tokens...</div>
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
        <span className="text-gray-300">NFTs</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4">
        <h1 className="text-3xl font-bold tracking-tight text-white flex items-center gap-2">
          <Image className="w-8 h-8 text-blue-500" />
          CW-721 NFT Collection Gallery
        </h1>
      </div>

      {/* Grid */}
      <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-6">
        {tokens.map((token) => (
          <div key={token.tokenId} className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden hover:border-gray-700 transition">
            <div className="aspect-square bg-gray-950 flex items-center justify-center relative">
              <img src={token.image} alt={`NFT #${token.tokenId}`} className="object-cover w-full h-full" />
            </div>
            <div className="p-4 space-y-2">
              <div className="font-bold text-white">Token ID #{token.tokenId}</div>
              <div className="text-xs text-gray-500 truncate font-mono">Owner: {token.owner}</div>
              <Link
                href={`/contracts/${addr}/nfts/${token.tokenId}`}
                className="text-xs text-blue-500 hover:text-blue-400 font-semibold block text-center bg-gray-850 py-1.5 rounded transition"
              >
                View Details &rarr;
              </Link>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
