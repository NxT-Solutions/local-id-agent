import { useEffect, useState } from "react";
import { Save } from "lucide-react";
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
        ...providers,
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

export function SettingsPage() {
  const [configPath, setConfigPath] = useState("");
  const [rawConfig, setRawConfig] = useState("");
  const [form, setForm] = useState<ConfigForm | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

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

  async function handleSave() {
    if (!form) return;

    setSaving(true);
    setMessage(null);
    setError(null);

    try {
      const next = buildConfig(form, rawConfig);
      await writeConfig(next);
      await restartAgent();
      setRawConfig(next);
      setMessage("Configuration saved and agent restarted.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save config");
    } finally {
      setSaving(false);
    }
  }

  if (!form) {
    return <p className="text-sm text-muted-foreground">Loading settings…</p>;
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Settings</h1>
        <p className="text-sm text-muted-foreground">
          Edit agent configuration stored at{" "}
          <code className="rounded bg-muted px-1 py-0.5 text-xs">{configPath}</code>
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Server</CardTitle>
          <CardDescription>Local bind address for the agent HTTP API</CardDescription>
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
          <CardTitle>Security</CardTitle>
          <CardDescription>One origin or backend URL per line</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="origins">Allowed origins</Label>
            <Textarea
              id="origins"
              value={form.allowedOrigins}
              onChange={(event) =>
                setForm({ ...form, allowedOrigins: event.target.value })
              }
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="backends">Allowed backends</Label>
            <Textarea
              id="backends"
              value={form.allowedBackends}
              onChange={(event) =>
                setForm({ ...form, allowedBackends: event.target.value })
              }
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Providers & logging</CardTitle>
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

      <div className="flex items-center gap-3">
        <Button onClick={() => void handleSave()} disabled={saving}>
          <Save className="h-4 w-4" />
          {saving ? "Saving…" : "Save & restart"}
        </Button>
        {message && <p className="text-sm text-emerald-600">{message}</p>}
        {error && <p className="text-sm text-destructive">{error}</p>}
      </div>
    </div>
  );
}
