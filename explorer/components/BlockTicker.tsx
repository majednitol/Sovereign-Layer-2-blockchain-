"use client";

import { useEffect, useState } from "react";
import { Blocks } from "lucide-react";

interface BlockTickerProps {
  apiBase?: string;
  refreshIntervalMs?: number;
}

export default function BlockTicker({
  apiBase = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082",
  refreshIntervalMs = 3000,
}: BlockTickerProps) {
  const [height, setHeight] = useState<number | null>(null);
  const [time, setTime] = useState<string>("");
  const [delta, setDelta] = useState<string>("");

  useEffect(() => {
    let lastTime: number | null = null;
    let timer: NodeJS.Timeout;
    let mounted = true;

    const tick = async () => {
      try {
        const res = await fetch(`${apiBase}/api/rest/v1/explorer/blocks?pagination.limit=1`);
        if (!res.ok) return;
        const data = await res.json();
        const block = data.blocks?.[0];
        if (!block || !mounted) return;

        const h = Number(block.height);
        const t = new Date(block.time).getTime();

        setHeight(h);
        setTime(new Date(block.time).toLocaleTimeString());

        if (lastTime !== null) {
          const diffSec = (t - lastTime) / 1000;
          setDelta(`${diffSec.toFixed(1)}s ago`);
        } else {
          setDelta(`${((Date.now() - t) / 1000).toFixed(1)}s ago`);
        }
        lastTime = t;
      } catch {
        // silent: network failure is non-fatal
      }
    };

    tick();
    timer = setInterval(tick, refreshIntervalMs);
    return () => {
      mounted = false;
      clearInterval(timer);
    };
  }, [apiBase, refreshIntervalMs]);

  return (
    <div className="flex items-center gap-3 bg-gray-950 border border-gray-900 rounded-lg px-4 py-2.5 shadow">
      <Blocks className="h-5 w-5 text-blue-500" />
      <div className="flex flex-col">
        <span className="text-[10px] uppercase font-bold text-gray-500 tracking-wider">
          Latest Block
        </span>
        <span className="text-sm font-bold text-white font-mono">
          {height !== null ? `#${height.toLocaleString()}` : "—"}
        </span>
      </div>
      {time && (
        <div className="ml-4 flex flex-col">
          <span className="text-[10px] uppercase font-bold text-gray-500 tracking-wider">
            Time
          </span>
          <span className="text-xs text-gray-300">{time}</span>
        </div>
      )}
      {delta && (
        <div className="ml-4 flex flex-col">
          <span className="text-[10px] uppercase font-bold text-gray-500 tracking-wider">
            Block time
          </span>
          <span className="text-xs text-green-400">{delta}</span>
        </div>
      )}
    </div>
  );
}
