//go:build !windows

package main

import "image"

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
			Category:     item.Category,
			PresetKey:    item.PresetKey,
			Score:        score,
			Found:        found,
			Matched:      matched,
		})
	}
	return results
}
