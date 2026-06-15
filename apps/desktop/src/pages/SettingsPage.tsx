import { useEffect, useMemo, useState } from "react";
import { Save, Server, Shield, Sliders } from "lucide-react";
import { AlertBanner } from "@/components/layout/AlertBanner";
import { CopyField } from "@/components/layout/CopyField";
import { PageHeader } from "@/components/layout/PageHeader";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Skeleton } from "@/components/ui/skeleton";
import { ActionFeedbackAnchor } from "@/components/ui/action-feedback";
import { useActionFeedback } from "@/hooks/useActionFeedback";
import { getConfigPath, readConfig, restartAgent, writeConfig } from "@/lib/tauri";

const PROVIDER_OPTIONS = [
  { value: "mock", label: "Mock (development)" },
  { value: "pkcs11", label: "PKCS#11" },
  { value: "belgian_eid", label: "Belgian eID" },
] as const;

const LOG_LEVEL_OPTIONS = [
  { value: "debug", label: "Debug" },
  { value: "info", label: "Info" },
  { value: "warn", label: "Warn" },
  { value: "error", label: "Error" },
] as const;

interface ConfigForm {
  host: string;
  port: string;
  defaultProvider: string;
  allowedOrigins: string;
  allowedBackends: string;
  logLevel: string;
}

function normalizeProvider(value: string): string {
  return PROVIDER_OPTIONS.some((option) => option.value === value)
    ? value
    : "mock";
}

function normalizeLogLevel(value: string): string {
  const level = value.toLowerCase();
  return LOG_LEVEL_OPTIONS.some((option) => option.value === level)
    ? level
    : "info";
}

function parseConfig(raw: string): ConfigForm {
  const parsed = JSON.parse(raw) as {
    server?: { host?: string; port?: number };
    security?: { allowed_origins?: string[]; allowed_backends?: string[] };
    providers?: { default?: string };
    logging?: { level?: string };
  };

  return {
    host: parsed.server?.host ?? "127.0.0.1",
    port: String(parsed.server?.port ?? 17443),
    defaultProvider: normalizeProvider(parsed.providers?.default ?? "mock"),
    allowedOrigins: (parsed.security?.allowed_origins ?? []).join("\n"),
    allowedBackends: (parsed.security?.allowed_backends ?? []).join("\n"),
    logLevel: normalizeLogLevel(parsed.logging?.level ?? "info"),
  };
}

function buildConfig(form: ConfigForm, existingRaw: string): string {
  const existing = JSON.parse(existingRaw) as Record<string, unknown>;
  const server = (existing.server as Record<string, unknown> | undefined) ?? {};
  const security =
    (existing.security as Record<string, unknown> | undefined) ?? {};
  const providers =
    (existing.providers as Record<string, unknown> | undefined) ?? {};
  const logging =
    (existing.logging as Record<string, unknown> | undefined) ?? {};
  const nextProviders = { ...providers };
  const selectedProvider = (nextProviders[form.defaultProvider] as
    | Record<string, unknown>
    | undefined) ?? { enabled: true };

  nextProviders[form.defaultProvider] = {
    ...selectedProvider,
    enabled: true,
  };

  return JSON.stringify(
    {
      ...existing,
      server: {
        ...server,
        host: form.host,
        port: Number(form.port),
      },
      security: {
        ...security,
        allowed_origins: form.allowedOrigins
          .split("\n")
          .map((value) => value.trim())
          .filter(Boolean),
        allowed_backends: form.allowedBackends
          .split("\n")
          .map((value) => value.trim())
          .filter(Boolean),
      },
      providers: {
        ...nextProviders,
        default: form.defaultProvider,
      },
      logging: {
        ...logging,
        level: form.logLevel,
      },
    },
    null,
    2,
  );
}

function countLines(value: string): number {
  return value.split("\n").map((line) => line.trim()).filter(Boolean).length;
}

