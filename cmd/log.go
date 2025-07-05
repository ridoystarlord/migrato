package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ridoystarlord/migrato/introspect"
	"github.com/ridoystarlord/migrato/runner"
)

var (
	logLimit int
	logFollow bool
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show recent migration activities",
	Long: `Show recent migration activities and logs.

Examples:
  migrato log                    # Show recent migration logs
  migrato log --limit 20         # Show last 20 log entries
  migrato log --follow           # Follow logs in real-time (future feature)
`,
	Run: func(cmd *cobra.Command, args []string) {
		// Connect to database
		db, err := introspect.Connect()
		if err != nil {
			fmt.Printf("❌ Error connecting to database: %v\n", err)
			os.Exit(1)
		}
		defer db.Close(context.Background())

		// Get recent logs
		logs, err := runner.GetMigrationLogs(db, logLimit)
		if err != nil {
			fmt.Printf("❌ Error getting migration logs: %v\n", err)
			os.Exit(1)
		}

		if len(logs) == 0 {
			fmt.Println("📋 No migration logs found")
			return
		}

		// Sort by timestamp (newest first)
		sort.Slice(logs, func(i, j int) bool {
			return logs[i].Timestamp.After(logs[j].Timestamp)
		})

		showMigrationLogs(logs)
	},
}

func showMigrationLogs(logs []runner.MigrationLog) {
	green := color.New(color.FgGreen, color.Bold)
	yellow := color.New(color.FgYellow, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	blue := color.New(color.FgBlue, color.Bold)
	cyan := color.New(color.FgCyan)

	fmt.Println("📋 Recent Migration Activities")
	fmt.Println(strings.Repeat("=", 60))

	for i, log := range logs {
		fmt.Printf("\n%d. ", i+1)
		
		// Level indicator
		switch log.Level {
		case "INFO":
			blue.Print("ℹ️  ")
		case "WARN":
			yellow.Print("⚠️  ")
		case "ERROR":
			red.Print("❌ ")
		case "SUCCESS":
			green.Print("✅ ")
		default:
			fmt.Print("📝 ")
		}

		// Timestamp
		cyan.Printf("[%s] ", log.Timestamp.Format("2006-01-02 15:04:05"))
		
		// Message
		fmt.Printf("%s", log.Message)
		
		// User if available
		if log.User != "" {
			fmt.Printf(" (by %s)", log.User)
		}
		
		fmt.Println()
		
		// Additional details if available
		if log.Details != "" {
			cyan.Printf("   📄 Details: %s\n", log.Details)
		}
	}

	// Summary
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("📊 Showing %d recent log entries\n", len(logs))
}

func init() {
	logCmd.Flags().IntVarP(&logLimit, "limit", "l", 50, "Limit number of log entries to show")
	logCmd.Flags().BoolVarP(&logFollow, "follow", "f", false, "Follow logs in real-time (future feature)")
} 