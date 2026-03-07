package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"video-sorter/internal/storage"
)

func TestParseConfigText(t *testing.T) {
	text := `# Subjects
Alice
Bob
Charlie

# Tags
slide
catch
throw
`
	subjects, tags := parseConfigText(text)

	wantSubjects := []string{"Alice", "Bob", "Charlie"}
	wantTags := []string{"slide", "catch", "throw"}

	if len(subjects) != len(wantSubjects) {
		t.Fatalf("subjects count = %d, want %d", len(subjects), len(wantSubjects))
	}
	for i, s := range subjects {
		if s != wantSubjects[i] {
			t.Errorf("subjects[%d] = %q, want %q", i, s, wantSubjects[i])
		}
	}

	if len(tags) != len(wantTags) {
		t.Fatalf("tags count = %d, want %d", len(tags), len(wantTags))
	}
	for i, tag := range tags {
		if tag != wantTags[i] {
			t.Errorf("tags[%d] = %q, want %q", i, tag, wantTags[i])
		}
	}
}

func TestParseConfigTextPlayers(t *testing.T) {
	// "# Players" is an alias for "# Subjects"
	text := `# Players
Player1
Player2

# Tags
tag1
`
	subjects, tags := parseConfigText(text)
	if len(subjects) != 2 || subjects[0] != "Player1" || subjects[1] != "Player2" {
		t.Errorf("subjects = %v, want [Player1 Player2]", subjects)
	}
	if len(tags) != 1 || tags[0] != "tag1" {
		t.Errorf("tags = %v, want [tag1]", tags)
	}
}

func TestParseConfigTextEmpty(t *testing.T) {
	subjects, tags := parseConfigText("")
	if len(subjects) != 0 {
		t.Errorf("expected no subjects, got %v", subjects)
	}
	if len(tags) != 0 {
		t.Errorf("expected no tags, got %v", tags)
	}
}

func TestParseConfigTextCommentsOnly(t *testing.T) {
	text := `# This is a comment
# Another comment
`
	subjects, tags := parseConfigText(text)
	if len(subjects) != 0 || len(tags) != 0 {
		t.Errorf("comments-only should return empty, got subjects=%v tags=%v", subjects, tags)
	}
}

func TestParseConfigTextBlankLines(t *testing.T) {
	text := `# Subjects

Alice

Bob

# Tags

slide

`
	subjects, tags := parseConfigText(text)
	if len(subjects) != 2 {
		t.Errorf("subjects = %v, want 2 entries", subjects)
	}
	if len(tags) != 1 {
		t.Errorf("tags = %v, want 1 entry", tags)
	}
}

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

func TestLoadConfigLegacyJSON(t *testing.T) {
	dir := t.TempDir()
	ls := &storage.LocalStorage{}

	cfg := ProjectConfig{
		Version:      1,
		OutputFormat: "{basename}__{tags}.{ext}",
		OutputMode:   "rename",
		Groups:       []GroupDef{{Name: "Tags", Key: "tags", Type: "multi-select"}},
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(filepath.Join(dir, "video-sorter-config.json"), data, 0644)

	loaded, err := loadConfig(ls, dir)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}

	if loaded.OutputFormat != "{basename}__{tags}.{ext}" {
		t.Errorf("should load legacy JSON config")
	}

	// Should have migrated to new name
	if _, err := os.Stat(filepath.Join(dir, configFileName)); os.IsNotExist(err) {
		t.Error("should have created new config file during migration")
	}
}

func TestLoadConfigLegacyTxt(t *testing.T) {
	dir := t.TempDir()
	ls := &storage.LocalStorage{}

	txt := `# Subjects
Alice
Bob

# Tags
slide
catch
`
	os.WriteFile(filepath.Join(dir, "video-sorter-config.txt"), []byte(txt), 0644)

	loaded, err := loadConfig(ls, dir)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}

	if loaded.Groups[0].Options[0] != "Alice" || loaded.Groups[0].Options[1] != "Bob" {
		t.Errorf("subjects not loaded from txt: %v", loaded.Groups[0].Options)
	}
	if loaded.Groups[1].Options[0] != "slide" || loaded.Groups[1].Options[1] != "catch" {
		t.Errorf("tags not loaded from txt: %v", loaded.Groups[1].Options)
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
