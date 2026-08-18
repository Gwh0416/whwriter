package sqlite

import (
	"errors"

	"whwriter/backend/internal/model"
	"whwriter/backend/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type radarRepo struct {
	db *gorm.DB
}

func NewRadarRepo(db *gorm.DB) repository.RadarRepository {
	return &radarRepo{db: db}
}

func (r *radarRepo) ListTaxonomies(platform string) ([]model.RadarTaxonomy, error) {
	var rows []model.RadarTaxonomy
	err := r.db.Where("platform = ? AND is_active = ?", platform, true).Order("id").Find(&rows).Error
	return rows, err
}

func (r *radarRepo) ListTags(platform, category string) ([]model.RadarTag, error) {
	tx := r.db.Where("platform = ? AND is_active = ?", platform, true)
	if category != "" {
		tx = tx.Where("category = ?", category)
	}
	var rows []model.RadarTag
	err := tx.Order("category, tag_name, id").Find(&rows).Error
	return rows, err
}

func (r *radarRepo) SaveTags(tags []model.RadarTag) error {
	if len(tags) == 0 {
		return nil
	}
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "platform"}, {Name: "tag_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"platform_tag_id", "category", "tag_type", "tag_name", "description", "is_active",
		}),
	}).Create(&tags).Error
}

func (r *radarRepo) SaveBookSetting(setting *model.RadarBookSetting) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "book_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"platform", "category", "tags_json", "updated_at"}),
	}).Create(setting).Error
}

func (r *radarRepo) GetBookSetting(bookID uint) (*model.RadarBookSetting, error) {
	var setting model.RadarBookSetting
	err := r.db.Where("book_id = ?", bookID).First(&setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &setting, err
}

func (r *radarRepo) CreateScanJob(job *model.RadarScanJob) error {
	return r.db.Create(job).Error
}

func (r *radarRepo) SaveScanJob(job *model.RadarScanJob) error {
	return r.db.Save(job).Error
}

func (r *radarRepo) GetScanJob(jobID uint) (*model.RadarScanJob, error) {
	var job model.RadarScanJob
	err := r.db.Where("id = ?", jobID).First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &job, err
}

func (r *radarRepo) ListScanJobs(limit int) ([]model.RadarScanJob, error) {
	var rows []model.RadarScanJob
	tx := r.db.Order("id desc")
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	err := tx.Find(&rows).Error
	return rows, err
}

func (r *radarRepo) DeleteScanJob(jobID uint) error {
	return r.db.Delete(&model.RadarScanJob{}, jobID).Error
}

func (r *radarRepo) SaveSource(source *model.RadarSource) error {
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "platform"}, {Name: "source_book_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"book_url", "title", "author", "category", "tags_json", "intro",
			"word_count", "chapter_count", "status", "scan_job_id", "confidence",
			"content_hash", "profile_version", "updated_at",
		}),
	}).Create(source).Error
}

func (r *radarRepo) GetSource(sourceID uint) (*model.RadarSource, error) {
	var source model.RadarSource
	err := r.db.Where("id = ?", sourceID).First(&source).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &source, err
}

func (r *radarRepo) FindSourceByBookID(platform, sourceBookID string) (*model.RadarSource, error) {
	var source model.RadarSource
	err := r.db.Where("platform = ? AND source_book_id = ?", platform, sourceBookID).First(&source).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &source, err
}

func (r *radarRepo) ListSources(limit int) ([]model.RadarSource, error) {
	var rows []model.RadarSource
	tx := r.db.Order("updated_at desc, id desc")
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	err := tx.Find(&rows).Error
	return rows, err
}

func (r *radarRepo) ListSourcesByCategory(platform, category string, limit int) ([]model.RadarSource, error) {
	var rows []model.RadarSource
	tx := r.db.Where(
		"platform = ? AND (category = ? OR tags_json LIKE ?)",
		platform, category, jsonStringContainsPattern(category),
	).Order("updated_at desc, id desc")
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	err := tx.Find(&rows).Error
	return rows, err
}

func (r *radarRepo) DeleteSourceCascade(sourceID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("source_id = ?", sourceID).Delete(&model.RadarChapterSample{}).Error; err != nil {
			return err
		}
		if err := tx.Where("source_id = ?", sourceID).Delete(&model.RadarBookProfile{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.RadarSource{}, sourceID).Error
	})
}

func (r *radarRepo) DeleteSourcesCascade(sourceIDs []uint) error {
	if len(sourceIDs) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("source_id IN ?", sourceIDs).Delete(&model.RadarChapterSample{}).Error; err != nil {
			return err
		}
		if err := tx.Where("source_id IN ?", sourceIDs).Delete(&model.RadarBookProfile{}).Error; err != nil {
			return err
		}
		return tx.Where("id IN ?", sourceIDs).Delete(&model.RadarSource{}).Error
	})
}

func (r *radarRepo) SaveChapterSamples(samples []model.RadarChapterSample) error {
	if len(samples) == 0 {
		return nil
	}
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "source_id"}, {Name: "chapter_no"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"title", "content", "word_count", "paragraph_count", "dialogue_ratio", "content_hash",
		}),
	}).Create(&samples).Error
}

func (r *radarRepo) GetChapterSamples(sourceID uint, limit int) ([]model.RadarChapterSample, error) {
	var rows []model.RadarChapterSample
	tx := r.db.Where("source_id = ?", sourceID).Order("chapter_no")
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	err := tx.Find(&rows).Error
	return rows, err
}

