package server

import (
	"encoding/json"
	"path/filepath"

	"media-sorter/internal/storage"
)

// GroupDef defines a metadata group (e.g. Subject, Tags, Quality)
type GroupDef struct {
	Name        string   `json:"name"`
	Key         string   `json:"key"`
	Type        string   `json:"type"`
	InputType   string   `json:"inputType"`
	Options     []string `json:"options"`
	AllowCustom bool     `json:"allowCustom"`
	Separator   string   `json:"separator"`
	Prefix      string   `json:"prefix"`
}

// ProjectConfig is the JSON config stored per media folder
type ProjectConfig struct {
	Version      int               `json:"version"`
	OutputFormat string            `json:"outputFormat"`
	OutputFolder string            `json:"outputFolder"`
	OutputMode   string            `json:"outputMode"`
	Groups       []GroupDef        `json:"groups"`
	Keybindings  map[string]string `json:"keybindings,omitempty"`
}

func defaultConfig() ProjectConfig {
	return ProjectConfig{
		Version:      1,
		OutputFormat: "{basename}__{S}__{tags}__{quality}.{ext}",
		OutputFolder: "",
		OutputMode:   "rename",
		Groups: []GroupDef{
			{
				Name:        "Subject",
				Key:         "S",
				Type:        "multi-select",
				InputType:   "number",
				Options:     []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15", "16", "17", "18", "19", "20", "21", "22", "23", "24", "25"},
				AllowCustom: true,
				Separator:   "__",
				Prefix:      "S",
			},
			{
				Name:        "Tags",
				Key:         "tags",
				Type:        "multi-select",
				InputType:   "text",
				Options:     []string{"single", "double", "triple", "home-run", "strikeout", "walk", "steal", "catch", "dive", "throw", "slide", "bunt", "sac-fly", "error", "celebration", "pitching", "hitting", "fielding", "running", "warmup"},
				AllowCustom: true,
				Separator:   "_",
				Prefix:      "",
			},
			{
				Name:        "Quality",
				Key:         "quality",
				Type:        "single-select",
				InputType:   "slider",
				Options:     []string{"bad", "ok", "good", "great"},
				AllowCustom: false,
				Separator:   "",
				Prefix:      "",
			},
		},
	}
}

const configFileName = "media-sorter-config.json"

func loadConfig(store storage.Provider, dir string) (ProjectConfig, error) {
	configPath := filepath.Join(dir, configFileName)

	if data, err := store.ReadFile(configPath); err == nil {
		var cfg ProjectConfig
		if err := json.Unmarshal(data, &cfg); err == nil {
			return cfg, nil
		}
	}

	cfg := defaultConfig()
	saveConfig(store, dir, cfg)
	return cfg, nil
}

func saveConfig(store storage.Provider, dir string, cfg ProjectConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return store.WriteFile(filepath.Join(dir, configFileName), data)
}

