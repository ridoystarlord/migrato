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
var dryRunGenerate bool

func init() {
	generateCmd.Flags().StringVarP(&schemaFile, "file", "f", "schema.yaml", "Schema YAML file to load")
	generateCmd.Flags().BoolVar(&dryRunGenerate, "dry-run", false, "Preview the SQL that would be generated without writing files")
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

		rollbackSqls, err := generator.GenerateRollbackSQL(ops)
		if err != nil {
			fmt.Println("❌ Generating rollback SQL:", err)
			os.Exit(1)
		}

		if dryRunGenerate {
			fmt.Println("\n================ DRY RUN: Migration Preview ================")
			fmt.Println("-- Up Migration SQL --")
			for _, stmt := range sqls {
				fmt.Println(stmt)
			}
			fmt.Println("\n-- Down Migration (Rollback) SQL --")
			for _, stmt := range rollbackSqls {
				fmt.Println(stmt)
			}
			fmt.Println("============================================================")
			fmt.Println("(Dry run only. No files were written.)")
			return
		}

		filename, err := generator.WriteMigrationFile(sqls, rollbackSqls)
		if err != nil {
			fmt.Println("❌ Writing migration file:", err)
			os.Exit(1)
		}

		fmt.Println("✅ Migration generated:", filename)
	},
}
