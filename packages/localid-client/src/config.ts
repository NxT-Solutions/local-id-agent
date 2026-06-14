export interface LocalIDClientConfig {
  agentUrl?: string;
  backendUrl?: string;
  fetchImpl?: typeof fetch | null;
}

let agentUrl = "http://127.0.0.1:17443";
let backendUrl = "http://localhost:8000";
let fetchImpl: typeof fetch | null = null;

export function configureLocalIDClient(config: LocalIDClientConfig = {}): void {
  if (config.agentUrl !== undefined) {
    agentUrl = config.agentUrl;
  }
  if (config.backendUrl !== undefined) {
    backendUrl = config.backendUrl;
  }
  if (config.fetchImpl !== undefined) {
    fetchImpl = config.fetchImpl;
  }
}

export function getAgentUrl(): string {
  return agentUrl;
}

export function getBackendUrl(): string {
  return backendUrl;
}

export function getFetch(): typeof fetch {
  if (fetchImpl) {
    return fetchImpl;
  }

  if (typeof globalThis.fetch === "function") {
    return globalThis.fetch.bind(globalThis);
  }

  throw new Error("No fetch implementation is available.");
}
