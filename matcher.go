package main

import "image"

// Matcher finds a template inside a screen image.
type Matcher interface {
	Find(screen image.Image, template image.Image) (score float64, found bool, err error)
}

// LookupMatcher uses NCC via github.com/deluan/lookup.
type LookupMatcher struct{}

func (LookupMatcher) Find(screen image.Image, template image.Image) (float64, bool, error) {
	return lookupMatch(screen, template)
}
