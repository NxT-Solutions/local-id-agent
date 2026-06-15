import { fetch as tauriFetch } from "@tauri-apps/plugin-http";
import { isAgentRequest, tauriAgentFetch } from "@/lib/agent-fetch";

function isTauriRuntime(): boolean {
  if (typeof window === "undefined") {
    return false;
  }

  const runtimeWindow = window as unknown as Record<string, unknown>;
  return (
    typeof runtimeWindow.__TAURI__ !== "undefined" ||
    typeof runtimeWindow.__TAURI_INTERNALS__ !== "undefined"
  );
}

export function appFetch(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<Response> {
  if (!isTauriRuntime()) {
    return globalThis.fetch(input, init);
  }

  if (isAgentRequest(input)) {
    return tauriAgentFetch(input, init);
  }

  // WebView fetch to localhost backends fails with "Load failed"; use the HTTP plugin.
  return tauriFetch(input, init);
}
