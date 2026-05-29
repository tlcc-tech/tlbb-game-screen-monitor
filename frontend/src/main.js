import "./style.css";
import "./app.css";

import {
  BrowserOpenURL,
  EventsOn,
  WindowMinimise,
} from "../wailsjs/runtime/runtime";
import {
  CaptureScreenBase64,
  DeleteTemplate,
  GetAppInfo,
  GetSettings,
  GetStatus,
  GetTemplateThumbnailBase64,
  ImportTemplate,
  ListTemplates,
  QuitApp,
  SaveSettings,
  SaveTemplate,
  StartMonitoring,
  StopMonitoring,
  TestMatch,
  UpdateTemplate,
} from "../wailsjs/go/main/App";

document.querySelector("#app").innerHTML = `
    <div class="container">
        <h1 class="title">游戏掉线监控</h1>

        <section class="section">
            <h2 class="section-title">识别模板</h2>
            <div class="btn-row">
                <button class="btn" id="captureTplBtn" type="button">截取模板</button>
                <button class="btn" id="importTplBtn" type="button">从文件导入</button>
                <button class="btn" id="testMatchBtn" type="button">测试匹配</button>
            </div>
            <div class="template-list" id="templateList"></div>
        </section>

        <section class="section">
            <h2 class="section-title">监控设置</h2>
            <div class="form-grid">
                <label class="form-label">微信推送链接</label>
                <input class="input full" id="channelKey" type="text" placeholder="https://xizhi.qqoq.net/XZxxxx.send" />

                <label class="form-label">轮询间隔(秒)</label>
                <input class="input short" id="pollInterval" type="number" min="1" max="60" value="2" />

                <label class="form-label">连续命中次数</label>
                <input class="input short" id="consecutiveHits" type="number" min="1" max="20" value="3" />

                <label class="form-label">推送冷却(分钟)</label>
                <input class="input short" id="pushCooldown" type="number" min="1" max="120" value="10" />

                <label class="form-label">网络等待上限(分钟)</label>
                <input class="input short" id="networkWaitMax" type="number" min="1" max="120" value="30" />

                <label class="form-label">游戏窗口标题包含</label>
                <input class="input full" id="gameWindowTitle" type="text" placeholder="留空=全屏，例如：天龙八部" />

                <label class="form-label">Ping 主机</label>
                <input class="input full" id="pingHost" type="text" value="xz.qqoq.net" />
                <label class="form-check"><input type="checkbox" id="usePing" checked /> 启用 Ping 探测</label>

                <label class="form-label">HTTP 探测 URL</label>
                <input class="input full" id="httpProbeUrl" type="text" value="https://xz.qqoq.net" />
                <label class="form-check"><input type="checkbox" id="useHttp" checked /> 启用 HTTP 探测</label>

                <label class="form-check span2"><input type="checkbox" id="notifyOnRecover" /> 掉线画面消失后发「疑似已重连」通知</label>
            </div>

            <div class="btn-row">
                <button class="btn primary" id="startBtn" type="button">开始监控</button>
                <button class="btn" id="stopBtn" type="button">停止监控</button>
                <button class="btn" id="saveSettingsBtn" type="button">保存设置</button>
            </div>
        </section>

        <div class="result" id="status">状态：加载中...</div>
        <textarea class="log" id="log" readonly spellcheck="false"></textarea>

        <div class="footer">
            <div class="footer-left">
                <div>说明：全屏/游戏窗口截图 + 模板匹配；连续命中后等待网络恢复再微信推送。</div>
                <button class="footer-btn" id="getPushLinkBtn" type="button">如何获取微信推送链接？</button>
                <div>作者：<span id="author"></span>　版本：<span id="version"></span></div>
            </div>
        </div>
    </div>

    <div class="crop-overlay hidden" id="cropOverlay">
        <div class="crop-toolbar">
            <span>拖拽框选掉线画面区域，然后保存</span>
            <input class="input" id="cropName" type="text" placeholder="模板名称" />
            <input class="input short" id="cropThreshold" type="number" min="0.5" max="0.99" step="0.01" value="0.85" title="相似度阈值" />
            <button class="btn primary" id="cropSaveBtn" type="button">保存模板</button>
            <button class="btn" id="cropCancelBtn" type="button">取消</button>
        </div>
        <canvas id="cropCanvas"></canvas>
    </div>

    <div class="modal-overlay hidden" id="closeModal">
        <div class="modal">
            <p>监控运行中，关闭窗口将：</p>
            <div class="btn-row">
                <button class="btn" id="minimizeBtn" type="button">最小化到托盘</button>
                <button class="btn" id="quitBtn" type="button">退出软件</button>
                <button class="btn" id="cancelCloseBtn" type="button">取消</button>
            </div>
        </div>
    </div>
`;

