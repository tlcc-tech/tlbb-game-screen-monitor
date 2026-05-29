//go:build !windows

package main

import (
	"errors"
	"image"
)

var errWindowsOnly = errors.New("截图功能仅支持 Windows")

func captureScreen(_ string) (image.Image, error) {
	return nil, errWindowsOnly
}

func captureScreenPNGBase64(_ string) (string, error) {
	return "", errWindowsOnly
}
