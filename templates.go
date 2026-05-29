package main

import (
	"encoding/base64"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type MatchScore struct {
	TemplateID   string  `json:"templateId"`
	TemplateName string  `json:"templateName"`
	Score        float64 `json:"score"`
	Found        bool    `json:"found"`
	Matched      bool    `json:"matched"`
}

func (m *Monitor) ListTemplates() []TemplateItem {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]TemplateItem, len(m.settings.Templates))
	copy(out, m.settings.Templates)
	return out
}

func (m *Monitor) SaveTemplate(name string, pngBase64 string, threshold float64) (TemplateItem, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return TemplateItem{}, errors.New("模板名称不能为空")
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(pngBase64))
	if err != nil {
		return TemplateItem{}, errors.New("图片数据无效")
	}
	if len(data) == 0 {
		return TemplateItem{}, errors.New("图片数据为空")
	}
	if err := ensureStoreDirs(); err != nil {
		return TemplateItem{}, err
	}

	id := uuid.NewString()
	path, err := templateFilePath(id)
	if err != nil {
		return TemplateItem{}, err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return TemplateItem{}, err
	}

	if threshold <= 0 || threshold > 1 {
		threshold = 0.85
	}

	item := TemplateItem{
		ID:        id,
		Name:      name,
		File:      filepath.Base(path),
		Threshold: threshold,
		Enabled:   true,
	}

	m.mu.Lock()
	m.settings.Templates = append(m.settings.Templates, item)
	err = saveSettings(m.settings)
	m.mu.Unlock()
	if err != nil {
		return TemplateItem{}, err
	}
	return item, nil
}

func (m *Monitor) DeleteTemplate(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("模板 ID 无效")
	}
	path, err := templateFilePath(id)
	if err != nil {
		return err
	}
	_ = os.Remove(path)

	m.mu.Lock()
	defer m.mu.Unlock()
	filtered := m.settings.Templates[:0]
	for _, t := range m.settings.Templates {
		if t.ID != id {
			filtered = append(filtered, t)
		}
	}
	m.settings.Templates = filtered
	return saveSettings(m.settings)
}

func (m *Monitor) UpdateTemplate(item TemplateItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, t := range m.settings.Templates {
		if t.ID == item.ID {
			if item.Threshold <= 0 || item.Threshold > 1 {
				item.Threshold = 0.85
			}
			if strings.TrimSpace(item.Name) != "" {
				t.Name = strings.TrimSpace(item.Name)
			}
			t.Threshold = item.Threshold
			t.Enabled = item.Enabled
			m.settings.Templates[i] = t
			return saveSettings(m.settings)
		}
	}
	return errors.New("模板不存在")
}

func (m *Monitor) ImportTemplateFromPath(name string, srcPath string, threshold float64) (TemplateItem, error) {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return TemplateItem{}, err
	}
	return m.SaveTemplate(name, base64.StdEncoding.EncodeToString(data), threshold)
}

func (m *Monitor) loadTemplateImage(id string) (image.Image, *TemplateItem, error) {
	m.mu.Lock()
	var item *TemplateItem
	for i := range m.settings.Templates {
		if m.settings.Templates[i].ID == id {
			cp := m.settings.Templates[i]
			item = &cp
			break
		}
	}
	m.mu.Unlock()
	if item == nil {
		return nil, nil, errors.New("模板不存在")
	}
	path, err := templateFilePath(id)
	if err != nil {
		return nil, nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, nil, err
	}
	return img, item, nil
}

func (m *Monitor) TestMatch() ([]MatchScore, error) {
	screen, err := captureScreen(m.getGameWindowTitle())
	if err != nil {
		return nil, err
	}
	return m.matchAll(screen), nil
}

func (m *Monitor) matchAll(screen image.Image) []MatchScore {
	matcher := LookupMatcher{}
	m.mu.Lock()
	templates := append([]TemplateItem(nil), m.settings.Templates...)
	m.mu.Unlock()

	var results []MatchScore
	for _, tpl := range templates {
		if !tpl.Enabled {
			continue
		}
		img, item, err := m.loadTemplateImage(tpl.ID)
		if err != nil || item == nil {
			continue
		}
		score, found, err := matcher.Find(screen, img)
		if err != nil {
			continue
		}
		matched := found && score >= item.Threshold
		results = append(results, MatchScore{
			TemplateID:   item.ID,
			TemplateName: item.Name,
			Score:        score,
			Found:        found,
			Matched:      matched,
		})
	}
	return results
}

func (m *Monitor) getGameWindowTitle() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.settings.GameWindowTitle
}

func (m *Monitor) GetTemplateThumbnailBase64(id string) (string, error) {
	path, err := templateFilePath(id)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}
