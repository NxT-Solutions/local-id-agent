import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

type AlertVariant = "info" | "success" | "warning" | "error";

interface AlertBannerProps {
  variant?: AlertVariant;
  icon?: LucideIcon;
  title?: string;
  children: ReactNode;
  className?: string;
}

const variantStyles: Record<AlertVariant, string> = {
  info: "border-border/80 bg-muted/50 text-foreground",
  success:
    "border-emerald-200/80 bg-emerald-50 text-emerald-950 dark:border-emerald-900/40 dark:bg-emerald-950/30 dark:text-emerald-100",
  warning:
    "border-amber-200/80 bg-amber-50 text-amber-950 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-100",
  error:
    "border-destructive/30 bg-destructive/5 text-destructive dark:bg-destructive/10",
};

export function AlertBanner({
  variant = "info",
  icon: Icon,
  title,
  children,
  className,
}: AlertBannerProps) {
  return (
    <div
      className={cn(
        "rounded-lg border px-4 py-3 text-sm leading-relaxed",
        variantStyles[variant],
        className,
      )}
      role={variant === "error" ? "alert" : "status"}
    >
      <div className="flex gap-3">
        {Icon && (
          <Icon className="mt-0.5 h-4 w-4 shrink-0 opacity-80" aria-hidden />
        )}
        <div className="min-w-0 space-y-1">
          {title && <p className="font-medium">{title}</p>}
          <div className="text-[0.8125rem] leading-relaxed opacity-90">
            {children}
          </div>
        </div>
      </div>
    </div>
  );
}
