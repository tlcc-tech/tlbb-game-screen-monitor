//go:build windows

// 32-bit sidecar (GOARCH=386) for dm.dmsoft BindWindow + Capture.
package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

type reqMsg struct {
	Op      string `json:"op"`
	DmPath  string `json:"dmPath,omitempty"`
	RegPath string `json:"regPath,omitempty"`
	Hwnd    int64  `json:"hwnd,omitempty"`
}

type respMsg struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	Ver   string `json:"ver,omitempty"`
	Mode  string `json:"mode,omitempty"`
	Bmp   string `json:"bmp,omitempty"`
	W     int32  `json:"w,omitempty"`
	H     int32  `json:"h,omitempty"`
}

var (
	dmObj         *ole.IDispatch
	coInitialized bool
	boundHwnd     int64
	boundMode     string
	captureDir    string
)

var bindModes = []string{"gdi", "dx2", "dx.graphic"}

func main() {
	var err error
	captureDir, err = os.MkdirTemp("", "dmcapture-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mkdir temp: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(captureDir)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	enc := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var req reqMsg
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			writeResp(enc, respMsg{OK: false, Error: err.Error()})
			continue
		}
		switch req.Op {
		case "quit":
			releaseDM()
			writeResp(enc, respMsg{OK: true})
			return
		case "init":
			ver, err := initDM(req.RegPath, req.DmPath)
			if err != nil {
				writeResp(enc, respMsg{OK: false, Error: err.Error()})
				continue
			}
			writeResp(enc, respMsg{OK: true, Ver: ver})
		case "bind":
			mode, err := bindWindow(req.Hwnd)
			if err != nil {
				writeResp(enc, respMsg{OK: false, Error: err.Error()})
				continue
			}
			writeResp(enc, respMsg{OK: true, Mode: mode})
		case "capture":
			bmp, w, h, err := captureBound()
			if err != nil {
				writeResp(enc, respMsg{OK: false, Error: err.Error()})
				continue
			}
			writeResp(enc, respMsg{OK: true, Bmp: bmp, W: w, H: h})
		case "unbind":
			unbindWindow()
			writeResp(enc, respMsg{OK: true})
		case "ping":
			if dmObj == nil {
				writeResp(enc, respMsg{OK: false, Error: "dm not initialized"})
				continue
			}
			writeResp(enc, respMsg{OK: true})
		default:
			writeResp(enc, respMsg{OK: false, Error: "unknown op: " + req.Op})
		}
	}
	releaseDM()
}

func writeResp(enc *json.Encoder, msg respMsg) {
	_ = enc.Encode(msg)
}

func initDM(regPath, dmPath string) (string, error) {
	releaseDM()

	regPath, err := absPath(regPath)
	if err != nil {
		return "", err
	}
	dmPath, err = absPath(dmPath)
	if err != nil {
		return "", err
	}

	if err := setDllPathA(regPath, dmPath); err != nil {
		return "", err
	}

	if err := ole.CoInitialize(0); err != nil {
		return "", fmt.Errorf("CoInitialize: %w", err)
	}
	coInitialized = true

	unknown, err := oleutil.CreateObject("dm.dmsoft")
	if err != nil {
		releaseDM()
		return "", fmt.Errorf("CreateObject dm.dmsoft: %w", err)
	}
	dispatch, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		unknown.Release()
		releaseDM()
		return "", err
	}
	unknown.Release()
	dmObj = dispatch

	if _, err := oleutil.CallMethod(dmObj, "SetPath", captureDir); err != nil {
		releaseDM()
		return "", fmt.Errorf("SetPath: %w", err)
	}

	verVal, err := oleutil.GetProperty(dmObj, "Ver")
	if err != nil {
		releaseDM()
		return "", fmt.Errorf("Ver: %w", err)
	}
	return verVal.ToString(), nil
}

