package models

type CreateBookRequest struct {
	Title            string `json:"title" binding:"required,min=1,max=255"`
	GenreID          uint   `json:"genre_id" binding:"required"`
	PlatformID       uint   `json:"platform_id" binding:"required"`
	ChapterWordCount int    `json:"chapter_word_count" binding:"required,min=500,max=50000"`
	TargetChapters   int    `json:"target_chapters" binding:"required,min=1,max=10000"`
	Description      string `json:"description" binding:"max=5000"`
	LLMConfigID      uint   `json:"llm_config_id"`
}
