package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ridoystarlord/migrato/database"
	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check database connectivity",
	Long: `Check if the database is accessible and responsive.

Examples:
  migrato health                    # Check default database connection
  migrato health --timeout 10s      # Set custom timeout
`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := checkDatabaseHealth(); err != nil {
			fmt.Printf("❌ Database health check failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Database is healthy and accessible")
	},
}

var healthTimeout time.Duration

func init() {
	healthCmd.Flags().DurationVarP(&healthTimeout, "timeout", "t", 5*time.Second, "Timeout for health check")
}

func checkDatabaseHealth() error {
	// Get database pool with timeout
	ctx, cancel := context.WithTimeout(context.Background(), healthTimeout)
	defer cancel()

	pool, err := database.GetPool()
	if err != nil {
		return fmt.Errorf("failed to get database pool: %v", err)
	}

	// Test connection with a simple query
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	// Check if schema_migrations table exists (indicates migrato is set up)
	var tableExists bool
	query := `SELECT EXISTS (
		SELECT FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_name = 'schema_migrations'
	)`
	
	if err := pool.QueryRow(ctx, query).Scan(&tableExists); err != nil {
		return fmt.Errorf("failed to check schema_migrations table: %v", err)
	}

	if !tableExists {
		fmt.Println("⚠️  Database is accessible but schema_migrations table not found")
		fmt.Println("   Run 'migrato init' to set up the migration tracking table")
		return nil
	}

	// Check migration status
	var count int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		return fmt.Errorf("failed to count migrations: %v", err)
	}

	fmt.Printf("📊 Found %d applied migrations\n", count)

	return nil
} 