package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server       ServerConfig
	Database     DatabaseConfig
	Auth         AuthConfig
	Storage      StorageConfig
	Edge         EdgeConfig
	Integrations IntegrationsConfig
	CORS         CORSConfig
}

type IntegrationsConfig struct {
	StoreSyncAgentURL string `mapstructure:"storesyncagent_url"`
}

type ServerConfig struct {
	Port int
	Mode string
}

type DatabaseConfig struct {
	Driver      string
	SQLitePath  string
	PostgresDSN string `mapstructure:"postgres_dsn"`
}

type AuthConfig struct {
	Enabled   bool
	JWTSecret string `mapstructure:"jwt_secret"`
}

type StorageConfig struct {
	Driver        string      `mapstructure:"driver"`
	LocalPath     string      `mapstructure:"local_path"`
	PublicBaseURL string      `mapstructure:"public_base_url"`
	Prefix        string      `mapstructure:"prefix"`
	MinIO         MinIOConfig `mapstructure:"minio"`
}

type MinIOConfig struct {
	Endpoint   string
	AccessKey  string `mapstructure:"access_key"`
	SecretKey  string `mapstructure:"secret_key"`
	Bucket     string
	UseSSL     bool   `mapstructure:"use_ssl"`
	Prefix     string
	PublicRead bool   `mapstructure:"public_read"`
}

type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
}

type EdgeConfig struct {
	Schema       string          `mapstructure:"schema"`
	Table        string          `mapstructure:"table"`
	CloudEdgeID  string          `mapstructure:"cloud_edge_id"`
	HealthPollSec int            `mapstructure:"health_poll_sec"`
	MinIO        EdgeMinIOConfig `mapstructure:"minio"`
}

type EdgeMinIOConfig struct {
	Endpoint      string `mapstructure:"endpoint"`
	AccessKey     string `mapstructure:"access_key"`
	SecretKey     string `mapstructure:"secret_key"`
	Bucket        string `mapstructure:"bucket"`
	UseSSL        bool   `mapstructure:"use_ssl"`
	PublicBaseURL string `mapstructure:"public_base_url"`
	PublicRead    bool   `mapstructure:"public_read"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8093
	}
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "postgres"
	}
	if cfg.Database.SQLitePath == "" {
		cfg.Database.SQLitePath = "./data/aftersalescore.db"
	}
	if cfg.Auth.JWTSecret == "" {
		cfg.Auth.JWTSecret = "change-me-in-production-use-long-random-string"
	}
	if cfg.Storage.Driver == "" {
		cfg.Storage.Driver = "minio"
	}
	if cfg.Storage.LocalPath == "" {
		cfg.Storage.LocalPath = "./data/uploads"
	}
	if cfg.Storage.PublicBaseURL == "" && cfg.Storage.Driver == "minio" {
		cfg.Storage.PublicBaseURL = "http://127.0.0.1:9100/aftersalescore"
	}
	if cfg.Storage.Prefix == "" {
		cfg.Storage.Prefix = "unboxing"
	}
	if cfg.Storage.MinIO.Prefix == "" {
		cfg.Storage.MinIO.Prefix = cfg.Storage.Prefix
	}
	if !v.IsSet("storage.minio.public_read") {
		cfg.Storage.MinIO.PublicRead = true
	}
	if len(cfg.CORS.AllowOrigins) == 0 {
		cfg.CORS.AllowOrigins = []string{
			"http://localhost:5176",
			"http://127.0.0.1:5176",
			"http://localhost:5174",
			"http://127.0.0.1:5174",
		}
	}
	if cfg.Edge.Schema == "" {
		cfg.Edge.Schema = "after_sales"
	}
	if cfg.Edge.Table == "" {
		cfg.Edge.Table = "edge_record"
	}
	if cfg.Edge.CloudEdgeID == "" {
		cfg.Edge.CloudEdgeID = "cloud"
	}
	if cfg.Edge.HealthPollSec == 0 {
		cfg.Edge.HealthPollSec = 30
	}
	if cfg.Edge.MinIO.Bucket == "" {
		cfg.Edge.MinIO.Bucket = "box-edge"
	}
	if cfg.Edge.MinIO.Endpoint == "" {
		cfg.Edge.MinIO.Endpoint = cfg.Storage.MinIO.Endpoint
		cfg.Edge.MinIO.AccessKey = cfg.Storage.MinIO.AccessKey
		cfg.Edge.MinIO.SecretKey = cfg.Storage.MinIO.SecretKey
		cfg.Edge.MinIO.UseSSL = cfg.Storage.MinIO.UseSSL
	}
	if cfg.Edge.MinIO.PublicBaseURL == "" && cfg.Edge.MinIO.Endpoint != "" {
		scheme := "http"
		if cfg.Edge.MinIO.UseSSL {
			scheme = "https"
		}
		cfg.Edge.MinIO.PublicBaseURL = fmt.Sprintf("%s://%s/%s", scheme, cfg.Edge.MinIO.Endpoint, cfg.Edge.MinIO.Bucket)
	}
	if !v.IsSet("edge.minio.public_read") {
		cfg.Edge.MinIO.PublicRead = true
	}
	return &cfg, nil
}
