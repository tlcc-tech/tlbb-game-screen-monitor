package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	storeDirName     = "tlbb-game-screen-monitor"
	settingsFileName = "settings.json"
	templatesSubDir  = "templates"
)

type TemplateItem struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	File      string  `json:"file"`
	Threshold float64 `json:"threshold"`
	Enabled   bool    `json:"enabled"`
	Category  string  `json:"category"`
	PresetKey string  `json:"presetKey"`
}

type AppSettings struct {
	ChannelKey          string         `json:"channelKey"`
	PollIntervalSec     int            `json:"pollIntervalSec"`
	ConsecutiveHits     int            `json:"consecutiveHits"`
	PushCooldownMin     int            `json:"pushCooldownMin"`
	NetworkWaitMaxMin   int            `json:"networkWaitMaxMin"`
	PingHost            string         `json:"pingHost"`
	HttpProbeURL        string         `json:"httpProbeUrl"`
	UsePing             bool           `json:"usePing"`
	UseHttp             bool           `json:"useHttp"`
	GameWindowTitle     string         `json:"gameWindowTitle"`
	GameWindowHwnd      int64          `json:"gameWindowHwnd"`
	HotkeyStart         string         `json:"hotkeyStart"`
	HotkeyStop          string         `json:"hotkeyStop"`
	NotifyOnRecover     bool           `json:"notifyOnRecover"`
	Templates           []TemplateItem `json:"templates"`
}

func defaultSettings() AppSettings {
	return AppSettings{
		PollIntervalSec:   2,
		ConsecutiveHits:   1,
		PushCooldownMin:   10,
		NetworkWaitMaxMin: 30,
		PingHost:          defaultProbeHost,
		HttpProbeURL:      "https://" + defaultProbeHost,
		UsePing:           true,
		UseHttp:           true,
		HotkeyStart:       "Home",
		HotkeyStop:        "End",
		Templates:         []TemplateItem{},
	}
}

type persistedSettings struct {
	AppSettings
	UpdatedAt string `json:"updatedAt"`
}

func storeDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	if dir == "" {
		return "", errors.New("无法获取用户配置目录")
	}
	return filepath.Join(dir, storeDirName), nil
}

func templatesDir() (string, error) {
	dir, err := storeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, templatesSubDir), nil
}

func settingsPath() (string, error) {
	dir, err := storeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, settingsFileName), nil
}

func ensureStoreDirs() error {
	dir, err := storeDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tplDir, err := templatesDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(tplDir, 0o755)
}

func loadSettings() (AppSettings, error) {
	path, err := settingsPath()
	if err != nil {
		return defaultSettings(), err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			s := defaultSettings()
			_ = ensureBuiltinTemplates(&s)
			return s, nil
		}
		return defaultSettings(), err
	}
	var ps persistedSettings
	if err := json.Unmarshal(data, &ps); err != nil {
		return defaultSettings(), err
	}
	s := ps.AppSettings
	normalizeSettings(&s)
	_ = ensureBuiltinTemplates(&s)
	return s, nil
}

func saveSettings(s AppSettings) error {
	if err := ensureStoreDirs(); err != nil {
		return err
	}
	normalizeSettings(&s)
	path, err := settingsPath()
	if err != nil {
		return err
	}
	ps := persistedSettings{
		AppSettings: s,
		UpdatedAt:   time.Now().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(ps, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func normalizeSettings(s *AppSettings) {
	if s.PollIntervalSec < 1 {
		s.PollIntervalSec = 2
	}
	if s.PollIntervalSec > 60 {
		s.PollIntervalSec = 60
	}
	if s.ConsecutiveHits < 1 {
		s.ConsecutiveHits = 1
	}
	if s.ConsecutiveHits > 20 {
		s.ConsecutiveHits = 20
	}
	if s.PushCooldownMin < 1 {
		s.PushCooldownMin = 1
	}
	if s.NetworkWaitMaxMin < 1 {
		s.NetworkWaitMaxMin = 30
	}
	if s.PingHost == "" {
		s.PingHost = defaultProbeHost
	}
	if s.HttpProbeURL == "" {
		s.HttpProbeURL = "https://" + defaultProbeHost
	}
	if strings.TrimSpace(s.HotkeyStart) == "" {
		s.HotkeyStart = "Home"
	}
	if strings.TrimSpace(s.HotkeyStop) == "" {
		s.HotkeyStop = "End"
	}
	for i := range s.Templates {
		if s.Templates[i].Threshold <= 0 || s.Templates[i].Threshold > 1 {
			s.Templates[i].Threshold = 0.85
		}
		if s.Templates[i].Category == "" {
			if s.Templates[i].PresetKey != "" {
				for _, def := range builtinPresets {
					if def.Key == s.Templates[i].PresetKey {
						s.Templates[i].Category = def.Category
						break
					}
				}
			}
			if s.Templates[i].Category == "" {
				s.Templates[i].Category = categoryCustom
			}
		}
	}
}

func templateFilePath(id string) (string, error) {
	dir, err := templatesDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, id+".png"), nil
}
