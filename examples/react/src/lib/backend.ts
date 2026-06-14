import { BACKEND_URL } from "./config";
import type { ChallengeResponse, VerifyRequest, VerifyResponse } from "./types";

async function parseJSON<T>(response: Response): Promise<T> {
  if (!response.ok) {
    let message = `Request failed (${response.status})`;
    try {
      const body = (await response.json()) as { error?: string };
      if (body.error) {
        message = body.error;
      }
    } catch {
      // ignore non-JSON error bodies
    }
    throw new Error(message);
  }

  return (await response.json()) as T;
}

export async function fetchChallenge(
  backendUrl: string = BACKEND_URL,
): Promise<ChallengeResponse> {
  const response = await fetch(`${backendUrl}/localid/challenge`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({}),
  });

  return parseJSON<ChallengeResponse>(response);
}

export async function verifyProof(
  backendUrl: string,
  proof: VerifyRequest,
): Promise<VerifyResponse> {
  const response = await fetch(`${backendUrl}/localid/verify`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(proof),
  });

  return parseJSON<VerifyResponse>(response);
}
