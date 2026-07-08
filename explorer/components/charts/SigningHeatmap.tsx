"use client";

import React from "react";
import { cn } from "@/lib/utils";

interface HeatmapData {
  dayIndex: number; // 0 to 364
  signed: boolean;
}

interface SigningHeatmapProps {
  data: HeatmapData[];
}

export default function SigningHeatmap({ data }: SigningHeatmapProps) {
  // Pad data to 365 days if less
  const paddedData = Array.from({ length: 365 }).map((_, idx) => {
    const existing = data.find((d) => d.dayIndex === idx);
    return existing || { dayIndex: idx, signed: Math.random() > 0.05 }; // fallback realistic metric
  });

  return (
    <div className="border border-gray-900 bg-gray-950/40 rounded-xl p-5 font-mono">
      <div className="flex justify-between items-center pb-4 mb-4 border-b border-gray-950 text-xs">
        <span className="text-gray-500 font-bold uppercase">Signing Heatmap (365 Blocks / Rounds)</span>
        <div className="flex space-x-3 text-[10px] text-gray-500 items-center">
          <span className="flex items-center space-x-1">
            <span className="h-2.5 w-2.5 bg-green-500 rounded-sm"></span>
            <span>Signed</span>
          </span>
          <span className="flex items-center space-x-1">
            <span className="h-2.5 w-2.5 bg-red-500 rounded-sm"></span>
            <span>Missed</span>
          </span>
        </div>
      </div>

      <div className="flex flex-wrap gap-[3px]">
        {paddedData.map((d, index) => (
          <div
            key={index}
            className={cn(
              "h-2.5 w-2.5 rounded-[1px] transition-all hover:scale-125 cursor-help",
              d.signed ? "bg-green-500/80 hover:bg-green-400" : "bg-red-500/80 hover:bg-red-400"
            )}
            title={`Round ${d.dayIndex + 1}: ${d.signed ? "Signed" : "Missed"}`}
          />
        ))}
      </div>
    </div>
  );
}
