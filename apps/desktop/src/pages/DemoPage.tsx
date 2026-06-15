import { useCallback, useEffect, useRef, useState } from "react";
import {
  checkAgentReadiness,
  fetchChallenge,
  getAgentUrl,
  getBackendUrl,
  signChallenge,
  verifyProof,
} from "@rqc-icu/localid-client";
import type { AuthState } from "@rqc-icu/localid-client";
import {
  CheckCircle2,
  Circle,
  Loader2,
  RefreshCw,
  ShieldCheck,
  User,
  XCircle,
} from "lucide-react";
import { AlertBanner } from "@/components/layout/AlertBanner";
import { CopyField } from "@/components/layout/CopyField";
import { MetadataGrid } from "@/components/layout/MetadataGrid";
import { PageHeader } from "@/components/layout/PageHeader";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ActionFeedbackAnchor } from "@/components/ui/action-feedback";
import { useActionFeedback } from "@/hooks/useActionFeedback";
import { useSpinWhile } from "@/hooks/useSpinWhile";
import { cn } from "@/lib/utils";

const DEFAULT_AUTH_STEP_TIMEOUT_MS = 12000;
const SIGN_CHALLENGE_TIMEOUT_MS = 120_000;
const AGENT_STATUS_POLL_MS = 5000;

const FLOW_STEPS = [
  { id: "challenge", label: "Fetch challenge", endpoint: "POST /localid/challenge" },
  { id: "sign", label: "Sign with agent", endpoint: "POST /sign-challenge" },
  { id: "verify", label: "Verify proof", endpoint: "POST /localid/verify" },
] as const;

type FlowStepId = (typeof FLOW_STEPS)[number]["id"];

function withTimeout<T>(
  promise: Promise<T>,
  label: string,
  timeoutMs = DEFAULT_AUTH_STEP_TIMEOUT_MS,
): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    const timer = window.setTimeout(() => {
      reject(new Error(`${label} timed out after ${timeoutMs}ms`));
    }, timeoutMs);

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

