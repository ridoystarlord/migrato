package generator

import (
	"fmt"
	"os"
	"time"

	"github.com/ridoystarlord/go-migration-buddy/diff"
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

		default:
			return nil, fmt.Errorf("unsupported operation: %s", op.Type)
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

// WriteMigrationFile saves the SQL statements into a timestamped .sql file
func WriteMigrationFile(sqlStatements []string) (string, error) {
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

	// Join statements with newlines
	content := ""
	for _, stmt := range sqlStatements {
		content += stmt + "\n\n"
	}

	// Write to file
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("writing migration file: %v", err)
	}

	return filename, nil
}
