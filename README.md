# migrato

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A lightweight, Prisma-like migration tool for Go and PostgreSQL.

## Features

- Generate SQL migrations from a YAML schema
- Introspect your existing PostgreSQL database
- Apply and track migrations
- **Migration rollbacks** with automatic rollback SQL generation
- Support for creating, adding, and dropping tables and columns
- **Index management** with support for various index types
- Foreign key relationships with configurable cascade options
- Support for one-to-many, many-to-many, and one-to-one relationships
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
        index: true
      - name: name
        type: text
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

## How it works

- Reads your schema YAML
- Introspects the current database
- Diffs schema vs. database
- Generates SQL for:
  - Creating new tables
  - Adding new columns
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
```

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

## Requirements

- Go 1.22+
- PostgreSQL database

## License

MIT

---

> [GitHub: ridoystarlord/migrato](https://github.com/ridoystarlord/migrato)
