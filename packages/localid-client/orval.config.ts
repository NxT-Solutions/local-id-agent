import { defineConfig } from "orval";

export default defineConfig({
  agent: {
    input: "../../openapi/agent.openapi.yaml",
    output: {
      mode: "single",
      target: "./src/openapi/agent.ts",
      client: "fetch",
      override: {
        mutator: {
          path: "./src/openapi/mutators.ts",
          name: "agentFetch",
        },
      },
    },
  },
  backend: {
    input: "../../openapi/backend.openapi.yaml",
    output: {
      mode: "single",
      target: "./src/openapi/backend.ts",
      client: "fetch",
      override: {
        mutator: {
          path: "./src/openapi/mutators.ts",
          name: "backendFetch",
        },
      },
    },
  },
});
