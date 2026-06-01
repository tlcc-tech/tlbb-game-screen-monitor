//go:build windows

package main

import (
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32Pick              = windows.NewLazySystemDLL("user32.dll")
	procGetCursorPos        = user32Pick.NewProc("GetCursorPos")
	procWindowFromPoint     = user32Pick.NewProc("WindowFromPoint")
	procGetWindowThreadPID  = user32Pick.NewProc("GetWindowThreadProcessId")
	procGetClassNameW       = user32Pick.NewProc("GetClassNameW")
	procGetAncestor         = user32Pick.NewProc("GetAncestor")
)

const gaRoot = 2

type point struct {
	X, Y int32
}

func pickGameWindowInteractive() (WindowInfo, error) {
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if asyncKeyDown(0x01) { // VK_LBUTTON
			time.Sleep(50 * time.Millisecond)
			x, y, err := getCursorPos()
			if err != nil {
				return WindowInfo{}, err
			}
			hwnd := windowFromPoint(x, y)
			if hwnd == 0 {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			root := getAncestor(hwnd, gaRoot)
			if root != 0 {
				hwnd = root
			}
			title := getWindowText(windows.HWND(hwnd))
			if title == "" {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			className := getClassName(windows.HWND(hwnd))
			for asyncKeyDown(0x01) {
				time.Sleep(20 * time.Millisecond)
			}
			return WindowInfo{
				Hwnd:      int64(hwnd),
				Title:     title,
				ClassName: className,
			}, nil
		}
		time.Sleep(30 * time.Millisecond)
	}
	return WindowInfo{}, fmt.Errorf("选窗超时：请在提示后点击目标游戏窗口")
}

func asyncKeyDown(vk int) bool {
	r, _, _ := user32Pick.NewProc("GetAsyncKeyState").Call(uintptr(vk))
	return r&0x8000 != 0
}

func getCursorPos() (int32, int32, error) {
	var p point
	ok, _, err := procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	if ok == 0 {
		return 0, 0, err
	}
	return p.X, p.Y, nil
}

func windowFromPoint(x, y int32) uintptr {
	r, _, _ := procWindowFromPoint.Call(uintptr(x), uintptr(y))
	return r
}

func getAncestor(hwnd uintptr, ga uint32) uintptr {
	r, _, _ := procGetAncestor.Call(hwnd, uintptr(ga))
	return r
}

func getClassName(hwnd windows.HWND) string {
	buf := make([]uint16, 256)
	procGetClassNameW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return windows.UTF16ToString(buf)
}

func isWindowAlive(hwnd int64) bool {
	if hwnd == 0 {
		return false
	}
	ok, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	return ok != 0
}
