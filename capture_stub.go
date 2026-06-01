//go:build !windows

package main

import (
	"errors"
	"image"
)

var errWindowsOnly = errors.New("截图功能仅支持 Windows")

func captureScreen(_ AppSettings) (image.Image, error) {
	return nil, errWindowsOnly
}

func captureScreenPNGBase64(_ AppSettings) (string, error) {
	return "", errWindowsOnly
}

func releaseDmCaptureBinding() {}
