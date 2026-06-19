import { open } from "@tauri-apps/plugin-shell";

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

export async function openExternalUrl(url: string): Promise<void> {
  if (isTauriRuntime()) {
    await open(url);
    return;
  }

  window.open(url, "_blank", "noopener,noreferrer");
}
