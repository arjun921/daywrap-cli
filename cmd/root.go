package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/arjun921/daywrap-cli/internal"
	"github.com/spf13/cobra"
)

var (
	since   string
	until   string
	repos   []string
	weekly  bool
	rawJSON bool
)

var rootCmd = &cobra.Command{
	Use:   "daywrap",
	Short: "Generate animated QR standup data from local git history",
	Long: `DayWrap reads your local git history, enriches commits with ticket IDs,
compresses the payload, and renders it as animated QR codes for scanning
with the DayWrap mobile app. No network required.`,
	RunE: run,
}

func run(cmd *cobra.Command, args []string) error {
	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// Resolve date range.
	sinceT, untilT, err := resolveDateRange(cfg)
	if err != nil {
		return err
	}

	// Resolve repo paths: --repo flags > config > current directory.
	repoPaths := repos
	if len(repoPaths) == 0 {
		repoPaths = cfg.Repos
	}
	if len(repoPaths) == 0 {
		cwd, werr := os.Getwd()
		if werr != nil {
			return fmt.Errorf("cannot determine working directory: %w", werr)
		}
		repoPaths = []string{cwd}
	}

	// Expand each path: use it directly if it's a git repo, otherwise
	// walk subdirectories to discover all nested git repositories.
	repoPaths, err = internal.DiscoverRepos(repoPaths)
	if err != nil {
		return fmt.Errorf("repo discovery: %w", err)
	}
	if len(repoPaths) == 0 {
		return fmt.Errorf("no git repositories found in the given paths")
	}

	// 1. Read git history.
	commits, err := internal.ReadCommits(repoPaths, sinceT, untilT)
	if err != nil {
		return fmt.Errorf("git: %w", err)
	}

	// 2. Enrich with ticket IDs (and optionally Jira titles).
	commits = internal.EnrichCommits(commits, cfg)

	// 3. Build payload.
	payload := buildPayload(commits, sinceT, untilT)

	// --raw: print JSON and exit.
	if rawJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}

	// 4. Compress.
	encoded, err := internal.Compress(payload)
	if err != nil {
		return fmt.Errorf("compress: %w", err)
	}

	// 5. Chunk — size each frame to fit the current terminal.
	termCols, termRows := internal.GetTermSize()
	chunkSize := internal.OptimalChunkSize(termCols, termRows)
	chunks := internal.Chunk(encoded, chunkSize)

	// 6. Display as animated QR.
	return internal.Display(chunks)
}

// buildPayload assembles the full Payload struct from enriched commits.
func buildPayload(commits []internal.Commit, since, until time.Time) *internal.Payload {
	// Collect unique branch names.
	branchSet := make(map[string]struct{})
	totalIns, totalDel, totalFiles := 0, 0, 0
	for _, c := range commits {
		if c.Branch != "" {
			branchSet[c.Branch] = struct{}{}
		}
		for _, f := range c.FilesChanged {
			totalIns += f.Insertions
			totalDel += f.Deletions
			totalFiles++
		}
	}
	branches := make([]string, 0, len(branchSet))
	for b := range branchSet {
		branches = append(branches, b)
	}
	sort.Strings(branches)

	return &internal.Payload{
		Version:     "1.0",
		GeneratedAt: time.Now().Format(time.RFC3339),
		Period: internal.Period{
			Start: since.Format(time.RFC3339),
			End:   until.Format(time.RFC3339),
		},
		Commits: commits,
		Stats: internal.SummaryStats{
			TotalCommits:      len(commits),
			TotalFilesChanged: totalFiles,
			TotalInsertions:   totalIns,
			TotalDeletions:    totalDel,
			BranchesActive:    branches,
		},
	}
}

// resolveDateRange returns the since/until time.Time values based on flags.
// It uses the package-level since/until string variables populated by cobra.
func resolveDateRange(_ *internal.Config) (sinceT, untilT time.Time, err error) {
	now := time.Now()

	if weekly {
		// Current week: Monday 00:00:00 → today 23:59:59.
		wd := int(now.Weekday())
		if wd == 0 {
			wd = 7 // treat Sunday as day 7
		}
		monday := now.AddDate(0, 0, -(wd - 1))
		sinceT = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, now.Location())
		untilT = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
		return
	}

	// Parse explicit --since flag.
	if since != "" {
		sinceT, err = parseDateFlag(since)
		if err != nil {
			return
		}
	} else {
		// Default: midnight today.
		sinceT = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}

	// Parse explicit --until flag.
	if until != "" {
		untilT, err = parseDateFlag(until)
		if err != nil {
			return
		}
		// --until is inclusive: extend to end of that day.
		untilT = time.Date(untilT.Year(), untilT.Month(), untilT.Day(), 23, 59, 59, 0, untilT.Location())
	} else {
		// Default: current moment.
		untilT = now
	}
	return
}

// parseDateFlag parses a YYYY-MM-DD string in local time.
func parseDateFlag(s string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q — use YYYY-MM-DD format", s)
	}
	return t, nil
}

// Execute is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVar(&since, "since", "", "Start date (YYYY-MM-DD), defaults to today midnight")
	rootCmd.Flags().StringVar(&until, "until", "", "End date (YYYY-MM-DD), defaults to now")
	rootCmd.Flags().StringArrayVar(&repos, "repo", nil, "Repo path(s) to scan (repeatable)")
	rootCmd.Flags().BoolVar(&weekly, "weekly", false, "Scan current week (Monday to today) instead of today")
	rootCmd.Flags().BoolVar(&rawJSON, "raw", false, "Output raw JSON instead of QR codes (debug)")
}
