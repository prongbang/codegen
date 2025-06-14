// codegen/src/database/postgres.rs
use super::common::DatabaseConnector;
use crate::ir::Column;
use crate::ir::DatabaseSchema;
use crate::ir::Table;
use anyhow::{Context, Result};
use async_trait::async_trait;
use sqlx::{PgPool, Row};

pub struct PostgresConnector {
    pool: PgPool,
}

#[async_trait]
impl DatabaseConnector for PostgresConnector {
    async fn new(dsn: &str) -> Result<Self> {
        let pool = PgPool::connect(dsn)
            .await
            .with_context(|| format!("Failed to connect to PostgreSQL: {}", dsn))?;
        Ok(PostgresConnector { pool })
    }

    async fn get_schema(&self, database_name: &str) -> Result<DatabaseSchema> {
        let mut schema = DatabaseSchema {
            name: database_name.to_string(),
            tables: Vec::new(),
        };

        // Get table names
        let table_rows = sqlx::query_scalar::<_, String>(
            r#"
            SELECT tablename
            FROM pg_catalog.pg_tables
            WHERE schemaname = 'public'
            ORDER BY tablename
            "#,
        )
        .fetch_all(&self.pool)
        .await
        .context("Failed to query PostgreSQL table names")?;

        for table_name in table_rows {
            let mut table = Table {
                name: table_name.clone(),
                columns: Vec::new(),
            };

            // Get column details for each table
            let column_rows = sqlx::query(
                r#"
                SELECT
                    column_name,
                    udt_name AS data_type,
                    is_nullable,
                    column_default,
                    pg_catalog.col_description(c.oid, c.attnum) AS column_comment
                FROM information_schema.columns isc
                JOIN pg_catalog.pg_class t ON t.relname = isc.table_name
                JOIN pg_catalog.pg_attribute c ON c.attrelid = t.oid AND c.attname = isc.column_name
                WHERE isc.table_schema = 'public' AND isc.table_name = $1
                ORDER BY ordinal_position
                "#,
            )
            .bind(&table_name)
            .fetch_all(&self.pool)
            .await
            .with_context(|| format!("Failed to query columns for table: {}", table_name))?;

            // Get primary keys separately as information_schema.columns doesn't easily expose it
            let pk_rows = sqlx::query(
                r#"
                SELECT
                    a.attname AS column_name
                FROM
                    pg_index i
                JOIN
                    pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
                WHERE
                    i.indrelid = $1::regclass AND i.indisprimary
                "#,
            )
            .bind(&table_name)
            .fetch_all(&self.pool)
            .await
            .with_context(|| format!("Failed to query primary keys for table: {}", table_name))?;

            let primary_keys: Vec<String> = pk_rows.into_iter().map(|r| r.get("column_name")).collect();

            for col_row in column_rows {
                let column_name: String = col_row.get("column_name");
                let is_nullable_str: Option<String> = col_row.get("is_nullable");
                let is_nullable = is_nullable_str.as_deref() == Some("YES");
                let is_primary_key = primary_keys.contains(&column_name);
                let data_type: Option<String> = col_row.get("data_type");

                // This generic type mapping should ideally come from config.type_mappings for robustness.
                let generic_type = match data_type.as_deref().unwrap_or("") {
                    "varchar" | "text" | "uuid" | "name" | "bpchar" => "string",
                    "int2" | "int4" | "int8" | "serial4" | "serial8" => "integer",
                    "float4" | "float8" | "numeric" => "float",
                    "bool" => "boolean",
                    "timestamptz" | "timestamp" | "date" => "datetime",
                    "bytea" => "bytes",
                    _ => "string", // Fallback for unknown types
                }
                .to_string();

                let column = Column {
                    name: column_name,
                    database_type: data_type.unwrap_or_default(),
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
