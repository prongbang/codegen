// codegen/src/config.rs
use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::{collections::HashMap, fs, path::PathBuf};

#[derive(Debug, Deserialize, Serialize)]
pub struct Config {
    pub active_database: String,
    pub databases: HashMap<String, DatabaseConfig>,
    pub generation: GenerationConfig,
    pub languages: HashMap<String, LanguageConfig>,
    pub type_mappings: HashMap<String, HashMap<String, TypeMapping>>,
    pub naming_conventions: NamingConventions,
}

#[derive(Debug, Deserialize, Serialize, Clone)]
pub struct DatabaseConfig {
    pub db_type: String,
    pub dsn: String,
    pub db_name: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct GenerationConfig {
    pub output_dir: PathBuf,
    pub target_languages: Vec<String>,
    pub template_dir: PathBuf,
    #[serde(default)]
    pub table_name_patterns: Option<TableNamePatterns>,
    #[serde(default = "default_output_structure")]
    pub output_structure: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct TableNamePatterns {
    #[serde(default = "default_include_patterns")]
    pub include: Vec<String>,
    #[serde(default)]
    pub exclude: Vec<String>,
}

fn default_include_patterns() -> Vec<String> {
    vec!["*".to_string()]
}

fn default_output_structure() -> String {
    "by_language".to_string()
}

fn default_nullable_strategy() -> String {
    "generic".to_string()
}

#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]
pub struct LanguageConfig {
    #[serde(default)]
    pub template_file: Option<String>,  // Optional - will use default based on language if not specified
    #[serde(default)]
    pub template_path: Option<String>,  // Optional custom template path
    #[serde(default)]
    pub output_extension: Option<String>,  // Optional - will be auto-detected if not specified
    pub struct_name_case: Option<String>,
    pub field_name_case: Option<String>,
    #[serde(default = "default_nullable_strategy")]
    pub nullable_strategy: String,
    pub tags: Option<Vec<String>>,
    pub default_imports: Option<Vec<String>>,
    pub field_prefix: Option<String>,
    #[serde(default)]
    pub package_name: Option<String>,  // Package name for Go and other languages that use packages
}

#[derive(Debug, Deserialize, Serialize)]
pub struct TypeMapping {
    pub generic: String,
    #[serde(flatten)]
    pub language_types: HashMap<String, String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct NamingConventions {
    pub table_to_struct_case: String,
    pub column_to_field_case: String,
}

impl Config {
    pub fn load(path: &PathBuf) -> Result<Self> {
        let content = fs::read_to_string(path)
            .with_context(|| format!("Failed to read config file: {:?}", path))?;
        let config: Config =
            serde_yaml::from_str(&content).with_context(|| "Failed to parse config YAML")?;
        Ok(config)
    }

    pub fn should_include_table(&self, table_name: &str) -> bool {
        if let Some(patterns) = &self.generation.table_name_patterns {
            // Check exclude patterns first
            for exclude_pattern in &patterns.exclude {
                if matches_pattern(table_name, exclude_pattern) {
                    return false;
                }
            }
            
            // Check include patterns
            for include_pattern in &patterns.include {
                if matches_pattern(table_name, include_pattern) {
                    return true;
                }
            }
            
            // If no include patterns match, don't include
            false
        } else {
            // If no patterns configured, include all tables
            true
        }
    }

    pub fn get_language_type(
        &self,
        db_type: &str,
        db_column_type: &str,
        generic_type: &str,
        lang: &str,
    ) -> Option<String> {
        // 1. Try to get the specific mapping for db_type + db_column_type + lang
        self.type_mappings
            .get(db_type)
            .and_then(|db_map| db_map.get(db_column_type))
            .and_then(|type_map| type_map.language_types.get(lang).cloned())
            // 2. Fallback to generic type mapping if specific is not found
            .or_else(|| {
                // This part still contains hardcoded fallbacks.
                // For *ultimate* flexibility, you could define another map in config
                // for generic_type -> language_type fallbacks, but this is usually sufficient.
                match (lang, generic_type) {
                    ("go", "string") => Some("string".to_string()),
                    ("go", "integer") => Some("int64".to_string()),
                    ("go", "float") => Some("float64".to_string()),
                    ("go", "boolean") => Some("bool".to_string()),
                    ("go", "datetime") => Some("time.Time".to_string()),
                    ("go", "bytes") => Some("[]byte".to_string()),

                    ("rust", "string") => Some("String".to_string()),
                    ("rust", "integer") => Some("i64".to_string()),
                    ("rust", "float") => Some("f64".to_string()),
                    ("rust", "boolean") => Some("bool".to_string()),
                    ("rust", "datetime") => Some("chrono::NaiveDateTime".to_string()),
                    ("rust", "bytes") => Some("Vec<u8>".to_string()),

                    ("typescript", "string") => Some("string".to_string()),
                    ("typescript", "integer") => Some("number".to_string()),
                    ("typescript", "float") => Some("number".to_string()),
                    ("typescript", "boolean") => Some("boolean".to_string()),
                    ("typescript", "datetime") => Some("Date".to_string()),
                    ("typescript", "bytes") => Some("Uint8Array".to_string()),

                    ("csharp", "string") => Some("string".to_string()),
                    ("csharp", "integer") => Some("long".to_string()),
                    ("csharp", "float") => Some("double".to_string()),
                    ("csharp", "boolean") => Some("bool".to_string()),
                    ("csharp", "datetime") => Some("DateTime".to_string()),
                    ("csharp", "bytes") => Some("byte[]".to_string()),

                    ("java", "string") => Some("String".to_string()),
                    ("java", "integer") => Some("int".to_string()), // Default to primitive
                    ("java", "float") => Some("double".to_string()), // Default to double
                    ("java", "boolean") => Some("boolean".to_string()), // Default to primitive
                    ("java", "datetime") => Some("java.time.LocalDateTime".to_string()),
                    ("java", "bytes") => Some("byte[]".to_string()),

                    _ => None,
                }
            })
    }
}

fn matches_pattern(text: &str, pattern: &str) -> bool {
    if pattern == "*" {
        return true;
    }
    
    if pattern.contains('*') {
        // Simple wildcard matching
        if pattern.starts_with('*') && pattern.ends_with('*') {
            let middle = &pattern[1..pattern.len()-1];
            text.contains(middle)
        } else if pattern.starts_with('*') {
            let suffix = &pattern[1..];
            text.ends_with(suffix)
        } else if pattern.ends_with('*') {
            let prefix = &pattern[..pattern.len()-1];
            text.starts_with(prefix)
        } else {
            // More complex pattern matching could be implemented here
            text == pattern
        }
    } else {
        text == pattern
    }
}
