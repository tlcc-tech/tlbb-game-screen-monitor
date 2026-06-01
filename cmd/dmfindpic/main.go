//go:build windows

// 32-bit sidecar (GOARCH=386) for dm.dmsoft FindPic via DmReg.dll.
package main

import (
	"bufio"
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
	Op      string  `json:"op"`
	DmPath  string  `json:"dmPath,omitempty"`
	RegPath string  `json:"regPath,omitempty"`
	TplPath string  `json:"tplPath,omitempty"`
	X1      int32   `json:"x1,omitempty"`
	Y1      int32   `json:"y1,omitempty"`
	X2      int32   `json:"x2,omitempty"`
	Y2      int32   `json:"y2,omitempty"`
	Pic     string  `json:"pic,omitempty"`
	Sim     float64 `json:"sim,omitempty"`
}

type respMsg struct {
	OK    bool    `json:"ok"`
	Error string  `json:"error,omitempty"`
	Ver   string  `json:"ver,omitempty"`
	Found bool    `json:"found,omitempty"`
	X     int32   `json:"x,omitempty"`
	Y     int32   `json:"y,omitempty"`
	Index int32   `json:"index,omitempty"`
	Sim   float64 `json:"sim,omitempty"`
}

var dmObj *ole.IDispatch
var coInitialized bool

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
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
			ver, err := initDM(req.RegPath, req.DmPath, req.TplPath)
			if err != nil {
				writeResp(enc, respMsg{OK: false, Error: err.Error()})
				continue
			}
			writeResp(enc, respMsg{OK: true, Ver: ver})
		case "find":
			res, err := findPic(req)
			if err != nil {
				writeResp(enc, respMsg{OK: false, Error: err.Error()})
				continue
			}
			writeResp(enc, res)
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

func initDM(regPath, dmPath, tplPath string) (string, error) {
	releaseDM()

	regPath, err := absPath(regPath)
	if err != nil {
		return "", err
	}
	dmPath, err = absPath(dmPath)
	if err != nil {
		return "", err
	}
	tplPath, err = absPath(tplPath)
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

	if _, err := oleutil.CallMethod(dmObj, "SetPath", tplPath); err != nil {
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

func findPic(req reqMsg) (respMsg, error) {
	if dmObj == nil {
		return respMsg{}, fmt.Errorf("dm not initialized")
	}
	if strings.TrimSpace(req.Pic) == "" {
		return respMsg{}, fmt.Errorf("pic is empty")
	}
	sim := req.Sim
	if sim <= 0 {
		sim = 0.85
	}
	if sim > 1 {
		sim = 1
	}

	xVar := ole.NewVariant(ole.VT_I4, int64(0))
	yVar := ole.NewVariant(ole.VT_I4, int64(0))
	defer xVar.Clear()
	defer yVar.Clear()

	retVal, err := oleutil.CallMethod(dmObj, "FindPic",
		req.X1, req.Y1, req.X2, req.Y2,
		req.Pic, "000000", sim, int32(0),
		xVar, yVar,
	)
	if err != nil {
		return respMsg{}, err
	}
	idx := int32(retVal.Val)
	if idx < 0 {
		return respMsg{OK: true, Found: false, Sim: 0}, nil
	}
	return respMsg{
		OK:    true,
		Found: true,
		X:     int32(xVar.Val),
		Y:     int32(yVar.Val),
		Index: idx,
		Sim:   sim,
	}, nil
}

func releaseDM() {
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
