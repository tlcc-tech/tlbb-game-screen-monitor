//go:build windows

package main

import (
	"image"
	"os"
)

func (m *Monitor) matchAll(screen image.Image) []MatchScore {
	m.mu.Lock()
	settings := m.settings
	templates := append([]TemplateItem(nil), m.settings.Templates...)
	m.mu.Unlock()

	if settings.UseDmMatcher && dmFilesExist(settings) {
		if scores, ok := m.matchAllDM(settings, templates); ok {
			return scores
		}
	}
	return m.matchAllLookup(screen, templates)
}

func (m *Monitor) matchAllLookup(screen image.Image, templates []TemplateItem) []MatchScore {
	matcher := LookupMatcher{}
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

func (m *Monitor) matchAllDM(settings AppSettings, templates []TemplateItem) ([]MatchScore, bool) {
	if err := ensureDMSidecar(settings); err != nil {
		return nil, false
	}

	rect, err := getSearchRect(settings.GameWindowTitle)
	if err != nil {
		return nil, false
	}

	var results []MatchScore
	for _, tpl := range templates {
		if !tpl.Enabled {
			continue
		}
		bmpName := tpl.ID + ".bmp"
		bmpPath, err := templateBmpPath(tpl.ID)
		if err != nil {
			continue
		}
		if _, err := os.Stat(bmpPath); err != nil {
			continue
		}

		resp, err := dmFindPic(rect.Left, rect.Top, rect.Right, rect.Bottom, bmpName, tpl.Threshold)
		if err != nil {
			return nil, false
		}
		score := 0.0
		found := resp.Found
		if found {
			score = resp.Sim
			if score <= 0 {
				score = tpl.Threshold
			}
		}
		matched := found && score >= tpl.Threshold
		results = append(results, MatchScore{
			TemplateID:   tpl.ID,
			TemplateName: tpl.Name,
			Score:        score,
			Found:        found,
			Matched:      matched,
		})
	}
	return results, true
}
