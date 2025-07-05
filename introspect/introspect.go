package introspect

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/ridoystarlord/migrato/utils"
)

type ExistingTable struct {
	TableName string
	Columns   []ExistingColumn
}

type ExistingColumn struct {
	ColumnName    string
	DataType      string
	IsNullable    bool
	ColumnDefault *string
	IsPrimaryKey  bool
	IsUnique      bool
}

func IntrospectDatabase() ([]ExistingTable, error) {
	utils.LoadEnv()
	connStr := utils.GetDatabaseURL()
	if connStr == "" {
		return nil, fmt.Errorf("DATABASE_URL not set in environment")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	// First, list all tables in public schema
	tablesQuery := `
	SELECT table_name
	FROM information_schema.tables
	WHERE table_schema = 'public' AND table_type='BASE TABLE'
	ORDER BY table_name;
	`

	rows, err := conn.Query(ctx, tablesQuery)
	if err != nil {
		return nil, fmt.Errorf("querying tables: %v", err)
	}
	defer rows.Close()

	// Collect table names into a slice
	var tableNames []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("scanning table name: %v", err)
		}
		tableNames = append(tableNames, tableName)
	}

	// Make sure any error from iteration is captured
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterating table rows: %v", rows.Err())
	}

	var tables []ExistingTable
	// Now that rows are done, loop over table names
	for _, tableName := range tableNames {
		columns, err := getColumns(ctx, conn, tableName)
		if err != nil {
			return nil, fmt.Errorf("getting columns for table %s: %v", tableName, err)
		}

		tables = append(tables, ExistingTable{
			TableName: tableName,
			Columns:   columns,
		})
	}

	return tables, nil
}

func getColumns(ctx context.Context, conn *pgx.Conn, tableName string) ([]ExistingColumn, error) {
	columnsQuery := `
	SELECT
		c.column_name,
		c.data_type,
		(c.is_nullable = 'YES') as is_nullable,
		c.column_default,
		(CASE WHEN tc.constraint_type = 'PRIMARY KEY' THEN true ELSE false END) as is_primary,
		(CASE WHEN tc.constraint_type = 'UNIQUE' THEN true ELSE false END) as is_unique
	FROM information_schema.columns c
	LEFT JOIN information_schema.key_column_usage kcu
		ON c.table_name = kcu.table_name AND c.column_name = kcu.column_name
	LEFT JOIN information_schema.table_constraints tc
		ON kcu.constraint_name = tc.constraint_name AND kcu.table_name = tc.table_name
	WHERE c.table_schema = 'public' AND c.table_name = $1
	ORDER BY c.ordinal_position;
	`

	rows, err := conn.Query(ctx, columnsQuery, tableName)
	if err != nil {
		return nil, fmt.Errorf("querying columns: %v", err)
	}
	defer rows.Close()

	var columns []ExistingColumn
	for rows.Next() {
		var col ExistingColumn
		var nullable bool
		if err := rows.Scan(
			&col.ColumnName,
			&col.DataType,
			&nullable,
			&col.ColumnDefault,
			&col.IsPrimaryKey,
			&col.IsUnique,
		); err != nil {
			return nil, fmt.Errorf("scanning column: %v", err)
		}
		col.IsNullable = nullable
		columns = append(columns, col)
	}

	// Make sure any error from iteration is captured
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterating column rows: %v", rows.Err())
	}

	return columns, nil
}
