import type { ReactNode } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

export interface MetadataItem {
  label: string;
  value: ReactNode;
  mono?: boolean;
  fullWidth?: boolean;
}

interface MetadataGridProps {
  items: MetadataItem[];
  columns?: 1 | 2;
  className?: string;
}

export function MetadataGrid({
  items,
  columns = 1,
  className,
}: MetadataGridProps) {
  return (
    <dl
      className={cn(
        "grid gap-3 text-sm",
        columns === 2 && "sm:grid-cols-2",
        className,
      )}
    >
      {items.map((item) => (
        <div
          key={item.label}
          className={cn(
            "min-w-0 rounded-lg border border-border/60 bg-muted/30 px-3 py-2.5",
            item.fullWidth && "sm:col-span-2",
          )}
        >
          <dt className="text-xs font-medium text-muted-foreground">
            {item.label}
          </dt>
          <dd
            className={cn(
              "mt-1 min-w-0 font-medium",
              item.mono && "font-mono text-xs font-normal",
            )}
          >
            {typeof item.value === "string" ? (
              <span
                className={cn(
                  "block",
                  item.mono ? "truncate" : "break-words",
                )}
                title={item.value}
              >
                {item.value}
              </span>
            ) : (
              item.value
            )}
          </dd>
        </div>
      ))}
    </dl>
  );
}

interface MetadataGridSkeletonProps {
  rows?: number;
  className?: string;
}

export function MetadataGridSkeleton({
  rows = 2,
  className,
}: MetadataGridSkeletonProps) {
  return (
    <div className={cn("grid gap-3", className)}>
      {Array.from({ length: rows }, (_, index) => (
        <div
          key={index}
          className="rounded-lg border border-border/60 bg-muted/30 px-3 py-2.5 space-y-2"
        >
          <Skeleton className="h-3 w-24" />
          <Skeleton className="h-4 w-3/4 max-w-[200px]" />
        </div>
      ))}
    </div>
  );
}
