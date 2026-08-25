package model

import "time"

type KnowledgeSourceType string

const (
	KnowledgeSourceFoundation KnowledgeSourceType = "foundation"
	KnowledgeSourceCharacter  KnowledgeSourceType = "character"
	KnowledgeSourceFact       KnowledgeSourceType = "fact"
	KnowledgeSourceHook       KnowledgeSourceType = "hook"
	KnowledgeSourceSummary    KnowledgeSourceType = "summary"
	KnowledgeSourceEvidence   KnowledgeSourceType = "evidence"
)

// KnowledgeDocument is a rebuildable retrieval projection of an authoritative
// truth-file record. It is not the source of truth for book state.
type KnowledgeDocument struct {
	ID                uint                `json:"id" gorm:"primaryKey"`
	BookID            uint                `json:"book_id" gorm:"uniqueIndex:idx_knowledge_document_source;index;not null"`
	SourceType        KnowledgeSourceType `json:"source_type" gorm:"uniqueIndex:idx_knowledge_document_source;size:32;not null"`
	SourceID          string              `json:"source_id" gorm:"uniqueIndex:idx_knowledge_document_source;size:128;not null"`
	Title             string              `json:"title" gorm:"type:text"`
	Content           string              `json:"content" gorm:"type:longtext"`
	ContentHash       string              `json:"content_hash" gorm:"size:64;index;not null"`
	Importance        int                 `json:"importance" gorm:"default:3"`
	ValidFromChapter  uint                `json:"valid_from_chapter" gorm:"index"`
	ValidUntilChapter *uint               `json:"valid_until_chapter" gorm:"index"`
	IsActive          bool                `json:"is_active" gorm:"index;default:true"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
}

type KnowledgeChunk struct {
	ID         uint                `json:"id" gorm:"primaryKey"`
	DocumentID uint                `json:"document_id" gorm:"uniqueIndex:idx_knowledge_chunk;index;not null"`
	BookID     uint                `json:"book_id" gorm:"index;not null"`
	SourceType KnowledgeSourceType `json:"source_type" gorm:"size:32;index;not null"`
	ChunkIndex uint                `json:"chunk_index" gorm:"uniqueIndex:idx_knowledge_chunk;not null"`
	Content    string              `json:"content" gorm:"type:longtext"`
	SearchText string              `json:"search_text" gorm:"type:longtext"`
	TokenCount int                 `json:"token_count"`
	CreatedAt  time.Time           `json:"created_at"`
	UpdatedAt  time.Time           `json:"updated_at"`
}

type KnowledgeSearchQuery struct {
	BookID        uint
	Query         string
	ChapterNumber uint
	SourceTypes   []KnowledgeSourceType
	Limit         int
}

type KnowledgeSearchResult struct {
	ChunkID          uint                `json:"chunk_id"`
	DocumentID       uint                `json:"document_id"`
	SourceType       KnowledgeSourceType `json:"source_type"`
	SourceID         string              `json:"source_id"`
	Title            string              `json:"title"`
	Content          string              `json:"content"`
	ChunkIndex       uint                `json:"chunk_index"`
	Importance       int                 `json:"importance"`
	ValidFromChapter uint                `json:"valid_from_chapter"`
	Score            float64             `json:"score"`
}
