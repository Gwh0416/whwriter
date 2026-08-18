package sqlite

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"whwriter/backend/internal/model"

	sqliteDriver "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDB(path string) (*gorm.DB, error) {
	if path == "" {
		path = "data/whwriter.db"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite dir: %w", err)
	}

	db, err := gorm.Open(sqliteDriver.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return nil, fmt.Errorf("enable sqlite foreign_keys: %w", err)
	}
	if err := db.Exec("PRAGMA journal_mode = WAL").Error; err != nil {
		return nil, fmt.Errorf("enable sqlite WAL: %w", err)
	}
	if err := db.Exec("PRAGMA busy_timeout = 5000").Error; err != nil {
		return nil, fmt.Errorf("set sqlite busy_timeout: %w", err)
	}

	if err := migratePersonalModeSchema(db); err != nil {
		return nil, fmt.Errorf("migrate personal mode schema: %w", err)
	}

	if err := db.AutoMigrate(
		&model.Genre{},
		&model.Platform{},
		&model.Book{},
		&model.Chapter{},
		&model.Character{},
		&model.BookState{},
		&model.Hook{},
		&model.Fact{},
		&model.ChapterSummary{},
		&model.BookFoundation{},
		&model.ChapterSnapshot{},
		&model.RuntimeArtifact{},
		&model.LLMConfig{},
		&model.LLMModel{},
		&model.AgentModelRoute{},
		&model.TokenUsage{},
		&model.ChapterWriteRun{},
		&model.ChapterWriteStageRun{},
		&model.ChapterWriteBaseline{},
		&model.RadarTaxonomy{},
		&model.RadarTag{},
		&model.RadarBookSetting{},
		&model.RadarScanJob{},
		&model.RadarSource{},
		&model.RadarChapterSample{},
		&model.RadarBookProfile{},
		&model.RadarIntroSample{},
		&model.RadarTaxonomyProfile{},
		&model.RadarRule{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	log.Printf("sqlite connected and migrated: %s", path)
	return db, nil
}
