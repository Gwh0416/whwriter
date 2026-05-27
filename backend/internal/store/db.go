package store

import (
	"fmt"
	"log"

	"whwriter/backend/pkg/models"

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
		&models.User{},
		&models.EmailVerification{},
		&models.Genre{},
		&models.Platform{},
		&models.Book{},
		&models.Chapter{},
		&models.Character{},
		&models.Hook{},
		&models.Fact{},
		&models.ChapterSummary{},
		&models.BookFoundation{},
		&models.ChapterSnapshot{},
		&models.RuntimeArtifact{},
		&models.Prompt{},
		&models.LLMConfig{},
		&models.AgentModelRoute{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	log.Println("mysql connected and migrated")
	return db, nil
}
