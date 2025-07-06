package diff

import (
	"strings"

	"github.com/ridoystarlord/migrato/introspect"
	"github.com/ridoystarlord/migrato/schema"
)

type OperationType string

const (
	CreateTable    OperationType = "CREATE_TABLE"
	AddColumn      OperationType = "ADD_COLUMN"
	DropColumn     OperationType = "DROP_COLUMN"
	ModifyColumn   OperationType = "MODIFY_COLUMN"
	RenameColumn   OperationType = "RENAME_COLUMN"
	DropTable      OperationType = "DROP_TABLE"
	AddForeignKey  OperationType = "ADD_FOREIGN_KEY"
	DropForeignKey OperationType = "DROP_FOREIGN_KEY"
	CreateIndex    OperationType = "CREATE_INDEX"
	DropIndex      OperationType = "DROP_INDEX"
)

type Operation struct {
	Type         OperationType
	TableName    string
	Columns      []schema.Column // for CREATE_TABLE
	Column       *schema.Column  // for ADD_COLUMN, MODIFY_COLUMN
	ColumnName   string          // for DROP_COLUMN, ADD_FOREIGN_KEY
	NewColumnName string         // for RENAME_COLUMN
	ForeignKey   *schema.ForeignKey // for ADD_FOREIGN_KEY
	FKName       string          // for DROP_FOREIGN_KEY
	Index        *schema.Index   // for CREATE_INDEX
	IndexName    string          // for DROP_INDEX
	// For MODIFY_COLUMN operations
	OldColumn    *introspect.ExistingColumn // original column definition
}

