package config

import "testing"

func TestValidateAllowsDevDefaults(t *testing.T) {
	cfg := Config{
		AppEnv:        "dev",
		JWTSecret:     defaultJWTSecret,
		EncryptionKey: defaultEncryptionKey,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected dev defaults to validate, got %v", err)
	}
}

func TestValidateRejectsProductionDefaultSecrets(t *testing.T) {
	cfg := Config{
		AppEnv:        "prod",
		JWTSecret:     defaultJWTSecret,
		EncryptionKey: defaultEncryptionKey,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected production defaults to be rejected")
	}
}

func TestValidateRejectsInvalidEncryptionKeyLength(t *testing.T) {
	cfg := Config{
		AppEnv:        "dev",
		JWTSecret:     defaultJWTSecret,
		EncryptionKey: "short",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid encryption key length to be rejected")
	}
}
