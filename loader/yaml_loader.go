package loader

import (
	"fmt"
	"io/ioutil"

	"github.com/ridoystarlord/migrato/schema"
	"gopkg.in/yaml.v3"
)

type yamlFile struct {
	Tables []yamlTable `yaml:"tables"`
}

type yamlTable struct {
	Name      string         `yaml:"name"`
	Columns   []yamlColumn   `yaml:"columns"`
	Relations []yamlRelation `yaml:"relations,omitempty"`
	Indexes   []yamlIndex    `yaml:"indexes,omitempty"`
}

type yamlColumn struct {
	Name        string         `yaml:"name"`
	Type        string         `yaml:"type"`
	Primary     bool           `yaml:"primary"`
	Unique      bool           `yaml:"unique"`
	NotNull     bool           `yaml:"not_null"`
	Default     *string        `yaml:"default"`
	ForeignKey  *yamlForeignKey `yaml:"foreign_key,omitempty"`
	Index       interface{}    `yaml:"index,omitempty"`
}

type yamlIndexConfig struct {
	Name    string   `yaml:"name,omitempty"`
	Columns []string `yaml:"columns,omitempty"`
	Unique  bool     `yaml:"unique,omitempty"`
	Type    string   `yaml:"type,omitempty"`
}

type yamlForeignKey struct {
	ReferencesTable  string `yaml:"references_table"`
	ReferencesColumn string `yaml:"references_column"`
	OnDelete         string `yaml:"on_delete,omitempty"`
	OnUpdate         string `yaml:"on_update,omitempty"`
}

type yamlRelation struct {
	Name          string                    `yaml:"name"`
	Type          string                    `yaml:"type"`
	FromTable     string                    `yaml:"from_table"`
	FromColumn    string                    `yaml:"from_column"`
	ToTable       string                    `yaml:"to_table"`
	ToColumn      string                    `yaml:"to_column"`
	JunctionTable string                    `yaml:"junction_table,omitempty"`
}

type yamlIndex struct {
	Name    string   `yaml:"name"`
	Columns []string `yaml:"columns"`
	Unique  bool     `yaml:"unique,omitempty"`
	Type    string   `yaml:"type,omitempty"`
}

func LoadModelsFromYAML(filename string) ([]schema.Model, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading schema file: %w", err)
	}

	var yf yamlFile
	if err := yaml.Unmarshal(data, &yf); err != nil {
		return nil, fmt.Errorf("unmarshalling YAML: %w", err)
	}

	var models []schema.Model
	for _, t := range yf.Tables {
		model := schema.Model{
			TableName: t.Name,
		}
		
		// Load columns
		for _, c := range t.Columns {
			column := schema.Column{
				Name:    c.Name,
				Type:    c.Type,
				Primary: c.Primary,
				Unique:  c.Unique,
				NotNull: c.NotNull,
				Default: c.Default,
			}
			
			// Handle foreign key
			if c.ForeignKey != nil {
				column.ForeignKey = &schema.ForeignKey{
					ReferencesTable:  c.ForeignKey.ReferencesTable,
					ReferencesColumn: c.ForeignKey.ReferencesColumn,
					OnDelete:         c.ForeignKey.OnDelete,
					OnUpdate:         c.ForeignKey.OnUpdate,
				}
			}

			// Handle index
			if c.Index != nil {
				switch indexValue := c.Index.(type) {
				case bool:
					if indexValue {
						// Simple boolean index - use default settings
						column.Index = &schema.IndexConfig{
							Name:    "",
							Columns: []string{c.Name},
							Unique:  false,
							Type:    "btree",
						}
					}
				case map[string]interface{}:
					// Complex index configuration
					indexConfig := &schema.IndexConfig{
						Columns: []string{c.Name}, // Default to column name
						Type:    "btree",          // Default type
					}
					
					if name, ok := indexValue["name"].(string); ok {
						indexConfig.Name = name
					}
					if columns, ok := indexValue["columns"].([]interface{}); ok {
						indexConfig.Columns = make([]string, len(columns))
						for i, col := range columns {
							if colStr, ok := col.(string); ok {
								indexConfig.Columns[i] = colStr
							}
						}
					}
					if unique, ok := indexValue["unique"].(bool); ok {
						indexConfig.Unique = unique
					}
					if indexType, ok := indexValue["type"].(string); ok {
						indexConfig.Type = indexType
					}
					
					column.Index = indexConfig
				}
			}
			
			model.Columns = append(model.Columns, column)
		}
		
		// Load relations
		for _, r := range t.Relations {
			relation := schema.Relation{
				Name:          r.Name,
				Type:          schema.RelationType(r.Type),
				FromTable:     r.FromTable,
				FromColumn:    r.FromColumn,
				ToTable:       r.ToTable,
				ToColumn:      r.ToColumn,
				JunctionTable: r.JunctionTable,
			}
			model.Relations = append(model.Relations, relation)
		}

		// Load indexes
		for _, idx := range t.Indexes {
			index := schema.Index{
				Name:    idx.Name,
				Table:   t.Name,
				Columns: idx.Columns,
				Unique:  idx.Unique,
				Type:    idx.Type,
			}
			model.Indexes = append(model.Indexes, index)
		}
		
		models = append(models, model)
	}

	return models, nil
}
