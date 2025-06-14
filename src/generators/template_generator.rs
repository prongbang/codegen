// codegen/src/generators/template_generator.rs
use anyhow::{Context, Result};
use async_trait::async_trait;
use handlebars::Handlebars;
use inflector::Inflector;
use std::fs;
use std::path::PathBuf;

use super::common::CodeGenerator;
use crate::config::{Config, DatabaseConfig, LanguageConfig};
use crate::ir::DatabaseSchema;

pub struct TemplateCodeGenerator<'a> {
    handlebars: Handlebars<'a>,
    overall_config: &'a Config,
    active_db_config: &'a DatabaseConfig,
    lang_config: &'a LanguageConfig,
}

#[async_trait]
impl<'a> CodeGenerator<'a> for TemplateCodeGenerator<'a> {
    fn new(
        overall_config: &'a Config,
        active_db_config: &'a DatabaseConfig,
        lang_config: &'a LanguageConfig,
    ) -> Result<Self> {
        let mut handlebars = Handlebars::new();
        handlebars.register_escape_fn(handlebars::no_escape);

        // Register common helpers for case conversion
        handlebars.register_helper(
            "to_snake_case",
            Box::new(
                |h: &handlebars::Helper,
                 _: &handlebars::Handlebars,
                 _: &handlebars::Context,
                 _: &mut handlebars::RenderContext,
                 out: &mut dyn handlebars::Output| {
                    let param = h.param(0).unwrap();
                    let value = param.value().as_str().unwrap_or("");
                    out.write(&value.to_snake_case())?;
                    Ok(())
                },
            ),
        );
        handlebars.register_helper(
            "to_pascal_case",
            Box::new(
                |h: &handlebars::Helper,
                 _: &handlebars::Handlebars,
                 _: &handlebars::Context,
                 _: &mut handlebars::RenderContext,
                 out: &mut dyn handlebars::Output| {
                    let param = h.param(0).unwrap();
                    let value = param.value().as_str().unwrap_or("");
                    out.write(&value.to_pascal_case())?;
                    Ok(())
                },
            ),
        );
        handlebars.register_helper(
            "to_camel_case",
            Box::new(
                |h: &handlebars::Helper,
                 _: &handlebars::Handlebars,
                 _: &handlebars::Context,
                 _: &mut handlebars::RenderContext,
                 out: &mut dyn handlebars::Output| {
                    let param = h.param(0).unwrap();
                    let value = param.value().as_str().unwrap_or("");
                    out.write(&value.to_camel_case())?;
                    Ok(())
                },
            ),
        );
        handlebars.register_helper(
            "to_kebab_case",
            Box::new(
                |h: &handlebars::Helper,
                 _: &handlebars::Handlebars,
                 _: &handlebars::Context,
                 _: &mut handlebars::RenderContext,
                 out: &mut dyn handlebars::Output| {
                    let param = h.param(0).unwrap();
                    let value = param.value().as_str().unwrap_or("");
                    out.write(&value.to_kebab_case())?;
                    Ok(())
                },
            ),
        );
        handlebars.register_helper(
            "to_screaming_snake_case",
            Box::new(
                |h: &handlebars::Helper,
                 _: &handlebars::Handlebars,
                 _: &handlebars::Context,
                 _: &mut handlebars::RenderContext,
                 out: &mut dyn handlebars::Output| {
                    let param = h.param(0).unwrap();
                    let value = param.value().as_str().unwrap_or("");
                    out.write(&value.to_screaming_snake_case())?;
                    Ok(())
                },
            ),
        );

        // Register helper to check if a slice contains a string (useful for imports)
        handlebars.register_helper(
            "contains",
            Box::new(
                |h: &handlebars::Helper,
                 _: &handlebars::Handlebars,
                 _: &handlebars::Context,
                 _: &mut handlebars::RenderContext,
                 out: &mut dyn handlebars::Output| {
                    let list = h.param(0).unwrap().value();
                    let search_val = h.param(1).unwrap().value();
                    let result = list
                        .as_array()
                        .map_or(false, |arr| arr.iter().any(|item| item == search_val));
                    out.write(&result.to_string())?;
                    Ok(())
                },
            ),
        );

        // Register helper for conditional output based on language
        handlebars.register_helper(
            "if_lang",
            Box::new(
                |h: &handlebars::Helper,
                 _: &handlebars::Handlebars,
                 ctx: &handlebars::Context,
                 _: &mut handlebars::RenderContext,
                 out: &mut dyn handlebars::Output| {
                    let target_lang = h.param(0).unwrap().value().as_str().unwrap_or("");
                    let current_lang = ctx
                        .data()
                        .get("CurrentLanguage")
                        .and_then(|v| v.as_str())
                        .unwrap_or("");
                    let content = h.param(1).unwrap().value().as_str().unwrap_or("");

                    if target_lang == current_lang {
                        out.write(content)?;
                    }
                    Ok(())
                },
            ),
        );

        // Register helper for pluralization
        handlebars.register_helper(
            "pluralize",
            Box::new(
                |h: &handlebars::Helper,
                 _: &handlebars::Handlebars,
                 _: &handlebars::Context,
                 _: &mut handlebars::RenderContext,
                 out: &mut dyn handlebars::Output| {
                    let word = h.param(0).unwrap().value().as_str().unwrap_or("");
                    let plural = if word.ends_with('y') {
                        format!("{}ies", &word[..word.len() - 1])
                    } else if word.ends_with("s") || word.ends_with("sh") || word.ends_with("ch") {
                        format!("{}es", word)
                    } else {
                        format!("{}s", word)
                    };
                    out.write(&plural)?;
                    Ok(())
                },
            ),
        );

        // Register helper for first letter uppercase
        handlebars.register_helper(
            "capitalize",
            Box::new(
                |h: &handlebars::Helper,
                 _: &handlebars::Handlebars,
                 _: &handlebars::Context,
                 _: &mut handlebars::RenderContext,
                 out: &mut dyn handlebars::Output| {
                    let text = h.param(0).unwrap().value().as_str().unwrap_or("");
                    let capitalized = text
                        .chars()
                        .enumerate()
                        .map(|(i, c)| {
                            if i == 0 {
                                c.to_uppercase().collect::<String>()
                            } else {
                                c.to_string()
                            }
                        })
                        .collect::<String>();
                    out.write(&capitalized)?;
                    Ok(())
                },
            ),
        );

        // Register helper for lowercasing first letter
        handlebars.register_helper(
            "uncapitalize",
            Box::new(
                |h: &handlebars::Helper,
                 _: &handlebars::Handlebars,
                 _: &handlebars::Context,
                 _: &mut handlebars::RenderContext,
                 out: &mut dyn handlebars::Output| {
                    let text = h.param(0).unwrap().value().as_str().unwrap_or("");
                    let uncapitalized = text
                        .chars()
                        .enumerate()
                        .map(|(i, c)| {
                            if i == 0 {
                                c.to_lowercase().collect::<String>()
                            } else {
                                c.to_string()
                            }
                        })
                        .collect::<String>();
                    out.write(&uncapitalized)?;
                    Ok(())
                },
            ),
        );

        // Determine which template to use based on config
        if let Some(custom_template_path) = &lang_config.template_path {
            // Use explicitly specified custom template path
            let template_path = PathBuf::from(custom_template_path);
            handlebars
                .register_template_file("main_template", &template_path)
                .with_context(|| {
                    format!("Failed to load custom template from {:?}", template_path)
                })?;
        } else if let Some(template_file) = &lang_config.template_file {
            // Check if template exists in the configured template directory
            let template_path = overall_config.generation.template_dir.join(template_file);

            if template_path.exists() {
                // Use template from template directory
                handlebars
                    .register_template_file("main_template", &template_path)
                    .with_context(|| format!("Failed to load template from {:?}", template_path))?;
            } else {
                // Fall back to built-in template
                let built_in_template = Self::get_built_in_template(template_file)?;
                handlebars
                    .register_template_string("main_template", built_in_template)
                    .with_context(|| {
                        format!(
                            "Failed to register built-in template for file '{}'",
                            template_file
                        )
                    })?;
            }
        } else {
            // No template_file specified, need to determine output extension first
            // Get language name from main config
            let lang_name = overall_config
                .languages
                .iter()
                .find(|(_, config)| *config == lang_config)
                .map(|(name, _)| name.clone())
                .unwrap_or_else(|| "unknown".to_string());

            // Get output extension (auto-detect if not specified)
            let output_extension = if let Some(ext) = &lang_config.output_extension {
                ext.clone()
            } else {
                Self::get_default_output_extension(&lang_name)?
            };

            // Auto-detect template based on extension
            let auto_template_file = Self::get_default_template_file(&output_extension)?;
            let built_in_template = Self::get_built_in_template(&auto_template_file)?;
            handlebars
                .register_template_string("main_template", built_in_template)
                .with_context(|| {
                    format!(
                        "Failed to register auto-detected built-in template for language '{}'",
                        lang_name
                    )
                })?;
        }

        Ok(TemplateCodeGenerator {
            handlebars,
            overall_config,
            active_db_config,
            lang_config,
        })
    }

    async fn generate_code(&self, schema: &DatabaseSchema, output_dir: &PathBuf) -> Result<()> {
        // Find the language name from the config
        let lang_name_for_mapping = self
            .overall_config
            .languages
            .iter()
            .find(|(_, config)| config == &self.lang_config)
            .map(|(name, _)| name.clone())
            .unwrap_or_else(|| "unknown".to_string());

        // Determine output extension (auto-detect if not specified)
        let output_extension = if let Some(ext) = &self.lang_config.output_extension {
            ext.clone()
        } else {
            Self::get_default_output_extension(&lang_name_for_mapping)?
        };

        fs::create_dir_all(&output_dir)
            .with_context(|| format!("Failed to create output directory: {:?}", output_dir))?;

        for table in &schema.tables {
            // Filter tables based on configuration patterns
            if !self.overall_config.should_include_table(&table.name) {
                println!("Skipping table '{}' due to filter patterns", table.name);
                continue;
            }
            let struct_name_case_fn = self
                .lang_config
                .struct_name_case
                .as_deref()
                .unwrap_or(&self.overall_config.naming_conventions.table_to_struct_case);
            let struct_name = apply_case_conversion(&table.name, struct_name_case_fn);

            let mut columns_data_for_template = Vec::new(); // Declare this outside the loop

            use serde_json::json;
            let mut template_data = json!({
                "StructName": struct_name,
                "TableName": table.name,
                "PackageName": self.lang_config.package_name.as_deref().unwrap_or("models"),
                "Columns": serde_json::Value::Array(vec![]),
                "Imports": self.lang_config.default_imports.clone().unwrap_or_default(),
                "CurrentLanguage": lang_name_for_mapping,
                "Config": {
                    "nullable_strategy": self.lang_config.nullable_strategy,
                    "field_prefix": self.lang_config.field_prefix,
                    "struct_name_case": self.lang_config.struct_name_case,
                    "field_name_case": self.lang_config.field_name_case,
                },
            });

            let mut dynamic_imports = Vec::new();

            for column in &table.columns {
                let raw_lang_type = self
                    .overall_config
                    .get_language_type(
                        &self.active_db_config.db_type,
                        &column.database_type,
                        &column.generic_type,
                        &lang_name_for_mapping,
                    )
                    .unwrap_or_else(|| {
                        println!(
                            "Warning: No type mapping found for {}.{} in {}, using fallback",
                            self.active_db_config.db_type,
                            column.database_type,
                            lang_name_for_mapping
                        );
                        match lang_name_for_mapping.as_str() {
                            "rust" => "String".to_string(),
                            "typescript" => "string".to_string(),
                            "python" => "str".to_string(),
                            "java" => "String".to_string(),
                            "csharp" => "string".to_string(),
                            "go" => "string".to_string(),
                            "php" => "string".to_string(),
                            "ruby" => "String".to_string(),
                            _ => "any".to_string(),
                        }
                    });

                let effective_lang_type = if column.is_nullable {
                    match self.lang_config.nullable_strategy.as_str() {
                        "pointer" => format!("*{}", raw_lang_type),
                        "option" => format!("Option<{}>", raw_lang_type),
                        "nullable_type" => {
                            // C#, Java, PHP primitives
                            match lang_name_for_mapping.as_str() {
                                "go" => {
                                    if raw_lang_type.starts_with("time.") {
                                        dynamic_imports.push("time".to_string());
                                        dynamic_imports.push("database/sql".to_string());
                                        "sql.NullTime".to_string()
                                    } else {
                                        dynamic_imports.push("database/sql".to_string());
                                        match raw_lang_type.as_str() {
                                            "string" => "sql.NullString".to_string(),
                                            "int64" => "sql.NullInt64".to_string(),
                                            "float64" => "sql.NullFloat64".to_string(),
                                            "bool" => "sql.NullBool".to_string(),
                                            _ => format!("*{}", raw_lang_type),
                                        }
                                    }
                                }
                                "csharp" => match raw_lang_type.as_str() {
                                    "string" => "string?".to_string(),
                                    _ => format!("{}?", raw_lang_type),
                                },
                                "java" => {
                                    match raw_lang_type.as_str() {
                                        "int" => "Integer".to_string(),
                                        "long" => "Long".to_string(),
                                        "boolean" => "Boolean".to_string(),
                                        "float" => "Float".to_string(),
                                        "double" => "Double".to_string(),
                                        _ => raw_lang_type.to_string(), // Other types like String are already objects/nullable
                                    }
                                }
                                "php" => {
                                    // PHP 7.4+ nullable types
                                    format!("?{}", raw_lang_type)
                                }
                                "kotlin" => format!("{}?", raw_lang_type), // Kotlin nullable types
                                "swift" => format!("{}?", raw_lang_type),  // Swift optional types
                                "dart" => format!("{}?", raw_lang_type),   // Dart nullable types
                                _ => format!("{} | null", raw_lang_type), // Default fallback for languages not specifically handled
                            }
                        }
                        "union" => format!("{} | null", raw_lang_type), // For TypeScript/JavaScript
                        "optional_property" => raw_lang_type.to_string(), // For TypeScript, handled by '?' in template
                        "optional_type" => {
                            // For Python Optional[Type] and Zig ?Type
                            match lang_name_for_mapping.as_str() {
                                "python" => {
                                    // Only add import if not already present
                                    if !dynamic_imports.contains(&"typing.Optional".to_string()) {
                                        dynamic_imports.push("typing.Optional".to_string());
                                    }
                                    format!("Optional[{}]", raw_lang_type)
                                }
                                "zig" => format!("?{}", raw_lang_type),
                                "nim" => format!("Option[{}]", raw_lang_type),
                                "haskell" => format!("Maybe {}", raw_lang_type),
                                "ocaml" => format!("{} option", raw_lang_type),
                                _ => raw_lang_type.to_string(),
                            }
                        }
                        "native" => raw_lang_type.to_string(), // For Ruby where `nil` is handled natively
                        _ => format!("{}", raw_lang_type),     // Generic fallback
                    }
                } else {
                    raw_lang_type
                };

                let field_name_case_fn = self
                    .lang_config
                    .field_name_case
                    .as_deref()
                    .unwrap_or(&self.overall_config.naming_conventions.column_to_field_case);

                let mut field_name = apply_case_conversion(&column.name, field_name_case_fn);
                if let Some(prefix) = &self.lang_config.field_prefix {
                    field_name = format!("{}{}", prefix, field_name);
                }

                let mut tags = Vec::new();
                if let Some(lang_tags_cfg) = &self.lang_config.tags {
                    for tag_template_str in lang_tags_cfg {
                        let mut tag_handlebars = Handlebars::new();
                        tag_handlebars.register_escape_fn(handlebars::no_escape);
                        tag_handlebars.register_helper(
                            "to_snake_case",
                            Box::new(
                                |h: &handlebars::Helper,
                                 _: &handlebars::Handlebars,
                                 _: &handlebars::Context,
                                 _: &mut handlebars::RenderContext,
                                 out: &mut dyn handlebars::Output| {
                                    let param = h.param(0).unwrap();
                                    let value = param.value().as_str().unwrap_or("");
                                    out.write(&value.to_snake_case())?;
                                    Ok(())
                                },
                            ),
                        );
                        tag_handlebars.register_helper(
                            "to_camel_case",
                            Box::new(
                                |h: &handlebars::Helper,
                                 _: &handlebars::Handlebars,
                                 _: &handlebars::Context,
                                 _: &mut handlebars::RenderContext,
                                 out: &mut dyn handlebars::Output| {
                                    let param = h.param(0).unwrap();
                                    let value = param.value().as_str().unwrap_or("");
                                    out.write(&value.to_camel_case())?;
                                    Ok(())
                                },
                            ),
                        );
                        tag_handlebars.register_helper(
                            "to_pascal_case",
                            Box::new(
                                |h: &handlebars::Helper,
                                 _: &handlebars::Handlebars,
                                 _: &handlebars::Context,
                                 _: &mut handlebars::RenderContext,
                                 out: &mut dyn handlebars::Output| {
                                    let param = h.param(0).unwrap();
                                    let value = param.value().as_str().unwrap_or("");
                                    out.write(&value.to_pascal_case())?;
                                    Ok(())
                                },
                            ),
                        );
                        tag_handlebars.register_helper(
                            "to_screaming_snake_case",
                            Box::new(
                                |h: &handlebars::Helper,
                                 _: &handlebars::Handlebars,
                                 _: &handlebars::Context,
                                 _: &mut handlebars::RenderContext,
                                 out: &mut dyn handlebars::Output| {
                                    let param = h.param(0).unwrap();
                                    let value = param.value().as_str().unwrap_or("");
                                    out.write(&value.to_screaming_snake_case())?;
                                    Ok(())
                                },
                            ),
                        );
                        let tag_data = json!({
                            "field_name": &column.name,
                            "column_name": &column.name,
                            "struct_name": &table.name,
                            "actual_field_name": field_name,
                            "OriginalColumnName": &column.name,
                            "FieldName": field_name,
                            "LangType": &effective_lang_type,
                            "IsNullable": column.is_nullable,
                            "DefaultValue": &column.default_value,
                        });
                        if let Ok(rendered_tag) =
                            tag_handlebars.render_template(tag_template_str, &tag_data)
                        {
                            if !rendered_tag.trim().is_empty() {
                                tags.push(rendered_tag.trim().to_string());
                            }
                        }
                    }
                }

                // Format tags based on language convention
                let final_tags_string = match lang_name_for_mapping.as_str() {
                    "go" => {
                        if tags.is_empty() {
                            "".to_string()
                        } else {
                            format!("`{}`", tags.join(" "))
                        }
                    }
                    "rust" => {
                        if tags.is_empty() {
                            "".to_string()
                        } else {
                            tags.join("\n    ")
                        }
                    }
                    "typescript" => "".to_string(), // TypeScript generally doesn't use field tags this way
                    "csharp" => {
                        // C# uses attributes - tags should include full syntax
                        if tags.is_empty() {
                            "".to_string()
                        } else {
                            tags.join("\n    ")
                        }
                    }
                    "java" => {
                        // Java uses annotations - tags should include full syntax
                        if tags.is_empty() {
                            "".to_string()
                        } else {
                            tags.join("\n    ")
                        }
                    }
                    "python" => {
                        // Python uses decorators - tags should include full syntax
                        if tags.is_empty() {
                            "".to_string()
                        } else {
                            tags.join("\n    ")
                        }
                    }
                    "php" => {
                        // PHP 8+ uses attributes - tags should include full syntax
                        if tags.is_empty() {
                            "".to_string()
                        } else {
                            tags.join("\n    ")
                        }
                    }
                    "ruby" => "".to_string(), // Ruby typically uses comments or meta-programming, not inline tags/attributes
                    _ => "".to_string(),      // Fallback for languages not explicitly handled
                };

                columns_data_for_template.push(json!({
                    "FieldName": field_name,
                    "LangType": effective_lang_type,
                    "LangTags": final_tags_string,
                    "IsNullable": column.is_nullable,
                    "OriginalColumnName": column.name,
                    "ColumnComment": column.comment,
                    "DefaultValue": column.default_value,
                    "IsPrimaryKey": column.is_primary_key,
                }));
            }
            template_data["Columns"] = serde_json::Value::Array(columns_data_for_template);

            let mut current_imports_vec: Vec<String> = template_data["Imports"]
                .as_array_mut()
                .unwrap()
                .iter()
                .filter_map(|v| v.as_str().map(|s| s.to_string()))
                .collect();
            for imp in dynamic_imports {
                if !current_imports_vec.contains(&imp) {
                    current_imports_vec.push(imp);
                }
            }
            template_data["Imports"] = serde_json::to_value(current_imports_vec)?;

            let rendered = self
                .handlebars
                .render("main_template", &template_data)
                .with_context(|| format!("Failed to render template for table: {}", table.name))?;

            let output_file_path = output_dir.join(format!(
                "{}.{}",
                table.name.to_snake_case(),
                output_extension
            ));
            fs::write(&output_file_path, rendered)
                .with_context(|| format!("Failed to write file: {:?}", output_file_path))?;
            println!("  ⏺ Generated: {:?}", output_file_path);
        }
        Ok(())
    }
}