const logEl = document.getElementById("log");
const statusEl = document.getElementById("status");
const templateListEl = document.getElementById("templateList");
const cropOverlay = document.getElementById("cropOverlay");
const cropCanvas = document.getElementById("cropCanvas");
const cropCtx = cropCanvas.getContext("2d");

let cropImage = null;
let cropDragging = false;
let cropStart = null;
let cropRect = null;
let templates = [];

function appendLog(line) {
  logEl.value += line + "\n";
  const lines = logEl.value.split("\n");
  if (lines.length > 2000) {
    logEl.value = lines.slice(-2000).join("\n");
  }
  logEl.scrollTop = logEl.scrollHeight;
}

function readSettingsFromUI() {
  return {
    channelKey: document.getElementById("channelKey").value.trim(),
    pollIntervalSec: parseInt(document.getElementById("pollInterval").value, 10) || 2,
    consecutiveHits: parseInt(document.getElementById("consecutiveHits").value, 10) || 3,
    pushCooldownMin: parseInt(document.getElementById("pushCooldown").value, 10) || 10,
    networkWaitMaxMin: parseInt(document.getElementById("networkWaitMax").value, 10) || 30,
    pingHost: document.getElementById("pingHost").value.trim(),
    httpProbeUrl: document.getElementById("httpProbeUrl").value.trim(),
    usePing: document.getElementById("usePing").checked,
    useHttp: document.getElementById("useHttp").checked,
    gameWindowTitle: document.getElementById("gameWindowTitle").value.trim(),
    notifyOnRecover: document.getElementById("notifyOnRecover").checked,
    templates: templates,
  };
}

function applySettingsToUI(s) {
  document.getElementById("channelKey").value = s.channelKey || "";
  document.getElementById("pollInterval").value = s.pollIntervalSec || 2;
  document.getElementById("consecutiveHits").value = s.consecutiveHits || 3;
  document.getElementById("pushCooldown").value = s.pushCooldownMin || 10;
  document.getElementById("networkWaitMax").value = s.networkWaitMaxMin || 30;
  document.getElementById("pingHost").value = s.pingHost || "xz.qqoq.net";
  document.getElementById("httpProbeUrl").value = s.httpProbeUrl || "https://xz.qqoq.net";
  document.getElementById("usePing").checked = s.usePing !== false;
  document.getElementById("useHttp").checked = s.useHttp !== false;
  document.getElementById("gameWindowTitle").value = s.gameWindowTitle || "";
  document.getElementById("notifyOnRecover").checked = !!s.notifyOnRecover;
  templates = s.templates || [];
}

function formatStatus(st) {
  const phaseMap = {
    idle: "空闲",
    watching: "监控中",
    matched_pending: "等待网络恢复推送",
    cooldown: "推送冷却",
  };
  let text = `状态：${st.running ? "运行中" : "已停止"} | 阶段：${phaseMap[st.phase] || st.phase}`;
  if (st.nextPollIn > 0) text += ` | 下次检测：${st.nextPollIn}s`;
  if (st.hitStreak > 0) text += ` | 连续命中：${st.hitStreak}`;
  if (st.pendingPush) text += " | 待推送";
  if (st.cooldownRemaining > 0) text += ` | 冷却：${st.cooldownRemaining}s`;
  if (st.lastMatchedName) text += `\n最近命中：${st.lastMatchedName} (${st.lastMatchedScore?.toFixed(3) || "-"})`;
  if (st.lastScores?.length) {
    const scores = st.lastScores
      .map((x) => `${x.templateName}:${x.score?.toFixed(3)}${x.matched ? "*" : ""}`)
      .join("  ");
    text += `\n分数：${scores}`;
  }
  if (st.lastError) text += `\n错误：${st.lastError}`;
  return text;
}

