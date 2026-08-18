package sqlite

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
	if book.GenreID == 0 {
		genre, err := ensureDefaultGenre(r.db)
		if err != nil {
			return err
		}
		book.GenreID = genre.ID
	}
	return r.db.Create(book).Error
}

func (r *bookRepo) List() ([]model.Book, error) {
	var books []model.Book
	err := r.db.Preload("Genre").Preload("Platform").
		Order("updated_at DESC").Find(&books).Error
	return books, err
}
