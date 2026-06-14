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

let tauriFetchPromise: Promise<typeof fetch | null> | null = null;

async function getTauriFetch(): Promise<typeof fetch | null> {
  if (!isTauriRuntime()) {
    return null;
  }

  if (!tauriFetchPromise) {
    tauriFetchPromise = import("@tauri-apps/plugin-http")
      .then((module) => module.fetch as typeof fetch)
      .catch((error) => {
        console.error("[desktop] Failed to load Tauri fetch plugin", error);
        return null;
      });
  }

  return tauriFetchPromise;
}

export async function appFetch(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<Response> {
  const tauriFetch = await getTauriFetch();
  if (tauriFetch) {
    try {
      return tauriFetch(input, init);
    } catch (error) {
      console.error("[desktop] Tauri fetch failed, falling back", error);
    }
  }

  return globalThis.fetch(input, init);
}
