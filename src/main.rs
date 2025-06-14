// codegen/src/main.rs
use anyhow::{Context, Result};
use clap::{Parser, Subcommand};
use std::fs;
use std::path::PathBuf;

mod config;
mod database;
mod generators;
mod ir;

use database::common::DatabaseConnector;
use generators::common::CodeGenerator;

#[derive(Parser, Debug)]
#[clap(author, version, about, long_about = None)]
#[clap(propagate_version = true)]
struct Cli {
    #[clap(subcommand)]
    command: Commands,
}

#[derive(Subcommand, Debug)]
enum Commands {
    /// Generates model structs/classes from database schema
    Model(ModelArgs),
}

#[derive(Parser, Debug)]
#[clap(about = "Generates model structs/classes from database schema")]
struct ModelArgs {
    /// Path to the configuration YAML file
    #[clap(short, long, value_parser)]
    config: PathBuf,
    /// Initialize a new config.yaml file with default settings
    #[clap(long)]
    init: bool,
    /// Override the active database name from config
    #[clap(short, long, value_parser)]
    db_name: Option<String>,
    /// Override database type (e.g., mysql, postgres, sqlite) - Applies to active DB
    #[clap(long, value_parser)]
    db_type: Option<String>,
    /// Override database connection string - Applies to active DB
    #[clap(long, value_parser)]
    dsn: Option<String>,
    /// Target language(s) to generate code for (e.g., go, rust, typescript) - Overrides config if present
    /// Use comma-separated values for multiple languages, e.g., "go,typescript"
    #[clap(short, long, value_parser, value_delimiter = ',')]
    lang: Option<Vec<String>>,
    /// Output directory for generated files - Overrides config if present
    #[clap(short, long, value_parser)]
    output: Option<PathBuf>,
    /// Specific table(s) to generate code for - Overrides config table patterns if present
    /// Use comma-separated values for multiple tables, e.g., "users,posts,comments"
    #[clap(short, long, value_parser, value_delimiter = ',')]
    table: Option<Vec<String>>,
}

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();

    match &cli.command {
        Commands::Model(args) => {
            // Handle --init flag to create default config
            if args.init {
                return init_config(&args.config).await;
            }

            // 1. Load configuration
            let mut cfg = config::Config::load(&args.config)?;

            // 2. Determine active database configuration
            let active_db_name = args
                .db_name
                .as_ref()
                .unwrap_or(&cfg.active_database)
                .clone();

            let mut active_db_config =
                cfg.databases.remove(&active_db_name).with_context(|| {
                    format!(
                        "Active database '{}' not found in config.databases",
                        active_db_name
                    )
                })?;

            // 3. Override active database config with CLI arguments if provided
            if let Some(db_type) = args.db_type.clone() {
                active_db_config.db_type = db_type;
            }
            if let Some(dsn) = args.dsn.clone() {
                active_db_config.dsn = dsn;
            }

            // 4. Override generation config with CLI arguments if provided
            if let Some(lang_cli_args) = args.lang.clone() {
                cfg.generation.target_languages = lang_cli_args;
            }
            if let Some(output) = args.output.clone() {
                cfg.generation.output_dir = output;
            }

            // 5. Override table filtering with CLI arguments if provided
            if let Some(tables) = args.table.clone() {
                println!("🎯 Filtering to specific tables: {}", tables.join(", "));
                // Create table patterns that include only specified tables
                cfg.generation.table_name_patterns = Some(crate::config::TableNamePatterns {
                    include: tables,
                    exclude: vec![],
                });
            }

            // 6. Connect to database and introspect schema using the active_db_config
            let db_connector: Box<dyn database::common::DatabaseConnector + Send> =
                match active_db_config.db_type.as_str() {
                    "mysql" => Box::new(
                        <database::mysql::MySqlConnector as DatabaseConnector>::new(
                            &active_db_config.dsn,
                        )
                        .await?,
                    ),
                    "postgres" => Box::new(
                        <database::postgres::PostgresConnector as DatabaseConnector>::new(
                            &active_db_config.dsn,
                        )
                        .await?,
                    ),
                    "sqlite" => Box::new(
                        <database::sqlite::SqliteConnector as DatabaseConnector>::new(
                            &active_db_config.dsn,
                        )
                        .await?,
                    ),
                    _ => anyhow::bail!("Unsupported database type: {}", active_db_config.db_type),
                };

            println!(
                "Connecting to database and introspecting schema '{}'...",
                active_db_config.db_name
            );
            let schema = db_connector.get_schema(&active_db_config.db_name).await?;
            println!(
                "Schema introspection complete for database: {}",
                schema.name
            );

            // 7. Generate code for each target language dynamically
            for lang_name in &cfg.generation.target_languages {
                let lang_cfg = cfg.languages.get(lang_name).with_context(|| {
                    format!(
                        "Language config for '{}' not found in config.languages",
                        lang_name
                    )
                })?;

                println!("🚀 Generating code for language: {}", lang_name);
                let generator = <generators::template_generator::TemplateCodeGenerator as CodeGenerator<'_>>::new(
                    &cfg,
                    &active_db_config,
                    lang_cfg,
                )?;
                generator
                    .generate_code(&schema, &cfg.generation.output_dir)
                    .await?;
            }

            println!("📦 Code generation complete!");
        }
    }

    Ok(())
}

