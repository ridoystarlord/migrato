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
		content := `# Schema definition with examples of default values and functions
tables:
  - name: users
    columns:
      - name: id
        type: serial
        primary: true
      - name: email
        type: text
        unique: true
        index: true
      - name: name
        type: text
        index:
          name: idx_users_name
          type: btree
      - name: status
        type: text
        default: 'active'
      - name: created_at
        type: timestamp
        default: now()
        index:
          name: idx_users_created_at
          type: btree
      - name: updated_at
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
      - name: status
        type: text
        default: 'draft'
      - name: user_id
        type: integer
        foreign_key:
          references_table: users
          references_column: id
          on_delete: CASCADE
      - name: created_at
        type: timestamp
        default: now()
      - name: updated_at
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
      - name: slug
        type: text
        default: 'default-slug'

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
    indexes:
      - name: idx_post_tags_unique
        columns: [post_id, tag_id]
        unique: true

# Default Value Examples:
# - default: now()                    # Current timestamp
# - default: 'active'                 # String literal
# - default: 0                        # Numeric literal
# - default: true                     # Boolean literal
# - default: uuid_generate_v4()       # UUID function (requires extension)
# - default: CURRENT_DATE             # Current date
# - default: CURRENT_TIME             # Current time
# - default: 'default-value'          # Quoted string for values with spaces
`
		err := os.WriteFile("schema.yaml", []byte(content), 0644)
		if err != nil {
			fmt.Println("❌ Error creating schema.yaml:", err)
			return
		}
		fmt.Println("✅ Created schema.yaml example file.")
	},
}
