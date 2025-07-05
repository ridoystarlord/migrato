package generator

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ridoystarlord/migrato/diff"
)

// GenerateSQL converts a list of Operations into raw SQL statements.
func GenerateSQL(ops []diff.Operation) ([]string, error) {
	var sqlStatements []string

	for _, op := range ops {
		switch op.Type {
		case diff.CreateTable:
			stmt, err := generateCreateTable(op)
			if err != nil {
				return nil, fmt.Errorf("generate CREATE TABLE: %v", err)
			}
			sqlStatements = append(sqlStatements, stmt)

		case diff.AddColumn:
			stmt := fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN "%s" %s`,
				op.TableName,
				op.Column.Name,
				op.Column.Type,
			)
			if op.Column.NotNull {
				stmt += " NOT NULL"
			}
			if op.Column.Default != nil {
				stmt += fmt.Sprintf(" DEFAULT %s", *op.Column.Default)
			}
			if op.Column.Unique {
				stmt += " UNIQUE"
			}
			sqlStatements = append(sqlStatements, stmt+";")

		case diff.DropColumn:
			stmt := fmt.Sprintf(`ALTER TABLE "%s" DROP COLUMN "%s";`,
				op.TableName,
				op.ColumnName,
			)
			sqlStatements = append(sqlStatements, stmt)

		case diff.ModifyColumn:
			stmt, err := generateModifyColumn(op)
			if err != nil {
				return nil, fmt.Errorf("generate MODIFY COLUMN: %v", err)
			}
			sqlStatements = append(sqlStatements, stmt)

		case diff.RenameColumn:
			stmt := fmt.Sprintf(`ALTER TABLE "%s" RENAME COLUMN "%s" TO "%s";`,
				op.TableName,
				op.ColumnName,
				op.NewColumnName,
			)
			sqlStatements = append(sqlStatements, stmt)

		case diff.DropTable:
			stmt := fmt.Sprintf(`DROP TABLE IF EXISTS "%s";`,
				op.TableName,
			)
			sqlStatements = append(sqlStatements, stmt)

		case diff.AddForeignKey:
			stmt := fmt.Sprintf(`ALTER TABLE "%s" ADD CONSTRAINT "fk_%s_%s" FOREIGN KEY ("%s") REFERENCES "%s" ("%s")`,
				op.TableName,
				op.TableName,
				op.ForeignKey.ReferencesTable,
				op.ColumnName,
				op.ForeignKey.ReferencesTable,
				op.ForeignKey.ReferencesColumn,
			)
			if op.ForeignKey.OnDelete != "" {
				stmt += fmt.Sprintf(" ON DELETE %s", op.ForeignKey.OnDelete)
			}
			if op.ForeignKey.OnUpdate != "" {
				stmt += fmt.Sprintf(" ON UPDATE %s", op.ForeignKey.OnUpdate)
			}
			sqlStatements = append(sqlStatements, stmt+";")

		case diff.DropForeignKey:
			stmt := fmt.Sprintf(`ALTER TABLE "%s" DROP CONSTRAINT "%s";`,
				op.TableName,
				op.FKName,
			)
			sqlStatements = append(sqlStatements, stmt)

		case diff.CreateIndex:
			stmt, err := generateCreateIndex(op)
			if err != nil {
				return nil, fmt.Errorf("generate CREATE INDEX: %v", err)
			}
			sqlStatements = append(sqlStatements, stmt)

		case diff.DropIndex:
			stmt := fmt.Sprintf(`DROP INDEX IF EXISTS "%s";`,
				op.IndexName,
			)
			sqlStatements = append(sqlStatements, stmt)

		default:
			return nil, fmt.Errorf("unsupported operation: %s", op.Type)
		}
	}

	return sqlStatements, nil
}

// GenerateRollbackSQL converts a list of Operations into rollback SQL statements.
func GenerateRollbackSQL(ops []diff.Operation) ([]string, error) {
	var sqlStatements []string

	// Process operations in reverse order for rollback
	for i := len(ops) - 1; i >= 0; i-- {
		op := ops[i]
		switch op.Type {
		case diff.CreateTable:
			stmt := fmt.Sprintf(`DROP TABLE IF EXISTS "%s";`,
				op.TableName,
			)
			sqlStatements = append(sqlStatements, stmt)

		case diff.AddColumn:
			stmt := fmt.Sprintf(`ALTER TABLE "%s" DROP COLUMN "%s";`,
				op.TableName,
				op.Column.Name,
			)
			sqlStatements = append(sqlStatements, stmt)

		case diff.DropColumn:
			// For rollback, we need to recreate the column
			// Note: This is simplified - we don't have the original column definition
			// In a real implementation, you might want to store the original column definition
			stmt := fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN "%s" %s;`,
				op.TableName,
				op.ColumnName,
				"text", // Default type - ideally we'd store the original type
			)
			sqlStatements = append(sqlStatements, stmt)

		case diff.ModifyColumn:
			// For rollback, we need to revert the column modifications
			if op.OldColumn != nil {
				stmt, err := generateModifyColumnRollback(op)
				if err != nil {
					return nil, fmt.Errorf("generate MODIFY COLUMN rollback: %v", err)
				}
				sqlStatements = append(sqlStatements, stmt)
			}

		case diff.RenameColumn:
			// For rollback, rename back to original name
			stmt := fmt.Sprintf(`ALTER TABLE "%s" RENAME COLUMN "%s" TO "%s";`,
				op.TableName,
				op.NewColumnName,
				op.ColumnName,
			)
			sqlStatements = append(sqlStatements, stmt)

		case diff.DropTable:
			// For rollback, we need to recreate the table
			// Note: This is simplified - we don't have the original table definition
			stmt := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" (id serial PRIMARY KEY);`,
				op.TableName,
			)
			sqlStatements = append(sqlStatements, stmt)

		case diff.AddForeignKey:
			stmt := fmt.Sprintf(`ALTER TABLE "%s" DROP CONSTRAINT "fk_%s_%s";`,
				op.TableName,
				op.TableName,
				op.ForeignKey.ReferencesTable,
			)
			sqlStatements = append(sqlStatements, stmt)

		case diff.DropForeignKey:
			// For rollback, we need to recreate the foreign key
			stmt := fmt.Sprintf(`ALTER TABLE "%s" ADD CONSTRAINT "%s" FOREIGN KEY ("%s") REFERENCES "%s" ("%s")`,
				op.TableName,
				op.FKName,
				op.ColumnName, // We need to store the column name in the operation
				op.ForeignKey.ReferencesTable,
				op.ForeignKey.ReferencesColumn,
			)
			if op.ForeignKey.OnDelete != "" {
				stmt += fmt.Sprintf(" ON DELETE %s", op.ForeignKey.OnDelete)
			}
			if op.ForeignKey.OnUpdate != "" {
				stmt += fmt.Sprintf(" ON UPDATE %s", op.ForeignKey.OnUpdate)
			}
			sqlStatements = append(sqlStatements, stmt+";")

		case diff.CreateIndex:
			stmt := fmt.Sprintf(`DROP INDEX IF EXISTS "%s";`,
				op.Index.Name,
			)
			sqlStatements = append(sqlStatements, stmt)

		case diff.DropIndex:
			// For rollback, we need to recreate the index
			// Note: This is simplified - we don't have the original index definition
			stmt := fmt.Sprintf(`CREATE INDEX "%s" ON "%s" ("%s");`,
				op.IndexName,
				op.TableName,
				"column_name", // Placeholder - ideally we'd store the original definition
			)
			sqlStatements = append(sqlStatements, stmt)

		default:
			return nil, fmt.Errorf("unsupported rollback operation: %s", op.Type)
		}
	}

	return sqlStatements, nil
}

