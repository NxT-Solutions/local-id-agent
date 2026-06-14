import { useEffect, useId, useRef, useState } from "react";
import { cn } from "@/lib/utils";

interface MermaidDiagramProps {
  chart: string;
  className?: string;
}

let mermaidInitialized = false;

export function MermaidDiagram({ chart, className }: MermaidDiagramProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const diagramId = useId().replace(/:/g, "");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    void (async () => {
      try {
        const mermaid = (await import("mermaid")).default;

        if (!mermaidInitialized) {
          mermaid.initialize({
            startOnLoad: false,
            theme: "neutral",
            securityLevel: "strict",
            fontFamily: "ui-sans-serif, system-ui, sans-serif",
            sequence: {
              diagramMarginX: 24,
              diagramMarginY: 16,
              actorMargin: 64,
              width: 160,
              height: 48,
              boxMargin: 8,
              boxTextMargin: 8,
              noteMargin: 12,
              messageMargin: 40,
              mirrorActors: true,
            },
          });
          mermaidInitialized = true;
        }

        const { svg } = await mermaid.render(`mermaid-${diagramId}`, chart.trim());

        if (!cancelled && containerRef.current) {
          containerRef.current.innerHTML = svg;
          setError(null);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to render diagram");
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [chart, diagramId]);

  if (error) {
    return (
      <div className="rounded-md border border-destructive/40 bg-destructive/5 px-4 py-3 text-sm text-destructive">
        Diagram error: {error}
      </div>
    );
  }

  return (
    <div
      ref={containerRef}
      className={cn(
        "overflow-x-auto rounded-md border bg-card p-6 [&_svg]:mx-auto [&_svg]:max-w-full",
        className,
      )}
    />
  );
}
