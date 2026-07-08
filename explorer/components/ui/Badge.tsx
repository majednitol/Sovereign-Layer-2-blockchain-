import React from "react";
import { cn } from "@/lib/utils";

interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: "success" | "warning" | "danger" | "info" | "neutral";
  size?: "sm" | "md";
}

export function Badge({
  className,
  variant = "neutral",
  size = "md",
  children,
  ...props
}: BadgeProps) {
  const baseStyles = "inline-flex items-center justify-center font-semibold rounded-md border tracking-wide uppercase transition-colors";
  
  const variantStyles = {
    success: "bg-green-950/40 text-green-400 border-green-900/50",
    warning: "bg-amber-950/40 text-amber-400 border-amber-900/50",
    danger: "bg-red-950/40 text-red-400 border-red-900/50",
    info: "bg-cyan-950/40 text-cyan-400 border-cyan-900/50",
    neutral: "bg-gray-900/60 text-gray-400 border-gray-800",
  };

  const sizeStyles = {
    sm: "px-2 py-0.5 text-[10px]",
    md: "px-2.5 py-1 text-xs",
  };

  return (
    <span
      className={cn(baseStyles, variantStyles[variant], sizeStyles[size], className)}
      {...props}
    >
      {children}
    </span>
  );
}
