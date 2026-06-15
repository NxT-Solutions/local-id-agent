import { invoke } from "@tauri-apps/api/core";
import { getAgentUrl } from "@rqc-icu/localid-client";

interface AgentFetchResponse {
  status: number;
  body: string;
}

function resolveRequestUrl(input: RequestInfo | URL): string {
  if (typeof input === "string") {
    return input;
  }
  if (input instanceof URL) {
    return input.href;
  }
  return input.url;
}

function headerValue(
  headers: HeadersInit | undefined,
  name: string,
): string | undefined {
  if (!headers) {
    return undefined;
  }

  if (headers instanceof Headers) {
    return headers.get(name) ?? undefined;
  }

  if (Array.isArray(headers)) {
    const match = headers.find(([key]) => key.toLowerCase() === name.toLowerCase());
    return match?.[1];
  }

  const direct = headers[name as keyof typeof headers];
  if (typeof direct === "string") {
    return direct;
  }

  const lower = headers[name.toLowerCase() as keyof typeof headers];
  return typeof lower === "string" ? lower : undefined;
}

export function isAgentRequest(input: RequestInfo | URL): boolean {
  const requestUrl = new URL(resolveRequestUrl(input));
  const agentUrl = new URL(getAgentUrl());
  return requestUrl.origin === agentUrl.origin;
}

export async function tauriAgentFetch(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<Response> {
  const requestUrl = new URL(resolveRequestUrl(input));
  const path = `${requestUrl.pathname}${requestUrl.search}`;

  const result = await invoke<AgentFetchResponse>("agent_fetch", {
    method: init?.method ?? "GET",
    path,
    body: typeof init?.body === "string" ? init.body : undefined,
    origin: headerValue(init?.headers, "Origin"),
  });

  return new Response(result.body, {
    status: result.status,
    headers: { "Content-Type": "application/json" },
  });
}
