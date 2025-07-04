package cmd

import (
	"fmt"
	"os"

	"github.com/ridoystarlord/migrato/diff"
	"github.com/ridoystarlord/migrato/generator"
	"github.com/ridoystarlord/migrato/introspect"
	"github.com/ridoystarlord/migrato/loader"
	"github.com/spf13/cobra"
)

var schemaFile string

func init() {
	generateCmd.Flags().StringVarP(&schemaFile, "file", "f", "schema.yaml", "Schema YAML file to load")
}


var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate migration file from schema",
	Run: func(cmd *cobra.Command, args []string) {

		models, err := loader.LoadModelsFromYAML(schemaFile)
		if err != nil {
			fmt.Println("❌ Loading schema.yaml:", err)
			os.Exit(1)
		}

		existing, err := introspect.IntrospectDatabase()
		if err != nil {
			fmt.Println("❌ Introspecting database:", err)
			os.Exit(1)
		}

		ops := diff.DiffSchemas(models, existing)
		if len(ops) == 0 {
			fmt.Println("✅ No changes detected.")
			return
		}

		sqls, err := generator.GenerateSQL(ops)
		if err != nil {
			fmt.Println("❌ Generating SQL:", err)
			os.Exit(1)
		}

		filename, err := generator.WriteMigrationFile(sqls)
		if err != nil {
			fmt.Println("❌ Writing migration file:", err)
			os.Exit(1)
		}

		fmt.Println("✅ Migration generated:", filename)
	},
}
