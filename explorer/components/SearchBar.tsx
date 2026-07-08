"use client";

import React, { useState, useEffect, useRef } from "react";
import { Search, Database, FileText, User, Settings, ShieldAlert } from "lucide-react";
import { useRouter } from "next/navigation";
import Link from "next/link";

interface SearchResultItem {
  type: string;
  id: string;
  label: string;
}

export default function SearchBar() {
  const [searchQuery, setSearchQuery] = useState("");
  const [results, setResults] = useState<SearchResultItem[]>([]);
  const [showDropdown, setShowDropdown] = useState(false);
  const [loading, setLoading] = useState(false);
  
  const router = useRouter();
  const dropdownRef = useRef<HTMLDivElement>(null);
  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  // Handle click outside to close dropdown
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setShowDropdown(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // Fetch search autocomplete results as the user types
  useEffect(() => {
    if (searchQuery.trim().length < 2) {
      setResults([]);
      setShowDropdown(false);
      return;
    }

    const delayDebounce = setTimeout(async () => {
      setLoading(true);
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/search?query=${encodeURIComponent(searchQuery.trim())}`);
        if (resp.ok) {
          const data = await resp.json();
          setResults(data.results || []);
          setShowDropdown(true);
        }
      } catch (err) {
        console.warn("Autocomplete fetch failed", err);
      } finally {
        setLoading(false);
      }
    }, 300);

    return () => clearTimeout(delayDebounce);
  }, [searchQuery, API_BASE]);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    if (!searchQuery.trim()) return;
    
    const query = searchQuery.trim();
    setShowDropdown(false);

    if (query.startsWith("0x") && query.length === 66) {
      router.push(`/txs/${query}`);
    } else if (query.length === 64) {
      router.push(`/txs/${query}`);
    } else if (/^\d+$/.test(query)) {
      router.push(`/blocks/${query}`);
    } else if (
      query.startsWith("cosmos1") ||
      query.startsWith("sovereign1") ||
      query.startsWith("sov1") ||
      (query.startsWith("0x") && query.length === 42)
    ) {
      router.push(`/address/${query}`);
    } else {
      router.push(`/search?q=${encodeURIComponent(query)}`);
    }
    setSearchQuery("");
  };

  const getEntityLink = (item: SearchResultItem) => {
    switch (item.type) {
      case "block": return `/blocks/${item.id}`;
      case "tx": return `/txs/${item.id}`;
      case "address": return `/address/${item.id}`;
      case "contract": return `/contracts/${item.id}`;
      case "validator": return `/validators/${item.id}`;
      case "proposal": return `/governance/${item.id}`;
      case "nft": return `/evm/nfts/${item.id}`;
      default: return "#";
    }
  };

  const getIcon = (type: string) => {
    switch (type) {
      case "block": return <Database className="h-3.5 w-3.5 text-emerald-400" />;
      case "tx": return <FileText className="h-3.5 w-3.5 text-blue-400" />;
      case "address": return <User className="h-3.5 w-3.5 text-yellow-400" />;
      case "contract": return <Settings className="h-3.5 w-3.5 text-purple-400" />;
      case "validator": return <ShieldAlert className="h-3.5 w-3.5 text-red-400" />;
      default: return <Search className="h-3.5 w-3.5 text-gray-400" />;
    }
  };

  // Group results by type
  const groupedResults = results.reduce((acc, curr) => {
    if (!acc[curr.type]) acc[curr.type] = [];
    acc[curr.type].push(curr);
    return acc;
  }, {} as Record<string, SearchResultItem[]>);

  return (
    <div ref={dropdownRef} className="relative flex flex-col w-full max-w-md">
      <form onSubmit={handleSearch} className="relative flex items-center w-full">
        <Search className="absolute left-3.5 top-2.5 h-4 w-4 text-gray-500" />
        <input
          type="text"
          placeholder="Search height, hash, address..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          onFocus={() => {
            if (results.length > 0) setShowDropdown(true);
          }}
          className="w-full pl-10 pr-10 py-2 bg-gray-950 border border-gray-900 focus:border-cyan-600 focus:ring-1 focus:ring-cyan-600 rounded-lg text-sm text-white outline-none transition font-mono"
        />
        {loading && (
          <div className="absolute right-3 top-2.5">
            <div className="animate-spin rounded-full h-4 w-4 border-2 border-cyan-500 border-t-transparent" />
          </div>
        )}
      </form>

      {/* Autocomplete Dropdown */}
      {showDropdown && Object.keys(groupedResults).length > 0 && (
        <div className="absolute top-full left-0 right-0 mt-1 bg-gray-950 border border-gray-850 rounded-lg shadow-2xl z-50 max-h-96 overflow-y-auto divide-y divide-gray-900">
          {Object.entries(groupedResults).map(([type, items]) => (
            <div key={type} className="p-2 space-y-1">
              <div className="text-[10px] font-bold text-gray-500 uppercase tracking-wider px-2 py-1 flex items-center gap-1.5">
                {getIcon(type)}
                <span>{type}s</span>
              </div>
              <div className="space-y-0.5">
                {items.map((item, idx) => (
                  <Link
                    key={idx}
                    href={getEntityLink(item)}
                    onClick={() => {
                      setShowDropdown(false);
                      setSearchQuery("");
                    }}
                    className="block px-2.5 py-1.5 hover:bg-gray-900 rounded-md transition text-xs font-mono text-gray-300 hover:text-white truncate"
                  >
                    {item.label}
                  </Link>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
