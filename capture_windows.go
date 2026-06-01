//go:build windows

package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/png"
	"strings"
	"sync"

	"github.com/kbinani/screenshot"
	"golang.org/x/image/bmp"
)

var (
	dmBindMu     sync.Mutex
	dmBoundHwnd  int64
	dmBoundMode  string
)

func captureScreen(settings AppSettings) (image.Image, error) {
	if settings.GameWindowHwnd != 0 && dmCaptureAvailable() {
		img, err := captureViaDmBind(settings.GameWindowHwnd)
		if err == nil && img != nil {
			return img, nil
		}
	}
	return captureScreenLegacy(settings.GameWindowTitle)
}

func captureViaDmBind(hwnd int64) (image.Image, error) {
	dmBindMu.Lock()
	defer dmBindMu.Unlock()

	if dmBoundHwnd != hwnd {
		stopDMCaptureSidecar()
		mode, err := dmBindWindow(hwnd)
		if err != nil {
			return nil, err
		}
		dmBoundHwnd = hwnd
		dmBoundMode = mode
	}

	data, err := dmCaptureBMP()
	if err != nil {
		return nil, err
	}
	return bmp.Decode(bytes.NewReader(data))
}

func releaseDmCaptureBinding() {
	dmBindMu.Lock()
	defer dmBindMu.Unlock()
	dmBoundHwnd = 0
	dmBoundMode = ""
	stopDMCaptureSidecar()
}

func captureScreenLegacy(gameWindowTitle string) (image.Image, error) {
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

func captureScreenPNGBase64(settings AppSettings) (string, error) {
	img, err := captureScreen(settings)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
