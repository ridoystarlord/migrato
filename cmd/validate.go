package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/ridoystarlord/migrato/loader"
	"github.com/ridoystarlord/migrato/schema"
	"github.com/ridoystarlord/migrato/validator"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate YAML schema against database constraints",
	Long: `Validate your YAML schema file against database constraints and best practices.

This command performs comprehensive validation including:
- Table and column naming (PostgreSQL identifier rules, reserved keywords)
- Data type compatibility (supported PostgreSQL types)
- Foreign key references (valid table/column references)
- Index definitions (valid names and column references)
- Default value compatibility (type-appropriate defaults)
- Cross-table constraints (foreign key relationships)
- Database state conflicts (when connected to database)

The validator works in two modes:
- Offline: Validates schema syntax and relationships (no database required)
- Online: Also checks against existing database state (requires DATABASE_URL)

Examples:
  migrato validate                    # Validate schema.yaml (offline)
  migrato validate --schema custom.yaml  # Validate custom schema file
  migrato validate --format json     # Output validation results as JSON
  DATABASE_URL=postgres://... migrato validate  # Online validation
`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := validateSchema(); err != nil {
			fmt.Printf("❌ Schema validation failed: %v\n", err)
			os.Exit(1)
		}
	},
}

var (
	validateSchemaFile string
	validateFormat     string
)

func init() {
	validateCmd.Flags().StringVarP(&validateSchemaFile, "schema", "s", "schema.yaml", "Schema file to validate")
	validateCmd.Flags().StringVarP(&validateFormat, "format", "f", "text", "Output format (text, json)")
}

func validateSchema() error {
	// Load schema
	models, err := loader.LoadModelsFromYAML(validateSchemaFile)
	if err != nil {
		return fmt.Errorf("failed to load schema: %v", err)
	}

	// Check for DATABASE_URL in environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Println("[DEBUG] DATABASE_URL not set, using offline schema validation.")
		return validateSchemaOffline(models)
	}

	// Only create DB validator if DATABASE_URL is set
	dbValidator, err := validator.NewSchemaValidator()
	if err != nil {
		return fmt.Errorf("failed to create schema validator: %v", err)
	}

	// Validate schema with database
	result, err := dbValidator.ValidateSchema(models)
	if err != nil {
		return fmt.Errorf("failed to validate schema: %v", err)
	}

	// Output results
	if validateFormat == "json" {
		return outputJSON(result)
	} else {
		return outputText(result)
	}
}

func validateSchemaOffline(models []schema.Model) error {
	validator := &validator.SchemaValidator{} // No DB connection
	result, err := validator.ValidateSchemaWithoutDB(models)
	if err != nil {
		return fmt.Errorf("failed to validate schema: %v", err)
	}
	if validateFormat == "json" {
		return outputJSON(result)
	} else {
		return outputText(result)
	}
}

func outputJSON(result *validator.ValidationResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func outputText(result *validator.ValidationResult) error {
	// Print summary
	if result.Valid {
		color.Green("✅ Schema validation passed!")
	} else {
		color.Red("❌ Schema validation failed!")
	}

	// Print errors
	if len(result.Errors) > 0 {
		fmt.Printf("\n🔴 Errors (%d):\n", len(result.Errors))
		for i, err := range result.Errors {
			fmt.Printf("  %d. ", i+1)
			if err.Table != "" {
				fmt.Printf("[%s]", err.Table)
			}
			if err.Column != "" {
				fmt.Printf(".%s", err.Column)
			}
			if err.Index != "" {
				fmt.Printf(" (index: %s)", err.Index)
			}
			fmt.Printf(": %s\n", err.Message)
		}
	}

	// Print warnings
	if len(result.Warnings) > 0 {
		fmt.Printf("\n🟡 Warnings (%d):\n", len(result.Warnings))
		for i, warning := range result.Warnings {
			fmt.Printf("  %d. ", i+1)
			if warning.Table != "" {
				fmt.Printf("[%s]", warning.Table)
			}
			if warning.Column != "" {
				fmt.Printf(".%s", warning.Column)
			}
			if warning.Index != "" {
				fmt.Printf(" (index: %s)", warning.Index)
			}
			fmt.Printf(": %s\n", warning.Message)
		}
	}

	// Print info
	if len(result.Info) > 0 {
		fmt.Printf("\n🔵 Info (%d):\n", len(result.Info))
		for i, info := range result.Info {
			fmt.Printf("  %d. ", i+1)
			if info.Table != "" {
				fmt.Printf("[%s]", info.Table)
			}
			if info.Column != "" {
				fmt.Printf(".%s", info.Column)
			}
			if info.Index != "" {
				fmt.Printf(" (index: %s)", info.Index)
			}
			fmt.Printf(": %s\n", info.Message)
		}
	}

	// Print summary
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("  • Errors: %d\n", len(result.Errors))
	fmt.Printf("  • Warnings: %d\n", len(result.Warnings))
	fmt.Printf("  • Info: %d\n", len(result.Info))

	if result.Valid {
		fmt.Printf("\n🎉 Your schema is valid and ready for migration generation!\n")
	} else {
		fmt.Printf("\n💡 Fix the errors above before generating migrations.\n")
	}

	return nil
} 