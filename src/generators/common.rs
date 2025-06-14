// codegen/src/generators/common.rs
use crate::config::{Config, DatabaseConfig, LanguageConfig};
use crate::ir::DatabaseSchema;
use anyhow::Result;
use async_trait::async_trait;
use std::path::PathBuf;

#[async_trait]
pub trait CodeGenerator<'a> {
    fn new(
        overall_config: &'a Config,
        active_db_config: &'a DatabaseConfig,
        lang_config: &'a LanguageConfig,
    ) -> Result<Self>
    where
        Self: Sized;
    async fn generate_code(&self, schema: &DatabaseSchema, output_dir: &PathBuf) -> Result<()>;
}
