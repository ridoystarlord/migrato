package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Version = "v1.0.0"

var rootCmd = &cobra.Command{
	Use:   "migrato",
	Short: "A lightweight Prisma-like migration tool for Go",
	Long: `migrato is a simple migration CLI.

Examples:

  migrato init
  migrato generate
  migrato migrate
`,
Version: Version,
}

// Execute runs the CLI
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("❌", err)
		os.Exit(1)
	}
}

// Register subcommands
func init() {
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(generateStructsCmd)
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(docsCmd)
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(logCmd)
}
