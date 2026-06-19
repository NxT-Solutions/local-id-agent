export const REPO_BASE = "https://github.com/NxT-Solutions/local-id-agent";

export const DOC_RESOURCES = [
  { label: "Architecture", path: "docs/architecture.md" },
  { label: "Agent config", path: "docs/agent-config.md" },
  { label: "Desktop guide", path: "docs/desktop.md" },
  { label: "OpenAPI specs", path: "openapi/" },
] as const;

export function repoDocUrl(path: string): string {
  if (path.endsWith("/")) {
    return `${REPO_BASE}/tree/main/${path.slice(0, -1)}`;
  }
  return `${REPO_BASE}/blob/main/${path}`;
}
