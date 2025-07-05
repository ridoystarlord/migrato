package cmd

import (
	"fmt"
	"os"

	"github.com/ridoystarlord/migrato/loader"
	"github.com/ridoystarlord/migrato/schema"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate schema integrity",
	Long: `Validate your YAML schema for integrity and consistency.

Examples:
  migrato validate                    # Validate default schema.yaml
  migrato validate -f custom.yaml     # Validate custom schema file
`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := validateSchema(); err != nil {
			fmt.Printf("❌ Schema validation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Schema is valid and consistent")
	},
}

func validateSchema() error {
	// Load and parse schema
	models, err := loader.LoadModelsFromYAML(schemaFile)
	if err != nil {
		return fmt.Errorf("failed to load schema: %v", err)
	}

	// Validate each model
	for _, model := range models {
		if err := validateModel(model, models); err != nil {
			return fmt.Errorf("model '%s': %v", model.TableName, err)
		}
	}

	fmt.Printf("📋 Validated %d models\n", len(models))
	return nil
}

func validateModel(model schema.Model, allModels []schema.Model) error {
	// Check for duplicate table names
	tableNames := make(map[string]bool)
	for _, m := range allModels {
		if tableNames[m.TableName] {
			return fmt.Errorf("duplicate table name: %s", m.TableName)
		}
		tableNames[m.TableName] = true
	}

	// Validate columns
	columnNames := make(map[string]bool)
	primaryKeyCount := 0

	for _, col := range model.Columns {
		// Check for duplicate column names
		if columnNames[col.Name] {
			return fmt.Errorf("duplicate column name: %s", col.Name)
		}
		columnNames[col.Name] = true

		// Count primary keys
		if col.Primary {
			primaryKeyCount++
		}

		// Validate foreign key references
		if col.ForeignKey != nil {
			if err := validateForeignKey(col.ForeignKey, allModels); err != nil {
				return fmt.Errorf("foreign key on column '%s': %v", col.Name, err)
			}
		}

		// Validate column type
		if err := validateColumnType(col.Type); err != nil {
			return fmt.Errorf("column '%s': %v", col.Name, err)
		}
	}

	// Check primary key constraints
	if primaryKeyCount == 0 {
		return fmt.Errorf("no primary key defined")
	}
	if primaryKeyCount > 1 {
		return fmt.Errorf("multiple primary keys defined (%d)", primaryKeyCount)
	}

	// Validate relations
	for _, rel := range model.Relations {
		if err := validateRelation(rel, model, allModels); err != nil {
			return fmt.Errorf("relation '%s': %v", rel.Name, err)
		}
	}

	// Validate indexes
	for _, idx := range model.Indexes {
		if err := validateIndex(idx, model); err != nil {
			return fmt.Errorf("index '%s': %v", idx.Name, err)
		}
	}

	return nil
}

func validateForeignKey(fk *schema.ForeignKey, allModels []schema.Model) error {
	// Check if referenced table exists
	var targetModel *schema.Model
	for _, model := range allModels {
		if model.TableName == fk.ReferencesTable {
			targetModel = &model
			break
		}
	}

	if targetModel == nil {
		return fmt.Errorf("referenced table '%s' does not exist", fk.ReferencesTable)
	}

	// Check if referenced column exists
	var targetColumn *schema.Column
	for _, col := range targetModel.Columns {
		if col.Name == fk.ReferencesColumn {
			targetColumn = &col
			break
		}
	}

	if targetColumn == nil {
		return fmt.Errorf("referenced column '%s' does not exist in table '%s'", fk.ReferencesColumn, fk.ReferencesTable)
	}

	// Validate cascade options
	if fk.OnDelete != "" {
		if err := validateCascadeOption(fk.OnDelete); err != nil {
			return fmt.Errorf("on_delete: %v", err)
		}
	}

	if fk.OnUpdate != "" {
		if err := validateCascadeOption(fk.OnUpdate); err != nil {
			return fmt.Errorf("on_update: %v", err)
		}
	}

	return nil
}

func validateRelation(rel schema.Relation, model schema.Model, allModels []schema.Model) error {
	// Check if from column exists
	var fromColumn *schema.Column
	for _, col := range model.Columns {
		if col.Name == rel.FromColumn {
			fromColumn = &col
			break
		}
	}

	if fromColumn == nil {
		return fmt.Errorf("from column '%s' does not exist", rel.FromColumn)
	}

	// Check if target table exists
	var targetModel *schema.Model
	for _, m := range allModels {
		if m.TableName == rel.ToTable {
			targetModel = &m
			break
		}
	}

	if targetModel == nil {
		return fmt.Errorf("target table '%s' does not exist", rel.ToTable)
	}

	// Validate relation type
	switch rel.Type {
	case schema.OneToOne, schema.OneToMany, schema.ManyToMany:
		// Valid types
	default:
		return fmt.Errorf("invalid relation type: %s", rel.Type)
	}

	return nil
}

func validateIndex(idx schema.Index, model schema.Model) error {
	// Check for duplicate index names
	indexNames := make(map[string]bool)
	for _, existingIdx := range model.Indexes {
		if indexNames[existingIdx.Name] {
			return fmt.Errorf("duplicate index name: %s", existingIdx.Name)
		}
		indexNames[existingIdx.Name] = true
	}

	// Validate that all indexed columns exist
	for _, colName := range idx.Columns {
		var found bool
		for _, col := range model.Columns {
			if col.Name == colName {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("indexed column '%s' does not exist", colName)
		}
	}

	// Validate index type if specified
	if idx.Type != "" {
		if err := validateIndexType(idx.Type); err != nil {
			return err
		}
	}

	return nil
}

func validateColumnType(colType string) error {
	validTypes := []string{
		"serial", "bigserial", "integer", "int", "int4", "bigint", "int8",
		"smallint", "int2", "text", "varchar", "character varying",
		"boolean", "bool", "timestamp", "timestamptz", "date",
		"numeric", "decimal", "real", "float4", "double precision", "float8",
		"uuid", "json", "jsonb",
	}

	for _, validType := range validTypes {
		if colType == validType {
			return nil
		}
	}

	return fmt.Errorf("unsupported column type: %s", colType)
}

func validateCascadeOption(option string) error {
	validOptions := []string{"CASCADE", "SET NULL", "RESTRICT", "NO ACTION"}
	
	for _, validOption := range validOptions {
		if option == validOption {
			return nil
		}
	}

	return fmt.Errorf("invalid cascade option: %s (valid options: %v)", option, validOptions)
}

func validateIndexType(indexType string) error {
	validTypes := []string{"btree", "hash", "gin", "gist", "spgist", "brin"}
	
	for _, validType := range validTypes {
		if indexType == validType {
			return nil
		}
	}

	return fmt.Errorf("invalid index type: %s (valid types: %v)", indexType, validTypes)
} 