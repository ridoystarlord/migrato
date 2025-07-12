package cmd

import (
	"fmt"
	"os"

	"github.com/ridoystarlord/migrato/runner"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show applied and pending migrations",
	Run: func(cmd *cobra.Command, args []string) {

		applied, pending, failed, err := runner.Status()
		if err != nil {
			fmt.Println("❌ Status error:", err)
			os.Exit(1)
		}

		fmt.Println("✅ Applied migrations:")
		for _, f := range applied {
			fmt.Println("   -", f)
		}

		if len(failed) > 0 {
			fmt.Println("\n❌ Failed migrations:")
			for _, f := range failed {
				fmt.Printf("   - %s: %s\n", f.MigrationName, f.ErrorMessage)
			}
		}

		fmt.Println("\n🕒 Pending migrations:")
		for _, f := range pending {
			fmt.Println("   -", f)
		}
	},
}
