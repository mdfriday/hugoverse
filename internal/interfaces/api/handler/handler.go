package handler

import (
	"context"
	"github.com/mdfriday/hugoverse/internal/application"
	adminEntity "github.com/mdfriday/hugoverse/internal/domain/admin/entity"
	contentEntity "github.com/mdfriday/hugoverse/internal/domain/content/entity"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/admin"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/auth"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/database"
	"github.com/mdfriday/hugoverse/pkg/images"
	"github.com/mdfriday/hugoverse/pkg/images/cache/memory"
	"github.com/mdfriday/hugoverse/pkg/images/storage/file"
	"github.com/mdfriday/hugoverse/pkg/logger"
	"github.com/mdfriday/hugoverse/pkg/loggers"
	"github.com/mdfriday/hugoverse/pkg/tracing/test"
	"go.uber.org/zap"
	"html/template"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const apiUploadPrefix = "/api/uploads/"

type Handler struct {
	res *Response
	log loggers.Logger

	uploadDir      string
	imageProcessor *images.Processor

	db         *database.Database
	contentApp *contentEntity.Content
	adminApp   *adminEntity.Admin
	adminView  *admin.View

	auth *auth.Auth

	deployments sync.Map // stores deployment sessions
}

func New(log loggers.Logger, db *database.Database,
	contentApp *contentEntity.Content, adminApp *adminEntity.Admin, auth *auth.Auth) *Handler {

	adminView := &admin.View{
		Logo:       adminApp.Name(),
		Types:      contentApp.AllContentTypes(),
		AdminTypes: contentApp.AllAdminTypes(),
		AdminEmail: adminApp.Conf.AdminEmail,
		Subview:    template.HTML(""),
	}

	// Initialize the image processor
	processor, err := newImageProcessor()
	if err != nil {
		log.Errorf("error initializing image processor: %s", err)
	}

	return &Handler{
		res: NewResponse(adminView),
		log: log,

		uploadDir:      application.UploadDir(),
		imageProcessor: processor,

		db:         db,
		contentApp: contentApp,
		adminApp:   adminApp,
		adminView:  adminView,

		auth: auth,
	}
}

func newImageProcessor() (*images.Processor, error) {
	ctx := context.Background()

	// Set up context for shutting down
	shutdownCtx, shutdown := signal.NotifyContext(ctx, os.Interrupt, os.Kill, syscall.SIGTERM)
	defer shutdown()

	// Initialize the logger
	log := logger.New(zap.InfoLevel)
	defer log.Sync()

	// Initialize tracing
	// tracerCtx, tracerCancel := context.WithCancel(ctx)
	// defer tracerCancel()

	// tracer, err := tracing.New(tracerCtx, log, "image-service")
	// if err != nil {
	// 	log.Fatalf("error initializing tracing: %s", err)
	// }
	// defer tracer.Shutdown(tracerCtx)
	tracer := test.Tracer(log)

	// Initialize the storage
	storage, err := file.New(application.UploadDir())
	if err != nil {
		log.Fatalf("error initializing storage: %s", err)
		return nil, err
	}

	// Initialize the cache
	cache := memory.New()
	defer cache.Shutdown()

	// Initialize the image processor
	return images.New(shutdownCtx, log, tracer, 3, images.NewCache(tracer, cache, storage))
}
