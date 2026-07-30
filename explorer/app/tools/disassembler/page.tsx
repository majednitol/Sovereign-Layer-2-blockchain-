"use client";

import React from "react";
import Link from "next/link";
import { Cpu } from "lucide-react";

export default function Page() {
  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white">Bytecode Disassembler</span>
      </nav>
      <div>
        <h1 className="text-3xl font-bold text-white flex items-center gap-3">
          <Cpu className="h-8 w-8 text-cyan-400" />
          Bytecode Disassembler
        </h1>
        <p className="text-gray-400 mt-2">Disassemble EVM bytecode to opcodes.</p>
      </div>
      <div className="bg-gray-950 border border-gray-900 rounded-xl p-8 text-center">
        <p className="text-gray-500 text-sm">This page is under construction. Full functionality will be available soon.</p>
      </div>
    </div>
  );
}
