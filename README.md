# CodeGen - Database Model Generator

A powerful, flexible code generator that creates model structs/classes from database schemas across multiple programming languages.

## Features

- 🗄️ **Multi-Database Support**: MySQL, PostgreSQL, SQLite
- 🌐 **Multi-Language Generation**: 13+ programming languages supported
- 🎨 **Template-Based**: Fully customizable Handlebars templates
- ⚙️ **Configurable**: Extensive configuration options via YAML
- 🔧 **Extensible**: Easy to add new languages without code changes
- 📝 **Type Mapping**: Intelligent database-to-language type conversion
- 🎯 **Pattern Filtering**: Include/exclude tables using patterns
- 📦 **CLI Interface**: Simple command-line interface

## Supported Languages

| Language   | Extension | Nullable Strategy | Template |
|------------|-----------|-------------------|----------|
| Rust       | `.rs`     | `Option<T>`       | `rust_struct.hbs` |
| TypeScript | `.ts`     | `T \| null`       | `typescript_interface.hbs` |
| Python     | `.py`     | `Optional[T]`     | `python_class.hbs` |
| Java       | `.java`   | `T` (wrapper)     | `java_class.hbs` |
| C#         | `.cs`     | `T?`              | `csharp_class.hbs` |
| Go         | `.go`     | `*T` or `sql.NullT` | `go_struct.hbs` |
| PHP        | `.php`    | `?T`              | `php_class.hbs` |
| Ruby       | `.rb`     | native `nil`      | `ruby_class.hbs` |
| Zig        | `.zig`    | `?T`              | `zig_struct.hbs` |
| Kotlin     | `.kt`     | `T?`              | `kotlin_class.hbs` |
| Swift      | `.swift`  | `T?`              | `swift_struct.hbs` |
| Dart       | `.dart`   | `T?`              | `dart_class.hbs` |
| Nim        | `.nim`    | `Option[T]`       | `nim_type.hbs` |

*Plus any custom language you add via templates!*

## Installation

### Prerequisites

- Rust 1.70+ (for building from source)
- Database access (MySQL, PostgreSQL, or SQLite)

### Build from Source

```bash
git clone https://github.com/prongbang/codegen.git
cd codegen
cargo build --release
```

### Install via Cargo

```bash
cargo install --path .
```

## Quick Start

### 1. Create a Database

For this example, we'll use SQLite:

```bash
# Create a sample database
sqlite3 data/example.db << EOF
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT,
    user_id INTEGER,
    published BOOLEAN DEFAULT 0,
    FOREIGN KEY(user_id) REFERENCES users(id)
);
EOF
```

### 2. Create Configuration File

Create `config.yaml`:

```yaml
active_database: sqlite_example

databases:
  sqlite_example:
    db_type: sqlite
    dsn: "sqlite:./data/example.db"
    db_name: main

generation:
  output_dir: ./generated
  target_languages: ["rust", "typescript", "python"]
  template_dir: ./templates

languages:
  rust:
    template_file: rust_struct.hbs
    output_extension: .rs
    struct_name_case: PascalCase
    field_name_case: snake_case
    nullable_strategy: option
    default_imports: ["serde::{Serialize, Deserialize}"]

  typescript:
    template_file: typescript_interface.hbs
    output_extension: .ts
    struct_name_case: PascalCase
    field_name_case: camelCase
    nullable_strategy: union

  python:
    template_file: python_class.hbs
    output_extension: .py
    struct_name_case: PascalCase
    field_name_case: snake_case
    nullable_strategy: optional_type
    default_imports: ["typing.Optional", "dataclasses"]

type_mappings:
  sqlite:
    TEXT: { rust: String, typescript: string, python: str }
    INTEGER: { rust: i64, typescript: number, python: int }
    BOOLEAN: { rust: bool, typescript: boolean, python: bool }
    DATETIME: { rust: chrono::NaiveDateTime, typescript: Date, python: datetime.datetime }

naming_conventions:
  table_to_struct_case: PascalCase
  column_to_field_case: PascalCase
```

### 3. Generate Models

```bash
# Generate for all configured languages
cargo run -- model --config config.yaml

# Generate for specific languages
cargo run -- model --config config.yaml --lang "rust,typescript"

# Override output directory
cargo run -- model --config config.yaml --output ./my_models
```

### 4. View Generated Files

```
generated/
├── .rs/
│   ├── users.rs
│   └── posts.rs
├── .ts/
│   ├── users.ts
│   └── posts.ts
└── .py/
    ├── users.py
    └── posts.py
```

## Configuration Guide

### Database Configuration

#### MySQL
```yaml
databases:
  mysql_prod:
    db_type: mysql
    dsn: "mysql://user:password@localhost:3306/database"
    db_name: your_database
```