async fn init_config(config_path: &PathBuf) -> Result<()> {
    // Check if config file already exists
    if config_path.exists() {
        anyhow::bail!(
            "Configuration file already exists at: {}",
            config_path.display()
        );
    }

    // Create default configuration
    let default_config = r#"# Database Code Generator Configuration
# This file defines how to connect to databases and generate code

# Active database to use for generation
active_database: "main"

# Database configurations
databases:
  main:
    db_type: "mysql"  # mysql, postgres, sqlite
    db_name: "your_database_name"
    dsn: "mysql://username:password@localhost:3306/your_database_name"

# Code generation settings
generation:
  # Languages to generate code for
  target_languages:
    - "rust"
    - "typescript"

  # Output directory for generated files
  output_dir: "./generated"

  # Template directory for custom templates (optional)
  template_dir: "./templates"

  # Table filtering patterns (optional)
  table_patterns:
    include:
      - "*"  # Include all tables by default
    exclude:
      - "_*"  # Exclude tables starting with underscore
      - "temp_*"  # Exclude temporary tables

# Language-specific configurations
# All fields are optional with sensible defaults:
# - template_file, template_path, output_extension: auto-detected
# - nullable_strategy: defaults to "generic"
languages:
  rust:
    nullable_strategy: "option"
    # Optional custom tags example:
    # tags:
    #   - '#[serde(rename = "{{ OriginalColumnName }}")]'
    #   - '#[sqlx(rename = "{{ OriginalColumnName }}")]'

  typescript:
    nullable_strategy: "union"

  go:
    nullable_strategy: "pointer"
    package_name: "models"
    # Optional custom tags example:
    # tags:
    #   - 'gorm:"column:{{ OriginalColumnName }}"'
    #   - 'db:"{{ OriginalColumnName }}"'

  python:
    nullable_strategy: "optional"

  java:
    nullable_strategy: "optional"
    # Optional custom tags example:
    # tags:
    #   - '@JsonProperty("{{ OriginalColumnName }}")'
    #   - '@Column(name = "{{ OriginalColumnName }}")'

  csharp:
    nullable_strategy: "nullable"
    # Optional custom tags example:
    # tags:
    #   - '[JsonProperty("{{ OriginalColumnName }}")]'
    #   - '[Column("{{ OriginalColumnName }}")]'

  php:
    nullable_strategy: "nullable"

  ruby:
    nullable_strategy: "nil"

  swift:
    nullable_strategy: "optional"

  kotlin:
    nullable_strategy: "nullable"

  dart:
    nullable_strategy: "nullable"

  zig:
    nullable_strategy: "optional"

  nim:
    nullable_strategy: "option"
"#;

    // Write default configuration to file
    fs::write(config_path, default_config)
        .with_context(|| format!("Failed to write config file to: {}", config_path.display()))?;

    println!(
        "✅ Created default configuration file: {}",
        config_path.display()
    );
    println!("📝 Please edit the file to configure your database connection and settings.");
    println!(
        "🚀 Run 'codegen model -c {}' to generate code after configuration.",
        config_path.display()
    );

    Ok(())
}
