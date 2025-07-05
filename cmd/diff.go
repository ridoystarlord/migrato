package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ridoystarlord/migrato/diff"
	"github.com/ridoystarlord/migrato/introspect"
	"github.com/ridoystarlord/migrato/loader"
	"github.com/ridoystarlord/migrato/schema"
)

var (
	diffVisual bool
	diffFile   string
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show differences between schema and database",
	Long: `Show differences between your schema.yaml and the current database.

Examples:
  migrato diff                    # Show differences in text format
  migrato diff --visual          # Show differences in tree format with colors
  migrato diff -f custom.yaml    # Use custom schema file
`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load schema
		schemaFile := diffFile
		if schemaFile == "" {
			schemaFile = "schema.yaml"
		}

		models, err := loader.LoadModelsFromYAML(schemaFile)
		if err != nil {
			fmt.Printf("❌ Error loading schema: %v\n", err)
			os.Exit(1)
		}

		// Introspect database
		existing, err := introspect.IntrospectDatabase()
		if err != nil {
			fmt.Printf("❌ Error introspecting database: %v\n", err)
			os.Exit(1)
		}

		// Generate diff
		operations := diff.DiffSchemas(models, existing)

		if len(operations) == 0 {
			fmt.Println("✅ No differences found between schema and database")
			return
		}

		if diffVisual {
			showVisualDiff(operations, models, existing)
		} else {
			showTextDiff(operations)
		}
	},
}

func showVisualDiff(operations []diff.Operation, models []schema.Model, existing []introspect.ExistingTable) {
	fmt.Println("🌳 Schema Changes (Visual Diff)")
	fmt.Println(strings.Repeat("=", 50))

	// Create maps for easier lookup
	existingTableMap := make(map[string]introspect.ExistingTable)
	modelTableMap := make(map[string]schema.Model)

	for _, t := range existing {
		existingTableMap[t.TableName] = t
	}
	for _, m := range models {
		modelTableMap[m.TableName] = m
	}

	// Show table-level changes
	showTableChanges(operations, modelTableMap, existingTableMap)

	// Show column-level changes
	showColumnChanges(operations, modelTableMap, existingTableMap)

	// Show index changes
	showIndexChanges(operations, modelTableMap, existingTableMap)

	// Show foreign key changes
	showForeignKeyChanges(operations, modelTableMap, existingTableMap)
}

