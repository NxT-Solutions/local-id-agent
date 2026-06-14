import { isTauri } from "@tauri-apps/api/core";
import { fetch as tauriFetch } from "@tauri-apps/plugin-http";

if (isTauri()) {
  globalThis.fetch = tauriFetch as typeof fetch;
}
