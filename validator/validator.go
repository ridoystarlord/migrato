package validator

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ridoystarlord/migrato/database"
	"github.com/ridoystarlord/migrato/schema"
)

// ValidationError represents a validation error with details
type ValidationError struct {
	Type    string `json:"type"`
	Table   string `json:"table,omitempty"`
	Column  string `json:"column,omitempty"`
	Index   string `json:"index,omitempty"`
	Message string `json:"message"`
	Severity string `json:"severity"` // "error", "warning", "info"
}

// ValidationResult contains all validation results
type ValidationResult struct {
	Valid   bool             `json:"valid"`
	Errors  []ValidationError `json:"errors"`
	Warnings []ValidationError `json:"warnings"`
	Info    []ValidationError `json:"info"`
}

// SchemaValidator validates YAML schemas against database constraints
type SchemaValidator struct {
	pool *pgxpool.Pool
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() (*SchemaValidator, error) {
	pool, err := database.GetPool()
	if err != nil {
		return nil, fmt.Errorf("failed to get database pool: %v", err)
	}

	return &SchemaValidator{
		pool: pool,
	}, nil
}

// ValidateSchema validates a complete schema against database constraints
func (v *SchemaValidator) ValidateSchema(models []schema.Model) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
		Info:     []ValidationError{},
	}

	ctx := context.Background()

	// Get current database state
	dbTables, err := v.getDatabaseTables(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database tables: %v", err)
	}

	// Validate each model
	for _, model := range models {
		if err := v.validateModel(ctx, model, dbTables, result); err != nil {
			return nil, fmt.Errorf("failed to validate model %s: %v", model.TableName, err)
		}
	}

	// Cross-table validations
	if err := v.validateCrossTableConstraints(models, result); err != nil {
		return nil, fmt.Errorf("failed to validate cross-table constraints: %v", err)
	}

	// Update overall validity
	result.Valid = len(result.Errors) == 0

	return result, nil
}

// ValidateSchemaWithoutDB validates a schema without database connection
func (v *SchemaValidator) ValidateSchemaWithoutDB(models []schema.Model) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
		Info:     []ValidationError{},
	}

	// Validate each model
	for _, model := range models {
		if err := v.validateModelWithoutDB(model, result); err != nil {
			return nil, fmt.Errorf("failed to validate model %s: %v", model.TableName, err)
		}
	}

	// Cross-table validations
	if err := v.validateCrossTableConstraints(models, result); err != nil {
		return nil, fmt.Errorf("failed to validate cross-table constraints: %v", err)
	}

	// Update overall validity
	result.Valid = len(result.Errors) == 0

	return result, nil
}

// validateModel validates a single model
func (v *SchemaValidator) validateModel(ctx context.Context, model schema.Model, dbTables map[string]bool, result *ValidationResult) error {
	// Validate table name
	if err := v.validateTableName(model.TableName); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Type:     "table_name",
			Table:    model.TableName,
			Message:  err.Error(),
			Severity: "error",
		})
	}

	// Check if table already exists
	if dbTables[model.TableName] {
		result.Info = append(result.Info, ValidationError{
			Type:     "table_exists",
			Table:    model.TableName,
			Message:  fmt.Sprintf("Table '%s' already exists in database", model.TableName),
			Severity: "info",
		})
	}

	// Validate columns
	if err := v.validateColumns(model, result); err != nil {
		return err
	}

	// Validate indexes
	if err := v.validateIndexes(model, result); err != nil {
		return err
	}

	// Validate foreign keys
	if err := v.validateForeignKeys(model, result); err != nil {
		return err
	}

	return nil
}

// validateModelWithoutDB validates a single model without database connection
func (v *SchemaValidator) validateModelWithoutDB(model schema.Model, result *ValidationResult) error {
	// Validate table name
	if err := v.validateTableName(model.TableName); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Type:     "table_name",
			Table:    model.TableName,
			Message:  err.Error(),
			Severity: "error",
		})
	}

	// Validate columns
	if err := v.validateColumns(model, result); err != nil {
		return err
	}

	// Validate indexes
	if err := v.validateIndexes(model, result); err != nil {
		return err
	}

	// Validate foreign keys
	if err := v.validateForeignKeys(model, result); err != nil {
		return err
	}

	return nil
}

