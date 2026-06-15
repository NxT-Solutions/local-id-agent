import { useState } from "react";
import { Check, Copy } from "lucide-react";
import { ActionFeedbackAnchor } from "@/components/ui/action-feedback";
import { Button } from "@/components/ui/button";
import { HoverTooltip } from "@/components/ui/hover-tooltip";
import { useActionFeedback } from "@/hooks/useActionFeedback";
import { cn } from "@/lib/utils";

interface CopyFieldProps {
  value: string;
  label?: string;
  className?: string;
  mono?: boolean;
}

export function CopyField({
  value,
  label,
  className,
  mono = true,
}: CopyFieldProps) {
  const [copied, setCopied] = useState(false);
  const copyFeedback = useActionFeedback();

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      copyFeedback.showSuccess("Copied");
      window.setTimeout(() => setCopied(false), 2000);
    } catch {
      copyFeedback.showError("Copy failed");
    }
  }

  return (
    <div className={cn("min-w-0 space-y-1.5", className)}>
      {label && (
        <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {label}
        </span>
      )}
      <div className="flex min-w-0 items-stretch gap-1 rounded-lg border bg-muted/50">
        <HoverTooltip label={value} side="top">
          <code
            className={cn(
              "block min-w-0 flex-1 truncate px-3 py-2 text-xs leading-relaxed",
              mono && "font-mono",
            )}
            title={value}
          >
            {value}
          </code>
        </HoverTooltip>
        <ActionFeedbackAnchor
          feedback={copyFeedback.feedback}
          onOpenChange={copyFeedback.onOpenChange}
          hoverLabel="Copy to clipboard"
        >
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="shrink-0 rounded-l-none border-l px-2.5"
            onClick={() => void handleCopy()}
            aria-label={copied ? "Copied" : "Copy to clipboard"}
          >
            {copied ? (
              <Check className="h-3.5 w-3.5 text-emerald-600" />
            ) : (
              <Copy className="h-3.5 w-3.5" />
            )}
          </Button>
        </ActionFeedbackAnchor>
      </div>
    </div>
  );
}
