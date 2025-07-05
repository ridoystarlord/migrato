package introspect

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ridoystarlord/migrato/utils"
)

type ExistingTable struct {
	TableName   string
	Columns     []ExistingColumn
	ForeignKeys []ExistingForeignKey
	Indexes     []ExistingIndex
}

type ExistingColumn struct {
	ColumnName    string
	DataType      string
	IsNullable    bool
	ColumnDefault *string
	IsPrimaryKey  bool
	IsUnique      bool
}

type ExistingForeignKey struct {
	ConstraintName    string
	ColumnName        string
	ReferencesTable   string
	ReferencesColumn  string
	OnDelete          string
	OnUpdate          string
}

type ExistingIndex struct {
	IndexName string
	TableName string
	Columns   []string
	IsUnique  bool
	IndexType string
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

		foreignKeys, err := getForeignKeys(ctx, pool, tableName)
		if err != nil {
			return nil, fmt.Errorf("getting foreign keys for table %s: %v", tableName, err)
		}

		indexes, err := getIndexes(ctx, pool, tableName)
		if err != nil {
			return nil, fmt.Errorf("getting indexes for table %s: %v", tableName, err)
		}

		tables = append(tables, ExistingTable{
			TableName:   tableName,
			Columns:     columns,
			ForeignKeys: foreignKeys,
			Indexes:     indexes,
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

func getForeignKeys(ctx context.Context, pool *pgxpool.Pool, tableName string) ([]ExistingForeignKey, error) {
	foreignKeysQuery := `
	SELECT
		tc.constraint_name,
		kcu.column_name,
		ccu.table_name AS foreign_table_name,
		ccu.column_name AS foreign_column_name,
		rc.delete_rule,
		rc.update_rule
	FROM information_schema.table_constraints AS tc
	JOIN information_schema.key_column_usage AS kcu
		ON tc.constraint_name = kcu.constraint_name
		AND tc.table_schema = kcu.table_schema
	JOIN information_schema.constraint_column_usage AS ccu
		ON ccu.constraint_name = tc.constraint_name
		AND ccu.table_schema = tc.table_schema
	LEFT JOIN information_schema.referential_constraints AS rc
		ON tc.constraint_name = rc.constraint_name
	WHERE tc.constraint_type = 'FOREIGN KEY' 
		AND tc.table_schema = 'public'
		AND tc.table_name = $1;
	`

	rows, err := pool.Query(ctx, foreignKeysQuery, tableName)
	if err != nil {
		return nil, fmt.Errorf("querying foreign keys: %v", err)
	}
	defer rows.Close()

	var foreignKeys []ExistingForeignKey
	for rows.Next() {
		var fk ExistingForeignKey
		if err := rows.Scan(
			&fk.ConstraintName,
			&fk.ColumnName,
			&fk.ReferencesTable,
			&fk.ReferencesColumn,
			&fk.OnDelete,
			&fk.OnUpdate,
		); err != nil {
			return nil, fmt.Errorf("scanning foreign key: %v", err)
		}
		foreignKeys = append(foreignKeys, fk)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("iterating foreign key rows: %v", rows.Err())
	}

	return foreignKeys, nil
}

func getIndexes(ctx context.Context, pool *pgxpool.Pool, tableName string) ([]ExistingIndex, error) {
	indexesQuery := `
	SELECT
		i.indexname,
		i.indexdef,
		(CASE WHEN i.indexdef LIKE '%UNIQUE%' THEN true ELSE false END) as is_unique,
		am.amname as index_type
	FROM pg_indexes i
	LEFT JOIN pg_class c ON i.indexname = c.relname
	LEFT JOIN pg_am am ON c.relam = am.oid
	WHERE i.tablename = $1 
		AND i.schemaname = 'public'
		AND i.indexname NOT LIKE '%_pkey'  -- Exclude primary key indexes
	ORDER BY i.indexname;
	`

	rows, err := pool.Query(ctx, indexesQuery, tableName)
	if err != nil {
		return nil, fmt.Errorf("querying indexes: %v", err)
	}
	defer rows.Close()

	var indexes []ExistingIndex
	for rows.Next() {
		var idx ExistingIndex
		var indexDef string
		if err := rows.Scan(
			&idx.IndexName,
			&indexDef,
			&idx.IsUnique,
			&idx.IndexType,
		); err != nil {
			return nil, fmt.Errorf("scanning index: %v", err)
		}
		
		idx.TableName = tableName
		// Extract column names from index definition
		// This is a simplified approach - in production you might want more robust parsing
		idx.Columns = extractColumnsFromIndexDef(indexDef)
		
		indexes = append(indexes, idx)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("iterating index rows: %v", rows.Err())
	}

	return indexes, nil
}

// extractColumnsFromIndexDef extracts column names from PostgreSQL index definition
// This is a simplified parser - in production you might want a more robust solution
func extractColumnsFromIndexDef(indexDef string) []string {
	// This is a basic implementation - you might want to use a proper SQL parser
	// For now, we'll return a placeholder
	return []string{"column_name"} // Placeholder
}
