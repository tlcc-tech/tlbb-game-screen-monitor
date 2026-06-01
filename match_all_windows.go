//go:build windows

package main

import (
	"image"

	"gocv.io/x/gocv"
)

func (m *Monitor) matchAll(screen image.Image) []MatchScore {
	screenGray, err := imageToGrayMat(screen)
	if err != nil {
		return nil
	}
	defer screenGray.Close()

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
		tplGray, err := imageToGrayMat(img)
		if err != nil {
			continue
		}
		score, found, err := matchTemplateScore(screenGray, tplGray)
		tplGray.Close()
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

func imageToGrayMat(img image.Image) (gocv.Mat, error) {
	gray := gocv.NewMat()

	var mat gocv.Mat
	var err error
	switch typed := img.(type) {
	case *image.Gray:
		mat, err = gocv.ImageGrayToMatGray(typed)
	default:
		mat, err = gocv.ImageToMatRGBA(img)
	}
	if err != nil {
		gray.Close()
		return gray, err
	}
	defer mat.Close()

	switch mat.Channels() {
	case 1:
		mat.CopyTo(&gray)
	case 4:
		gocv.CvtColor(mat, &gray, gocv.ColorBGRAToGray)
	default:
		gocv.CvtColor(mat, &gray, gocv.ColorBGRToGray)
	}
	return gray, nil
}

func matchTemplateScore(screenGray gocv.Mat, tplGray gocv.Mat) (float64, bool, error) {
	if screenGray.Empty() || tplGray.Empty() {
		return 0, false, nil
	}
	if screenGray.Cols() < tplGray.Cols() || screenGray.Rows() < tplGray.Rows() {
		return 0, false, nil
	}

	result := gocv.NewMat()
	defer result.Close()

	mask := gocv.NewMat()
	defer mask.Close()

	gocv.MatchTemplate(screenGray, tplGray, &result, gocv.TmCcoeffNormed, mask)
	_, maxVal, _, _ := gocv.MinMaxLoc(result)
	return float64(maxVal), true, nil
}
