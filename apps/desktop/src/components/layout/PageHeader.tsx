import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface PageHeaderProps {
  title: string;
  description?: ReactNode;
  actions?: ReactNode;
  status?: ReactNode;
  className?: string;
}

export function PageHeader({
  title,
  description,
  actions,
  status,
  className,
}: PageHeaderProps) {
  return (
    <div className={cn("min-w-0 space-y-4 overflow-x-hidden", className)}>
      <div className="flex min-w-0 flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0 space-y-1.5">
          <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
          {description && (
            <div className="max-w-2xl text-sm leading-relaxed text-muted-foreground">
              {description}
            </div>
          )}
        </div>
        {actions && (
          <div className="flex min-w-0 shrink-0 flex-wrap items-center gap-2">
            {actions}
          </div>
        )}
      </div>
      {status}
    </div>
  );
}
