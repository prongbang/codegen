// codegen/src/database/common.rs
use crate::ir::DatabaseSchema;
use anyhow::Result;
use async_trait::async_trait;

#[async_trait]
pub trait DatabaseConnector {
    async fn new(dsn: &str) -> Result<Self>
    where
        Self: Sized;
    async fn get_schema(&self, database_name: &str) -> Result<DatabaseSchema>;
}
