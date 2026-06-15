import { getAgentUrl, getFetch } from "./config";
import type {
  AgentReadiness,
  HealthResponse,
  SignChallengeRequest,
  SignChallengeResponse,
  StatusResponse,
} from "./types";

async function parseJSON<T>(response: Response): Promise<T> {
  if (!response.ok) {
    let message = `Request failed (${response.status})`;
    try {
      const body = (await response.json()) as { error?: string; message?: string };
      if (body.message) {
        message = body.message;
      } else if (body.error) {
        message = body.error;
      }
    } catch {
      // ignore non-JSON error bodies
    }
    throw new Error(message);
  }

  return (await response.json()) as T;
}

export async function checkAgentReadiness(): Promise<AgentReadiness> {
  const agentUrl = getAgentUrl();
  const doFetch = getFetch();

  try {
    const [healthRes, statusRes] = await Promise.all([
      doFetch(`${agentUrl}/health`),
      doFetch(`${agentUrl}/status`),
    ]);

    const health = await parseJSON<HealthResponse>(healthRes);
    const status = await parseJSON<StatusResponse>(statusRes);

    return {
      healthy: health.ok,
      ready: status.ready,
      provider: status.provider,
    };
  } catch (error) {
    const message =
      error instanceof Error ? error.message : "Agent is unreachable";
    return {
      healthy: false,
      ready: false,
      provider: "unknown",
      error: message,
    };
  }
}

export async function fetchHealth(): Promise<HealthResponse> {
  const response = await getFetch()(`${getAgentUrl()}/health`);
  return parseJSON<HealthResponse>(response);
}

export async function fetchStatus(): Promise<StatusResponse> {
  const response = await getFetch()(`${getAgentUrl()}/status`);
  return parseJSON<StatusResponse>(response);
}

export async function signChallenge(
  request: SignChallengeRequest,
): Promise<SignChallengeResponse> {
  const response = await getFetch()(`${getAgentUrl()}/sign-challenge`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Origin: request.origin },
    body: JSON.stringify(request),
  });

  return parseJSON<SignChallengeResponse>(response);
}
