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
const REQUEST_TIMEOUT_MS = 4000;

function withTimeout<T>(promise: Promise<T>, label: string): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    const timer = window.setTimeout(() => {
      reject(new Error(`${label} timed out after ${REQUEST_TIMEOUT_MS}ms`));
    }, REQUEST_TIMEOUT_MS);

    void promise.then(
      (value) => {
        window.clearTimeout(timer);
        resolve(value);
      },
      (error) => {
        window.clearTimeout(timer);
        reject(error);
      },
    );
  });
}

export function DashboardPage() {
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [restarting, setRestarting] = useState(false);
  const [isChecking, setIsChecking] = useState(true);
  const [lastUpdated, setLastUpdated] = useState<string | null>(null);
  const refreshInFlight = useRef(false);

  const refresh = useCallback(async () => {
    if (refreshInFlight.current) {
      return;
    }
    refreshInFlight.current = true;
    setIsChecking(true);

    try {
      const [healthResult, statusResult] = await Promise.all([
        withTimeout(fetchHealth(), "Health check"),
        withTimeout(fetchStatus(), "Status check"),
      ]);
      setHealth(healthResult);
      setStatus(statusResult);
      setError(null);
      setLastUpdated(new Date().toLocaleTimeString());
    } catch (err) {
      setError(err instanceof Error ? err.message : "Agent unreachable");
    } finally {
      refreshInFlight.current = false;
      setIsChecking(false);
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
          <CardContent className="space-y-2 pt-6 text-sm">
            <p className="font-medium text-destructive">Agent is unreachable.</p>
            <p className="text-destructive">{error}</p>
            <p className="text-muted-foreground">
              Check the provider in Settings, then click <strong>Restart agent</strong>.
              If it still fails, reset config at <code>~/Library/Application Support/icu.rqc.localid-agent/config.json</code>.
            </p>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Health</CardTitle>
            <CardDescription>Agent liveness endpoint</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <Badge
              variant={
                health?.ok
                  ? "success"
                  : isChecking
                    ? "secondary"
                    : "destructive"
              }
            >
              {health?.ok
                ? "Healthy"
                : isChecking
                  ? "Checking…"
                  : "Unavailable"}
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
              <div className="flex justify-between gap-4">
                <dt className="text-muted-foreground">Last update</dt>
                <dd>{lastUpdated ?? "—"}</dd>
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
            <Badge
              variant={
                status?.ready
                  ? "success"
                  : isChecking
                    ? "secondary"
                    : "warning"
              }
            >
              {status?.ready
                ? "Ready"
                : isChecking
                  ? "Checking…"
                  : "Not ready"}
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
              {status?.message && !status.ready && (
                <div className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/40 dark:text-amber-100">
                  {status.message}
                </div>
              )}
            </dl>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
