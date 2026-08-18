package model

import "time"

const (
	RadarPlatformFanqie = "fanqie"
)

type RadarJobMode string

const (
	RadarJobCategoryAuto RadarJobMode = "category_auto"
	RadarJobManualBook   RadarJobMode = "manual_book"
)

type RadarJobStatus string

const (
	RadarJobQueued    RadarJobStatus = "queued"
	RadarJobRunning   RadarJobStatus = "running"
	RadarJobSucceeded RadarJobStatus = "succeeded"
	RadarJobFailed    RadarJobStatus = "failed"
)

type RadarTaxonomy struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Platform     string    `json:"platform" gorm:"uniqueIndex:idx_radar_taxonomy;size:32;not null"`
	Category     string    `json:"category" gorm:"uniqueIndex:idx_radar_taxonomy;size:64;not null"`
	CategoryName string    `json:"category_name" gorm:"size:64;not null"`
	Description  string    `json:"description" gorm:"type:text"`
	IsActive     bool      `json:"is_active" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RadarTag struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	Platform      string    `json:"platform" gorm:"uniqueIndex:idx_radar_tag;size:32;not null"`
	PlatformTagID int64     `json:"platform_tag_id" gorm:"index"`
	Category      string    `json:"category" gorm:"index;size:64;not null"`
	TagType       string    `json:"tag_type" gorm:"size:32;index"`
	TagKey        string    `json:"tag_key" gorm:"uniqueIndex:idx_radar_tag;size:64;not null"`
	TagName       string    `json:"tag_name" gorm:"size:64;not null"`
	Description   string    `json:"description" gorm:"type:text"`
	IsActive      bool      `json:"is_active" gorm:"default:true"`
	CreatedAt     time.Time `json:"created_at"`
}

type RadarScanJob struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	Platform     string         `json:"platform" gorm:"index:idx_radar_job;size:32;not null"`
	Category     string         `json:"category" gorm:"index:idx_radar_job;size:64;not null"`
	LLMModelID   uint           `json:"llm_model_id" gorm:"default:0"`
	Mode         RadarJobMode   `json:"mode" gorm:"size:32;not null"`
	Status       RadarJobStatus `json:"status" gorm:"index:idx_radar_job;size:32;not null"`
	Cursor       string         `json:"cursor" gorm:"size:512"`
	TargetCount  int            `json:"target_count"`
	ScannedCount int            `json:"scanned_count"`
	ErrorMessage string         `json:"error_message" gorm:"type:text"`
	StartedAt    *time.Time     `json:"started_at"`
	FinishedAt   *time.Time     `json:"finished_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type RadarBookSetting struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	BookID    uint      `json:"book_id" gorm:"uniqueIndex;not null"`
	Platform  string    `json:"platform" gorm:"size:32;not null;default:fanqie"`
	Category  string    `json:"category" gorm:"size:64;index"`
	TagsJSON  string    `json:"tags_json" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RadarSource struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	Platform       string    `json:"platform" gorm:"uniqueIndex:idx_platform_book;index:idx_radar_source_category;size:32;not null"`
	SourceBookID   string    `json:"source_book_id" gorm:"uniqueIndex:idx_platform_book;size:128;not null"`
	BookURL        string    `json:"book_url" gorm:"size:1024"`
	Title          string    `json:"title" gorm:"size:255;not null"`
	Author         string    `json:"author" gorm:"size:128"`
	Category       string    `json:"category" gorm:"index:idx_radar_source_category;size:64;not null"`
	TagsJSON       string    `json:"tags_json" gorm:"type:text"`
	Intro          string    `json:"intro" gorm:"type:text"`
	WordCount      int64     `json:"word_count"`
	ChapterCount   int       `json:"chapter_count"`
	Status         string    `json:"status" gorm:"size:32;default:active"`
	ScanJobID      uint      `json:"scan_job_id" gorm:"index"`
	Confidence     float64   `json:"confidence"`
	ContentHash    string    `json:"content_hash" gorm:"size:64"`
	ProfileVersion int       `json:"profile_version" gorm:"default:0"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type RadarChapterSample struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	SourceID       uint      `json:"source_id" gorm:"uniqueIndex:idx_radar_source_chapter;index;not null"`
	ChapterNo      int       `json:"chapter_no" gorm:"uniqueIndex:idx_radar_source_chapter;not null"`
	Title          string    `json:"title" gorm:"size:255"`
	Content        string    `json:"content" gorm:"type:longtext"`
	WordCount      int       `json:"word_count"`
	ParagraphCount int       `json:"paragraph_count"`
	DialogueRatio  float64   `json:"dialogue_ratio"`
	ContentHash    string    `json:"content_hash" gorm:"size:64;not null"`
	CreatedAt      time.Time `json:"created_at"`
}

