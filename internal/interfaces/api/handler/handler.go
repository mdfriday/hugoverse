package handler

import (
	"context"
	"github.com/mdfriday/hugoverse/internal/infrastructure/couchdb"
	"html/template"
	"os"
	"os/signal"
	"sync"
	"syscall"

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
)

const apiUploadPrefix = "/api/uploads/"

type Handler struct {
	res *Response
	log loggers.Logger

	uploadDir      string
	imageProcessor *images.Processor
	shutdownFuncs  []func() // 存储各组件的shutdown函数

	db         *database.Database
	contentApp *contentEntity.Content
	adminApp   *adminEntity.Admin
	adminView  *admin.View

	auth *auth.Auth

	// CouchDB client for License management
	couchClient *couchdb.Client

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

	couchClient := couchdb.NewClient(&couchdb.Config{
		URL:       adminApp.CouchDBURL(),
		AdminUser: adminApp.CouchDBAdminName(),
		AdminPass: adminApp.CouchDBAdminPassword(),
		DBPrefix:  adminApp.CouchDBPrefix(),
	})

	// 创建基本Handler结构
	h := &Handler{
		res: NewResponse(adminView),
		log: log,

		uploadDir:     application.UploadDir(),
		shutdownFuncs: make([]func(), 0),

		db:         db,
		contentApp: contentApp,
		adminApp:   adminApp,
		adminView:  adminView,

		auth: auth,

		// CouchDB client
		couchClient: couchClient,
	}

	// 初始化图像处理器
	processor, shutdown, err := newImageProcessor()
	if err != nil {
		log.Errorf("error initializing image processor: %s", err)
	} else {
		h.imageProcessor = processor
		// 注册shutdown函数以便后续清理
		if shutdown != nil {
			h.shutdownFuncs = append(h.shutdownFuncs, shutdown)
		}
		h.shutdownFuncs = append(h.shutdownFuncs, h.imageProcessor.Shutdown)
	}

	return h
}

// Shutdown gracefully shuts down all components in the handler
func (s *Handler) Shutdown() {
	// 按注册的相反顺序执行shutdown函数
	for i := len(s.shutdownFuncs) - 1; i >= 0; i-- {
		s.shutdownFuncs[i]()
	}
}

// 修改newImageProcessor不再在内部defer shutdown()
func newImageProcessor() (*images.Processor, func(), error) {
	ctx := context.Background()

	// 设置用于关闭的context
	shutdownCtx, shutdown := signal.NotifyContext(ctx, os.Interrupt, os.Kill, syscall.SIGTERM)
	// 不在这里defer shutdown()，而是返回给调用者

	// 初始化日志记录器
	log := logger.New(zap.InfoLevel)

	// 初始化跟踪
	tracer := test.Tracer(log)

	// 初始化存储
	storage, err := file.New(application.UploadDir())
	if err != nil {
		log.Fatalf("error initializing storage: %s", err)
		// 如果失败，需要调用shutdown以避免资源泄漏
		shutdown()
		return nil, nil, err
	}

	// 初始化缓存
	cache := memory.New()

	// 创建组合的shutdown函数
	combinedShutdown := func() {
		shutdown()
		cache.Shutdown()
		// 可以在此添加其他需要在关闭时执行的操作
	}

	// 初始化图像处理器
	processor, err := images.New(shutdownCtx, log, tracer, 3, images.NewCache(tracer, cache, storage))
	if err != nil {
		combinedShutdown()
		return nil, nil, err
	}

	return processor, combinedShutdown, nil
}
