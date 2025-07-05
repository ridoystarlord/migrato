package cmd

import (
	"fmt"
	"os"

	"github.com/ridoystarlord/migrato/runner"
	"github.com/spf13/cobra"
)

var steps int

func init() {
	rollbackCmd.Flags().IntVarP(&steps, "steps", "s", 1, "Number of migrations to rollback")
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback migrations",
	Long: `Rollback the last migration or multiple migrations.

Examples:
  migrato rollback          # Rollback the last migration
  migrato rollback --steps=3 # Rollback the last 3 migrations
  migrato rollback -s 5      # Rollback the last 5 migrations
`,
	Run: func(cmd *cobra.Command, args []string) {
		if steps < 1 {
			fmt.Println("❌ Steps must be at least 1")
			os.Exit(1)
		}

		err := runner.RollbackMigrations(steps)
		if err != nil {
			fmt.Println("❌ Rollback failed:", err)
			os.Exit(1)
		}

		if steps == 1 {
			fmt.Println("✅ Rolled back 1 migration.")
		} else {
			fmt.Printf("✅ Rolled back %d migrations.\n", steps)
		}
	},
} 