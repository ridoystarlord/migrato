package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ridoystarlord/migrato/loader"
	"github.com/ridoystarlord/migrato/schema"
)

var (
	docsFormat string
	docsOutput  string
	docsFile    string
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate documentation from schema",
	Long: `Generate ERD diagrams and API documentation from your schema.yaml.

Supported formats:
  - plantuml: PlantUML ERD diagram
  - mermaid: Mermaid ERD diagram  
  - graphviz: Graphviz DOT format
  - api: REST API documentation

Examples:
  migrato docs --format plantuml --output erd.puml
  migrato docs --format mermaid --output erd.md
  migrato docs --format api --output api.md
  migrato docs --format all --output docs/
`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load schema
		schemaFilePath := docsFile
		if schemaFilePath == "" {
			schemaFilePath = "schema.yaml"
		}

		models, err := loader.LoadModelsFromYAML(schemaFilePath)
		if err != nil {
			fmt.Printf("❌ Error loading schema: %v\n", err)
			os.Exit(1)
		}

		if len(models) == 0 {
			fmt.Println("❌ No tables found in schema")
			os.Exit(1)
		}

		// Create output directory if needed
		if docsFormat == "all" {
			if err := os.MkdirAll(docsOutput, 0755); err != nil {
				fmt.Printf("❌ Error creating output directory: %v\n", err)
				os.Exit(1)
			}
		}

		switch docsFormat {
		case "plantuml":
			generatePlantUML(models)
		case "mermaid":
			generateMermaid(models)
		case "graphviz":
			generateGraphviz(models)
		case "api":
			generateAPIDocs(models)
		case "all":
			generateAllFormats(models)
		default:
			fmt.Printf("❌ Unsupported format: %s\n", docsFormat)
			fmt.Println("Supported formats: plantuml, mermaid, graphviz, api, all")
			os.Exit(1)
		}

		fmt.Println("✅ Documentation generated successfully!")
	},
}