func generateCreateTable(op diff.Operation) (string, error) {
	stmt := fmt.Sprintf(`CREATE TABLE "%s" (`, op.TableName)

	for i, col := range op.Columns {
		stmt += fmt.Sprintf(`"%s" %s`, col.Name, col.Type)
		if col.Primary {
			stmt += " PRIMARY KEY"
		}
		if col.Unique {
			stmt += " UNIQUE"
		}
		if col.NotNull {
			stmt += " NOT NULL"
		}
		if col.Default != nil {
			stmt += fmt.Sprintf(" DEFAULT %s", *col.Default)
		}
		if i < len(op.Columns)-1 {
			stmt += ", "
		}
	}

	stmt += ");"

	return stmt, nil
}

func generateCreateIndex(op diff.Operation) (string, error) {
	if op.Index == nil {
		return "", fmt.Errorf("index is nil")
	}

	stmt := "CREATE"
	if op.Index.Unique {
		stmt += " UNIQUE"
	}
	
	stmt += " INDEX"
	if op.Index.Name != "" {
		stmt += fmt.Sprintf(` "%s"`, op.Index.Name)
	}
	
	stmt += fmt.Sprintf(` ON "%s"`, op.Index.Table)
	
	// Add index type if specified
	if op.Index.Type != "" && op.Index.Type != "btree" {
		stmt += fmt.Sprintf(" USING %s", op.Index.Type)
	}
	
	// Add columns
	stmt += " ("
	for i, col := range op.Index.Columns {
		if i > 0 {
			stmt += ", "
		}
		stmt += fmt.Sprintf(`"%s"`, col)
	}
	stmt += ");"

	return stmt, nil
}

