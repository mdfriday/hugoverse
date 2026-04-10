package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/mdfriday/hugoverse/internal/application"
	"github.com/mdfriday/hugoverse/internal/domain/admin/entity"
	"github.com/mdfriday/hugoverse/internal/domain/admin/factory"
	"github.com/mdfriday/hugoverse/internal/infrastructure/caddy"
	"github.com/mdfriday/hugoverse/internal/infrastructure/couchdb"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/auth"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/cache"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/compression"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/cors"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/database"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/form"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/handler"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/license"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/record"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/tls"
	"github.com/mdfriday/hugoverse/pkg/loggers"
)

type PORT string

const (
	HttpsPort    PORT = "https_port"
	HttpPort     PORT = "http_port"
	DevHttpsPort PORT = "dev_https_port"
)

const BindAddress = "bind_addr"

type ENV int

func (s ENV) String() string {
	switch s {
	case DEV:
		return "dev"
	case PROD:
		return "prod"
	default:
		return "unknown"
	}
}

const (
	DEV ENV = iota
	PROD
)

type Server struct {
	mux     *mux.Router
	Log     loggers.Logger
	LogFile *os.File

	Bind         string
	HttpsPort    int
	HttpPort     int
	DevHttpsPort int
	Env          ENV // 环境变量：DEV 或 PROD

	db       *database.Database
	adminApp *entity.Admin

	tls *tls.Tls

	record  *record.Record
	content *form.Content
	comp    *compression.Compression
	cache   *cache.Cache
	cors    *cors.Cors
	auth    *auth.Auth
	license *license.License

	handler    *handler.Handler
	httpServer *http.Server
}

func NewServer(options ...func(s *Server) error) (*Server, error) {
	db, err := database.New(application.DataDir())
	if err != nil {
		return nil, err
	}

	s := &Server{
		mux:          mux.NewRouter(),
		Bind:         "localhost",
		HttpPort:     80,
		HttpsPort:    443,
		DevHttpsPort: 10443,

		db:      db,
		record:  record.New(application.DataDir()),
		content: &form.Content{},
		auth:    &auth.Auth{},
	}
	for _, o := range options {
		if err := o(s); err != nil {
			return nil, err
		}
	}
	if s.Log == nil {
		return nil, fmt.Errorf("must provide an option func that specifies a logger")
	}

	contentApp := application.NewContentServer(s.db)
	s.db.RegisterContentBuckets(contentApp.AllContentTypeNames())
	if err := s.db.StartAdminDatabase(contentApp.AllAdminTypeNames()); err != nil {
		return nil, err
	}

	server, err := factory.NewAdminServer(s.Env.String(), s.db)
	if err != nil {
		return nil, err
	}
	s.adminApp = server

	s.comp = compression.New(s.Log, s.adminApp)
	s.cache = cache.New(s.Log, s.adminApp)
	s.cors = cors.New(s.Log, s.adminApp, s.cache)

	s.record.Start()

	s.tls = tls.NewTls(s, s.adminApp, application.TLSDir())

	s.license = license.New(contentApp, s.Log)

	s.handler = handler.New(s.Log, s.db, contentApp, s.adminApp, s.auth)

	s.registerHandler()

	// 尝试自动初始化（在启动后台任务之前）
	if err := application.AutoInitialize(s.adminApp, s.db, s.Log); err != nil {
		s.Log.Warnf("Auto-initialization failed: %v", err)
		s.Log.Println("Please visit /admin/init to configure manually")
	}

	//go application.PreviewSiteRecycle(contentApp, s.adminApp.Token())
	go application.LicenseResourceRecycle(contentApp, s.adminApp, s.Log)
	go application.FridayResourceRecycle(s.Log)

	// 在生产环境启动备份调度器
	if s.Env == PROD {
		go func() {
			s.Log.Println("Starting backup scheduler for production environment...")
			couchdbCfg := s.GetCouchDBConfig()
			caddyCfg := s.GetCaddyConfig()

			if couchdbCfg != nil && caddyCfg != nil {
				scheduler := application.NewBackupScheduler(couchdbCfg, caddyCfg, contentApp, s.Log)
				scheduler.Start()
			} else {
				s.Log.Warnln("Backup scheduler not started: missing CouchDB or Caddy configuration")
			}
		}()
	}

	return s, nil
}

func (s *Server) Close() {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.httpServer.Shutdown(ctx)
	}

	if s.handler != nil {
		s.handler.Shutdown()
	}

	s.db.Close()
	s.record.Close()

	if s.LogFile != nil {
		s.LogFile.Close()
	}
}

