# 游戏掉线监控

Windows 桌面工具：全屏/游戏窗口截图 + 模板匹配，检测掉线画面后等待网络恢复，再通过[息知](https://xz.qqoq.net/)推送微信通知。

## 技术栈

- Go 1.24 + Wails v2.11
- 前端：Vite + Vanilla JS
- 截图：`kbinani/screenshot`
- 匹配（Windows 优先）：[大漠 dm.dll FindPic](https://github.com/xxxxue/xDM)（32 位 sidecar + DmReg 免注册）
- 匹配（回退）：`deluan/lookup`（NCC）

## 功能

- 截取屏幕区域或导入图片作为识别模板
- 多模板、可调相似度阈值、连续命中防误报
- 可选仅监控指定标题的游戏窗口
- 命中后等待 Ping/HTTP 探测网络恢复再推送
- 可选「疑似已重连」通知
- 系统托盘 + 关闭时最小化

## 大漠插件（FindPic）

构建时会从 [xxxxue/xDM](https://github.com/xxxxue/xDM) 自动下载 `dm.dll` 与 `DmReg.dll`（`scripts/fetch-dm.ps1`），并打入 Release：

```
build/bin/
  游戏掉线监控-windows-amd64.exe
  dmfindpic.exe          # 32 位 sidecar
  dm/
    dm.dll
    DmReg.dll
```

- 主程序为 64 位，通过 `dmfindpic.exe` 调用大漠 COM（与 xDM C# 示例相同的 `SetDllPathA` 免注册方式）。
- 模板除 `.png`（UI 缩略图）外会同步生成 `.bmp` 供 FindPic 使用。
- 若 dm 初始化失败，自动回退到 lookup 匹配。

## 开发

```bash
cd frontend && npm install && npm run build
cd .. && wails generate module
wails dev   # 需在 Windows 上验证截图
```

Windows 打包：

```powershell
.\scripts\build-win.ps1
```

产物见 `build/bin/`。

## 配置目录

`%AppData%\tlbb-game-screen-monitor\`

- `settings.json` — 监控与推送设置
- `templates/*.png` — 识别模板（UI）
- `templates/*.bmp` — 大漠 FindPic 用图
