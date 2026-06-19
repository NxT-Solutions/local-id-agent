import { invoke } from "@tauri-apps/api/core";
import { fetchHealth, fetchStatus } from "@rqc-icu/localid-client";

export interface DiagnosticsInfo {
  appVersion: string;
  agentUrl: string;
  platform: string;
  sidecarRunning: boolean;
}

export interface AdminDiagnosticsInfo extends DiagnosticsInfo {
  configPath: string;
}

export interface AdminLockStatus {
  configured: boolean;
  unlocked: boolean;
  expiresAt?: number;
  setupRequired: boolean;
  sessionToken?: string;
}

export interface UnlockResult {
  sessionToken: string;
  expiresAt: number;
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

export async function getAdminDiagnostics(): Promise<AdminDiagnosticsInfo> {
  return invoke<AdminDiagnosticsInfo>("get_admin_diagnostics");
}

export async function getAdminLockStatus(): Promise<AdminLockStatus> {
  return invoke<AdminLockStatus>("get_admin_lock_status");
}

export async function setupAdminPasscode(passcode: string): Promise<UnlockResult> {
  return invoke<UnlockResult>("setup_admin_passcode", { passcode });
}

export async function unlockAdmin(passcode: string): Promise<UnlockResult> {
  return invoke<UnlockResult>("unlock_admin", { passcode });
}

export async function lockAdmin(): Promise<void> {
  return invoke("lock_admin");
}

export async function changeAdminPasscode(
  currentPasscode: string,
  newPasscode: string,
): Promise<void> {
  return invoke("change_admin_passcode", { currentPasscode, newPasscode });
}

export async function copyDiagnostics(includeConfigPath: boolean): Promise<string> {
  const diagnostics = includeConfigPath
    ? await getAdminDiagnostics()
    : await getDiagnostics();
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
