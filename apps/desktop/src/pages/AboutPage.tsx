import { useCallback, useEffect, useRef, useState } from "react";
import {
  fetchHealth,
  fetchStatus,
} from "@rqc-icu/localid-client";
import {
  Copy,
  ExternalLink,
  FileJson,
  Info,
  Monitor,
  RefreshCw,
  Server,
} from "lucide-react";
import { AlertBanner } from "@/components/layout/AlertBanner";
import { CopyField } from "@/components/layout/CopyField";
import { MetadataGrid, MetadataGridSkeleton } from "@/components/layout/MetadataGrid";
import { PageHeader } from "@/components/layout/PageHeader";
import { StatusCard } from "@/components/layout/StatusCard";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ActionFeedbackAnchor } from "@/components/ui/action-feedback";
import { useActionFeedback } from "@/hooks/useActionFeedback";
import { useSpinWhile } from "@/hooks/useSpinWhile";
import { cn } from "@/lib/utils";
import { copyDiagnostics, getDiagnostics, type DiagnosticsInfo } from "@/lib/tauri";

export function AboutPage() {
  const [info, setInfo] = useState<DiagnosticsInfo | null>(null);
  const [healthOk, setHealthOk] = useState<boolean | null>(null);
  const [providerReady, setProviderReady] = useState<boolean | null>(null);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isInitialLoad, setIsInitialLoad] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const refreshInFlight = useRef(false);
  const hasLoadedRef = useRef(false);
  const refreshSpinning = useSpinWhile(isRefreshing);
  const copyFeedback = useActionFeedback();
  const refreshFeedback = useActionFeedback();

  const refresh = useCallback(async (): Promise<boolean> => {
    if (refreshInFlight.current) {
      return false;
    }
    refreshInFlight.current = true;
    setIsRefreshing(true);

    try {
      const diagnostics = await getDiagnostics();
      setInfo(diagnostics);

      const [health, status] = await Promise.all([
        fetchHealth().catch(() => null),
        fetchStatus().catch(() => null),
      ]);
      if (health !== null) {
        setHealthOk(health.ok);
      } else if (!hasLoadedRef.current) {
        setHealthOk(false);
      }
      if (status !== null) {
        setProviderReady(status.ready);
      } else if (!hasLoadedRef.current) {
        setProviderReady(false);
      }
      setError(null);
      hasLoadedRef.current = true;
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load diagnostics");
      return false;
    } finally {
      refreshInFlight.current = false;
      setIsRefreshing(false);
      setIsInitialLoad(false);
    }
  }, []);

  function handleManualRefresh() {
    void refresh().then((ok) => {
      if (ok) {
        refreshFeedback.showSuccess("Refreshed");
      } else {
        refreshFeedback.showError("Refresh failed");
      }
    });
  }

  useEffect(() => {
    void refresh();
  }, [refresh]);

  async function handleCopy() {
    try {
      const payload = await copyDiagnostics();
      await navigator.clipboard.writeText(payload);
      setCopied(true);
      copyFeedback.showSuccess("Copied");
      window.setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Copy failed";
      setError(message);
      copyFeedback.showError("Copy failed");
    }
  }

  const systemHealthy =
    info?.sidecarRunning && healthOk === true && providerReady === true;
  const awaitingFirstData = isInitialLoad && !info;
  const badgeStaleClass =
    isRefreshing && !awaitingFirstData ? "opacity-80" : undefined;

  return (
    <div className="space-y-8">
      <PageHeader
        title="Diagnostics"
        description="Version details, runtime health, and troubleshooting data for support requests."
        actions={
          <>
            <ActionFeedbackAnchor
              feedback={refreshFeedback.feedback}
              onOpenChange={refreshFeedback.onOpenChange}
            >
              <Button
                variant="outline"
                size="sm"
                onClick={handleManualRefresh}
              >
                <RefreshCw
                  className={cn("h-4 w-4", refreshSpinning && "animate-spin")}
                />
                Refresh
              </Button>
            </ActionFeedbackAnchor>
            <ActionFeedbackAnchor
              feedback={copyFeedback.feedback}
              onOpenChange={copyFeedback.onOpenChange}
            >
              <Button variant="outline" size="sm" onClick={() => void handleCopy()}>
                <Copy className="h-4 w-4" />
                {copied ? "Copied" : "Copy all diagnostics"}
              </Button>
            </ActionFeedbackAnchor>
          </>
        }
        status={
          awaitingFirstData ? (
            <Skeleton className="h-6 w-40 rounded-md" />
          ) : (
            <Badge
              variant={
                systemHealthy
                  ? "success"
                  : healthOk === null
                    ? "secondary"
                    : "warning"
              }
              className={cn("gap-1.5", badgeStaleClass)}
            >
              {systemHealthy
                ? "All systems operational"
                : healthOk === null
                  ? "Loading…"
                  : "Attention needed"}
            </Badge>
          )
        }
      />

      {error && (
        <AlertBanner variant="error" title="Error">
          {error}
        </AlertBanner>
      )}

      <div className="grid gap-4 md:grid-cols-3">
        {awaitingFirstData ? (
          Array.from({ length: 3 }, (_, index) => (
            <Card key={index} className="overflow-hidden">
              <CardHeader className="pb-3">
                <div className="flex items-start justify-between gap-3">
                  <div className="flex items-start gap-3">
                    <Skeleton className="h-9 w-9 shrink-0 rounded-lg" />
                    <div className="space-y-2">
                      <Skeleton className="h-4 w-20" />
                      <Skeleton className="h-3 w-32" />
                    </div>
                  </div>
                  <Skeleton className="h-5 w-16 rounded-md" />
                </div>
              </CardHeader>
              <CardContent>
                <Skeleton className="h-4 w-full" />
              </CardContent>
            </Card>
          ))
        ) : (
          <>
            <StatusCard
              title="Sidecar"
              description="Bundled Go agent process"
              icon={Server}
              accent={info?.sidecarRunning ? "success" : "destructive"}
              badge={
                <Badge
                  variant={info?.sidecarRunning ? "success" : "destructive"}
                  className={badgeStaleClass}
                >
                  {info?.sidecarRunning ? "Running" : "Stopped"}
                </Badge>
              }
            >
              <p className="text-sm text-muted-foreground">
                Managed by the Tauri shell. Restart from Dashboard or after saving
                Settings.
              </p>
            </StatusCard>

            <StatusCard
              title="Agent health"
              description="GET /health"
              icon={Monitor}
              accent={
                healthOk === true
                  ? "success"
                  : healthOk === false
                    ? "destructive"
                    : "default"
              }
              badge={
                <Badge
                  variant={
                    healthOk === true
                      ? "success"
                      : healthOk === false
                        ? "destructive"
                        : "secondary"
                  }
                  className={badgeStaleClass}
                >
                  {healthOk === true
                    ? "Healthy"
                    : healthOk === false
                      ? "Unreachable"
                      : "Checking…"}
                </Badge>
              }
            >
              {info?.agentUrl && (
                <CopyField value={info.agentUrl} className="mt-1" />
              )}
            </StatusCard>

            <StatusCard
              title="Provider"
              description="GET /status"
              icon={FileJson}
              accent={
                providerReady === true
                  ? "success"
                  : providerReady === false
                    ? "warning"
                    : "default"
              }
              badge={
                <Badge
                  variant={
                    providerReady === true
                      ? "success"
                      : providerReady === false
                        ? "warning"
                        : "secondary"
                  }
                  className={badgeStaleClass}
                >
                  {providerReady === true
                    ? "Ready"
                    : providerReady === false
                      ? "Not ready"
                      : "Checking…"}
                </Badge>
              }
            >
              <p className="text-sm text-muted-foreground">
                Check Dashboard for card presence and provider-specific messages.
              </p>
            </StatusCard>
          </>
        )}
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Info className="h-4 w-4" />
            Application details
          </CardTitle>
          <CardDescription>Desktop shell and bundled sidecar metadata</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {awaitingFirstData ? (
            <MetadataGridSkeleton rows={3} />
          ) : (
            <MetadataGrid
              columns={2}
              items={[
                { label: "App version", value: info?.appVersion ?? "—", mono: true },
                { label: "Platform", value: info?.platform ?? "—" },
                {
                  label: "Agent URL",
                  value: info?.agentUrl ?? "—",
                  mono: true,
                  fullWidth: true,
                },
              ]}
            />
          )}
          {info?.configPath && (
            <CopyField label="Config path" value={info.configPath} />
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Resources</CardTitle>
          <CardDescription>
            Reference documentation in the monorepo
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-2 sm:grid-cols-2">
          {[
            { label: "Architecture", path: "docs/architecture.md" },
            { label: "Agent config", path: "docs/agent-config.md" },
            { label: "Desktop guide", path: "docs/desktop.md" },
            { label: "OpenAPI specs", path: "openapi/" },
          ].map((resource) => (
            <div
              key={resource.path}
              className="flex items-center justify-between rounded-lg border bg-muted/30 px-3 py-2.5 text-sm"
            >
              <span>{resource.label}</span>
              <code className="flex items-center gap-1 font-mono text-xs text-muted-foreground">
                {resource.path}
                <ExternalLink className="h-3 w-3 opacity-50" />
              </code>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}
