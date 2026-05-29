//go:build windows

package main

import (
	"sync"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var trayOnce sync.Once

func setupTray(app *App) {
	trayOnce.Do(func() {
		go systray.Run(func() {
			systray.SetTitle(AppName)
			systray.SetTooltip(AppName)

			showItem := systray.AddMenuItem("显示", "显示主窗口")
			systray.AddSeparator()
			quitItem := systray.AddMenuItem("退出", "退出程序")

			go func() {
				for {
					select {
					case <-showItem.ClickedCh:
						if app != nil && app.ctx != nil {
							runtime.WindowShow(app.ctx)
							runtime.WindowUnminimise(app.ctx)
						}
					case <-quitItem.ClickedCh:
						if app != nil {
							app.allowQuit.Store(true)
						}
						if app != nil && app.ctx != nil {
							runtime.Quit(app.ctx)
						}
						systray.Quit()
						return
					}
				}
			}()
		}, func() {})
	})
}

func trayQuit() {
	systray.Quit()
}