async function refreshTemplates() {
  templates = await ListTemplates();
  templateListEl.innerHTML = "";
  if (!templates.length) {
    templateListEl.innerHTML = '<p class="hint">暂无模板，请先截取或导入掉线/login 画面。</p>';
    return;
  }
  for (const t of templates) {
    let thumbSrc = "";
    try {
      const b64 = await GetTemplateThumbnailBase64(t.id);
      if (b64) thumbSrc = `data:image/png;base64,${b64}`;
    } catch (_) {}

    const card = document.createElement("div");
    card.className = "template-card";
    card.innerHTML = `
      <img class="template-thumb" src="${thumbSrc}" alt="" />
      <div class="template-meta">
        <div class="template-name">${escapeHtml(t.name)}</div>
        <label>阈值 <input type="range" min="0.5" max="0.99" step="0.01" value="${t.threshold}" data-id="${t.id}" class="threshold-range" /></label>
        <span class="threshold-val">${t.threshold.toFixed(2)}</span>
        <label class="form-check"><input type="checkbox" class="tpl-enabled" data-id="${t.id}" ${t.enabled ? "checked" : ""} /> 启用</label>
        <button class="btn small" data-del="${t.id}" type="button">删除</button>
      </div>
    `;
    templateListEl.appendChild(card);
  }

  templateListEl.querySelectorAll(".threshold-range").forEach((el) => {
    el.addEventListener("input", () => {
      el.parentElement.nextElementSibling.textContent = parseFloat(el.value).toFixed(2);
    });
    el.addEventListener("change", async () => {
      const id = el.dataset.id;
      const item = templates.find((x) => x.id === id);
      if (!item) return;
      item.threshold = parseFloat(el.value);
      await UpdateTemplate(item);
      appendLog(`已更新模板阈值：${item.name}`);
    });
  });

  templateListEl.querySelectorAll(".tpl-enabled").forEach((el) => {
    el.addEventListener("change", async () => {
      const id = el.dataset.id;
      const item = templates.find((x) => x.id === id);
      if (!item) return;
      item.enabled = el.checked;
      await UpdateTemplate(item);
    });
  });

  templateListEl.querySelectorAll("[data-del]").forEach((el) => {
    el.addEventListener("click", async () => {
      const id = el.dataset.del;
      el.disabled = true;
      try {
        await DeleteTemplate(id);
        await refreshTemplates();
        appendLog("已删除模板");
      } catch (e) {
        appendLog("删除失败: " + e);
      } finally {
        el.disabled = false;
      }
    });
  });
}

