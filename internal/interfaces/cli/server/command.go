package server

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/bep/logg"
	"github.com/mdfriday/hugoverse/internal/application"
	"github.com/mdfriday/hugoverse/internal/interfaces/api"
	"github.com/mdfriday/hugoverse/pkg/loggers"
)

// Command represents the serve command
type Command struct{}

// Name returns the name of the command
func (c *Command) Name() string {
	return "server"
}

// Description returns the description of the command
func (c *Command) Description() string {
	return "Start the Hugoverse API server"
}

// Run executes the command
func (c *Command) Run() error {
	// 从环境变量读取配置
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "1314"
	}

	env := api.DEV
	if os.Getenv("ENV") == "prod" {
		env = api.PROD
	}

	enableHttps := false
	if os.Getenv("ENABLE_HTTPS") == "true" {
		enableHttps = true
	}

	// Docker 环境中绑定到 0.0.0.0，否则绑定到 localhost
	bind := "localhost"
	if os.Getenv("DOCKER_CONTAINER") == "true" {
		bind = "0.0.0.0"
	}

	s, err := api.NewServer(setupEnv(env), setupLogger(env), setupPort(port), setupBind(bind))
	if err != nil {
		return fmt.Errorf("error creating server: %v", err)
	}
	defer s.Close()

	// 启动服务器
	err = s.ListenAndServe(enableHttps)
	if err != nil {
		s.Log.Errorf("Error with server: %v", err)
		return err
	}

	return nil
}

func setupBind(bind string) func(s *api.Server) error {
	return func(s *api.Server) error {
		s.Bind = bind
		return nil
	}
}

func setupEnv(env api.ENV) func(s *api.Server) error {
	return func(s *api.Server) error {
		s.Env = env
		return nil
	}
}

func setupLogger(env api.ENV) func(s *api.Server) error {
	return func(s *api.Server) error {
		logFile := filepath.Join(application.LogDir(), fmt.Sprintf("hugoverse_%s.log", time.Now().Format("20060102_150405")))
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			// Fallback to stdout/stderr
			s.Log = loggers.New(loggers.Options{
				Level:         logg.LevelInfo,
				DistinctLevel: logg.LevelInfo,
				Stdout:        os.Stdout,
				Stderr:        os.Stderr,
				WithColor:     true,
			})
			s.Log.Warnf("Failed to create log file, using stdout: %v", err)
			return nil
		}

		// Docker 下同时写入文件与 stdout/stderr，便于 docker logs 看到 AutoInitialize、Caddy 企业配置等
		outW, errW := io.Writer(f), io.Writer(f)
		if os.Getenv("DOCKER_CONTAINER") == "true" {
			outW = io.MultiWriter(os.Stdout, f)
			errW = io.MultiWriter(os.Stderr, f)
		}

		switch env {
		case api.DEV:
			s.Log = loggers.New(loggers.Options{
				Level:         logg.LevelDebug,
				DistinctLevel: logg.LevelDebug,
				Stdout:        outW,
				Stderr:        errW,
				WithColor:     false,
			})
			s.LogFile = f

		case api.PROD:
			s.Log = loggers.New(loggers.Options{
				Level:         logg.LevelInfo,
				DistinctLevel: logg.LevelInfo,
				Stdout:        outW,
				Stderr:        errW,
				WithColor:     false,
			})
			s.LogFile = f
		}

		loggers.SetGlobal(s.Log)
		return nil
	}
}

func setupPort(port string) func(s *api.Server) error {
	return func(s *api.Server) error {
		p, err := strconv.Atoi(port)
		if err != nil {
			return err
		}
		s.HttpPort = p

		return nil
	}
}
