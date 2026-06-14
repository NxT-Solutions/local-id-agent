import { useEffect, useState } from "react";
import { Copy, Info } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { copyDiagnostics, getDiagnostics, type DiagnosticsInfo } from "@/lib/tauri";

export function AboutPage() {
  const [info, setInfo] = useState<DiagnosticsInfo | null>(null);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    void getDiagnostics()
      .then(setInfo)
      .catch((err) =>
        setError(err instanceof Error ? err.message : "Failed to load diagnostics"),
      );
  }, []);

  async function handleCopy() {
    try {
      const payload = await copyDiagnostics();
      await navigator.clipboard.writeText(payload);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Copy failed");
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Diagnostics</h1>
        <p className="text-sm text-muted-foreground">
          Version details and troubleshooting information.
        </p>
      </div>

      {error && (
        <Card className="border-destructive/40">
          <CardContent className="pt-6 text-sm text-destructive">{error}</CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Info className="h-5 w-5" />
            About LocalID Agent
          </CardTitle>
          <CardDescription>Desktop shell and bundled sidecar</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4 text-sm">
          <div className="flex items-center justify-between gap-4">
            <span className="text-muted-foreground">App version</span>
            <span>{info?.appVersion ?? "—"}</span>
          </div>
          <div className="flex items-center justify-between gap-4">
            <span className="text-muted-foreground">Platform</span>
            <span>{info?.platform ?? "—"}</span>
          </div>
          <div className="flex items-center justify-between gap-4">
            <span className="text-muted-foreground">Agent URL</span>
            <span>{info?.agentUrl ?? "—"}</span>
          </div>
          <div className="flex items-center justify-between gap-4">
            <span className="text-muted-foreground">Sidecar running</span>
            <Badge variant={info?.sidecarRunning ? "success" : "destructive"}>
              {info?.sidecarRunning ? "Yes" : "No"}
            </Badge>
          </div>
          <div className="space-y-1">
            <span className="text-muted-foreground">Config path</span>
            <p className="break-all rounded-md bg-muted px-3 py-2 font-mono text-xs">
              {info?.configPath ?? "—"}
            </p>
          </div>
          <Button variant="outline" onClick={() => void handleCopy()}>
            <Copy className="h-4 w-4" />
            {copied ? "Copied" : "Copy diagnostics"}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
