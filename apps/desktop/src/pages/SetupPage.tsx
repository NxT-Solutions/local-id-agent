import { BookOpen, CheckCircle2 } from "lucide-react";
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
import {
  AUTH_FLOW_DIAGRAM,
  INTEGRATION_OVERVIEW_DIAGRAM,
} from "@/lib/setup-diagrams";

export function SetupPage() {
  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Setup guide</h1>
        <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
          How to connect your backend API and frontend (React or similar) to the
          LocalID Agent. The agent signs challenges locally — it does not issue
          login tokens or sessions.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BookOpen className="h-5 w-5" />
            What you need
          </CardTitle>
          <CardDescription>Three pieces working together</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 text-sm md:grid-cols-3">
          <div className="rounded-lg border p-4">
            <p className="font-medium">1. LocalID Agent</p>
            <p className="mt-1 text-muted-foreground">
              This desktop app bundles the agent on{" "}
              <code className="text-xs">127.0.0.1:17443</code>. It signs
              challenges with your smartcard, eID, or mock provider.
            </p>
          </div>
          <div className="rounded-lg border p-4">
            <p className="font-medium">2. Your backend API</p>
            <p className="mt-1 text-muted-foreground">
              Issues one-time challenges and verifies signatures. You own
              sessions and user login — the agent never does.
            </p>
          </div>
          <div className="rounded-lg border p-4">
            <p className="font-medium">3. Your frontend</p>
            <p className="mt-1 text-muted-foreground">
              A web app (React, Vue, etc.) that calls your backend, then the
              agent, then sends the proof back to your backend.
            </p>
          </div>
        </CardContent>
        <CardContent className="pt-0">
          <MermaidDiagram chart={INTEGRATION_OVERVIEW_DIAGRAM} />
        </CardContent>
      </Card>

      <section className="space-y-4">
        <h2 className="text-lg font-semibold">Auth flow</h2>
        <p className="text-sm text-muted-foreground">
          Step-by-step request flow between your frontend, backend, and the local
          agent.
        </p>
        <MermaidDiagram chart={AUTH_FLOW_DIAGRAM} />
      </section>

      <Separator />

      <section className="space-y-4">
        <h2 className="text-lg font-semibold">1. Configure the agent</h2>
        <p className="text-sm text-muted-foreground">
          Open <strong>Settings</strong> in this app. Two allowlists must
          include your exact URLs (scheme + host + port, no trailing slash):
        </p>
        <div className="grid gap-4 md:grid-cols-2">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-base">Allowed origins</CardTitle>
              <CardDescription>Browser frontends that may call the agent</CardDescription>
            </CardHeader>
            <CardContent className="text-sm text-muted-foreground">
              <p>Examples:</p>
              <ul className="mt-2 list-inside list-disc space-y-1 font-mono text-xs">
                <li>http://localhost:5173</li>
                <li>http://localhost:5174</li>
                <li>https://app.example.com</li>
                <li>tauri://localhost</li>
              </ul>
              <p className="mt-3">
                Must match the browser <code>Origin</code> header and the{" "}
                <code>origin</code> field in the sign request.
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-base">Allowed backends</CardTitle>
              <CardDescription>APIs whose challenges may be signed</CardDescription>
            </CardHeader>
            <CardContent className="text-sm text-muted-foreground">
              <p>Examples:</p>
              <ul className="mt-2 list-inside list-disc space-y-1 font-mono text-xs">
                <li>http://localhost:8000</li>
                <li>https://api.example.com</li>
              </ul>
              <p className="mt-3">
                Must match the <code>backend</code> field in{" "}
                <code>POST /sign-challenge</code> exactly.
              </p>
            </CardContent>
          </Card>
        </div>
        <p className="text-sm text-muted-foreground">
          After saving, the agent restarts automatically. Use the{" "}
          <strong>Dashboard</strong> to confirm health and provider status.
        </p>
      </section>

      <Separator />

      <section className="space-y-4">
        <h2 className="text-lg font-semibold">2. Backend API contract</h2>
        <p className="text-sm text-muted-foreground">
          Your backend must expose two endpoints. The mock backend in this repo
          (<code>services/agent/cmd/mock-backend</code>) implements this contract
          for local development.
        </p>

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
            <DocCode>{`// Response 200
{
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
            <DocCode>{`// Request
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
            <CardTitle className="text-base">Canonical payload (for verification)</CardTitle>
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
            <DocCode>{`{"backend":"https://api.example.com","challenge":"...","origin":"https://app.example.com","purpose":"login","timestamp":"2026-06-14T12:00:00Z"}`}</DocCode>
            <p className="text-sm text-muted-foreground">
              Verify with RSA PKCS#1 v1.5 + SHA-256 (RS256).{" "}
              <code>signature</code> is base64url; <code>certificate</code> is
              standard base64 DER.
            </p>
          </CardContent>
        </Card>
      </section>

      <Separator />

      <section className="space-y-4">
        <h2 className="text-lg font-semibold">3. Agent API (localhost)</h2>
        <p className="text-sm text-muted-foreground">
          Default base URL:{" "}
          <code className="text-xs">http://127.0.0.1:17443</code>. Only
          callable from allowed browser origins (CORS).
        </p>

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
            <DocCode>{`// Request
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
      </section>

      <Separator />

      <section className="space-y-4">
        <h2 className="text-lg font-semibold">4. Frontend setup (React)</h2>
        <p className="text-sm text-muted-foreground">
          Use the shared client{" "}
          <code className="text-xs">@rqc-icu/localid-client</code> from this
          monorepo, or mirror the same fetch calls in your framework.
        </p>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-base">Install & configure</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <DocCode>{`# From repo root
pnpm install

# Environment (.env)
VITE_AGENT_URL=http://127.0.0.1:17443
VITE_BACKEND_URL=https://api.example.com

# Run browser demo
pnpm run dev:react   # http://localhost:5173`}</DocCode>
            <p className="text-sm text-muted-foreground">
              Add your frontend origin to agent <strong>Allowed origins</strong>{" "}
              and your API URL to <strong>Allowed backends</strong> before
              testing.
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-base">Auth handler (minimal)</CardTitle>
          </CardHeader>
          <CardContent>
            <DocCode>{`import {
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
          Full working example: <code>examples/react</code> in this repository.
          Try the <strong>Demo</strong> page here after starting the mock
          backend on port 8000.
        </p>
      </section>

      <Separator />

      <section className="space-y-4">
        <h2 className="text-lg font-semibold">5. Typed clients (OpenAPI + Orval)</h2>
        <p className="text-sm text-muted-foreground">
          OpenAPI specs in <code>openapi/</code> describe the agent and backend
          HTTP APIs. With <code>server.dev_mode: true</code>, the agent serves
          its spec at <code>GET /openapi.json</code> for local tooling.
        </p>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-base">Generate from the monorepo</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <DocCode>{`# Regenerate Orval clients in @rqc-icu/localid-client
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
              Or point{" "}
              <a
                href="https://orval.dev/"
                className="underline underline-offset-4"
                target="_blank"
                rel="noreferrer"
              >
                Orval
              </a>{" "}
              at <code>openapi/agent.openapi.yaml</code> or the live{" "}
              <code>/openapi.json</code> URL. See{" "}
              <code>packages/localid-client/orval.config.ts</code>.
            </p>
          </CardContent>
        </Card>
      </section>

      <Separator />

      <section className="space-y-4">
        <h2 className="text-lg font-semibold">Checklist</h2>
        <ul className="space-y-2 text-sm">
          {[
            "Agent running (Dashboard shows Healthy + Ready)",
            "Frontend origin added to Allowed origins in Settings",
            "Backend URL added to Allowed backends in Settings",
            "Backend implements POST /localid/challenge and POST /localid/verify",
            "Backend verifies canonical JSON + RS256 signature",
            "Frontend uses window.location.origin for the origin field",
            "CORS on your backend allows your frontend origin",
          ].map((item) => (
            <li key={item} className="flex items-start gap-2">
              <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
              <span>{item}</span>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}
