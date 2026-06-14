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

void (async () => {
  if (!isTauriRuntime()) {
    return;
  }

  try {
    const { fetch: tauriFetch } = await import("@tauri-apps/plugin-http");
    globalThis.fetch = tauriFetch as typeof fetch;
  } catch (error) {
    console.error("[desktop] Failed to initialize Tauri fetch shim", error);
  }
})();
