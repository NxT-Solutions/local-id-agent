import { useCallback, useEffect, useRef, useState } from "react";
import { fetchHealth, fetchStatus } from "@rqc-icu/localid-client";
import type { HealthResponse, StatusResponse } from "@rqc-icu/localid-client";
import { RefreshCw, RotateCcw } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { restartAgent } from "@/lib/tauri";

const POLL_INTERVAL_MS = 3000;

export function DashboardPage() {
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [restarting, setRestarting] = useState(false);
  const refreshInFlight = useRef(false);

  const refresh = useCallback(async () => {
    if (refreshInFlight.current) {
      return;
    }
    refreshInFlight.current = true;

    try {
      const [healthResult, statusResult] = await Promise.all([
        fetchHealth(),
        fetchStatus(),
      ]);
      setHealth(healthResult);
      setStatus(statusResult);
      setError(null);
    } catch (err) {
      setHealth(null);
      setStatus(null);
      setError(err instanceof Error ? err.message : "Agent unreachable");
    } finally {
      refreshInFlight.current = false;
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    let timer: number | null = null;

    const scheduleNext = () => {
      timer = window.setTimeout(() => {
        void runPoll();
      }, POLL_INTERVAL_MS);
    };

    const runPoll = async () => {
      if (cancelled) return;
      await refresh();
      if (!cancelled) {
        scheduleNext();
      }
    };

    void refresh();
    scheduleNext();

    return () => {
      cancelled = true;
      if (timer !== null) {
        window.clearTimeout(timer);
      }
    };
  }, [refresh]);

  async function handleRestart() {
    setRestarting(true);
    try {
      await restartAgent();
      await new Promise((resolve) => window.setTimeout(resolve, 800));
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Restart failed");
    } finally {
      setRestarting(false);
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
          <p className="text-sm text-muted-foreground">
            Live health and provider status from the bundled agent.
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => void refresh()}>
            <RefreshCw className="h-4 w-4" />
            Refresh
          </Button>
          <Button onClick={() => void handleRestart()} disabled={restarting}>
            <RotateCcw className="h-4 w-4" />
            {restarting ? "Restarting…" : "Restart agent"}
          </Button>
        </div>
      </div>

      {error && (
        <Card className="border-destructive/40">
          <CardContent className="pt-6 text-sm text-destructive">{error}</CardContent>
        </Card>
      )}

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Health</CardTitle>
            <CardDescription>Agent liveness endpoint</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <Badge variant={health?.ok ? "success" : "destructive"}>
              {health?.ok ? "Healthy" : "Unavailable"}
            </Badge>
            <dl className="grid gap-2 text-sm">
              <div className="flex justify-between gap-4">
                <dt className="text-muted-foreground">Name</dt>
                <dd>{health?.name ?? "—"}</dd>
              </div>
              <div className="flex justify-between gap-4">
                <dt className="text-muted-foreground">Version</dt>
                <dd>{health?.version ?? "—"}</dd>
              </div>
            </dl>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Provider</CardTitle>
            <CardDescription>Active signing provider</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <Badge variant={status?.ready ? "success" : "warning"}>
              {status?.ready ? "Ready" : "Not ready"}
            </Badge>
            <dl className="grid gap-2 text-sm">
              <div className="flex justify-between gap-4">
                <dt className="text-muted-foreground">Provider</dt>
                <dd>{status?.provider ?? "—"}</dd>
              </div>
              <div className="flex justify-between gap-4">
                <dt className="text-muted-foreground">Card present</dt>
                <dd>{status ? (status.cardPresent ? "Yes" : "No") : "—"}</dd>
              </div>
            </dl>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
