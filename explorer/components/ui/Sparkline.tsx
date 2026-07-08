"use client";

import React from "react";
import { ResponsiveContainer, AreaChart, Area } from "recharts";

interface SparklineProps {
  data: number[];
  color?: string;
  className?: string;
}

export function Sparkline({ data, color = "#06b6d4", className }: SparklineProps) {
  const chartData = data.map((value, index) => ({ index, value }));

  return (
    <div className={className} style={{ width: "100%", height: "100%" }}>
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={chartData} margin={{ top: 2, right: 2, bottom: 2, left: 2 }}>
          <defs>
            <linearGradient id="sparklineGrad" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={color} stopOpacity={0.25} />
              <stop offset="95%" stopColor={color} stopOpacity={0} />
            </linearGradient>
          </defs>
          <Area
            type="monotone"
            dataKey="value"
            stroke={color}
            strokeWidth={1.5}
            fillOpacity={1}
            fill="url(#sparklineGrad)"
            isAnimationActive={false}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
