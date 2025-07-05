package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/ridoystarlord/migrato/utils"
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
	// Load environment and get database connection string
	utils.LoadEnv()
	dsn := utils.GetDatabaseURL()
	if dsn == "" {
		return fmt.Errorf("database connection string not found. Set DATABASE_URL environment variable")
	}

	// Connect to database with timeout
	ctx, cancel := context.WithTimeout(context.Background(), healthTimeout)
	defer cancel()

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	// Test connection with a simple query
	if err := conn.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	// Check if schema_migrations table exists (indicates migrato is set up)
	var tableExists bool
	query := `SELECT EXISTS (
		SELECT FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_name = 'schema_migrations'
	)`
	
	if err := conn.QueryRow(ctx, query).Scan(&tableExists); err != nil {
		return fmt.Errorf("failed to check schema_migrations table: %v", err)
	}

	if !tableExists {
		fmt.Println("⚠️  Database is accessible but schema_migrations table not found")
		fmt.Println("   Run 'migrato init' to set up the migration tracking table")
		return nil
	}

	// Check migration status
	var count int
	if err := conn.QueryRow(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		return fmt.Errorf("failed to count migrations: %v", err)
	}

	fmt.Printf("📊 Found %d applied migrations\n", count)

	return nil
} 