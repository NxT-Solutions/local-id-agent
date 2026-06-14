import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { configureLocalIDClient } from "../src/config";
import { agentFetch, backendFetch } from "../src/openapi/mutators";

type FetchMock = ReturnType<typeof vi.fn>;

function jsonResponse(payload: unknown, status = 200): Response {
  return new Response(JSON.stringify(payload), {
    status,
    headers: { "content-type": "application/json" },
  });
}

describe("openapi mutators", () => {
  let fetchMock: FetchMock;

  beforeEach(() => {
    configureLocalIDClient({
      agentUrl: "http://127.0.0.1:17443",
      backendUrl: "http://localhost:8000",
    });
    fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("adds origin header for string json bodies", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse({ ok: true }, 201));

    const result = await agentFetch<{
      data: unknown;
      status: number;
      headers: Headers;
    }>("/sign-challenge", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ origin: "http://localhost:5173" }),
    });

    const call = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(call[0]).toBe("http://127.0.0.1:17443/sign-challenge");
    const headers = new Headers(call[1].headers);
    expect(headers.get("Origin")).toBe("http://localhost:5173");
    expect(result.status).toBe(201);
    expect(result.data).toEqual({ ok: true });
  });

  it("ignores non-string bodies and invalid json bodies", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse({ ok: true }));
    await agentFetch("/health", { body: new URLSearchParams("x=y") });
    let call = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(new Headers(call[1].headers).get("Origin")).toBeNull();

    fetchMock.mockResolvedValueOnce(jsonResponse({ ok: true }));
    await agentFetch("/health", {
      headers: { "x-test": "1" },
      body: "{",
    });
    call = fetchMock.mock.calls[1] as [string, RequestInit];
    expect(new Headers(call[1].headers).get("Origin")).toBeNull();
  });

  it("returns undefined data for non-json and invalid-json responses", async () => {
    fetchMock.mockResolvedValueOnce(
      new Response("plain text", { status: 200, headers: { "content-type": "text/plain" } }),
    );
    await expect(agentFetch("/health")).resolves.toMatchObject({
      data: undefined,
      status: 200,
    });

    fetchMock.mockResolvedValueOnce(
      new Response("{", { status: 200, headers: { "content-type": "application/json" } }),
    );
    await expect(agentFetch("/health")).resolves.toMatchObject({
      data: undefined,
      status: 200,
    });

    fetchMock.mockResolvedValueOnce(new Response(null, { status: 200 }));
    await expect(agentFetch("/health")).resolves.toMatchObject({
      data: undefined,
      status: 200,
    });
  });

  it("joins backend urls and preserves status/headers", async () => {
    configureLocalIDClient({ backendUrl: "http://localhost:8000/" });
    fetchMock.mockResolvedValueOnce(jsonResponse({ success: true }, 202));

    const result = await backendFetch<{
      data: unknown;
      status: number;
      headers: Headers;
    }>("/localid/verify", { method: "POST" });

    const call = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(call[0]).toBe("http://localhost:8000/localid/verify");
    expect(call[1]).toMatchObject({ method: "POST" });
    expect(result.status).toBe(202);
    expect(result.data).toEqual({ success: true });
    expect(result.headers).toBeInstanceOf(Headers);
  });
});

