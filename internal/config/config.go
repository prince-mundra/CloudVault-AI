package config

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Server             ServerConfig
	Database           DatabaseConfig
	JWT                JWTConfig
	StorageProvider    string
	LocalStorageRoot   string
	StorageQuotaBytes  int64
	MaxUploadBytes     int64
	CORSAllowedOrigins []string
	AIServiceURL       string
	AISharedSecret     string
	GoogleCloud        GoogleCloudConfig
	AWS                AWSConfig
}

type ServerConfig struct {
	Port string
	Env  string
}
type DatabaseConfig struct {
	Driver   string
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

func (d DatabaseConfig) DSN() string {
	u := &url.URL{Scheme: "postgres", User: url.UserPassword(d.User, d.Password), Host: net.JoinHostPort(d.Host, d.Port), Path: "/" + d.Name}
	query := url.Values{}
	query.Set("sslmode", d.SSLMode)
	u.RawQuery = query.Encode()
	return u.String()
}

type JWTConfig struct{ SecretKey string }
type GoogleCloudConfig struct {
	ProjectID       string
	BucketName      string
	CredentialsFile string
}
type AWSConfig struct {
	Region          string
	BucketName      string
	AccessKeyID     string
	SecretAccessKey string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading configuration from environment")
	}
	quotaBytes, err := envInt64("STORAGE_QUOTA_BYTES", 5<<30)
	if err != nil {
		return nil, err
	}
	maxUploadBytes, err := envInt64("MAX_UPLOAD_BYTES", 50<<20)
	if err != nil {
		return nil, err
	}
	origins := splitOrigins(getEnvOrDefault("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173"))
	cfg := &Config{
		Server:          ServerConfig{Port: getEnvOrDefault("SERVER_PORT", "8080"), Env: getEnvOrDefault("APP_ENV", "development")},
		Database:        DatabaseConfig{Driver: getEnvOrDefault("DB_DRIVER", "postgres"), Host: getEnvOrDefault("DB_HOST", "localhost"), Port: getEnvOrDefault("DB_PORT", "5432"), Name: getEnvOrDefault("DB_DATABASE", "gostore"), User: getEnvOrDefault("DB_USERNAME", "postgres"), Password: os.Getenv("DB_PASSWORD"), SSLMode: getEnvOrDefault("DB_SSL_MODE", "disable")},
		JWT:             JWTConfig{SecretKey: os.Getenv("JWT_SECRET_KEY")},
		StorageProvider: getEnvOrDefault("STORAGE_PROVIDER", "local"), LocalStorageRoot: getEnvOrDefault("LOCAL_STORAGE_ROOT", "./data/files"),
		StorageQuotaBytes: quotaBytes, MaxUploadBytes: maxUploadBytes, CORSAllowedOrigins: origins,
		AIServiceURL: os.Getenv("AI_SERVICE_URL"), AISharedSecret: os.Getenv("AI_SHARED_SECRET"),
		GoogleCloud: GoogleCloudConfig{ProjectID: os.Getenv("GOOGLE_CLOUD_PROJECT_ID"), BucketName: os.Getenv("GOOGLE_CLOUD_BUCKET_NAME"), CredentialsFile: os.Getenv("GOOGLE_CLOUD_CREDENTIALS_FILE")},
		AWS:         AWSConfig{Region: getEnvOrDefault("AWS_REGION", "us-east-1"), BucketName: os.Getenv("AWS_BUCKET_NAME"), AccessKeyID: os.Getenv("AWS_ACCESS_KEY_ID"), SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY")},
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	return cfg, nil
}

func envInt64(name string, fallback int64) (int64, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}
	return v, nil
}
func splitOrigins(raw string) []string {
	out := make([]string, 0)
	for _, item := range strings.Split(raw, ",") {
		if v := strings.TrimSpace(item); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func (c *Config) validate() error {
	if c.JWT.SecretKey == "" {
		return fmt.Errorf("JWT_SECRET_KEY is required")
	}
	if len(c.JWT.SecretKey) < 32 {
		return fmt.Errorf("JWT_SECRET_KEY must be at least 32 bytes")
	}
	if c.StorageQuotaBytes <= 0 {
		return fmt.Errorf("STORAGE_QUOTA_BYTES must be greater than zero")
	}
	if c.MaxUploadBytes <= 0 {
		return fmt.Errorf("MAX_UPLOAD_BYTES must be greater than zero")
	}
	if c.MaxUploadBytes > 1<<30 {
		return fmt.Errorf("MAX_UPLOAD_BYTES must not exceed 1 GiB")
	}
	if c.Database.Driver != "postgres" {
		return fmt.Errorf("DB_DRIVER=%q is unsupported; PostgreSQL is the supported database", c.Database.Driver)
	}
	if c.StorageProvider != "local" && c.StorageProvider != "s3" && c.StorageProvider != "gcs" {
		return fmt.Errorf("STORAGE_PROVIDER must be local, s3, or gcs")
	}
	if c.AISharedSecret != "" && len(c.AISharedSecret) < 32 {
		return fmt.Errorf("AI_SHARED_SECRET must be at least 32 bytes when configured")
	}
	if c.AIServiceURL != "" && len(c.AISharedSecret) < 32 {
		return fmt.Errorf("AI_SHARED_SECRET must be at least 32 bytes when AI_SERVICE_URL is configured")
	}
	if c.AIServiceURL != "" {
		u, err := url.Parse(c.AIServiceURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("AI_SERVICE_URL must be an HTTP(S) origin without credentials, path, or query parameters")
		}
		host := strings.ToLower(u.Hostname())
		ip := net.ParseIP(host)
		localHost := host == "localhost" || host == "host.docker.internal" || strings.HasSuffix(host, ".local") || !strings.Contains(host, ".")
		if ip != nil {
			localHost = ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
		}
		if !localHost {
			return fmt.Errorf("AI_SERVICE_URL must target a local or private service; hosted AI APIs are not supported")
		}
	}
	for _, origin := range c.CORSAllowedOrigins {
		if origin == "*" {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS must not contain wildcard origins")
		}
		u, err := url.Parse(origin)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("invalid CORS origin %q", origin)
		}
	}
	if c.Database.Password == "" {
		log.Println("Warning: DB_PASSWORD is not set")
	}
	return nil
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
