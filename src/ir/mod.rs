// codegen/src/ir/mod.rs
#[derive(Debug)]
pub struct DatabaseSchema {
    pub name: String,
    pub tables: Vec<Table>,
}

#[derive(Debug)]
pub struct Table {
    pub name: String,
    pub columns: Vec<Column>,
    // Add primary keys, foreign keys if needed
}

#[derive(Debug)]
pub struct Column {
    pub name: String,
    pub database_type: String, // e.g., "varchar", "int"
    pub generic_type: String,  // e.g., "string", "integer"
    pub is_nullable: bool,
    pub default_value: Option<String>,
    pub comment: Option<String>,
    pub is_primary_key: bool,
    // Add other relevant metadata
}
