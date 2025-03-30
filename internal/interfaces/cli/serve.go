package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/bep/logg"
	"github.com/mdfriday/hugoverse/internal/application"
	"github.com/mdfriday/hugoverse/internal/interfaces/api"
	"github.com/mdfriday/hugoverse/pkg/loggers"
)

type serverCmd struct {
	parent *flag.FlagSet
	cmd    *flag.FlagSet
	port   *string
	env    *string
	https  *bool
}

func NewServeCmd(parent *flag.FlagSet) (*serverCmd, error) {
	nCmd := &serverCmd{
		parent: parent,
	}

	nCmd.cmd = flag.NewFlagSet("normal", flag.ExitOnError)
	nCmd.port = nCmd.cmd.String("port", "1314",
		fmt.Sprintln("[optional] server listening port, default is `1314`"))
	nCmd.env = nCmd.cmd.String("env", "dev",
		fmt.Sprintln("[optional, dev|prod] development environment, default is `dev`"))
	nCmd.https = nCmd.cmd.Bool("https", false,
		fmt.Sprintln("[optional] enable https, default is `false`"))

	err := nCmd.cmd.Parse(parent.Args()[1:])
	if err != nil {
		return nil, err
	}

	return nCmd, nil
}

func (c *serverCmd) Usage() {
	c.cmd.Usage()
}

func (c *serverCmd) Run() error {
	env := api.DEV
	if *c.env == "prod" {
		env = api.PROD
	}
	s, err := api.NewServer(setupLogger(env), setupPort(*c.port))
	if err != nil {
		return fmt.Errorf("error creating server: %v", err)
	}
	defer s.Close()

	// 启动服务器并等待它完成（现在ListenAndServe会阻塞直到收到终止信号）
	err = s.ListenAndServe(env, *c.https)
	if err != nil {
		s.Log.Errorf("Error with server: %v", err)
		return err
	}

	return nil
}

func setupLogger(env api.ENV) func(s *api.Server) error {
	return func(s *api.Server) error {
		switch env {
		case api.DEV:
			// Create log file with timestamp
			logFile := filepath.Join(application.LogDir(), fmt.Sprintf("hv_dev_%s.log", time.Now().Format("20060102_150405")))
			f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				return fmt.Errorf("failed to create log file: %w", err)
			}
			s.Log = loggers.New(loggers.Options{
				Level:         logg.LevelInfo,
				DistinctLevel: logg.LevelWarn,
				Stdout:        f,
				Stderr:        f,
				WithColor:     false,
			})
			s.LogFile = f

		case api.PROD:
			// Create log file with timestamp
			logFile := filepath.Join(application.LogDir(), fmt.Sprintf("hv_prod_%s.log", time.Now().Format("20060102_150405")))
			f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				return fmt.Errorf("failed to create log file: %w", err)
			}
			s.Log = loggers.New(loggers.Options{
				Level:         logg.LevelError,
				DistinctLevel: logg.LevelError,
				Stdout:        f,
				Stderr:        f,
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
