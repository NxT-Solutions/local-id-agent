export interface LocalIDClientConfig {
  agentUrl?: string;
  backendUrl?: string;
}

let agentUrl = "http://127.0.0.1:17443";
let backendUrl = "http://localhost:8000";

export function configureLocalIDClient(config: LocalIDClientConfig = {}): void {
  if (config.agentUrl !== undefined) {
    agentUrl = config.agentUrl;
  }
  if (config.backendUrl !== undefined) {
    backendUrl = config.backendUrl;
  }
}

export function getAgentUrl(): string {
  return agentUrl;
}

export function getBackendUrl(): string {
  return backendUrl;
}