// validateTableName validates table name format
func (v *SchemaValidator) validateTableName(tableName string) error {
	if tableName == "" {
		return fmt.Errorf("table name cannot be empty")
	}

	if len(tableName) > 63 {
		return fmt.Errorf("table name '%s' is too long (max 63 characters)", tableName)
	}

	// Check for valid characters (PostgreSQL identifier rules)
	for _, char := range tableName {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_') {
			return fmt.Errorf("table name '%s' contains invalid character '%c'", tableName, char)
		}
	}

	// Check for reserved keywords
	reservedKeywords := []string{"user", "order", "group", "table", "index", "view", "schema"}
	for _, keyword := range reservedKeywords {
		if strings.ToLower(tableName) == keyword {
			return fmt.Errorf("table name '%s' is a reserved keyword", tableName)
		}
	}

	return nil
}

// validateColumns validates all columns in a model
func (v *SchemaValidator) validateColumns(model schema.Model, result *ValidationResult) error {
	if len(model.Columns) == 0 {
		result.Errors = append(result.Errors, ValidationError{
			Type:     "no_columns",
			Table:    model.TableName,
			Message:  fmt.Sprintf("Table '%s' must have at least one column", model.TableName),
			Severity: "error",
		})
		return nil
	}

	columnNames := make(map[string]bool)
	hasPrimaryKey := false

	for _, column := range model.Columns {
		// Check for duplicate column names
		if columnNames[column.Name] {
			result.Errors = append(result.Errors, ValidationError{
				Type:     "duplicate_column",
				Table:    model.TableName,
				Column:   column.Name,
				Message:  fmt.Sprintf("Duplicate column name '%s' in table '%s'", column.Name, model.TableName),
				Severity: "error",
			})
			continue
		}
		columnNames[column.Name] = true

		// Validate column name
		if err := v.validateColumnName(column.Name); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Type:     "column_name",
				Table:    model.TableName,
				Column:   column.Name,
				Message:  err.Error(),
				Severity: "error",
			})
		}

		// Validate data type
		if err := v.validateDataType(column.Type); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Type:     "data_type",
				Table:    model.TableName,
				Column:   column.Name,
				Message:  err.Error(),
				Severity: "error",
			})
		}

		// Track primary key
		if column.Primary {
			hasPrimaryKey = true
		}

		// Validate default value
		if column.Default != nil {
			if err := v.validateDefaultValue(column.Type, *column.Default); err != nil {
				result.Warnings = append(result.Warnings, ValidationError{
					Type:     "default_value",
					Table:    model.TableName,
					Column:   column.Name,
					Message:  err.Error(),
					Severity: "warning",
				})
			}
		}

		// Validate foreign key
		if column.ForeignKey != nil {
			if err := v.validateForeignKeyDefinition(column, model.TableName); err != nil {
				result.Errors = append(result.Errors, ValidationError{
					Type:     "foreign_key",
					Table:    model.TableName,
					Column:   column.Name,
					Message:  err.Error(),
					Severity: "error",
				})
			}
		}
	}

	// Check for primary key
	if !hasPrimaryKey {
		result.Warnings = append(result.Warnings, ValidationError{
			Type:     "no_primary_key",
			Table:    model.TableName,
			Message:  fmt.Sprintf("Table '%s' has no primary key defined", model.TableName),
			Severity: "warning",
		})
	}

	return nil
}

// validateColumnName validates column name format
func (v *SchemaValidator) validateColumnName(columnName string) error {
	if columnName == "" {
		return fmt.Errorf("column name cannot be empty")
	}

	if len(columnName) > 63 {
		return fmt.Errorf("column name '%s' is too long (max 63 characters)", columnName)
	}

	// Check for valid characters
	for _, char := range columnName {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_') {
			return fmt.Errorf("column name '%s' contains invalid character '%c'", columnName, char)
		}
	}

	return nil
}

