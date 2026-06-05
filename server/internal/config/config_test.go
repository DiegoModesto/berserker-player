package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 4533 || cfg.AccessTokenTTL != 15*time.Minute || !cfg.ScanOnStart {
		t.Fatalf("defaults inesperados: %+v", cfg)
	}
}

func TestFlagsOverride(t *testing.T) {
	cfg, err := Load([]string{"--port", "9000", "--music", "/m", "--data", "/d", "--log-level", "debug", "--admin-password", "x"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 9000 || cfg.MusicFolder != "/m" || cfg.DataFolder != "/d" || cfg.LogLevel != "debug" || cfg.AdminPassword != "x" {
		t.Fatalf("flags não aplicadas: %+v", cfg)
	}
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("BERSERKER_PORT", "7777")
	t.Setenv("BERSERKER_MUSIC_FOLDER", "/envmusic")
	t.Setenv("BERSERKER_WATCH", "true")
	t.Setenv("BERSERKER_SCAN_INTERVAL", "15m")
	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 7777 || cfg.MusicFolder != "/envmusic" || !cfg.Watch || cfg.ScanInterval != 15*time.Minute {
		t.Fatalf("env não aplicado: %+v", cfg)
	}
}

func TestFlagBeatsEnv(t *testing.T) {
	t.Setenv("BERSERKER_PORT", "7777")
	cfg, _ := Load([]string{"--port", "8888"})
	if cfg.Port != 8888 {
		t.Fatalf("flag deveria vencer env, %d", cfg.Port)
	}
}

func TestConfigFile(t *testing.T) {
	f := t.TempDir() + "/c.toml"
	_ = os.WriteFile(f, []byte("port = 6000\nmusic_folder = \"/tomlmusic\"\nlog_level = \"warn\"\n"), 0o644)
	cfg, err := Load([]string{"--config", f})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 6000 || cfg.MusicFolder != "/tomlmusic" || cfg.LogLevel != "warn" {
		t.Fatalf("toml não aplicado: %+v", cfg)
	}
}
