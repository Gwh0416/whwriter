package mysql

import (
	"whwriter/backend/internal/model"
	"whwriter/backend/internal/repository"

	"gorm.io/gorm"
)

type genreRepo struct {
	db *gorm.DB
}

func NewGenreRepo(db *gorm.DB) repository.GenreRepository {
	return &genreRepo{db: db}
}

func (r *genreRepo) ListBuiltin() ([]model.Genre, error) {
	var genres []model.Genre
	err := r.db.Where("user_id = 0 AND is_active = ?", true).Order("sort_order").Find(&genres).Error
	return genres, err
}

func (r *genreRepo) ListByUser(userID uint) ([]model.Genre, error) {
	var genres []model.Genre
	err := r.db.Where("(user_id = 0 OR user_id = ?) AND is_active = ?", userID, true).
		Order("sort_order").Find(&genres).Error
	return genres, err
}
