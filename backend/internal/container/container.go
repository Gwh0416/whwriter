package container

import (
	"whwriter/backend/internal/agent"
	"whwriter/backend/internal/config"
	"whwriter/backend/internal/llm"
	"whwriter/backend/internal/pipeline"
	"whwriter/backend/internal/repository"
	"whwriter/backend/internal/repository/mysql"
	"whwriter/backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Container struct {
	Config *config.Config
	DB     *gorm.DB

	UserRepo       repository.UserRepository
	GenreRepo      repository.GenreRepository
	PlatformRepo   repository.PlatformRepository
	LLMConfigRepo  repository.LLMConfigRepository
	LLMModelRepo   repository.LLMModelRepository
	TokenUsageRepo repository.TokenUsageRepository
	BookRepo       repository.BookRepository
	TruthFileRepo  repository.TruthFileRepository

	EmailSvc *service.EmailService
	AuthSvc  *service.AuthService

	LLMClient     *llm.Client
	Pipeline      *pipeline.Pipeline
	AgentRegistry *agent.Registry

	Engine *gin.Engine
}

func New(cfg *config.Config) (*Container, error) {
	c := &Container{Config: cfg}

	db, err := mysql.NewDB(cfg.MySQLDSN())
	if err != nil {
		return nil, err
	}
	c.DB = db

	mysql.SeedAdmin(db, cfg.Admin.Email, cfg.Admin.Username, cfg.Admin.Password)

	c.UserRepo = mysql.NewUserRepo(db)
	c.GenreRepo = mysql.NewGenreRepo(db)
	c.PlatformRepo = mysql.NewPlatformRepo(db)
	c.LLMConfigRepo = mysql.NewLLMConfigRepo(db)
	c.LLMModelRepo = mysql.NewLLMModelRepo(db)
	c.TokenUsageRepo = mysql.NewTokenUsageRepo(db)
	c.BookRepo = mysql.NewBookRepo(db)
	c.TruthFileRepo = mysql.NewTruthFileRepo(db)

	c.EmailSvc = service.NewEmailService(cfg.SMTP)
	c.AuthSvc = service.NewAuthService(c.UserRepo, c.EmailSvc, cfg)

	c.LLMClient = llm.NewClient(c.LLMModelRepo, c.LLMConfigRepo, c.TokenUsageRepo, cfg.LLM)
	c.Pipeline = pipeline.New(c.LLMClient, c.TruthFileRepo)
	c.AgentRegistry = agent.NewRegistry()

	c.Engine = SetupRouter(c)

	return c, nil
}