// validateDataType validates PostgreSQL data type
func (v *SchemaValidator) validateDataType(dataType string) error {
	validTypes := map[string]bool{
		// Numeric types
		"smallint": true, "integer": true, "bigint": true,
		"decimal": true, "numeric": true, "real": true, "double precision": true,
		"serial": true, "bigserial": true, "smallserial": true,
		
		// Character types
		"character varying": true, "varchar": true, "character": true, "char": true,
		"text": true,
		
		// Binary data types
		"bytea": true,
		
		// Date/time types
		"timestamp": true, "timestamp with time zone": true, "timestamptz": true,
		"date": true, "time": true, "time with time zone": true, "timetz": true,
		"interval": true,
		
		// Boolean type
		"boolean": true, "bool": true,
		
		// JSON types
		"json": true, "jsonb": true,
		
		// UUID type
		"uuid": true,
		
		// Geometric types
		"point": true, "line": true, "lseg": true, "box": true,
		"path": true, "polygon": true, "circle": true,
		
		// Network address types
		"cidr": true, "inet": true, "macaddr": true, "macaddr8": true,
		
		// Bit string types
		"bit": true, "bit varying": true,
		
		// Text search types
		"tsvector": true, "tsquery": true,
		
		// Array types (basic support)
		"integer[]": true, "text[]": true, "varchar[]": true,
	}

	if !validTypes[strings.ToLower(dataType)] {
		return fmt.Errorf("unsupported data type '%s'", dataType)
	}

	return nil
}

// validateDefaultValue validates default value against data type
func (v *SchemaValidator) validateDefaultValue(dataType, defaultValue string) error {
	// This is a basic validation - in a real implementation, you'd want more sophisticated type checking
	dataType = strings.ToLower(dataType)
	
	switch {
	case strings.Contains(dataType, "int") || strings.Contains(dataType, "serial"):
		// Check if it's a valid number or function
		if defaultValue != "now()" && defaultValue != "uuid_generate_v4()" {
			// Try to parse as number
			if !strings.Contains(defaultValue, "(") && !strings.Contains(defaultValue, "'") {
				// Should be a number
				if strings.Contains(defaultValue, ".") {
					return fmt.Errorf("integer type cannot have decimal default value '%s'", defaultValue)
				}
			}
		}
	case strings.Contains(dataType, "char") || dataType == "text":
		// String types should have quoted defaults (except for functions)
		if !strings.Contains(defaultValue, "(") && !strings.HasPrefix(defaultValue, "'") && !strings.HasPrefix(defaultValue, "\"") {
			return fmt.Errorf("string type should have quoted default value '%s'", defaultValue)
		}
	case dataType == "boolean" || dataType == "bool":
		validBools := []string{"true", "false", "TRUE", "FALSE"}
		isValid := false
		for _, valid := range validBools {
			if defaultValue == valid {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("boolean type should have true/false default value, got '%s'", defaultValue)
		}
	}

	return nil
}

// validateForeignKeyDefinition validates foreign key definition
func (v *SchemaValidator) validateForeignKeyDefinition(column schema.Column, tableName string) error {
	if column.ForeignKey == nil {
		return nil
	}

	fk := column.ForeignKey

	if fk.ReferencesTable == "" {
		return fmt.Errorf("foreign key references table cannot be empty")
	}

	if fk.ReferencesColumn == "" {
		return fmt.Errorf("foreign key references column cannot be empty")
	}

	if fk.ReferencesTable == tableName && fk.ReferencesColumn == column.Name {
		return fmt.Errorf("foreign key cannot reference itself")
	}

	// Validate onDelete and onUpdate actions
	validActions := []string{"CASCADE", "SET NULL", "SET DEFAULT", "RESTRICT", "NO ACTION"}
	
	if fk.OnDelete != "" {
		isValid := false
		for _, action := range validActions {
			if strings.ToUpper(fk.OnDelete) == action {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid onDelete action '%s', must be one of: %v", fk.OnDelete, validActions)
		}
	}

	if fk.OnUpdate != "" {
		isValid := false
		for _, action := range validActions {
			if strings.ToUpper(fk.OnUpdate) == action {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid onUpdate action '%s', must be one of: %v", fk.OnUpdate, validActions)
		}
	}

	return nil
}

// validateIndexes validates indexes in a model
func (v *SchemaValidator) validateIndexes(model schema.Model, result *ValidationResult) error {
	indexNames := make(map[string]bool)
	columnNames := make(map[string]bool)

	// Build column name map
	for _, column := range model.Columns {
		columnNames[column.Name] = true
	}

	for _, index := range model.Indexes {
		// Check for duplicate index names
		if indexNames[index.Name] {
			result.Errors = append(result.Errors, ValidationError{
				Type:     "duplicate_index",
				Table:    model.TableName,
				Index:    index.Name,
				Message:  fmt.Sprintf("Duplicate index name '%s' in table '%s'", index.Name, model.TableName),
				Severity: "error",
			})
			continue
		}
		indexNames[index.Name] = true

		// Validate index name
		if err := v.validateIndexName(index.Name); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Type:     "index_name",
				Table:    model.TableName,
				Index:    index.Name,
				Message:  err.Error(),
				Severity: "error",
			})
		}

		// Validate index columns exist
		for _, columnName := range index.Columns {
			if !columnNames[columnName] {
				result.Errors = append(result.Errors, ValidationError{
					Type:     "index_column_not_found",
					Table:    model.TableName,
					Index:    index.Name,
					Column:   columnName,
					Message:  fmt.Sprintf("Index '%s' references non-existent column '%s' in table '%s'", index.Name, columnName, model.TableName),
					Severity: "error",
				})
			}
		}
	}

	return nil
}

