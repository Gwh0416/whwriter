package model

import "time"

type BookStatus string

const (
	BookStatusOutlining BookStatus = "outlining"
	BookStatusActive    BookStatus = "active"
	BookStatusPaused    BookStatus = "paused"
	BookStatusCompleted BookStatus = "completed"
)

type AutomationMode string

const (
	AutomationAuto   AutomationMode = "auto"
	AutomationSemi   AutomationMode = "semi"
	AutomationManual AutomationMode = "manual"
)

type Book struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	UserID           uint           `json:"user_id" gorm:"index;not null"`
	GenreID          uint           `json:"genre_id" gorm:"not null"`
	PlatformID       uint           `json:"platform_id" gorm:"not null"`
	Title            string         `json:"title" gorm:"size:255;not null"`
	Language         string         `json:"language" gorm:"size:8;default:zh"`
	Status           BookStatus     `json:"status" gorm:"size:16;not null;default:outlining"`
	ChapterWordCount int            `json:"chapter_word_count" gorm:"default:3000"`
	TargetChapters   int            `json:"target_chapters" gorm:"default:200"`
	AutomationMode   AutomationMode `json:"automation_mode" gorm:"size:16;default:semi"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`

	Genre    Genre    `json:"genre" gorm:"foreignKey:GenreID"`
	Platform Platform `json:"platform" gorm:"foreignKey:PlatformID"`
}

type ChapterStatus string

const (
	ChapterDraft     ChapterStatus = "draft"
	ChapterReviewed  ChapterStatus = "reviewed"
	ChapterRevised   ChapterStatus = "revised"
	ChapterPublished ChapterStatus = "published"
)

type Chapter struct {
	ID            uint          `json:"id" gorm:"primaryKey"`
	BookID        uint          `json:"book_id" gorm:"uniqueIndex:idx_book_chapter;not null"`
	ChapterNumber uint          `json:"chapter_number" gorm:"uniqueIndex:idx_book_chapter;not null"`
	Title         string        `json:"title" gorm:"size:255"`
	Content       string        `json:"content" gorm:"type:longtext"`
	WordCount     uint          `json:"word_count"`
	Status        ChapterStatus `json:"status" gorm:"size:16;default:draft"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

type CharacterRoleType string

const (
	CharacterProtagonist CharacterRoleType = "protagonist"
	CharacterMajor       CharacterRoleType = "major"
	CharacterMinor       CharacterRoleType = "minor"
)

type Character struct {
	ID        uint              `json:"id" gorm:"primaryKey"`
	BookID    uint              `json:"book_id" gorm:"index;not null"`
	Name      string            `json:"name" gorm:"size:128;not null"`
	RoleType  CharacterRoleType `json:"role_type" gorm:"size:16;not null"`
	Profile   string            `json:"profile" gorm:"type:longtext"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type HookType string

const (
	HookPlot      HookType = "plot"
	HookConflict  HookType = "conflict"
	HookItem      HookType = "item"
	HookMystery   HookType = "mystery"
	HookCharacter HookType = "character"
)

type HookStatus string

const (
	HookSeed        HookStatus = "seed"
	HookOpen        HookStatus = "open"
	HookProgressing HookStatus = "progressing"
	HookResolved    HookStatus = "resolved"
	HookDeferred    HookStatus = "deferred"
	HookStale       HookStatus = "stale"
)

type PayoffTiming string

const (
	PayoffImmediate PayoffTiming = "immediate"
	PayoffNearTerm  PayoffTiming = "near-term"
	PayoffMidTerm   PayoffTiming = "mid-term"
	PayoffSlowBurn  PayoffTiming = "slow-burn"
)

type Hook struct {
	ID                  uint         `json:"id" gorm:"primaryKey"`
	BookID              uint         `json:"book_id" gorm:"uniqueIndex:idx_book_hook;not null"`
	HookID              string       `json:"hook_id" gorm:"uniqueIndex:idx_book_hook;size:128;not null"`
	StartChapter        uint         `json:"start_chapter"`
	Type                HookType     `json:"type" gorm:"size:32"`
	Status              HookStatus   `json:"status" gorm:"size:16;default:seed"`
	LastAdvancedChapter uint         `json:"last_advanced_chapter"`
	ExpectedPayoff      string       `json:"expected_payoff" gorm:"type:text"`
	PayoffTiming        PayoffTiming `json:"payoff_timing" gorm:"size:16"`
	PayoffVolume        *uint        `json:"payoff_volume"`
	UpstreamDependency  string       `json:"upstream_dependency" gorm:"type:text"`
	IsCritical          bool         `json:"is_critical" gorm:"default:false"`
	HalfLife            *uint        `json:"half_life"`
	Notes               string       `json:"notes" gorm:"type:text"`
	CreatedAt           time.Time    `json:"created_at"`
	UpdatedAt           time.Time    `json:"updated_at"`
}

type Fact struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	BookID            uint      `json:"book_id" gorm:"index:idx_book_subject;not null"`
	Subject           string    `json:"subject" gorm:"index:idx_book_subject;size:128;not null"`
	Predicate         string    `json:"predicate" gorm:"size:128;not null"`
	Object            string    `json:"object" gorm:"type:text"`
	ValidFromChapter  uint      `json:"valid_from_chapter" gorm:"index;not null"`
	ValidUntilChapter *uint     `json:"valid_until_chapter"`
	SourceChapter     uint      `json:"source_chapter"`
	CreatedAt         time.Time `json:"created_at"`
}

