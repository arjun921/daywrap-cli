# DayWrap CLI — Agent Instructions

A privacy-first Go CLI that reads local git history, compresses it, and renders animated QR codes for the [DayWrap mobile app](https://daywr.app). No network traffic leaves the machine.

## Build & Run

```bash
# Build (version injected at link time)
go build -ldflags "-X github.com/arjun921/daywrap-cli/cmd.Version=v0.1.1" -o daywrap .

# Run directly
go run . [flags]

# Test
go test ./...
```

No Makefile. No generated code. Dependencies: `cobra`, `go-qrcode`, `yaml.v3`.

## Architecture

```
cmd/root.go          CLI entry, flag parsing, pipeline orchestration
internal/
  config.go          Load ~/.daywrap.yml (missing file is OK)
  discover.go        Expand repo paths → find nested .git dirs (max depth 3)
  git.go             git log --numstat parsing → []Commit; resolves repo name from remote
  enrich.go          Extract ticket IDs (regex) + optional Jira REST fetch
  compress.go        JSON → zlib → base64url (no padding)
  chunk.go           Split encoded payload into QR-sized frames (daywrap://scan?d=DW:<i>:<n>:<data>)
  qr.go              Optimal QR version for terminal size; padding equalisation; 5 fps animation
  termsize_unix.go   TIOCGWINSZ ioctl; termsize_windows.go for Windows
  types.go           All shared structs: Payload, Commit, FileChange, Ticket, SummaryStats
```

## Data Pipeline

`Config + flags` → `DiscoverRepos` → `ReadCommits` → `EnrichCommits` → `buildPayload` → `Compress` → `Chunk` → `Display` (QR) or JSON (`--raw`)

## Key Conventions

- **No network calls from the CLI** — the only allowed outbound call is optional Jira enrichment, guarded by an explicit `cfg.Jira.BaseURL` check. Never add telemetry or update checks.
- **Housekeeping commits are filtered out** in `parseGitLog`: merge commits, submodule bumps, `.gitmodules`-only changes.
- **Frame equalisation** (`equalisedChunks` in `qr.go`): all QR frames must encode at the same QR version to prevent size-flicker. Pad shorter frames with `&p=~~~...` (byte-mode tilde, ignored by mobile decoder).
- **Auth precedence for Jira**: `$JIRA_TOKEN` (Bearer) → `~/.netrc` (basic auth). Never store credentials.
- **Repo name** comes from `git remote get-url origin`, not the directory name. Falls back to folder basename.
- **Version** is injected via `-ldflags`; `cmd.Version` is the single source of truth.
- **No root required, single binary** — install target is `~/.local/bin` (see [install.sh](install.sh)).

## Config Schema (`~/.daywrap.yml`)

```yaml
jira:
  base_url: "https://your-company.atlassian.net"
repos:
  - ~/work/backend
ticket_pattern: "(ENG|PROJ|PLAT)-\\d+"   # default if omitted
```

## Security Notes

- `fetchJiraTitle` validates `base_url` is HTTP/HTTPS before making requests (SSRF prevention).
- Jira response body is limited to 64 KB.
- No source code, only commit metadata (messages, file names, timestamps, branch names) enters the payload.

## Reference

- [README.md](README.md) — usage examples and full flag reference
- [internal/types.go](internal/types.go) — canonical data model
