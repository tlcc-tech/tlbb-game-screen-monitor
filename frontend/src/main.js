import "./style.css";
import "./app.css";

import { EventsOn, WindowMinimise } from "../wailsjs/runtime/runtime";
import { GetAppInfo, QuitApp } from "../wailsjs/go/main/App";
import { createLog } from "./log.js";
import { icon } from "./icons.js";
import { loadSettings } from "./settings.js";
import { renderHome } from "./views/home.js";
import { createMonitorView } from "./views/monitor.js";
import { renderSettingsModal } from "./views/settings.js";
import { renderAboutModal } from "./views/about.js";

const APP_TITLE = "CC科技，永久免费";

const log = createLog();
let currentView = "home";
let monitorController = null;
let appInfo = { author: "", version: "", repoUrl: "" };

const appEl = document.querySelector("#app");

function closeModal(id) {
  appEl.querySelector(id)?.classList.add("hidden");
}

function renderShell() {
  const isHome = currentView === "home";
  const subTitle = currentView === "monitor" ? "挂机监控" : APP_TITLE;

  appEl.innerHTML = `
    <div class="app-shell">
      <header class="app-header">
        <div class="header-left">
          ${isHome
      ? `<h1 class="app-title">${APP_TITLE}</h1>`
      : `<button class="icon-btn back-btn" type="button" id="backBtn" title="返回首页">${icon("back")}</button>
                 <h1 class="app-title app-title--sub">${subTitle}</h1>`
    }
        </div>
        <div class="header-actions">
          <button class="icon-btn" type="button" id="settingsBtn" title="设置">${icon("settings")}</button>
          <button class="icon-btn" type="button" id="infoBtn" title="关于">${icon("info")}</button>
        </div>
      </header>
      <main class="app-main" id="viewRoot"></main>
    </div>

    <div class="modal-overlay hidden" id="aboutModal">
      <div id="aboutModalBody"></div>
    </div>

    <div class="modal-overlay hidden" id="settingsModal">
      <div id="settingsModalBody"></div>
    </div>

    <div class="modal-overlay hidden" id="closeModal">
      <div class="modal">
        <p class="modal-text">监控运行中，关闭窗口将：</p>
        <div class="btn-row modal-actions">
          <button class="btn" id="minimizeBtn" type="button">最小化到托盘</button>
          <button class="btn" id="quitBtn" type="button">退出软件</button>
          <button class="btn" id="cancelCloseBtn" type="button">取消</button>
        </div>
      </div>
    </div>
  `;

  const viewRoot = appEl.querySelector("#viewRoot");

  if (currentView === "home") {
    const logEl = renderHome(viewRoot, {
      onOpenFeature: handleOpenFeature,
      onClearLog: () => log.clear(),
    });
    log.mount(logEl);
  } else if (currentView === "monitor") {
    monitorController = createMonitorView(viewRoot, log);
  }

  appEl.querySelector("#settingsBtn").addEventListener("click", async () => {
    await loadSettings();
    const body = appEl.querySelector("#settingsModalBody");
    renderSettingsModal(body, log, () => closeModal("#settingsModal"));
    appEl.querySelector("#settingsModal").classList.remove("hidden");
  });

  appEl.querySelector("#infoBtn").addEventListener("click", () => {
    const body = appEl.querySelector("#aboutModalBody");
    renderAboutModal(body, appInfo, () => closeModal("#aboutModal"));
    appEl.querySelector("#aboutModal").classList.remove("hidden");
  });

  const backBtn = appEl.querySelector("#backBtn");
  if (backBtn) {
    backBtn.addEventListener("click", () => navigate("home"));
  }

  appEl.querySelector("#minimizeBtn").addEventListener("click", () => {
    closeModal("#closeModal");
    WindowMinimise();
  });
  appEl.querySelector("#quitBtn").addEventListener("click", () => QuitApp());
  appEl.querySelector("#cancelCloseBtn").addEventListener("click", () => {
    closeModal("#closeModal");
  });

  for (const modal of appEl.querySelectorAll(".modal-overlay")) {
    modal.addEventListener("click", (ev) => {
      if (ev.target === modal) modal.classList.add("hidden");
    });
  }
}

function navigate(view) {
  if (currentView === "home") {
    log.unmount();
  }
  if (monitorController) {
    monitorController.destroy();
    monitorController = null;
  }
  currentView = view;
  renderShell();
}

function handleOpenFeature(featureId, route) {
  if (route === "monitor") {
    log.append("进入挂机监控", { highlight: true });
    navigate("monitor");
    return;
  }
  const names = {
    keyboard: "按键打怪",
    cursor: "鼠标连点",
    broom: "副本扫荡",
    grid: "更多功能",
  };
  log.append(`${names[featureId] || featureId} 即将上线，敬请期待`);
}

EventsOn("log", (line) => log.append(line, { withTime: false }));
EventsOn("match", (payload) => {
  monitorController?.onMatch?.(payload);
});
EventsOn("app:close-requested", () => {
  appEl.querySelector("#closeModal")?.classList.remove("hidden");
});

async function init() {
  try {
    appInfo = await GetAppInfo();
  } catch (_) { }

  try {
    await loadSettings();
  } catch (_) { }

  log.append("系统启动完成", { highlight: true });
  renderShell();
}

init();
