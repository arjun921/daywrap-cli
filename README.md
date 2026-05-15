# daywrap

Turn your daily git commits into standups and brag docs — powered by an on-device AI model. No data leaves your machine.

**[daywrap.dev](https://daywrap.dev)** · [Releases](https://github.com/arjun921/daywrap-cli/releases) · [Issues](https://github.com/arjun921/daywrap-cli/issues)

---

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/arjun921/daywrap-cli/main/install.sh | bash
```

Or download a pre-built binary from [GitHub Releases](https://github.com/arjun921/daywrap-cli/releases).

> macOS and Linux · Intel and Apple Silicon · no root required

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
```

Point the DayWrap mobile app at the QR codes on your screen. No network required.

---

## Optional config (`~/.daywrap.yml`)

```yaml
jira:
  base_url: "https://your-company.atlassian.net"

repos:
  - ~/work/backend
  - ~/work/frontend
```

Jira credentials are read from `$JIRA_TOKEN` or `~/.netrc` — never stored by daywrap.

---

## Privacy

daywrap is fully open source. It reads only your local git history, never opens a network connection of its own, and never stores or transmits your data. The QR transfer happens directly from your screen to your phone — no server in between.
