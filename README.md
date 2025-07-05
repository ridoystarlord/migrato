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
- **Schema validation**: Validate schema syntax and relationships
- **Issue detection**: Find and suggest fixes for schema issues
- **Default values**: Support for literal values and functions
- **Visual schema diff**: Preview changes with color-coded tree format
- **Schema documentation**: Generate ERD diagrams and API docs (PlantUML, Mermaid, Graphviz)
- **Go struct generation**: Generate Go structs and repositories (experimental)
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

1. **Set up your database connection**

   - Set the `DATABASE_URL` environment variable (or create a `.env` file):
     ```env
     DATABASE_URL=postgres://user:password@localhost:5432/dbname?sslmode=disable
     ```

2. **Initialize a schema**

   ```sh
   migrato init
   # Creates a sample schema.yaml
   ```

3. **Edit `schema.yaml`** to define your tables and columns.

4. **Generate a migration**

   ```sh
   migrato generate
   # Generates a SQL migration file in the migrations/ folder
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

- `migrato init` — Create an example `schema.yaml` file
- `migrato generate` — Generate a migration file from your schema
  - `-f, --file` — Specify a custom schema YAML file (default: `schema.yaml`)
- `migrato generate-structs` — Generate Go structs and repositories from schema (Experimental)
  - `-f, --file` — Specify a custom schema YAML file (default: `schema.yaml`)
  - `-o, --output` — Output directory for generated structs (default: `models`)
  - `-p, --package` — Package name for generated structs (default: `models`)
- `migrato migrate` — Apply all pending migrations
- `migrato rollback` — Rollback migrations
  - `-s, --steps` — Number of migrations to rollback (default: 1)
- `migrato status` — Show applied and pending migrations
- `migrato health` — Check database connectivity
  - `-t, --timeout` — Timeout for health check (default: 5s)
- `migrato validate` — Validate schema integrity
  - `-f, --file` — Specify a custom schema YAML file (default: `schema.yaml`)
- `migrato check` — Check for potential issues
  - `-f, --fix-suggestions` — Show suggestions for fixing issues
- `migrato diff` — Show differences between schema and database
  - `-v, --visual` — Show changes in visual tree format with colors
  - `-f, --file` — Specify a custom schema YAML file (default: `schema.yaml`)
- `migrato docs` — Generate documentation from schema
  - `-f, --format` — Output format (plantuml, mermaid, graphviz, api, all)
  - `-o, --output` — Output file or directory (default: format-specific filename)
  - `--file` — Schema file to use (default: `schema.yaml`)

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

### Visual Schema Diff

Preview your schema changes before generating migrations with the visual diff feature:

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

### POST /users

Create a new user.

**Request Body:**

```json
{
  "email": "user@example.com",
  "name": "John Doe",
  "age": 25
}
```

````

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

## Go Struct Generation (Experimental)

Generate type-safe Go structs and repositories from your schema:

```sh
migrato generate-structs
````

This creates a clean, modular structure:

- **models/** - Individual model files (user.go, post.go, etc.)
- **repositories/** - Repository implementations (user_repository.go, post_repository.go, etc.)
- **models/db.go** - Database interface definition

### Generated Structure

```
models/
├── models/
│   ├── user.go
│   ├── post.go
│   └── db.go
└── repositories/
    ├── user_repository.go
    └── post_repository.go
```

### Example Generated Code

```go
// models/user.go
type User struct {
	ID    int    `db:"id" json:"id" migrato:"primary"`
	Email string `db:"email" json:"email" migrato:"unique"`
	Name  string `db:"name" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// repositories/user_repository.go
type UserRepository struct {
	db *DB
}

func (r *UserRepository) Create(user *User) error {
	return r.db.Create(user).Error
}

// models/db.go
type DB interface {
	Create(value interface{}) *DB
	Where(query interface{}, args ...interface{}) *DB
	Find(dest interface{}) *DB
	First(dest interface{}) *DB
	Save(value interface{}) *DB
	Delete(value interface{}) *DB
	Error() error
}
```

Perfect for building your own ORM or integrating with any database driver!

> **Note**: This feature is experimental. The primary focus is on the migration CLI functionality. We do not currently recommend using this feature in production.

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

### Schema Validation

Validate your YAML schema for integrity and consistency:

```sh
migrato validate                  # Validate default schema.yaml
migrato validate -f custom.yaml   # Validate custom schema file
```

This validates:

- Duplicate table/column names
- Primary key constraints
- Foreign key references
- Column types
- Index definitions
- Relationship definitions

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

## Requirements

- Go 1.22+
- PostgreSQL database

## License

MIT

---

> [GitHub: ridoystarlord/migrato](https://github.com/ridoystarlord/migrato)
