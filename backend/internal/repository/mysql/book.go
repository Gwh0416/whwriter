package mysql

import (
	"whwriter/backend/internal/model"
	"whwriter/backend/internal/repository"

	"gorm.io/gorm"
)

type bookRepo struct {
	db *gorm.DB
}

func NewBookRepo(db *gorm.DB) repository.BookRepository {
	return &bookRepo{db: db}
}

func (r *bookRepo) Create(book *model.Book) error {
	return r.db.Create(book).Error
}

func (r *bookRepo) ListByUser(userID uint) ([]model.Book, error) {
	var books []model.Book
	err := r.db.Where("user_id = ?", userID).
		Preload("Genre").Preload("Platform").
		Order("updated_at DESC").Find(&books).Error
	return books, err
}
