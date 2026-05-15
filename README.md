# daywrap

Turn your daily git commits into standups and brag docs — powered by an on-device AI model. No data leaves your machine.

**[daywr.app](https://daywr.app)** · [Releases](https://github.com/arjun921/daywrap-cli/releases) · [Issues](https://github.com/arjun921/daywrap-cli/issues)

[![License](https://img.shields.io/badge/license-MIT-brightgreen)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.22-blue?logo=go)](https://golang.org)
[![Release](https://img.shields.io/badge/release-v0.1.1-blue)](https://github.com/arjun921/daywrap-cli/releases/latest)

---

Most developers undersell six months of work in a 15-minute self-review. DayWrap captures it automatically, every day.

---

## How it works

```
[Your laptop]                        [Your phone]
     |                                     |
  daywrap CLI                     DayWrap mobile app
     |                                     |
  reads local .git history        scans QR code from screen
  enriches with file stats        reassembles payload
  compresses into QR payload      runs LLM on-device
     |                                     |
     +------- AIR GAP (QR only) ----------+
```

No server. No network between devices. No OAuth. No account.

The CLI reads your local git history, enriches it with file-level stats and ticket IDs from your branch names, and renders a QR code in your terminal. Point the DayWrap mobile app at your screen. The app picks up the payload and generates a standup or brag doc using an on-device LLM. Everything stays on your hardware.

---

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/arjun921/daywrap-cli/main/install.sh | bash
```

Or download a pre-built binary from [GitHub Releases](https://github.com/arjun921/daywrap-cli/releases).

> macOS and Linux · Intel and Apple Silicon · no root required · single binary, zero dependencies

---

## Usage

```bash
# Today's activity
daywrap

# Specific date range
daywrap --since 2026-05-13 --until 2026-05-15

# Current week (Monday to today)
daywrap --weekly

# Scan specific repos
daywrap --repo ~/work/backend --repo ~/work/frontend

# Filter by author (defaults to current git user)
daywrap --author "you@example.com"

# Output raw JSON instead of QR (for debugging)
daywrap --raw
```

Point the DayWrap mobile app at the QR code on your screen. No network required.

---

## Optional config (`~/.daywrap.yml`)

```yaml
jira:
  base_url: "https://your-company.atlassian.net"

repos:
  - ~/work/backend
  - ~/work/frontend

ticket_pattern: "(ENG|PROJ|PLAT)-\\d+"
```

Jira credentials are read from `$JIRA_TOKEN` or `~/.netrc` — never stored by daywrap.

---

## Security & Privacy

daywrap is designed for engineers at companies with strict security policies. Here is what it guarantees:

- **Zero outbound network connections.** The CLI never phones home, never checks for updates over the network, and never transmits any data anywhere. Verify this yourself — the entire CLI is under 1,000 lines of Go.
- **No source code is extracted.** Only commit metadata is read: messages, file names, timestamps, and branch names. Your actual code is never included in the QR payload.
- **No credentials are stored.** Jira enrichment is optional and uses your existing local auth (`$JIRA_TOKEN` or `~/.netrc`). daywrap never writes credentials to disk.
- **Fully open source.** Audit every line at [github.com/arjun921/daywrap-cli](https://github.com/arjun921/daywrap-cli).

---

## License

MIT