package store

import (
	"whwriter/backend/pkg/models"

	"gorm.io/gorm"
)

type GenreStore struct {
	db *gorm.DB
}

func NewGenreStore(db *gorm.DB) *GenreStore {
	return &GenreStore{db: db}
}

func (s *GenreStore) ListBuiltin() ([]models.Genre, error) {
	var genres []models.Genre
	err := s.db.Where("user_id = 0 AND is_active = ?", true).Order("sort_order").Find(&genres).Error
	return genres, err
}

func (s *GenreStore) ListByUser(userID uint) ([]models.Genre, error) {
	var genres []models.Genre
	err := s.db.Where("(user_id = 0 OR user_id = ?) AND is_active = ?", userID, true).
		Order("sort_order").Find(&genres).Error
	return genres, err
}
