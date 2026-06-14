import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    include: ["test/**/*.test.ts"],
    coverage: {
      provider: "v8",
      reporter: ["text", "json-summary"],
      include: [
        "src/agent.ts",
        "src/backend.ts",
        "src/config.ts",
        "src/index.ts",
        "src/openapi/mutators.ts",
      ],
      exclude: [
        "src/generated/**",
        "src/openapi/agent.ts",
        "src/openapi/backend.ts",
      ],
      thresholds: {
        lines: 100,
        statements: 100,
        branches: 100,
        functions: 100,
      },
    },
  },
});

