package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/ridoystarlord/migrato/introspect"
	"github.com/ridoystarlord/migrato/loader"
	"github.com/ridoystarlord/migrato/utils"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for potential issues",
	Long: `Check for potential issues in your schema and database state.

Examples:
  migrato check                    # Check for issues in schema and database
  migrato check --fix-suggestions  # Show suggestions for fixing issues
`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := checkForIssues(); err != nil {
			fmt.Printf("❌ Check failed: %v\n", err)
			os.Exit(1)
		}
	},
}

var showFixSuggestions bool

func init() {
	checkCmd.Flags().BoolVarP(&showFixSuggestions, "fix-suggestions", "f", false, "Show suggestions for fixing issues")
}

func checkForIssues() error {
	// Load schema
	models, err := loader.LoadModelsFromYAML(schemaFile)
	if err != nil {
		return fmt.Errorf("failed to load schema: %v", err)
	}

	// Connect to database
	utils.LoadEnv()
	dsn := utils.GetDatabaseURL()
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL not set")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	// Get current database state
	dbTables, err := introspect.IntrospectDatabase()
	if err != nil {
		return fmt.Errorf("failed to introspect database: %v", err)
	}

	// Convert to map for easier lookup
	dbTableMap := make(map[string]introspect.ExistingTable)
	for _, table := range dbTables {
		dbTableMap[table.TableName] = table
	}

	issues := []string{}
	warnings := []string{}

	// Check 1: Orphaned tables (tables in DB but not in schema)
	for tableName := range dbTableMap {
		found := false
		for _, model := range models {
			if model.TableName == tableName {
				found = true
				break
			}
		}
		if !found && tableName != "schema_migrations" {
			issues = append(issues, fmt.Sprintf("Orphaned table: '%s' exists in database but not in schema", tableName))
			if showFixSuggestions {
				fmt.Printf("💡 Suggestion: Add table '%s' to schema.yaml or drop it from database\n", tableName)
			}
		}
	}

	// Check 2: Missing tables (tables in schema but not in DB)
	for _, model := range models {
		if _, exists := dbTableMap[model.TableName]; !exists {
			warnings = append(warnings, fmt.Sprintf("Missing table: '%s' defined in schema but not in database", model.TableName))
			if showFixSuggestions {
				fmt.Printf("💡 Suggestion: Run 'migrato generate' and 'migrato migrate' to create table '%s'\n", model.TableName)
			}
		}
	}

	// Check 3: Column mismatches
	for _, model := range models {
		if dbTable, exists := dbTableMap[model.TableName]; exists {
			// Convert columns to map for easier lookup
			dbColumnMap := make(map[string]introspect.ExistingColumn)
			for _, col := range dbTable.Columns {
				dbColumnMap[col.ColumnName] = col
			}

			// Check for missing columns
			for _, col := range model.Columns {
				if _, colExists := dbColumnMap[col.Name]; !colExists {
					warnings = append(warnings, fmt.Sprintf("Missing column: '%s.%s' defined in schema but not in database", model.TableName, col.Name))
					if showFixSuggestions {
						fmt.Printf("💡 Suggestion: Run 'migrato generate' and 'migrato migrate' to add column '%s' to table '%s'\n", col.Name, model.TableName)
					}
				}
			}

			// Check for orphaned columns
			for colName := range dbColumnMap {
				found := false
				for _, col := range model.Columns {
					if col.Name == colName {
						found = true
						break
					}
				}
				if !found {
					issues = append(issues, fmt.Sprintf("Orphaned column: '%s.%s' exists in database but not in schema", model.TableName, colName))
					if showFixSuggestions {
						fmt.Printf("💡 Suggestion: Add column '%s' to table '%s' in schema.yaml or drop it from database\n", colName, model.TableName)
					}
				}
			}
		}
	}

	// Check 4: Index issues
	for _, model := range models {
		if dbTable, exists := dbTableMap[model.TableName]; exists {
			// Check for missing indexes
			for _, idx := range model.Indexes {
				found := false
				for _, dbIdx := range dbTable.Indexes {
					if dbIdx.IndexName == idx.Name {
						found = true
						break
					}
				}
				if !found {
					warnings = append(warnings, fmt.Sprintf("Missing index: '%s' on table '%s' defined in schema but not in database", idx.Name, model.TableName))
					if showFixSuggestions {
						fmt.Printf("💡 Suggestion: Run 'migrato generate' and 'migrato migrate' to create index '%s'\n", idx.Name)
					}
				}
			}
		}
	}

	// Check 5: Foreign key issues
	for _, model := range models {
		for _, col := range model.Columns {
			if col.ForeignKey != nil {
				if dbTable, exists := dbTableMap[model.TableName]; exists {
					// Check if foreign key constraint exists
					found := false
					for _, fk := range dbTable.ForeignKeys {
						if fk.ColumnName == col.Name && fk.ReferencesTable == col.ForeignKey.ReferencesTable {
							found = true
							break
						}
					}
					if !found {
						warnings = append(warnings, fmt.Sprintf("Missing foreign key: '%s.%s' -> '%s.%s' defined in schema but not in database", 
							model.TableName, col.Name, col.ForeignKey.ReferencesTable, col.ForeignKey.ReferencesColumn))
						if showFixSuggestions {
							fmt.Printf("💡 Suggestion: Run 'migrato generate' and 'migrato migrate' to add foreign key constraint\n")
						}
					}
				}
			}
		}
	}

	// Check 6: Migration status
	pendingMigrations, err := getPendingMigrations(conn, ctx)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Could not check migration status: %v", err))
	} else if len(pendingMigrations) > 0 {
		warnings = append(warnings, fmt.Sprintf("Found %d pending migrations", len(pendingMigrations)))
		if showFixSuggestions {
			fmt.Printf("💡 Suggestion: Run 'migrato migrate' to apply pending migrations\n")
		}
	}

	// Report results
	if len(issues) == 0 && len(warnings) == 0 {
		fmt.Println("✅ No issues found! Your schema and database are in sync.")
		return nil
	}

	if len(issues) > 0 {
		fmt.Printf("\n❌ Found %d issues:\n", len(issues))
		for _, issue := range issues {
			fmt.Printf("  • %s\n", issue)
		}
	}

	if len(warnings) > 0 {
		fmt.Printf("\n⚠️  Found %d warnings:\n", len(warnings))
		for _, warning := range warnings {
			fmt.Printf("  • %s\n", warning)
		}
	}

	if showFixSuggestions {
		fmt.Printf("\n💡 Use 'migrato check --fix-suggestions' to see detailed suggestions\n")
	}

	return nil
}

func getPendingMigrations(conn *pgx.Conn, ctx context.Context) ([]string, error) {
	// Get applied migrations
	rows, err := conn.Query(ctx, "SELECT filename FROM schema_migrations ORDER BY applied_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return nil, err
		}
		applied[filename] = true
	}

	// Get all migration files
	files, err := os.ReadDir("migrations")
	if err != nil {
		return nil, err
	}

	var pending []string
	for _, file := range files {
		if !file.IsDir() && len(file.Name()) > 4 && file.Name()[len(file.Name())-4:] == ".sql" {
			if !applied[file.Name()] {
				pending = append(pending, file.Name())
			}
		}
	}

	return pending, nil
} 