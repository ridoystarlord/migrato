package runner

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/ridoystarlord/migrato/utils"
)

func getConn() (*pgx.Conn, context.Context, error) {
	utils.LoadEnv()
	connStr := utils.GetDatabaseURL()
	if connStr == "" {
		return nil, nil, fmt.Errorf("DATABASE_URL not set")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return nil, nil, fmt.Errorf("connect: %v", err)
	}
	return conn, ctx, nil
}

func ensureMigrationsTable(conn *pgx.Conn, ctx context.Context) error {
	_, err := conn.Exec(ctx, `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		id SERIAL PRIMARY KEY,
		filename TEXT NOT NULL UNIQUE,
		applied_at TIMESTAMP DEFAULT now()
	);
	`)
	return err
}

func getAppliedMigrations(conn *pgx.Conn, ctx context.Context) (map[string]bool, error) {
	rows, err := conn.Query(ctx, `SELECT filename FROM schema_migrations;`)
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

func applyMigration(conn *pgx.Conn, ctx context.Context, filename string) error {
	content, err := os.ReadFile(filepath.Join("migrations", filename))
	if err != nil {
		return fmt.Errorf("read file %s: %v", filename, err)
	}

	_, err = conn.Exec(ctx, string(content))
	if err != nil {
		return fmt.Errorf("executing migration %s: %v", filename, err)
	}

	_, err = conn.Exec(ctx, `INSERT INTO schema_migrations (filename) VALUES ($1);`, filename)
	if err != nil {
		return fmt.Errorf("recording migration %s: %v", filename, err)
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

func Status() ([]string, []string, error) {
	conn, ctx, err := getConn()
	if err != nil {
		return nil, nil, err
	}
	defer conn.Close(ctx)

	if err := ensureMigrationsTable(conn, ctx); err != nil {
		return nil, nil, err
	}

	appliedMap, err := getAppliedMigrations(conn, ctx)
	if err != nil {
		return nil, nil, err
	}

	var applied []string
	for k := range appliedMap {
		applied = append(applied, k)
	}

	files, err := getMigrationFiles()
	if err != nil {
		return nil, nil, err
	}

	var pending []string
	for _, f := range files {
		if !appliedMap[f] {
			pending = append(pending, f)
		}
	}

	return applied, pending, nil
}

