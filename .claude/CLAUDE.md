# skemid — project memory

Project-local memory (lives in the repo so it survives folder renames).

## What this is

**skemid** — a CLI that connects to a PostgreSQL database, introspects the schema,
and generates documentation as a single self-contained file (HTML or JSON).

**Positioning / #1 selling point:** runs entirely locally — the schema never leaves
the machine. No cloud, no account, no network calls. Aimed at regulated / air-gapped /
GDPR-constrained users who can't send their schema to a SaaS. This privacy angle leads
the README.

- Naming history: was `db-reader`, then `schemato` (taken), now **skemid**. Don't
  reintroduce the old names.
- Module path: `github.com/giorgiodots/skemid`. Repo dir was `db-schema` and is being
  renamed — don't hardcode the old dir name.

## Structure decision (deliberate, don't "fix" it)

Flat single-module layout: `go.mod` + `main.go` + `schema.html.gohtml` at repo root.
**No `cmd/`, `internal/`, `pkg/`** — rejected on purpose as over-engineering for one
binary. The "standard project layout" is not official Go.

**Trigger to revisit:** when a future module (a UI is planned) needs to *reuse* the
introspection logic. At that point: extract the schema-reading code into an importable
package (e.g. `schema/`), leave `main.go` as a thin CLI wrapper, and add `cmd/skemid/`
only if the binary should leave the root. Driven by a real import need, not aesthetics.

## Status & roadmap

- PostgreSQL only. This is v1.
- Not implemented yet (no code exists — don't describe as if present): ER diagram,
  indexes, other databases (MySQL/SQLite), UI module.

## Working preferences

- **Lazy/minimal by default** (ponytail mode): smallest change that works, stdlib over
  deps, native over libraries, no speculative abstractions. HTML is rendered via stdlib
  `html/template` (auto-escaping) with the markup in `schema.html.gohtml`, not in Go.
- **Catppuccin themes** — uses Catppuccin Latte (light) accents in the HTML output;
  accent is a CSS variable (`--accent`, currently mauve) for easy swapping. Keep base
  white, monospace, no fancy fonts.
- Reviews come in from another LLM (in Italian). Verify each claim against the code
  before acting — at least one such note was a false alarm (claimed identifiers weren't
  escaped; `html/template` already escapes them). Don't double-escape.
- READMEs/docs: dry, technical, accurate over impressive. Use explicit `<TODO: ...>`
  placeholders rather than inventing details. `<TODO: license>` and screenshot TODO
  are currently open.
