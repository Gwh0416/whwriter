package container

import (
	"whwriter/backend/internal/agent"
	"whwriter/backend/internal/config"
	"whwriter/backend/internal/llm"
	"whwriter/backend/internal/pipeline"
	"whwriter/backend/internal/repository"
	"whwriter/backend/internal/repository/sqlite"
	"whwriter/backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Container struct {
	Config *config.Config
	DB     *gorm.DB

	GenreRepo      repository.GenreRepository
	PlatformRepo   repository.PlatformRepository
	LLMConfigRepo  repository.LLMConfigRepository
	LLMModelRepo   repository.LLMModelRepository
	TokenUsageRepo repository.TokenUsageRepository
	BookRepo       repository.BookRepository
	TruthFileRepo  repository.TruthFileRepository
	RadarRepo      repository.RadarRepository

	RadarSvc *service.RadarService

	LLMClient     *llm.Client
	Pipeline      *pipeline.Pipeline
	AgentRegistry *agent.Registry

	Engine *gin.Engine
}

func New(cfg *config.Config) (*Container, error) {
	c := &Container{Config: cfg}

	db, err := sqlite.NewDB(cfg.SQLitePath())
	if err != nil {
		return nil, err
	}
	c.DB = db

	sqlite.SeedGenres(db)
	sqlite.SeedRadarTaxonomies(db)

	c.GenreRepo = sqlite.NewGenreRepo(db)
	c.PlatformRepo = sqlite.NewPlatformRepo(db)
	c.LLMConfigRepo = sqlite.NewLLMConfigRepo(db)
	c.LLMModelRepo = sqlite.NewLLMModelRepo(db)
	c.TokenUsageRepo = sqlite.NewTokenUsageRepo(db)
	c.BookRepo = sqlite.NewBookRepo(db)
	c.TruthFileRepo = sqlite.NewTruthFileRepo(db)
	c.RadarRepo = sqlite.NewRadarRepo(db)

	c.LLMClient = llm.NewClient(c.LLMModelRepo, c.LLMConfigRepo, c.TokenUsageRepo, cfg.LLM)
	c.RadarSvc = service.NewRadarService(
		c.RadarRepo,
		c.LLMClient,
		cfg.BrowserCDPURL(),
		cfg.BrowserChapterFetchTimeoutSeconds(),
		cfg.BrowserAutoLaunch(),
		cfg.BrowserChromeAppName(),
		cfg.BrowserUserDataDir(),
		cfg.FanqieContentAPIURL(),
		cfg.FanqieContentAPITimeoutSeconds(),
	)
	c.Pipeline = pipeline.New(c.LLMClient, c.TruthFileRepo, c.RadarRepo, c.TokenUsageRepo)
	c.AgentRegistry = agent.NewRegistry()

	c.Engine = SetupRouter(c)

	return c, nil
}
