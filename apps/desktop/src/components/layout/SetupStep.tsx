import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface SetupStepProps {
  step: number;
  title: string;
  description?: string;
  children: ReactNode;
  className?: string;
}

export function SetupStep({
  step,
  title,
  description,
  children,
  className,
}: SetupStepProps) {
  return (
    <section className={cn("relative pl-0 sm:pl-12", className)}>
      <div className="absolute left-0 top-0 hidden sm:flex sm:h-full sm:flex-col sm:items-center">
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full border-2 border-primary bg-background text-sm font-semibold text-primary">
          {step}
        </div>
        <div className="mt-2 w-px flex-1 bg-border" aria-hidden />
      </div>
      <div className="space-y-4">
        <div className="space-y-1">
          <div className="flex items-center gap-2 sm:hidden">
            <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary text-xs font-semibold text-primary-foreground">
              {step}
            </span>
            <h2 className="text-lg font-semibold tracking-tight">{title}</h2>
          </div>
          <h2 className="hidden text-lg font-semibold tracking-tight sm:block">
            {title}
          </h2>
          {description && (
            <p className="text-sm leading-relaxed text-muted-foreground">
              {description}
            </p>
          )}
        </div>
        {children}
      </div>
    </section>
  );
}