export function DemoPage() {
  const [open, setOpen] = useState(false);
  const [authState, setAuthState] = useState<AuthState>("idle");
  const [activeStep, setActiveStep] = useState<FlowStepId | null>(null);
  const [userName, setUserName] = useState<string | null>(null);
  const [userId, setUserId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [agentReady, setAgentReady] = useState(false);
  const [agentStatusKnown, setAgentStatusKnown] = useState(false);
  const [agentMessage, setAgentMessage] = useState<string | null>(null);
  const [isInitialLoad, setIsInitialLoad] = useState(true);
  const [isManualRefreshing, setIsManualRefreshing] = useState(false);
  const refreshInFlight = useRef(false);
  const hasLoadedRef = useRef(false);
  const refreshSpinning = useSpinWhile(isManualRefreshing);
  const refreshFeedback = useActionFeedback();
  const authFeedback = useActionFeedback();

  const agentUrl = getAgentUrl();
  const backendUrl = getBackendUrl();

  const refreshAgentStatus = useCallback(async () => {
    if (refreshInFlight.current) {
      return null;
    }
    refreshInFlight.current = true;

    try {
      const readiness = await checkAgentReadiness();
      setAgentReady(Boolean(readiness.healthy && readiness.ready));
      setAgentMessage(readiness.error ?? null);
      setAgentStatusKnown(true);
      hasLoadedRef.current = true;
      return readiness;
    } finally {
      refreshInFlight.current = false;
      setIsInitialLoad(false);
    }
  }, []);

  function handleManualRefresh() {
    setIsManualRefreshing(true);
    void refreshAgentStatus()
      .then((readiness) => {
        if (readiness) {
          refreshFeedback.showSuccess("Refreshed");
        } else {
          refreshFeedback.showError("Refresh failed");
        }
      })
      .catch(() => refreshFeedback.showError("Refresh failed"))
      .finally(() => setIsManualRefreshing(false));
  }

  useEffect(() => {
    void refreshAgentStatus();
    const timer = window.setInterval(
      () => void refreshAgentStatus(),
      AGENT_STATUS_POLL_MS,
    );
    return () => window.clearInterval(timer);
  }, [refreshAgentStatus]);

  async function handleAuthenticate() {
    setAuthState("loading");
    setError(null);
    setUserName(null);
    setUserId(null);
    setActiveStep(null);
    setOpen(true);

    try {
      const readiness = await withTimeout(
        refreshAgentStatus().then((result) => result ?? checkAgentReadiness()),
        "Agent readiness",
      );

      if (!readiness.healthy || !readiness.ready) {
        throw new Error(readiness.error ?? "Agent is not ready");
      }

      const origin = window.location.origin;

      setActiveStep("challenge");
      const { challenge } = await withTimeout(
        fetchChallenge(backendUrl),
        "Challenge request",
      );

      setActiveStep("sign");
      const proof = await withTimeout(
        signChallenge({
          challenge,
          backend: backendUrl,
          purpose: "login",
          origin,
        }),
        "Challenge signing",
        SIGN_CHALLENGE_TIMEOUT_MS,
      );

      setActiveStep("verify");
      const result = await withTimeout(
        verifyProof(backendUrl, {
          challenge: proof.challenge,
          backend: backendUrl,
          origin,
          purpose: "login",
          provider: proof.provider,
          algorithm: proof.algorithm,
          signature: proof.signature,
          certificate: proof.certificate ?? "",
          signedAt: proof.signedAt,
        }),
        "Proof verification",
      );

      setUserName(result.user.name);
      setUserId(result.user.id);
      setAuthState("success");
      setActiveStep(null);
      authFeedback.showSuccess("Authenticated");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Authentication failed";
      setError(message);
      setAuthState("error");
      setActiveStep(null);
      authFeedback.showError("Authentication failed");
    }
  }

  function stepStatus(stepId: FlowStepId) {
    if (authState === "success") return "complete";
    if (authState === "error" && activeStep === stepId) return "error";
    if (activeStep === stepId) return "active";
    if (
      authState === "loading" &&
      activeStep &&
      FLOW_STEPS.findIndex((s) => s.id === stepId) <
        FLOW_STEPS.findIndex((s) => s.id === activeStep)
    ) {
      return "complete";
    }
    return "pending";
  }

  return (
    <div className="space-y-8">
      <PageHeader
        title="Auth demo"
        description="End-to-end challenge flow against the mock backend. Start the mock backend on port 8000 before testing."
        actions={
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
              Refresh status
            </Button>
          </ActionFeedbackAnchor>
        }
        status={
          <Badge
            variant={
              isInitialLoad && !agentStatusKnown
                ? "secondary"
                : agentReady
                  ? "success"
                  : "warning"
            }
            className="gap-1.5"
          >
            {isInitialLoad && !agentStatusKnown
              ? "Checking agent…"
              : agentReady
                ? "Agent ready"
                : "Agent not ready"}
          </Badge>
        }
      />

      <div className="grid gap-4 md:grid-cols-2">
        <CopyField label="Agent URL" value={agentUrl} />
        <CopyField label="Backend URL" value={backendUrl} />
      </div>

      {!agentReady && agentStatusKnown && agentMessage && (
        <AlertBanner variant="warning" title="Agent not ready">
          {agentMessage}
        </AlertBanner>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <ShieldCheck className="h-4 w-4" />
            LocalID authentication
          </CardTitle>
          <CardDescription>
            Same flow as the React browser example — challenge, sign, verify.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid gap-2">
            {FLOW_STEPS.map((step, index) => {
              const status = stepStatus(step.id);
              return (
                <div
                  key={step.id}
                  className="flex items-center gap-3 rounded-lg border bg-muted/20 px-3 py-2.5"
                >
                  <div className="flex h-6 w-6 shrink-0 items-center justify-center">
                    {status === "complete" && (
                      <CheckCircle2 className="h-5 w-5 text-emerald-600" />
                    )}
                    {status === "active" && (
                      <Loader2 className="h-5 w-5 animate-spin text-primary" />
                    )}
                    {status === "error" && (
                      <XCircle className="h-5 w-5 text-destructive" />
                    )}
                    {status === "pending" && (
                      <Circle className="h-5 w-5 text-muted-foreground/40" />
                    )}
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-medium">
                      {index + 1}. {step.label}
                    </p>
                    <p className="font-mono text-xs text-muted-foreground">
                      {step.endpoint}
                    </p>
                  </div>
                </div>
              );
            })}
          </div>

          <ActionFeedbackAnchor
            feedback={authFeedback.feedback}
            onOpenChange={authFeedback.onOpenChange}
          >
            <Button
              onClick={() => void handleAuthenticate()}
              disabled={authState === "loading"}
              className="w-full sm:w-auto"
            >
              {authState === "loading" ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Authenticating…
                </>
              ) : (
                "Authenticate with LocalID"
              )}
            </Button>
          </ActionFeedbackAnchor>
        </CardContent>
      </Card>

      {authState === "success" && userName && (
        <Card className="border-emerald-200/60 bg-emerald-50/50 dark:border-emerald-900/40 dark:bg-emerald-950/20">
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-base text-emerald-800 dark:text-emerald-300">
              <User className="h-4 w-4" />
              Authenticated identity
            </CardTitle>
          </CardHeader>
          <CardContent>
            <MetadataGrid
              items={[
                { label: "Name", value: userName },
                { label: "User ID", value: userId ?? "—", mono: true },
                { label: "Backend", value: backendUrl, mono: true, fullWidth: true },
              ]}
            />
          </CardContent>
        </Card>
      )}

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {authState === "loading"
                ? "Authenticating…"
                : authState === "success"
                  ? "Authentication successful"
                  : authState === "error"
                    ? "Authentication failed"
                    : "LocalID demo"}
            </DialogTitle>
            <DialogDescription>
              {authState === "loading" &&
                "Requesting challenge, signing with the local agent, and verifying with the backend. If prompted, enter your PIN in the eID dialog — signing may take up to two minutes."}
              {authState === "success" &&
                userName &&
                `Signed in as ${userName}.`}
              {authState === "error" && error}
            </DialogDescription>
          </DialogHeader>
          {authState === "success" && userName && (
            <MetadataGrid
              items={[
                { label: "Name", value: userName },
                { label: "User ID", value: userId ?? "—", mono: true },
              ]}
            />
          )}
          <div className="flex justify-end">
            <Button variant="outline" onClick={() => setOpen(false)}>
              {authState === "loading" ? "Hide" : "Close"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
