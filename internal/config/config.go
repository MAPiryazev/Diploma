package config

import (
	"fmt"

	"github.com/wb-go/wbf/config"
)

type Config struct {
	Database Database
	Server   Server
	App      App
}

type Database struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

type Server struct {
	Port         int
	ReadTimeout  int
	WriteTimeout int
}

type App struct {
	Env      string
	LogLevel string
}

func Load(envPath ...string) (*Config, error) {
	cfg := config.New()

	path := "environment"
	if len(envPath) > 0 && envPath[0] != "" {
		path = envPath[0]
	}

	if err := cfg.LoadEnvFiles(path); err != nil {
		return nil, fmt.Errorf("load env file: %w", err)
	}

	cfg.EnableEnv("")

	cfg.SetDefault("postgres.host", "localhost")
	cfg.SetDefault("postgres.port", 5432)
	cfg.SetDefault("postgres.user", "postgres")
	cfg.SetDefault("postgres.password", "password")
	cfg.SetDefault("postgres.name", "salestracker")
	cfg.SetDefault("server.port", 8080)
	cfg.SetDefault("server.read_timeout", 10)
	cfg.SetDefault("server.write_timeout", 10)
	cfg.SetDefault("app.env", "development")
	cfg.SetDefault("app.log_level", "debug")

	appCfg := &Config{
		Database: Database{
			Host:     cfg.GetString("postgres.host"),
			Port:     cfg.GetInt("postgres.port"),
			User:     cfg.GetString("postgres.user"),
			Password: cfg.GetString("postgres.password"),
			Name:     cfg.GetString("postgres.name"),
		},
		Server: Server{
			Port:         cfg.GetInt("server.port"),
			ReadTimeout:  cfg.GetInt("server.read_timeout"),
			WriteTimeout: cfg.GetInt("server.write_timeout"),
		},
		App: App{
			Env:      cfg.GetString("app.env"),
			LogLevel: cfg.GetString("app.log_level"),
		},
	}

	return appCfg, nil
}
