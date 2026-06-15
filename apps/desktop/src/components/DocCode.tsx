import { useState } from "react";
import { Check, Copy } from "lucide-react";
import { ActionFeedbackAnchor } from "@/components/ui/action-feedback";
import { Button } from "@/components/ui/button";
import { useActionFeedback } from "@/hooks/useActionFeedback";
import { cn } from "@/lib/utils";

interface DocCodeProps {
  children: string;
  className?: string;
  title?: string;
}

export function DocCode({ children, className, title }: DocCodeProps) {
  const [copied, setCopied] = useState(false);
  const copyFeedback = useActionFeedback();

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(children);
      setCopied(true);
      copyFeedback.showSuccess("Copied");
      window.setTimeout(() => setCopied(false), 2000);
    } catch {
      copyFeedback.showError("Copy failed");
    }
  }

  return (
    <div className={cn("overflow-hidden rounded-lg border bg-muted/40", className)}>
      <div className="flex items-center justify-between gap-2 border-b bg-muted/60 px-3 py-1.5">
        <span className="text-xs font-medium text-muted-foreground">
          {title ?? "Snippet"}
        </span>
        <ActionFeedbackAnchor
          feedback={copyFeedback.feedback}
          onOpenChange={copyFeedback.onOpenChange}
          hoverLabel="Copy snippet"
        >
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="h-7 px-2 text-xs"
            onClick={() => void handleCopy()}
            aria-label="Copy snippet"
          >
            {copied ? (
              <>
                <Check className="h-3 w-3 text-emerald-600" />
                Copied
              </>
            ) : (
              <>
                <Copy className="h-3 w-3" />
                Copy
              </>
            )}
          </Button>
        </ActionFeedbackAnchor>
      </div>
      <pre className="overflow-x-auto px-4 py-3 font-mono text-xs leading-relaxed">
        <code>{children}</code>
      </pre>
    </div>
  );
}
