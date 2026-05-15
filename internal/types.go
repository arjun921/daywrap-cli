package internal

// Payload is the top-level JSON structure transferred via animated QR.
type Payload struct {
	Version     string       `json:"version"`
	GeneratedAt string       `json:"generated_at"`
	Period      Period       `json:"period"`
	Commits     []Commit     `json:"commits"`
	Stats       SummaryStats `json:"summary_stats"`
}

// Period holds the start/end timestamps for the scanned activity window.
type Period struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// Commit represents a single git commit enriched with file stats and ticket info.
type Commit struct {
	Hash         string       `json:"hash"`
	Message      string       `json:"message"`
	Branch       string       `json:"branch"`
	Timestamp    string       `json:"timestamp"`
	FilesChanged []FileChange `json:"files_changed"`
	Ticket       *Ticket      `json:"ticket,omitempty"`
}

// FileChange holds per-file diff statistics for a commit.
type FileChange struct {
	Path       string `json:"path"`
	Insertions int    `json:"insertions"`
	Deletions  int    `json:"deletions"`
}

// Ticket holds the extracted ticket ID and (optionally) its title from Jira.
type Ticket struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
}

// SummaryStats aggregates high-level metrics across all commits in the payload.
type SummaryStats struct {
	TotalCommits      int      `json:"total_commits"`
	TotalFilesChanged int      `json:"total_files_changed"`
	TotalInsertions   int      `json:"total_insertions"`
	TotalDeletions    int      `json:"total_deletions"`
	BranchesActive    []string `json:"branches_active"`
}
