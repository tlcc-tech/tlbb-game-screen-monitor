# 游戏掉线监控

Windows 桌面工具：全屏/游戏窗口截图 + 模板匹配，检测掉线画面后等待网络恢复，再通过[息知](https://xz.qqoq.net/)推送微信通知。

## 技术栈

- Go 1.24 + Wails v2.11
- 前端：Vite + Vanilla JS
- 截图：`kbinani/screenshot`
- 匹配：`deluan/lookup`（NCC，类似按键精灵 FindPic）

## 功能

- 截取屏幕区域或导入图片作为识别模板
- 多模板、可调相似度阈值、连续命中防误报
- 可选仅监控指定标题的游戏窗口
- 命中后等待 Ping/HTTP 探测网络恢复再推送
- 可选「疑似已重连」通知
- 系统托盘 + 关闭时最小化

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

产物：`build/bin/游戏掉线监控-windows-amd64.exe`

## 配置目录

`%AppData%\tlbb-game-screen-monitor\`

- `settings.json` — 监控与推送设置
- `templates/*.png` — 识别模板图片
