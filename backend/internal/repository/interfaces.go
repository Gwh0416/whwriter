package repository

import "whwriter/backend/internal/model"

type UserRepository interface {
	Create(user *model.User) error
	FindByEmail(email string) (*model.User, error)
	FindByUsername(username string) (*model.User, error)
	FindByID(id uint) (*model.User, error)
	SaveVerificationCode(email, code string) error
	VerifyCode(email, code string) (bool, error)
	UpdatePassword(userID uint, passwordHash string) error
	ListUsers(page, pageSize int) ([]model.User, int64, error)
	GetStats() (*DashboardStats, error)
	UpdateStatus(userID uint, status model.UserStatus) error
	AddBalance(userID uint, amount int64) error
}

type DashboardStats struct {
	TotalUsers    int64 `json:"total_users"`
	ActiveBooks   int64 `json:"active_books"`
	TotalChapters int64 `json:"total_chapters"`
}

type GenreRepository interface {
	ListBuiltin() ([]model.Genre, error)
	ListByUser(userID uint) ([]model.Genre, error)
	ListMyGenres(userID uint) ([]model.Genre, error)
	ListAll() ([]model.Genre, error)
	FindByID(id uint) (*model.Genre, error)
	Create(genre *model.Genre) error
	Update(genre *model.Genre) error
	Delete(id uint) error
}

type PlatformRepository interface {
	List() ([]model.Platform, error)
	ListAll() ([]model.Platform, error)
	Create(platform *model.Platform) error
	Update(platform *model.Platform) error
	Delete(id uint) error
}

type LLMConfigRepository interface {
	ListAll() ([]model.LLMConfig, error)
	ListAllWithModels() ([]model.LLMConfig, error)
	FindByID(id uint) (*model.LLMConfig, error)
	Create(config *model.LLMConfig) error
	Update(config *model.LLMConfig) error
	Delete(id uint) error
}

type LLMModelRepository interface {
	ListByConfig(configID uint) ([]model.LLMModel, error)
	ListEnabled() ([]model.LLMModel, error)
	FindByID(id uint) (*model.LLMModel, error)
	Create(model *model.LLMModel) error
	Update(model *model.LLMModel) error
	Delete(id uint) error
	DeleteByConfig(configID uint) error
	SetDefault(modelID uint) error
	GetTokenUsage(modelID uint) (int64, error)
	GetTotalTokenUsage() (int64, error)
	GetTokenUsageByModel() (map[uint]int64, error)
}

type TokenUsageRepository interface {
	Record(usage *model.TokenUsage) error
}

type BookRepository interface {
	Create(book *model.Book) error
	ListByUser(userID uint) ([]model.Book, error)
}

type TruthFileRepository interface {
	GetCharacters(bookID uint) ([]model.Character, error)
	SaveCharacter(c *model.Character) error
	GetFacts(bookID uint) ([]model.Fact, error)
	SaveFact(f *model.Fact) error
	GetHooks(bookID uint) ([]model.Hook, error)
	SaveHook(h *model.Hook) error
	GetChapterSummaries(bookID uint) ([]model.ChapterSummary, error)
	SaveChapterSummary(s *model.ChapterSummary) error
	GetFoundation(bookID uint, fileType model.FoundationFileType) (*model.BookFoundation, error)
	ListFoundations(bookID uint) ([]model.BookFoundation, error)
	SaveFoundation(f *model.BookFoundation) error
	GetChapter(bookID uint, chapterNumber uint) (*model.Chapter, error)
	SaveChapter(ch *model.Chapter) error
	ListChapters(bookID uint) ([]model.Chapter, error)
	GetNextChapterNumber(bookID uint) (uint, error)
	GetBook(bookID uint) (*model.Book, error)
	GetBookState(bookID uint) (*model.BookState, error)
	SaveBookState(s *model.BookState) error
	UpdateBookStatus(bookID uint, status model.BookStatus) error
	TransitionBookStatus(bookID uint, from []model.BookStatus, to model.BookStatus) (bool, error)
	SaveChapterSnapshot(s *model.ChapterSnapshot) error
	GetChapterSnapshots(bookID uint) ([]model.ChapterSnapshot, error)
	SaveRuntimeArtifact(a *model.RuntimeArtifact) error
	GetRuntimeArtifacts(bookID uint, chapterNumber uint) ([]model.RuntimeArtifact, error)
	GetAgentModelRoute(bookID uint, agentName string) (*model.AgentModelRoute, error)
	SaveAgentModelRoute(r *model.AgentModelRoute) error
	CreateChapterWriteRun(run *model.ChapterWriteRun) error
	SaveChapterWriteRun(run *model.ChapterWriteRun) error
	GetChapterWriteRun(runID uint) (*model.ChapterWriteRun, error)
	ListChapterWriteRuns(bookID uint, limit int) ([]model.ChapterWriteRun, error)
	GetActiveChapterWriteRun(bookID uint) (*model.ChapterWriteRun, error)
	CreateChapterWriteBaseline(b *model.ChapterWriteBaseline) error
	GetChapterWriteBaseline(runID uint) (*model.ChapterWriteBaseline, error)
	CreateChapterWriteStageRun(stage *model.ChapterWriteStageRun) error
	SaveChapterWriteStageRun(stage *model.ChapterWriteStageRun) error
	GetChapterWriteStages(runID uint) ([]model.ChapterWriteStageRun, error)
	GetChapterWriteStage(runID uint, stage model.ChapterWriteStage) (*model.ChapterWriteStageRun, error)
	DeleteBookCascade(bookID uint) error
	DeleteLatestChapterCascade(bookID uint, chapterNumber uint) error
	WithinTx(fn func(TruthFileRepository) error) error
}