func (r *radarRepo) SaveBookProfile(profile *model.RadarBookProfile) error {
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "source_id"}, {Name: "version"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"platform", "category", "tags_json", "profile_json", "profile_markdown",
			"sample_chapters", "confidence", "updated_at",
		}),
	}).Create(profile).Error
}

func (r *radarRepo) ListBookProfiles(platform, category string, limit int) ([]model.RadarBookProfile, error) {
	var rows []model.RadarBookProfile
	tx := r.db.Where("platform = ?", platform)
	if category != "" {
		tx = tx.Where(
			"category = ? OR tags_json LIKE ?",
			category, jsonStringContainsPattern(category),
		)
	}
	tx = tx.Order("updated_at desc, id desc")
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	err := tx.Find(&rows).Error
	return rows, err
}

func (r *radarRepo) SaveIntroSamples(samples []model.RadarIntroSample) error {
	if len(samples) == 0 {
		return nil
	}
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "platform"}, {Name: "source_book_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"book_url", "title", "author", "category", "tags_json", "intro",
			"word_count", "content_hash", "updated_at",
		}),
	}).Create(&samples).Error
}

func (r *radarRepo) ListIntroSamples(platform, category string, limit int) ([]model.RadarIntroSample, error) {
	var rows []model.RadarIntroSample
	tx := r.db.Where("platform = ?", platform)
	if category != "" {
		tx = tx.Where(
			"category = ? OR tags_json LIKE ?",
			category, jsonStringContainsPattern(category),
		)
	}
	tx = tx.Order("updated_at desc, id desc")
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	err := tx.Find(&rows).Error
	return rows, err
}

func (r *radarRepo) DeleteIntroSamples(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Where("id IN ?", ids).Delete(&model.RadarIntroSample{}).Error
}

func (r *radarRepo) SaveTaxonomyProfile(profile *model.RadarTaxonomyProfile) error {
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "platform"}, {Name: "category"}, {Name: "tag_key"}, {Name: "version"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"profile_json", "profile_markdown", "profile_summary", "writer_brief",
			"planner_brief", "auditor_brief", "source_count", "sample_chapter_count",
			"confidence", "is_active", "updated_at",
		}),
	}).Create(profile).Error
}

func (r *radarRepo) ListTaxonomyProfiles(platform, category string) ([]model.RadarTaxonomyProfile, error) {
	var rows []model.RadarTaxonomyProfile
	tx := r.db.Where("platform = ?", platform)
	if category != "" {
		tx = tx.Where("category = ?", category)
	}
	err := tx.Order("category, tag_key, version desc").Find(&rows).Error
	return rows, err
}

func (r *radarRepo) DeleteTaxonomyProfile(profileID uint) error {
	return r.db.Delete(&model.RadarTaxonomyProfile{}, profileID).Error
}

func (r *radarRepo) DeleteTaxonomyProfilesByCategories(platform string, categories []string) error {
	if len(categories) == 0 {
		return nil
	}
	return r.db.Where("platform = ? AND category IN ?", platform, categories).Delete(&model.RadarTaxonomyProfile{}).Error
}

func (r *radarRepo) ListActiveTaxonomyProfiles(platform, category string, tags []string) ([]model.RadarTaxonomyProfile, error) {
	tagSet := uniqueRadarKeys(append([]string{category}, tags...))
	var rows []model.RadarTaxonomyProfile
	if len(tagSet) == 0 {
		return rows, nil
	}
	err := r.db.Where(
		"platform = ? AND category IN ? AND is_active = ?",
		platform, tagSet, true,
	).Order("category, version desc").Find(&rows).Error
	return rows, err
}

func (r *radarRepo) ReplaceRules(platform, category, tagKey string, rules []model.RadarRule) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("platform = ? AND category = ? AND tag_key = ?", platform, category, tagKey).
			Delete(&model.RadarRule{}).Error; err != nil {
			return err
		}
		if len(rules) == 0 {
			return nil
		}
		return tx.Create(&rules).Error
	})
}

func (r *radarRepo) ListRules(platform, category string, limit int) ([]model.RadarRule, error) {
	var rows []model.RadarRule
	tx := r.db.Where("platform = ?", platform)
	if category != "" {
		tx = tx.Where("category = ?", category)
	}
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	err := tx.Order("weight desc, confidence desc, id desc").Find(&rows).Error
	return rows, err
}

func (r *radarRepo) ListActiveRules(platform, category string, tags []string, limit int) ([]model.RadarRule, error) {
	tagSet := uniqueRadarKeys(append([]string{category}, tags...))
	var rows []model.RadarRule
	if len(tagSet) == 0 {
		return rows, nil
	}
	tx := r.db.Where(
		"platform = ? AND category IN ? AND is_active = ? AND confidence >= ?",
		platform, tagSet, true, 0.7,
	).Order("weight desc, confidence desc, id desc")
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	err := tx.Find(&rows).Error
	return rows, err
}

func (r *radarRepo) DeleteRule(ruleID uint) error {
	return r.db.Delete(&model.RadarRule{}, ruleID).Error
}

func (r *radarRepo) DeleteRulesByCategories(platform string, categories []string) error {
	if len(categories) == 0 {
		return nil
	}
	return r.db.Where("platform = ? AND category IN ?", platform, categories).Delete(&model.RadarRule{}).Error
}

func jsonStringContainsPattern(key string) string {
	return "%\"" + key + "\"%"
}

func uniqueRadarKeys(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
