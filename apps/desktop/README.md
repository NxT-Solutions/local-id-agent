# LocalID Agent Desktop

Tauri 2 desktop shell. See [docs/desktop.md](../../docs/desktop.md) for setup, development, and build instructions.

```bash
# from repo root
pnpm run build:sidecar              # build Go sidecar (once per agent change)
pnpm --filter desktop tauri dev     # full app (needs Rust)

# UI only in browser (no Rust):
pnpm run dev:desktop
```

See [docs/desktop.md](../../docs/desktop.md) for details.
