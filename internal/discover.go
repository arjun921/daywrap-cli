package internal

// discover.go — finds git repositories under given root paths.

import (
	"os"
	"path/filepath"
)

const maxDiscoverDepth = 3

// DiscoverRepos expands each root path into concrete git repository paths.
//
// Rules:
//   - If the root itself is a git repo (.git exists), it is used as-is.
//   - Otherwise the function walks up to maxDiscoverDepth sub-directories,
//     collecting every directory that contains a .git entry (file or dir),
//     without recursing further into found repos.
//   - Duplicate paths are deduplicated.
//   - Hidden directories, node_modules and vendor are skipped during traversal.
func DiscoverRepos(roots []string) ([]string, error) {
	seen := make(map[string]struct{})
	var result []string

	add := func(p string) {
		if _, ok := seen[p]; !ok {
			seen[p] = struct{}{}
			result = append(result, p)
		}
	}

	for _, root := range roots {
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}
		if isGitRepo(abs) {
			add(abs)
			continue
		}
		found, err := walkRepos(abs, 0)
		if err != nil {
			return nil, err
		}
		for _, r := range found {
			add(r)
		}
	}
	return result, nil
}

// isGitRepo reports whether dir contains a .git entry (file or directory).
// A .git file indicates a submodule; a .git directory indicates a normal repo.
func isGitRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

// walkRepos recursively searches dir for git repositories up to maxDiscoverDepth.
// It stops descending into a directory once it has been identified as a repo.
func walkRepos(dir string, depth int) ([]string, error) {
	if depth > maxDiscoverDepth {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		// Silently skip directories we can't read (permissions etc.).
		return nil, nil
	}

	var repos []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if shouldSkip(name) {
			continue
		}
		child := filepath.Join(dir, name)
		if isGitRepo(child) {
			repos = append(repos, child)
			// Don't recurse into a found repo (avoids double-counting submodules).
		} else {
			sub, err := walkRepos(child, depth+1)
			if err != nil {
				return nil, err
			}
			repos = append(repos, sub...)
		}
	}
	return repos, nil
}

// shouldSkip returns true for directory names that are never project roots.
func shouldSkip(name string) bool {
	switch name {
	case "node_modules", "vendor", ".git":
		return true
	}
	// Skip hidden directories (e.g. .cache, .venv, .build).
	return len(name) > 0 && name[0] == '.'
}
