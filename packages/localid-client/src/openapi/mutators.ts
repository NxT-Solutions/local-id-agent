import { getAgentUrl, getBackendUrl, getFetch } from "../config";

async function readResponseData(response: Response): Promise<unknown> {
  const contentType = response.headers.get("content-type") ?? "";
  if (!contentType.includes("application/json")) {
    return undefined;
  }

  try {
    return await response.json();
  } catch {
    return undefined;
  }
}

function joinUrl(base: string, path: string): string {
  return new URL(path, base.endsWith("/") ? base : `${base}/`).toString();
}

function withOriginHeader(options: RequestInit): RequestInit {
  const headers = new Headers(options.headers);

  if (options.body && typeof options.body === "string") {
    try {
      const parsed = JSON.parse(options.body) as { origin?: string };
      if (parsed.origin) {
        headers.set("Origin", parsed.origin);
      }
    } catch {
      // ignore invalid JSON bodies
    }
  }

  return { ...options, headers };
}

export const agentFetch = async <T>(
  url: string,
  options: RequestInit = {},
): Promise<T> => {
  const response = await getFetch()(
    joinUrl(getAgentUrl(), url),
    withOriginHeader(options),
  );
  const data = await readResponseData(response);

  return {
    data,
    status: response.status,
    headers: response.headers,
  } as T;
};

export const backendFetch = async <T>(
  url: string,
  options: RequestInit = {},
): Promise<T> => {
  const response = await getFetch()(joinUrl(getBackendUrl(), url), options);
  const data = await readResponseData(response);

  return {
    data,
    status: response.status,
    headers: response.headers,
  } as T;
};