type ChapterSummary struct {
	ID                 uint      `json:"id" gorm:"primaryKey"`
	BookID             uint      `json:"book_id" gorm:"uniqueIndex:idx_book_chapter_summary;not null"`
	ChapterNumber      uint      `json:"chapter_number" gorm:"uniqueIndex:idx_book_chapter_summary;not null"`
	Title              string    `json:"title" gorm:"size:255"`
	CharactersAppeared string    `json:"characters_appeared" gorm:"type:text"`
	KeyEvents          string    `json:"key_events" gorm:"type:text"`
	StateChanges       string    `json:"state_changes" gorm:"type:text"`
	HookActivity       string    `json:"hook_activity" gorm:"type:text"`
	Mood               string    `json:"mood" gorm:"size:64"`
	ChapterType        string    `json:"chapter_type" gorm:"size:64"`
	CreatedAt          time.Time `json:"created_at"`
}

type FoundationFileType string

const (
	FoundationStoryFrame   FoundationFileType = "story_frame"
	FoundationVolumeMap    FoundationFileType = "volume_map"
	FoundationBookRules    FoundationFileType = "book_rules"
	FoundationAuthorIntent FoundationFileType = "author_intent"
	FoundationStyleGuide   FoundationFileType = "style_guide"
	FoundationCurrentFocus FoundationFileType = "current_focus"
	FoundationAuditDrift   FoundationFileType = "audit_drift"
)

type BookFoundation struct {
	ID        uint               `json:"id" gorm:"primaryKey"`
	BookID    uint               `json:"book_id" gorm:"uniqueIndex:idx_book_foundation;not null"`
	FileType  FoundationFileType `json:"file_type" gorm:"uniqueIndex:idx_book_foundation;size:32;not null"`
	Content   string             `json:"content" gorm:"type:longtext"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

type ChapterSnapshot struct {
	ID                   uint      `json:"id" gorm:"primaryKey"`
	BookID               uint      `json:"book_id" gorm:"uniqueIndex:idx_book_snapshot;not null"`
	ChapterNumber        uint      `json:"chapter_number" gorm:"uniqueIndex:idx_book_snapshot;not null"`
	CurrentStateJSON     string    `json:"current_state_json" gorm:"type:json"`
	HooksJSON            string    `json:"hooks_json" gorm:"type:json"`
	ChapterSummariesJSON string    `json:"chapter_summaries_json" gorm:"type:json"`
	ManifestJSON         string    `json:"manifest_json" gorm:"type:json"`
	CreatedAt            time.Time `json:"created_at"`
}

type ArtifactType string

const (
	ArtifactIntent    ArtifactType = "intent"
	ArtifactPlan      ArtifactType = "plan"
	ArtifactContext   ArtifactType = "context"
	ArtifactRuleStack ArtifactType = "rule_stack"
	ArtifactTrace     ArtifactType = "trace"
)

type RuntimeArtifact struct {
	ID            uint         `json:"id" gorm:"primaryKey"`
	BookID        uint         `json:"book_id" gorm:"uniqueIndex:idx_book_artifact;not null"`
	ChapterNumber uint         `json:"chapter_number" gorm:"uniqueIndex:idx_book_artifact;not null"`
	ArtifactType  ArtifactType `json:"artifact_type" gorm:"uniqueIndex:idx_book_artifact;size:16;not null"`
	Content       string       `json:"content" gorm:"type:longtext"`
	CreatedAt     time.Time    `json:"created_at"`
}