func DiffSchemas(models []schema.Model, existing []introspect.ExistingTable) []Operation {
	var ops []Operation

	// Create maps for easier lookup
	existingTableMap := map[string]introspect.ExistingTable{}
	modelTableMap := map[string]schema.Model{}
	
	for _, t := range existing {
		// Skip system tables
		if t.TableName == "schema_migrations" || t.TableName == "migration_logs" {
			continue
		}
		existingTableMap[t.TableName] = t
	}
	
	for _, m := range models {
		modelTableMap[m.TableName] = m
	}

	// Check for tables to create or modify
	for _, model := range models {
		table, exists := existingTableMap[model.TableName]
		if !exists {
			// Table doesn't exist: CREATE TABLE
			ops = append(ops, Operation{
				Type:      CreateTable,
				TableName: model.TableName,
				Columns:   model.Columns,
			})
			continue
		}

		// Table exists: check for missing columns and safe modifications
		existingCols := map[string]introspect.ExistingColumn{}
		modelCols := map[string]schema.Column{}
		
		for _, c := range table.Columns {
			existingCols[c.ColumnName] = c
		}
		
		for _, c := range model.Columns {
			modelCols[c.Name] = c
		}

		// Check for columns to add (in model but not in existing) - SAFE
		for _, col := range model.Columns {
			if _, exists := existingCols[col.Name]; !exists {
				ops = append(ops, Operation{
					Type:      AddColumn,
					TableName: model.TableName,
					Column:    &col,
				})
			}
		}

		// Check for columns to drop (in existing but not in model) - DESTRUCTIVE
		for _, col := range table.Columns {
			if _, exists := modelCols[col.ColumnName]; !exists {
				ops = append(ops, Operation{
					Type:       DropColumn,
					TableName:  model.TableName,
					ColumnName: col.ColumnName,
				})
			}
		}

		// Check for column modifications - be very conservative
		for _, modelCol := range model.Columns {
			if existingCol, exists := existingCols[modelCol.Name]; exists {
				// Only modify if it's a significant change
				if needsSignificantColumnModification(existingCol, modelCol) {
					ops = append(ops, Operation{
						Type:       ModifyColumn,
						TableName:  model.TableName,
						Column:     &modelCol,
						OldColumn:  &existingCol,
					})
				}
			}
		}

		// Check for foreign keys to add - SAFE
		existingFKs := map[string]introspect.ExistingForeignKey{}
		for _, fk := range table.ForeignKeys {
			existingFKs[fk.ColumnName] = fk
		}

		for _, col := range model.Columns {
			if col.ForeignKey != nil {
				if _, exists := existingFKs[col.Name]; !exists {
					// Foreign key doesn't exist: ADD FOREIGN KEY
					ops = append(ops, Operation{
						Type:        AddForeignKey,
						TableName:   model.TableName,
						ColumnName:  col.Name,
						ForeignKey:  col.ForeignKey,
					})
				}
			}
		}

		// Check for indexes to add - SAFE, but be more conservative
		existingIndexes := map[string]introspect.ExistingIndex{}
		for _, idx := range table.Indexes {
			existingIndexes[idx.IndexName] = idx
		}

		// Check table-level indexes - only add if explicitly defined and missing
		for _, idx := range model.Indexes {
			if idx.Name != "" { // Only check named indexes
				if _, exists := existingIndexes[idx.Name]; !exists {
					ops = append(ops, Operation{
						Type:      CreateIndex,
						TableName: model.TableName,
						Index:     &idx,
					})
				}
			}
		}

		// Check column-level indexes - only add if explicitly defined and missing
		for _, col := range model.Columns {
			if col.Index != nil && col.Index.Name != "" { // Only check named indexes
				if _, exists := existingIndexes[col.Index.Name]; !exists {
					// Create index operation
					index := schema.Index{
						Name:    col.Index.Name,
						Table:   model.TableName,
						Columns: col.Index.Columns,
						Unique:  col.Index.Unique,
						Type:    col.Index.Type,
					}
					if len(index.Columns) == 0 {
						index.Columns = []string{col.Name}
					}
					
					ops = append(ops, Operation{
						Type:      CreateIndex,
						TableName: model.TableName,
						Index:     &index,
					})
				}
			}
		}
	}

	// Check for tables to drop (in existing but not in model) - DESTRUCTIVE
	for _, table := range existing {
		// Skip system tables
		if table.TableName == "schema_migrations" || table.TableName == "migration_logs" {
			continue
		}
		if _, exists := modelTableMap[table.TableName]; !exists {
			ops = append(ops, Operation{
				Type:      DropTable,
				TableName: table.TableName,
			})
		}
	}

	return ops
}

// needsSignificantColumnModification checks if a column needs significant modification
// Only triggers for actual schema changes, not system differences
func needsSignificantColumnModification(existing introspect.ExistingColumn, model schema.Column) bool {
	// Skip primary key columns - they're handled by the database
	if existing.IsPrimaryKey {
		return false
	}

	// Check if data type changed - be very lenient with type comparisons
	if !isCompatibleType(existing.DataType, model.Type) {
		// Only consider it a change if it's a significant type change
		return isSignificantTypeChange(existing.DataType, model.Type)
	}

	// Check if nullable constraint changed - be more precise
	modelNullable := !model.NotNull // model.NotNull = true means NOT NULL
	if existing.IsNullable != modelNullable {
		return true
	}

	// Check if default value changed - be more lenient with default comparisons
	existingDefault := existing.ColumnDefault
	modelDefault := model.Default
	
	// Handle different default value representations
	if existingDefault == nil && modelDefault != nil {
		return true
	}
	if existingDefault != nil && modelDefault == nil {
		return true
	}
	if existingDefault != nil && modelDefault != nil {
		// Normalize default values for comparison
		existingStr := normalizeDefaultValue(*existingDefault)
		modelStr := normalizeDefaultValue(*modelDefault)
		if existingStr != modelStr {
			return true
		}
	}

	return false
}

