package introspect

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
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
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %v", err)
	}
	defer pool.Close()

	tablesQuery := `
	SELECT table_name
	FROM information_schema.tables
	WHERE table_schema = 'public' AND table_type='BASE TABLE'
	ORDER BY table_name;
	`

	rows, err := pool.Query(ctx, tablesQuery)
	if err != nil {
		return nil, fmt.Errorf("querying tables: %v", err)
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("scanning table name: %v", err)
		}
		tableNames = append(tableNames, tableName)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("iterating table rows: %v", rows.Err())
	}

	var tables []ExistingTable
	for _, tableName := range tableNames {
		columns, err := getColumns(ctx, pool, tableName)
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

func getColumns(ctx context.Context, pool *pgxpool.Pool, tableName string) ([]ExistingColumn, error) {
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

	rows, err := pool.Query(ctx, columnsQuery, tableName)
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

	if rows.Err() != nil {
		return nil, fmt.Errorf("iterating column rows: %v", rows.Err())
	}

	return columns, nil
}
