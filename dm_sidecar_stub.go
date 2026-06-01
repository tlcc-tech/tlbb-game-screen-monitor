//go:build !windows

package main

func dmCaptureAvailable() bool { return false }

func dmBindWindow(hwnd int64) (string, error) {
	return "", nil
}

func dmCaptureBMP() ([]byte, error) {
	return nil, nil
}

func stopDMCaptureSidecar() {}