// validateIndexName validates index name format
func (v *SchemaValidator) validateIndexName(indexName string) error {
	if indexName == "" {
		return fmt.Errorf("index name cannot be empty")
	}

	if len(indexName) > 63 {
		return fmt.Errorf("index name '%s' is too long (max 63 characters)", indexName)
	}

	// Check for valid characters
	for _, char := range indexName {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_') {
			return fmt.Errorf("index name '%s' contains invalid character '%c'", indexName, char)
		}
	}

	return nil
}

// validateForeignKeys validates foreign keys in a model
func (v *SchemaValidator) validateForeignKeys(model schema.Model, result *ValidationResult) error {
	// Foreign key validation is already done in validateColumns
	// This method can be used for additional cross-table validations
	return nil
}

// validateCrossTableConstraints validates constraints across tables
func (v *SchemaValidator) validateCrossTableConstraints(models []schema.Model, result *ValidationResult) error {
	// Build table and column maps
	tableMap := make(map[string]schema.Model)
	columnMap := make(map[string]map[string]schema.Column)

	for _, model := range models {
		tableMap[model.TableName] = model
		columnMap[model.TableName] = make(map[string]schema.Column)
		for _, column := range model.Columns {
			columnMap[model.TableName][column.Name] = column
		}
	}

	// Validate foreign key references
	for _, model := range models {
		for _, column := range model.Columns {
			if column.ForeignKey != nil {
				fk := column.ForeignKey
				
				// Check if referenced table exists
				if _, exists := tableMap[fk.ReferencesTable]; !exists {
					result.Errors = append(result.Errors, ValidationError{
						Type:     "foreign_key_table_not_found",
						Table:    model.TableName,
						Column:   column.Name,
						Message:  fmt.Sprintf("Foreign key references non-existent table '%s'", fk.ReferencesTable),
						Severity: "error",
					})
					continue
				}

				// Check if referenced column exists
				if _, exists := columnMap[fk.ReferencesTable][fk.ReferencesColumn]; !exists {
					result.Errors = append(result.Errors, ValidationError{
						Type:     "foreign_key_column_not_found",
						Table:    model.TableName,
						Column:   column.Name,
						Message:  fmt.Sprintf("Foreign key references non-existent column '%s' in table '%s'", fk.ReferencesColumn, fk.ReferencesTable),
						Severity: "error",
					})
				}
			}
		}
	}

	return nil
}

// getDatabaseTables gets list of existing tables from database
func (v *SchemaValidator) getDatabaseTables(ctx context.Context) (map[string]bool, error) {
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
	`

	rows, err := v.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := make(map[string]bool)
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables[tableName] = true
	}

	return tables, nil
} 