impl<'a> TemplateCodeGenerator<'a> {
    fn get_default_template_file(output_extension: &str) -> Result<String> {
        let template_file = match output_extension.trim_start_matches('.') {
            "rs" => "rust_struct.hbs",
            "ts" => "typescript_interface.hbs",
            "go" => "go_struct.hbs",
            "py" => "python_class.hbs",
            "java" => "java_class.hbs",
            "cs" => "csharp_class.hbs",
            "php" => "php_class.hbs",
            "rb" => "ruby_class.hbs",
            "swift" => "swift_struct.hbs",
            "kt" => "kotlin_class.hbs",
            "dart" => "dart_class.hbs",
            "zig" => "zig_struct.hbs",
            "nim" => "nim_type.hbs",
            "hs" => "haskell_data.hbs",
            "ex" | "exs" => "elixir_struct.hbs",
            "cr" => "crystal_class.hbs",
            "ml" | "mli" => "ocaml_type.hbs",
            _ => anyhow::bail!("Unknown output extension: {}", output_extension),
        };
        Ok(template_file.to_string())
    }

    fn get_default_output_extension(language_name: &str) -> Result<String> {
        let extension = match language_name {
            "rust" => "rs",
            "typescript" => "ts",
            "go" => "go",
            "python" => "py",
            "java" => "java",
            "csharp" => "cs",
            "php" => "php",
            "ruby" => "rb",
            "swift" => "swift",
            "kotlin" => "kt",
            "dart" => "dart",
            "zig" => "zig",
            "nim" => "nim",
            "haskell" => "hs",
            "elixir" => "ex",
            "crystal" => "cr",
            "ocaml" => "ml",
            _ => anyhow::bail!("Unknown language: {}", language_name),
        };
        Ok(extension.to_string())
    }

