package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)



var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new migrato project (Go structs recommended)",
	Long: `Initialize a new migrato project with your preferred schema definition method.

Recommended: Go structs with migrato tags (--structs)
- Type-safe, IDE-friendly schema definition
- Version control friendly
- Better for complex relationships and constraints

Alternative: YAML schema file (--yaml)
- Simple, declarative schema definition
- Good for simple projects or rapid prototyping

Examples:
  migrato init                    # Initialize with Go structs (recommended)
  migrato init --yaml             # Initialize with YAML schema
  migrato init --structs          # Explicitly use Go structs`,
	Run: func(cmd *cobra.Command, args []string) {
		// Determine which approach to use (default to structs)
		if useYAML {
			// Initialize with YAML schema
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
        not_null: true
        index: true
      - name: name
        type: text
        not_null: true
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
        not_null: true
      - name: content
        type: text
        not_null: true
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
			fmt.Println("📝 Edit schema.yaml to define your database schema")
			fmt.Println("🚀 Run 'migrato generate' to create migrations from your schema")
			return
		}

		// Initialize with Go structs (default)
		if _, err := os.Stat("models"); err == nil {
			fmt.Println("❌ models directory already exists!")
			return
		}
		// Create models directory
		if err := os.MkdirAll("models", 0755); err != nil {
			fmt.Println("❌ Failed to create models directory:", err)
			return
		}

		// Create main.go file with example structs
		mainContent := `package models

import (
	"time"
	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID        int       ` + "`migrato:\"primary;type:serial\"`" + `
	Email     string    ` + "`migrato:\"unique;not_null;index\"`" + `
	Name      string    ` + "`migrato:\"not_null\"`" + `
	Status    string    ` + "`migrato:\"default:active\"`" + `
	CreatedAt time.Time ` + "`migrato:\"default:now()\"`" + `
	UpdatedAt time.Time ` + "`migrato:\"default:now()\"`" + `
}

// Post represents a blog post
type Post struct {
	ID        int       ` + "`migrato:\"primary;type:serial\"`" + `
	Title     string    ` + "`migrato:\"not_null\"`" + `
	Content   string    ` + "`migrato:\"not_null\"`" + `
	Status    string    ` + "`migrato:\"default:draft\"`" + `
	UserID    int       ` + "`migrato:\"fk:users.id:CASCADE\"`" + `
	CreatedAt time.Time ` + "`migrato:\"default:now()\"`" + `
	UpdatedAt time.Time ` + "`migrato:\"default:now()\"`" + `
}

// Tag represents a tag for categorizing posts
type Tag struct {
	ID   int    ` + "`migrato:\"primary;type:serial\"`" + `
	Name string ` + "`migrato:\"unique;not_null\"`" + `
	Slug string ` + "`migrato:\"default:default-slug\"`" + `
}

// PostTag represents the many-to-many relationship between posts and tags
type PostTag struct {
	ID     int ` + "`migrato:\"primary;type:serial\"`" + `
	PostID int ` + "`migrato:\"fk:posts.id:CASCADE\"`" + `
	TagID  int ` + "`migrato:\"fk:tags.id:CASCADE\"`" + `
}

// Product represents a product in an e-commerce system
type Product struct {
	ID          uuid.UUID ` + "`migrato:\"primary;type:uuid;default:uuid_generate_v4()\"`" + `
	Name        string    ` + "`migrato:\"not_null;index\"`" + `
	Description string    ` + "`migrato:\"type:text\"`" + `
	Price       float64   ` + "`migrato:\"type:numeric(10,2);not_null\"`" + `
	CategoryID  int       ` + "`migrato:\"fk:categories.id:RESTRICT\"`" + `
	IsActive    bool      ` + "`migrato:\"default:true\"`" + `
	CreatedAt   time.Time ` + "`migrato:\"default:now()\"`" + `
	UpdatedAt   time.Time ` + "`migrato:\"default:now()\"`" + `
}

// Category represents a product category
type Category struct {
	ID          int       ` + "`migrato:\"primary;type:serial\"`" + `
	Name        string    ` + "`migrato:\"not_null;unique\"`" + `
	Description string    ` + "`migrato:\"type:text\"`" + `
	ParentID    *int      ` + "`migrato:\"fk:categories.id:SET NULL\"`" + ` // Self-referencing foreign key
	CreatedAt   time.Time ` + "`migrato:\"default:now()\"`" + `
}
`

		mainPath := "models/main.go"
		if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
			fmt.Println("❌ Failed to create main.go:", err)
			return
		}

		// Create README.md file
		readmeContent := `# Database Models

This directory contains Go structs that define your database schema using migrato tags.

## How to use

1. **Edit the structs** in main.go to define your database tables, columns, indexes, and relationships
2. **Run migrations**: migrato generate --structs to create SQL migrations from your structs
3. **Apply migrations**: migrato migrate to apply the migrations to your database

## Schema Definition with Tags

### Basic Struct Definition

` + "```" + `go
type User struct {
    ID        int       ` + "`migrato:\"primary;type:serial\"`" + `
    Email     string    ` + "`migrato:\"unique;not_null;index\"`" + `
    Name      string    ` + "`migrato:\"not_null\"`" + `
    CreatedAt time.Time ` + "`migrato:\"default:now()\"`" + `
}
` + "```" + `

### Tag Syntax

The ` + "`migrato`" + ` tag uses a simple syntax: ` + "`migrato:\"option1;option2;key:value\"`" + `

#### Basic Options

- ` + "`primary`" + ` - Primary key
- ` + "`unique`" + ` - Unique constraint
- ` + "`not_null`" + ` - NOT NULL constraint
- ` + "`index`" + ` - Create an index on this column

#### Key-Value Options

- ` + "`type:postgres_type`" + ` - Specify PostgreSQL data type
- ` + "`default:value`" + ` - Default value
- ` + "`fk:table.column:on_delete:on_update`" + ` - Foreign key reference
- ` + "`index:name:type:unique`" + ` - Index configuration

### Column Types

Go types are automatically mapped to PostgreSQL types:

- ` + "`int`" + ` → ` + "`integer`" + `
- ` + "`int64`" + ` → ` + "`bigint`" + `
- ` + "`string`" + ` → ` + "`text`" + `
- ` + "`bool`" + ` → ` + "`boolean`" + `
- ` + "`float64`" + ` → ` + "`numeric`" + `
- ` + "`time.Time`" + ` → ` + "`timestamp`" + `
- ` + "`uuid.UUID`" + ` → ` + "`uuid`" + `

You can override the type using ` + "`type:custom_type`" + ` in the tag.

### Foreign Keys

` + "```" + `go
type Post struct {
    ID     int ` + "`migrato:\"primary;type:serial\"`" + `
    UserID int ` + "`migrato:\"fk:users.id:CASCADE\"`" + ` // References users.id with CASCADE delete
}
` + "```" + `

Foreign key format: ` + "`fk:table.column:on_delete:on_update`" + `

- ` + "`CASCADE`" + ` - Delete/update referenced records
- ` + "`SET NULL`" + ` - Set foreign key to NULL
- ` + "`RESTRICT`" + ` - Prevent delete/update if referenced

### Indexes

` + "```" + `go
type Product struct {
    Name string ` + "`migrato:\"index:idx_product_name:btree\"`" + ` // Named index
    Code string ` + "`migrato:\"index\"`" + ` // Simple index
    SKU  string ` + "`migrato:\"index:idx_sku:btree:unique\"`" + ` // Unique index
}
` + "```" + `

Index format: ` + "`index:name:type:unique`" + `

### Table Names

Table names are automatically generated from struct names:
- ` + "`User`" + ` → ` + "`users`" + `
- ` + "`Product`" + ` → ` + "`products`" + `
- ` + "`PostTag`" + ` → ` + "`post_tags`" + `

### Examples

See the examples in main.go for complete struct definitions including:
- Users with indexes and constraints
- Posts with foreign keys
- Many-to-many relationships (PostTag)
- Products with UUID primary keys
- Self-referencing foreign keys (Category)
`

		readmePath := "models/README.md"
		if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
			fmt.Println("❌ Failed to create README.md:", err)
			return
		}

		fmt.Println("✅ Models directory created successfully!")
		fmt.Println("📁 Directory: models")
		fmt.Println("📝 Edit the structs in models/main.go to define your database schema")
		fmt.Println("🚀 Run 'migrato generate --structs' to create migrations from your structs")
	},
}
