import type { ReactNode } from "react";
import type { ActionFeedbackState } from "@/hooks/useActionFeedback";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

interface ActionFeedbackAnchorProps {
  feedback: ActionFeedbackState | null;
  onOpenChange?: (open: boolean) => void;
  hoverLabel?: string;
  children: ReactNode;
  className?: string;
  placement?: "top" | "bottom";
}

export function ActionFeedbackAnchor({
  feedback,
  onOpenChange,
  hoverLabel,
  children,
  className,
  placement = "top",
}: ActionFeedbackAnchorProps) {
  const isFeedbackOpen = feedback !== null;
  const tooltipContent = feedback?.message ?? hoverLabel;

  return (
    <Tooltip
      open={isFeedbackOpen ? true : undefined}
      onOpenChange={onOpenChange}
      delayDuration={isFeedbackOpen ? 0 : undefined}
    >
      <TooltipTrigger asChild>
        <span className={cn("inline-flex", className)}>{children}</span>
      </TooltipTrigger>
      {tooltipContent && (
        <TooltipContent
          side={placement}
          className={cn(
            "px-2 py-0.5 text-[11px] font-normal leading-tight",
            feedback?.variant === "error" &&
              "border-destructive/20 bg-destructive/10 text-destructive",
          )}
        >
          <p>{tooltipContent}</p>
        </TooltipContent>
      )}
    </Tooltip>
  );
}
