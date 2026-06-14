import { beforeEach, describe, expect, it } from "vitest";

import {
  configureLocalIDClient,
  getAgentUrl,
  getBackendUrl,
} from "../src/config";

describe("config", () => {
  beforeEach(() => {
    configureLocalIDClient({
      agentUrl: "http://127.0.0.1:17443",
      backendUrl: "http://localhost:8000",
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
});

