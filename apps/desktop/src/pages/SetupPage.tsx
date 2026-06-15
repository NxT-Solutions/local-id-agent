import {
  BookOpen,
  CheckCircle2,
  Globe,
  KeyRound,
  Layers,
  Terminal,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { DocCode } from "@/components/DocCode";
import { MermaidDiagram } from "@/components/MermaidDiagram";
import { CopyField } from "@/components/layout/CopyField";
import { PageHeader } from "@/components/layout/PageHeader";
import { SetupStep } from "@/components/layout/SetupStep";
import {
  AUTH_FLOW_DIAGRAM,
  INTEGRATION_OVERVIEW_DIAGRAM,
} from "@/lib/setup-diagrams";
import { getAgentUrl } from "@rqc-icu/localid-client";

const CHECKLIST = [
  "Agent running (Dashboard shows Healthy + Ready)",
  "Frontend origin added to Allowed origins in Settings",
  "Backend URL added to Allowed backends in Settings",
  "Backend implements POST /localid/challenge and POST /localid/verify",
  "Backend verifies canonical JSON + RS256 signature",
  "Frontend uses window.location.origin for the origin field",
  "CORS on your backend allows your frontend origin",
];

export function SetupPage() {
  const agentUrl = getAgentUrl();

  return (
    <div className="space-y-10">
      <PageHeader
        title="Setup guide"
        description="Connect your backend API and frontend to the LocalID Agent. The agent signs challenges locally — it does not issue login tokens or sessions."
      />

      <Card className="surface-elevated overflow-hidden">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Layers className="h-4 w-4" />
            Architecture overview
          </CardTitle>
          <CardDescription>
            Three components working together — agent, backend, frontend
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid gap-3 md:grid-cols-3">
            {[
              {
                step: "1",
                title: "LocalID Agent",
                body: (
                  <>
                    Bundled sidecar on{" "}
                    <code className="font-mono text-xs">{agentUrl}</code>.
                    Signs challenges with smartcard, eID, or mock provider.
                  </>
                ),
              },
              {
                step: "2",
                title: "Your backend API",
                body: "Issues one-time challenges and verifies signatures. You own sessions and user login.",
              },
              {
                step: "3",
                title: "Your frontend",
                body: "Web app that calls backend → agent → backend with the cryptographic proof.",
              },
            ].map((item) => (
              <div
                key={item.step}
                className="relative rounded-lg border bg-muted/20 p-4 pt-8"
              >
                <span className="absolute left-3 top-3 flex h-6 w-6 items-center justify-center rounded-full bg-primary text-xs font-semibold text-primary-foreground">
                  {item.step}
                </span>
                <p className="font-medium">{item.title}</p>
                <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                  {item.body}
                </p>
              </div>
            ))}
          </div>
          <MermaidDiagram
            chart={INTEGRATION_OVERVIEW_DIAGRAM}
            className="bg-muted/10"
          />
        </CardContent>
      </Card>

      <section className="space-y-4">
        <div className="space-y-1">
          <h2 className="text-lg font-semibold tracking-tight">Auth flow</h2>
          <p className="text-sm text-muted-foreground">
            Request sequence between frontend, backend, and local agent.
          </p>
        </div>
        <MermaidDiagram chart={AUTH_FLOW_DIAGRAM} className="surface-elevated" />
      </section>

      <Separator />

      <SetupStep
        step={1}
        title="Configure the agent"
        description="Open Settings in this app. Two allowlists must include your exact URLs."
      >
        <CopyField label="Default agent URL" value={agentUrl} />
        <div className="grid gap-4 md:grid-cols-2">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="flex items-center gap-2 text-base">
                <Globe className="h-4 w-4" />
                Allowed origins
              </CardTitle>
              <CardDescription>
                Browser frontends that may call the agent
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3 text-sm text-muted-foreground">
              <DocCode title="Examples">{`http://localhost:5173
http://localhost:5174
https://app.example.com
tauri://localhost`}</DocCode>
              <p>
                Must match the browser <code>Origin</code> header and the{" "}
                <code>origin</code> field in the sign request.
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="flex items-center gap-2 text-base">
                <KeyRound className="h-4 w-4" />
                Allowed backends
              </CardTitle>
              <CardDescription>
                APIs whose challenges may be signed
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3 text-sm text-muted-foreground">
              <DocCode title="Examples">{`http://localhost:8000
https://api.example.com`}</DocCode>
              <p>
                Must match the <code>backend</code> field in{" "}
                <code>POST /sign-challenge</code> exactly.
              </p>
            </CardContent>
          </Card>
        </div>
        <p className="text-sm text-muted-foreground">
          After saving, the agent restarts automatically. Use the Dashboard to
          confirm health and provider status.
        </p>
      </SetupStep>

      <Separator />

      <SetupStep
        step={2}
        title="Backend API contract"
        description="Your backend must expose two endpoints. The mock backend implements this for local development."
      >
        <Card>
          <CardHeader className="pb-2">
            <div className="flex items-center gap-2">
              <Badge variant="secondary">POST</Badge>
              <CardTitle className="text-base">/localid/challenge</CardTitle>
            </div>
            <CardDescription>Issue a one-time challenge</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Generate a random base64url string (32 bytes recommended). Store
              it server-side with a short TTL (60 seconds). One-time use — delete
              after successful verify.
            </p>
            <DocCode title="Response 200">{`{
  "challenge": "xK9mP2vQ8nR4wL6jH3fT1yU5bN0cA7dE"
}`}</DocCode>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <div className="flex items-center gap-2">
              <Badge variant="secondary">POST</Badge>
              <CardTitle className="text-base">/localid/verify</CardTitle>
            </div>
            <CardDescription>Validate proof and log the user in</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Accept the agent response plus context fields. Verify challenge
              exists, rebuild the canonical signed payload, check RSA-SHA256
              signature against the certificate, then map identity to a user.
            </p>
            <DocCode title="Request & response">{`// Request
{
  "challenge": "...",
  "backend": "https://api.example.com",
  "origin": "https://app.example.com",
  "purpose": "login",
  "provider": "mock",
  "algorithm": "RS256",
  "signature": "...",
  "certificate": "...",
  "signedAt": "2026-06-14T12:00:00Z"
}

// Response 200
{
  "success": true,
  "user": { "id": "...", "name": "..." }
}`}</DocCode>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-base">
              Canonical payload (for verification)
            </CardTitle>
            <CardDescription>
              The agent signs this JSON — not the raw challenge alone
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Keys in alphabetical order, compact JSON, no extra whitespace. Use{" "}
              <code>signedAt</code> from the agent response as{" "}
              <code>timestamp</code>.
            </p>
            <DocCode title="Signed payload">{`{"backend":"https://api.example.com","challenge":"...","origin":"https://app.example.com","purpose":"login","timestamp":"2026-06-14T12:00:00Z"}`}</DocCode>
            <p className="text-sm text-muted-foreground">
              Verify with RSA PKCS#1 v1.5 + SHA-256 (RS256).{" "}
              <code>signature</code> is base64url; <code>certificate</code> is
              standard base64 DER.
            </p>
          </CardContent>
        </Card>
      </SetupStep>

      <Separator />

      <SetupStep
        step={3}
        title="Agent API (localhost)"
        description="Only callable from allowed browser origins (CORS)."
      >
        <CopyField label="Base URL" value={agentUrl} />

        <Card>
          <CardHeader className="pb-2">
            <div className="flex items-center gap-2">
              <Badge variant="outline">POST</Badge>
              <CardTitle className="text-base">/sign-challenge</CardTitle>
            </div>
          </CardHeader>
          <CardContent className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Required headers: <code>Content-Type: application/json</code>,{" "}
              <code>Origin</code> (set automatically by the browser).
            </p>
            <DocCode title="Request & response">{`// Request
{
  "challenge": "<from your backend>",
  "backend": "https://api.example.com",
  "purpose": "login",
  "origin": "https://app.example.com"
}

// Response 200
{
  "provider": "mock",
  "algorithm": "RS256",
  "challenge": "...",
  "signature": "...",
  "certificate": "...",
  "signedAt": "2026-06-14T12:00:00Z"
}`}</DocCode>
            <p className="text-sm text-muted-foreground">
              <code>purpose</code> must be <code>"login"</code> today. Unknown
              origin or backend → 403.
            </p>
          </CardContent>
        </Card>
      </SetupStep>

      <Separator />

      <SetupStep
        step={4}
        title="Frontend setup (React)"
        description="Use @rqc-icu/localid-client from this monorepo, or mirror the same fetch calls."
      >
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-base">
              <Terminal className="h-4 w-4" />
              Install & configure
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <DocCode title="Shell">{`# From repo root
pnpm install

# Environment (.env)
VITE_AGENT_URL=http://127.0.0.1:17443
VITE_BACKEND_URL=https://api.example.com

# Run browser demo
pnpm run dev:react   # http://localhost:5173`}</DocCode>
            <p className="text-sm text-muted-foreground">
              Add your frontend origin to agent Allowed origins and your API URL
              to Allowed backends before testing.
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-base">Auth handler (minimal)</CardTitle>
          </CardHeader>
          <CardContent>
            <DocCode title="TypeScript">{`import {
  fetchChallenge,
  signChallenge,
  verifyProof,
  getBackendUrl,
} from "@rqc-icu/localid-client";

async function authenticate() {
  const origin = window.location.origin;
  const backend = getBackendUrl();

  const { challenge } = await fetchChallenge(backend);
  const proof = await signChallenge({
    challenge,
    backend,
    purpose: "login",
    origin,
  });

  return verifyProof(backend, {
    ...proof,
    backend,
    origin,
    purpose: "login",
    certificate: proof.certificate ?? "",
  });
}`}</DocCode>
          </CardContent>
        </Card>

        <p className="text-sm text-muted-foreground">
          Full working example: <code>examples/react</code>. Try the Demo page
          after starting the mock backend on port 8000.
        </p>
      </SetupStep>

      <Separator />

      <SetupStep
        step={5}
        title="Typed clients (OpenAPI + Orval)"
        description="OpenAPI specs describe agent and backend HTTP APIs."
      >
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-base">
              <BookOpen className="h-4 w-4" />
              Generate from the monorepo
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <DocCode title="Orval">{`# Regenerate Orval clients in @rqc-icu/localid-client
pnpm generate:api

# Use in your frontend
import { agentOpenAPI, backendOpenAPI } from "@rqc-icu/localid-client";

const { data } = await agentOpenAPI.getHealth();
const proof = await agentOpenAPI.signChallenge({
  challenge,
  backend,
  purpose: "login",
  origin: window.location.origin,
});`}</DocCode>
            <p className="text-sm text-muted-foreground">
              With <code>server.dev_mode: true</code>, the agent serves its spec
              at <code>GET /openapi.json</code>. See{" "}
              <code>packages/localid-client/orval.config.ts</code>.
            </p>
          </CardContent>
        </Card>
      </SetupStep>

      <Separator />

      <section className="rounded-xl border bg-muted/20 p-6">
        <h2 className="text-lg font-semibold tracking-tight">Checklist</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          Verify each item before going to production.
        </p>
        <ul className="mt-4 grid gap-2 sm:grid-cols-2">
          {CHECKLIST.map((item) => (
            <li
              key={item}
              className="flex items-start gap-2 rounded-lg border bg-card px-3 py-2 text-sm"
            >
              <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-emerald-600/70" />
              <span>{item}</span>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}
