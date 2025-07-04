package loader

import (
	"fmt"
	"io/ioutil"

	"github.com/ridoystarlord/go-migration-buddy/schema"
	"gopkg.in/yaml.v3"
)

type yamlFile struct {
	Tables []yamlTable `yaml:"tables"`
}

type yamlTable struct {
	Name    string       `yaml:"name"`
	Columns []yamlColumn `yaml:"columns"`
}

type yamlColumn struct {
	Name     string  `yaml:"name"`
	Type     string  `yaml:"type"`
	Primary  bool    `yaml:"primary"`
	Unique   bool    `yaml:"unique"`
	Default  *string `yaml:"default"`
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
		for _, c := range t.Columns {
			model.Columns = append(model.Columns, schema.Column{
				Name:    c.Name,
				Type:    c.Type,
				Primary: c.Primary,
				Unique:  c.Unique,
				Default: c.Default,
			})
		}
		models = append(models, model)
	}

	return models, nil
}
