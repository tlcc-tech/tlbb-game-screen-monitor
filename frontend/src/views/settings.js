import { ClearGameWindow, PickGameWindow } from "../../wailsjs/go/main/App";
import { BrowserOpenURL } from "../../wailsjs/runtime/runtime";
import {
  applyGlobalToForm,
  formatWindowBindLabel,
  getCachedSettings,
  loadSettings,
  mergeGlobalSettings,
  readGlobalFromForm,
  saveSettings,
} from "../settings.js";

export function renderSettingsModal(container, log, onClose) {
  let windowBindState = { hwnd: 0, title: "" };

  container.innerHTML = `
    <div class="modal modal--wide">
      <h2 class="modal-title">设置</h2>
      <div class="modal-scroll">
        <section class="settings-section">
          <h3 class="settings-section-title">游戏窗口</h3>
          <div class="btn-row">
            <button class="btn" id="pickWindowBtn" type="button">选择游戏窗口</button>
            <button class="btn" id="clearWindowBtn" type="button">清除绑定</button>
          </div>
          <p class="hint" id="windowBindLabel">未绑定（将使用全屏截图）</p>
        </section>

        <section class="settings-section">
          <h3 class="settings-section-title">微信推送</h3>
          <label class="field-label" for="channelKey">推送链接</label>
          <input class="input full" id="channelKey" type="text" placeholder="https://xizhi.qqoq.net/XZxxxx.send" />
          <button class="link-btn" id="getPushLinkBtn" type="button">如何获取微信推送链接？</button>
        </section>

        <section class="settings-section">
          <h3 class="settings-section-title">网络检测</h3>
          <div class="form-grid settings-grid">
            <label class="form-label" for="pingHost">Ping 主机</label>
            <input class="input full" id="pingHost" type="text" value="xz.qqoq.net" />
            <span></span>
            <label class="form-check"><input type="checkbox" id="usePing" checked /> 启用 Ping 探测</label>

            <label class="form-label" for="httpProbeUrl">HTTP 探测 URL</label>
            <input class="input full" id="httpProbeUrl" type="text" value="https://xz.qqoq.net" />
            <span></span>
            <label class="form-check"><input type="checkbox" id="useHttp" checked /> 启用 HTTP 探测</label>

            <label class="form-label" for="networkWaitMax">网络等待上限(分钟)</label>
            <input class="input short" id="networkWaitMax" type="number" min="1" max="120" value="30" />
          </div>
        </section>
      </div>
      <div class="btn-row modal-actions">
        <button class="btn primary" id="settingsSaveBtn" type="button">保存</button>
        <button class="btn" id="settingsCancelBtn" type="button">取消</button>
      </div>
    </div>
  `;

  function updateWindowBindLabel() {
    const el = container.querySelector("#windowBindLabel");
    if (el) el.textContent = formatWindowBindLabel(windowBindState);
  }

  async function refreshForm() {
    const s = await loadSettings();
    applyGlobalToForm(container, s, windowBindState);
    updateWindowBindLabel();
  }

  container.querySelector("#pickWindowBtn").addEventListener("click", async () => {
    try {
      log.append("即将最小化本窗口，请点击目标游戏窗口…");
      const info = await PickGameWindow();
      windowBindState = { hwnd: info.hwnd, title: info.title };
      updateWindowBindLabel();
      log.append(`已绑定：${info.title}`, { highlight: true });
      const base = getCachedSettings() || (await loadSettings());
      await saveSettings(
        mergeGlobalSettings(base, readGlobalFromForm(container, windowBindState)),
      );
    } catch (e) {
      log.append("选窗失败: " + e);
    }
  });

  container.querySelector("#clearWindowBtn").addEventListener("click", async () => {
    try {
      await ClearGameWindow();
      windowBindState = { hwnd: 0, title: "" };
      updateWindowBindLabel();
      log.append("已清除窗口绑定");
      const base = getCachedSettings() || (await loadSettings());
      await saveSettings(
        mergeGlobalSettings(base, readGlobalFromForm(container, windowBindState)),
      );
    } catch (e) {
      log.append("清除失败: " + e);
    }
  });

  container.querySelector("#getPushLinkBtn").addEventListener("click", () => {
    BrowserOpenURL("https://xz.qqoq.net/");
  });

  container.querySelector("#settingsSaveBtn").addEventListener("click", async () => {
    try {
      const base = getCachedSettings() || (await loadSettings());
      await saveSettings(
        mergeGlobalSettings(base, readGlobalFromForm(container, windowBindState)),
      );
      log.append("全局设置已保存");
      onClose();
    } catch (e) {
      log.append("保存失败: " + e);
    }
  });

  container.querySelector("#settingsCancelBtn").addEventListener("click", onClose);

  refreshForm();
}
