import { fetch as tauriFetch } from "@tauri-apps/plugin-http";

function isTauriRuntime(): boolean {
  if (typeof window === "undefined") {
    return false;
  }

  const runtimeWindow = window as unknown as Record<string, unknown>;
  return "__TAURI_INTERNALS__" in runtimeWindow;
}

export async function appFetch(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<Response> {
  if (isTauriRuntime()) {
    return tauriFetch(input, init);
  }

  return globalThis.fetch(input, init);
}
