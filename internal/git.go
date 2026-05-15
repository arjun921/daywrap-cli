package internal

// git.go — reads local git log and parses it into the DayWrap commit schema.

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// ReadCommits runs git log on the given repo paths and returns parsed commits.
// Commits are grouped by repo in timestamp order within each repo, matching
// the multi-repo merge strategy from the spec.
func ReadCommits(repoPaths []string, since, until time.Time) ([]Commit, error) {
	var all []Commit
	for _, path := range repoPaths {
		commits, err := readCommitsFromRepo(path, since, until)
		if err != nil {
			return nil, fmt.Errorf("repo %s: %w", path, err)
		}
		all = append(all, commits...)
	}
	return all, nil
}

func readCommitsFromRepo(repoPath string, since, until time.Time) ([]Commit, error) {
	// Expand ~ shorthand safely without shell interpolation.
	if strings.HasPrefix(repoPath, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			repoPath = home + repoPath[1:]
		}
	}

	// Validate the path exists before invoking git to give a clear error.
	if _, err := os.Stat(repoPath); err != nil {
		return nil, fmt.Errorf("path does not exist: %s", repoPath)
	}

	sinceStr := since.Format(time.RFC3339)
	untilStr := until.Format(time.RFC3339)

	// COMMIT: prefix lets us reliably split the interleaved --numstat output.
	cmd := exec.Command("git",
		"-C", repoPath,
		"log",
		"--since="+sinceStr,
		"--until="+untilStr,
		`--pretty=format:COMMIT:%h|%s|%D|%ai`,
		"--numstat",
	)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			// An uninitialised repo (no commits yet) is not an error worth
			// aborting the whole run; just return nothing for this path.
			if strings.Contains(stderr, "does not have any commits") ||
				strings.Contains(stderr, "bad default revision") {
				return nil, nil
			}
			return nil, fmt.Errorf("git log failed: %s", stderr)
		}
		return nil, err
	}

	return parseGitLog(string(out))
}

// parseGitLog parses the interleaved --pretty=format:COMMIT:... --numstat output.
func parseGitLog(output string) ([]Commit, error) {
	var commits []Commit
	var current *Commit

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "COMMIT:") {
			if current != nil {
				commits = append(commits, *current)
			}
			// Format: COMMIT:<hash>|<subject>|<decoration>|<author_date>
			rest := line[len("COMMIT:"):]
			parts := strings.SplitN(rest, "|", 4)
			if len(parts) < 4 {
				continue
			}
			current = &Commit{
				Hash:      strings.TrimSpace(parts[0]),
				Message:   strings.TrimSpace(parts[1]),
				Branch:    extractBranchFromDecoration(parts[2]),
				Timestamp: normaliseTimestamp(parts[3]),
			}
			continue
		}

		// numstat lines: "<insertions>\t<deletions>\t<path>"
		// Binary files show "-\t-\t<path>"; we record 0 for those.
		if current != nil && !isAllWhitespace(line) {
			parts := strings.SplitN(line, "\t", 3)
			if len(parts) == 3 {
				ins, _ := strconv.Atoi(parts[0])
				del, _ := strconv.Atoi(parts[1])
				current.FilesChanged = append(current.FilesChanged, FileChange{
					Path:       strings.TrimSpace(parts[2]),
					Insertions: ins,
					Deletions:  del,
				})
			}
		}
	}
	if current != nil {
		commits = append(commits, *current)
	}
	return commits, scanner.Err()
}

// extractBranchFromDecoration extracts a local branch name from git's %D decoration.
//
// Examples:
//
//	"HEAD -> feat/ENG-402-oauth-token-fix, origin/feat/ENG-402-oauth-token-fix"
//	"tag: v1.0.0, main, origin/main"
//	"" — commit in the middle of history (no decoration)
func extractBranchFromDecoration(decoration string) string {
	if decoration == "" {
		return ""
	}
	// Prefer the explicit HEAD -> ref.
	for _, ref := range strings.Split(decoration, ",") {
		ref = strings.TrimSpace(ref)
		if strings.HasPrefix(ref, "HEAD -> ") {
			return strings.TrimPrefix(ref, "HEAD -> ")
		}
	}
	// Fallback: first local (non-remote, non-tag) ref.
	for _, ref := range strings.Split(decoration, ",") {
		ref = strings.TrimSpace(ref)
		if strings.HasPrefix(ref, "tag:") {
			continue
		}
		if strings.Contains(ref, "/") {
			continue // remote ref like origin/main
		}
		return ref
	}
	return ""
}

// normaliseTimestamp converts git's %ai output ("2006-01-02 15:04:05 -0700")
// to RFC3339 ("2006-01-02T15:04:05-07:00").
func normaliseTimestamp(raw string) string {
	raw = strings.TrimSpace(raw)
	t, err := time.Parse("2006-01-02 15:04:05 -0700", raw)
	if err != nil {
		return raw
	}
	return t.Format(time.RFC3339)
}

func isAllWhitespace(s string) bool {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

