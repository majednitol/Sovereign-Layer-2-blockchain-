import React from "react";
import { cn } from "@/lib/utils";

interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  title?: string;
  description?: string;
  icon?: React.ReactNode;
  headerAction?: React.ReactNode;
}

export function Card({
  className,
  title,
  description,
  icon,
  headerAction,
  children,
  ...props
}: CardProps) {
  return (
    <div
      className={cn(
        "rounded-xl border border-gray-800 bg-gray-950/65 backdrop-blur-md p-6 shadow-lg shadow-black/40 hover:border-cyan-500/20 transition-all duration-300",
        className
      )}
      {...props}
    >
      {(title || icon || headerAction) && (
        <div className="flex items-start justify-between pb-4 mb-4 border-b border-gray-900">
          <div className="flex items-center space-x-3">
            {icon && <div className="text-cyan-400 p-1.5 bg-cyan-950/40 rounded-lg border border-cyan-900/30">{icon}</div>}
            <div>
              {title && <h3 className="text-sm font-semibold text-gray-400 uppercase tracking-wider">{title}</h3>}
              {description && <p className="text-xs text-gray-500 mt-0.5">{description}</p>}
            </div>
          </div>
          {headerAction && <div>{headerAction}</div>}
        </div>
      )}
      {children}
    </div>
  );
}
