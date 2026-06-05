// Package config carrega a configuração do servidor a partir de (em ordem de
// precedência crescente): valores padrão, arquivo TOML, variáveis de ambiente
// (BERSERKER_*) e flags de linha de comando.
package config

import (
	"flag"
	"os"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	MusicFolder        string        `toml:"music_folder"`
	DataFolder         string        `toml:"data_folder"`
	Port               int           `toml:"port"`
	BaseURL            string        `toml:"base_url"`
	LogLevel           string        `toml:"log_level"`
	ScanInterval       time.Duration `toml:"scan_interval"`
	ScanOnStart        bool          `toml:"scan_on_start"`
	JWTSecret          string        `toml:"jwt_secret"`
	AccessTokenTTL     time.Duration `toml:"access_token_ttl"`
	RefreshTokenTTL    time.Duration `toml:"refresh_token_ttl"`
	MediaTokenTTL      time.Duration `toml:"media_token_ttl"`
	AdminUser          string        `toml:"admin_user"`
	AdminPassword      string        `toml:"admin_password"`
	TranscodingEnabled bool          `toml:"transcoding_enabled"`
	FFmpegPath         string        `toml:"ffmpeg_path"`
	FFprobePath        string        `toml:"ffprobe_path"`
	WebappDir          string        `toml:"webapp_dir"`
	AllowedOrigins     string        `toml:"allowed_origins"`
}

func defaults() Config {
	return Config{
		MusicFolder:        "./music",
		DataFolder:         "./data",
		Port:               4533,
		BaseURL:            "",
		LogLevel:           "info",
		ScanInterval:       0, // 0 = desabilitado (scan só no boot)
		ScanOnStart:        true,
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTL:    30 * 24 * time.Hour,
		MediaTokenTTL:      6 * time.Hour,
		AdminUser:          "admin",
		AdminPassword:      "",
		TranscodingEnabled: true,
		FFmpegPath:         "ffmpeg",
		FFprobePath:        "ffprobe",
		WebappDir:          "",
		AllowedOrigins:     "",
	}
}

// Load resolve a configuração final.
func Load(args []string) (Config, error) {
	cfg := defaults()

	// 1) Arquivo de configuração (path via flag/env, resolvido em duas passadas).
	fs := flag.NewFlagSet("berserker", flag.ContinueOnError)
	configPath := fs.String("config", envStr("CONFIG", ""), "caminho do arquivo de configuração TOML")
	musicFolder := fs.String("music", "", "pasta da biblioteca de música")
	dataFolder := fs.String("data", "", "pasta de dados (db, cache)")
	port := fs.Int("port", 0, "porta HTTP")
	logLevel := fs.String("log-level", "", "nível de log (debug|info|warn|error)")
	adminPass := fs.String("admin-password", "", "senha inicial do admin (seed)")
	webappDir := fs.String("webapp-dir", "", "pasta do build estático do webapp a servir")
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}

	if *configPath != "" {
		if _, err := toml.DecodeFile(*configPath, &cfg); err != nil {
			return cfg, err
		}
	}

	// 2) Variáveis de ambiente (BERSERKER_*).
	applyEnv(&cfg)

	// 3) Flags explícitas vencem.
	if *musicFolder != "" {
		cfg.MusicFolder = *musicFolder
	}
	if *dataFolder != "" {
		cfg.DataFolder = *dataFolder
	}
	if *port != 0 {
		cfg.Port = *port
	}
	if *logLevel != "" {
		cfg.LogLevel = *logLevel
	}
	if *adminPass != "" {
		cfg.AdminPassword = *adminPass
	}
	if *webappDir != "" {
		cfg.WebappDir = *webappDir
	}
	return cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("BERSERKER_MUSIC_FOLDER"); v != "" {
		cfg.MusicFolder = v
	}
	if v := os.Getenv("BERSERKER_DATA_FOLDER"); v != "" {
		cfg.DataFolder = v
	}
	if v := os.Getenv("BERSERKER_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Port = n
		}
	}
	if v := os.Getenv("BERSERKER_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv("BERSERKER_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("BERSERKER_JWT_SECRET"); v != "" {
		cfg.JWTSecret = v
	}
	if v := os.Getenv("BERSERKER_ADMIN_USER"); v != "" {
		cfg.AdminUser = v
	}
	if v := os.Getenv("BERSERKER_ADMIN_PASSWORD"); v != "" {
		cfg.AdminPassword = v
	}
	if v := os.Getenv("BERSERKER_WEBAPP_DIR"); v != "" {
		cfg.WebappDir = v
	}
	if v := os.Getenv("BERSERKER_ALLOWED_ORIGINS"); v != "" {
		cfg.AllowedOrigins = v
	}
	if v := os.Getenv("BERSERKER_SCAN_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.ScanInterval = d
		}
	}
}

func envStr(key, def string) string {
	if v := os.Getenv("BERSERKER_" + key); v != "" {
		return v
	}
	return def
}