function escapeHtml(s) {
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function drawCropCanvas() {
  if (!cropImage) return;
  const maxW = window.innerWidth;
  const maxH = window.innerHeight - 56;
  const scale = Math.min(maxW / cropImage.width, maxH / cropImage.height, 1);
  cropCanvas.width = Math.floor(cropImage.width * scale);
  cropCanvas.height = Math.floor(cropImage.height * scale);
  cropCtx.drawImage(cropImage, 0, 0, cropCanvas.width, cropCanvas.height);
  if (cropRect) {
    cropCtx.strokeStyle = "#00e676";
    cropCtx.lineWidth = 2;
    cropCtx.strokeRect(cropRect.x, cropRect.y, cropRect.w, cropRect.h);
  }
}

function canvasPoint(evt) {
  const r = cropCanvas.getBoundingClientRect();
  return {
    x: evt.clientX - r.left,
    y: evt.clientY - r.top,
  };
}

cropCanvas.addEventListener("mousedown", (evt) => {
  cropDragging = true;
  cropStart = canvasPoint(evt);
  cropRect = { x: cropStart.x, y: cropStart.y, w: 0, h: 0 };
});

cropCanvas.addEventListener("mousemove", (evt) => {
  if (!cropDragging || !cropStart) return;
  const p = canvasPoint(evt);
  cropRect = {
    x: Math.min(cropStart.x, p.x),
    y: Math.min(cropStart.y, p.y),
    w: Math.abs(p.x - cropStart.x),
    h: Math.abs(p.y - cropStart.y),
  };
  drawCropCanvas();
});

cropCanvas.addEventListener("mouseup", () => {
  cropDragging = false;
});

document.getElementById("captureTplBtn").addEventListener("click", async () => {
  try {
    const b64 = await CaptureScreenBase64();
    if (!b64) {
      appendLog("截图失败：当前平台可能非 Windows");
      return;
    }
    cropImage = new Image();
    cropImage.onload = () => {
      cropRect = null;
      cropOverlay.classList.remove("hidden");
      drawCropCanvas();
    };
    cropImage.src = `data:image/png;base64,${b64}`;
  } catch (e) {
    appendLog("截图失败: " + e);
  }
});

document.getElementById("cropCancelBtn").addEventListener("click", () => {
  cropOverlay.classList.add("hidden");
  cropImage = null;
});

document.getElementById("cropSaveBtn").addEventListener("click", async () => {
  if (!cropImage || !cropRect || cropRect.w < 4 || cropRect.h < 4) {
    appendLog("请先框选有效区域");
    return;
  }
  const name = document.getElementById("cropName").value.trim() || "掉线模板";
  const threshold = parseFloat(document.getElementById("cropThreshold").value) || 0.85;

  const scaleX = cropImage.width / cropCanvas.width;
  const scaleY = cropImage.height / cropCanvas.height;
  const off = document.createElement("canvas");
  off.width = Math.floor(cropRect.w * scaleX);
  off.height = Math.floor(cropRect.h * scaleY);
  off.getContext("2d").drawImage(
    cropImage,
    cropRect.x * scaleX,
    cropRect.y * scaleY,
    off.width,
    off.height,
    0,
    0,
    off.width,
    off.height,
  );
  const dataUrl = off.toDataURL("image/png");
  const b64 = dataUrl.split(",")[1];
  try {
    await SaveTemplate(name, b64, threshold);
    cropOverlay.classList.add("hidden");
    await refreshTemplates();
    appendLog(`已保存模板：${name}`);
  } catch (e) {
    appendLog("保存失败: " + e);
  }
});

document.getElementById("importTplBtn").addEventListener("click", async () => {
  try {
    await ImportTemplate("", 0.85);
    await refreshTemplates();
    appendLog("已导入模板");
  } catch (e) {
    if (String(e).includes("取消")) return;
    appendLog("导入失败: " + e);
  }
});

document.getElementById("testMatchBtn").addEventListener("click", async () => {
  try {
    const scores = await TestMatch();
    if (!scores?.length) {
      appendLog("测试匹配：无启用模板或无结果");
      return;
    }
    for (const s of scores) {
      appendLog(
        `测试 ${s.templateName}: ${s.score?.toFixed(3)} ${s.matched ? "[命中]" : ""}`,
      );
    }
  } catch (e) {
    appendLog("测试失败: " + e);
  }
});

document.getElementById("saveSettingsBtn").addEventListener("click", async () => {
  try {
    await SaveSettings(readSettingsFromUI());
    appendLog("设置已保存");
  } catch (e) {
    appendLog("保存失败: " + e);
  }
});

document.getElementById("startBtn").addEventListener("click", async () => {
  const s = readSettingsFromUI();
  if (!s.templates?.some((t) => t.enabled)) {
    appendLog("请至少启用一个模板");
    return;
  }
  try {
    await StartMonitoring(s.channelKey, s);
    appendLog("监控已启动");
  } catch (e) {
    appendLog("启动失败: " + e);
  }
});

document.getElementById("stopBtn").addEventListener("click", () => {
  StopMonitoring();
  appendLog("已请求停止");
});

document.getElementById("getPushLinkBtn").addEventListener("click", () => {
  BrowserOpenURL("https://xz.qqoq.net/");
});

document.getElementById("minimizeBtn").addEventListener("click", () => {
  document.getElementById("closeModal").classList.add("hidden");
  WindowMinimise();
});

document.getElementById("quitBtn").addEventListener("click", () => {
  QuitApp();
});

document.getElementById("cancelCloseBtn").addEventListener("click", () => {
  document.getElementById("closeModal").classList.add("hidden");
});

EventsOn("log", (line) => appendLog(line));
EventsOn("match", (payload) => {
  if (payload?.scores?.length) {
    const brief = payload.scores
      .map((x) => `${x.templateName}:${x.score?.toFixed(3)}`)
      .join(" ");
    statusEl.textContent = `最近匹配 ${payload.at || ""}\n${brief}`;
  }
});
EventsOn("app:close-requested", () => {
  document.getElementById("closeModal").classList.remove("hidden");
});

async function init() {
  try {
    const info = await GetAppInfo();
    document.getElementById("author").textContent = info.author;
    document.getElementById("version").textContent = info.version;
  } catch (_) {}

  try {
    const s = await GetSettings();
    applySettingsToUI(s);
    await refreshTemplates();
  } catch (e) {
    appendLog("加载设置失败: " + e);
  }

  setInterval(async () => {
    try {
      const st = await GetStatus();
      statusEl.textContent = formatStatus(st);
    } catch (_) {}
  }, 2000);
}

init();
