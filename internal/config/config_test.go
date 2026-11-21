package config

import "testing"

func validConfig() *Config {
	return &Config{JWT: JWTConfig{SecretKey: "0123456789abcdef0123456789abcdef"}, Database: DatabaseConfig{Driver: "postgres"}, StorageProvider: "local", StorageQuotaBytes: 1, MaxUploadBytes: 50 << 20, CORSAllowedOrigins: []string{"http://localhost:5173"}}
}

func TestValidateRequiresStrongJWTAndPositiveLimits(t *testing.T) {
	if err := validConfig().validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	weak := validConfig()
	weak.JWT.SecretKey = "short"
	if err := weak.validate(); err == nil {
		t.Fatal("expected weak JWT secret to be rejected")
	}
	noQuota := validConfig()
	noQuota.StorageQuotaBytes = 0
	if err := noQuota.validate(); err == nil {
		t.Fatal("expected zero quota to be rejected")
	}
	noUpload := validConfig()
	noUpload.MaxUploadBytes = 0
	if err := noUpload.validate(); err == nil {
		t.Fatal("expected zero upload limit to be rejected")
	}
}
func TestValidatePostgresAndExplicitCORSOnly(t *testing.T) {
	mysql := validConfig()
	mysql.Database.Driver = "mysql"
	if err := mysql.validate(); err == nil {
		t.Fatal("expected unsupported DB driver to be rejected")
	}
	wildcard := validConfig()
	wildcard.CORSAllowedOrigins = []string{"*"}
	if err := wildcard.validate(); err == nil {
		t.Fatal("expected wildcard origin to be rejected")
	}
	badOrigin := validConfig()
	badOrigin.CORSAllowedOrigins = []string{"https://example.com/path"}
	if err := badOrigin.validate(); err == nil {
		t.Fatal("expected non-origin URL to be rejected")
	}
}

func TestValidateAIServiceConfiguration(t *testing.T) {
	good := validConfig()
	good.AIServiceURL = "http://ai-api:8000"
	good.AISharedSecret = "0123456789abcdef0123456789abcdef"
	if err := good.validate(); err != nil {
		t.Fatalf("valid AI service config rejected: %v", err)
	}
	disabled := validConfig()
	if err := disabled.validate(); err != nil {
		t.Fatalf("AI-disabled config rejected: %v", err)
	}
	missingSecret := validConfig()
	missingSecret.AIServiceURL = "http://ai-api:8000"
	if err := missingSecret.validate(); err == nil {
		t.Fatal("expected configured AI service without shared secret to be rejected")
	}
	shortSecret := validConfig()
	shortSecret.AISharedSecret = "short"
	if err := shortSecret.validate(); err == nil {
		t.Fatal("expected short AI shared secret to be rejected")
	}
	badURL := validConfig()
	badURL.AIServiceURL = "ftp://ai-api:8000"
	if err := badURL.validate(); err == nil {
		t.Fatal("expected unsupported AI service URL scheme to be rejected")
	}
	external := validConfig()
	external.AIServiceURL = "https://api.openai.com"
	if err := external.validate(); err == nil {
		t.Fatal("expected hosted AI service URL to be rejected")
	}
	private := validConfig()
	private.AIServiceURL = "http://192.168.1.20:8000"
	private.AISharedSecret = "0123456789abcdef0123456789abcdef"
	if err := private.validate(); err != nil {
		t.Fatalf("private internal AI service URL rejected: %v", err)
	}
}
