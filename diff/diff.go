package diff

import (
	"fmt"
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

		// Table exists: check for missing columns and extra columns
		existingCols := map[string]introspect.ExistingColumn{}
		modelCols := map[string]schema.Column{}
		
		for _, c := range table.Columns {
			existingCols[c.ColumnName] = c
		}
		
		for _, c := range model.Columns {
			modelCols[c.Name] = c
		}

		// Check for columns to add (in model but not in existing)
		for _, col := range model.Columns {
			if _, exists := existingCols[col.Name]; !exists {
				ops = append(ops, Operation{
					Type:      AddColumn,
					TableName: model.TableName,
					Column:    &col,
				})
			}
		}

		// Check for columns to drop (in existing but not in model)
		for _, col := range table.Columns {
			if _, exists := modelCols[col.ColumnName]; !exists {
				ops = append(ops, Operation{
					Type:       DropColumn,
					TableName:  model.TableName,
					ColumnName: col.ColumnName,
				})
			}
		}

		// Check for column modifications (existing columns that have changed)
		for _, modelCol := range model.Columns {
			if existingCol, exists := existingCols[modelCol.Name]; exists {
				// Check if column needs modification
				if needsColumnModification(existingCol, modelCol) {
					ops = append(ops, Operation{
						Type:       ModifyColumn,
						TableName:  model.TableName,
						Column:     &modelCol,
						OldColumn:  &existingCol,
					})
				}
			}
		}

		// Check for foreign keys to add
		existingFKs := map[string]introspect.ExistingForeignKey{}
		for _, fk := range table.ForeignKeys {
			existingFKs[fk.ColumnName] = fk
		}

		for _, col := range model.Columns {
			if col.ForeignKey != nil {
				existingFK, exists := existingFKs[col.Name]
				if !exists {
					// Foreign key doesn't exist: ADD FOREIGN KEY
					ops = append(ops, Operation{
						Type:        AddForeignKey,
						TableName:   model.TableName,
						ColumnName:  col.Name,
						ForeignKey:  col.ForeignKey,
					})
				} else {
					// Check if foreign key definition changed
					if existingFK.ReferencesTable != col.ForeignKey.ReferencesTable ||
						existingFK.ReferencesColumn != col.ForeignKey.ReferencesColumn ||
						existingFK.OnDelete != col.ForeignKey.OnDelete ||
						existingFK.OnUpdate != col.ForeignKey.OnUpdate {
						
						// Drop existing foreign key and add new one
						ops = append(ops, Operation{
							Type:    DropForeignKey,
							TableName: model.TableName,
							FKName:  existingFK.ConstraintName,
						})
						ops = append(ops, Operation{
							Type:        AddForeignKey,
							TableName:   model.TableName,
							ColumnName:  col.Name,
							ForeignKey:  col.ForeignKey,
						})
					}
				}
			}
		}

		// Check for foreign keys to drop (in existing but not in model)
		for _, fk := range table.ForeignKeys {
			col, exists := modelCols[fk.ColumnName]
			if !exists || col.ForeignKey == nil {
				ops = append(ops, Operation{
					Type:    DropForeignKey,
					TableName: model.TableName,
					FKName:  fk.ConstraintName,
				})
			}
		}

		// Check for indexes to add
		existingIndexes := map[string]introspect.ExistingIndex{}
		for _, idx := range table.Indexes {
			existingIndexes[idx.IndexName] = idx
		}

		// Check table-level indexes
		for _, idx := range model.Indexes {
			if _, exists := existingIndexes[idx.Name]; !exists {
				ops = append(ops, Operation{
					Type:      CreateIndex,
					TableName: model.TableName,
					Index:     &idx,
				})
			}
		}

		// Check column-level indexes
		for _, col := range model.Columns {
			if col.Index != nil {
				indexName := col.Index.Name
				if indexName == "" {
					// Generate index name if not provided
					if len(col.Index.Columns) > 0 {
						indexName = fmt.Sprintf("idx_%s_%s", model.TableName, strings.Join(col.Index.Columns, "_"))
					} else {
						indexName = fmt.Sprintf("idx_%s_%s", model.TableName, col.Name)
					}
				}
				
				if _, exists := existingIndexes[indexName]; !exists {
					// Create index operation
					index := schema.Index{
						Name:    indexName,
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

		// Check for indexes to drop (in existing but not in model)
		for _, idx := range table.Indexes {
			found := false
			// Check table-level indexes
			for _, modelIdx := range model.Indexes {
				if modelIdx.Name == idx.IndexName {
					found = true
					break
				}
			}
			// Check column-level indexes
			if !found {
				for _, col := range model.Columns {
					if col.Index != nil {
						indexName := col.Index.Name
						if indexName == "" {
							if len(col.Index.Columns) > 0 {
								indexName = fmt.Sprintf("idx_%s_%s", model.TableName, strings.Join(col.Index.Columns, "_"))
							} else {
								indexName = fmt.Sprintf("idx_%s_%s", model.TableName, col.Name)
							}
						}
						if indexName == idx.IndexName {
							found = true
							break
						}
					}
				}
			}
			
			if !found {
				ops = append(ops, Operation{
					Type:      DropIndex,
					TableName: model.TableName,
					IndexName: idx.IndexName,
				})
			}
		}
	}

	// Check for tables to drop (in existing but not in model)
	for _, table := range existing {
		if _, exists := modelTableMap[table.TableName]; !exists {
			ops = append(ops, Operation{
				Type:      DropTable,
				TableName: table.TableName,
			})
		}
	}

	return ops
}

// needsColumnModification checks if a column needs to be modified
func needsColumnModification(existing introspect.ExistingColumn, model schema.Column) bool {
	// Check if data type changed
	if !strings.EqualFold(existing.DataType, model.Type) {
		return true
	}

	// Check if nullable constraint changed
	// Note: We need to infer the nullable constraint from the schema
	// For now, we'll assume all columns are nullable unless explicitly marked
	modelNullable := true // Default to nullable
	if model.NotNull {
		modelNullable = false
	}
	if existing.IsNullable != modelNullable {
		return true
	}

	// Check if default value changed
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
