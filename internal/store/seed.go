package store

import (
	"log"

	"whwriter/pkg/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func SeedAdmin(db *gorm.DB, email, username, password string) {
	var count int64
	db.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&count)
	if count > 0 {
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("seed admin: failed to hash password: %v", err)
		return
	}

	admin := &models.User{
		Email:        email,
		Username:     username,
		PasswordHash: string(hash),
		Role:         models.RoleAdmin,
	}

	if err := db.Create(admin).Error; err != nil {
		log.Printf("seed admin: failed to create: %v", err)
		return
	}

	log.Printf("seed admin: created admin user (%s)", email)
}