#### PostgreSQL
```yaml
databases:
  postgres_dev:
    db_type: postgres
    dsn: "postgres://user:password@localhost:5432/database"
    db_name: your_database
```

#### SQLite
```yaml
databases:
  sqlite_local:
    db_type: sqlite
    dsn: "sqlite:./path/to/database.db"
    db_name: main
```

### Generation Options

```yaml
generation:
  output_dir: ./generated
  target_languages: ["rust", "go", "typescript"]
  template_dir: ./templates

  # Advanced options
  include_timestamps: true
  include_serialization: true
  include_validation: false
  generate_interfaces: false

  # Table filtering
  table_name_patterns:
    include: ["users", "posts", "products_*"]
    exclude: ["migrations", "schema_*", "*_temp"]

  # Output structure
  output_structure: "by_language" # "by_language" | "by_table" | "flat"
```

### Language Configuration

```yaml
languages:
  your_language:
    template_file: your_template.hbs        # Template file name
    output_extension: .ext                  # File extension
    struct_name_case: PascalCase           # Struct/class naming
    field_name_case: camelCase             # Field naming
    nullable_strategy: union               # How to handle nulls
    tags:                                  # Language-specific annotations
      JsonProperty: '"{{ column_name }}"'
    default_imports:                       # Default imports
      - "json"
      - "datetime"
    field_prefix: "m_"                     # Optional field prefix
```

#### Nullable Strategies

- `option`: Rust `Option<T>`, Zig `?T`
- `union`: TypeScript `T | null`
- `nullable_type`: C# `T?`, Java `Wrapper<T>`
- `optional_type`: Python `Optional[T]`, Haskell `Maybe T`
- `pointer`: Go `*T`
- `native`: Ruby native `nil`

### Type Mappings

```yaml
type_mappings:
  mysql:  # Database type
    varchar:  # Database column type
      rust: String
      go: string
      typescript: string
      python: str
      java: String
    int:
      rust: i64
      go: int64
      typescript: number
      python: int
      java: Long
```

## CLI Usage

### Basic Commands

```bash
# Show help
codegen --help
codegen model --help

# Generate with config file
codegen model --config config.yaml

# Override active database
codegen model --config config.yaml --db-name production

# Override database type and DSN
codegen model --config config.yaml --db-type postgres --dsn "postgres://..."

# Specify target languages
codegen model --config config.yaml --lang "rust,go,typescript"

# Override output directory
codegen model --config config.yaml --output ./generated_models
```

### Examples

```bash
# Generate Rust models only
codegen model -c config.yaml -l rust -o ./rust_models

# Generate for multiple databases
codegen model -c config.yaml --db-name staging --lang "typescript,python"

# Generate with pattern filtering (via config)
codegen model -c config.yaml  # Uses table_name_patterns from config
```

## Adding New Languages

You can add support for any programming language without modifying the Rust code:

### 1. Create Template File

Create `templates/mylang_model.hbs`:

```handlebars
// {{ StructName }} model for {{ TableName }}
// Generated from database

{{#if Imports}}
{{#each Imports}}
import {{this}};
{{/each}}

{{/if}}

{{#if (eq CurrentLanguage "mylang")}}
// This content only appears for MyLang
{{/if}}

class {{ StructName }} {
{{#each Columns}}
    {{#if ColumnComment}}
    // {{ ColumnComment }}
    {{/if}}
    {{#if LangTags}}
    {{{ LangTags }}}
    {{/if}}
    {{ FieldName }}: {{ LangType }};
{{/each}}

    constructor(
{{#each Columns}}
        {{ FieldName }}: {{ LangType }}{{#unless @last}},{{/unless}}
{{/each}}
    ) {
{{#each Columns}}
        this.{{ FieldName }} = {{ FieldName }};
{{/each}}
    }

    static fromJson(json: string): {{ StructName }} {
        const data = JSON.parse(json);
        return new {{ StructName }}(
{{#each Columns}}
            data.{{ OriginalColumnName }}{{#unless @last}},{{/unless}}
{{/each}}
        );
    }

    toJson(): string {
        return JSON.stringify({
{{#each Columns}}
            {{ OriginalColumnName }}: this.{{ FieldName }}{{#unless @last}},{{/unless}}
{{/each}}
        });
    }
}
```

### 2. Add Language Configuration

Add to your `config.yaml`:

```yaml
languages:
  mylang:
    template_file: mylang_model.hbs
    output_extension: .mylang
    struct_name_case: PascalCase
    field_name_case: camelCase
    nullable_strategy: union
    tags:
      JsonProperty: '"{{ column_name }}"'
    default_imports: ["json", "datetime"]
    field_prefix: ~
```

### 3. Add Type Mappings

```yaml
type_mappings:
  sqlite:
    TEXT: { mylang: "String" }
    INTEGER: { mylang: "Integer" }
    BOOLEAN: { mylang: "Boolean" }
    DATETIME: { mylang: "DateTime" }
```

