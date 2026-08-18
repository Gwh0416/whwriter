package sqlite

import (
	"log"

	"whwriter/backend/internal/model"

	"gorm.io/gorm"
)

const defaultGenreName = "番茄标签写作"

func SeedGenres(db *gorm.DB) {
	genre, err := ensureDefaultGenre(db)
	if err != nil {
		log.Printf("seed genres: ensure default genre failed: %v", err)
		return
	}
	cleanupLegacyGenres(db, genre.ID)
	log.Printf("seed genres: kept default genre %q", defaultGenreName)
}

func ensureDefaultGenre(db *gorm.DB) (*model.Genre, error) {
	genre := model.Genre{
		Name:            defaultGenreName,
		ProfileMarkdown: defaultGenreProfile(),
		SortOrder:       1,
		IsActive:        true,
	}
	var existing model.Genre
	if tx := db.Where("name = ?", genre.Name).Limit(1).Find(&existing); tx.Error == nil && tx.RowsAffected > 0 {
		if err := db.Model(&existing).Updates(map[string]any{
			"profile_markdown": genre.ProfileMarkdown,
			"sort_order":       genre.SortOrder,
			"is_active":        true,
		}).Error; err != nil {
			return nil, err
		}
		existing.ProfileMarkdown = genre.ProfileMarkdown
		existing.IsActive = true
		return &existing, nil
	}
	if err := db.Create(&genre).Error; err != nil {
		return nil, err
	}
	return &genre, nil
}

func cleanupLegacyGenres(db *gorm.DB, defaultGenreID uint) {
	if defaultGenreID == 0 {
		return
	}
	_ = db.Model(&model.Book{}).Where("genre_id <> ?", defaultGenreID).Update("genre_id", defaultGenreID).Error
	_ = db.Where("id <> ?", defaultGenreID).Delete(&model.Genre{}).Error
}

func defaultGenreProfile() string {
	return `---
name: ` + defaultGenreName + `
平台分类: 番茄小说
---

## 分类定位

本项目不再维护独立题材模板。书籍写法由创建书籍时选择的番茄官方标签决定，雷达会按标签加载画像和规则。

## 写作指导

- 题材判断以番茄官方标签为准。
- 写作时优先加载书籍绑定标签下的雷达画像和规则。
- 本默认题材仅用于兼容旧写作链路中的 Genre 字段。`
}

func SeedPlatforms(db *gorm.DB) {
	var count int64
	db.Model(&model.Platform{}).Count(&count)
	if count > 0 {
		return
	}

	platforms := []model.Platform{
		{Name: "番茄小说", SortOrder: 1, StyleGuide: "免费阅读平台，偏好快节奏爽文。开头必须有强钩子，每章结尾留悬念。节奏紧凑，三章内必有明确反馈（打脸/升级/收益兑现）。章节字数 2000-3000 为宜。"},
		{Name: "起点中文网", SortOrder: 2, StyleGuide: "付费阅读平台，偏好精品长篇。世界观设定需详实，修炼体系需严谨。节奏可稍慢但需有持续吸引力。章节字数 3000-5000 为宜。注重逻辑自洽和设定深度。"},
		{Name: "纵横中文网", SortOrder: 3, StyleGuide: "男频为主，偏好传统玄幻和都市题材。注重战斗场面和升级体系。章节字数 3000-4000 为宜。"},
		{Name: "QQ阅读", SortOrder: 4, StyleGuide: "移动端为主，偏好轻松易读的内容。章节不宜过长，2000-3000 字为宜。开头节奏要快，对话比例可适当提高。"},
		{Name: "书旗小说", SortOrder: 5, StyleGuide: "免费+付费混合模式。内容风格介于番茄和起点之间。章节字数 2500-3500 为宜。"},
		{Name: "其他", SortOrder: 6, StyleGuide: "通用平台，根据具体发布渠道调整。"},
	}

	for _, p := range platforms {
		if err := db.Create(&p).Error; err != nil {
			log.Printf("seed platforms: failed to create %s: %v", p.Name, err)
		}
	}

	log.Printf("seed platforms: created %d built-in platforms", len(platforms))
}
