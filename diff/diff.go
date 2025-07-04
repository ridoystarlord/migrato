package diff

import (
	"github.com/ridoystarlord/migrato/introspect"
	"github.com/ridoystarlord/migrato/schema"
)

type OperationType string

const (
	CreateTable OperationType = "CREATE_TABLE"
	AddColumn   OperationType = "ADD_COLUMN"
)

type Operation struct {
	Type      OperationType
	TableName string
	Columns   []schema.Column // for CREATE_TABLE
	Column    *schema.Column  // for ADD_COLUMN
}


func DiffSchemas(models []schema.Model, existing []introspect.ExistingTable) []Operation {
	var ops []Operation

	existingTableMap := map[string]introspect.ExistingTable{}
	for _, t := range existing {
		existingTableMap[t.TableName] = t
	}

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

		// Table exists: check for missing columns
		existingCols := map[string]bool{}
		for _, c := range table.Columns {
			existingCols[c.ColumnName] = true
		}

		for _, col := range model.Columns {
			if !existingCols[col.Name] {
				ops = append(ops, Operation{
					Type:      AddColumn,
					TableName: model.TableName,
					Column:    &col,
				})
			}
		}
	}

	return ops
}
