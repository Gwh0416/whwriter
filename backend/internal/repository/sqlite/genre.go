package sqlite

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

func (r *genreRepo) FindByID(id uint) (*model.Genre, error) {
	var genre model.Genre
	err := r.db.First(&genre, id).Error
	if err != nil {
		return nil, err
	}
	return &genre, nil
}

func (r *genreRepo) ListAll() ([]model.Genre, error) {
	var genres []model.Genre
	err := r.db.Where("is_active = ?", true).Order("sort_order").Find(&genres).Error
	return genres, err
}

func (r *genreRepo) Create(genre *model.Genre) error {
	return r.db.Create(genre).Error
}

func (r *genreRepo) Update(genre *model.Genre) error {
	return r.db.Save(genre).Error
}

func (r *genreRepo) Delete(id uint) error {
	return r.db.Delete(&model.Genre{}, id).Error
}
