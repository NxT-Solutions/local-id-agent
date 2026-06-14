import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  checkAgentReadiness,
  fetchHealth,
  fetchStatus,
  signChallenge,
} from "../src/agent";
import { fetchChallenge, verifyProof } from "../src/backend";
import { configureLocalIDClient } from "../src/config";
import type { VerifyRequest } from "../src/types";

type FetchMock = ReturnType<typeof vi.fn>;

function jsonResponse(payload: unknown, status = 200): Response {
  return new Response(JSON.stringify(payload), {
    status,
    headers: { "content-type": "application/json" },
  });
}

describe("agent and backend clients", () => {
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

  it("checks readiness successfully", async () => {
    fetchMock
      .mockResolvedValueOnce(jsonResponse({ ok: true, name: "LocalID Agent", version: "0.1.0" }))
      .mockResolvedValueOnce(jsonResponse({ provider: "mock", ready: true, cardPresent: true }));

    await expect(checkAgentReadiness()).resolves.toEqual({
      healthy: true,
      ready: true,
      provider: "mock",
    });
    expect(fetchMock).toHaveBeenNthCalledWith(1, "http://127.0.0.1:17443/health");
    expect(fetchMock).toHaveBeenNthCalledWith(2, "http://127.0.0.1:17443/status");
  });

  it("returns readiness fallback when request fails", async () => {
    fetchMock.mockRejectedValueOnce(new Error("network down"));

    await expect(checkAgentReadiness()).resolves.toEqual({
      healthy: false,
      ready: false,
      provider: "unknown",
      error: "network down",
    });
  });

  it("handles non-error throw in readiness check", async () => {
    fetchMock.mockRejectedValueOnce("boom");

    await expect(checkAgentReadiness()).resolves.toEqual({
      healthy: false,
      ready: false,
      provider: "unknown",
      error: "Agent is unreachable",
    });
  });

  it("fetches health and status", async () => {
    fetchMock
      .mockResolvedValueOnce(jsonResponse({ ok: true, name: "Agent", version: "1" }))
      .mockResolvedValueOnce(jsonResponse({ provider: "mock", ready: true, cardPresent: true }));

    await expect(fetchHealth()).resolves.toEqual({ ok: true, name: "Agent", version: "1" });
    await expect(fetchStatus()).resolves.toEqual({ provider: "mock", ready: true, cardPresent: true });
  });

  it("throws api error from json body", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse({ error: "bad request" }, 400));
    await expect(fetchHealth()).rejects.toThrow("bad request");
  });

  it("throws fallback error for non-json body", async () => {
    fetchMock.mockResolvedValueOnce(
      new Response("oops", { status: 500, headers: { "content-type": "text/plain" } }),
    );
    await expect(fetchStatus()).rejects.toThrow("Request failed (500)");
  });

  it("throws fallback error for json body without error field", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse({ message: "bad" }, 401));
    await expect(signChallenge({
      challenge: "YWJj",
      backend: "http://localhost:8000",
      origin: "http://localhost:5173",
      purpose: "login",
    })).rejects.toThrow("Request failed (401)");
  });

  it("signs challenge", async () => {
    const request = {
      challenge: "YWJj",
      backend: "http://localhost:8000",
      origin: "http://localhost:5173",
      purpose: "login",
    };
    fetchMock.mockResolvedValueOnce(
      jsonResponse({
        provider: "mock",
        algorithm: "RS256",
        challenge: "YWJj",
        signature: "sig",
        certificate: "cert",
        signedAt: "2026-06-14T12:00:00Z",
      }),
    );

    await signChallenge(request);
    expect(fetchMock).toHaveBeenCalledWith("http://127.0.0.1:17443/sign-challenge", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(request),
    });
  });

  it("fetches backend challenge using default backend", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse({ challenge: "abc" }));
    await expect(fetchChallenge()).resolves.toEqual({ challenge: "abc" });
    expect(fetchMock).toHaveBeenCalledWith("http://localhost:8000/localid/challenge", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({}),
    });
  });

  it("verifies proof with explicit backend", async () => {
    const proof: VerifyRequest = {
      challenge: "abc",
      backend: "http://localhost:8000",
      origin: "http://localhost:5173",
      purpose: "login",
      provider: "mock",
      algorithm: "RS256",
      signature: "sig",
      certificate: "cert",
      signedAt: "2026-06-14T12:00:00Z",
    };
    fetchMock.mockResolvedValueOnce(
      jsonResponse({ success: true, user: { id: "1", name: "Dev User" } }),
    );

    await expect(verifyProof("http://backend.test", proof)).resolves.toEqual({
      success: true,
      user: { id: "1", name: "Dev User" },
    });
    expect(fetchMock).toHaveBeenCalledWith("http://backend.test/localid/verify", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(proof),
    });
  });

  it("throws backend json error message", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse({ error: "backend failed" }, 403));
    await expect(fetchChallenge("http://backend.test")).rejects.toThrow("backend failed");
  });

  it("throws backend fallback on non-json error body", async () => {
    fetchMock.mockResolvedValueOnce(
      new Response("nope", { status: 502, headers: { "content-type": "text/plain" } }),
    );
    await expect(
      verifyProof("http://backend.test", {
        challenge: "abc",
        backend: "http://localhost:8000",
        origin: "http://localhost:5173",
        purpose: "login",
        provider: "mock",
        algorithm: "RS256",
        signature: "sig",
        certificate: "cert",
        signedAt: "2026-06-14T12:00:00Z",
      }),
    ).rejects.toThrow("Request failed (502)");
  });

  it("throws backend fallback when json has no error field", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse({ message: "no error field" }, 418));
    await expect(
      fetchChallenge("http://backend.test"),
    ).rejects.toThrow("Request failed (418)");
  });
});

