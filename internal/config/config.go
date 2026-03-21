package config

import (
	"fmt"
	"strings"

	"github.com/wb-go/wbf/config"
)

type Config struct {
	Database Database
	Server   Server
	App      App
	Kafka    Kafka
	Consumer Consumer
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

type Kafka struct {
	Brokers         []string
	Topic           string
	DLQTopic        string
	ConsumerGroupID string
}

type Consumer struct {
	MetricsPort int
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
	cfg.SetDefault("kafka.brokers", []string{"localhost:9094"})
	cfg.SetDefault("kafka.topic", "transactions.events")
	cfg.SetDefault("kafka.dlq_topic", "transactions.events.dlq")
	cfg.SetDefault("kafka.consumer_group_id", "diploma-transactions-consumer")
	cfg.SetDefault("consumer.metrics_port", 2112)

	brokersRaw := cfg.GetString("kafka.brokers")
	brokers := []string{"localhost:9094"}
	if brokersRaw != "" {
		parts := strings.Split(brokersRaw, ",")
		brokers = brokers[:0]
		for _, part := range parts {
			b := strings.TrimSpace(part)
			if b != "" {
				brokers = append(brokers, b)
			}
		}
		if len(brokers) == 0 {
			brokers = []string{"localhost:9094"}
		}
	}

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
		Kafka: Kafka{
			Brokers:         brokers,
			Topic:           cfg.GetString("kafka.topic"),
			DLQTopic:        cfg.GetString("kafka.dlq_topic"),
			ConsumerGroupID: cfg.GetString("kafka.consumer_group_id"),
		},
		Consumer: Consumer{
			MetricsPort: cfg.GetInt("consumer.metrics_port"),
		},
	}

	return appCfg, nil
}
