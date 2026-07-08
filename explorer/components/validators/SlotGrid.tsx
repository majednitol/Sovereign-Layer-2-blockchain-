"use client";

import React from "react";
import { ShieldCheck, ShieldAlert, Zap, Cpu } from "lucide-react";
import Link from "next/link";
import { Badge } from "@/components/ui/Badge";

export interface ValidatorSlot {
  slotIndex: number;
  validatorAddress: string;
  moniker?: string;
  power: number;
  status: "active" | "inactive" | "ejected";
  missedBlocks: number;
  certificationScore: number;
}

interface SlotGridProps {
  slots: ValidatorSlot[];
  loading?: boolean;
}

export default function SlotGrid({ slots, loading = false }: SlotGridProps) {
  if (loading) {
    return (
      <div className="grid grid-cols-2 md:grid-cols-5 lg:grid-cols-6 gap-4">
        {Array.from({ length: 30 }).map((_, i) => (
          <div
            key={i}
            className="h-28 rounded-xl border border-gray-900 bg-gray-950/40 animate-pulse flex flex-col justify-between p-4"
          />
        ))}
      </div>
    );
  }

  // Pre-fill 30 slots to guarantee the layout maintains the visual structure
  const gridSlots = Array.from({ length: 30 }).map((_, index) => {
    const occupant = slots.find((s) => s.slotIndex === index);
    if (occupant) return occupant;
    return {
      slotIndex: index,
      validatorAddress: "",
      moniker: `Slot #${index + 1} (Empty)`,
      power: 0,
      status: "inactive" as const,
      missedBlocks: 0,
      certificationScore: 0,
    };
  });

  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 lg:grid-cols-6 gap-4 font-mono">
      {gridSlots.map((slot) => {
        const isEmpty = !slot.validatorAddress;
        const isActive = slot.status === "active";
        const isEjected = slot.status === "ejected";

        const cardContent = (
          <div className="flex flex-col justify-between h-full">
            <div className="flex items-start justify-between">
              <span className="text-xs font-bold text-gray-500">#{slot.slotIndex + 1}</span>
              {!isEmpty && (
                slot.certificationScore >= 95 ? (
                  <ShieldCheck className="h-4 w-4 text-cyan-400" />
                ) : (
                  <ShieldAlert className="h-4 w-4 text-amber-500" />
                )
              )}
            </div>

            <div className="mt-2">
              <div className="text-xs font-semibold text-white truncate max-w-[120px]">
                {slot.moniker || `Val #${slot.slotIndex}`}
              </div>
              {!isEmpty && (
                <div className="text-[10px] text-gray-500 truncate mt-0.5">
                  {slot.validatorAddress.slice(0, 10)}...
                </div>
              )}
            </div>

            <div className="mt-3 flex items-center justify-between">
              {isEmpty ? (
                <Badge variant="neutral" size="sm">Empty</Badge>
              ) : isActive ? (
                <Badge variant="success" size="sm">Active</Badge>
              ) : isEjected ? (
                <Badge variant="danger" size="sm">Ejected</Badge>
              ) : (
                <Badge variant="warning" size="sm">Offline</Badge>
              )}

              {!isEmpty && (
                <span className="text-[10px] text-gray-400 font-bold">
                  {slot.certificationScore}%
                </span>
              )}
            </div>
          </div>
        );

        const cardClass = `rounded-xl border p-4 transition-all duration-300 ${
          isEmpty
            ? "border-gray-900 bg-gray-950/20 opacity-60"
            : isEjected
            ? "border-red-900/50 bg-red-950/5 hover:border-red-500/30"
            : "border-gray-900 bg-gray-950/60 hover:border-cyan-500/30 hover:shadow-lg hover:shadow-cyan-950/10 cursor-pointer"
        }`;

        if (isEmpty || isEjected) {
          return (
            <div key={slot.slotIndex} className={cardClass}>
              {cardContent}
            </div>
          );
        }

        return (
          <Link href={`/validators/${slot.validatorAddress}`} key={slot.slotIndex} className={cardClass}>
            {cardContent}
          </Link>
        );
      })}
    </div>
  );
}
