package cmd

import (
	"fmt"
	"os"

	"github.com/ridoystarlord/migrato/runner"
	"github.com/spf13/cobra"
)

var dryRunMigrate bool

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply pending migrations",
	Run: func(cmd *cobra.Command, args []string) {

		if dryRunMigrate {
			err := runner.PreviewMigrations()
			if err != nil {
				fmt.Println("❌ Dry run failed:", err)
				os.Exit(1)
			}
			return
		}

		err := runner.ApplyMigrations()
		if err != nil {
			fmt.Println("❌ Migration failed:", err)
			os.Exit(1)
		}
	},
}

func init() {
	migrateCmd.Flags().BoolVar(&dryRunMigrate, "dry-run", false, "Preview the SQL that would be executed without applying migrations")
}
