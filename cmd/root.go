package cmd

import (
	"fmt"
	"os"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("daywrap: not yet implemented")
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVar(&since, "since", "", "Start date (YYYY-MM-DD), defaults to today")
	rootCmd.Flags().StringVar(&until, "until", "", "End date (YYYY-MM-DD), defaults to today")
	rootCmd.Flags().StringArrayVar(&repos, "repo", nil, "Repo path(s) to scan (repeatable)")
	rootCmd.Flags().BoolVar(&weekly, "weekly", false, "Generate weekly summary instead of daily")
	rootCmd.Flags().BoolVar(&rawJSON, "raw", false, "Output raw JSON instead of QR codes (debug)")
}