func (s *Server) registerHandler() {
	s.registerHealthHandler()
	s.registerLicenseHandler()
	s.registerContentHandler()
	s.registerAdminHandler()
	s.registerUserHandler()
}

func (s *Server) ListenAndServe(enableHttps bool) error {
	if err := s.saveConfig(); err != nil {
		s.Log.Errorln("System failed to save config. Please try to run again.", err)
		return err
	}

	if enableHttps {
		if err := s.enableTLS(s.Env); err != nil {
			s.Log.Errorln("System failed to enable TLS. Please try to run again.", err)
			return err
		}
	}

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.Bind, s.HttpPort),
		Handler: s,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		s.Log.Printf("Listening on %s:%d", s.Bind, s.HttpPort)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.Log.Errorf("ListenAndServe error: %v", err)
		}
	}()

	<-quit
	s.Log.Println("Server is shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.Log.Errorf("Server shutdown error: %v", err)
	}

	if s.handler != nil {
		s.handler.Shutdown()
	}

	s.Log.Println("Server exited properly")
	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 记录请求详情（调试用）
	xForwardedProto := r.Header.Get("X-Forwarded-Proto")
	dockerContainer := os.Getenv("DOCKER_CONTAINER")
	
	s.Log.Printf("[REQUEST] Path=%s Host=%s Scheme=%s X-Forwarded-Proto=%s Env=%s Docker=%s", 
		r.URL.Path, r.Host, r.URL.Scheme, xForwardedProto, s.Env, dockerContainer)
	
	// 只在生产环境且非 Docker 容器时才强制重定向到 HTTPS
	if s.Env == PROD && xForwardedProto == "http" && dockerContainer != "true" {
		s.Log.Printf("[REDIRECT] Redirecting to HTTPS: %s -> https://%s%s", r.URL.String(), r.Host, r.URL.Path)
		r.URL.Scheme = "https"
		r.URL.Host = r.Host
		http.Redirect(w, r, r.URL.String(), http.StatusFound)
		return
	}
	
	if xForwardedProto == "https" {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; preload")
	}
	
	s.mux.ServeHTTP(w, r)
}

func (s *Server) enableTLS(env ENV) error {
	switch env {
	case DEV:
		go func() {
			if err := s.tls.EnableDev(); err != nil {
				s.Log.Errorf("System failed to enable TLS. Please try to run again.", err)
			}
		}()
	case PROD:
		// todo
		return nil
	}
	return nil
}

func (s *Server) saveConfig() error {
	err := s.adminApp.PutConfig(string(HttpsPort), s.HttpsPort)
	if err != nil {
		s.Log.Errorln("System failed to save Https Port config. Please try to run again.", err)
		return err
	}
	err = s.adminApp.PutConfig(string(HttpPort), s.HttpPort)
	if err != nil {
		s.Log.Errorln("System failed to save Http Port config. Please try to run again.", err)
		return err
	}
	err = s.adminApp.PutConfig(string(DevHttpsPort), s.DevHttpsPort)
	if err != nil {
		s.Log.Errorln("System failed to save DevHttpsPort config. Please try to run again.", err)
		return err
	}
	err = s.adminApp.PutConfig(string(BindAddress), s.Bind)
	if err != nil {
		s.Log.Errorln("System failed to save bind address config. Please try to run again.", err)
		return err
	}
	return nil
}

// GetCouchDBConfig 获取 CouchDB 配置（用于备份调度器）
func (s *Server) GetCouchDBConfig() *couchdb.Config {
	if s.adminApp == nil || s.adminApp.Conf == nil {
		return nil
	}

	return &couchdb.Config{
		URL:       s.adminApp.CouchDBURL(),
		AdminUser: s.adminApp.CouchDBAdminName(),
		AdminPass: s.adminApp.CouchDBAdminPassword(),
		DBPrefix:  s.adminApp.CouchDBPrefix(),
	}
}

// GetCaddyConfig 获取 Caddy 配置（用于备份调度器）
func (s *Server) GetCaddyConfig() *caddy.Config {
	if s.adminApp == nil || s.adminApp.Conf == nil {
		return nil
	}

	return &caddy.Config{
		AdminAPI:       "http://127.0.0.1:2019",
		DefaultBackend: "127.0.0.1:1314",
		CouchDBBackend: "127.0.0.1:5984",
		CoreDomain:     s.adminApp.Conf.Domain,
		ServerIP:       s.adminApp.Conf.ServerIP,
		DNSPodToken:    "", // DNSPod Token 不存储在配置中，只在启动时使用
		PidFile:        "/tmp/caddy.pid",
		LogFile:        "/tmp/caddy.log",
	}
}
