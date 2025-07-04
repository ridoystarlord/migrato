# migrato

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A lightweight, Prisma-like migration tool for Go and PostgreSQL.

## Features

- Generate SQL migrations from a YAML schema
- Introspect your existing PostgreSQL database
- Apply and track migrations
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

## CLI Commands

- `migrato init` — Create an example `schema.yaml` file
- `migrato generate` — Generate a migration file from your schema
  - `-f, --file` — Specify a custom schema YAML file (default: `schema.yaml`)
- `migrato migrate` — Apply all pending migrations
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
      - name: created_at
        type: timestamp
        default: now()
```

## How it works

- Reads your schema YAML
- Introspects the current database
- Diffs schema vs. database
- Generates SQL for new tables/columns
- Writes migration files to `migrations/`
- Applies migrations and tracks them in `schema_migrations` table

## Requirements

- Go 1.22+
- PostgreSQL database

## License

MIT

---

> [GitHub: ridoystarlord/migrato](https://github.com/ridoystarlord/migrato)
