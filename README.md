# migrato

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A lightweight, Prisma-like migration tool for Go and PostgreSQL.

## Features

- Generate SQL migrations from a YAML schema
- Introspect your existing PostgreSQL database
- Apply and track migrations
- **Migration rollbacks** with automatic rollback SQL generation
- Support for creating, adding, and dropping tables and columns
- **Column modifications**: Change column types, add/remove NOT NULL constraints, modify default values
- **Column renaming**: Rename columns with proper rollback support
- **Index management** with support for various index types
- Foreign key relationships with configurable cascade options
- Support for one-to-many, many-to-many, and one-to-one relationships
- **Health checks**: Verify database connectivity and migration status
- **Schema validation**: Validate YAML schema against database constraints and best practices
- **Issue detection**: Find and suggest fixes for schema issues
- **Default values**: Support for literal values and functions
- **Visual schema diff**: Preview changes with color-coded tree format
- **Schema documentation**: Generate ERD diagrams and API docs (PlantUML, Mermaid, Graphviz)
- **Migration history**: Track execution times, users, and detailed migration records
- **Migration logging**: Comprehensive activity logging with timestamps and user tracking
- **Go struct-based schema**: Define database schema using Go structs with migrato tags
- **Database browser**: Web-based interface for viewing and exploring table data (like Prisma Studio)
- Simple CLI interface
- Inspired by Prisma Migrate, but for Go

## Installation

### Prebuilt Releases

