package container

import (
	"whwriter/backend/internal/config"
	"whwriter/backend/internal/repository"
	"whwriter/backend/internal/repository/mysql"
	"whwriter/backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Container struct {
	Config *config.Config
	DB     *gorm.DB

	UserRepo     repository.UserRepository
	GenreRepo    repository.GenreRepository
	PlatformRepo repository.PlatformRepository
	LLMConfigRepo repository.LLMConfigRepository
	BookRepo     repository.BookRepository

	EmailSvc *service.EmailService
	AuthSvc  *service.AuthService

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
	mysql.SeedGenres(db)
	mysql.SeedPlatforms(db)

	c.UserRepo = mysql.NewUserRepo(db)
	c.GenreRepo = mysql.NewGenreRepo(db)
	c.PlatformRepo = mysql.NewPlatformRepo(db)
	c.LLMConfigRepo = mysql.NewLLMConfigRepo(db)
	c.BookRepo = mysql.NewBookRepo(db)

	c.EmailSvc = service.NewEmailService(cfg.SMTP)
	c.AuthSvc = service.NewAuthService(c.UserRepo, c.EmailSvc, cfg)

	c.Engine = SetupRouter(c)

	return c, nil
}
