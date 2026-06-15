import { invoke } from "@tauri-apps/api/core";
import { fetchHealth, fetchStatus } from "@rqc-icu/localid-client";

export interface DiagnosticsInfo {
  appVersion: string;
  configPath: string;
  agentUrl: string;
  platform: string;
  sidecarRunning: boolean;
}

export async function getConfigPath(): Promise<string> {
  return invoke<string>("get_config_path");
}

export async function readConfig(): Promise<string> {
  return invoke<string>("read_config");
}

export async function writeConfig(contents: string): Promise<void> {
  return invoke("write_config", { contents });
}

export async function restartAgent(): Promise<void> {
  return invoke("restart_agent");
}

export async function getDiagnostics(): Promise<DiagnosticsInfo> {
  return invoke<DiagnosticsInfo>("get_diagnostics");
}

export async function copyDiagnostics(): Promise<string> {
  const diagnostics = await getDiagnostics();
  const health = await fetchHealth().catch(() => ({ error: "unreachable" }));
  const status = await fetchStatus().catch(() => ({ error: "unreachable" }));

  return JSON.stringify(
    {
      ...diagnostics,
      health,
      status,
      timestamp: new Date().toISOString(),
    },
    null,
    2,
  );
}
