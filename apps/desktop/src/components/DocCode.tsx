import { cn } from "@/lib/utils";

interface DocCodeProps {
  children: string;
  className?: string;
}

export function DocCode({ children, className }: DocCodeProps) {
  return (
    <pre
      className={cn(
        "overflow-x-auto rounded-md border bg-muted px-4 py-3 font-mono text-xs leading-relaxed",
        className,
      )}
    >
      <code>{children}</code>
    </pre>
  );
}
