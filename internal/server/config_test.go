package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"media-sorter/internal/storage"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}
	if cfg.OutputMode != "rename" {
		t.Errorf("outputMode = %q, want 'rename'", cfg.OutputMode)
	}
	if len(cfg.Groups) != 3 {
		t.Fatalf("groups count = %d, want 3", len(cfg.Groups))
	}
	if cfg.Groups[0].Key != "S" {
		t.Errorf("first group key = %q, want 'S'", cfg.Groups[0].Key)
	}
	if cfg.Groups[1].Key != "tags" {
		t.Errorf("second group key = %q, want 'tags'", cfg.Groups[1].Key)
	}
	if cfg.Groups[2].Key != "quality" {
		t.Errorf("third group key = %q, want 'quality'", cfg.Groups[2].Key)
	}
}

func TestLoadConfigDefault(t *testing.T) {
	dir := t.TempDir()
	ls := &storage.LocalStorage{}

	cfg, err := loadConfig(ls, dir)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}

	// Should have created the config file
	configPath := filepath.Join(dir, configFileName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file should have been created")
	}
}

func TestLoadConfigExisting(t *testing.T) {
	dir := t.TempDir()
	ls := &storage.LocalStorage{}

	cfg := ProjectConfig{
		Version:      1,
		OutputFormat: "{basename}.{ext}",
		OutputMode:   "copy",
		Groups:       []GroupDef{{Name: "Test", Key: "T", Type: "single-select"}},
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(filepath.Join(dir, configFileName), data, 0644)

	loaded, err := loadConfig(ls, dir)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}

	if loaded.OutputFormat != "{basename}.{ext}" {
		t.Errorf("outputFormat = %q, want '{basename}.{ext}'", loaded.OutputFormat)
	}
	if loaded.OutputMode != "copy" {
		t.Errorf("outputMode = %q, want 'copy'", loaded.OutputMode)
	}
	if len(loaded.Groups) != 1 || loaded.Groups[0].Key != "T" {
		t.Errorf("groups = %v, want single group with key 'T'", loaded.Groups)
	}
}

func TestSaveConfig(t *testing.T) {
	dir := t.TempDir()
	ls := &storage.LocalStorage{}

	cfg := ProjectConfig{
		Version:      1,
		OutputFormat: "{basename}.{ext}",
		OutputMode:   "rename",
		Groups:       []GroupDef{{Name: "Test", Key: "test"}},
	}

	if err := saveConfig(ls, dir, cfg); err != nil {
		t.Fatalf("saveConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, configFileName))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	var loaded ProjectConfig
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if loaded.OutputFormat != cfg.OutputFormat {
		t.Errorf("saved config mismatch")
	}
}
