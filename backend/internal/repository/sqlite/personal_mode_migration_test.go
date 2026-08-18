package sqlite

import (
	"path/filepath"
	"testing"

	sqliteDriver "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNewDBCreatesPersonalModeSchema(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "whwriter.db"))
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	defer closeGormDB(t, db)

	assertNoLegacyUserSchema(t, db)
}

func TestNewDBMigratesLegacyUserSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	legacy, err := gorm.Open(sqliteDriver.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	for _, stmt := range legacyPersonalModeSchema {
		if err := legacy.Exec(stmt).Error; err != nil {
			t.Fatalf("seed legacy schema: %v\nsql: %s", err, stmt)
		}
	}
	closeGormDB(t, legacy)

	db, err := NewDB(path)
	if err != nil {
		t.Fatalf("migrate legacy db: %v", err)
	}
	defer closeGormDB(t, db)

	assertNoLegacyUserSchema(t, db)

	var genreNames []string
	if err := db.Raw("SELECT name FROM genres ORDER BY id LIMIT 2").Scan(&genreNames).Error; err != nil {
		t.Fatalf("load migrated genres: %v", err)
	}
	if len(genreNames) != 2 || genreNames[0] != "玄幻" || genreNames[1] == "玄幻" {
		t.Fatalf("legacy duplicate genres were not normalized: %#v", genreNames)
	}

	var sourceBookIDs []string
	if err := db.Raw("SELECT source_book_id FROM radar_sources ORDER BY id").Scan(&sourceBookIDs).Error; err != nil {
		t.Fatalf("load migrated radar sources: %v", err)
	}
	if len(sourceBookIDs) != 2 || sourceBookIDs[0] != "10001" || sourceBookIDs[1] == "10001" {
		t.Fatalf("legacy duplicate radar sources were not normalized: %#v", sourceBookIDs)
	}
}

func assertNoLegacyUserSchema(t *testing.T, db *gorm.DB) {
	t.Helper()

	for _, table := range []string{"users", "email_verifications"} {
		exists, err := tableExists(db, table)
		if err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if exists {
			t.Fatalf("legacy table still exists: %s", table)
		}
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
			exists, err := columnExists(db, table, column)
			if err != nil {
				t.Fatalf("check column %s.%s: %v", table, column, err)
			}
			if exists {
				t.Fatalf("legacy column still exists: %s.%s", table, column)
			}
		}
	}
}

func closeGormDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}
}

