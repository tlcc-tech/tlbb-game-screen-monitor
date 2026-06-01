//go:build windows

package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

type dmSidecarClient struct {
	mu    sync.Mutex
	cmd   *exec.Cmd
	in    io.WriteCloser
	out   *bufio.Scanner
	alive bool
	ver   string
	mode  string
}

type dmSidecarReq struct {
	Op      string `json:"op"`
	DmPath  string `json:"dmPath,omitempty"`
	RegPath string `json:"regPath,omitempty"`
	Hwnd    int64  `json:"hwnd,omitempty"`
}

type dmSidecarResp struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	Ver   string `json:"ver,omitempty"`
	Mode  string `json:"mode,omitempty"`
	Bmp   string `json:"bmp,omitempty"`
	W     int32  `json:"w,omitempty"`
	H     int32  `json:"h,omitempty"`
}

var globalDMSidecar dmSidecarClient

func runtimeDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "runtime"), nil
}

func dmCaptureExePath() (string, error) {
	dir, err := runtimeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "dmcapture.exe"), nil
}

func dmDllPaths() (dmDll, regDll string, err error) {
	dir, err := runtimeDir()
	if err != nil {
		return "", "", err
	}
	return filepath.Join(dir, "dm", "dm.dll"), filepath.Join(dir, "dm", "DmReg.dll"), nil
}

func dmCaptureAvailable() bool {
	exe, err := dmCaptureExePath()
	if err != nil {
		return false
	}
	if _, err := os.Stat(exe); err != nil {
		return false
	}
	dmDll, regDll, err := dmDllPaths()
	if err != nil {
		return false
	}
	if _, err := os.Stat(dmDll); err != nil {
		return false
	}
	if _, err := os.Stat(regDll); err != nil {
		return false
	}
	return true
}

func (c *dmSidecarClient) ensureStarted() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.alive {
		return nil
	}

	sidecar, err := dmCaptureExePath()
	if err != nil {
		return err
	}
	dmDll, regDll, err := dmDllPaths()
	if err != nil {
		return err
	}

	cmd := exec.Command(sidecar)
	cmd.Stderr = os.Stderr
	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		in.Close()
		return err
	}
	if err := cmd.Start(); err != nil {
		in.Close()
		return err
	}

	scanner := bufio.NewScanner(outPipe)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)

	c.cmd = cmd
	c.in = in
	c.out = scanner
	c.alive = true

	resp, err := c.callLocked(dmSidecarReq{Op: "init", DmPath: dmDll, RegPath: regDll})
	if err != nil {
		c.stopLocked()
		return err
	}
	if !resp.OK {
		c.stopLocked()
		return errors.New(resp.Error)
	}
	c.ver = resp.Ver
	return nil
}

func (c *dmSidecarClient) bind(hwnd int64) (string, error) {
	if err := c.ensureStarted(); err != nil {
		return "", err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	resp, err := c.callLocked(dmSidecarReq{Op: "bind", Hwnd: hwnd})
	if err != nil {
		return "", err
	}
	if !resp.OK {
		return "", errors.New(resp.Error)
	}
	c.mode = resp.Mode
	return resp.Mode, nil
}

func (c *dmSidecarClient) captureBMP() ([]byte, error) {
	if err := c.ensureStarted(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	resp, err := c.callLocked(dmSidecarReq{Op: "capture"})
	if err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, errors.New(resp.Error)
	}
	data, err := base64.StdEncoding.DecodeString(resp.Bmp)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (c *dmSidecarClient) stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopLocked()
}

func (c *dmSidecarClient) stopLocked() {
	if !c.alive {
		return
	}
	_, _ = c.callLocked(dmSidecarReq{Op: "unbind"})
	_, _ = c.callLocked(dmSidecarReq{Op: "quit"})
	if c.in != nil {
		_ = c.in.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
		_, _ = c.cmd.Process.Wait()
	}
	c.cmd = nil
	c.in = nil
	c.out = nil
	c.alive = false
	c.ver = ""
	c.mode = ""
}

func (c *dmSidecarClient) callLocked(req dmSidecarReq) (dmSidecarResp, error) {
	var resp dmSidecarResp
	if c.in == nil || c.out == nil {
		return resp, errors.New("sidecar pipes unavailable")
	}
	data, err := json.Marshal(req)
	if err != nil {
		return resp, err
	}
	if _, err := c.in.Write(append(data, '\n')); err != nil {
		return resp, err
	}
	if !c.out.Scan() {
		if err := c.out.Err(); err != nil {
			return resp, err
		}
		return resp, errors.New("sidecar closed stdout")
	}
	if err := json.Unmarshal(c.out.Bytes(), &resp); err != nil {
		return resp, err
	}
	if !resp.OK {
		return resp, fmt.Errorf("%s", resp.Error)
	}
	return resp, nil
}

func dmBindWindow(hwnd int64) (string, error) {
	return globalDMSidecar.bind(hwnd)
}

func dmCaptureBMP() ([]byte, error) {
	return globalDMSidecar.captureBMP()
}

func stopDMCaptureSidecar() {
	globalDMSidecar.stop()
}
