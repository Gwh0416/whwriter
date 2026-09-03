package model

import "time"

type WikiEntityType string

const (
	WikiEntityCharacter    WikiEntityType = "character"
	WikiEntityPlace        WikiEntityType = "place"
	WikiEntityItem         WikiEntityType = "item"
	WikiEntityOrganization WikiEntityType = "organization"
	WikiEntityEvent        WikiEntityType = "event"
	WikiEntityHook         WikiEntityType = "hook"
	WikiEntityRule         WikiEntityType = "rule"
	WikiEntityConcept      WikiEntityType = "concept"
)

type WikiEntityStatus string

const (
	WikiEntityActive   WikiEntityStatus = "active"
	WikiEntityInactive WikiEntityStatus = "inactive"
	WikiEntityUnknown  WikiEntityStatus = "unknown"
)

type WikiEntity struct {
	ID               uint             `json:"id" gorm:"primaryKey"`
	BookID           uint             `json:"book_id" gorm:"uniqueIndex:idx_wiki_entity_name;index;not null"`
	EntityType       WikiEntityType   `json:"entity_type" gorm:"uniqueIndex:idx_wiki_entity_name;index;size:32;not null"`
	CanonicalName    string           `json:"canonical_name" gorm:"size:255;not null"`
	NormalizedName   string           `json:"-" gorm:"uniqueIndex:idx_wiki_entity_name;size:255;not null"`
	Summary          string           `json:"summary" gorm:"type:longtext"`
	Status           WikiEntityStatus `json:"status" gorm:"size:16;index;default:active"`
	FirstSeenChapter uint             `json:"first_seen_chapter" gorm:"index"`
	LastSeenChapter  uint             `json:"last_seen_chapter" gorm:"index"`
	MetadataJSON     string           `json:"metadata_json" gorm:"type:json"`
	Managed          bool             `json:"managed" gorm:"index"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

type WikiEntityAlias struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	BookID          uint           `json:"book_id" gorm:"index;not null"`
	EntityID        uint           `json:"entity_id" gorm:"uniqueIndex:idx_wiki_entity_alias;index;not null"`
	EntityType      WikiEntityType `json:"entity_type" gorm:"index;size:32;not null"`
	Alias           string         `json:"alias" gorm:"size:255;not null"`
	NormalizedAlias string         `json:"-" gorm:"uniqueIndex:idx_wiki_entity_alias;index;size:255;not null"`
	IsCanonical     bool           `json:"is_canonical" gorm:"default:false"`
	IsDerived       bool           `json:"is_derived" gorm:"index"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type WikiEntitySource struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	BookID        uint      `json:"book_id" gorm:"index;not null"`
	EntityID      uint      `json:"entity_id" gorm:"uniqueIndex:idx_wiki_entity_source;index;not null"`
	SourceType    string    `json:"source_type" gorm:"uniqueIndex:idx_wiki_entity_source;index;size:32;not null"`
	SourceID      string    `json:"source_id" gorm:"uniqueIndex:idx_wiki_entity_source;size:128;not null"`
	SourceChapter uint      `json:"source_chapter" gorm:"index"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type WikiRelation struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	BookID            uint      `json:"book_id" gorm:"uniqueIndex:idx_wiki_relation_source;index;not null"`
	SubjectEntityID   uint      `json:"subject_entity_id" gorm:"index;not null"`
	Predicate         string    `json:"predicate" gorm:"size:128;index;not null"`
	ObjectEntityID    *uint     `json:"object_entity_id,omitempty" gorm:"index"`
	ObjectLiteral     string    `json:"object_literal,omitempty" gorm:"type:text"`
	QualifierJSON     string    `json:"qualifier_json" gorm:"type:json"`
	ValidFromChapter  uint      `json:"valid_from_chapter" gorm:"index;not null"`
	ValidUntilChapter *uint     `json:"valid_until_chapter,omitempty" gorm:"index"`
	SourceChapter     uint      `json:"source_chapter" gorm:"index"`
	SourceType        string    `json:"source_type" gorm:"uniqueIndex:idx_wiki_relation_source;size:32;not null"`
	SourceID          string    `json:"source_id" gorm:"uniqueIndex:idx_wiki_relation_source;size:128;not null"`
	Confidence        float64   `json:"confidence" gorm:"default:1"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type WikiRelationView struct {
	WikiRelation
	SubjectName string         `json:"subject_name"`
	SubjectType WikiEntityType `json:"subject_type"`
	ObjectName  string         `json:"object_name,omitempty"`
	ObjectType  WikiEntityType `json:"object_type,omitempty"`
}
