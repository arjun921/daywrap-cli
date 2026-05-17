package internal

// git.go — reads local git log and parses it into the DayWrap commit schema.

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// ReadCommits runs git log on the given repo paths and returns parsed commits.
// author filters to commits by that author pattern (git --author= format); pass
// an empty string to include all authors.
func ReadCommits(repoPaths []string, runDir string, since, until time.Time, author string) ([]Commit, error) {
	runAbs, err := filepath.Abs(runDir)
	if err != nil {
		return nil, fmt.Errorf("resolve run directory: %w", err)
	}

	var all []Commit
	for _, repoPath := range repoPaths {
		relRepoPath := relativeRepoPath(runAbs, repoPath)
		commits, err := readCommitsFromRepo(repoPath, relRepoPath, since, until, author)
		if err != nil {
			return nil, fmt.Errorf("repo %s: %w", repoPath, err)
		}
		all = append(all, commits...)
	}
	return all, nil
}

// CurrentGitAuthor returns the email from `git config user.email`, falling back
// to the name if no email is configured, or an empty string if neither is set.
func CurrentGitAuthor() string {
	for _, field := range []string{"user.email", "user.name"} {
		out, err := exec.Command("git", "config", "--global", field).Output()
		if err == nil {
			if v := strings.TrimSpace(string(out)); v != "" {
				return v
			}
		}
	}
	return ""
}

func readCommitsFromRepo(repoPath, relativePath string, since, until time.Time, author string) ([]Commit, error) {
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

	repoName := resolveRepoName(repoPath)

	sinceStr := since.Format(time.RFC3339)
	untilStr := until.Format(time.RFC3339)

	// COMMIT: prefix lets us reliably split the interleaved --numstat output.
	args := []string{
		"-C", repoPath,
		"log",
		"--since=" + sinceStr,
		"--until=" + untilStr,
		`--pretty=format:COMMIT:%h|%s|%D|%ai`,
		"--numstat",
	}
	if author != "" {
		args = append(args, "--author="+author)
	}
	cmd := exec.Command("git", args...)
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

	return parseGitLog(string(out), repoName, relativePath)
}

func relativeRepoPath(runDir, repoPath string) string {
	repoAbs, err := filepath.Abs(repoPath)
	if err != nil {
		return filepath.Clean(repoPath)
	}
	rel, err := filepath.Rel(runDir, repoAbs)
	if err != nil {
		return repoAbs
	}
	if rel == "" {
		return "."
	}
	return rel
}

// resolveRepoName returns the canonical repository name derived from
// `git remote get-url origin`, falling back to the local folder name.
//
// Examples:
//   - git@github.com:arjun921/daywrap-cli.git -> daywrap-cli
//   - https://github.com/arjun921/daywrap-cli.git -> daywrap-cli
//   - ssh://git@github.com/arjun921/daywrap-cli -> daywrap-cli
func resolveRepoName(repoPath string) string {
	fallback := strings.TrimSpace(filepath.Base(repoPath))

	out, err := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin").Output()
	if err != nil {
		return fallback
	}

	url := strings.TrimSpace(string(out))
	if url == "" {
		return fallback
	}

	url = strings.TrimSuffix(url, "/")

	// SCP-style URLs use "host:owner/repo.git" without a URI scheme.
	if strings.Contains(url, ":") && !strings.Contains(url, "://") {
		if idx := strings.LastIndex(url, ":"); idx >= 0 && idx+1 < len(url) {
			url = url[idx+1:]
		}
	}

	name := path.Base(url)
	name = strings.TrimSuffix(name, ".git")
	if name == "" || name == "." || name == "/" {
		return fallback
	}

	return strings.TrimSpace(name)
}

// parseGitLog parses the interleaved --pretty=format:COMMIT:... --numstat output.
func parseGitLog(output string, repo, repoPath string) ([]Commit, error) {
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
				Repo:      repo,
				RepoPath:  repoPath,
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
	// Post-process: filter out unhelpful commits to save LLM context.
	// Exclude merge housekeeping, explicit submodule update chores, and
	// commits that only touch the .gitmodules file.
	filtered := make([]Commit, 0, len(commits))
	for _, c := range commits {
		if shouldExcludeCommit(c) {
			continue
		}
		filtered = append(filtered, c)
	}
	return filtered, scanner.Err()
}

// shouldExcludeCommit returns true for commits that are safe to drop before
// sending payloads to the on-device model: merge housekeeping, submodule
// updates, and commits that only modify .gitmodules.
func shouldExcludeCommit(c Commit) bool {
	msg := strings.TrimSpace(c.Message)
	if msg == "" {
		return false
	}

	// Common merge messages we don't need in the standup context.
	if strings.HasPrefix(msg, "Merge remote-tracking") ||
		strings.HasPrefix(msg, "Merge branch") ||
		strings.HasPrefix(msg, "Merge pull request") {
		return true
	}

	lower := strings.ToLower(msg)
	if strings.HasPrefix(lower, "chore: update submodule") || strings.Contains(lower, "update submodule") {
		return true
	}

	// If the commit only changes .gitmodules, drop it.
	if len(c.FilesChanged) > 0 {
		onlyGitmodules := true
		for _, f := range c.FilesChanged {
			p := strings.TrimSpace(f.Path)
			if p == "" {
				continue
			}
			if p != ".gitmodules" && p != "./.gitmodules" {
				onlyGitmodules = false
				break
			}
		}
		if onlyGitmodules {
			return true
		}
	}

	return false
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