### 4. Generate!

```bash
codegen model --config config.yaml --lang mylang
```

## Template Reference

### Available Variables

In your Handlebars templates, you have access to:

#### Global Variables
- `{{ StructName }}` - The struct/class name (case-converted)
- `{{ TableName }}` - Original database table name
- `{{ CurrentLanguage }}` - Current language being generated
- `{{ Imports }}` - Array of default imports
- `{{ Config }}` - Configuration object

#### Column Variables (in `{{#each Columns}}`)
- `{{ FieldName }}` - Field name (case-converted)
- `{{ LangType }}` - Language-specific type
- `{{ LangTags }}` - Language-specific tags/annotations
- `{{ IsNullable }}` - Boolean, true if column is nullable
- `{{ OriginalColumnName }}` - Original database column name
- `{{ ColumnComment }}` - Database column comment

### Helper Functions

- `{{ to_snake_case text }}` - Convert to snake_case
- `{{ to_pascal_case text }}` - Convert to PascalCase
- `{{ to_camel_case text }}` - Convert to camelCase
- `{{ to_kebab_case text }}` - Convert to kebab-case
- `{{ to_screaming_snake_case text }}` - Convert to SCREAMING_SNAKE_CASE
- `{{ capitalize text }}` - Capitalize first letter
- `{{ uncapitalize text }}` - Lowercase first letter
- `{{ pluralize word }}` - Make word plural
- `{{ if_lang "language" "content" }}` - Conditional content for specific language
- `{{ contains array item }}` - Check if array contains item

### Template Examples

#### Conditional Language Content
```handlebars
{{#if (eq CurrentLanguage "rust")}}
#[derive(Debug, Serialize, Deserialize)]
{{/if}}
{{#if (eq CurrentLanguage "typescript")}}
export interface I{{ StructName }} {
{{/if}}
```

#### Using Helpers
```handlebars
// Convert cases
{{ to_snake_case StructName }}        // user_profile
{{ to_pascal_case FieldName }}        // UserId
{{ capitalize "hello world" }}        // Hello world

// Conditional helpers
{{ if_lang "python" "@dataclass" }}
{{ if_lang "java" "implements Serializable" }}
```

## Advanced Usage

### Environment-Specific Configs

Create different config files for different environments:

```bash
# Development
codegen model --config config.dev.yaml

# Production
codegen model --config config.prod.yaml

# Testing
codegen model --config config.test.yaml
```

### Batch Generation

```bash
# Generate for multiple databases
for db in dev staging prod; do
    codegen model --config config.yaml --db-name $db --output ./models/$db
done

# Generate for different language sets
codegen model -c config.yaml -l "rust,go" -o ./backend_models
codegen model -c config.yaml -l "typescript,dart" -o ./frontend_models
```

### Custom Output Structures

Configure different output structures in `config.yaml`:

```yaml
generation:
  output_structure: "by_language"  # Default: ./generated/.rs/, ./generated/.ts/
  # output_structure: "by_table"   # Alternative: ./generated/users/, ./generated/posts/
  # output_structure: "flat"       # Alternative: ./generated/ (all files in one dir)
```

## Troubleshooting

### Common Issues

#### Database Connection Errors
```bash
# Check your DSN format
mysql://user:password@host:port/database
postgres://user:password@host:port/database
sqlite:./path/to/file.db
```

#### Template Syntax Errors
- Use `{{ }}` for variables, not `${ }`
- Use `{{#each}}` for loops, not `for`
- Check helper function syntax: `{{ to_snake_case field }}` not `{{ field | to_snake_case }}`

#### Type Mapping Issues
- Ensure all target languages have type mappings for your database types
- Check database introspection output for actual column types
- Add fallback type mappings for unknown types

### Debug Mode

Run with debug output:

```bash
RUST_LOG=debug cargo run -- model --config config.yaml
```

### Validate Configuration

```bash
# Test config loading
codegen model --config config.yaml --lang "nonexistent"
# Will show available languages and validation errors
```

## Contributing

### Adding Database Support

1. Implement `DatabaseConnector` trait in `src/database/your_db.rs`
2. Add database type to `src/main.rs` match statement
3. Add type mappings to default `config.yaml`

### Adding Template Helpers

1. Add helper in `src/generators/template_generator.rs`
2. Register with Handlebars in the `new()` function
3. Document in README template reference

### Submitting Changes

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Update documentation
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Examples

Check out the `examples/` directory for:
- Complete configuration examples
- Custom template examples
- Multi-database setups
- Language-specific patterns

## Support

- 📖 [Documentation](https://github.com/prongbang/codegen/wiki)
- 🐛 [Issue Tracker](https://github.com/prongbang/codegen/issues)
- 💬 [Discussions](https://github.com/prongbang/codegen/discussions)

---

**Happy Code Generating! 🚀**
