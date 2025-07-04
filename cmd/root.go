package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mymigrate",
	Short: "A lightweight Prisma-like migration tool for Go",
	Long: `mymigrate is a simple migration CLI.

Examples:

  mymigrate init
  mymigrate generate
  mymigrate migrate
`,
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
}
