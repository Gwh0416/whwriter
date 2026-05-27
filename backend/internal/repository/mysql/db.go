package mysql

import (
	"fmt"
	"log"

	"whwriter/backend/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.EmailVerification{},
		&model.Genre{},
		&model.Platform{},
		&model.Book{},
		&model.Chapter{},
		&model.Character{},
		&model.Hook{},
		&model.Fact{},
		&model.ChapterSummary{},
		&model.BookFoundation{},
		&model.ChapterSnapshot{},
		&model.RuntimeArtifact{},
		&model.LLMConfig{},
		&model.AgentModelRoute{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	log.Println("mysql connected and migrated")
	return db, nil
}
