# daywrap CLI

Privacy-first CLI tool for generating animated QR-based daily standup data from local git history.

## Install

```bash
brew tap arjun921/tap
brew install daywrap
```

Or download a pre-built binary from [GitHub Releases](https://github.com/arjun921/daywrap-cli/releases).

## Usage

```bash
# Scan today's git activity and display animated QR codes
daywrap

# Specify a date range
daywrap --since "2026-05-13" --until "2026-05-15"

# Scan specific repo paths
daywrap --repo ~/work/backend --repo ~/work/frontend

# Weekly summary
daywrap --weekly

# Output raw JSON (debug)
daywrap --raw
```

## How it works

1. Reads local `.git` log for the target period.
2. Enriches commits with ticket IDs extracted from branch names.
3. Compresses the JSON payload with zlib.
4. Splits into QR-sized chunks (`DW:<index>:<total>:<base64>`).
5. Renders each chunk as an animated QR code in the terminal at 5 fps.

Point your phone's DayWrap app at the screen to receive the data — no network required.

## Optional config (`~/.daywrap.yml`)

```yaml
jira:
  base_url: "https://company.atlassian.net"

repos:
  - ~/work/backend
  - ~/work/frontend

ticket_pattern: "(ENG|PROJ|PLAT)-\\d+"
```

## Privacy

The CLI is fully open source and contains zero proprietary logic. It never stores credentials, never opens network connections, and never transmits data outside the local machine.
