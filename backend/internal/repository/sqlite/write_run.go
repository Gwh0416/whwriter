package sqlite

import (
	"errors"

	"whwriter/backend/internal/model"

	"gorm.io/gorm"
)

func (r *truthFileRepo) CreateChapterWriteRun(run *model.ChapterWriteRun) error {
	return r.db.Create(run).Error
}

func (r *truthFileRepo) SaveChapterWriteRun(run *model.ChapterWriteRun) error {
	return r.db.Save(run).Error
}

func (r *truthFileRepo) GetChapterWriteRun(runID uint) (*model.ChapterWriteRun, error) {
	var run model.ChapterWriteRun
	if err := r.db.First(&run, runID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &run, nil
}

func (r *truthFileRepo) ListChapterWriteRuns(bookID uint, limit int) ([]model.ChapterWriteRun, error) {
	var runs []model.ChapterWriteRun
	tx := r.db.Where("book_id = ?", bookID).Order("id desc")
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	if err := tx.Find(&runs).Error; err != nil {
		return nil, err
	}
	return runs, nil
}

func (r *truthFileRepo) GetActiveChapterWriteRun(bookID uint) (*model.ChapterWriteRun, error) {
	var run model.ChapterWriteRun
	if err := r.db.Where("book_id = ? AND status IN ?", bookID, []model.ChapterWriteRunStatus{model.WriteRunQueued, model.WriteRunRunning}).
		Order("id desc").First(&run).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &run, nil
}

func (r *truthFileRepo) ListInterruptedChapterWriteRuns() ([]model.ChapterWriteRun, error) {
	var runs []model.ChapterWriteRun
	if err := r.db.Where("status IN ?", []model.ChapterWriteRunStatus{model.WriteRunQueued, model.WriteRunRunning}).Find(&runs).Error; err != nil {
		return nil, err
	}
	return runs, nil
}

func (r *truthFileRepo) CreateChapterWriteBaseline(b *model.ChapterWriteBaseline) error {
	return r.db.Create(b).Error
}

func (r *truthFileRepo) GetChapterWriteBaseline(runID uint) (*model.ChapterWriteBaseline, error) {
	var baseline model.ChapterWriteBaseline
	if err := r.db.Where("run_id = ?", runID).First(&baseline).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &baseline, nil
}

func (r *truthFileRepo) CreateChapterWriteStageRun(stage *model.ChapterWriteStageRun) error {
	return r.db.Create(stage).Error
}

func (r *truthFileRepo) SaveChapterWriteStageRun(stage *model.ChapterWriteStageRun) error {
	return r.db.Save(stage).Error
}

func (r *truthFileRepo) GetChapterWriteStages(runID uint) ([]model.ChapterWriteStageRun, error) {
	var stages []model.ChapterWriteStageRun
	if err := r.db.Where("run_id = ?", runID).Order("id").Find(&stages).Error; err != nil {
		return nil, err
	}
	return stages, nil
}

func (r *truthFileRepo) GetChapterWriteStage(runID uint, stage model.ChapterWriteStage) (*model.ChapterWriteStageRun, error) {
	var stageRun model.ChapterWriteStageRun
	if err := r.db.Where("run_id = ? AND stage = ?", runID, stage).Order("attempt desc").First(&stageRun).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &stageRun, nil
}
