import { HoverTooltip } from "@/components/ui/hover-tooltip";
import { cn } from "@/lib/utils";

type EndpointStatus = "online" | "offline" | "checking" | "unknown";

interface EndpointBadgeProps {
  label: string;
  url: string;
  status: EndpointStatus;
  refreshing?: boolean;
  className?: string;
}

const statusStyles: Record<EndpointStatus, string> = {
  online: "bg-emerald-500",
  offline: "bg-destructive",
  checking: "bg-muted-foreground/50",
  unknown: "bg-muted-foreground/30",
};

const statusLabels: Record<EndpointStatus, string> = {
  online: "Reachable",
  offline: "Unreachable",
  checking: "Checking…",
  unknown: "Unknown",
};

export function EndpointBadge({
  label,
  url,
  status,
  refreshing = false,
  className,
}: EndpointBadgeProps) {
  const tooltipLabel = `${label}: ${url} (${statusLabels[status]})`;

  return (
    <HoverTooltip label={tooltipLabel} side="bottom">
      <div
        className={cn(
          "flex min-w-0 items-center gap-1.5 rounded-md border bg-card px-2 py-1 text-xs transition-opacity duration-300 md:gap-2 md:px-2.5 md:py-1.5 md:max-w-xs lg:max-w-sm",
          refreshing && status !== "checking" && "opacity-80",
          className,
        )}
        aria-label={tooltipLabel}
      >
        <span
          className={cn(
            "h-2 w-2 shrink-0 rounded-full transition-colors duration-300",
            statusStyles[status],
            status === "checking" && "animate-pulse",
            refreshing &&
              status !== "checking" &&
              "ring-2 ring-current/20 ring-offset-1 animate-pulse",
          )}
          aria-hidden
        />
        <span className="shrink-0 font-medium text-muted-foreground">{label}</span>
        <span className="hidden h-3 w-px shrink-0 bg-border md:block" aria-hidden />
        <code
          className="hidden min-w-0 truncate font-mono text-muted-foreground md:block"
          title={url}
        >
          {url}
        </code>
        <span className="sr-only">{statusLabels[status]}</span>
      </div>
    </HoverTooltip>
  );
}