Download the latest release for your platform from the [releases page](https://github.com/ridoystarlord/migrato/releases):

```sh
# For macOS (Intel)
curl -L https://github.com/ridoystarlord/migrato/releases/latest/download/migrato_darwin_amd64.tar.gz | tar -xz
sudo mv migrato /usr/local/bin/

# For macOS (Apple Silicon)
curl -L https://github.com/ridoystarlord/migrato/releases/latest/download/migrato_darwin_arm64.tar.gz | tar -xz
sudo mv migrato /usr/local/bin/

# For Linux (Intel)
curl -L https://github.com/ridoystarlord/migrato/releases/latest/download/migrato_linux_amd64.tar.gz | tar -xz
sudo mv migrato /usr/local/bin/

# For Linux (ARM64)
curl -L https://github.com/ridoystarlord/migrato/releases/latest/download/migrato_linux_arm64.tar.gz | tar -xz
sudo mv migrato /usr/local/bin/

# For Windows
# Download migrato_windows_amd64.tar.gz from the releases page and extract
```

### Go Install (latest)

```sh
go install github.com/ridoystarlord/migrato@latest
```

### From Source

```sh
git clone https://github.com/ridoystarlord/migrato.git
cd migrato
go build -o migrato ./main.go
```

## Quickstart

### Option 1: Go Structs (Recommended)

1. **Set up your database connection**

   - Set the `DATABASE_URL` environment variable (or create a `.env` file):
     ```env
     DATABASE_URL=postgres://user:password@localhost:5432/dbname?sslmode=disable
     ```

2. **Initialize the project with Go structs**

   ```sh
   migrato init
   # Creates a models/ directory with example Go structs
   ```

3. **Edit the structs in `models/main.go`** to define your database schema using Go code.

4. **Generate a migration**

   ```sh
   migrato generate --structs
   # Generates a SQL migration file from your Go structs
   ```

5. **Apply migrations**

   ```sh
   migrato migrate
   # Applies all pending migrations to your database
   ```

6. **Check migration status**

   ```sh
   migrato status
   # Shows applied and pending migrations
   ```

7. **Rollback migrations (if needed)**
   ```sh
   migrato rollback        # Rollback the last migration
   migrato rollback -s 3   # Rollback the last 3 migrations
   ```

### Option 2: YAML Schema (Alternative)

1. **Set up your database connection**

   - Set the `DATABASE_URL` environment variable (or create a `.env` file):
     ```env
     DATABASE_URL=postgres://user:password@localhost:5432/dbname?sslmode=disable
     ```

2. **Initialize with YAML schema**

   ```sh
   migrato init --yaml
   # Creates a schema.yaml file with example tables
   ```

3. **Edit `schema.yaml`** to define your tables and columns.

4. **Generate a migration**

   ```sh
   migrato generate
   # Generates a SQL migration file from your YAML schema
   ```

5. **Apply migrations**

   ```sh
   migrato migrate
   # Applies all pending migrations to your database
   ```

6. **Check migration status**

   ```sh
   migrato status
   # Shows applied and pending migrations
   ```

7. **Rollback migrations (if needed)**
   ```sh
   migrato rollback        # Rollback the last migration
   migrato rollback -s 3   # Rollback the last 3 migrations
   ```

## CLI Commands

### Schema Management

- `migrato init` — Initialize a new project (Go structs recommended)
  - `--structs` — Initialize with Go structs (default)
  - `--yaml` — Initialize with YAML schema file

### Migration Generation

- `migrato generate` — Generate a migration file from your schema

  - `-f, --file` — Specify a custom schema YAML file (default: `schema.yaml`)
  - `--structs` — Use Go structs instead of YAML schema
  - `-m, --models` — Models directory to load structs from (default: `models`)

  - `-f, --file` — Specify a custom schema YAML file (default: `schema.yaml`)
  - `-o, --output` — Output directory for generated structs (default: `models`)
  - `-p, --package` — Package name for generated structs (default: `models`)

- `migrato migrate` — Apply all pending migrations
- `migrato rollback` — Rollback migrations
  - `-s, --steps` — Number of migrations to rollback (default: 1)
- `migrato status` — Show applied and pending migrations
- `migrato health` — Check database connectivity
  - `-t, --timeout` — Timeout for health check (default: 5s)
- `migrato validate` — Validate YAML schema against database constraints
  - `-s, --schema` — Specify a custom schema YAML file (default: `schema.yaml`)
  - `-f, --format` — Output format (text, json) (default: text)
- `migrato check` — Check for potential issues
  - `-f, --fix-suggestions` — Show suggestions for fixing issues
- `migrato diff` — Show differences between schema and database
  - `-v, --visual` — Show changes in visual tree format with colors
  - `-f, --file` — Specify a custom schema YAML file (default: `schema.yaml`)
  - `--structs` — Use Go structs instead of YAML schema
  - `-m, --models` — Models directory to use (default: `models`)
- `migrato docs` — Generate documentation from schema
  - `-f, --format` — Output format (plantuml, mermaid, graphviz, api, all)
  - `-o, --output` — Output file or directory (default: format-specific filename)
  - `--file` — Schema file to use (default: `schema.yaml`)
- `migrato history` — Show detailed migration history
  - `-l, --limit` — Limit number of records to show (0 = all)
  - `-t, --table` — Filter by table name
  - `-d, --detailed` — Show detailed information
- `migrato log` — Show recent migration activities
  - `-l, --limit` — Limit number of log entries to show (default: 50)
  - `-f, --follow` — Follow logs in real-time (future feature)
- `migrato studio` — Launch web-based database browser with inline editing
  - `--port` — Port to run the web server on (default: 8080)

## Schema Example

```yaml
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
      - name: created_at
        type: timestamp
        default: now()
        index:
          name: idx_users_created_at
          type: btree

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
    indexes:
      - name: idx_post_tags_unique
        columns: [post_id, tag_id]
        unique: true
```

### Relationship Types

The tool supports different types of relationships:

1. **One-to-Many**: A user can have many posts (user_id in posts table)
2. **Many-to-Many**: Posts can have many tags and tags can have many posts (via post_tags junction table)
3. **One-to-One**: Can be implemented with a unique foreign key

### Foreign Key Options

- `references_table`: The table being referenced
- `references_column`: The column being referenced (usually 'id')
- `on_delete`: Action when referenced record is deleted (CASCADE, SET NULL, RESTRICT)
- `on_update`: Action when referenced record is updated (CASCADE, SET NULL, RESTRICT)

### Index Management

The tool supports both column-level and table-level indexes:

#### Column-Level Indexes

```yaml
columns:
  - name: email
    type: text
    index: true # Simple index on the column

  - name: name
    type: text
    index:
      name: idx_users_name
      type: btree
      unique: false
```

#### Table-Level Indexes

```yaml
tables:
  - name: post_tags
    columns:
      # ... columns
    indexes:
      - name: idx_post_tags_unique
        columns: [post_id, tag_id]
        unique: true
        type: btree
```

#### Index Options

- `name`: Custom index name (auto-generated if not provided)
- `columns`: Array of column names for composite indexes
- `unique`: Whether the index enforces uniqueness
- `type`: Index type (btree, hash, gin, gist, etc.)

## Go Structs Schema (Recommended)

Instead of YAML, you can define your database schema using Go structs. This provides better type safety, IDE support, and more flexibility.

### Creating Models

```sh
migrato init
# Creates a models/ directory with example structs
```

### Schema Definition with Tags

```go
package models

import (
	"time"
	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID        int       `migrato:"primary;type:serial"`
	Email     string    `migrato:"unique;not_null;index"`
	Name      string    `migrato:"not_null"`
	CreatedAt time.Time `migrato:"default:now()"`
}

// Post represents a blog post
type Post struct {
	ID        int       `migrato:"primary;type:serial"`
	Title     string    `migrato:"not_null"`
	Content   string    `migrato:"not_null"`
	UserID    int       `migrato:"fk:users.id:CASCADE"`
	CreatedAt time.Time `migrato:"default:now()"`
}

// Product represents a product in an e-commerce system
type Product struct {
	ID          uuid.UUID `migrato:"primary;type:uuid;default:uuid_generate_v4()"`
	Name        string    `migrato:"not_null;index"`
	Description string    `migrato:"type:text"`
	Price       float64   `migrato:"type:numeric(10,2);not_null"`
	CategoryID  int       `migrato:"fk:categories.id:RESTRICT"`
	IsActive    bool      `migrato:"default:true"`
	CreatedAt   time.Time `migrato:"default:now()"`
}
```

### Tag Syntax

The `migrato` tag uses a simple syntax: `migrato:"option1;option2;key:value"`

#### Basic Options

- `primary` - Primary key
- `unique` - Unique constraint
- `not_null` - NOT NULL constraint
- `index` - Create an index on this column

#### Key-Value Options

- `type:postgres_type` - Specify PostgreSQL data type
- `default:value` - Default value
- `fk:table.column:on_delete:on_update` - Foreign key reference
- `index:name:type:unique` - Index configuration

### Type Mapping

Go types are automatically mapped to PostgreSQL types:

- `int` → `integer`
- `int64` → `bigint`
- `string` → `text`
- `bool` → `boolean`
- `float64` → `numeric`
- `time.Time` → `timestamp`
- `uuid.UUID` → `uuid`

### Generating Migrations from Structs

```sh
migrato generate --structs
# Generates migrations from your Go structs
```

### Advantages of Tag-Based Structs

- **Type Safety**: Compile-time checking of schema definitions
- **IDE Support**: Auto-completion, refactoring, and error detection
- **Version Control**: Better diff tracking and merge conflict resolution
- **Reusability**: Can be imported and used in your application code
- **Validation**: Can add custom validation logic
- **Documentation**: Self-documenting code with comments
- **Familiar**: Similar to GORM and other Go ORMs

### Default Values

You can specify default values for columns using the `default` property:

```yaml
columns:
  - name: status
    type: text
    default: "active" # String literal

  - name: created_at
    type: timestamp
    default: now() # Database function

  - name: count
    type: integer
    default: 0 # Numeric literal

  - name: is_active
    type: boolean
    default: true # Boolean literal

  - name: user_id
    type: uuid
    default: uuid_generate_v4() # UUID function (requires extension)
```

#### Supported Default Value Types

- **String literals**: `'active'`, `'default-value'`
- **Numeric literals**: `0`, `1`, `42`
- **Boolean literals**: `true`, `false`
- **Database functions**: `now()`, `CURRENT_DATE`, `CURRENT_TIME`
- **Custom functions**: `uuid_generate_v4()`, `gen_random_uuid()`

> **Note**: For functions like `uuid_generate_v4()`, you may need to install the `uuid-ossp` extension in PostgreSQL first.

### Column Modifications

The tool supports modifying existing columns with automatic migration generation:

#### Changing Column Types

```yaml
# Before
columns:
  - name: age
    type: integer

# After
columns:
  - name: age
    type: bigint
```

#### Adding/Removing NOT NULL Constraints

```yaml
# Before
columns:
  - name: email
    type: text

# After - Add NOT NULL constraint
columns:
  - name: email
    type: text
    not_null: true

# After - Remove NOT NULL constraint
columns:
  - name: email
    type: text
    not_null: false
```

#### Modifying Default Values

```yaml
# Before
columns:
  - name: status
    type: text
    default: 'active'

# After
columns:
  - name: status
    type: text
    default: 'pending'
```

#### Column Renaming

Column renaming is supported through the diff detection:

```yaml
# Before
columns:
  - name: user_name
    type: text

# After
columns:
  - name: full_name
    type: text
```

> **Note**: Column modifications are detected automatically when you run `migrato generate`. The tool compares your schema with the existing database and generates the appropriate ALTER TABLE statements.

### Schema Validation

Validate your YAML schema against database constraints and best practices before generating migrations:

```sh
migrato validate                    # Validate schema.yaml
migrato validate --schema custom.yaml  # Validate custom schema file
migrato validate --format json      # Output validation results as JSON
```

#### Validation Features

The schema validator checks:

- **Table and column naming**: Valid PostgreSQL identifiers, reserved keyword checks
- **Data type compatibility**: Supported PostgreSQL data types
- **Foreign key references**: Valid table and column references
- **Index definitions**: Valid index names and column references
- **Default value compatibility**: Type-appropriate default values
- **Cross-table constraints**: Foreign key relationship validation
- **Database state**: Existing table conflicts (when connected to database)

#### Validation Output Example

```sh
✅ Schema validation passed!

📊 Summary:
  • Errors: 0
  • Warnings: 0
  • Info: 0

🎉 Your schema is valid and ready for migration generation!
```

#### Offline vs Online Validation

- **Offline validation** (default): Works without database connection, validates schema syntax and relationships
- **Online validation**: When `DATABASE_URL` is set, also checks against existing database state

#### JSON Output Format

```sh
migrato validate --format json
```

```json
{
  "valid": true,
  "errors": [],
  "warnings": [
    {
      "type": "no_primary_key",
      "table": "posts",
      "message": "Table 'posts' has no primary key defined",
      "severity": "warning"
    }
  ],
  "info": [
    {
      "type": "table_exists",
      "table": "users",
      "message": "Table 'users' already exists in database",
      "severity": "info"
    }
  ]
}
```

### Visual Schema Diff

```sh
migrato diff                    # Show differences in text format
migrato diff --visual          # Show differences in tree format with colors
```

#### Visual Diff Output Example

```
🌳 Schema Changes (Visual Diff)
==================================================

📋 Tables:
  ➕ CREATE users
  ⚡ MODIFY posts

📝 Columns:
  📋 users:
    ➕ ADD email (text) NOT NULL
    ➕ ADD name (text) NOT NULL DEFAULT 'John Doe'

  📋 posts:
    🔄 MODIFY title:
      📊 TYPE: varchar → text
      🚫 NOT NULL: ADDED

🔍 Indexes:
  📋 users:
    ➕ CREATE INDEX idx_users_email

🔗 Foreign Keys:
  📋 posts:
    ➕ ADD FK user_id → users.id
```

The visual diff uses color coding:

- 🟢 **Green**: Additions (new tables, columns, indexes, foreign keys)
- 🔴 **Red**: Deletions (dropped tables, columns, indexes, foreign keys)
- 🔵 **Blue**: Modifications (column type changes, constraint changes)
- 🟡 **Yellow**: Tables with modifications

### Schema Documentation Generation

Generate comprehensive documentation from your schema including ERD diagrams and API documentation:

```sh
migrato docs --format plantuml --output erd.puml
migrato docs --format mermaid --output erd.md
migrato docs --format graphviz --output erd.dot
migrato docs --format api --output api.md
migrato docs --format all --output docs/
```

#### Supported Formats

1. **PlantUML** (`.puml`) - Entity Relationship Diagrams

   - Professional ERD diagrams
   - Shows primary keys, unique constraints, NOT NULL constraints
   - Displays foreign key relationships
   - Can be rendered with PlantUML tools

2. **Mermaid** (`.md`) - Markdown-compatible diagrams

   - Works with GitHub, GitLab, and other markdown renderers
   - Interactive diagrams in documentation
   - Shows table relationships clearly

3. **Graphviz** (`.dot`) - DOT format diagrams

   - Can be rendered with Graphviz tools
   - Customizable styling and layout
   - Professional diagram output

4. **API Documentation** (`.md`) - REST API docs
   - Complete CRUD endpoint documentation
   - Request/response examples
   - Field descriptions and constraints
   - Ready-to-use API documentation

#### Example PlantUML Output

```plantuml
@startuml
!theme plain
skinparam linetype ortho

entity "users" {
  id : INTEGER <<PK>>
  email : TEXT <<UQ>> <<NN>>
  name : TEXT <<NN>> <<DEFAULT: John Doe>>
  age : INTEGER <<DEFAULT: 25>>
  created_at : TIMESTAMP <<DEFAULT: now()>>
}

entity "posts" {
  id : INTEGER <<PK>>
  title : TEXT <<NN>>
  content : TEXT <<NN>>
  user_id : INTEGER
  created_at : TIMESTAMP <<DEFAULT: now()>>
}

"users" ||--o{ "posts" : "user_id"
@enduml
```

#### Example API Documentation Output

````markdown
# REST API Documentation

## Users

### GET /users

Retrieve all users.

**Response:**

```json
[
  {
    "id": 1,
    "email": "user@example.com",
    "name": "John Doe",
    "age": 25,
    "created_at": "2024-01-01T00:00:00Z"
  }
]
```
````

````

### Migration History & Logging

Track and audit your migration activities with comprehensive history and logging:

#### Migration History

View detailed migration records with execution times and user information:

```sh
migrato history                    # Show all migration history
migrato history --limit 10         # Show last 10 migrations
migrato history --table users      # Show migrations for specific table
migrato history --detailed         # Show detailed information
````

**History Output Example:**

```
📋 Migration History
============================================================
ID   Status   Migration              Duration    User       Date
1    ✅       20240101120000_init.sql 2.5s        john       2024-01-01 12:00
2    ✅       20240101130000_users.sql 1.8s       jane       2024-01-01 13:00
3    ❌       20240101140000_posts.sql 0.5s       john       2024-01-01 14:00

📊 Summary: 3 total, 2 successful, 1 failed
⏱️  Total execution time: 4.8s
```

#### Migration Logs

View recent migration activities and logs:

```sh
migrato log                    # Show recent migration logs
migrato log --limit 20         # Show last 20 log entries
```

**Log Output Example:**

```
📋 Recent Migration Activities
============================================================

1. ℹ️  [2024-01-01 12:00:05] Starting migration: 20240101120000_init.sql (by john)
   📄 Details: Migration execution started

2. ✅ [2024-01-01 12:00:07] Migration completed: 20240101120000_init.sql (by john)
   📄 Details: Execution time: 2.5s

3. ℹ️  [2024-01-01 13:00:10] Starting migration: 20240101130000_users.sql (by jane)
   📄 Details: Migration execution started

4. ✅ [2024-01-01 13:00:12] Migration completed: 20240101130000_users.sql (by jane)
   📄 Details: Execution time: 1.8s

------------------------------------------------------------
📊 Showing 4 recent log entries
```

#### Enhanced Tracking Features

- **Execution Time Tracking**: Monitor how long each migration takes
- **User Tracking**: See who ran each migration
- **Status Tracking**: Track successful, failed, and pending migrations
- **Error Logging**: Detailed error messages for failed migrations
- **Checksum Verification**: Ensure migration integrity
- **Table Filtering**: Filter history by affected tables
- **Detailed Views**: Comprehensive information for debugging

## How it works

- Reads your schema YAML
- Introspects the current database
- Diffs schema vs. database
- Generates SQL for:
  - Creating new tables
  - Adding new columns
  - Modifying existing columns (type, NOT NULL, default values)
  - Dropping existing columns
  - Dropping existing tables
- **Automatically generates rollback SQL** for each migration
- Writes migration files to `migrations/` with up/down sections
- Applies migrations and tracks them in `schema_migrations` table
- Supports rolling back migrations using the generated rollback SQL

## Database Browser (Studio)

Launch a web-based database browser similar to Prisma Studio:

```sh
migrato studio                    # Start on default port 8080
migrato studio --port 3000       # Start on custom port
```

### Features

- **Table Browser**: View all tables in your database
- **Data Viewer**: Browse table data with pagination
- **Search & Filter**: Search across text columns
- **Inline Editing**: Real-time data editing with validation
- **Table Relationships**: Visual representation of foreign key relationships
- **Export/Import**: Export data as CSV, JSON, or SQL; import from files
- **Responsive Design**: Works on desktop and mobile
- **Real-time Updates**: Live data from your database

### Usage

1. **Set your database URL**:

   ```env
   DATABASE_URL=postgres://user:password@localhost:5432/dbname?sslmode=disable
   ```

2. **Start the studio**:

   ```sh
   migrato studio
   ```

3. **Open your browser** to `http://localhost:8080`

4. **Browse your data**:
   - Select tables from the sidebar
   - View data with pagination
   - Search across columns
   - Navigate through large datasets

### Interface

The web interface provides:

- **Sidebar**: List of all tables in your database
- **Main Content**: Table data with search and pagination
- **Search Box**: Filter data across text columns
- **Pagination**: Navigate through large datasets
- **Responsive Layout**: Works on all screen sizes

### Table Relationships

View and explore foreign key relationships between your tables:

1. **Switch to Relationships Tab**: Click the "Relationships" tab in the main interface
2. **Visual Diagrams**: View relationships as Mermaid ERD diagrams
3. **Text View**: Browse relationships in a structured text format
4. **Interactive Navigation**: Click between different visualization modes

#### Relationship Features

- **Mermaid Diagrams**: Professional ERD diagrams showing table relationships
- **Foreign Key Mapping**: Clear visualization of source → target relationships
- **Constraint Names**: View the actual constraint names from your database
- **Multiple Views**: Switch between diagram and text representations
- **Real-time Loading**: Relationships loaded directly from your database schema

#### Example Relationship View

```
users ||--o{ posts : "user_id -> id"
posts ||--o{ comments : "post_id -> id"
categories ||--o{ posts : "category_id -> id"
```

This shows how tables are connected through foreign keys, making it easy to understand your database structure and data flow.

### Inline Editing

Migrato Studio includes powerful inline editing capabilities:

#### Enable Edit Mode

1. **Click the "Enable Edit Mode" button** in the table view
2. **Click on any cell** to start editing
3. **Press Enter** to save or **Escape** to cancel
4. **Click outside** the cell to save changes

#### Features

- **Real-time Validation**: Server-side validation of data types and constraints
- **Visual Feedback**: Success/error notifications for all operations
- **Keyboard Shortcuts**: Enter to save, Escape to cancel
- **Data Type Support**: Handles text, numbers, booleans, dates, and NULL values
- **Constraint Validation**: Respects NOT NULL, unique, and foreign key constraints
- **Error Handling**: Clear error messages for validation failures

#### Supported Operations

- **Text Fields**: Edit string values with full validation
- **Numeric Fields**: Edit integers and decimals with type checking
- **Boolean Fields**: Toggle true/false values
- **NULL Values**: Set fields to NULL (if allowed by schema)
- **Date/Time**: Edit timestamp fields with proper formatting

#### Safety Features

- **Server-side Validation**: All changes validated against database schema
- **Transaction Safety**: Updates use proper SQL transactions
- **Error Recovery**: Failed updates don't affect the original data
- **Audit Trail**: All changes logged for tracking

Perfect for exploring and managing your database during development!

## Health Checks & Validation

### Database Health Check

Check if your database is accessible and properly configured:

```sh
migrato health                    # Basic health check
migrato health --timeout 10s      # Custom timeout
```

This checks:

- Database connectivity
- Schema migrations table existence
- Applied migration count

### Issue Detection

Check for potential issues between your schema and database:

```sh
migrato check                     # Basic check
migrato check --fix-suggestions   # Show detailed suggestions
```

This detects:

- Orphaned tables/columns in database
- Missing tables/columns from schema
- Index mismatches
- Foreign key constraint issues
- Pending migrations

## Release Notes

For detailed information about each release, see the [CHANGELOG.md](CHANGELOG.md) file.

### Latest Release: v1.3.0

**Key Features:**

- **Go struct-based schema definition** with `migrato` tags (recommended approach)
- **Improved diff logic** for minimal migration generation
- **Enhanced rollback support** with robust error handling
- **Dual schema support** for both Go structs and YAML

**Breaking Changes:** None - existing YAML workflows continue to work.

**Migration Guide:** New projects are recommended to use Go structs for better type safety and IDE support.

### Previous Releases

- **v1.2.0**: Database browser (Studio), enhanced logging, schema documentation
- **v1.1.x**: Core migration functionality, YAML schema support

## Requirements

- Go 1.22+
- PostgreSQL database

## License

MIT

---

> [GitHub: ridoystarlord/migrato](https://github.com/ridoystarlord/migrato)
