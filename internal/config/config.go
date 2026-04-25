package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Database   Database   `yaml:"postgres"`
	Server     Server     `yaml:"server"`
	App        App        `yaml:"app"`
	Kafka      Kafka      `yaml:"kafka"`
	Consumer   Consumer   `yaml:"consumer"`
	Security   Security   `yaml:"security"`
	Monitoring Monitoring `yaml:"monitoring"`
}

type Database struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
	SSLMode  string `yaml:"sslmode"`
}

type Server struct {
	Port         int `yaml:"port"`
	ReadTimeout  int `yaml:"read_timeout"`
	WriteTimeout int `yaml:"write_timeout"`
}

type App struct {
	Env      string `yaml:"env"`
	LogLevel string `yaml:"log_level"`
}

type Kafka struct {
	Brokers         []string `yaml:"brokers"`
	Topic           string   `yaml:"topic"`
	DLQTopic        string   `yaml:"dlq_topic"`
	ConsumerGroupID string   `yaml:"consumer_group_id"`
}

type Consumer struct {
	MetricsPort int `yaml:"metrics_port"`
}

type Security struct {
	AuthTokens []AuthToken `yaml:"auth_tokens"`
}

type AuthToken struct {
	Token  string `yaml:"token"`
	UserID string `yaml:"user_id"`
	Role   string `yaml:"role"`
}

type Monitoring struct {
	LargeAmountThreshold float64 `yaml:"large_amount_threshold"`
}

func Load(envPath ...string) (*Config, error) {
	path := filepath.Join("environment", ".env")
	if len(envPath) > 0 && envPath[0] != "" {
		path = envPath[0]
	}

	appCfg := defaultConfig()
	configPath := resolveConfigPath(path)
	if err := loadYAMLConfig(configPath, appCfg); err != nil {
		return nil, err
	}

	if err := loadEnvFile(path); err != nil {
		return nil, err
	}

	applyEnv(appCfg)

	return appCfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Database: Database{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "password",
			Name:     "salestracker",
			SSLMode:  "disable",
		},
		Server: Server{
			Port:         8080,
			ReadTimeout:  10,
			WriteTimeout: 10,
		},
		App: App{
			Env:      "development",
			LogLevel: "debug",
		},
		Kafka: Kafka{
			Brokers:         []string{"localhost:9094"},
			Topic:           "transactions.events",
			DLQTopic:        "transactions.events.dlq",
			ConsumerGroupID: "diploma-transactions-consumer",
		},
		Consumer: Consumer{
			MetricsPort: 2112,
		},
		Security: Security{
			AuthTokens: []AuthToken{{
				Token:  "dev-token",
				UserID: "11111111-1111-1111-1111-111111111111",
				Role:   "operator",
			}},
		},
		Monitoring: Monitoring{
			LargeAmountThreshold: 100000,
		},
	}
}

func resolveConfigPath(envPath string) string {
	candidates := []string{
		"config.yaml",
		filepath.Join("..", "config.yaml"),
		filepath.Join("..", "..", "config.yaml"),
	}
	if envPath != "" {
		envDir := filepath.Dir(envPath)
		candidates = append(candidates, filepath.Join(filepath.Dir(envDir), "config.yaml"))
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func loadYAMLConfig(path string, cfg *Config) error {
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config yaml: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse config yaml: %w", err)
	}
	return nil
}

func loadEnvFile(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := godotenv.Load(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("load env file: %w", err)
	}
	return nil
}

func applyEnv(cfg *Config) {
	cfg.Database.Host = envString("POSTGRES_HOST", cfg.Database.Host)
	cfg.Database.Port = envInt("POSTGRES_PORT", cfg.Database.Port)
	cfg.Database.User = envString("POSTGRES_USER", cfg.Database.User)
	cfg.Database.Password = envString("POSTGRES_PASSWORD", cfg.Database.Password)
	cfg.Database.Name = envString("POSTGRES_NAME", envString("POSTGRES_DB", cfg.Database.Name))
	cfg.Database.SSLMode = envString("POSTGRES_SSLMODE", cfg.Database.SSLMode)

	cfg.Server.Port = envInt("SERVER_PORT", cfg.Server.Port)
	cfg.Server.ReadTimeout = envInt("SERVER_READ_TIMEOUT", cfg.Server.ReadTimeout)
	cfg.Server.WriteTimeout = envInt("SERVER_WRITE_TIMEOUT", cfg.Server.WriteTimeout)

	cfg.App.Env = envString("APP_ENV", cfg.App.Env)
	cfg.App.LogLevel = envString("APP_LOG_LEVEL", cfg.App.LogLevel)

	cfg.Kafka.Brokers = envStringSlice("KAFKA_BROKERS", cfg.Kafka.Brokers)
	cfg.Kafka.Topic = envString("KAFKA_TOPIC", cfg.Kafka.Topic)
	cfg.Kafka.DLQTopic = envString("KAFKA_DLQ_TOPIC", cfg.Kafka.DLQTopic)
	cfg.Kafka.ConsumerGroupID = envString("KAFKA_CONSUMER_GROUP_ID", cfg.Kafka.ConsumerGroupID)

	cfg.Consumer.MetricsPort = envInt("CONSUMER_METRICS_PORT", cfg.Consumer.MetricsPort)

	if raw := strings.TrimSpace(os.Getenv("SECURITY_AUTH_TOKENS")); raw != "" {
		cfg.Security.AuthTokens = parseAuthTokens(raw)
	}
	cfg.Monitoring.LargeAmountThreshold = envFloat("MONITORING_LARGE_AMOUNT_THRESHOLD", cfg.Monitoring.LargeAmountThreshold)
}

func parseAuthTokens(raw string) []AuthToken {
	parts := strings.Split(raw, ",")
	tokens := make([]AuthToken, 0, len(parts))
	for _, part := range parts {
		fields := strings.Split(strings.TrimSpace(part), ":")
		if len(fields) < 2 {
			continue
		}

		token := strings.TrimSpace(fields[0])
		userID := strings.TrimSpace(fields[1])
		role := "operator"
		if len(fields) >= 3 && strings.TrimSpace(fields[2]) != "" {
			role = strings.TrimSpace(fields[2])
		}
		if token == "" || userID == "" {
			continue
		}

		tokens = append(tokens, AuthToken{
			Token:  token,
			UserID: userID,
			Role:   role,
		})
	}
	return tokens
}

func envString(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func envStringSlice(name string, fallback []string) []string {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}

	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		return fallback
	}
	return values
}

func envInt(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func envFloat(name string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return value
}