export function SettingsPage() {
  const [configPath, setConfigPath] = useState("");
  const [rawConfig, setRawConfig] = useState("");
  const [form, setForm] = useState<ConfigForm | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const headerSaveFeedback = useActionFeedback();
  const footerSaveFeedback = useActionFeedback();

  useEffect(() => {
    void (async () => {
      try {
        const [path, contents] = await Promise.all([
          getConfigPath(),
          readConfig(),
        ]);
        setConfigPath(path);
        setRawConfig(contents);
        setForm(parseConfig(contents));
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load config");
      }
    })();
  }, []);

  const originCount = useMemo(
    () => (form ? countLines(form.allowedOrigins) : 0),
    [form],
  );
  const backendCount = useMemo(
    () => (form ? countLines(form.allowedBackends) : 0),
    [form],
  );

  async function handleSave(source: "header" | "footer" = "header") {
    if (!form) return;

    const feedback = source === "header" ? headerSaveFeedback : footerSaveFeedback;

    setSaving(true);
    setMessage(null);
    setError(null);

    try {
      const next = buildConfig(form, rawConfig);
      await writeConfig(next);
      await restartAgent();
      setRawConfig(next);
      setMessage("Configuration saved and agent restarted.");
      feedback.showSuccess("Settings saved");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to save config";
      setError(message);
      feedback.showError("Save failed");
    } finally {
      setSaving(false);
    }
  }

  if (!form) {
    return (
      <div className="space-y-4">
        <PageHeader
          title="Settings"
          description="Loading agent configuration…"
        />
        <div className="space-y-4">
          <Skeleton className="h-32 rounded-xl" />
          <Skeleton className="h-48 rounded-xl" />
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      <PageHeader
        title="Settings"
        description="Edit the bundled Go agent configuration. Saving writes to disk and restarts the sidecar automatically."
        actions={
          <ActionFeedbackAnchor
            feedback={headerSaveFeedback.feedback}
            onOpenChange={headerSaveFeedback.onOpenChange}
          >
            <Button
              onClick={() => void handleSave("header")}
              disabled={saving}
              size="sm"
            >
              <Save className="h-4 w-4" />
              {saving ? "Saving…" : "Save & restart"}
            </Button>
          </ActionFeedbackAnchor>
        }
      />

      {configPath && <CopyField label="Config file" value={configPath} />}

      {message && (
        <AlertBanner variant="success" title="Saved">
          {message} Check the Dashboard to confirm health and provider readiness.
        </AlertBanner>
      )}

      {error && (
        <AlertBanner variant="error" title="Save failed">
          {error}
        </AlertBanner>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Server className="h-4 w-4" />
            Server
          </CardTitle>
          <CardDescription>
            Local bind address for the agent HTTP API (default{" "}
            <code className="font-mono text-xs">127.0.0.1:17443</code>)
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="host">Host</Label>
            <Input
              id="host"
              value={form.host}
              onChange={(event) =>
                setForm({ ...form, host: event.target.value })
              }
            />
            <p className="text-xs text-muted-foreground">
              Loopback only — do not expose to the network.
            </p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="port">Port</Label>
            <Input
              id="port"
              type="number"
              value={form.port}
              onChange={(event) =>
                setForm({ ...form, port: event.target.value })
              }
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Shield className="h-4 w-4" />
            Security allowlists
          </CardTitle>
          <CardDescription>
            Exact string match — scheme + host + port, no wildcards, no trailing
            slash. One entry per line.
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-2">
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label htmlFor="origins">Allowed origins</Label>
              <span className="text-xs text-muted-foreground">
                {originCount} entries
              </span>
            </div>
            <Textarea
              id="origins"
              className="min-h-[140px] font-mono text-xs"
              placeholder={"http://localhost:5173\ntauri://localhost"}
              value={form.allowedOrigins}
              onChange={(event) =>
                setForm({ ...form, allowedOrigins: event.target.value })
              }
            />
            <p className="text-xs text-muted-foreground">
              Must match the browser <code>Origin</code> header and sign request{" "}
              <code>origin</code> field.
            </p>
          </div>
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label htmlFor="backends">Allowed backends</Label>
              <span className="text-xs text-muted-foreground">
                {backendCount} entries
              </span>
            </div>
            <Textarea
              id="backends"
              className="min-h-[140px] font-mono text-xs"
              placeholder={"http://localhost:8000\nhttps://api.example.com"}
              value={form.allowedBackends}
              onChange={(event) =>
                setForm({ ...form, allowedBackends: event.target.value })
              }
            />
            <p className="text-xs text-muted-foreground">
              Must match the <code>backend</code> field in{" "}
              <code>POST /sign-challenge</code> exactly.
            </p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Sliders className="h-4 w-4" />
            Providers & logging
          </CardTitle>
          <CardDescription>
            Default signing provider and log verbosity for the sidecar process.
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="provider">Default provider</Label>
            <Select
              value={form.defaultProvider}
              onValueChange={(value) =>
                setForm({ ...form, defaultProvider: value })
              }
            >
              <SelectTrigger id="provider">
                <SelectValue placeholder="Select provider" />
              </SelectTrigger>
              <SelectContent>
                {PROVIDER_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {form.defaultProvider === "belgian_eid" && (
              <p className="text-xs text-muted-foreground">
                Requires a Belgian eID card reader and middleware. Insert card
                before signing.
              </p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="log-level">Log level</Label>
            <Select
              value={form.logLevel}
              onValueChange={(value) => setForm({ ...form, logLevel: value })}
            >
              <SelectTrigger id="log-level">
                <SelectValue placeholder="Select log level" />
              </SelectTrigger>
              <SelectContent>
                {LOG_LEVEL_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      <div className="flex items-center gap-3 border-t pt-4">
        <ActionFeedbackAnchor
          feedback={footerSaveFeedback.feedback}
          onOpenChange={footerSaveFeedback.onOpenChange}
        >
          <Button onClick={() => void handleSave("footer")} disabled={saving}>
            <Save className="h-4 w-4" />
            {saving ? "Saving…" : "Save & restart"}
          </Button>
        </ActionFeedbackAnchor>
        <p className="text-xs text-muted-foreground">
          Changes take effect after the sidecar restarts (~1s).
        </p>
      </div>
    </div>
  );
}
