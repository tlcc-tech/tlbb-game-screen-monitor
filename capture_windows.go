//go:build windows

package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/png"
	"strings"

	"github.com/kbinani/screenshot"
)

func captureScreen(gameWindowTitle string) (image.Image, error) {
	title := strings.TrimSpace(gameWindowTitle)
	if title != "" {
		img, err := captureGameWindowByTitle(title)
		if err == nil && img != nil {
			return img, nil
		}
	}
	n := screenshot.NumActiveDisplays()
	if n < 1 {
		return nil, errors.New("未检测到显示器")
	}
	return screenshot.CaptureDisplay(0)
}

func captureScreenPNGBase64(gameWindowTitle string) (string, error) {
	img, err := captureScreen(gameWindowTitle)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
