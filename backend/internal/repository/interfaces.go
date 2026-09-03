package repository

import "whwriter/backend/internal/model"

type DashboardStats struct {
	ActiveBooks   int64 `json:"active_books"`
	TotalChapters int64 `json:"total_chapters"`
}

type GenreRepository interface {
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
	GetTokenUsage(modelID uint) (model.TokenUsageSummary, error)
	GetTotalTokenUsage() (model.TokenUsageSummary, error)
	GetTokenUsageByModel() (map[uint]model.TokenUsageSummary, error)
	GetTokenUsageByAgent() ([]model.AgentTokenUsageSummary, error)
}

type TokenUsageRepository interface {
	Record(usage *model.TokenUsage) error
	LatestID() (uint, error)
	SummaryAfterID(afterID uint) ([]model.AgentTokenUsageSummary, error)
}

type WikiRepository interface {
	RefreshWikiEntities(bookID uint) error
	UpsertWikiEntity(entity *model.WikiEntity, aliases []string) error
	ResolveWikiEntity(bookID uint, name string, entityType model.WikiEntityType) (*model.WikiEntity, error)
	ResolveWikiEntityMentions(bookID uint, text string, limit int) ([]model.WikiEntity, error)
	ListWikiEntities(bookID uint, entityTypes []model.WikiEntityType, limit int) ([]model.WikiEntity, error)
	GetWikiEntityAliases(entityID uint) ([]model.WikiEntityAlias, error)
	RefreshWikiGraph(bookID uint) error
	RefreshWikiRelations(bookID uint) error
	ListWikiRelations(bookID uint, chapterNumber uint, entityIDs []uint, limit int) ([]model.WikiRelationView, error)
	ReplaceChapterWikiEvents(bookID uint, chapterNumber uint, events []model.WikiEventDraft) error
	ListWikiEvents(bookID uint, entityIDs []uint, limit int) ([]model.WikiEvent, error)
	GetWikiRelationEvidence(relationIDs []uint) ([]model.WikiRelationEvidence, error)
	BuildWikiGraphContext(query model.WikiGraphQuery) (*model.WikiGraphContext, error)
}

type BookRepository interface {
	Create(book *model.Book) error
	List() ([]model.Book, error)
}

type RadarRepository interface {
	ListTaxonomies(platform string) ([]model.RadarTaxonomy, error)
	ListTags(platform, category string) ([]model.RadarTag, error)
	SaveTags(tags []model.RadarTag) error
	SaveBookSetting(setting *model.RadarBookSetting) error
	GetBookSetting(bookID uint) (*model.RadarBookSetting, error)
	CreateScanJob(job *model.RadarScanJob) error
	SaveScanJob(job *model.RadarScanJob) error
	GetScanJob(jobID uint) (*model.RadarScanJob, error)
	ListScanJobs(limit int) ([]model.RadarScanJob, error)
	DeleteScanJob(jobID uint) error
	SaveSource(source *model.RadarSource) error
	GetSource(sourceID uint) (*model.RadarSource, error)
	FindSourceByBookID(platform, sourceBookID string) (*model.RadarSource, error)
	ListSources(limit int) ([]model.RadarSource, error)
	ListSourcesByCategory(platform, category string, limit int) ([]model.RadarSource, error)
	DeleteSourceCascade(sourceID uint) error
	DeleteSourcesCascade(sourceIDs []uint) error
	SaveChapterSamples(samples []model.RadarChapterSample) error
	GetChapterSamples(sourceID uint, limit int) ([]model.RadarChapterSample, error)
	SaveBookProfile(profile *model.RadarBookProfile) error
	ListBookProfiles(platform, category string, limit int) ([]model.RadarBookProfile, error)
	SaveIntroSamples(samples []model.RadarIntroSample) error
	ListIntroSamples(platform, category string, limit int) ([]model.RadarIntroSample, error)
	DeleteIntroSamples(ids []uint) error
	SaveTaxonomyProfile(profile *model.RadarTaxonomyProfile) error
	ListTaxonomyProfiles(platform, category string) ([]model.RadarTaxonomyProfile, error)
	ListActiveTaxonomyProfiles(platform, category string, tags []string) ([]model.RadarTaxonomyProfile, error)
	DeleteTaxonomyProfile(profileID uint) error
	DeleteTaxonomyProfilesByCategories(platform string, categories []string) error
	ReplaceRules(platform, category, tagKey string, rules []model.RadarRule) error
	ListRules(platform, category string, limit int) ([]model.RadarRule, error)
	ListActiveRules(platform, category string, tags []string, limit int) ([]model.RadarRule, error)
	DeleteRule(ruleID uint) error
	DeleteRulesByCategories(platform string, categories []string) error
}

type TruthFileRepository interface {
	WikiRepository
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
	RefreshKnowledgeIndex(bookID uint) error
	SearchKnowledge(query model.KnowledgeSearchQuery) ([]model.KnowledgeSearchResult, error)
	SaveRuntimeArtifact(a *model.RuntimeArtifact) error
	GetRuntimeArtifacts(bookID uint, chapterNumber uint) ([]model.RuntimeArtifact, error)
	GetAgentModelRoute(bookID uint, agentName string) (*model.AgentModelRoute, error)
	SaveAgentModelRoute(r *model.AgentModelRoute) error
	CreateChapterWriteRun(run *model.ChapterWriteRun) error
	SaveChapterWriteRun(run *model.ChapterWriteRun) error
	GetChapterWriteRun(runID uint) (*model.ChapterWriteRun, error)
	ListChapterWriteRuns(bookID uint, limit int) ([]model.ChapterWriteRun, error)
	GetActiveChapterWriteRun(bookID uint) (*model.ChapterWriteRun, error)
	ListInterruptedChapterWriteRuns() ([]model.ChapterWriteRun, error)
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
