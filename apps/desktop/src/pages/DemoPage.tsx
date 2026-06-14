import { useState } from "react";
import {
  checkAgentReadiness,
  fetchChallenge,
  getBackendUrl,
  signChallenge,
  verifyProof,
} from "@rqc-icu/localid-client";
import type { AuthState } from "@rqc-icu/localid-client";
import { ShieldCheck } from "lucide-react";
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

export function DemoPage() {
  const [open, setOpen] = useState(false);
  const [authState, setAuthState] = useState<AuthState>("idle");
  const [userName, setUserName] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [agentReady, setAgentReady] = useState(false);

  async function handleAuthenticate() {
    setAuthState("loading");
    setError(null);
    setUserName(null);
    setOpen(true);

    try {
      const readiness = await checkAgentReadiness();
      setAgentReady(Boolean(readiness.healthy && readiness.ready));

      if (!readiness.healthy || !readiness.ready) {
        throw new Error(readiness.error ?? "Agent is not ready");
      }

      const origin = window.location.origin;
      const backend = getBackendUrl();
      const { challenge } = await fetchChallenge(backend);
      const proof = await signChallenge({
        challenge,
        backend,
        purpose: "login",
        origin,
      });

      const result = await verifyProof(backend, {
        challenge: proof.challenge,
        backend,
        origin,
        purpose: "login",
        provider: proof.provider,
        algorithm: proof.algorithm,
        signature: proof.signature,
        certificate: proof.certificate ?? "",
        signedAt: proof.signedAt,
      });

      setUserName(result.user.name);
      setAuthState("success");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Authentication failed";
      setError(message);
      setAuthState("error");
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Auth demo</h1>
        <p className="text-sm text-muted-foreground">
          End-to-end challenge flow against{" "}
          <code className="rounded bg-muted px-1 py-0.5 text-xs">
            {getBackendUrl()}
          </code>
          . Start the mock backend before testing.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <ShieldCheck className="h-5 w-5" />
            LocalID authentication
          </CardTitle>
          <CardDescription>
            Same flow as the React browser example, using a dialog for feedback.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Badge variant={agentReady ? "success" : "secondary"}>
            {agentReady ? "Agent ready" : "Agent status unknown"}
          </Badge>
          <Button onClick={() => void handleAuthenticate()}>
            Authenticate with LocalID
          </Button>
        </CardContent>
      </Card>

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
                "Requesting challenge, signing with the local agent, and verifying with the backend."}
              {authState === "success" &&
                userName &&
                `Signed in as ${userName}.`}
              {authState === "error" && error}
            </DialogDescription>
          </DialogHeader>
          {authState !== "loading" && (
            <div className="flex justify-end">
              <Button variant="outline" onClick={() => setOpen(false)}>
                Close
              </Button>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
