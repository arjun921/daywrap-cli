package internal

// enrich.go — extracts ticket IDs from branch names and optionally fetches
// ticket titles via the laptop's existing Jira/Linear local auth.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// EnrichCommits adds Ticket metadata to each commit by:
//  1. Extracting the ticket ID from the branch name (or commit message fallback).
//  2. Optionally fetching the ticket title from Jira when cfg.Jira.BaseURL is set.
func EnrichCommits(commits []Commit, cfg *Config) []Commit {
	pattern := cfg.TicketPattern
	if pattern == "" {
		pattern = DefaultTicketPattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		// Invalid user-supplied pattern — skip enrichment rather than panic.
		fmt.Fprintf(os.Stderr, "daywrap: invalid ticket_pattern %q: %v\n", pattern, err)
		return commits
	}

	for i, c := range commits {
		id := re.FindString(c.Branch)
		if id == "" {
			id = re.FindString(c.Message)
		}
		if id == "" {
			continue
		}
		t := &Ticket{ID: id}
		if cfg.Jira.BaseURL != "" {
			if title, err := fetchJiraTitle(cfg.Jira.BaseURL, id); err == nil && title != "" {
				t.Title = title
			}
		}
		commits[i].Ticket = t
	}
	return commits
}

// fetchJiraTitle retrieves the issue summary from the Jira REST API.
// Auth order: JIRA_TOKEN env var (Bearer) → ~/.netrc basic auth.
func fetchJiraTitle(baseURL, ticketID string) (string, error) {
	// Validate the configured base URL to prevent SSRF.
	parsed, err := url.ParseRequestURI(baseURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return "", fmt.Errorf("jira base_url must be a valid HTTP/HTTPS URL")
	}
	if parsed.Scheme == "http" {
		fmt.Fprintln(os.Stderr, "daywrap: warning: jira base_url uses plain HTTP — credentials will be sent unencrypted")
	}

	// url.PathEscape ensures ticket IDs with unexpected characters (e.g. from a
	// custom ticket_pattern) cannot introduce path traversal in the API endpoint.
	endpoint := strings.TrimRight(baseURL, "/") + "/rest/api/2/issue/" + url.PathEscape(ticketID) + "?fields=summary"

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	if token := os.Getenv("JIRA_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else {
		// Fall back to ~/.netrc basic auth.
		if user, pass, nerr := readNetrc(parsed.Hostname()); nerr == nil && user != "" {
			req.SetBasicAuth(user, pass)
		}
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("jira: HTTP %d for %s", resp.StatusCode, ticketID)
	}

	// Limit response body to 64 KB to guard against oversized responses.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", err
	}

	var result struct {
		Fields struct {
			Summary string `json:"summary"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	return result.Fields.Summary, nil
}

// readNetrc parses ~/.netrc and returns credentials for the given hostname.
// A missing netrc file is silently ignored (returns empty strings, nil error).
func readNetrc(hostname string) (username, password string, err error) {
	home, herr := os.UserHomeDir()
	if herr != nil {
		return
	}
	f, ferr := os.Open(filepath.Join(home, ".netrc"))
	if os.IsNotExist(ferr) {
		return
	}
	if ferr != nil {
		err = ferr
		return
	}
	defer f.Close()

	// Tokenise by whitespace; handle both "machine h login u password p" on one
	// line and the multi-line form across multiple lines.
	var tokens []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// Strip inline comments (# ...)
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		tokens = append(tokens, strings.Fields(line)...)
	}
	if serr := scanner.Err(); serr != nil {
		err = serr
		return
	}

	inMachine := false
	for i := 0; i < len(tokens); i++ {
		switch tokens[i] {
		case "machine":
			i++
			if i < len(tokens) {
				inMachine = tokens[i] == hostname
				// Reset collected credentials when entering a new machine stanza.
				if !inMachine {
					username = ""
					password = ""
				}
			}
		case "default":
			inMachine = true
		case "login":
			i++
			if i < len(tokens) && inMachine && username == "" {
				username = tokens[i]
			}
		case "password":
			i++
			if i < len(tokens) && inMachine && password == "" {
				password = tokens[i]
			}
		}
		if inMachine && username != "" && password != "" {
			return
		}
	}
	return
}

