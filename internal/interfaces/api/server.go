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
	"github.com/mdfriday/hugoverse/internal/interfaces/api/auth"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/cache"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/compression"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/cors"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/database"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/form"
	"github.com/mdfriday/hugoverse/internal/interfaces/api/handler"
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

	db       *database.Database
	adminApp *entity.Admin

	tls *tls.Tls

	record  *record.Record
	content *form.Content
	comp    *compression.Compression
	cache   *cache.Cache
	cors    *cors.Cors
	auth    *auth.Auth

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

	server, err := factory.NewAdminServer(s.db)
	if err != nil {
		return nil, err
	}
	s.adminApp = server

	s.comp = compression.New(s.Log, s.adminApp)
	s.cache = cache.New(s.Log, s.adminApp)
	s.cors = cors.New(s.Log, s.adminApp, s.cache)

	s.record.Start()

	s.tls = tls.NewTls(s, s.adminApp, application.TLSDir())

	s.handler = handler.New(s.Log, s.db, contentApp, s.adminApp, s.auth)

	s.registerHandler()

	go application.PreviewSiteRecycle(contentApp, s.adminApp.Token())

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
	s.registerLicenseHandler()
	s.registerContentHandler()
	s.registerAdminHandler()
	s.registerUserHandler()
}

func (s *Server) ListenAndServe(env ENV, enableHttps bool) error {
	if err := s.saveConfig(); err != nil {
		s.Log.Errorln("System failed to save config. Please try to run again.", err)
		return err
	}

	if enableHttps {
		if err := s.enableTLS(env); err != nil {
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
	if r.Header.Get("X-Forwarded-Proto") == "http" {
		r.URL.Scheme = "https"
		r.URL.Host = r.Host
		http.Redirect(w, r, r.URL.String(), http.StatusFound)
		return
	}
	if r.Header.Get("X-Forwarded-Proto") == "https" {
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
