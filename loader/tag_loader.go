package loader

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/ridoystarlord/migrato/schema"
)

// TagLoader loads database schema from Go structs with database tags
type TagLoader struct {
	modelsDir string
}

// NewTagLoader creates a new tag loader
func NewTagLoader(modelsDir string) *TagLoader {
	return &TagLoader{
		modelsDir: modelsDir,
	}
}

// LoadModelsFromTags loads database schema from Go structs with tags
func LoadModelsFromTags(modelsDir string) ([]schema.Model, error) {
	loader := NewTagLoader(modelsDir)
	return loader.Load()
}

// Load loads all models from the models directory
func (tl *TagLoader) Load() ([]schema.Model, error) {
	// Check if models directory exists
	if _, err := os.Stat(tl.modelsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("models directory '%s' does not exist. Run 'migrato init' first", tl.modelsDir)
	}

	var models []schema.Model

	// Walk through all .go files in the models directory
	err := filepath.Walk(tl.modelsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Parse the Go file
		fileModels, err := tl.parseGoFile(path)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %v", path, err)
		}

		models = append(models, fileModels...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to load models: %v", err)
	}

	return models, nil
}

// parseGoFile parses a single Go file and extracts models
func (tl *TagLoader) parseGoFile(filePath string) ([]schema.Model, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file: %v", err)
	}

	var models []schema.Model

	// Walk through the AST to find struct declarations
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.TypeSpec:
			if structType, ok := x.Type.(*ast.StructType); ok {
				model := tl.parseStruct(x.Name.Name, structType)
				if model != nil {
					models = append(models, *model)
				}
			}
		}
		return true
	})

	return models, nil
}

// parseStruct converts a Go struct to a schema.Model
func (tl *TagLoader) parseStruct(structName string, structType *ast.StructType) *schema.Model {
	model := &schema.Model{
		TableName: tl.getTableName(structName),
		Columns:   []schema.Column{},
		Indexes:   []schema.Index{},
	}

	// Parse struct fields
	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue // Skip embedded fields for now
		}

		fieldName := field.Names[0].Name
		if !ast.IsExported(fieldName) {
			continue // Skip unexported fields
		}

		column := tl.parseField(fieldName, field)
		if column != nil {
			model.Columns = append(model.Columns, *column)
		}
	}

	// Extract table-level indexes from struct tags
	tl.parseTableIndexes(model, structType)

	return model
}

// parseField converts a struct field to a schema.Column
func (tl *TagLoader) parseField(fieldName string, field *ast.Field) *schema.Column {
	// Get the field type
	fieldType := tl.getFieldType(field.Type)
	if fieldType == "" {
		return nil
	}

	// Parse the tag
	tag := tl.parseTag(field.Tag)

	// Skip if field is marked to be ignored
	if tag.Ignore {
		return nil
	}

	column := &schema.Column{
		Name:     tag.ColumnName,
		Type:     tag.DataType,
		Primary:  tag.Primary,
		Unique:   tag.Unique,
		NotNull:  tag.NotNull,
		Default:  tag.Default,
		Index:    tag.Index,
		ForeignKey: tag.ForeignKey,
	}

	// If no column name specified, use the field name (converted to snake_case)
	if column.Name == "" {
		column.Name = tl.toSnakeCase(fieldName)
	}

	// If no data type specified, infer from Go type
	if column.Type == "" {
		column.Type = tl.inferDataType(fieldType)
	}

	return column
}

// parseTag parses the struct tag for database information
func (tl *TagLoader) parseTag(tag *ast.BasicLit) *FieldTag {
	if tag == nil {
		return &FieldTag{}
	}

	// Remove quotes from tag
	tagValue := strings.Trim(tag.Value, "`")
	
	// Parse the tag using reflection
	tagStruct := reflect.StructTag(tagValue)
	migratoTag := tagStruct.Get("migrato")
	
	return tl.parseDBTag(migratoTag)
}

