package runner

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/ridoystarlord/migrato/database"
)

// MigrationRecord represents a migration execution record
type MigrationRecord struct {
	ID             int
	MigrationName  string
	ExecutedAt     time.Time
	ExecutionTime  time.Duration
	ExecutedBy     string
	Status         string
	ErrorMessage   string
	Checksum       string
	TableAffected  string
}

// MigrationLog represents a migration log entry
type MigrationLog struct {
	ID        int
	Timestamp time.Time
	Level     string
	Message   string
	User      string
	Details   string
	MigrationName string
}

func getConn() (*pgx.Conn, context.Context, error) {
	ctx := context.Background()
	conn, err := database.GetConnection(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get connection: %v", err)
	}
	return conn, ctx, nil
}

func ensureMigrationsTable(conn *pgx.Conn, ctx context.Context) error {
	fmt.Println("🔧 Ensuring migration tables exist...")
	
	// Create enhanced migrations table with history tracking
	_, err := conn.Exec(ctx, `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		id SERIAL PRIMARY KEY,
		filename TEXT NOT NULL UNIQUE,
		applied_at TIMESTAMP DEFAULT now(),
		execution_time INTERVAL,
		executed_by TEXT,
		status TEXT DEFAULT 'success',
		error_message TEXT,
		checksum TEXT,
		table_affected TEXT
	);
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %v", err)
	}
	fmt.Println("✅ schema_migrations table ensured")

	// Create migration logs table
	_, err = conn.Exec(ctx, `
	CREATE TABLE IF NOT EXISTS migration_logs (
		id SERIAL PRIMARY KEY,
		timestamp TIMESTAMP DEFAULT now(),
		level TEXT NOT NULL,
		message TEXT NOT NULL,
		user_name TEXT,
		details TEXT,
		migration_name TEXT
	);
	`)
	if err != nil {
		return fmt.Errorf("failed to create migration_logs table: %v", err)
	}
	fmt.Println("✅ migration_logs table ensured")
	
	return nil
}

func getCurrentUser() string {
	currentUser, err := user.Current()
	if err != nil {
		return "unknown"
	}
	return currentUser.Username
}

func calculateChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

func logMigrationActivity(conn *pgx.Conn, ctx context.Context, level, message, migrationName, details string) error {
	userName := getCurrentUser()
	_, err := conn.Exec(ctx, `
		INSERT INTO migration_logs (level, message, user_name, migration_name, details)
		VALUES ($1, $2, $3, $4, $5)
	`, level, message, userName, migrationName, details)
	return err
}

func getAppliedMigrations(conn *pgx.Conn, ctx context.Context) (map[string]bool, error) {
	rows, err := conn.Query(ctx, `SELECT filename FROM schema_migrations WHERE status = 'success';`)
	if err != nil {
		return nil, fmt.Errorf("query applied migrations: %v", err)
	}
	defer rows.Close()

	applied := map[string]bool{}
	for rows.Next() {
		var fname string
		if err := rows.Scan(&fname); err != nil {
			return nil, fmt.Errorf("scan filename: %v", err)
		}
		applied[fname] = true
	}
	return applied, nil
}

func getAppliedMigrationsOrdered(conn *pgx.Conn, ctx context.Context) ([]string, error) {
	rows, err := conn.Query(ctx, `SELECT filename FROM schema_migrations WHERE status = 'success' ORDER BY applied_at DESC;`)
	if err != nil {
		return nil, fmt.Errorf("query applied migrations: %v", err)
	}
	defer rows.Close()

	var applied []string
	for rows.Next() {
		var fname string
		if err := rows.Scan(&fname); err != nil {
			return nil, fmt.Errorf("scan filename: %v", err)
		}
		applied = append(applied, fname)
	}
	return applied, nil
}

func getFailedMigrations(conn *pgx.Conn, ctx context.Context) ([]MigrationRecord, error) {
	rows, err := conn.Query(ctx, `SELECT filename, error_message FROM schema_migrations WHERE status = 'failed';`)
	if err != nil {
		return nil, fmt.Errorf("query failed migrations: %v", err)
	}
	defer rows.Close()

	var failed []MigrationRecord
	for rows.Next() {
		var record MigrationRecord
		if err := rows.Scan(&record.MigrationName, &record.ErrorMessage); err != nil {
			return nil, fmt.Errorf("scan failed migration: %v", err)
		}
		failed = append(failed, record)
	}
	return failed, nil
}

func getMigrationFiles() ([]string, error) {
	files, err := ioutil.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %v", err)
	}

	var filenames []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
			filenames = append(filenames, f.Name())
		}
	}
	sort.Strings(filenames) // Ensure in order
	return filenames, nil
}

func parseMigrationFile(filename string) (string, string, error) {
	content, err := os.ReadFile(filepath.Join("migrations", filename))
	if err != nil {
		return "", "", fmt.Errorf("read file %s: %v", filename, err)
	}

	contentStr := string(content)
	
	// Split content into up and down sections
	parts := strings.Split(contentStr, "-- Down Migration (Rollback)")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("migration file %s does not contain rollback section", filename)
	}

	upSection := parts[0]
	downSection := parts[1]

	// Extract SQL from up section (after "-- Up Migration")
	upParts := strings.Split(upSection, "-- Up Migration")
	if len(upParts) < 2 {
		return "", "", fmt.Errorf("migration file %s does not contain up migration section", filename)
	}

	// Extract SQL from down section (after "-- =======================")
	downParts := strings.Split(downSection, "-- =======================")
	if len(downParts) < 2 {
		return "", "", fmt.Errorf("migration file %s does not contain valid rollback section", filename)
	}

	upSQL := strings.TrimSpace(upParts[1])
	downSQL := strings.TrimSpace(downParts[1])

	return upSQL, downSQL, nil
}

func applyMigration(conn *pgx.Conn, ctx context.Context, filename string) error {
	startTime := time.Now()
	upSQL, _, err := parseMigrationFile(filename)
	if err != nil {
		return fmt.Errorf("parse migration file %s: %v", filename, err)
	}

	// Log migration start
	logMigrationActivity(conn, ctx, "INFO", fmt.Sprintf("Starting migration: %s", filename), filename, "Migration execution started")

	// Execute migration
	_, err = conn.Exec(ctx, upSQL)
	executionTime := time.Since(startTime)
	
	if err != nil {
		// Log failure
		logMigrationActivity(conn, ctx, "ERROR", fmt.Sprintf("Migration failed: %s", filename), filename, err.Error())
		
		// Record failed migration
		checksum := calculateChecksum(upSQL)
		userName := getCurrentUser()
		_, insertErr := conn.Exec(ctx, `
			INSERT INTO schema_migrations (filename, execution_time, executed_by, status, error_message, checksum)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, filename, executionTime, userName, "failed", err.Error(), checksum)
		
		if insertErr != nil {
			return fmt.Errorf("recording failed migration %s: %v", filename, insertErr)
		}
		
		return fmt.Errorf("executing migration %s: %v", filename, err)
	}

	// Log success
	logMigrationActivity(conn, ctx, "SUCCESS", fmt.Sprintf("Migration completed: %s", filename), filename, fmt.Sprintf("Execution time: %v", executionTime))

	// Record successful migration
	checksum := calculateChecksum(upSQL)
	userName := getCurrentUser()
	_, err = conn.Exec(ctx, `
		INSERT INTO schema_migrations (filename, execution_time, executed_by, status, checksum)
		VALUES ($1, $2, $3, $4, $5)
	`, filename, executionTime, userName, "success", checksum)
	
	if err != nil {
		return fmt.Errorf("recording migration %s: %v", filename, err)
	}

	return nil
}

