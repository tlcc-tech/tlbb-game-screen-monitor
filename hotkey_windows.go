//go:build windows

package main

import (
	"fmt"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	wmHotkey      = 0x0312
	hotkeyIDStart = 1
	hotkeyIDStop  = 2
)

var (
	hotkeyMu       sync.Mutex
	hotkeyMonitor  *Monitor
	hotkeyApp      *App
	hotkeyStartStr string
	hotkeyStopStr  string
)

var vkNames = map[string]int{
	"Home": 0x24, "End": 0x23, "Insert": 0x2D, "Delete": 0x2E,
	"PageUp": 0x21, "PageDown": 0x22,
	"F1": 0x70, "F2": 0x71, "F3": 0x72, "F4": 0x73,
	"F5": 0x74, "F6": 0x75, "F7": 0x76, "F8": 0x77,
	"F9": 0x78, "F10": 0x79, "F11": 0x7A, "F12": 0x7B,
	"Pause": 0x13, "ScrollLock": 0x91,
}

func parseHotkey(name string) (uint32, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("热键名为空")
	}
	if vk, ok := vkNames[name]; ok {
		return uint32(vk), nil
	}
	if len(name) == 1 {
		c := strings.ToUpper(name)[0]
		if c >= 'A' && c <= 'Z' {
			return uint32(c), nil
		}
		if c >= '0' && c <= '9' {
			return uint32(c), nil
		}
	}
	return 0, fmt.Errorf("不支持的热键: %s", name)
}

func setupHotkeys(app *App, m *Monitor) {
	hotkeyMu.Lock()
	hotkeyApp = app
	hotkeyMonitor = m
	s := m.GetSettings()
	hotkeyStartStr = s.HotkeyStart
	hotkeyStopStr = s.HotkeyStop
	hotkeyMu.Unlock()

	go hotkeyMessageLoop(s.HotkeyStart, s.HotkeyStop)
}

func refreshHotkeys(m *Monitor) {
	s := m.GetSettings()
	hotkeyMu.Lock()
	hotkeyStartStr = s.HotkeyStart
	hotkeyStopStr = s.HotkeyStop
	hotkeyMu.Unlock()
	// loop reads updated strings on next registration cycle — restart via stop/start in SaveSettings path
}

func stopHotkeys() {
	// message loop exits when hwnd destroyed on quit; best-effort unregister happens in loop teardown
}

func hotkeyMessageLoop(startKey, stopKey string) {
	user32 := windows.NewLazySystemDLL("user32.dll")
	registerHotKey := user32.NewProc("RegisterHotKey")
	unregisterHotKey := user32.NewProc("UnregisterHotKey")
	createWindowEx := user32.NewProc("CreateWindowExW")
	destroyWindow := user32.NewProc("DestroyWindow")
	getMessage := user32.NewProc("GetMessageW")
	translateMessage := user32.NewProc("TranslateMessage")
	dispatchMessage := user32.NewProc("DispatchMessageW")

	className, _ := windows.UTF16PtrFromString("tlbbHotkeyClass")
	wndClass := struct {
		style         uint32
		lpfnWndProc   uintptr
		cbClsExtra    int32
		cbWndExtra    int32
		hInstance     windows.Handle
		hIcon         windows.Handle
		hCursor       windows.Handle
		hbrBackground windows.Handle
		lpszMenuName  *uint16
		lpszClassName *uint16
	}{
		lpfnWndProc:   syscall.NewCallback(hotkeyWndProc),
		hInstance:     windows.Handle(0),
		lpszClassName: className,
	}

	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	getModuleHandle := kernel32.NewProc("GetModuleHandleW")
	hInst, _, _ := getModuleHandle.Call(0)
	wndClass.hInstance = windows.Handle(hInst)

	registerClass := user32.NewProc("RegisterClassW")
	registerClass.Call(uintptr(unsafe.Pointer(&wndClass)))

	title, _ := windows.UTF16PtrFromString("tlbbHotkey")
	hwnd, _, _ := createWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		0, 0, 0, 0, 0,
		0xFFFFFFFD, // HWND_MESSAGE
		0,
		uintptr(hInst),
		0,
	)
	if hwnd == 0 {
		return
	}
	defer destroyWindow.Call(hwnd)

	startVK, err := parseHotkey(startKey)
	if err == nil {
		registerHotKey.Call(hwnd, hotkeyIDStart, 0, uintptr(startVK))
	}
	stopVK, err2 := parseHotkey(stopKey)
	if err2 == nil {
		registerHotKey.Call(hwnd, hotkeyIDStop, 0, uintptr(stopVK))
	}
	defer func() {
		unregisterHotKey.Call(hwnd, hotkeyIDStart)
		unregisterHotKey.Call(hwnd, hotkeyIDStop)
	}()

	var msg struct {
		hwnd    windows.Handle
		message uint32
		wParam  uintptr
		lParam  uintptr
		time    uint32
		pt      point
	}
	for {
		ret, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if ret == 0 || ret == ^uintptr(0) {
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func hotkeyWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	user32 := windows.NewLazySystemDLL("user32.dll")
	defWindowProc := user32.NewProc("DefWindowProcW")

	if msg == wmHotkey {
		switch wParam {
		case hotkeyIDStart:
			handleHotkeyStart()
		case hotkeyIDStop:
			handleHotkeyStop()
		}
		return 0
	}
	r, _, _ := defWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func handleHotkeyStart() {
	hotkeyMu.Lock()
	m := hotkeyMonitor
	hotkeyMu.Unlock()
	if m == nil {
		return
	}
	s := m.GetSettings()
	if strings.TrimSpace(s.ChannelKey) == "" {
		m.emitLog("热键启动失败：请先填写并保存微信推送链接")
		return
	}
	if len(s.Templates) == 0 {
		m.emitLog("热键启动失败：请先添加识别模板")
		return
	}
	if s.GameWindowHwnd == 0 && strings.TrimSpace(s.GameWindowTitle) == "" {
		m.emitLog("热键启动失败：请先选择游戏窗口或填写窗口标题")
		return
	}
	status := m.Status()
	if status.Running {
		return
	}
	if err := m.Start(s.ChannelKey, s); err != nil {
		m.emitLog("热键启动失败: " + err.Error())
	}
}

func handleHotkeyStop() {
	hotkeyMu.Lock()
	m := hotkeyMonitor
	hotkeyMu.Unlock()
	if m == nil {
		return
	}
	if !m.Status().Running {
		return
	}
	m.Stop()
}
