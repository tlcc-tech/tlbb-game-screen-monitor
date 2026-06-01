//go:build windows

package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type dmSidecarClient struct {
	mu   sync.Mutex
	cmd  *exec.Cmd
	in   io.WriteCloser
	out  *bufio.Scanner
	ver  string
	alive bool
}

type dmSidecarReq struct {
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

type dmSidecarResp struct {
	OK    bool    `json:"ok"`
	Error string  `json:"error,omitempty"`
	Ver   string  `json:"ver,omitempty"`
	Found bool    `json:"found,omitempty"`
	X     int32   `json:"x,omitempty"`
	Y     int32   `json:"y,omitempty"`
	Index int32   `json:"index,omitempty"`
	Sim   float64 `json:"sim,omitempty"`
}

var globalDMSidecar dmSidecarClient

func defaultDmPaths() (dmDll, regDll, sidecarExe string) {
	exe, err := os.Executable()
	if err != nil {
		return "", "", ""
	}
	dir := filepath.Dir(exe)
	return filepath.Join(dir, "dm", "dm.dll"),
		filepath.Join(dir, "dm", "DmReg.dll"),
		filepath.Join(dir, "dmfindpic.exe")
}

func resolveDmPaths(settings AppSettings) (dmDll, regDll string) {
	dmDll = strings.TrimSpace(settings.DmDllPath)
	regDll = strings.TrimSpace(settings.DmRegDllPath)
	if dmDll == "" || regDll == "" {
		defDm, defReg, _ := defaultDmPaths()
		if dmDll == "" {
			dmDll = defDm
		}
		if regDll == "" {
			regDll = defReg
		}
	}
	return dmDll, regDll
}

func dmSidecarExePath() string {
	_, _, sidecar := defaultDmPaths()
	return sidecar
}

func dmFilesExist(settings AppSettings) bool {
	dmDll, regDll := resolveDmPaths(settings)
	if dmDll == "" || regDll == "" {
		return false
	}
	if _, err := os.Stat(dmDll); err != nil {
		return false
	}
	if _, err := os.Stat(regDll); err != nil {
		return false
	}
	if _, err := os.Stat(dmSidecarExePath()); err != nil {
		return false
	}
	return true
}

func (c *dmSidecarClient) start(settings AppSettings) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.alive {
		return nil
	}

	dmDll, regDll := resolveDmPaths(settings)
	sidecar := dmSidecarExePath()
	if _, err := os.Stat(sidecar); err != nil {
		return fmt.Errorf("dmfindpic.exe 不存在: %s", sidecar)
	}
	if _, err := os.Stat(dmDll); err != nil {
		return fmt.Errorf("dm.dll 不存在: %s", dmDll)
	}
	if _, err := os.Stat(regDll); err != nil {
		return fmt.Errorf("DmReg.dll 不存在: %s", regDll)
	}

	tplDir, err := templatesDir()
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
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	c.cmd = cmd
	c.in = in
	c.out = scanner
	c.alive = true

	resp, err := c.callLocked(dmSidecarReq{
		Op:      "init",
		DmPath:  dmDll,
		RegPath: regDll,
		TplPath: tplDir,
	})
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

func (c *dmSidecarClient) stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopLocked()
}

func (c *dmSidecarClient) stopLocked() {
	if !c.alive {
		return
	}
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
}

func (c *dmSidecarClient) find(x1, y1, x2, y2 int32, pic string, sim float64) (dmSidecarResp, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.alive {
		return dmSidecarResp{}, errors.New("dm sidecar not running")
	}
	return c.callLocked(dmSidecarReq{
		Op:  "find",
		X1:  x1,
		Y1:  y1,
		X2:  x2,
		Y2:  y2,
		Pic: pic,
		Sim: sim,
	})
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
		return resp, errors.New(resp.Error)
	}
	return resp, nil
}

func ensureDMSidecar(settings AppSettings) error {
	if !settings.UseDmMatcher {
		return errors.New("dm matcher disabled")
	}
	return globalDMSidecar.start(settings)
}

func stopDMSidecar() {
	globalDMSidecar.stop()
}

func dmFindPic(x1, y1, x2, y2 int32, pic string, sim float64) (dmSidecarResp, error) {
	return globalDMSidecar.find(x1, y1, x2, y2, pic, sim)
}