func rollbackMigration(conn *pgx.Conn, ctx context.Context, filename string) error {
	startTime := time.Now()
	_, downSQL, err := parseMigrationFile(filename)
	if err != nil {
		return fmt.Errorf("parse migration file %s: %v", filename, err)
	}

	// Log rollback start
	logMigrationActivity(conn, ctx, "INFO", fmt.Sprintf("Starting rollback: %s", filename), filename, "Rollback execution started")

	// Execute rollback
	_, err = conn.Exec(ctx, downSQL)
	executionTime := time.Since(startTime)
	
	if err != nil {
		// Log failure
		logMigrationActivity(conn, ctx, "ERROR", fmt.Sprintf("Rollback failed: %s", filename), filename, err.Error())
		return fmt.Errorf("executing rollback for %s: %v", filename, err)
	}

	// Log success
	logMigrationActivity(conn, ctx, "SUCCESS", fmt.Sprintf("Rollback completed: %s", filename), filename, fmt.Sprintf("Execution time: %v", executionTime))

	// Remove migration record
	_, err = conn.Exec(ctx, `DELETE FROM schema_migrations WHERE filename = $1;`, filename)
	if err != nil {
		return fmt.Errorf("removing migration record for %s: %v", filename, err)
	}

	return nil
}

func ApplyMigrations() error {
	conn, ctx, err := getConn()
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	// Ensure tracking table exists
	if err := ensureMigrationsTable(conn, ctx); err != nil {
		return fmt.Errorf("ensure migrations table: %v", err)
	}

	// Check for failed migrations first
	failedMigrations, err := getFailedMigrations(conn, ctx)
	if err != nil {
		return fmt.Errorf("check failed migrations: %v", err)
	}
	
	if len(failedMigrations) > 0 {
		fmt.Println("❌ Found failed migrations that need to be resolved:")
		for _, migration := range failedMigrations {
			fmt.Printf("   - %s: %s\n", migration.MigrationName, migration.ErrorMessage)
		}
		fmt.Println("💡 Please fix the issues and run 'migrato migrate' again.")
		return fmt.Errorf("failed migrations detected")
	}

	// Get applied migrations
	applied, err := getAppliedMigrations(conn, ctx)
	if err != nil {
		return err
	}

	// Get all migration files
	files, err := getMigrationFiles()
	if err != nil {
		return err
	}

	var pending []string
	for _, f := range files {
		if !applied[f] {
			pending = append(pending, f)
		}
	}

	if len(pending) == 0 {
		fmt.Println("✅ No pending migrations.")
		return nil
	}

	fmt.Printf("Applying %d migration(s)...\n", len(pending))
	for _, f := range pending {
		fmt.Printf("Applying: %s\n", f)
		if err := applyMigration(conn, ctx, f); err != nil {
			return err
		}
	}

	fmt.Println("✅ All migrations applied.")
	return nil
}

