package generator

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ridoystarlord/migrato/diff"
)

// formatDefaultValue properly formats a default value for SQL
func formatDefaultValue(defaultVal string) string {
	// If it's already quoted, return as is
	if strings.HasPrefix(defaultVal, "'") && strings.HasSuffix(defaultVal, "'") {
		return defaultVal
	}
	
	// If it's a function call (like now()), return as is
	if strings.Contains(defaultVal, "(") && strings.Contains(defaultVal, ")") {
		return defaultVal
	}
	
	// If it's a boolean, return as is
	if defaultVal == "true" || defaultVal == "false" {
		return defaultVal
	}
	
	// If it's a number, return as is
	if strings.ContainsAny(defaultVal, "0123456789") && !strings.ContainsAny(defaultVal, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return defaultVal
	}
	
	// Otherwise, quote it as a string
	return fmt.Sprintf("'%s'", strings.ReplaceAll(defaultVal, "'", "''"))
}

// GenerateSQL converts a list of Operations into raw SQL statements.
func GenerateSQL(ops []diff.Operation) ([]string, error) {
	var sqlStatements []string
	needsUUIDExtension := false

	// Check if any operation uses UUID types
	for _, op := range ops {
		if op.Type == diff.CreateTable {
			for _, col := range op.Columns {
				if strings.Contains(strings.ToLower(col.Type), "uuid") {
					needsUUIDExtension = true
					break
				}
			}
		}
		if needsUUIDExtension {
			break
		}
	}

	// Add UUID extension if needed
	if needsUUIDExtension {
		sqlStatements = append(sqlStatements, `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`)
	}

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
				stmt += fmt.Sprintf(" DEFAULT %s", formatDefaultValue(*op.Column.Default))
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
	needsUUIDExtension := false

	// Check if any operation uses UUID types
	for _, op := range ops {
		if op.Type == diff.CreateTable {
			for _, col := range op.Columns {
				if strings.Contains(strings.ToLower(col.Type), "uuid") {
					needsUUIDExtension = true
					break
				}
			}
		}
		if needsUUIDExtension {
			break
		}
	}

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
			if op.Column == nil {
				return nil, fmt.Errorf("rollback AddColumn: missing Column for table %s", op.TableName)
			}
			stmt := fmt.Sprintf(`ALTER TABLE "%s" DROP COLUMN "%s";`,
				op.TableName,
				op.Column.Name,
			)
			sqlStatements = append(sqlStatements, stmt)

		case diff.DropColumn:
			// For rollback, we need to recreate the column with original definition
			if op.OldColumn != nil {
				stmt := fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN "%s" %s`,
					op.TableName,
					op.ColumnName,
					op.OldColumn.DataType,
				)
				if !op.OldColumn.IsNullable {
					stmt += " NOT NULL"
				}
				if op.OldColumn.ColumnDefault != nil {
					stmt += fmt.Sprintf(" DEFAULT %s", formatDefaultValue(*op.OldColumn.ColumnDefault))
				}
				sqlStatements = append(sqlStatements, stmt+";")
			} else {
				// Fallback: create a basic text column if we don't have the original definition
				stmt := fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN "%s" text;`,
					op.TableName,
					op.ColumnName,
				)
				sqlStatements = append(sqlStatements, stmt)
			}

		case diff.ModifyColumn:
			if op.Column == nil || op.OldColumn == nil {
				return nil, fmt.Errorf("rollback ModifyColumn: missing Column or OldColumn for table %s", op.TableName)
			}
			stmt, err := generateModifyColumnRollback(op)
			if err != nil {
				return nil, fmt.Errorf("generate MODIFY COLUMN rollback: %v", err)
			}
			sqlStatements = append(sqlStatements, stmt)

		case diff.RenameColumn:
			if op.NewColumnName == "" || op.ColumnName == "" {
				return nil, fmt.Errorf("rollback RenameColumn: missing NewColumnName or ColumnName for table %s", op.TableName)
			}
			// For rollback, rename back to original name
			stmt := fmt.Sprintf(`ALTER TABLE "%s" RENAME COLUMN "%s" TO "%s";`,
				op.TableName,
				op.NewColumnName,
				op.ColumnName,
			)
			sqlStatements = append(sqlStatements, stmt)

		case diff.DropTable:
			// For rollback, we need to recreate the table with original definition
			if len(op.Columns) > 0 {
				stmt, err := generateCreateTable(op)
				if err != nil {
					return nil, fmt.Errorf("generate CREATE TABLE rollback: %v", err)
				}
				sqlStatements = append(sqlStatements, stmt)
			} else {
				// Fallback: create a basic table if we don't have the original definition
				stmt := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" (id serial PRIMARY KEY);`,
					op.TableName,
				)
				sqlStatements = append(sqlStatements, stmt)
			}

		case diff.AddForeignKey:
			if op.TableName == "" || op.ForeignKey == nil {
				return nil, fmt.Errorf("rollback AddForeignKey: missing TableName or ForeignKey")
			}
			// For rollback, drop the foreign key constraint
			constraintName := fmt.Sprintf("fk_%s_%s", op.TableName, op.ForeignKey.ReferencesTable)
			stmt := fmt.Sprintf(`ALTER TABLE "%s" DROP CONSTRAINT "%s";`,
				op.TableName,
				constraintName,
			)
			sqlStatements = append(sqlStatements, stmt)

		case diff.DropForeignKey:
			if op.TableName == "" || op.FKName == "" || op.ForeignKey == nil || op.ColumnName == "" {
				return nil, fmt.Errorf("rollback DropForeignKey: missing TableName, FKName, ForeignKey, or ColumnName")
			}
			// For rollback, we need to recreate the foreign key
			stmt := fmt.Sprintf(`ALTER TABLE "%s" ADD CONSTRAINT "%s" FOREIGN KEY ("%s") REFERENCES "%s" ("%s")`,
				op.TableName,
				op.FKName,
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

		case diff.CreateIndex:
			if op.Index == nil || op.Index.Name == "" {
				return nil, fmt.Errorf("rollback CreateIndex: missing Index or Index.Name for table %s", op.TableName)
			}
			stmt := fmt.Sprintf(`DROP INDEX IF EXISTS "%s";`,
				op.Index.Name,
			)
			sqlStatements = append(sqlStatements, stmt)

		case diff.DropIndex:
			if op.IndexName != "" && op.TableName != "" {
				// For rollback, we need to recreate the index
				// Note: We don't have the original index definition, so we'll create a basic index
				columnName := "id" // Default column name
				if op.ColumnName != "" {
					columnName = op.ColumnName
				}
				stmt := fmt.Sprintf(`CREATE INDEX "%s" ON "%s" ("%s");`,
					op.IndexName,
					op.TableName,
					columnName,
				)
				sqlStatements = append(sqlStatements, stmt)
			} else {
				return nil, fmt.Errorf("rollback DropIndex: missing IndexName or TableName for index rollback")
			}

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
			stmt += fmt.Sprintf(" DEFAULT %s", formatDefaultValue(*col.Default))
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
				formatDefaultValue(*newDefault),
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
				formatDefaultValue(*oldDefault),
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
