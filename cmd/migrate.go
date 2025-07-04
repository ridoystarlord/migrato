package cmd

import (
	"fmt"
	"os"

	"github.com/ridoystarlord/migrato/runner"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply pending migrations",
	Run: func(cmd *cobra.Command, args []string) {

		err := runner.ApplyMigrations()
		if err != nil {
			fmt.Println("❌ Migration failed:", err)
			os.Exit(1)
		}
	},
}
