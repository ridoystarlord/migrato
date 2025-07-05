package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ridoystarlord/migrato/introspect"
	"github.com/ridoystarlord/migrato/runner"
)

var (
	historyLimit int
	historyTable string
	historyDetailed bool
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show detailed migration history",
	Long: `Show detailed migration history with timestamps, execution times, and user information.

Examples:
  migrato history                    # Show all migration history
  migrato history --limit 10         # Show last 10 migrations
  migrato history --table users      # Show migrations for specific table
  migrato history --detailed         # Show detailed information
`,
	Run: func(cmd *cobra.Command, args []string) {
		// Connect to database
		db, err := introspect.Connect()
		if err != nil {
			fmt.Printf("❌ Error connecting to database: %v\n", err)
			os.Exit(1)
		}
		defer db.Close(context.Background())

		// Get migration history
		history, err := runner.GetMigrationHistory(db, historyLimit, historyTable)
		if err != nil {
			fmt.Printf("❌ Error getting migration history: %v\n", err)
			os.Exit(1)
		}

		if len(history) == 0 {
			fmt.Println("📋 No migration history found")
			return
		}

		// Sort by timestamp (newest first)
		sort.Slice(history, func(i, j int) bool {
			return history[i].ExecutedAt.After(history[j].ExecutedAt)
		})

		showMigrationHistory(history, historyDetailed)
	},
}

func showMigrationHistory(history []runner.MigrationRecord, detailed bool) {
	fmt.Println("📋 Migration History")
	fmt.Println(strings.Repeat("=", 60))

	if detailed {
		showDetailedHistory(history)
	} else {
		showSummaryHistory(history)
	}
}

func showDetailedHistory(history []runner.MigrationRecord) {
	green := color.New(color.FgGreen, color.Bold)
	yellow := color.New(color.FgYellow, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	blue := color.New(color.FgBlue, color.Bold)
	cyan := color.New(color.FgCyan)

	for i, record := range history {
		fmt.Printf("\n%d. ", i+1)
		
		// Status indicator
		if record.Status == "success" {
			green.Print("✅ ")
		} else if record.Status == "failed" {
			red.Print("❌ ")
		} else {
			yellow.Print("⚠️ ")
		}

		// Migration name
		blue.Printf("%s\n", record.MigrationName)
		
		// Timestamp
		cyan.Printf("   📅 Executed: %s\n", record.ExecutedAt.Format("2006-01-02 15:04:05"))
		
		// Execution time
		if record.ExecutionTime > 0 {
			cyan.Printf("   ⏱️  Duration: %v\n", record.ExecutionTime)
		}
		
		// User
		if record.ExecutedBy != "" {
			cyan.Printf("   👤 User: %s\n", record.ExecutedBy)
		}
		
		// Status
		cyan.Printf("   📊 Status: %s\n", record.Status)
		
		// Error message if failed
		if record.Status == "failed" && record.ErrorMessage != "" {
			red.Printf("   💥 Error: %s\n", record.ErrorMessage)
		}
		
		// Checksum
		if record.Checksum != "" {
			cyan.Printf("   🔍 Checksum: %s\n", record.Checksum[:8]+"...")
		}
	}
}

func showSummaryHistory(history []runner.MigrationRecord) {
	green := color.New(color.FgGreen, color.Bold)
	yellow := color.New(color.FgYellow, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	blue := color.New(color.FgBlue, color.Bold)

	fmt.Printf("%-4s %-8s %-25s %-12s %-10s %s\n", "ID", "Status", "Migration", "Duration", "User", "Date")
	fmt.Println(strings.Repeat("-", 80))

	for i, record := range history {
		// Status indicator
		var status string
		if record.Status == "success" {
			status = green.Sprint("✅")
		} else if record.Status == "failed" {
			status = red.Sprint("❌")
		} else {
			status = yellow.Sprint("⚠️")
		}

		// Duration
		var duration string
		if record.ExecutionTime > 0 {
			duration = record.ExecutionTime.String()
		} else {
			duration = "N/A"
		}

		// User
		user := record.ExecutedBy
		if user == "" {
			user = "N/A"
		}

		// Migration name (truncate if too long)
		migrationName := record.MigrationName
		if len(migrationName) > 23 {
			migrationName = migrationName[:20] + "..."
		}

		fmt.Printf("%-4d %-8s %-25s %-12s %-10s %s\n",
			i+1,
			status,
			blue.Sprint(migrationName),
			duration,
			user,
			record.ExecutedAt.Format("2006-01-02 15:04"),
		)
	}

	// Summary statistics
	fmt.Println(strings.Repeat("-", 80))
	
	successCount := 0
	failedCount := 0
	totalDuration := time.Duration(0)
	
	for _, record := range history {
		if record.Status == "success" {
			successCount++
		} else if record.Status == "failed" {
			failedCount++
		}
		if record.ExecutionTime > 0 {
			totalDuration += record.ExecutionTime
		}
	}

	fmt.Printf("📊 Summary: %d total, %d successful, %d failed\n", 
		len(history), successCount, failedCount)
	
	if totalDuration > 0 {
		fmt.Printf("⏱️  Total execution time: %v\n", totalDuration)
	}
}

func init() {
	historyCmd.Flags().IntVarP(&historyLimit, "limit", "l", 0, "Limit number of records to show (0 = all)")
	historyCmd.Flags().StringVarP(&historyTable, "table", "t", "", "Filter by table name")
	historyCmd.Flags().BoolVarP(&historyDetailed, "detailed", "d", false, "Show detailed information")
} 