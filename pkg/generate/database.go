package generate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/prongbang/codegen/template"
)

// NormalizeDatabases turns a raw -d/-driver value ("mariadb,influxdb3") into a
// deduplicated, ordered list of supported database names. Unknown names are
// returned separately so callers can report them instead of silently generating
// a project without the database the user asked for.
func NormalizeDatabases(raw string) (databases []string, unknown []string) {
	seen := map[string]bool{}
	supported := map[string]bool{}
	for _, db := range template.SupportedDatabases() {
		supported[db] = true
	}

	for _, part := range strings.Split(raw, ",") {
		name := strings.ToLower(strings.TrimSpace(part))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true

		// "mysql" is accepted as an alias because the CRUD generator already
		// treats it as mariadb.
		if name == "mysql" {
			name = template.DatabaseMariaDB
			if seen[name] {
				continue
			}
			seen[name] = true
		}

		if !supported[name] {
			unknown = append(unknown, name)
			continue
		}
		databases = append(databases, name)
	}

	sort.Strings(databases)
	return databases, unknown
}

// getDatabaseConfig returns the internal/database files for the selected
// databases. Shared by project creation and "codegen database init" so both
// produce an identical package.
func getDatabaseConfig(rootDir string, data template.Project) []FileConfig {
	if !data.HasDatabase() {
		return nil
	}

	configs := []FileConfig{
		{
			Path:     fmt.Sprintf("%s/internal/database/drivers.go", rootDir),
			Template: template.DatabaseDriversTemplate,
			Data:     data,
		},
		{
			Path:     fmt.Sprintf("%s/internal/database/wire.go", rootDir),
			Template: template.DatabaseWireTemplate,
			Data:     data,
		},
		{
			Path:     fmt.Sprintf("%s/internal/database/wire_gen.go", rootDir),
			Template: template.DatabaseWireGenTemplate,
			Data:     data,
		},
	}

	if data.HasMariaDB() {
		configs = append(configs,
			// db.go adapts the bun connection for the CRUD datasources, so it is
			// only meaningful when MariaDB is present.
			FileConfig{
				Path:     fmt.Sprintf("%s/internal/database/db.go", rootDir),
				Template: template.DatabaseDbTemplate,
				Data:     data,
			},
			FileConfig{
				Path:     fmt.Sprintf("%s/internal/database/mariadb.go", rootDir),
				Template: template.DatabaseMariaDBTemplate,
				Data:     data,
			},
		)
	}

	if data.HasMongoDB() {
		configs = append(configs, FileConfig{
			Path:     fmt.Sprintf("%s/internal/database/mongodb.go", rootDir),
			Template: template.DatabaseMongoDBTemplate,
			Data:     data,
		})
	}

	if data.HasInfluxDB3() {
		configs = append(configs, FileConfig{
			Path:     fmt.Sprintf("%s/internal/database/influxdb3.go", rootDir),
			Template: template.DatabaseInfluxDB3Template,
			Data:     data,
		})
	}

	return configs
}
