type RuntimeConfig = {
  agentUrl?: string;
  backendUrl?: string;
};

declare global {
  interface Window {
    __LOCALID_CONFIG__?: RuntimeConfig;
  }
}

const defaultConfig = {
  agentUrl: "http://127.0.0.1:17443",
  backendUrl: "http://localhost:8000",
};

function normalize(value: unknown): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }

  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : undefined;
}

export function getRuntimeConfig() {
  const runtime = window.__LOCALID_CONFIG__;

  return {
    agentUrl:
      normalize(runtime?.agentUrl) ??
      normalize(import.meta.env.VITE_AGENT_URL) ??
      defaultConfig.agentUrl,
    backendUrl:
      normalize(runtime?.backendUrl) ??
      normalize(import.meta.env.VITE_BACKEND_URL) ??
      defaultConfig.backendUrl,
  };
}
