//go:build !windows

package main

func dmFilesExist(settings AppSettings) bool {
	return false
}

func stopDMSidecar() {}

func (m *Monitor) matchAllDM(settings AppSettings, templates []TemplateItem) ([]MatchScore, bool) {
	return nil, false
}