var legacyPersonalModeSchema = []string{
	`CREATE TABLE users (
		id integer PRIMARY KEY AUTOINCREMENT,
		email text NOT NULL,
		username text NOT NULL,
		password_hash text NOT NULL,
		role text NOT NULL DEFAULT "user",
		status text NOT NULL DEFAULT "active",
		balance integer DEFAULT 0,
		created_at datetime,
		updated_at datetime
	)`,
	`CREATE UNIQUE INDEX idx_users_email ON users(email)`,
	`CREATE UNIQUE INDEX idx_users_username ON users(username)`,
	`CREATE TABLE email_verifications (
		id integer PRIMARY KEY AUTOINCREMENT,
		email text NOT NULL,
		code text NOT NULL,
		expires_at datetime NOT NULL,
		used numeric DEFAULT false,
		created_at datetime
	)`,
	`CREATE INDEX idx_email_code ON email_verifications(email, code)`,
	`CREATE TABLE genres (
		id integer PRIMARY KEY AUTOINCREMENT,
		user_id integer DEFAULT 0,
		name text NOT NULL,
		profile_markdown longtext,
		sort_order integer DEFAULT 0,
		is_active numeric DEFAULT true,
		created_at datetime,
		updated_at datetime
	)`,
	`CREATE UNIQUE INDEX idx_genre_user_name ON genres(user_id, name)`,
	`INSERT INTO genres(id, user_id, name) VALUES (1, 0, '玄幻'), (2, 1, '玄幻')`,
	`CREATE TABLE books (
		id integer PRIMARY KEY AUTOINCREMENT,
		user_id integer NOT NULL,
		genre_id integer NOT NULL,
		platform_id integer NOT NULL,
		llm_model_id integer DEFAULT 0,
		title text NOT NULL,
		description text,
		language text DEFAULT "zh",
		status text NOT NULL DEFAULT "outlining",
		chapter_word_count integer DEFAULT 3000,
		target_chapters integer DEFAULT 200,
		automation_mode text DEFAULT "semi",
		radar_category text,
		radar_tags_json text,
		created_at datetime,
		updated_at datetime
	)`,
	`CREATE INDEX idx_books_user_id ON books(user_id)`,
	`CREATE TABLE radar_book_settings (
		id integer PRIMARY KEY AUTOINCREMENT,
		user_id integer NOT NULL,
		book_id integer NOT NULL,
		platform text NOT NULL DEFAULT "fanqie",
		category text,
		tags_json text,
		created_at datetime,
		updated_at datetime
	)`,
	`CREATE UNIQUE INDEX idx_radar_book_setting ON radar_book_settings(user_id, book_id)`,
	`CREATE TABLE radar_scan_jobs (
		id integer PRIMARY KEY AUTOINCREMENT,
		user_id integer NOT NULL,
		platform text NOT NULL,
		category text NOT NULL,
		mode text NOT NULL,
		status text NOT NULL,
		cursor text,
		target_count integer,
		scanned_count integer,
		error_message text,
		started_at datetime,
		finished_at datetime,
		created_at datetime,
		updated_at datetime
	)`,
	`CREATE INDEX idx_radar_scan_jobs_user_id ON radar_scan_jobs(user_id)`,
	`CREATE TABLE radar_sources (
		id integer PRIMARY KEY AUTOINCREMENT,
		user_id integer NOT NULL,
		platform text NOT NULL,
		source_book_id text NOT NULL,
		book_url text,
		title text NOT NULL,
		author text,
		category text NOT NULL,
		tags_json text,
		intro text,
		word_count integer,
		chapter_count integer,
		status text DEFAULT "active",
		scan_job_id integer,
		confidence real,
		content_hash text,
		profile_version integer DEFAULT 0,
		created_at datetime,
		updated_at datetime
	)`,
	`CREATE UNIQUE INDEX idx_user_platform_book ON radar_sources(user_id, platform, source_book_id)`,
	`INSERT INTO radar_sources(id, user_id, platform, source_book_id, title, category) VALUES
		(1, 1, 'fanqie', '10001', '样本A', 'urban_brainhole'),
		(2, 2, 'fanqie', '10001', '样本A副本', 'urban_brainhole')`,
	`CREATE TABLE radar_book_profiles (
		id integer PRIMARY KEY AUTOINCREMENT,
		user_id integer NOT NULL,
		source_id integer NOT NULL,
		platform text NOT NULL,
		category text NOT NULL,
		tags_json text,
		profile_json longtext,
		profile_markdown longtext,
		sample_chapters integer,
		confidence real,
		version integer DEFAULT 1,
		created_at datetime,
		updated_at datetime
	)`,
	`CREATE UNIQUE INDEX idx_radar_book_profile ON radar_book_profiles(user_id, source_id, version)`,
	`CREATE TABLE radar_taxonomy_profiles (
		id integer PRIMARY KEY AUTOINCREMENT,
		user_id integer NOT NULL,
		platform text NOT NULL,
		category text NOT NULL,
		tag_key text DEFAULT "",
		profile_json longtext,
		profile_markdown longtext,
		profile_summary text,
		writer_brief text,
		planner_brief text,
		auditor_brief text,
		source_count integer,
		sample_chapter_count integer,
		confidence real,
		version integer DEFAULT 1,
		is_active numeric DEFAULT true,
		created_at datetime,
		updated_at datetime
	)`,
	`CREATE UNIQUE INDEX idx_radar_taxonomy_profile ON radar_taxonomy_profiles(user_id, platform, category, tag_key, version)`,
	`CREATE TABLE radar_rules (
		id integer PRIMARY KEY AUTOINCREMENT,
		user_id integer NOT NULL,
		platform text NOT NULL,
		category text NOT NULL,
		tag_key text DEFAULT "",
		rule_type text NOT NULL,
		title text NOT NULL,
		content text NOT NULL,
		evidence_summary text,
		confidence real,
		weight integer DEFAULT 50,
		is_active numeric DEFAULT true,
		created_at datetime,
		updated_at datetime
	)`,
	`CREATE INDEX idx_radar_rules_user_id ON radar_rules(user_id)`,
}
