package schema

import (
	"fmt"
	"reflect"
	"strings"
)

func LoadModels() ([]Model, error) {
	// List all your schema structs here manually
	// Later you can automate scanning
	models := []interface{}{
	}

	var result []Model

	for _, m := range models {
		t := reflect.TypeOf(m)
		model := Model{
			TableName: t.Name(),
			Columns:   []Column{},
		}

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			tag := field.Tag.Get("db")
			if tag == "" {
				continue
			}
			col, err := parseDBTag(field.Name, tag)
			if err != nil {
				return nil, fmt.Errorf("error parsing tag on %s.%s: %v", t.Name(), field.Name, err)
			}
			model.Columns = append(model.Columns, col)
		}

		result = append(result, model)
	}

	return result, nil
}

func parseDBTag(fieldName, tag string) (Column, error) {
	parts := strings.Split(tag, ",")
	col := Column{
		Name: fieldName,
	}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "primary" {
			col.Primary = true
		} else if part == "unique" {
			col.Unique = true
		} else if strings.HasPrefix(part, "type:") {
			col.Type = strings.TrimPrefix(part, "type:")
		} else if strings.HasPrefix(part, "default:") {
			val := strings.TrimPrefix(part, "default:")
			col.Default = &val
		} else {
			// assume it's the column name
			col.Name = part
		}
	}
	return col, nil
}
