package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create an example schema.yaml",
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat("schema.yaml"); err == nil {
			fmt.Println("❌ schema.yaml already exists!")
			return
		}
		content := `tables:
  - name: example_table
    columns:
      - name: id
        type: serial
        primary: true
      - name: name
        type: text
`
		err := os.WriteFile("schema.yaml", []byte(content), 0644)
		if err != nil {
			fmt.Println("❌ Error creating schema.yaml:", err)
			return
		}
		fmt.Println("✅ Created schema.yaml example file.")
	},
}