func RollbackMigrations(steps int) error {
	conn, ctx, err := getConn()
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	// Ensure tracking table exists
	if err := ensureMigrationsTable(conn, ctx); err != nil {
		return fmt.Errorf("ensure migrations table: %v", err)
	}

	// Get applied migrations in reverse order (most recent first)
	applied, err := getAppliedMigrationsOrdered(conn, ctx)
	if err != nil {
		return err
	}

	if len(applied) == 0 {
		fmt.Println("✅ No migrations to rollback.")
		return nil
	}

	// Determine how many migrations to rollback
	toRollback := steps
	if toRollback > len(applied) {
		toRollback = len(applied)
		fmt.Printf("⚠️  Only %d migrations available, rolling back all.\n", len(applied))
	}

	// Get the migrations to rollback (most recent first)
	migrationsToRollback := applied[:toRollback]

	fmt.Printf("Rolling back %d migration(s)...\n", toRollback)
	for _, f := range migrationsToRollback {
		fmt.Printf("Rolling back: %s\n", f)
		if err := rollbackMigration(conn, ctx, f); err != nil {
			return err
		}
	}

	fmt.Println("✅ Rollback completed.")
	return nil
}

func Status() ([]string, []string, []MigrationRecord, error) {
	conn, ctx, err := getConn()
	if err != nil {
		return nil, nil, nil, err
	}
	defer conn.Close(ctx)

	if err := ensureMigrationsTable(conn, ctx); err != nil {
		return nil, nil, nil, err
	}

	appliedMap, err := getAppliedMigrations(conn, ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	var applied []string
	for k := range appliedMap {
		applied = append(applied, k)
	}

	files, err := getMigrationFiles()
	if err != nil {
		return nil, nil, nil, err
	}

	var pending []string
	for _, f := range files {
		if !appliedMap[f] {
			pending = append(pending, f)
		}
	}

	// Get failed migrations
	failed, err := getFailedMigrations(conn, ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	return applied, pending, failed, nil
}

// GetMigrationHistory retrieves migration history with optional filtering
func GetMigrationHistory(conn *pgx.Conn, limit int, tableFilter string) ([]MigrationRecord, error) {
	ctx := context.Background()
	
	query := `
		SELECT id, filename, applied_at, execution_time, executed_by, 
		       status, error_message, checksum, table_affected
		FROM schema_migrations
	`
	
	var args []interface{}
	argCount := 0
	
	if tableFilter != "" {
		argCount++
		query += fmt.Sprintf(" WHERE table_affected ILIKE $%d", argCount)
		args = append(args, "%"+tableFilter+"%")
	}
	
	query += " ORDER BY applied_at DESC"
	
	if limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
	}
	
	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query migration history: %v", err)
	}
	defer rows.Close()

	var records []MigrationRecord
	for rows.Next() {
		var record MigrationRecord
		var executionTime *time.Duration
		
		err := rows.Scan(
			&record.ID,
			&record.MigrationName,
			&record.ExecutedAt,
			&executionTime,
			&record.ExecutedBy,
			&record.Status,
			&record.ErrorMessage,
			&record.Checksum,
			&record.TableAffected,
		)
		if err != nil {
			return nil, fmt.Errorf("scan migration record: %v", err)
		}
		
		if executionTime != nil {
			record.ExecutionTime = *executionTime
		}
		
		records = append(records, record)
	}
	
	return records, nil
}

