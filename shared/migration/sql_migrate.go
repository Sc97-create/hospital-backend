package migration

import (
	"embed"
	"sort"
	"strings"

	"gorm.io/gorm"
)

//go:embed sql/*.up.sql
var upSQL embed.FS

// ApplySQLMigrations runs each embedded .up.sql file once, in filename order.
func ApplySQLMigrations(db *gorm.DB) error {
	if err := ensureSchemaMigrations(db); err != nil {
		return err
	}
	names, err := upFileNames()
	if err != nil {
		return err
	}
	applied, err := appliedVersions(db)
	if err != nil {
		return err
	}
	for _, name := range names {
		version := migrationVersion(name)
		if _, ok := applied[version]; ok {
			continue
		}
		if err := applyUpFile(db, name, version); err != nil {
			return err
		}
	}
	return nil
}

func ensureSchemaMigrations(db *gorm.DB) error {
	return db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version text PRIMARY KEY,
		applied_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`).Error
}

func upFileNames() ([]string, error) {
	entries, err := upSQL.ReadDir("sql")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".up.sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func appliedVersions(db *gorm.DB) (map[string]struct{}, error) {
	var versions []string
	err := db.Raw(`SELECT version FROM schema_migrations`).Scan(&versions).Error
	if err != nil {
		return nil, err
	}
	applied := make(map[string]struct{}, len(versions))
	for _, version := range versions {
		applied[version] = struct{}{}
	}
	return applied, nil
}

func applyUpFile(db *gorm.DB, name, version string) error {
	body, err := upSQL.ReadFile("sql/" + name)
	if err != nil {
		return err
	}
	return applyVersion(db, version, string(body))
}

func applyVersion(db *gorm.DB, version, script string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, stmt := range splitStatements(script) {
			if err := tx.Exec(stmt).Error; err != nil {
				return err
			}
		}
		return tx.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, version).Error
	})
}

func migrationVersion(name string) string {
	return strings.TrimSuffix(name, ".up.sql")
}

func splitStatements(script string) []string {
	var statements []string
	var current strings.Builder
	inDollar := false
	for i := 0; i < len(script); i++ {
		if strings.HasPrefix(script[i:], "$$") {
			inDollar = !inDollar
			current.WriteString("$$")
			i++
			continue
		}
		if script[i] == ';' && !inDollar {
			appendStatement(&statements, current.String())
			current.Reset()
			continue
		}
		current.WriteByte(script[i])
	}
	appendStatement(&statements, current.String())
	return statements
}

func appendStatement(statements *[]string, raw string) {
	stmt := strings.TrimSpace(raw)
	if stmt != "" {
		*statements = append(*statements, stmt)
	}
}
