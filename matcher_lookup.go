package main

import (
	"image"

	"github.com/deluan/lookup"
)

func lookupMatch(screen image.Image, template image.Image) (float64, bool, error) {
	if screen == nil || template == nil {
		return 0, false, nil
	}

	l := lookup.NewLookup(screen)
	points, err := l.FindAll(template, 0.5)
	if err != nil {
		return 0, false, err
	}
	if len(points) == 0 {
		return 0, false, nil
	}

	best := points[0].G
	for _, p := range points[1:] {
		if p.G > best {
			best = p.G
		}
	}
	return best, true, nil
}
