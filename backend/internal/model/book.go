package model

import "time"

type BookStatus string

const (
	BookStatusInitializing BookStatus = "initializing"
	BookStatusOutlining    BookStatus = "outlining"
	BookStatusActive       BookStatus = "active"
	BookStatusWriting      BookStatus = "writing"
	BookStatusPaused       BookStatus = "paused"
	BookStatusCompleted    BookStatus = "completed"
)

type AutomationMode string

const (
	AutomationAuto   AutomationMode = "auto"
	AutomationSemi   AutomationMode = "semi"
	AutomationManual AutomationMode = "manual"
)

type Book struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	GenreID          uint           `json:"genre_id" gorm:"not null"`
	PlatformID       uint           `json:"platform_id" gorm:"not null"`
	LLMModelID       uint           `json:"llm_model_id" gorm:"default:0"`
	Title            string         `json:"title" gorm:"size:255;not null"`
	Description      string         `json:"description" gorm:"type:text"`
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
	ID                  uint              `json:"id" gorm:"primaryKey"`
	BookID              uint              `json:"book_id" gorm:"index;not null"`
	Name                string            `json:"name" gorm:"size:128;not null"`
	RoleType            CharacterRoleType `json:"role_type" gorm:"size:16;not null"`
	CoreTags            string            `json:"core_tags" gorm:"type:longtext"`
	ContrastDetails     string            `json:"contrast_details" gorm:"type:longtext"`
	Backstory           string            `json:"backstory" gorm:"type:longtext"`
	CharacterArc        string            `json:"character_arc" gorm:"type:longtext"`
	CurrentStatus       string            `json:"current_status" gorm:"type:longtext"`
	RelationshipNetwork string            `json:"relationship_network" gorm:"type:longtext"`
	InnerDrive          string            `json:"inner_drive" gorm:"type:longtext"`
	GrowthArc           string            `json:"growth_arc" gorm:"type:longtext"`
	Profile             string            `json:"profile" gorm:"type:longtext"`
	IsPlaceholder       bool              `json:"is_placeholder" gorm:"default:false"`
	SourceChapter       uint              `json:"source_chapter" gorm:"index"`
	LastSeenChapter     uint              `json:"last_seen_chapter" gorm:"index"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
}

type BookState struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	BookID            uint      `json:"book_id" gorm:"uniqueIndex;not null"`
	CurrentChapter    uint      `json:"current_chapter"`
	ProtagonistName   string    `json:"protagonist_name" gorm:"size:128"`
	SituationSummary  string    `json:"situation_summary" gorm:"type:longtext"`
	CurrentLocation   string    `json:"current_location" gorm:"type:text"`
	ProtagonistState  string    `json:"protagonist_state" gorm:"type:text"`
	CurrentGoal       string    `json:"current_goal" gorm:"type:text"`
	CurrentConstraint string    `json:"current_constraint" gorm:"type:text"`
	CurrentAlliances  string    `json:"current_alliances" gorm:"type:text"`
	CurrentConflict   string    `json:"current_conflict" gorm:"type:text"`
	SourceChapter     uint      `json:"source_chapter"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
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
	SubjectEntityID   *uint     `json:"subject_entity_id,omitempty" gorm:"index"`
	ObjectEntityID    *uint     `json:"object_entity_id,omitempty" gorm:"index"`
	Subject           string    `json:"subject" gorm:"index:idx_book_subject;size:128;not null"`
	Predicate         string    `json:"predicate" gorm:"size:128;not null"`
	Object            string    `json:"object" gorm:"type:text"`
	Category          string    `json:"category" gorm:"size:32;index"`
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
	BookStateJSON        string    `json:"book_state_json" gorm:"type:json"`
	FoundationsJSON      string    `json:"foundations_json" gorm:"type:json"`
	CharactersJSON       string    `json:"characters_json" gorm:"type:json"`
	FactsJSON            string    `json:"facts_json" gorm:"type:json"`
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
	ArtifactEvidence  ArtifactType = "evidence"
)

type RuntimeArtifact struct {
	ID            uint         `json:"id" gorm:"primaryKey"`
	BookID        uint         `json:"book_id" gorm:"uniqueIndex:idx_book_artifact;not null"`
	ChapterNumber uint         `json:"chapter_number" gorm:"uniqueIndex:idx_book_artifact;not null"`
	ArtifactType  ArtifactType `json:"artifact_type" gorm:"uniqueIndex:idx_book_artifact;size:16;not null"`
	Content       string       `json:"content" gorm:"type:longtext"`
	CreatedAt     time.Time    `json:"created_at"`
}
