package main

import (
	"embed"
	"os"

	"github.com/google/uuid"
)

//go:embed all:assets/monitor-presets
var presetAssets embed.FS

const (
	categoryCombat  = "combat"
	categoryStatus  = "status"
	categoryNetwork = "network"
	categoryCustom  = "custom"
)

type PresetDefinition struct {
	Key              string
	Name             string
	Category         string
	EmbedPath        string
	DefaultThreshold float64
}

var builtinPresets = []PresetDefinition{
	{Key: "declare_war", Name: "有人宣战", Category: categoryCombat, EmbedPath: "assets/monitor-presets/combat/declare_war.png", DefaultThreshold: 0.7},
	{Key: "under_attack", Name: "受到攻击", Category: categoryCombat, EmbedPath: "assets/monitor-presets/combat/under_attack.png", DefaultThreshold: 0.7},
	{Key: "death", Name: "死亡", Category: categoryStatus, EmbedPath: "assets/monitor-presets/status/death.png", DefaultThreshold: 0.85},
	{Key: "disconnect", Name: "掉线", Category: categoryNetwork, EmbedPath: "assets/monitor-presets/network/disconnect.png", DefaultThreshold: 0.85},
}

func findTemplateByPresetKey(templates []TemplateItem, key string) *TemplateItem {
	for i := range templates {
		if templates[i].PresetKey == key {
			return &templates[i]
		}
	}
	return nil
}

func migrateTemplateCategories(s *AppSettings) {
	for i := range s.Templates {
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

func syncBuiltinThresholds(s *AppSettings) bool {
	changed := false
	for i := range s.Templates {
		pk := s.Templates[i].PresetKey
		if pk == "" {
			continue
		}
		for _, def := range builtinPresets {
			if def.Key == pk && s.Templates[i].Threshold != def.DefaultThreshold {
				s.Templates[i].Threshold = def.DefaultThreshold
				changed = true
			}
		}
	}
	return changed
}

func ensureBuiltinTemplates(s *AppSettings) error {
	if err := ensureStoreDirs(); err != nil {
		return err
	}
	migrateTemplateCategories(s)

	changed := syncBuiltinThresholds(s)
	for _, def := range builtinPresets {
		if findTemplateByPresetKey(s.Templates, def.Key) != nil {
			continue
		}
		data, err := presetAssets.ReadFile(def.EmbedPath)
		if err != nil {
			return err
		}
		id := uuid.NewString()
		path, err := templateFilePath(id)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return err
		}
		s.Templates = append(s.Templates, TemplateItem{
			ID:        id,
			Name:      def.Name,
			File:      id + ".png",
			Threshold: def.DefaultThreshold,
			Enabled:   true,
			Category:  def.Category,
			PresetKey: def.Key,
		})
		changed = true
	}
	if changed {
		return saveSettings(*s)
	}
	return nil
}
