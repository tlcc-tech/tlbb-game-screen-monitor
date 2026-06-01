//go:build !windows

package main

import "errors"

func pickGameWindowInteractive() (WindowInfo, error) {
	return WindowInfo{}, errors.New("选窗功能仅支持 Windows")
}

func isWindowAlive(_ int64) bool { return false }
