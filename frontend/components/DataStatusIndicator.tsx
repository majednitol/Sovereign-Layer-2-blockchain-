"use client";

import React from "react";

interface DataStatusIndicatorProps {
  status: "live" | "degraded" | "offline";
  lastUpdated: Date | null;
  onRefresh?: () => void;
}

export default function DataStatusIndicator({
  status,
  lastUpdated,
  onRefresh,
}: DataStatusIndicatorProps) {
  const getStatusColor = () => {
    switch (status) {
      case "live":
        return "var(--accent-success)";
      case "degraded":
        return "var(--accent-warning)";
      case "offline":
        return "#ef4444"; // red
    }
  };

  const getStatusText = () => {
    switch (status) {
      case "live":
        return "System Live";
      case "degraded":
        return "API Degraded";
      case "offline":
        return "Connection Offline";
    }
  };

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        background: "rgba(255, 255, 255, 0.02)",
        padding: "0.75rem 1.25rem",
        borderRadius: "10px",
        border: "1px solid var(--border-color)",
        fontSize: "0.875rem",
        width: "100%",
        marginBottom: "1rem",
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: "0.50rem" }}>
        <span
          style={{
            display: "inline-block",
            width: "10px",
            height: "10px",
            borderRadius: "50%",
            backgroundColor: getStatusColor(),
            boxShadow: `0 0 8px ${getStatusColor()}`,
          }}
        ></span>
        <span style={{ fontWeight: 600, color: "var(--text-primary)" }}>
          {getStatusText()}
        </span>
      </div>

      <div style={{ display: "flex", alignItems: "center", gap: "0.75rem", color: "var(--text-secondary)" }}>
        {lastUpdated && (
          <span>
            Updated: {lastUpdated.toLocaleTimeString()}
          </span>
        )}
        {onRefresh && (
          <button
            onClick={onRefresh}
            style={{
              background: "none",
              border: "none",
              color: "var(--accent-primary)",
              cursor: "pointer",
              fontSize: "0.825rem",
              fontWeight: 500,
              padding: "0.2rem 0.5rem",
              borderRadius: "4px",
              display: "flex",
              alignItems: "center",
              gap: "0.25rem",
              transition: "var(--transition)",
            }}
            onMouseOver={(e) => {
              (e.target as HTMLButtonElement).style.backgroundColor = "rgba(99, 102, 241, 0.1)";
            }}
            onMouseOut={(e) => {
              (e.target as HTMLButtonElement).style.backgroundColor = "transparent";
            }}
          >
            🔄 Refresh
          </button>
        )}
      </div>
    </div>
  );
}