// isSignificantTypeChange checks if a type change is significant enough to warrant a migration
func isSignificantTypeChange(oldType, newType string) bool {
	oldType = strings.ToLower(strings.TrimSpace(oldType))
	newType = strings.ToLower(strings.TrimSpace(newType))
	
	// Extract base types (remove size specifications)
	oldBase := extractBaseType(oldType)
	newBase := extractBaseType(newType)
	
	// If base types are the same, it's not significant
	if oldBase == newBase {
		return false
	}
	
	// Significant type changes that require migration
	significantChanges := map[string][]string{
		"text": {"integer", "bigint", "smallint", "numeric", "decimal", "boolean", "date", "timestamp"},
		"varchar": {"integer", "bigint", "smallint", "numeric", "decimal", "boolean", "date", "timestamp"},
		"integer": {"text", "varchar", "boolean", "date", "timestamp"},
		"bigint": {"text", "varchar", "boolean", "date", "timestamp"},
		"smallint": {"text", "varchar", "boolean", "date", "timestamp"},
		"numeric": {"text", "varchar", "boolean", "date", "timestamp"},
		"decimal": {"text", "varchar", "boolean", "date", "timestamp"},
		"boolean": {"text", "varchar", "integer", "bigint", "smallint", "numeric", "decimal", "date", "timestamp"},
		"date": {"text", "varchar", "integer", "bigint", "smallint", "numeric", "decimal", "boolean", "timestamp"},
		"timestamp": {"text", "varchar", "integer", "bigint", "smallint", "numeric", "decimal", "boolean", "date"},
	}
	
	if incompatible, exists := significantChanges[oldBase]; exists {
		for _, incompatibleType := range incompatible {
			if newBase == incompatibleType {
				return true
			}
		}
	}
	
	return false
}

// isCompatibleType checks if two types are compatible (allows for minor differences)
func isCompatibleType(existingType, modelType string) bool {
	existingType = strings.ToLower(strings.TrimSpace(existingType))
	modelType = strings.ToLower(strings.TrimSpace(modelType))
	
	// Extract base types (remove size specifications)
	existingBase := extractBaseType(existingType)
	modelBase := extractBaseType(modelType)
	
	// If base types are the same, they're compatible
	if existingBase == modelBase {
		return true
	}
	
	// Allow some common type variations
	compatibleTypes := map[string][]string{
		"varchar": {"text", "character varying"},
		"text": {"varchar", "character varying"},
		"character varying": {"varchar", "text"},
		"int": {"integer", "int4"},
		"integer": {"int", "int4"},
		"int4": {"int", "integer"},
		"bigint": {"int8"},
		"int8": {"bigint"},
		"smallint": {"int2"},
		"int2": {"smallint"},
		"numeric": {"decimal"},
		"decimal": {"numeric"},
		"timestamp": {"timestamp without time zone"},
		"timestamp without time zone": {"timestamp"},
		"timestamptz": {"timestamp with time zone"},
		"timestamp with time zone": {"timestamptz"},
	}
	
	if allowed, exists := compatibleTypes[existingBase]; exists {
		for _, compatible := range allowed {
			if modelBase == compatible {
				return true
			}
		}
	}
	
	return false
}

// extractBaseType extracts the base type from a PostgreSQL type
func extractBaseType(typeStr string) string {
	// Remove size specifications like (255), (10,2), etc.
	if idx := strings.Index(typeStr, "("); idx != -1 {
		return typeStr[:idx]
	}
	return typeStr
}

// normalizeDefaultValue normalizes default values for comparison
func normalizeDefaultValue(defaultVal string) string {
	// Remove quotes and normalize function calls
	defaultVal = strings.TrimSpace(defaultVal)
	
	// Handle quoted strings
	if (strings.HasPrefix(defaultVal, "'") && strings.HasSuffix(defaultVal, "'")) ||
		(strings.HasPrefix(defaultVal, "\"") && strings.HasSuffix(defaultVal, "\"")) {
		return strings.Trim(defaultVal, "'\"")
	}
	
	// Normalize function calls
	defaultVal = strings.ToLower(defaultVal)
	
	return defaultVal
}