func setDllPathA(regPath, dmPath string) error {
	handle, err := syscall.LoadLibrary(regPath)
	if err != nil {
		return fmt.Errorf("LoadLibrary DmReg: %w", err)
	}
	defer syscall.FreeLibrary(handle)

	proc, err := syscall.GetProcAddress(handle, "SetDllPathA")
	if err != nil {
		return fmt.Errorf("GetProcAddress SetDllPathA: %w", err)
	}

	dmPathPtr, err := syscall.BytePtrFromString(dmPath)
	if err != nil {
		return err
	}

	r1, _, callErr := syscall.SyscallN(proc, uintptr(unsafe.Pointer(dmPathPtr)), uintptr(1))
	if r1 == 0 {
		if callErr != 0 && callErr != syscall.Errno(0) {
			return fmt.Errorf("SetDllPathA failed: %v", callErr)
		}
		return fmt.Errorf("SetDllPathA returned 0")
	}
	return nil
}

func bindWindow(hwnd int64) (string, error) {
	if dmObj == nil {
		return "", fmt.Errorf("dm not initialized")
	}
	if hwnd == 0 {
		return "", fmt.Errorf("hwnd is zero")
	}
	unbindWindow()

	for _, mode := range bindModes {
		ret, err := oleutil.CallMethod(dmObj, "BindWindow", hwnd, mode, "windows", "windows", int32(0))
		if err != nil {
			continue
		}
		if int32(ret.Val) == 1 {
			boundHwnd = hwnd
			boundMode = mode
			return mode, nil
		}
		_, _ = oleutil.CallMethod(dmObj, "UnBindWindow")
	}
	return "", fmt.Errorf("BindWindow failed for all modes (gdi/dx2/dx.graphic)")
}

func unbindWindow() {
	if dmObj == nil {
		boundHwnd = 0
		boundMode = ""
		return
	}
	_, _ = oleutil.CallMethod(dmObj, "UnBindWindow")
	boundHwnd = 0
	boundMode = ""
}

func captureBound() (string, int32, int32, error) {
	if dmObj == nil {
		return "", 0, 0, fmt.Errorf("dm not initialized")
	}
	if boundHwnd == 0 {
		return "", 0, 0, fmt.Errorf("window not bound")
	}

	wVar := ole.NewVariant(ole.VT_I4, int64(0))
	hVar := ole.NewVariant(ole.VT_I4, int64(0))
	defer wVar.Clear()
	defer hVar.Clear()

	ret, err := oleutil.CallMethod(dmObj, "GetClientSize", boundHwnd, wVar, hVar)
	if err != nil {
		return "", 0, 0, err
	}
	if int32(ret.Val) != 1 {
		return "", 0, 0, fmt.Errorf("GetClientSize failed")
	}
	w := int32(wVar.Val)
	h := int32(hVar.Val)
	if w <= 0 || h <= 0 {
		return "", 0, 0, fmt.Errorf("invalid client size %dx%d", w, h)
	}

	fileName := "cap.bmp"
	capRet, err := oleutil.CallMethod(dmObj, "Capture", int32(0), int32(0), w-1, h-1, fileName)
	if err != nil {
		return "", 0, 0, err
	}
	if int32(capRet.Val) != 1 {
		return "", 0, 0, fmt.Errorf("Capture failed")
	}

	fullPath := filepath.Join(captureDir, fileName)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", 0, 0, err
	}
	_ = os.Remove(fullPath)
	return base64.StdEncoding.EncodeToString(data), w, h, nil
}

func releaseDM() {
	unbindWindow()
	if dmObj != nil {
		dmObj.Release()
		dmObj = nil
	}
	if coInitialized {
		ole.CoUninitialize()
		coInitialized = false
	}
}

func absPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", fmt.Errorf("path is empty")
	}
	if !filepath.IsAbs(p) {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		p = filepath.Join(wd, p)
	}
	return filepath.Clean(p), nil
}
