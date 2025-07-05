package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ridoystarlord/migrato/loader"
	"github.com/ridoystarlord/migrato/schema"
	"github.com/spf13/cobra"
)

var outputDir string
var packageName string

func init() {
	generateStructsCmd.Flags().StringVarP(&schemaFile, "file", "f", "schema.yaml", "Schema YAML file to load")
	generateStructsCmd.Flags().StringVarP(&outputDir, "output", "o", "models", "Output directory for generated structs")
	generateStructsCmd.Flags().StringVarP(&packageName, "package", "p", "models", "Package name for generated structs")
}

var generateStructsCmd = &cobra.Command{
	Use:   "generate-structs",
	Short: "Generate Go structs from schema",
	Long: `Generate Go structs from your YAML schema with proper tags and relationships.

Examples:
  migrato generate-structs                    # Generate structs in ./models/
  migrato generate-structs -o ./internal/models  # Custom output directory
  migrato generate-structs -p entities         # Custom package name
`,
	Run: func(cmd *cobra.Command, args []string) {
		models, err := loader.LoadModelsFromYAML(schemaFile)
		if err != nil {
			fmt.Println("❌ Loading schema.yaml:", err)
			os.Exit(1)
		}

		// Create output directories
		modelsDir := filepath.Join(outputDir, "models")
		repoDir := filepath.Join(outputDir, "repositories")
		
		if err := os.MkdirAll(modelsDir, 0755); err != nil {
			fmt.Println("❌ Creating models directory:", err)
			os.Exit(1)
		}
		
		if err := os.MkdirAll(repoDir, 0755); err != nil {
			fmt.Println("❌ Creating repositories directory:", err)
			os.Exit(1)
		}

		// Generate structs
		if err := generateStructs(models, modelsDir, repoDir); err != nil {
			fmt.Println("❌ Generating structs:", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Generated Go structs in %s/\n", outputDir)
		fmt.Printf("   Models: %s/\n", modelsDir)
		fmt.Printf("   Repositories: %s/\n", repoDir)
	},
}

type StructData struct {
	PackageName string
	Model       ModelData
}

type ModelData struct {
	Name       string
	Fields     []FieldData
	Relations  []RelationData
	Indexes    []IndexData
	TableName  string
}

type FieldData struct {
	Name       string
	Type       string
	Tags       string
	Comment    string
}

type RelationData struct {
	Name       string
	Type       string
	FieldName  string
	TargetType string
	IsMany     bool
}

type IndexData struct {
	Name    string
	Columns []string
	Unique  bool
}

func generateStructs(models []schema.Model, modelsDir, repoDir string) error {
	for _, model := range models {
		md := ModelData{
			Name:      toPascalCase(model.TableName),
			TableName: model.TableName,
		}

		// Generate fields from columns
		for _, col := range model.Columns {
			field := FieldData{
				Name:    toPascalCase(col.Name),
				Type:    mapColumnTypeToGoType(col.Type),
				Tags:    generateTags(col, model.TableName),
				Comment: fmt.Sprintf("// %s", col.Name),
			}
			md.Fields = append(md.Fields, field)
		}

		// Generate relations
		for _, rel := range model.Relations {
			relation := RelationData{
				Name:       toPascalCase(rel.Name),
				Type:       string(rel.Type),
				FieldName:  toPascalCase(rel.FromColumn),
				TargetType: toPascalCase(rel.ToTable),
				IsMany:     rel.Type == schema.OneToMany || rel.Type == schema.ManyToMany,
			}
			md.Relations = append(md.Relations, relation)
		}

		// Generate indexes info
		for _, idx := range model.Indexes {
			index := IndexData{
				Name:    idx.Name,
				Columns: idx.Columns,
				Unique:  idx.Unique,
			}
			md.Indexes = append(md.Indexes, index)
		}

		data := StructData{
			PackageName: packageName,
			Model:       md,
		}

		// Generate individual model file
		if err := generateModelFile(data, modelsDir); err != nil {
			return fmt.Errorf("generating model %s: %v", md.Name, err)
		}

		// Generate individual repository file
		if err := generateRepositoryFile(data, repoDir); err != nil {
			return fmt.Errorf("generating repository %s: %v", md.Name, err)
		}
	}

	// Generate main models file with imports
	return generateMainModelsFile(models, modelsDir)
}

func generateModelFile(data StructData, modelsDir string) error {
	const modelTemplate = `package {{.PackageName}}

import (
	"time"
)

// {{.Model.Name}} represents the {{.Model.TableName}} table
type {{.Model.Name}} struct {
{{range .Model.Fields}}	{{.Name}} {{.Type}} {{.Tags}} {{.Comment}}
{{end}}
	CreatedAt time.Time  ` + "`json:\"created_at\"`" + `
	UpdatedAt time.Time  ` + "`json:\"updated_at\"`" + `
}

// TableName returns the table name for {{.Model.Name}}
func ({{.Model.Name}}) TableName() string {
	return "{{.Model.TableName}}"
}

{{if .Model.Relations}}
// Relations
{{range .Model.Relations}}
{{if .IsMany}}
// {{.Name}} returns the related {{.TargetType}} records
func (m *{{$.Model.Name}}) {{.Name}}(db *DB) ([]{{.TargetType}}, error) {
	var {{.Name}} []{{.TargetType}}
	err := db.Where("{{.FieldName}} = ?", m.ID).Find(&{{.Name}}).Error
	return {{.Name}}, err
}
{{else}}
// {{.Name}} returns the related {{.TargetType}} record
func (m *{{$.Model.Name}}) {{.Name}}(db *DB) (*{{.TargetType}}, error) {
	var {{.Name}} {{.TargetType}}
	err := db.Where("id = ?", m.{{.FieldName}}).First(&{{.Name}}).Error
	if err != nil {
		return nil, err
	}
	return &{{.Name}}, nil
}
{{end}}
{{end}}
{{end}}

{{if .Model.Indexes}}
// Indexes
{{range .Model.Indexes}}
// Index: {{.Name}} {{if .Unique}}(Unique){{end}}
// Columns: {{join .Model.Indexes.Columns ", "}}
{{end}}
{{end}}
`

	tmpl, err := template.New("model").Funcs(template.FuncMap{
		"join": strings.Join,
	}).Parse(modelTemplate)
	if err != nil {
		return fmt.Errorf("parsing model template: %v", err)
	}

	outputFile := filepath.Join(modelsDir, strings.ToLower(data.Model.Name)+".go")
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("creating model file: %v", err)
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}

func generateRepositoryFile(data StructData, repoDir string) error {
	const repositoryTemplate = `package {{.PackageName}}

// {{.Model.Name}}Repository provides database operations for {{.Model.Name}}
type {{.Model.Name}}Repository struct {
	db *DB
}

// New{{.Model.Name}}Repository creates a new {{.Model.Name}}Repository
func New{{.Model.Name}}Repository(db *DB) *{{.Model.Name}}Repository {
	return &{{.Model.Name}}Repository{db: db}
}

// Create creates a new {{.Model.Name}}
func (r *{{.Model.Name}}Repository) Create({{.Model.Name | lower}} *{{.Model.Name}}) error {
	return r.db.Create({{.Model.Name | lower}}).Error
}

// FindByID finds a {{.Model.Name}} by ID
func (r *{{.Model.Name}}Repository) FindByID(id int) (*{{.Model.Name}}, error) {
	var {{.Model.Name | lower}} {{.Model.Name}}
	err := r.db.Where("id = ?", id).First(&{{.Model.Name | lower}}).Error
	if err != nil {
		return nil, err
	}
	return &{{.Model.Name | lower}}, nil
}

// FindAll finds all {{.Model.Name}} records
func (r *{{.Model.Name}}Repository) FindAll() ([]{{.Model.Name}}, error) {
	var {{.Model.Name | lower}}s []{{.Model.Name}}
	err := r.db.Find(&{{.Model.Name | lower}}s).Error
	return {{.Model.Name | lower}}s, err
}

// Update updates a {{.Model.Name}}
func (r *{{.Model.Name}}Repository) Update({{.Model.Name | lower}} *{{.Model.Name}}) error {
	return r.db.Save({{.Model.Name | lower}}).Error
}

// Delete deletes a {{.Model.Name}} by ID
func (r *{{.Model.Name}}Repository) Delete(id int) error {
	return r.db.Delete(&{{.Model.Name}}{}, id).Error
}

{{range .Model.Fields}}
{{if eq .Name "Email"}}
// FindByEmail finds a {{$.Model.Name}} by email
func (r *{{$.Model.Name}}Repository) FindByEmail(email string) (*{{$.Model.Name}}, error) {
	var {{$.Model.Name | lower}} {{$.Model.Name}}
	err := r.db.Where("email = ?", email).First(&{{$.Model.Name | lower}}).Error
	if err != nil {
		return nil, err
	}
	return &{{$.Model.Name | lower}}, nil
}
{{end}}
{{end}}
`

	tmpl, err := template.New("repository").Funcs(template.FuncMap{
		"join": strings.Join,
		"lower": strings.ToLower,
	}).Parse(repositoryTemplate)
	if err != nil {
		return fmt.Errorf("parsing repository template: %v", err)
	}

	outputFile := filepath.Join(repoDir, strings.ToLower(data.Model.Name)+"_repository.go")
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("creating repository file: %v", err)
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}

func generateMainModelsFile(models []schema.Model, modelsDir string) error {
	const mainTemplate = `package {{.PackageName}}

// This file contains imports for all generated models
// Add your custom DB interface here

// DB interface for database operations
type DB interface {
	Create(value interface{}) *DB
	Where(query interface{}, args ...interface{}) *DB
	Find(dest interface{}) *DB
	First(dest interface{}) *DB
	Save(value interface{}) *DB
	Delete(value interface{}) *DB
	Error() error
}

// Example usage:
// type PostgresDB struct {
//     db *sql.DB
// }
// 
// Implement the DB interface methods for your database driver
`
	
	// Get all model names for imports
	var modelNames []string
	for _, model := range models {
		modelNames = append(modelNames, toPascalCase(model.TableName))
	}

	data := struct {
		PackageName string
		Models      []string
	}{
		PackageName: packageName,
		Models:      modelNames,
	}

	tmpl, err := template.New("main").Parse(mainTemplate)
	if err != nil {
		return fmt.Errorf("parsing main template: %v", err)
	}

	outputFile := filepath.Join(modelsDir, "db.go")
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("creating main file: %v", err)
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}

func mapColumnTypeToGoType(dbType string) string {
	switch strings.ToLower(dbType) {
	case "serial", "bigserial", "integer", "int", "int4":
		return "int"
	case "bigint", "int8":
		return "int64"
	case "smallint", "int2":
		return "int16"
	case "text", "varchar", "character varying":
		return "string"
	case "boolean", "bool":
		return "bool"
	case "timestamp", "timestamptz":
		return "time.Time"
	case "date":
		return "time.Time"
	case "numeric", "decimal":
		return "float64"
	case "real", "float4":
		return "float32"
	case "double precision", "float8":
		return "float64"
	case "uuid":
		return "string"
	case "json", "jsonb":
		return "string" // or map[string]interface{} for more complex cases
	default:
		return "string" // fallback
	}
}

func generateTags(col schema.Column, tableName string) string {
	var tags []string
	
	// Database tag
	dbTag := fmt.Sprintf("db:\"%s\"", col.Name)
	tags = append(tags, dbTag)
	
	// JSON tag
	jsonTag := fmt.Sprintf("json:\"%s\"", toSnakeCase(col.Name))
	tags = append(tags, jsonTag)
	
	// Custom tags for future ORM
	var customTags []string
	if col.Primary {
		customTags = append(customTags, "primary")
	}
	if col.Unique {
		customTags = append(customTags, "unique")
	}
	if col.ForeignKey != nil {
		customTags = append(customTags, fmt.Sprintf("foreignKey:%s", toPascalCase(col.ForeignKey.ReferencesColumn)))
	}
	if col.Index != nil {
		if col.Index.Unique {
			customTags = append(customTags, "uniqueIndex")
		} else {
			customTags = append(customTags, "index")
		}
	}
	
	if len(customTags) > 0 {
		customTag := fmt.Sprintf("migrato:\"%s\"", strings.Join(customTags, ";"))
		tags = append(tags, customTag)
	}
	
	return fmt.Sprintf("`%s`", strings.Join(tags, " "))
}

func toPascalCase(s string) string {
	// Convert snake_case to PascalCase
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, "")
}

func toSnakeCase(s string) string {
	// Convert PascalCase to snake_case
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
} 