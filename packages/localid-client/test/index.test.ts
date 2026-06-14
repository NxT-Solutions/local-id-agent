import { describe, expect, it } from "vitest";

import * as client from "../src/index";

describe("index exports", () => {
  it("re-exports runtime api", () => {
    expect(typeof client.configureLocalIDClient).toBe("function");
    expect(typeof client.getAgentUrl).toBe("function");
    expect(typeof client.getBackendUrl).toBe("function");
    expect(typeof client.checkAgentReadiness).toBe("function");
    expect(typeof client.fetchHealth).toBe("function");
    expect(typeof client.fetchStatus).toBe("function");
    expect(typeof client.signChallenge).toBe("function");
    expect(typeof client.fetchChallenge).toBe("function");
    expect(typeof client.verifyProof).toBe("function");
    expect(client.agentOpenAPI).toBeTypeOf("object");
    expect(client.backendOpenAPI).toBeTypeOf("object");
  });
});