    fn get_built_in_template(template_file: &str) -> Result<&'static str> {
        match template_file {
            "rust_struct.hbs" => Ok(include_str!("../../templates/rust_struct.hbs")),
            "typescript_interface.hbs" => {
                Ok(include_str!("../../templates/typescript_interface.hbs"))
            }
            "go_struct.hbs" => Ok(include_str!("../../templates/go_struct.hbs")),
            "python_class.hbs" => Ok(include_str!("../../templates/python_class.hbs")),
            "java_class.hbs" => Ok(include_str!("../../templates/java_class.hbs")),
            "csharp_class.hbs" => Ok(include_str!("../../templates/csharp_class.hbs")),
            "php_class.hbs" => Ok(include_str!("../../templates/php_class.hbs")),
            "ruby_class.hbs" => Ok(include_str!("../../templates/ruby_class.hbs")),
            "swift_struct.hbs" => Ok(include_str!("../../templates/swift_struct.hbs")),
            "kotlin_class.hbs" => Ok(include_str!("../../templates/kotlin_class.hbs")),
            "dart_class.hbs" => Ok(include_str!("../../templates/dart_class.hbs")),
            "zig_struct.hbs" => Ok(include_str!("../../templates/zig_struct.hbs")),
            "nim_type.hbs" => Ok(include_str!("../../templates/nim_type.hbs")),
            "haskell_data.hbs" => Ok(include_str!("../../templates/haskell_data.hbs")),
            "elixir_struct.hbs" => Ok(include_str!("../../templates/elixir_struct.hbs")),
            "crystal_class.hbs" => Ok(include_str!("../../templates/crystal_class.hbs")),
            "ocaml_type.hbs" => Ok(include_str!("../../templates/ocaml_type.hbs")),
            _ => anyhow::bail!("Unknown built-in template: {}", template_file),
        }
    }
}

fn apply_case_conversion(input: &str, case_type: &str) -> String {
    match case_type {
        "PascalCase" => input.to_pascal_case(),
        "camelCase" => input.to_camel_case(),
        "snake_case" => input.to_snake_case(),
        "kebab-case" => input.to_kebab_case(),
        "SCREAMING_SNAKE_CASE" => input.to_screaming_snake_case(),
        _ => input.to_string(),
    }
}