// parseDBTag parses the "db" tag value
func (tl *TagLoader) parseDBTag(dbTag string) *FieldTag {
	tag := &FieldTag{}

	if dbTag == "" || dbTag == "-" {
		tag.Ignore = true
		return tag
	}

	// Parse tag parts (e.g., "column_name;type:text;primary;unique;not_null;default:value")
	parts := strings.Split(dbTag, ";")
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Handle key-value pairs
		if strings.Contains(part, ":") {
			kv := strings.SplitN(part, ":", 2)
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])
				
				switch key {
				case "column":
					tag.ColumnName = value
				case "type":
					tag.DataType = value
				case "default":
					tag.Default = &value
				case "fk":
					tag.ForeignKey = tl.parseForeignKey(value)
				case "index":
					tag.Index = tl.parseIndexConfig(value)
				}
			}
		} else {
			// Handle boolean flags
			switch part {
			case "primary":
				tag.Primary = true
			case "unique":
				tag.Unique = true
			case "not_null":
				tag.NotNull = true
			case "index":
				tag.Index = &schema.IndexConfig{
					Type: "btree",
				}
			}
		}
	}

	return tag
}

// parseForeignKey parses foreign key specification
func (tl *TagLoader) parseForeignKey(fkSpec string) *schema.ForeignKey {
	// Format: "table.column:on_delete:on_update"
	parts := strings.Split(fkSpec, ":")
	if len(parts) < 1 {
		return nil
	}

	refParts := strings.Split(parts[0], ".")
	if len(refParts) != 2 {
		return nil
	}

	fk := &schema.ForeignKey{
		ReferencesTable:  refParts[0],
		ReferencesColumn: refParts[1],
	}

	if len(parts) > 1 {
		fk.OnDelete = parts[1]
	}
	if len(parts) > 2 {
		fk.OnUpdate = parts[2]
	}

	return fk
}

// parseIndexConfig parses index configuration
func (tl *TagLoader) parseIndexConfig(indexSpec string) *schema.IndexConfig {
	// Format: "name:type:unique"
	parts := strings.Split(indexSpec, ":")
	
	config := &schema.IndexConfig{
		Type: "btree", // default
	}

	if len(parts) > 0 && parts[0] != "" {
		config.Name = parts[0]
	}
	if len(parts) > 1 {
		config.Type = parts[1]
	}
	if len(parts) > 2 && parts[2] == "unique" {
		config.Unique = true
	}

	return config
}

// parseTableIndexes extracts table-level indexes from struct tags
func (tl *TagLoader) parseTableIndexes(model *schema.Model, structType *ast.StructType) {
	// Look for table-level index tags in struct comments or special fields
	// This is a simplified implementation - you might want to enhance this
}

// getFieldType extracts the Go type name from an ast.Expr
func (tl *TagLoader) getFieldType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return tl.getFieldType(t.X)
	case *ast.ArrayType:
		return "[]" + tl.getFieldType(t.Elt)
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name + "." + t.Sel.Name
		}
	}
	return ""
}

// getTableName converts struct name to table name
func (tl *TagLoader) getTableName(structName string) string {
	// Convert PascalCase to snake_case and pluralize
	tableName := tl.toSnakeCase(structName)
	
	// Simple pluralization rules
	if strings.HasSuffix(tableName, "y") {
		// Change y to ies (e.g., category -> categories)
		tableName = strings.TrimSuffix(tableName, "y") + "ies"
	} else if !strings.HasSuffix(tableName, "s") {
		// Add s if it doesn't end with s
		tableName += "s"
	}
	
	return tableName
}

// inferDataType infers PostgreSQL data type from Go type
func (tl *TagLoader) inferDataType(goType string) string {
	switch goType {
	case "int", "int32":
		return "integer"
	case "int64":
		return "bigint"
	case "string":
		return "text"
	case "bool":
		return "boolean"
	case "float32", "float64":
		return "numeric"
	case "time.Time":
		return "timestamp"
	case "uuid.UUID":
		return "uuid"
	default:
		// Handle common patterns
		if strings.HasPrefix(goType, "[]") {
			return "jsonb" // Arrays as JSON
		}
		if strings.Contains(goType, "time.Time") {
			return "timestamp"
		}
		if strings.Contains(goType, "uuid.UUID") {
			return "uuid"
		}
		return "text" // Default fallback
	}
}

// toSnakeCase converts PascalCase to snake_case
func (tl *TagLoader) toSnakeCase(s string) string {
	var result string
	var prev rune
	
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' && prev >= 'a' && prev <= 'z' {
			result += "_"
		}
		result += string(r)
		prev = r
	}
	return strings.ToLower(result)
}

// FieldTag represents parsed database tag information
type FieldTag struct {
	Ignore      bool
	ColumnName  string
	DataType    string
	Primary     bool
	Unique      bool
	NotNull     bool
	Default     *string
	Index       *schema.IndexConfig
	ForeignKey  *schema.ForeignKey
} 