package store

import (
	"whwriter/backend/pkg/models"

	"gorm.io/gorm"
)

type BookStore struct {
	db *gorm.DB
}

func NewBookStore(db *gorm.DB) *BookStore {
	return &BookStore{db: db}
}

func (s *BookStore) Create(book *models.Book) error {
	return s.db.Create(book).Error
}

func (s *BookStore) ListByUser(userID uint) ([]models.Book, error) {
	var books []models.Book
	err := s.db.Where("user_id = ?", userID).
		Preload("Genre").Preload("Platform").
		Order("updated_at DESC").Find(&books).Error
	return books, err
}
