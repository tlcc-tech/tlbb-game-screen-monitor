# 游戏掉线监控

Windows 桌面工具：绑定游戏窗口截图 + OpenCV 模板匹配，检测掉线画面后等待网络恢复，再通过[息知](https://xz.qqoq.net/)推送微信通知。

## 技术栈

- Go 1.24 + Wails v2.11
- 前端：Vite + Vanilla JS
- **窗口截图**：大漠 `BindWindow` + `Capture`（32 位 sidecar `runtime/dmcapture.exe`）
- **模板识别**：OpenCV + `gocv.io/x/gocv`（`MatchTemplate`）

## 功能

- **点选绑定游戏窗口**（无需手输标题）
- 多模板、可调相似度阈值、连续命中防误报
- **全局热键** 启动/停止监控（默认 Home / End，可自定义）
- 命中后等待 Ping/HTTP 探测网络恢复再推送
- 系统托盘 + 关闭时最小化

## 安装与目录

从 GitHub **Releases** 下载 **`游戏掉线监控-windows-amd64.exe`** 及同目录下的全部 `.dll` 文件（也可下载 zip 解压）。GitHub Actions 的 **windows-build** 产物已是解压后的目录结构，无需再套一层 zip。

```
游戏掉线监控-windows-amd64.exe    ← 主程序
libopencv_*.dll                    ← OpenCV（须在 exe 同目录，Windows 启动器要求）
libgcc_s_seh-1.dll                 ← MinGW 运行时（同上）
libstdc++-6.dll
libwinpthread-1.dll
runtime/
  opencv/       libopencv_*.dll（备份）
  mingw/        MinGW 运行时（备份）
  dm/           dm.dll, DmReg.dll
  dmcapture.exe 大漠 sidecar（386）
```

说明：

- Release 页面会同时提供 **散文件** 和 **zip**（zip 仅供程序内自动更新）。
- zip 内 **不会再套一层 zip**，也不会包含 `tlbb-game-screen-monitor.exe` 等多余文件。
- **OpenCV 与 MinGW 的 DLL 必须放在 exe 同目录**；Windows 在 Go 代码运行前就会加载这些依赖，`runtime/opencv/` 里的备份无法替代。

大漠插件构建时从 [xxxxue/xDM](https://github.com/xxxxue/xDM) 自动下载，**不进 Git 仓库**。

## 使用流程

1. 解压 zip，双击根目录 exe
2. 配置微信推送链接、添加识别模板
3. 点击 **「选择游戏窗口」**，按提示点击游戏窗口完成绑定
4. 点击 **「开始监控」**，或在游戏内按 **Home** 启动、**End** 停止
5. 游戏重启后需 **重新选择窗口**（HWND 会变化）

## 热键

- 默认：**Home** 启动、**End** 停止（全局，游戏内可用）
- 可在设置中修改并 **录制** 新键；**修改热键后需重启软件**
- 热键启动会使用 **已保存** 的设置（推送链接、模板、窗口绑定）

## 绑定模式

大漠 `BindWindow` 自动尝试：`gdi` → `dx2` → `dx.graphic`，日志中会显示成功模式。

## 开发

```bash
cd frontend && npm install && npm run build
cd .. && wails generate module
wails dev   # 需在 Windows 上验证
```

Windows 打包：

```powershell
.\scripts\build-win.ps1
```

## 备选匹配方案

| 方案 | 说明 | 参考 |
|------|------|------|
| **OpenCV（当前识别）** | gocv MatchTemplate | `match_all_windows.go` |
| **NCC / lookup** | 纯 Go，无 OpenCV DLL | `match_all_stub.go` |
| **大漠 FindPic** | sidecar 内 FindPic | tag `v1.1.3` |

## 配置目录

`%AppData%\tlbb-game-screen-monitor\`

- `settings.json` — 含 `gameWindowHwnd`、热键等
- `templates/*.png` — 识别模板

## 限制

- 游戏 **最小化** 后多数无法继续出图，监控不可靠
- 部分游戏可能检测大漠绑定，请自行评估风险
- 自动更新优先下载 **zip** 包并解压覆盖当前目录（含 exe、全部 DLL、`runtime/`）
