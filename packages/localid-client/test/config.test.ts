import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  configureLocalIDClient,
  getAgentUrl,
  getBackendUrl,
  getFetch,
} from "../src/config";

describe("config", () => {
  beforeEach(() => {
    configureLocalIDClient({
      agentUrl: "http://127.0.0.1:17443",
      backendUrl: "http://localhost:8000",
      fetchImpl: null,
    });
  });

  it("returns defaults", () => {
    expect(getAgentUrl()).toBe("http://127.0.0.1:17443");
    expect(getBackendUrl()).toBe("http://localhost:8000");
  });

  it("updates both urls", () => {
    configureLocalIDClient({
      agentUrl: "http://agent.test",
      backendUrl: "http://backend.test",
    });

    expect(getAgentUrl()).toBe("http://agent.test");
    expect(getBackendUrl()).toBe("http://backend.test");
  });

  it("updates partially", () => {
    configureLocalIDClient({ agentUrl: "http://agent.test" });
    expect(getAgentUrl()).toBe("http://agent.test");
    expect(getBackendUrl()).toBe("http://localhost:8000");

    configureLocalIDClient({ backendUrl: "http://backend.test" });
    expect(getAgentUrl()).toBe("http://agent.test");
    expect(getBackendUrl()).toBe("http://backend.test");
  });

  it("uses injected fetch implementation", async () => {
    const fetchImpl = vi.fn().mockResolvedValue(new Response("{}"));
    configureLocalIDClient({ fetchImpl: fetchImpl as typeof fetch });

    await getFetch()("http://example.test");

    expect(fetchImpl).toHaveBeenCalledWith("http://example.test");
  });

  it("throws when fetch is unavailable", () => {
    const originalFetch = globalThis.fetch;
    // @ts-expect-error test stub
    delete globalThis.fetch;
    configureLocalIDClient({ fetchImpl: null });

    expect(() => getFetch()).toThrow("No fetch implementation is available.");

    globalThis.fetch = originalFetch;
  });
});