func showTableChanges(operations []diff.Operation, modelTableMap map[string]schema.Model, existingTableMap map[string]introspect.ExistingTable) {
	green := color.New(color.FgGreen, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	yellow := color.New(color.FgYellow, color.Bold)

	fmt.Println("\n📋 Tables:")
	
	// Find table operations
	createTables := make(map[string]bool)
	dropTables := make(map[string]bool)

	for _, op := range operations {
		switch op.Type {
		case diff.CreateTable:
			createTables[op.TableName] = true
		case diff.DropTable:
			dropTables[op.TableName] = true
		}
	}

	// Show created tables
	for tableName := range createTables {
		green.Printf("  ➕ CREATE %s\n", tableName)
	}

	// Show dropped tables
	for tableName := range dropTables {
		red.Printf("  ❌ DROP %s\n", tableName)
	}

	// Show existing tables
	for tableName := range modelTableMap {
		if !createTables[tableName] && !dropTables[tableName] {
			if _, exists := existingTableMap[tableName]; exists {
				yellow.Printf("  ⚡ MODIFY %s\n", tableName)
			}
		}
	}
}

func showColumnChanges(operations []diff.Operation, modelTableMap map[string]schema.Model, existingTableMap map[string]introspect.ExistingTable) {
	green := color.New(color.FgGreen, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	blue := color.New(color.FgBlue, color.Bold)

	fmt.Println("\n📝 Columns:")

	// Group operations by table
	tableOps := make(map[string][]diff.Operation)
	for _, op := range operations {
		if op.Type == diff.AddColumn || op.Type == diff.DropColumn || op.Type == diff.ModifyColumn {
			tableOps[op.TableName] = append(tableOps[op.TableName], op)
		}
	}

	for tableName, ops := range tableOps {
		fmt.Printf("  📋 %s:\n", tableName)
		
		for _, op := range ops {
			switch op.Type {
			case diff.AddColumn:
				green.Printf("    ➕ ADD %s (%s)", op.Column.Name, op.Column.Type)
				if op.Column.NotNull {
					green.Print(" NOT NULL")
				}
				if op.Column.Default != nil {
					green.Printf(" DEFAULT %s", *op.Column.Default)
				}
				green.Println()
				
			case diff.DropColumn:
				red.Printf("    ❌ DROP %s\n", op.ColumnName)
				
			case diff.ModifyColumn:
				blue.Printf("    🔄 MODIFY %s:\n", op.Column.Name)
				showColumnModifications(op)
			}
		}
	}
}

func showColumnModifications(op diff.Operation) {
	blue := color.New(color.FgBlue)
	cyan := color.New(color.FgCyan)
	magenta := color.New(color.FgMagenta)

	if op.OldColumn == nil || op.Column == nil {
		return
	}

	// Type changes
	if !strings.EqualFold(op.OldColumn.DataType, op.Column.Type) {
		blue.Printf("      📊 TYPE: %s → %s\n", op.OldColumn.DataType, op.Column.Type)
	}

	// NOT NULL changes
	oldNullable := op.OldColumn.IsNullable
	newNullable := !op.Column.NotNull
	if oldNullable != newNullable {
		if newNullable {
			cyan.Printf("      🚫 NOT NULL: ADDED\n")
		} else {
			cyan.Printf("      ✅ NOT NULL: REMOVED\n")
		}
	}

	// Default value changes
	oldDefault := op.OldColumn.ColumnDefault
	newDefault := op.Column.Default
	if (oldDefault == nil && newDefault != nil) ||
		(oldDefault != nil && newDefault == nil) ||
		(oldDefault != nil && newDefault != nil && *oldDefault != *newDefault) {
		
		if oldDefault == nil {
			magenta.Printf("      🔧 DEFAULT: ADDED %s\n", *newDefault)
		} else if newDefault == nil {
			magenta.Printf("      🔧 DEFAULT: REMOVED (was %s)\n", *oldDefault)
		} else {
			magenta.Printf("      🔧 DEFAULT: %s → %s\n", *oldDefault, *newDefault)
		}
	}
}

func showIndexChanges(operations []diff.Operation, modelTableMap map[string]schema.Model, existingTableMap map[string]introspect.ExistingTable) {
	green := color.New(color.FgGreen, color.Bold)
	red := color.New(color.FgRed, color.Bold)

	fmt.Println("\n🔍 Indexes:")

	// Group operations by table
	tableOps := make(map[string][]diff.Operation)
	for _, op := range operations {
		if op.Type == diff.CreateIndex || op.Type == diff.DropIndex {
			tableOps[op.TableName] = append(tableOps[op.TableName], op)
		}
	}

	for tableName, ops := range tableOps {
		if len(ops) > 0 {
			fmt.Printf("  📋 %s:\n", tableName)
			
			for _, op := range ops {
				switch op.Type {
				case diff.CreateIndex:
					green.Printf("    ➕ CREATE INDEX %s\n", op.Index.Name)
					
				case diff.DropIndex:
					red.Printf("    ❌ DROP INDEX %s\n", op.IndexName)
				}
			}
		}
	}
}

func showForeignKeyChanges(operations []diff.Operation, modelTableMap map[string]schema.Model, existingTableMap map[string]introspect.ExistingTable) {
	green := color.New(color.FgGreen, color.Bold)
	red := color.New(color.FgRed, color.Bold)

	fmt.Println("\n🔗 Foreign Keys:")

	// Group operations by table
	tableOps := make(map[string][]diff.Operation)
	for _, op := range operations {
		if op.Type == diff.AddForeignKey || op.Type == diff.DropForeignKey {
			tableOps[op.TableName] = append(tableOps[op.TableName], op)
		}
	}

	for tableName, ops := range tableOps {
		if len(ops) > 0 {
			fmt.Printf("  📋 %s:\n", tableName)
			
			for _, op := range ops {
				switch op.Type {
				case diff.AddForeignKey:
					green.Printf("    ➕ ADD FK %s → %s.%s\n", 
						op.ColumnName, 
						op.ForeignKey.ReferencesTable, 
						op.ForeignKey.ReferencesColumn)
					
				case diff.DropForeignKey:
					red.Printf("    ❌ DROP FK %s\n", op.FKName)
				}
			}
		}
	}
}

func showTextDiff(operations []diff.Operation) {
	fmt.Println("📋 Schema Changes (Text Format)")
	fmt.Println(strings.Repeat("=", 40))

	for i, op := range operations {
		fmt.Printf("%d. ", i+1)
		
		switch op.Type {
		case diff.CreateTable:
			fmt.Printf("CREATE TABLE %s\n", op.TableName)
			
		case diff.DropTable:
			fmt.Printf("DROP TABLE %s\n", op.TableName)
			
		case diff.AddColumn:
			fmt.Printf("ADD COLUMN %s.%s (%s)", op.TableName, op.Column.Name, op.Column.Type)
			if op.Column.NotNull {
				fmt.Print(" NOT NULL")
			}
			if op.Column.Default != nil {
				fmt.Printf(" DEFAULT %s", *op.Column.Default)
			}
			fmt.Println()
			
		case diff.DropColumn:
			fmt.Printf("DROP COLUMN %s.%s\n", op.TableName, op.ColumnName)
			
		case diff.ModifyColumn:
			fmt.Printf("MODIFY COLUMN %s.%s\n", op.TableName, op.Column.Name)
			
		case diff.RenameColumn:
			fmt.Printf("RENAME COLUMN %s.%s TO %s\n", op.TableName, op.ColumnName, op.NewColumnName)
			
		case diff.CreateIndex:
			fmt.Printf("CREATE INDEX %s ON %s\n", op.Index.Name, op.TableName)
			
		case diff.DropIndex:
			fmt.Printf("DROP INDEX %s\n", op.IndexName)
			
		case diff.AddForeignKey:
			fmt.Printf("ADD FOREIGN KEY %s.%s → %s.%s\n", 
				op.TableName, op.ColumnName, 
				op.ForeignKey.ReferencesTable, op.ForeignKey.ReferencesColumn)
			
		case diff.DropForeignKey:
			fmt.Printf("DROP FOREIGN KEY %s\n", op.FKName)
		}
	}
}

func init() {
	diffCmd.Flags().BoolVarP(&diffVisual, "visual", "v", false, "Show changes in visual tree format")
	diffCmd.Flags().StringVarP(&diffFile, "file", "f", "", "Schema file to use (default: schema.yaml)")
} 