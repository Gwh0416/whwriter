package sqlite

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

func migratePersonalModeSchema(db *gorm.DB) error {
	if err := db.Exec("PRAGMA foreign_keys = OFF").Error; err != nil {
		return err
	}
	defer func() {
		_ = db.Exec("PRAGMA foreign_keys = ON").Error
	}()

	if err := dropTablesIfExists(db, "users", "email_verifications"); err != nil {
		return err
	}
	if err := dropIndexesReferencingUserID(db); err != nil {
		return err
	}
	if err := dedupeGenreNames(db); err != nil {
		return err
	}
	if err := dedupeRadarSources(db); err != nil {
		return err
	}

	legacyColumns := map[string][]string{
		"books":                   {"user_id", "radar_category", "radar_tags_json"},
		"genres":                  {"user_id"},
		"radar_book_settings":     {"user_id"},
		"radar_scan_jobs":         {"user_id"},
		"radar_sources":           {"user_id"},
		"radar_book_profiles":     {"user_id"},
		"radar_taxonomy_profiles": {"user_id"},
		"radar_rules":             {"user_id"},
	}
	for table, columns := range legacyColumns {
		for _, column := range columns {
			if err := dropColumnIfExists(db, table, column); err != nil {
				return err
			}
		}
	}

	return nil
}

func dropTablesIfExists(db *gorm.DB, tables ...string) error {
	for _, table := range tables {
		if err := db.Exec("DROP TABLE IF EXISTS " + quoteSQLiteIdent(table)).Error; err != nil {
			return err
		}
	}
	return nil
}

func dropIndexesReferencingUserID(db *gorm.DB) error {
	var indexes []string
	if err := db.Raw(`
		SELECT name
		FROM sqlite_master
		WHERE type = 'index'
			AND sql IS NOT NULL
			AND lower(sql) LIKE '%user_id%'
	`).Scan(&indexes).Error; err != nil {
		return err
	}
	for _, index := range indexes {
		if err := db.Exec("DROP INDEX IF EXISTS " + quoteSQLiteIdent(index)).Error; err != nil {
			return err
		}
	}
	return nil
}

func dropColumnIfExists(db *gorm.DB, table string, column string) error {
	exists, err := tableExists(db, table)
	if err != nil || !exists {
		return err
	}
	hasColumn, err := columnExists(db, table, column)
	if err != nil || !hasColumn {
		return err
	}
	return db.Exec(fmt.Sprintf(
		"ALTER TABLE %s DROP COLUMN %s",
		quoteSQLiteIdent(table),
		quoteSQLiteIdent(column),
	)).Error
}

func tableExists(db *gorm.DB, table string) (bool, error) {
	var count int64
	if err := db.Raw(
		"SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?",
		table,
	).Scan(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func columnExists(db *gorm.DB, table string, column string) (bool, error) {
	var rows []struct {
		Name string `gorm:"column:name"`
	}
	if err := db.Raw("PRAGMA table_info(" + quoteSQLiteIdent(table) + ")").Scan(&rows).Error; err != nil {
		return false, err
	}
	for _, row := range rows {
		if row.Name == column {
			return true, nil
		}
	}
	return false, nil
}

func dedupeGenreNames(db *gorm.DB) error {
	exists, err := tableExists(db, "genres")
	if err != nil || !exists {
		return err
	}

	var rows []struct {
		ID   uint
		Name string
	}
	if err := db.Raw("SELECT id, name FROM genres ORDER BY id").Scan(&rows).Error; err != nil {
		return err
	}

	used := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			name = fmt.Sprintf("未命名题材 #%d", row.ID)
		}
		next := name
		if _, ok := used[next]; ok {
			base := name
			for {
				next = fmt.Sprintf("%s #%d", base, row.ID)
				if _, exists := used[next]; !exists {
					break
				}
				base = next
			}
		}
		used[next] = struct{}{}
		if next != row.Name {
			if err := db.Exec("UPDATE genres SET name = ? WHERE id = ?", next, row.ID).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func dedupeRadarSources(db *gorm.DB) error {
	exists, err := tableExists(db, "radar_sources")
	if err != nil || !exists {
		return err
	}

	var rows []struct {
		ID           uint
		Platform     string
		SourceBookID string `gorm:"column:source_book_id"`
	}
	if err := db.Raw("SELECT id, platform, source_book_id FROM radar_sources ORDER BY id").Scan(&rows).Error; err != nil {
		return err
	}

	used := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		key := row.Platform + "\x00" + row.SourceBookID
		if _, ok := used[key]; !ok {
			used[key] = struct{}{}
			continue
		}

		nextSourceBookID := fmt.Sprintf("%s#%d", row.SourceBookID, row.ID)
		for {
			nextKey := row.Platform + "\x00" + nextSourceBookID
			if _, exists := used[nextKey]; !exists {
				used[nextKey] = struct{}{}
				break
			}
			nextSourceBookID = fmt.Sprintf("%s#%d", nextSourceBookID, row.ID)
		}
		if err := db.Exec("UPDATE radar_sources SET source_book_id = ? WHERE id = ?", nextSourceBookID, row.ID).Error; err != nil {
			return err
		}
	}
	return nil
}

func quoteSQLiteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