func generatePlantUML(models []schema.Model) {
	output := docsOutput
	if output == "" {
		output = "erd.puml"
	}

	content := generatePlantUMLContent(models)
	
	if err := os.WriteFile(output, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Error writing PlantUML file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ PlantUML ERD saved to: %s\n", output)
}

func generatePlantUMLContent(models []schema.Model) string {
	var content strings.Builder
	
	content.WriteString("@startuml\n")
	content.WriteString("!theme plain\n")
	content.WriteString("skinparam linetype ortho\n\n")

	// Generate entities
	for _, model := range models {
		content.WriteString(fmt.Sprintf("entity \"%s\" {\n", model.TableName))
		
		for _, col := range model.Columns {
			// Determine column type display
			displayType := col.Type
			if col.Type == "serial" {
				displayType = "INTEGER"
			} else if col.Type == "integer" {
				displayType = "INTEGER"
			} else if col.Type == "text" {
				displayType = "TEXT"
			} else if col.Type == "timestamp" {
				displayType = "TIMESTAMP"
			} else if col.Type == "boolean" {
				displayType = "BOOLEAN"
			}

			// Build column line
			line := fmt.Sprintf("  %s : %s", col.Name, displayType)
			
			if col.Primary {
				line += " <<PK>>"
			}
			if col.Unique {
				line += " <<UQ>>"
			}
			if col.NotNull {
				line += " <<NN>>"
			}
			if col.Default != nil {
				line += fmt.Sprintf(" <<DEFAULT: %s>>", *col.Default)
			}
			
			content.WriteString(line + "\n")
		}
		content.WriteString("}\n\n")
	}

	// Generate relationships
	for _, model := range models {
		for _, col := range model.Columns {
			if col.ForeignKey != nil {
				content.WriteString(fmt.Sprintf("\"%s\" ||--o{ \"%s\" : \"%s\"\n", 
					col.ForeignKey.ReferencesTable, 
					model.TableName, 
					col.Name))
			}
		}
	}

	content.WriteString("@enduml\n")
	return content.String()
}

func generateMermaid(models []schema.Model) {
	output := docsOutput
	if output == "" {
		output = "erd.md"
	}

	content := generateMermaidContent(models)
	
	if err := os.WriteFile(output, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Error writing Mermaid file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Mermaid ERD saved to: %s\n", output)
}

func generateMermaidContent(models []schema.Model) string {
	var content strings.Builder
	
	content.WriteString("# Database Schema ERD\n\n")
	content.WriteString("```mermaid\nerDiagram\n")

	// Generate entities
	for _, model := range models {
		content.WriteString(fmt.Sprintf("    %s {\n", model.TableName))
		
		for _, col := range model.Columns {
			// Determine column type display
			displayType := col.Type
			if col.Type == "serial" {
				displayType = "INTEGER"
			} else if col.Type == "integer" {
				displayType = "INTEGER"
			} else if col.Type == "text" {
				displayType = "TEXT"
			} else if col.Type == "timestamp" {
				displayType = "TIMESTAMP"
			} else if col.Type == "boolean" {
				displayType = "BOOLEAN"
			}

			// Build column line
			line := fmt.Sprintf("        %s %s", displayType, col.Name)
			
			if col.Primary {
				line += " PK"
			}
			if col.Unique {
				line += " UQ"
			}
			if col.NotNull {
				line += " NN"
			}
			if col.Default != nil {
				line += fmt.Sprintf(" \"%s\"", *col.Default)
			}
			
			content.WriteString(line + "\n")
		}
		content.WriteString("    }\n")
	}

	// Generate relationships
	for _, model := range models {
		for _, col := range model.Columns {
			if col.ForeignKey != nil {
				content.WriteString(fmt.Sprintf("    %s ||--o{ %s : %s\n", 
					col.ForeignKey.ReferencesTable, 
					model.TableName, 
					col.Name))
			}
		}
	}

	content.WriteString("```\n")
	return content.String()
}

func generateGraphviz(models []schema.Model) {
	output := docsOutput
	if output == "" {
		output = "erd.dot"
	}

	content := generateGraphvizContent(models)
	
	if err := os.WriteFile(output, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Error writing Graphviz file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Graphviz ERD saved to: %s\n", output)
}

func generateGraphvizContent(models []schema.Model) string {
	var content strings.Builder
	
	content.WriteString("digraph ERD {\n")
	content.WriteString("  rankdir=LR;\n")
	content.WriteString("  node [shape=record];\n\n")

	// Generate entities
	for _, model := range models {
		content.WriteString(fmt.Sprintf("  %s [label=\"%s|", model.TableName, model.TableName))
		
		var columns []string
		for _, col := range model.Columns {
			displayType := col.Type
			if col.Type == "serial" {
				displayType = "INTEGER"
			} else if col.Type == "integer" {
				displayType = "INTEGER"
			} else if col.Type == "text" {
				displayType = "TEXT"
			} else if col.Type == "timestamp" {
				displayType = "TIMESTAMP"
			} else if col.Type == "boolean" {
				displayType = "BOOLEAN"
			}

			line := fmt.Sprintf("%s: %s", col.Name, displayType)
			
			if col.Primary {
				line += " (PK)"
			}
			if col.Unique {
				line += " (UQ)"
			}
			if col.NotNull {
				line += " (NN)"
			}
			if col.Default != nil {
				line += fmt.Sprintf(" (DEFAULT: %s)", *col.Default)
			}
			
			columns = append(columns, line)
		}
		
		content.WriteString(strings.Join(columns, "\\l"))
		content.WriteString("\"];\n")
	}

	// Generate relationships
	for _, model := range models {
		for _, col := range model.Columns {
			if col.ForeignKey != nil {
				content.WriteString(fmt.Sprintf("  %s -> %s [label=\"%s\"];\n", 
					col.ForeignKey.ReferencesTable, 
					model.TableName, 
					col.Name))
			}
		}
	}

	content.WriteString("}\n")
	return content.String()
}

func generateAPIDocs(models []schema.Model) {
	output := docsOutput
	if output == "" {
		output = "api.md"
	}

	content := generateAPIDocsContent(models)
	
	if err := os.WriteFile(output, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Error writing API docs file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ API documentation saved to: %s\n", output)
}

func generateAPIDocsContent(models []schema.Model) string {
	var content strings.Builder
	
	content.WriteString("# REST API Documentation\n\n")
	content.WriteString("This document describes the REST API endpoints generated from the database schema.\n\n")

	// Generate endpoints for each model
	for _, model := range models {
		tableName := model.TableName
		resourceName := strings.TrimSuffix(tableName, "s") // Simple pluralization
		if !strings.HasSuffix(tableName, "s") {
			resourceName = tableName + "s"
		}

		content.WriteString(fmt.Sprintf("## %s\n\n", strings.Title(resourceName)))
		
		// List endpoint
		content.WriteString(fmt.Sprintf("### GET /%s\n\n", resourceName))
		content.WriteString("Retrieve all records.\n\n")
		content.WriteString("**Response:**\n")
		content.WriteString("```json\n")
		content.WriteString("[\n")
		content.WriteString("  {\n")
		for i, col := range model.Columns {
			content.WriteString(fmt.Sprintf("    \"%s\": %s", col.Name, getJSONExample(col)))
			if i < len(model.Columns)-1 {
				content.WriteString(",")
			}
			content.WriteString("\n")
		}
		content.WriteString("  }\n")
		content.WriteString("]\n")
		content.WriteString("```\n\n")

		// Get by ID endpoint
		content.WriteString(fmt.Sprintf("### GET /%s/{id}\n\n", resourceName))
		content.WriteString("Retrieve a specific record by ID.\n\n")
		content.WriteString("**Response:**\n")
		content.WriteString("```json\n")
		content.WriteString("{\n")
		for i, col := range model.Columns {
			content.WriteString(fmt.Sprintf("  \"%s\": %s", col.Name, getJSONExample(col)))
			if i < len(model.Columns)-1 {
				content.WriteString(",")
			}
			content.WriteString("\n")
		}
		content.WriteString("}\n")
		content.WriteString("```\n\n")

		// Create endpoint
		content.WriteString(fmt.Sprintf("### POST /%s\n\n", resourceName))
		content.WriteString("Create a new record.\n\n")
		content.WriteString("**Request Body:**\n")
		content.WriteString("```json\n")
		content.WriteString("{\n")
		requiredFields := []string{}
		for i, col := range model.Columns {
			if !col.Primary && col.NotNull {
				requiredFields = append(requiredFields, col.Name)
				content.WriteString(fmt.Sprintf("  \"%s\": %s", col.Name, getJSONExample(col)))
				if i < len(model.Columns)-1 {
					content.WriteString(",")
				}
				content.WriteString("\n")
			}
		}
		content.WriteString("}\n")
		content.WriteString("```\n\n")
		
		// Add required fields note if any
		if len(requiredFields) > 0 {
			content.WriteString("**Required Fields:** ")
			content.WriteString(strings.Join(requiredFields, ", "))
			content.WriteString("\n\n")
		}

		// Update endpoint
		content.WriteString(fmt.Sprintf("### PUT /%s/{id}\n\n", resourceName))
		content.WriteString("Update an existing record.\n\n")
		content.WriteString("**Request Body:**\n")
		content.WriteString("```json\n")
		content.WriteString("{\n")
		for i, col := range model.Columns {
			if !col.Primary {
				content.WriteString(fmt.Sprintf("  \"%s\": %s", col.Name, getJSONExample(col)))
				if i < len(model.Columns)-1 {
					content.WriteString(",")
				}
				content.WriteString("\n")
			}
		}
		content.WriteString("}\n")
		content.WriteString("```\n\n")

		// Delete endpoint
		content.WriteString(fmt.Sprintf("### DELETE /%s/{id}\n\n", resourceName))
		content.WriteString("Delete a record.\n\n")
		content.WriteString("**Response:** 204 No Content\n\n")

		content.WriteString("---\n\n")
	}

	return content.String()
}

func getJSONExample(col schema.Column) string {
	switch col.Type {
	case "serial", "integer":
		return "1"
	case "text":
		return "\"example\""
	case "boolean":
		return "true"
	case "timestamp":
		return "\"2024-01-01T00:00:00Z\""
	default:
		return "\"value\""
	}
}

func generateAllFormats(models []schema.Model) {
	// Generate PlantUML
	plantUMLPath := filepath.Join(docsOutput, "erd.puml")
	content := generatePlantUMLContent(models)
	if err := os.WriteFile(plantUMLPath, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Error writing PlantUML file: %v\n", err)
		os.Exit(1)
	}

	// Generate Mermaid
	mermaidPath := filepath.Join(docsOutput, "erd.md")
	content = generateMermaidContent(models)
	if err := os.WriteFile(mermaidPath, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Error writing Mermaid file: %v\n", err)
		os.Exit(1)
	}

	// Generate Graphviz
	graphvizPath := filepath.Join(docsOutput, "erd.dot")
	content = generateGraphvizContent(models)
	if err := os.WriteFile(graphvizPath, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Error writing Graphviz file: %v\n", err)
		os.Exit(1)
	}

	// Generate API docs
	apiPath := filepath.Join(docsOutput, "api.md")
	content = generateAPIDocsContent(models)
	if err := os.WriteFile(apiPath, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Error writing API docs file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ All documentation generated in: %s/\n", docsOutput)
	fmt.Printf("  - PlantUML: %s\n", plantUMLPath)
	fmt.Printf("  - Mermaid: %s\n", mermaidPath)
	fmt.Printf("  - Graphviz: %s\n", graphvizPath)
	fmt.Printf("  - API Docs: %s\n", apiPath)
}

func init() {
	docsCmd.Flags().StringVarP(&docsFormat, "format", "f", "plantuml", "Output format (plantuml, mermaid, graphviz, api, all)")
	docsCmd.Flags().StringVarP(&docsOutput, "output", "o", "", "Output file or directory (default: format-specific filename)")
	docsCmd.Flags().StringVarP(&docsFile, "file", "", "", "Schema file to use (default: schema.yaml)")
} 