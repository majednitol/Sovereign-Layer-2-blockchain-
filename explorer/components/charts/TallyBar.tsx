"use client";

import React from "react";

interface TallyBarProps {
  yes: number;
  no: number;
  abstain: number;
  veto: number;
}

export default function TallyBar({ yes, no, abstain, veto }: TallyBarProps) {
  const total = yes + no + abstain + veto;
  const yesPct = total > 0 ? (yes / total) * 100 : 0;
  const noPct = total > 0 ? (no / total) * 100 : 0;
  const abstainPct = total > 0 ? (abstain / total) * 100 : 0;
  const vetoPct = total > 0 ? (veto / total) * 100 : 0;

  return (
    <div className="space-y-4 font-mono text-xs">
      <div className="h-3.5 w-full bg-gray-900 rounded-full overflow-hidden flex border border-gray-800">
        <div style={{ width: `${yesPct}%` }} className="h-full bg-green-500 transition-all duration-500" title={`Yes: ${yesPct.toFixed(1)}%`} />
        <div style={{ width: `${noPct}%` }} className="h-full bg-red-500 transition-all duration-500" title={`No: ${noPct.toFixed(1)}%`} />
        <div style={{ width: `${abstainPct}%` }} className="h-full bg-gray-500 transition-all duration-500" title={`Abstain: ${abstainPct.toFixed(1)}%`} />
        <div style={{ width: `${vetoPct}%` }} className="h-full bg-amber-600 transition-all duration-500" title={`No With Veto: ${vetoPct.toFixed(1)}%`} />
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="border border-gray-900 bg-gray-950/40 rounded-lg p-3">
          <div className="text-gray-500 uppercase text-[10px] font-bold">Yes</div>
          <div className="text-sm font-bold text-green-400 mt-1">{yesPct.toFixed(2)}%</div>
          <div className="text-[10px] text-gray-600 mt-0.5">{yes.toLocaleString()} CSOV</div>
        </div>

        <div className="border border-gray-900 bg-gray-950/40 rounded-lg p-3">
          <div className="text-gray-500 uppercase text-[10px] font-bold">No</div>
          <div className="text-sm font-bold text-red-400 mt-1">{noPct.toFixed(2)}%</div>
          <div className="text-[10px] text-gray-600 mt-0.5">{no.toLocaleString()} CSOV</div>
        </div>

        <div className="border border-gray-900 bg-gray-950/40 rounded-lg p-3">
          <div className="text-gray-500 uppercase text-[10px] font-bold">Abstain</div>
          <div className="text-sm font-bold text-gray-400 mt-1">{abstainPct.toFixed(2)}%</div>
          <div className="text-[10px] text-gray-600 mt-0.5">{abstain.toLocaleString()} CSOV</div>
        </div>

        <div className="border border-gray-900 bg-gray-950/40 rounded-lg p-3">
          <div className="text-gray-500 uppercase text-[10px] font-bold">No with Veto</div>
          <div className="text-sm font-bold text-amber-500 mt-1">{vetoPct.toFixed(2)}%</div>
          <div className="text-[10px] text-gray-600 mt-0.5">{veto.toLocaleString()} CSOV</div>
        </div>
      </div>
    </div>
  );
}