func generateModifyColumn(op diff.Operation) (string, error) {
	if op.Column == nil || op.OldColumn == nil {
		return "", fmt.Errorf("column or old column is nil")
	}

	var statements []string

	// Type change
	if !strings.EqualFold(op.OldColumn.DataType, op.Column.Type) {
		stmt := fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "%s" TYPE %s`,
			op.TableName,
			op.Column.Name,
			op.Column.Type,
		)
		statements = append(statements, stmt)
	}

	// NOT NULL constraint change
	oldNullable := op.OldColumn.IsNullable
	newNullable := !op.Column.NotNull

	if oldNullable != newNullable {
		if newNullable {
			// Remove NOT NULL constraint
			stmt := fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "%s" DROP NOT NULL`,
				op.TableName,
				op.Column.Name,
			)
			statements = append(statements, stmt)
		} else {
			// Add NOT NULL constraint
			stmt := fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "%s" SET NOT NULL`,
				op.TableName,
				op.Column.Name,
			)
			statements = append(statements, stmt)
		}
	}

	// Default value change
	oldDefault := op.OldColumn.ColumnDefault
	newDefault := op.Column.Default

	if (oldDefault == nil && newDefault != nil) ||
		(oldDefault != nil && newDefault == nil) ||
		(oldDefault != nil && newDefault != nil && *oldDefault != *newDefault) {
		
		if newDefault == nil {
			// Remove default
			stmt := fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "%s" DROP DEFAULT`,
				op.TableName,
				op.Column.Name,
			)
			statements = append(statements, stmt)
		} else {
			// Set new default
			stmt := fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "%s" SET DEFAULT %s`,
				op.TableName,
				op.Column.Name,
				*newDefault,
			)
			statements = append(statements, stmt)
		}
	}

	if len(statements) == 0 {
		return "", fmt.Errorf("no modifications needed")
	}

	return strings.Join(statements, ";\n") + ";", nil
}

func generateModifyColumnRollback(op diff.Operation) (string, error) {
	if op.Column == nil || op.OldColumn == nil {
		return "", fmt.Errorf("column or old column is nil")
	}

	var statements []string

	// Type change rollback
	if !strings.EqualFold(op.OldColumn.DataType, op.Column.Type) {
		stmt := fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "%s" TYPE %s`,
			op.TableName,
			op.Column.Name,
			op.OldColumn.DataType,
		)
		statements = append(statements, stmt)
	}

	// NOT NULL constraint change rollback
	oldNullable := op.OldColumn.IsNullable
	newNullable := !op.Column.NotNull

	if oldNullable != newNullable {
		if oldNullable {
			// Remove NOT NULL constraint (rollback: add it back)
			stmt := fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "%s" DROP NOT NULL`,
				op.TableName,
				op.Column.Name,
			)
			statements = append(statements, stmt)
		} else {
			// Add NOT NULL constraint (rollback: remove it)
			stmt := fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "%s" SET NOT NULL`,
				op.TableName,
				op.Column.Name,
			)
			statements = append(statements, stmt)
		}
	}

	// Default value change rollback
	oldDefault := op.OldColumn.ColumnDefault
	newDefault := op.Column.Default

	if (oldDefault == nil && newDefault != nil) ||
		(oldDefault != nil && newDefault == nil) ||
		(oldDefault != nil && newDefault != nil && *oldDefault != *newDefault) {
		
		if oldDefault == nil {
			// Remove default (rollback: add it back)
			stmt := fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "%s" DROP DEFAULT`,
				op.TableName,
				op.Column.Name,
			)
			statements = append(statements, stmt)
		} else {
			// Set old default (rollback: restore original)
			stmt := fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "%s" SET DEFAULT %s`,
				op.TableName,
				op.Column.Name,
				*oldDefault,
			)
			statements = append(statements, stmt)
		}
	}

	if len(statements) == 0 {
		return "", fmt.Errorf("no rollback modifications needed")
	}

	return strings.Join(statements, ";\n") + ";", nil
}

// WriteMigrationFile saves the SQL statements into a timestamped .sql file with up/down sections
func WriteMigrationFile(sqlStatements []string, rollbackStatements []string) (string, error) {
	// Ensure migrations folder exists
	if _, err := os.Stat("migrations"); os.IsNotExist(err) {
		err = os.Mkdir("migrations", 0755)
		if err != nil {
			return "", fmt.Errorf("creating migrations folder: %v", err)
		}
	}

	// Create filename
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("migrations/%s_migration.sql", timestamp)

	// Create content with up/down sections
	content := "-- Migration: " + timestamp + "\n"
	content += "-- Description: Auto-generated migration\n\n"
	
	// Up migration
	content += "-- Up Migration\n"
	content += "-- ============\n"
	for _, stmt := range sqlStatements {
		content += stmt + "\n"
	}
	
	content += "\n-- Down Migration (Rollback)\n"
	content += "-- =======================\n"
	for _, stmt := range rollbackStatements {
		content += stmt + "\n"
	}

	// Write to file
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("writing migration file: %v", err)
	}

	return filename, nil
}
