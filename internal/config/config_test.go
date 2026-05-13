package config

import (
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\") returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load(\"\") returned nil config")
	}
	if cfg.RefreshInterval <= 0 {
		t.Errorf("RefreshInterval = %v, want > 0", cfg.RefreshInterval)
	}
	if cfg.ProcessLimit <= 0 {
		t.Errorf("ProcessLimit = %d, want > 0", cfg.ProcessLimit)
	}
}
