# Antigravity & Gemini Project Context

See [AGENTS.md](AGENTS.md) for full architecture, design guidelines, and commands.

## Key Directives:
- Always run `make test-race` and `make lint` before finishing tasks.
- Keep the binary minimal and statically compiled without CGO (`CGO_ENABLED=0`).
- Ensure all new action endpoints maintain compatibility with ZITADEL Actions V2 `append_claims`.
