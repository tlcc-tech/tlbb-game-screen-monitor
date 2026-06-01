# 游戏掉线监控

Windows 桌面工具：全屏/游戏窗口截图 + 模板匹配，检测掉线画面后等待网络恢复，再通过[息知](https://xz.qqoq.net/)推送微信通知。

## 技术栈

- Go 1.24 + Wails v2.11
- 前端：Vite + Vanilla JS
- 截图：`kbinani/screenshot`
- **匹配（当前方案）**：OpenCV + `gocv.io/x/gocv`（`MatchTemplate` + `TmCcoeffNormed`）

## 功能

- 截取屏幕区域或导入图片作为识别模板
- 多模板、可调相似度阈值、连续命中防误报
- 可选仅监控指定标题的游戏窗口
- 命中后等待 Ping/HTTP 探测网络恢复再推送
- 可选「疑似已重连」通知
- 系统托盘 + 关闭时最小化

## 模板匹配方案

当前 **v1.2.0** 仅使用 **OpenCV（gocv）**，无 NCC / 大漠兜底。Release 需附带 OpenCV 与 MinGW 运行时 DLL（CI 自动复制到 `build/bin/`）。

| 方案 | 说明 | 适用场景 | 历史版本 / 切换参考 |
|------|------|----------|---------------------|
| **OpenCV（当前）** | 内存中 PNG 模板 + `MatchTemplate`，速度与精度均衡 | **生产默认** | `v1.0.8`、`v1.2.0`；实现见 `match_all_windows.go` |
| **NCC / lookup** | `github.com/deluan/lookup`，纯 Go、无 OpenCV DLL | 轻量打包、非 Windows 开发 | `match_all_stub.go`（非 Windows）；可参考 `matcher_lookup.go` |
| **大漠 FindPic** | `dm.dll` COM + 32 位 sidecar，可跳过截图 | 按键精灵同款 API、需 dm 插件 | `v1.1.0`–`v1.1.3`；参考 tag [`v1.1.3`](https://github.com/tlcc-tech/tlbb-game-screen-monitor/releases/tag/v1.1.3) 中 `cmd/dmfindpic`、`dm_sidecar_windows.go` |

### 切换为 NCC（lookup）

1. 将 `match_all_windows.go` 改为调用 `LookupMatcher`（可参考 `match_all_stub.go`）。
2. 移除 `scripts/setup-opencv.ps1`、`copy-runtime-dlls.ps1` 及 CI 中 OpenCV 步骤。
3. Release 不再打包 `libopencv_*.dll` / MinGW DLL。

### 切换为大漠 FindPic

1. 恢复 `v1.1.3` 中的 sidecar 与 dm 相关文件（或 cherry-pick 对应 commit）。
2. 模板需同步生成 `.bmp`；dm 插件从 [xxxxue/xDM](https://github.com/xxxxue/xDM) 获取（`DmReg.dll` + `dm.dll`）。
3. 主程序 amd64 + `dmfindpic.exe`（386）同目录部署；详见 `v1.1.3` README / `scripts/fetch-dm.ps1`。

## 开发

```bash
cd frontend && npm install && npm run build
cd .. && wails generate module   # 更新 Go API 后重新生成并提交 frontend/wailsjs
wails dev   # 需在 Windows 上验证截图与 OpenCV 匹配
```

Windows 打包（需 MinGW + CMake，首次会编译 OpenCV，耗时较长）：

```powershell
.\scripts\build-win.ps1
```

产物：

```
build/bin/
  游戏掉线监控-windows-amd64.exe
  libopencv_*.dll
  libgcc_s_seh-1.dll / libstdc++-6.dll / libwinpthread-1.dll
```

## 配置目录

`%AppData%\tlbb-game-screen-monitor\`

- `settings.json` — 监控与推送设置
- `templates/*.png` — 识别模板图片

## CI 说明

- Tag 推送触发 GitHub Actions 构建与 Release。
- OpenCV 4.11 + contrib 缓存 key：`opencv-install-mingw-4110-gocv-v3`。
- 前端 `frontend/wailsjs` 已入库；CI 在 `wails build` 前执行 `npm ci && npm run build` 生成 `frontend/dist`。
