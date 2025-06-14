// codegen/src/database/sqlite.rs
use super::common::DatabaseConnector;
use crate::ir::{Column, DatabaseSchema, Table};
use anyhow::{Context, Result};
use async_trait::async_trait;
use sqlx::Row;
use sqlx::SqlitePool;

pub struct SqliteConnector {
    pool: SqlitePool,
}

#[async_trait]
impl DatabaseConnector for SqliteConnector {
    async fn new(dsn: &str) -> Result<Self> {
        let pool = SqlitePool::connect(dsn)
            .await
            .with_context(|| format!("Failed to connect to SQLite: {}", dsn))?;
        Ok(SqliteConnector { pool })
    }

    async fn get_schema(&self, database_name: &str) -> Result<DatabaseSchema> {
        let mut schema = DatabaseSchema {
            name: database_name.to_string(),
            tables: Vec::new(),
        };

        // Get table names
        let table_rows = sqlx::query(
            r#"
            SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'
            ORDER BY name
            "#,
        )
        .fetch_all(&self.pool)
        .await
        .context("Failed to query SQLite table names")?;

        for table_row in table_rows {
            let table_name: String = table_row.get("name");
            let mut table = Table {
                name: table_name.clone(),
                columns: Vec::new(),
            };

            // Get column details for each table using PRAGMA table_info
            let column_rows = sqlx::query(&format!("PRAGMA table_info({});", table_name))
                .fetch_all(&self.pool)
                .await
                .with_context(|| format!("Failed to query columns for table: {}", table_name))?;

            for col_row in column_rows {
                let name: String = col_row.get("name");
                let data_type: String = col_row.get("type");
                let not_null: i64 = col_row.get("notnull");
                let dflt_value: Option<String> = col_row.get("dflt_value");
                let pk: i64 = col_row.get("pk");

                let is_nullable = not_null == 0;
                let is_primary_key = pk > 0;

                // This generic type mapping should ideally come from config.type_mappings.
                let generic_type = match data_type.to_lowercase().as_str() {
                    "text" | "varchar" | "character" | "varying character" | "nchar"
                    | "native character" | "nvarchar" | "clob" => "string",
                    "integer" | "int" | "tinyint" | "smallint" | "mediumint" | "bigint"
                    | "unsigned big int" | "int2" | "int8" => "integer",
                    "real" | "double" | "double precision" | "float" | "numeric" => "float",
                    "boolean" => "boolean", // SQLite doesn't have native boolean, often integer (0/1)
                    "blob" => "bytes",
                    "datetime" | "date" => "datetime", // SQLite uses TEXT for datetime often
                    _ => "string",                     // Fallback for unknown types
                }
                .to_string();

                let column = Column {
                    name: name,
                    database_type: data_type,
                    generic_type: generic_type,
                    is_nullable,
                    default_value: dflt_value,
                    comment: None, // SQLite PRAGMA table_info does not provide comments directly
                    is_primary_key,
                };
                table.columns.push(column);
            }
            schema.tables.push(table);
        }

        Ok(schema)
    }
}
