# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- New features that will be added in the next release

### Changed

- Changes in existing functionality

### Deprecated

- Features that will be removed in upcoming releases

### Removed

- Features that have been removed

### Fixed

- Bug fixes

### Security

- Security vulnerability fixes

## [1.3.0] - 2025-07-13

### Added

- **Go struct-based schema definition** with `migrato` tags

  - Define database schema using Go structs instead of YAML
  - Type-safe schema definitions with IDE support
  - Automatic type mapping from Go types to PostgreSQL types
  - Support for all PostgreSQL data types including UUID, JSON, etc.
  - Foreign key relationships with configurable cascade options
  - Index definitions with custom names and types
  - Default values for columns
  - Primary key, unique, and NOT NULL constraints

- **Improved diff logic** for minimal migration generation

  - Only generates SQL for actual schema changes
  - Eliminates unnecessary foreign key operations for unrelated tables
  - Conservative approach to schema modifications
  - Better handling of foreign key action normalization

- **Enhanced rollback support**

  - Robust error handling in rollback SQL generation
  - Proper nil pointer checks to prevent panics
  - Clear error messages for debugging

- **Dual schema support**
  - Support for both Go structs (recommended) and YAML schemas
  - `migrato init` command with flags to choose schema type
  - Backward compatibility with existing YAML workflows

### Changed

- **Default schema approach**: Go structs are now the recommended approach
- **Migration generation**: More precise and minimal SQL generation
- **Foreign key handling**: Improved detection and comparison logic

### Fixed

- **Panic issues**: Fixed nil pointer dereferences in rollback SQL generation
- **Unnecessary operations**: Eliminated redundant foreign key drops/adds
- **Type comparison**: Better handling of PostgreSQL type variations
- **Constraint normalization**: Proper handling of empty vs "NO ACTION" foreign key actions

### Migration Guide

- Existing YAML schemas continue to work without changes
- New projects are recommended to use Go structs for better type safety
- Use `migrato init` to create a new project with Go structs
- Use `migrato generate --models models` to generate migrations from structs

## [1.2.0] - 2024-12-XX

### Added

- Database browser (Studio) with web-based interface
- Enhanced migration history and logging
- Schema documentation generation (PlantUML, Mermaid, Graphviz)
- Visual schema diff with color-coded tree format
- Health checks and validation features

### Changed

- Improved CLI interface and command organization
- Better error handling and user feedback

## [1.1.10] - 2024-XX-XX

### Added

- Column modification support
- Index management features
- Foreign key relationship support

### Fixed

- Various bug fixes and improvements

## [1.1.9] - 2024-XX-XX

### Added

- Migration rollback functionality
- Schema validation features

### Changed

- Improved migration tracking

## [1.1.8] - 2024-XX-XX

### Added

- Basic migration generation from YAML schema
- Database introspection capabilities

### Changed

- Initial stable release features

## [1.1.7] - 2024-XX-XX

### Added

- Initial release with basic functionality
- YAML schema support
- PostgreSQL integration

---

## Release Notes Format

Each release includes:

- **Added**: New features
- **Changed**: Changes in existing functionality
- **Deprecated**: Features that will be removed
- **Removed**: Features that have been removed
- **Fixed**: Bug fixes
- **Security**: Security vulnerability fixes

## Migration Guide

When upgrading between major versions, check this section for breaking changes and migration steps.

### Upgrading to v1.3.0

- **No breaking changes** for existing YAML-based workflows
- **New Go struct approach** is optional but recommended
- **Improved diff logic** provides more precise migrations
- **Enhanced error handling** prevents crashes during rollback generation