// GetMigrationLogs retrieves migration logs with optional limit
func GetMigrationLogs(conn *pgx.Conn, limit int) ([]MigrationLog, error) {
	ctx := context.Background()
	
	query := `
		SELECT id, timestamp, level, message, user_name, details, migration_name
		FROM migration_logs
		ORDER BY timestamp DESC
	`
	
	var args []interface{}
	if limit > 0 {
		query += " LIMIT $1"
		args = append(args, limit)
	}
	
	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query migration logs: %v", err)
	}
	defer rows.Close()

	var logs []MigrationLog
	for rows.Next() {
		var log MigrationLog
		
		err := rows.Scan(
			&log.ID,
			&log.Timestamp,
			&log.Level,
			&log.Message,
			&log.User,
			&log.Details,
			&log.MigrationName,
		)
		if err != nil {
			return nil, fmt.Errorf("scan migration log: %v", err)
		}
		
		logs = append(logs, log)
	}
	
	return logs, nil
}

// PreviewMigrations prints the SQL of all pending migrations without applying them.
func PreviewMigrations() error {
	conn, ctx, err := getConn()
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	if err := ensureMigrationsTable(conn, ctx); err != nil {
		return fmt.Errorf("ensure migrations table: %v", err)
	}

	applied, err := getAppliedMigrations(conn, ctx)
	if err != nil {
		return err
	}

	files, err := getMigrationFiles()
	if err != nil {
		return err
	}

	var pending []string
	for _, f := range files {
		if !applied[f] {
			pending = append(pending, f)
		}
	}

	if len(pending) == 0 {
		fmt.Println("✅ No pending migrations.")
		return nil
	}

	fmt.Println("\n================ DRY RUN: Migration Preview ================")
	for _, f := range pending {
		fmt.Printf("\n-- Migration: %s --\n", f)
		upSQL, downSQL, err := parseMigrationFile(f)
		if err != nil {
			return fmt.Errorf("parse migration file %s: %v", f, err)
		}
		fmt.Println("-- Up Migration SQL --")
		fmt.Println(upSQL)
		fmt.Println("\n-- Down Migration (Rollback) SQL --")
		fmt.Println(downSQL)
	}
	fmt.Println("============================================================")
	fmt.Println("(Dry run only. No migrations were applied.)")
	return nil
}

