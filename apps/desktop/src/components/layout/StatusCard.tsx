import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { HoverTooltip } from "@/components/ui/hover-tooltip";
import { cn } from "@/lib/utils";

interface StatusCardProps {
  title: string;
  description?: string;
  icon: LucideIcon;
  iconLabel?: string;
  badge: ReactNode;
  children: ReactNode;
  className?: string;
  accent?: "default" | "success" | "warning" | "destructive";
}

const accentStyles = {
  default: "bg-muted/80 text-foreground",
  success: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-400",
  warning: "bg-amber-500/10 text-amber-700 dark:text-amber-400",
  destructive: "bg-destructive/10 text-destructive",
};

export function StatusCard({
  title,
  description,
  icon: Icon,
  iconLabel,
  badge,
  children,
  className,
  accent = "default",
}: StatusCardProps) {
  const resolvedIconLabel = iconLabel ?? title;

  return (
    <Card className={cn("min-w-0 overflow-hidden", className)}>
      <CardHeader className="pb-3">
        <div className="flex min-w-0 items-start justify-between gap-3">
          <div className="flex min-w-0 items-start gap-3">
            <HoverTooltip label={resolvedIconLabel}>
              <div
                className={cn(
                  "flex h-9 w-9 shrink-0 items-center justify-center rounded-lg transition-colors duration-300",
                  accentStyles[accent],
                )}
                aria-label={resolvedIconLabel}
              >
                <Icon className="h-4 w-4" aria-hidden />
              </div>
            </HoverTooltip>
            <div className="min-w-0 space-y-1">
              <CardTitle className="text-base">{title}</CardTitle>
              {description && (
                <CardDescription className="break-words">
                  {description}
                </CardDescription>
              )}
            </div>
          </div>
          <div className="shrink-0">{badge}</div>
        </div>
      </CardHeader>
      <CardContent className="min-w-0">{children}</CardContent>
    </Card>
  );
}
