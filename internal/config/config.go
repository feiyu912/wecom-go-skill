package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	envConfigDir  = "WECOM_GO_CONFIG_DIR"
	envCorpID     = "WECOM_CORP_ID"
	envCorpSecret = "WECOM_CORP_SECRET"
	envBaseURL    = "WECOM_BASE_URL"
	envTimeout    = "WECOM_TIMEOUT"
)

type FileConfig struct {
	CorpID     string `json:"corp_id"`
	CorpSecret string `json:"corp_secret"`
	BaseURL    string `json:"base_url"`
	TimeoutSec int    `json:"timeout_sec"`
}

type EffectiveConfig struct {
	CorpID     string
	CorpSecret string
	BaseURL    string
	Timeout    time.Duration
}

func ConfigDir() (string, error) {
	if override := strings.TrimSpace(os.Getenv(envConfigDir)); override != "" {
		return override, nil
	}

	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "wecom-go"), nil
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func LoadFile() (FileConfig, error) {
	path, err := ConfigPath()
	if err != nil {
		return FileConfig{}, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return FileConfig{}, nil
	}
	if err != nil {
		return FileConfig{}, err
	}

	var cfg FileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return FileConfig{}, err
	}
	return cfg, nil
}

func SaveFile(cfg FileConfig) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func ClearFile() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); errors.Is(err, os.ErrNotExist) {
		return nil
	} else {
		return err
	}
}

func Resolve() (EffectiveConfig, error) {
	fileCfg, err := LoadFile()
	if err != nil {
		return EffectiveConfig{}, err
	}

	timeoutSec := fileCfg.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	if raw := strings.TrimSpace(os.Getenv(envTimeout)); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return EffectiveConfig{}, err
		}
		timeoutSec = parsed
	}

	cfg := EffectiveConfig{
		CorpID:     firstNonEmpty(os.Getenv(envCorpID), fileCfg.CorpID),
		CorpSecret: firstNonEmpty(os.Getenv(envCorpSecret), fileCfg.CorpSecret),
		BaseURL:    firstNonEmpty(os.Getenv(envBaseURL), fileCfg.BaseURL, "https://qyapi.weixin.qq.com"),
		Timeout:    time.Duration(timeoutSec) * time.Second,
	}

	if strings.TrimSpace(cfg.CorpID) == "" || strings.TrimSpace(cfg.CorpSecret) == "" {
		return EffectiveConfig{}, errors.New("missing corp credentials; run `wecom-go config set` or set WECOM_CORP_ID / WECOM_CORP_SECRET")
	}

	return cfg, nil
}

func MaskSecret(secret string) string {
	trimmed := strings.TrimSpace(secret)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= 8 {
		return "********"
	}
	return trimmed[:4] + strings.Repeat("*", len(trimmed)-8) + trimmed[len(trimmed)-4:]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
