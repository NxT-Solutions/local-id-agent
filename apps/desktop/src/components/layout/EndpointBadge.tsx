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
          "flex min-w-0 max-w-full items-center gap-2 rounded-md border bg-card px-2.5 py-1.5 text-xs transition-opacity duration-300 sm:max-w-xs lg:max-w-sm",
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
        <span className="hidden h-3 w-px shrink-0 bg-border md:block" />
        <code className="min-w-0 flex-1 truncate font-mono" title={url}>
          {url}
        </code>
        <span className="sr-only">{statusLabels[status]}</span>
      </div>
    </HoverTooltip>
  );
}
