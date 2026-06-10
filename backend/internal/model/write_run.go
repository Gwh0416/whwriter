package model

import "time"

type ChapterWriteRunStatus string

const (
	WriteRunQueued    ChapterWriteRunStatus = "queued"
	WriteRunRunning   ChapterWriteRunStatus = "running"
	WriteRunSucceeded ChapterWriteRunStatus = "succeeded"
	WriteRunFailed    ChapterWriteRunStatus = "failed"
	WriteRunCancelled ChapterWriteRunStatus = "cancelled"
)

type ChapterWriteStageStatus string

const (
	WriteStagePending   ChapterWriteStageStatus = "pending"
	WriteStageRunning   ChapterWriteStageStatus = "running"
	WriteStageSucceeded ChapterWriteStageStatus = "succeeded"
	WriteStageFailed    ChapterWriteStageStatus = "failed"
	WriteStageCancelled ChapterWriteStageStatus = "cancelled"
	WriteStageSkipped   ChapterWriteStageStatus = "skipped"
)

type ChapterWriteRetryMode string

const (
	WriteRetryRestart           ChapterWriteRetryMode = "restart"
	WriteRetryResumeFailedStage ChapterWriteRetryMode = "resume_failed_stage"
)

type ChapterWriteStage string

const (
	WriteStageContext    ChapterWriteStage = "context"
	WriteStagePlanning   ChapterWriteStage = "planning"
	WriteStageWriting    ChapterWriteStage = "writing"
	WriteStageAuditing   ChapterWriteStage = "auditing"
	WriteStageRevising   ChapterWriteStage = "revising"
	WriteStagePolishing  ChapterWriteStage = "polishing"
	WriteStageExtracting ChapterWriteStage = "extracting"
	WriteStageSnapshot   ChapterWriteStage = "snapshot"
)

type ChapterWriteRun struct {
	ID               uint                  `json:"id" gorm:"primaryKey"`
	BookID           uint                  `json:"book_id" gorm:"index;not null"`
	TargetChapter    uint                  `json:"target_chapter" gorm:"not null"`
	RequestedModelID uint                  `json:"requested_model_id"`
	UserInput        string                `json:"user_input" gorm:"type:longtext"`
	Status           ChapterWriteRunStatus `json:"status" gorm:"size:16;index;not null"`
	CurrentStage     ChapterWriteStage     `json:"current_stage" gorm:"size:32"`
	RetryMode        ChapterWriteRetryMode `json:"retry_mode" gorm:"size:32"`
	ParentRunID      *uint                 `json:"parent_run_id" gorm:"index"`
	ResumeFromStage  ChapterWriteStage     `json:"resume_from_stage" gorm:"size:32"`
	CancelRequested  bool                  `json:"cancel_requested" gorm:"default:false"`
	ErrorMessage     string                `json:"error_message" gorm:"type:longtext"`
	StartedAt        *time.Time            `json:"started_at"`
	FinishedAt       *time.Time            `json:"finished_at"`
	CreatedAt        time.Time             `json:"created_at"`
	UpdatedAt        time.Time             `json:"updated_at"`
}

type ChapterWriteStageRun struct {
	ID            uint                    `json:"id" gorm:"primaryKey"`
	RunID         uint                    `json:"run_id" gorm:"uniqueIndex:idx_run_stage_attempt;not null"`
	Stage         ChapterWriteStage       `json:"stage" gorm:"uniqueIndex:idx_run_stage_attempt;size:32;not null"`
	Attempt       uint                    `json:"attempt" gorm:"uniqueIndex:idx_run_stage_attempt;default:1"`
	Status        ChapterWriteStageStatus `json:"status" gorm:"size:16;index;not null"`
	InputSummary  string                  `json:"input_summary" gorm:"type:text"`
	InputPayload  string                  `json:"input_payload" gorm:"type:longtext"`
	OutputSummary string                  `json:"output_summary" gorm:"type:text"`
	OutputPayload string                  `json:"output_payload" gorm:"type:longtext"`
	ErrorMessage  string                  `json:"error_message" gorm:"type:longtext"`
	StartedAt     *time.Time              `json:"started_at"`
	FinishedAt    *time.Time              `json:"finished_at"`
	CreatedAt     time.Time               `json:"created_at"`
	UpdatedAt     time.Time               `json:"updated_at"`
}

type ChapterWriteBaseline struct {
	ID                uint       `json:"id" gorm:"primaryKey"`
	RunID             uint       `json:"run_id" gorm:"uniqueIndex;not null"`
	BookID            uint       `json:"book_id" gorm:"index;not null"`
	BaseChapterNumber uint       `json:"base_chapter_number"`
	RecoverStatus     BookStatus `json:"recover_status" gorm:"size:16;not null"`
	CreatedAt         time.Time  `json:"created_at"`
}

type StartWriteRunRequest struct {
	ModelID     uint                  `json:"model_id"`
	UserInput   string                `json:"user_input"`
	RetryMode   ChapterWriteRetryMode `json:"retry_mode"`
	ParentRunID uint                  `json:"parent_run_id"`
}

type RetryWriteRunRequest struct {
	Mode ChapterWriteRetryMode `json:"mode"`
}
