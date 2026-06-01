package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context

	allowQuit atomic.Bool
	monitor   *Monitor
}

func NewApp() *App {
	return &App{monitor: NewMonitor()}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.monitor.Attach(ctx)
	setupTray(a)
	setupHotkeys(a, a.monitor)
	a.startAutoUpdateCheck()
}

func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	if a.allowQuit.Load() {
		return false
	}
	if a.monitor != nil {
		status := a.monitor.Status()
		if status.Running {
			runtime.EventsEmit(ctx, "app:close-requested")
			return true
		}
	}
	return false
}

func (a *App) QuitApp() {
	a.allowQuit.Store(true)
	stopHotkeys()
	releaseDmCaptureBinding()
	trayQuit()
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}

func (a *App) CaptureScreenBase64() (string, error) {
	return captureScreenPNGBase64(a.monitor.GetSettings())
}

func (a *App) EnsureBuiltinTemplates() error {
	return a.monitor.ensureTemplatesReady()
}

func (a *App) ListTemplates() []TemplateItem {
	return a.monitor.ListTemplates()
}

func (a *App) SaveTemplate(name string, pngBase64 string, threshold float64) (TemplateItem, error) {
	return a.monitor.SaveTemplate(name, pngBase64, threshold)
}

func (a *App) DeleteTemplate(id string) error {
	return a.monitor.DeleteTemplate(id)
}

func (a *App) UpdateTemplate(item TemplateItem) error {
	return a.monitor.UpdateTemplate(item)
}

func (a *App) GetTemplateThumbnailBase64(id string) (string, error) {
	return a.monitor.GetTemplateThumbnailBase64(id)
}

func (a *App) ImportTemplate(name string, threshold float64) (TemplateItem, error) {
	if a.ctx == nil {
		return TemplateItem{}, errNotReady()
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择模板图片",
		Filters: []runtime.FileFilter{
			{DisplayName: "图片", Pattern: "*.png;*.jpg;*.jpeg"},
		},
	})
	if err != nil {
		return TemplateItem{}, err
	}
	if strings.TrimSpace(path) == "" {
		return TemplateItem{}, errCancelled()
	}
	if strings.TrimSpace(name) == "" {
		name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	return a.monitor.ImportTemplateFromPath(name, path, threshold)
}

func (a *App) TestMatch() ([]MatchScore, error) {
	return a.monitor.TestMatch()
}

func (a *App) StartMonitoring(channelKey string, settings AppSettings) error {
	return a.monitor.Start(channelKey, settings)
}

func (a *App) StopMonitoring() {
	a.monitor.Stop()
}

func (a *App) GetStatus() MonitorStatus {
	return a.monitor.Status()
}

func (a *App) GetSettings() AppSettings {
	_ = a.monitor.ensureTemplatesReady()
	return a.monitor.GetSettings()
}

func (a *App) SaveSettings(settings AppSettings) error {
	if err := a.monitor.SaveSettings(settings); err != nil {
		return err
	}
	refreshHotkeys(a.monitor)
	return nil
}

type WindowInfo struct {
	Hwnd      int64  `json:"hwnd"`
	Title     string `json:"title"`
	ClassName string `json:"className"`
}

func (a *App) PickGameWindow() (WindowInfo, error) {
	if a.ctx == nil {
		return WindowInfo{}, errNotReady()
	}
	runtime.WindowMinimise(a.ctx)
	a.monitor.emitLog("请在 8 秒内点击目标游戏窗口…")
	info, err := pickGameWindowInteractive()
	if err != nil {
		runtime.WindowShow(a.ctx)
		return WindowInfo{}, err
	}
	s := a.monitor.GetSettings()
	s.GameWindowHwnd = info.Hwnd
	s.GameWindowTitle = info.Title
	if err := a.monitor.SaveSettings(s); err != nil {
		runtime.WindowShow(a.ctx)
		return WindowInfo{}, err
	}
	runtime.WindowShow(a.ctx)
	a.monitor.emitLog(fmt.Sprintf("已绑定窗口: %s (0x%X) [%s]", info.Title, info.Hwnd, info.ClassName))
	return info, nil
}

func (a *App) ClearGameWindow() error {
	s := a.monitor.GetSettings()
	s.GameWindowHwnd = 0
	s.GameWindowTitle = ""
	releaseDmCaptureBinding()
	return a.monitor.SaveSettings(s)
}

func (a *App) GetAppInfo() AppInfo {
	return AppInfo{
		Name:    AppName,
		Author:  AppAuthor,
		Version: AppVersion,
		RepoURL: "https://github.com/" + UpdateRepoOwner + "/" + UpdateRepoName,
	}
}

func errNotReady() error {
	return errors.New("应用未就绪")
}

func errCancelled() error {
	return errors.New("已取消")
}
