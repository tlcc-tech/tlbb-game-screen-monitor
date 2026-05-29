//go:build windows

package main

import (
	"errors"
	"image"
	"strings"
	"syscall"
	"unsafe"

	"github.com/kbinani/screenshot"
	"golang.org/x/sys/windows"
)

var (
	user32                 = windows.NewLazySystemDLL("user32.dll")
	procEnumWindows        = user32.NewProc("EnumWindows")
	procGetWindowTextW     = user32.NewProc("GetWindowTextW")
	procIsWindowVisible    = user32.NewProc("IsWindowVisible")
	procGetWindowRect      = user32.NewProc("GetWindowRect")
)

type rect struct {
	Left, Top, Right, Bottom int32
}

func captureGameWindowByTitle(titleSubstring string) (image.Image, error) {
	titleSubstring = strings.TrimSpace(titleSubstring)
	if titleSubstring == "" {
		return nil, errors.New("窗口标题为空")
	}

	var foundHWND windows.HWND
	cb := syscall.NewCallback(func(hwnd uintptr, lParam uintptr) uintptr {
		if !isWindowVisible(windows.HWND(hwnd)) {
			return 1
		}
		title := getWindowText(windows.HWND(hwnd))
		if title == "" {
			return 1
		}
		if strings.Contains(title, titleSubstring) {
			foundHWND = windows.HWND(hwnd)
			return 0
		}
		return 1
	})

	procEnumWindows.Call(cb, 0)
	if foundHWND == 0 {
		return nil, errors.New("未找到匹配的游戏窗口: " + titleSubstring)
	}

	r, err := getWindowRect(foundHWND)
	if err != nil {
		return nil, err
	}
	w := int(r.Right - r.Left)
	h := int(r.Bottom - r.Top)
	if w <= 0 || h <= 0 {
		return nil, errors.New("游戏窗口尺寸无效")
	}
	bounds := image.Rect(int(r.Left), int(r.Top), int(r.Right), int(r.Bottom))
	return screenshot.CaptureRect(bounds)
}

func isWindowVisible(hwnd windows.HWND) bool {
	r, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	return r != 0
}

func getWindowText(hwnd windows.HWND) string {
	buf := make([]uint16, 512)
	procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return windows.UTF16ToString(buf)
}

func getWindowRect(hwnd windows.HWND) (rect, error) {
	var r rect
	ok, _, err := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&r)))
	if ok == 0 {
		return r, err
	}
	return r, nil
}
