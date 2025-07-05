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
  - name: users
    columns:
      - name: id
        type: serial
        primary: true
      - name: email
        type: text
        unique: true
      - name: name
        type: text
      - name: created_at
        type: timestamp
        default: now()

  - name: posts
    columns:
      - name: id
        type: serial
        primary: true
      - name: title
        type: text
      - name: content
        type: text
      - name: user_id
        type: integer
        foreign_key:
          references_table: users
          references_column: id
          on_delete: CASCADE
      - name: created_at
        type: timestamp
        default: now()

  - name: tags
    columns:
      - name: id
        type: serial
        primary: true
      - name: name
        type: text
        unique: true

  - name: post_tags
    columns:
      - name: id
        type: serial
        primary: true
      - name: post_id
        type: integer
        foreign_key:
          references_table: posts
          references_column: id
          on_delete: CASCADE
      - name: tag_id
        type: integer
        foreign_key:
          references_table: tags
          references_column: id
          on_delete: CASCADE
`
		err := os.WriteFile("schema.yaml", []byte(content), 0644)
		if err != nil {
			fmt.Println("❌ Error creating schema.yaml:", err)
			return
		}
		fmt.Println("✅ Created schema.yaml example file.")
	},
}