type RadarBookProfile struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	SourceID        uint      `json:"source_id" gorm:"uniqueIndex:idx_radar_book_profile;not null"`
	Platform        string    `json:"platform" gorm:"index:idx_radar_book_profile_category;size:32;not null"`
	Category        string    `json:"category" gorm:"index:idx_radar_book_profile_category;size:64;not null"`
	TagsJSON        string    `json:"tags_json" gorm:"type:text"`
	ProfileJSON     string    `json:"profile_json" gorm:"type:longtext"`
	ProfileMarkdown string    `json:"profile_markdown" gorm:"type:longtext"`
	SampleChapters  int       `json:"sample_chapters"`
	Confidence      float64   `json:"confidence"`
	Version         int       `json:"version" gorm:"uniqueIndex:idx_radar_book_profile;default:1"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type RadarIntroSample struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Platform     string    `json:"platform" gorm:"uniqueIndex:idx_radar_intro_book;index:idx_radar_intro_category;size:32;not null"`
	SourceBookID string    `json:"source_book_id" gorm:"uniqueIndex:idx_radar_intro_book;size:128;not null"`
	BookURL      string    `json:"book_url" gorm:"size:1024"`
	Title        string    `json:"title" gorm:"size:255;not null"`
	Author       string    `json:"author" gorm:"size:128"`
	Category     string    `json:"category" gorm:"index:idx_radar_intro_category;size:64;not null"`
	TagsJSON     string    `json:"tags_json" gorm:"type:text"`
	Intro        string    `json:"intro" gorm:"type:text;not null"`
	WordCount    int       `json:"word_count"`
	ContentHash  string    `json:"content_hash" gorm:"size:64;not null"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RadarTaxonomyProfile struct {
	ID                 uint      `json:"id" gorm:"primaryKey"`
	Platform           string    `json:"platform" gorm:"uniqueIndex:idx_radar_taxonomy_profile;index:idx_radar_taxonomy_profile_match;size:32;not null"`
	Category           string    `json:"category" gorm:"uniqueIndex:idx_radar_taxonomy_profile;index:idx_radar_taxonomy_profile_match;size:64;not null"`
	TagKey             string    `json:"tag_key" gorm:"uniqueIndex:idx_radar_taxonomy_profile;index:idx_radar_taxonomy_profile_match;size:64;default:''"`
	ProfileJSON        string    `json:"profile_json" gorm:"type:longtext"`
	ProfileMarkdown    string    `json:"profile_markdown" gorm:"type:longtext"`
	ProfileSummary     string    `json:"profile_summary" gorm:"type:text"`
	WriterBrief        string    `json:"writer_brief" gorm:"type:text"`
	PlannerBrief       string    `json:"planner_brief" gorm:"type:text"`
	AuditorBrief       string    `json:"auditor_brief" gorm:"type:text"`
	SourceCount        int       `json:"source_count"`
	SampleChapterCount int       `json:"sample_chapter_count"`
	Confidence         float64   `json:"confidence"`
	Version            int       `json:"version" gorm:"uniqueIndex:idx_radar_taxonomy_profile;default:1"`
	IsActive           bool      `json:"is_active" gorm:"default:true"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type RadarRule struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	Platform        string    `json:"platform" gorm:"index:idx_radar_rule_match;size:32;not null"`
	Category        string    `json:"category" gorm:"index:idx_radar_rule_match;size:64;not null"`
	TagKey          string    `json:"tag_key" gorm:"index:idx_radar_rule_match;size:64;default:''"`
	RuleType        string    `json:"rule_type" gorm:"index:idx_radar_rule_match;size:64;not null"`
	Title           string    `json:"title" gorm:"size:255;not null"`
	Content         string    `json:"content" gorm:"type:text;not null"`
	EvidenceSummary string    `json:"evidence_summary" gorm:"type:text"`
	Confidence      float64   `json:"confidence"`
	Weight          int       `json:"weight" gorm:"default:50"`
	IsActive        bool      `json:"is_active" gorm:"index:idx_radar_rule_match;default:true"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CreateRadarSourceRequest struct {
	Platform     string   `json:"platform"`
	SourceBookID string   `json:"source_book_id"`
	BookURL      string   `json:"book_url"`
	Title        string   `json:"title"`
	Author       string   `json:"author"`
	Category     string   `json:"category"`
	Tags         []string `json:"tags"`
	Intro        string   `json:"intro"`
	SampleText   string   `json:"sample_text"`
	ScanJobID    uint     `json:"scan_job_id"`
}

type CreateRadarScanJobRequest struct {
	Platform    string `json:"platform"`
	Category    string `json:"category"`
	TagKey      string `json:"tag_key"`
	TargetCount int    `json:"target_count"`
	LLMModelID  uint   `json:"model_id"`
}

type CreateRadarIntroScanRequest struct {
	Platform    string `json:"platform"`
	Category    string `json:"category"`
	TagKey      string `json:"tag_key"`
	TargetCount int    `json:"target_count"`
}

type GenerateRadarIntroRequest struct {
	Platform    string   `json:"platform"`
	Category    string   `json:"category"`
	TagKey      string   `json:"tag_key"`
	Tags        []string `json:"tags"`
	Requirement string   `json:"requirement"`
	ModelID     uint     `json:"model_id"`
}
