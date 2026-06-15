import { useCallback, useEffect, useRef, useState } from "react";
import { fetchHealth, getAgentUrl, getBackendUrl } from "@rqc-icu/localid-client";
import { RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { EndpointBadge } from "@/components/layout/EndpointBadge";
import { ActionFeedbackAnchor } from "@/components/ui/action-feedback";
import { useActionFeedback } from "@/hooks/useActionFeedback";
import { useSpinWhile } from "@/hooks/useSpinWhile";
import { appFetch } from "@/lib/fetch";
import { cn } from "@/lib/utils";

const POLL_INTERVAL_MS = 5000;
const REQUEST_TIMEOUT_MS = 3000;

type Reachability = "online" | "offline" | "checking";

async function checkBackendReachable(url: string): Promise<boolean> {
  try {
    await Promise.race([
      appFetch(url, { method: "GET" }),
      new Promise<never>((_, reject) => {
        window.setTimeout(() => reject(new Error("timeout")), REQUEST_TIMEOUT_MS);
      }),
    ]);
    return true;
  } catch {
    return false;
  }
}

export function SystemStatusBar() {
  const [agentStatus, setAgentStatus] = useState<Reachability>("checking");
  const [backendStatus, setBackendStatus] = useState<Reachability>("checking");
  const [lastChecked, setLastChecked] = useState<string | null>(null);
  const [isManualRefreshing, setIsManualRefreshing] = useState(false);
  const [isBackgroundPolling, setIsBackgroundPolling] = useState(false);
  const refreshInFlight = useRef(false);
  const hasLoadedRef = useRef(false);
  const refreshSpinning = useSpinWhile(isManualRefreshing);
  const refreshFeedback = useActionFeedback();

  const refresh = useCallback(async (source: "manual" | "background" = "background") => {
    if (refreshInFlight.current) {
      return;
    }
    refreshInFlight.current = true;
    if (source === "background" && hasLoadedRef.current) {
      setIsBackgroundPolling(true);
    }

    const isFirstLoad = !hasLoadedRef.current;
    if (isFirstLoad) {
      setAgentStatus("checking");
      setBackendStatus("checking");
    }

    const backendUrl = getBackendUrl();

    let agentOk: boolean | null = null;
    try {
      const health = await fetchHealth();
      agentOk = health.ok;
    } catch {
      // Keep last-known agent status on transient failures (restart, poll overlap).
    }

    const backendOk = await checkBackendReachable(backendUrl);

    if (agentOk !== null) {
      setAgentStatus(agentOk ? "online" : "offline");
    } else if (isFirstLoad) {
      setAgentStatus("offline");
    }

    setBackendStatus(backendOk ? "online" : "offline");

    setLastChecked(new Date().toLocaleTimeString());
    hasLoadedRef.current = true;
    refreshInFlight.current = false;
    setIsBackgroundPolling(false);
  }, []);

  function handleManualRefresh() {
    setIsManualRefreshing(true);
    void refresh("manual")
      .then(() => refreshFeedback.showSuccess("Refreshed"))
      .finally(() => setIsManualRefreshing(false));
  }

  useEffect(() => {
    void refresh();
    const timer = window.setInterval(() => void refresh(), POLL_INTERVAL_MS);
    return () => window.clearInterval(timer);
  }, [refresh]);

  return (
    <div className="flex min-w-0 flex-col gap-2 overflow-x-hidden border-b bg-background px-3 py-2 sm:flex-row sm:items-center sm:justify-between sm:gap-3 md:px-4">
      <div className="flex items-center justify-between gap-2 sm:contents">
        <span className="shrink-0 text-xs font-medium uppercase tracking-wide text-muted-foreground">
          Endpoints
        </span>
        <div className="flex shrink-0 items-center gap-2 text-xs text-muted-foreground sm:order-3">
          {lastChecked && (
            <span
              className={cn(
                "max-w-[9rem] truncate whitespace-nowrap sm:max-w-none",
                isBackgroundPolling && "animate-pulse",
              )}
              title={`Updated ${lastChecked}`}
            >
              Updated {lastChecked}
            </span>
          )}
          <ActionFeedbackAnchor
            feedback={refreshFeedback.feedback}
            onOpenChange={refreshFeedback.onOpenChange}
            hoverLabel="Refresh endpoint status"
          >
            <Button
              variant="ghost"
              size="sm"
              className="h-7 px-2"
              onClick={handleManualRefresh}
              aria-label="Refresh endpoint status"
            >
              <RefreshCw
                className={cn("h-3.5 w-3.5", refreshSpinning && "animate-spin")}
              />
            </Button>
          </ActionFeedbackAnchor>
        </div>
      </div>
      <div className="flex min-w-0 flex-wrap items-center gap-2 sm:order-2 sm:min-w-0 sm:flex-1">
        <EndpointBadge
          label="Agent"
          url={getAgentUrl()}
          status={agentStatus}
          refreshing={isBackgroundPolling || isManualRefreshing}
        />
        <EndpointBadge
          label="Backend"
          url={getBackendUrl()}
          status={backendStatus}
          refreshing={isBackgroundPolling || isManualRefreshing}
        />
      </div>
    </div>
  );
}
