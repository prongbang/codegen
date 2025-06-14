// codegen/src/database/mysql.rs
use super::common::DatabaseConnector;
use crate::ir::Column;
use crate::ir::DatabaseSchema;
use crate::ir::Table;
use anyhow::{Context, Result};
use async_trait::async_trait;
use sqlx::{MySqlPool, Row};

pub struct MySqlConnector {
    pool: MySqlPool,
}

#[async_trait]
impl DatabaseConnector for MySqlConnector {
    async fn new(dsn: &str) -> Result<Self> {
        let pool = MySqlPool::connect(dsn)
            .await
            .with_context(|| format!("Failed to connect to MySQL: {}", dsn))?;
        Ok(MySqlConnector { pool })
    }

    async fn get_schema(&self, database_name: &str) -> Result<DatabaseSchema> {
        let mut schema = DatabaseSchema {
            name: database_name.to_string(),
            tables: Vec::new(),
        };

        // Get table names
        let table_rows = sqlx::query(
            r#"
            SELECT table_name
            FROM information_schema.tables
            WHERE table_schema = ? AND table_type = 'BASE TABLE'
            "#,
        )
        .bind(database_name)
        .fetch_all(&self.pool)
        .await
        .context("Failed to query MySQL table names")?;

        for table_row in table_rows {
            let table_name: String = table_row.get("table_name");
            let mut table = Table {
                name: table_name.clone(),
                columns: Vec::new(),
            };

            // Get column details for each table
            let column_rows = sqlx::query(
                r#"
                SELECT
                    column_name,
                    data_type,
                    is_nullable,
                    column_key,
                    column_default,
                    extra,
                    column_comment
                FROM information_schema.columns
                WHERE table_schema = ? AND table_name = ?
                ORDER BY ordinal_position
                "#,
            )
            .bind(database_name)
            .bind(&table_name)
            .fetch_all(&self.pool)
            .await
            .with_context(|| format!("Failed to query columns for table: {}", table_name))?;

            for col_row in column_rows {
                let is_nullable: String = col_row.get("is_nullable");
                let is_nullable = is_nullable == "YES";
                let column_key: Option<String> = col_row.get("column_key");
                let is_primary_key = column_key.as_deref() == Some("PRI");
                let data_type: String = col_row.get("data_type");

                let generic_type = match data_type.as_str() {
                    "varchar" | "text" | "longtext" | "mediumtext" | "char" => "string",
                    "int" | "tinyint" | "smallint" | "mediumint" | "bigint" => "integer",
                    "float" | "double" | "decimal" => "float",
                    "boolean" => "boolean",
                    "datetime" | "timestamp" | "date" => "datetime",
                    "blob" | "longblob" | "mediumblob" | "tinyblob" | "binary" | "varbinary" => {
                        "bytes"
                    }
                    _ => "string",
                }
                .to_string();

                let column = Column {
                    name: col_row.get("column_name"),
                    database_type: data_type,
                    generic_type,
                    is_nullable,
                    default_value: col_row.get("column_default"),
                    comment: col_row.get("column_comment"),
                    is_primary_key,
                };
                table.columns.push(column);
            }
            schema.tables.push(table);
        }

        Ok(schema)
    }
}
