package generator

import (
	"fmt"
	"os"
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
