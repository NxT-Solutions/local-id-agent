import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import {
  fetchHealth,
  fetchStatus,
  getAgentUrl,
  getBackendUrl,
} from "@rqc-icu/localid-client";
import type { HealthResponse, StatusResponse } from "@rqc-icu/localid-client";
import {
  Activity,
  ArrowRight,
  BookOpen,
  CreditCard,
  Globe,
  HeartPulse,
  PlayCircle,
  RefreshCw,
  RotateCcw,
  Server,
  Settings,
  Shield,
} from "lucide-react";
import { AlertBanner } from "@/components/layout/AlertBanner";
import { CopyField } from "@/components/layout/CopyField";
import { MetadataGrid, MetadataGridSkeleton } from "@/components/layout/MetadataGrid";
import { PageHeader } from "@/components/layout/PageHeader";
import { StatusCard } from "@/components/layout/StatusCard";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { ActionFeedbackAnchor } from "@/components/ui/action-feedback";
import { useAdminLock } from "@/context/AdminLockContext";
import { useActionFeedback } from "@/hooks/useActionFeedback";
import { useSpinWhile } from "@/hooks/useSpinWhile";
import { cn } from "@/lib/utils";
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
  const { unlocked } = useAdminLock();
  const location = useLocation();
  const adminRequired =
    typeof location.state === "object" &&
    location.state !== null &&
    "adminRequired" in location.state &&
    location.state.adminRequired === true;

  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [restarting, setRestarting] = useState(false);
  const [isInitialLoad, setIsInitialLoad] = useState(true);
  const [isManualRefreshing, setIsManualRefreshing] = useState(false);
  const [isBackgroundPolling, setIsBackgroundPolling] = useState(false);
  const [lastUpdated, setLastUpdated] = useState<string | null>(null);
  const refreshInFlight = useRef(false);
  const hasLoadedRef = useRef(false);
  const refreshSpinning = useSpinWhile(isManualRefreshing);
  const restartSpinning = useSpinWhile(restarting);
  const refreshFeedback = useActionFeedback();
  const restartFeedback = useActionFeedback();

  const refresh = useCallback(async (source: "manual" | "background" = "background"): Promise<boolean> => {
    if (refreshInFlight.current) {
      return false;
    }
    refreshInFlight.current = true;
    if (source === "background" && hasLoadedRef.current) {
      setIsBackgroundPolling(true);
    }

    try {
      const [healthResult, statusResult] = await Promise.all([
        withTimeout(fetchHealth(), "Health check"),
        withTimeout(fetchStatus(), "Status check"),
      ]);
      setHealth(healthResult);
      setStatus(statusResult);
      setError(null);
      setLastUpdated(new Date().toLocaleTimeString());
      hasLoadedRef.current = true;
      return true;
    } catch (err) {
      if (!hasLoadedRef.current) {
        setError(err instanceof Error ? err.message : "Agent unreachable");
      }
      return false;
    } finally {
      refreshInFlight.current = false;
      setIsBackgroundPolling(false);
      setIsInitialLoad(false);
    }
  }, []);

  function handleManualRefresh() {
    setIsManualRefreshing(true);
    void refresh("manual")
      .then((ok) => {
        if (ok) {
          refreshFeedback.showSuccess("Refreshed");
        } else {
          refreshFeedback.showError("Refresh failed");
        }
      })
      .finally(() => setIsManualRefreshing(false));
  }

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
      const ok = await refresh();
      if (ok) {
        restartFeedback.showSuccess("Agent restarted");
      } else {
        restartFeedback.showError("Agent restarted but unreachable");
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : "Restart failed";
      setError(message);
      restartFeedback.showError("Restart failed");
    } finally {
      setRestarting(false);
    }
  }

  const overallHealthy = health?.ok && status?.ready;
  const agentUrl = getAgentUrl();
  const backendUrl = getBackendUrl();
  const awaitingFirstData = isInitialLoad && !health && !status;
  const hasLoadedData = !awaitingFirstData;
  const statusRefreshing = restarting || isBackgroundPolling || isManualRefreshing;
  const badgeStaleClass = statusRefreshing ? "opacity-80" : undefined;

  return (
    <div className="min-w-0 space-y-8 overflow-x-hidden">
      <PageHeader
        title="Dashboard"
        description="Live health, provider status, and endpoint summary for the bundled Go agent sidecar."
        actions={
          <>
            <ActionFeedbackAnchor
              feedback={refreshFeedback.feedback}
              onOpenChange={refreshFeedback.onOpenChange}
            >
              <Button variant="outline" size="sm" onClick={handleManualRefresh}>
                <RefreshCw
                  className={cn("h-4 w-4", refreshSpinning && "animate-spin")}
                />
                Refresh
              </Button>
            </ActionFeedbackAnchor>
            <ActionFeedbackAnchor
              feedback={restartFeedback.feedback}
              onOpenChange={restartFeedback.onOpenChange}
            >
              <Button
                size="sm"
                onClick={() => void handleRestart()}
                disabled={restarting}
              >
                <RotateCcw
                  className={cn("h-4 w-4", restartSpinning && "animate-spin-reverse")}
                />
                {restarting ? "Restarting…" : "Restart agent"}
              </Button>
            </ActionFeedbackAnchor>
          </>
        }
        status={
          <div className="flex flex-wrap items-center gap-2">
            <Badge
              variant={
                overallHealthy
                  ? "success"
                  : awaitingFirstData
                    ? "secondary"
                    : error
                      ? "destructive"
                      : "warning"
              }
              className={cn("gap-1.5 px-2.5 py-1", badgeStaleClass)}
            >
              <Activity className="h-3 w-3" />
              {overallHealthy
                ? "System ready"
                : awaitingFirstData
                  ? "Checking status…"
                  : error
                    ? "Agent unreachable"
                    : "Provider not ready"}
            </Badge>
            {lastUpdated && (
              <span
                className={cn(
                  "text-xs text-muted-foreground",
                  isBackgroundPolling && "animate-pulse",
                )}
              >
                Last polled {lastUpdated}
              </span>
            )}
          </div>
        }
      />

      {adminRequired && !unlocked && (
        <AlertBanner variant="warning" title="Admin access required">
          Unlock admin from the sidebar to open settings, setup, or the auth demo.
        </AlertBanner>
      )}

      {error && (
        <AlertBanner variant="error" title="Agent is unreachable">
          <p>{error}</p>
          <p className="mt-2">
            {unlocked ? (
              <>
                Check the provider in Settings, then click{" "}
                <strong>Restart agent</strong>. If it still fails, reset config at{" "}
                <code className="rounded bg-black/5 px-1 py-0.5 font-mono text-[0.75rem] dark:bg-white/10">
                  ~/Library/Application Support/icu.rqc.localid-agent/config.json
                </code>
                .
              </>
            ) : (
              <>
                Click <strong>Restart agent</strong> to retry. If the problem
                persists, contact your administrator.
              </>
            )}
          </p>
        </AlertBanner>
      )}

      <div className="grid min-w-0 gap-4 lg:grid-cols-2">
        {awaitingFirstData ? (
          <>
            {[0, 1].map((index) => (
              <Card key={index} className="overflow-hidden">
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex items-start gap-3">
                      <Skeleton className="h-9 w-9 shrink-0 rounded-lg" />
                      <div className="space-y-2">
                        <Skeleton className="h-4 w-24" />
                        <Skeleton className="h-3 w-40" />
                      </div>
                    </div>
                    <Skeleton className="h-5 w-16 rounded-md" />
                  </div>
                </CardHeader>
                <CardContent>
                  <MetadataGridSkeleton rows={index === 0 ? 3 : 2} />
                </CardContent>
              </Card>
            ))}
          </>
        ) : (
          <>
        <StatusCard
          title="Health"
          description="GET /health — agent liveness"
          icon={HeartPulse}
          accent={
            health?.ok ? "success" : error && !health ? "destructive" : "default"
          }
          badge={
            <Badge
              variant={
                health?.ok
                  ? "success"
                  : health
                    ? "destructive"
                    : hasLoadedData
                      ? "destructive"
                      : "secondary"
              }
              className={badgeStaleClass}
            >
              {health?.ok ? "Healthy" : hasLoadedData ? "Unavailable" : "Checking…"}
            </Badge>
          }
        >
          <MetadataGrid
            items={[
              { label: "Service name", value: health?.name ?? "—" },
              { label: "Version", value: health?.version ?? "—", mono: true },
              {
                label: "Endpoint",
                value: `${agentUrl}/health`,
                mono: true,
                fullWidth: true,
              },
            ]}
          />
        </StatusCard>

        <StatusCard
          title="Provider"
          description="GET /status — signing readiness"
          icon={CreditCard}
          accent={
            status?.ready ? "success" : status && !status.ready ? "warning" : "default"
          }
          badge={
            <Badge
              variant={
                status?.ready
                  ? "success"
                  : status
                    ? "warning"
                    : hasLoadedData
                      ? "warning"
                      : "secondary"
              }
              className={badgeStaleClass}
            >
              {status?.ready
                ? "Ready"
                : hasLoadedData
                  ? "Not ready"
                  : "Checking…"}
            </Badge>
          }
        >
          <MetadataGrid
            items={[
              { label: "Provider", value: status?.provider ?? "—", mono: true },
              {
                label: "Card present",
                value: status
                  ? status.cardPresent
                    ? "Yes"
                    : "No"
                  : "—",
              },
            ]}
          />
          {status?.message && !status.ready && (
            <AlertBanner variant="warning" className="mt-3">
              {status.message}
            </AlertBanner>
          )}
        </StatusCard>
          </>
        )}
      </div>

      <Card className="surface-elevated min-w-0 overflow-hidden">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Globe className="h-4 w-4 shrink-0" />
            Connection topology
          </CardTitle>
          <CardDescription>
            Browser frontends call your backend for challenges, then the local
            agent to sign — verify happens on your backend.
          </CardDescription>
        </CardHeader>
        <CardContent className="min-w-0 space-y-4">
          <div className="flex min-w-0 flex-col items-stretch gap-3 lg:flex-row lg:items-center lg:justify-center">
            <div className="min-w-0 flex-1 rounded-lg border bg-muted/30 p-4 text-center">
              <Server className="mx-auto h-5 w-5 text-muted-foreground" />
              <p className="mt-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                Backend
              </p>
              <p
                className="mt-1 truncate font-mono text-xs"
                title={backendUrl}
              >
                {backendUrl}
              </p>
              <p className="mt-2 text-[0.6875rem] text-muted-foreground">
                POST /localid/challenge · /verify
              </p>
            </div>
            <div className="flex shrink-0 justify-center px-2">
              <ArrowRight className="h-4 w-4 rotate-90 text-muted-foreground lg:rotate-0" />
            </div>
            <div className="min-w-0 flex-1 rounded-lg border border-primary/20 bg-primary/5 p-4 text-center">
              <Shield className="mx-auto h-5 w-5 text-primary" />
              <p className="mt-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                Agent
              </p>
              <p className="mt-1 truncate font-mono text-xs" title={agentUrl}>
                {agentUrl}
              </p>
              <p className="mt-2 text-[0.6875rem] text-muted-foreground">
                POST /sign-challenge
              </p>
            </div>
          </div>
          <CopyField label="Agent base URL" value={agentUrl} />
        </CardContent>
      </Card>

      {unlocked && (
        <Card className="min-w-0 overflow-hidden">
          <CardHeader className="pb-3">
            <CardTitle className="text-base">Quick actions</CardTitle>
            <CardDescription>Common next steps during integration</CardDescription>
          </CardHeader>
          <CardContent className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
            {[
              {
                to: "/settings",
                icon: Settings,
                label: "Configure allowlists",
                hint: "Origins & backends",
              },
              {
                to: "/setup",
                icon: BookOpen,
                label: "Integration guide",
                hint: "API contracts & flow",
              },
              {
                to: "/demo",
                icon: PlayCircle,
                label: "Run auth demo",
                hint: "End-to-end test",
              },
            ].map((action) => (
              <Link
                key={action.to}
                to={action.to}
                className="group flex min-w-0 cursor-pointer items-start gap-3 rounded-lg border bg-card p-3 transition-colors hover:border-primary/30 hover:bg-accent/50"
              >
                <action.icon className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground group-hover:text-primary" />
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium">{action.label}</p>
                  <p className="truncate text-xs text-muted-foreground">
                    {action.hint}
                  </p>
                </div>
              </Link>
            ))}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
