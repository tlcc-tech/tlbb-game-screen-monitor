import {
  CaptureScreenBase64,
  DeleteTemplate,
  EnsureBuiltinTemplates,
  GetSettings,
  GetStatus,
  GetTemplateThumbnailBase64,
  ListTemplates,
  SaveSettings,
  SaveTemplate,
  StartMonitoring,
  StopMonitoring,
  TestMatch,
  UpdateTemplate,
} from "../../wailsjs/go/main/App";
import {
  applyMonitorToForm,
  loadSettings,
  mergeMonitorSettings,
  readMonitorFromForm,
  saveSettings,
} from "../settings.js";
import { findBuiltin, groupTemplates, PRESET_ORDER } from "../monitor-presets.js";

export function createMonitorView(root, log) {
  let templates = [];
  let cropImage = null;
  let cropDragging = false;
  let cropStart = null;
  let cropRect = null;
  let statusTimer = null;

  root.innerHTML = `
    <div class="monitor-view">
      <div class="monitor-scroll">
        <section class="panel">
          <h3 class="panel-title">监控项目</h3>
          <div class="preset-grid" id="presetGrid"></div>
        </section>

        <section class="panel">
          <h3 class="panel-title">检测参数</h3>
          <div class="form-grid">
            <label class="form-label">轮询间隔(秒)</label>
            <input class="input short" id="pollInterval" type="number" min="1" max="60" value="2" />

            <label class="form-label">推送冷却(分钟)</label>
            <input class="input short" id="pushCooldown" type="number" min="1" max="120" value="10" />

            <label class="form-check span2"><input type="checkbox" id="notifyOnRecover" /> 掉线画面消失后发「疑似已重连」通知（仅对「掉线」有效）</label>
          </div>
        </section>

        <section class="panel custom-section">
          <h3 class="panel-title">自定义监控</h3>
          <div class="btn-row">
            <button class="btn primary" id="captureTplBtn" type="button">添加自定义截图监控</button>
            <button class="btn" id="testMatchBtn" type="button">测试匹配</button>
          </div>
          <div class="template-list" id="templateList"></div>
        </section>

        <div class="btn-row">
          <button class="btn primary" id="startBtn" type="button">开始监控</button>
          <button class="btn" id="stopBtn" type="button">停止监控</button>
          <button class="btn" id="saveSettingsBtn" type="button">保存设置</button>
        </div>

        <div class="status-box" id="status">状态：加载中...</div>

        <div class="monitor-footer">
          <span>内置项命中 1 次即推送（掉线等待网络恢复）；窗口与推送请在「设置」中配置。</span>
        </div>
      </div>
    </div>

    <div class="crop-overlay hidden" id="cropOverlay">
      <div class="crop-toolbar">
        <span>拖拽框选监控区域，然后保存</span>
        <input class="input" id="cropName" type="text" placeholder="监控名称" />
        <input class="input short" id="cropThreshold" type="number" min="0.5" max="0.99" step="0.01" value="0.85" title="相似度阈值" />
        <button class="btn primary" id="cropSaveBtn" type="button">保存</button>
        <button class="btn" id="cropCancelBtn" type="button">取消</button>
      </div>
      <canvas id="cropCanvas"></canvas>
    </div>
  `;

  const presetGridEl = root.querySelector("#presetGrid");
  const templateListEl = root.querySelector("#templateList");
  const statusEl = root.querySelector("#status");
  const cropOverlay = root.querySelector("#cropOverlay");
  const cropCanvas = root.querySelector("#cropCanvas");
  const cropCtx = cropCanvas.getContext("2d");

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

  async function renderPresetGrid() {
    presetGridEl.innerHTML = "";
    for (const meta of PRESET_ORDER) {
      const tpl = findBuiltin(templates, meta.key);
      if (!tpl) continue;

      let thumbSrc = "";
      try {
        const b64 = await GetTemplateThumbnailBase64(tpl.id);
        if (b64) thumbSrc = `data:image/png;base64,${b64}`;
      } catch (_) {}

      const card = document.createElement("label");
      card.className = "preset-item";
      card.innerHTML = `
        <input type="checkbox" class="preset-check" data-id="${tpl.id}" ${tpl.enabled ? "checked" : ""} />
        <img class="preset-thumb" src="${thumbSrc}" alt="" />
        <span class="preset-name">${escapeHtml(meta.label)}</span>
      `;
      presetGridEl.appendChild(card);
    }

    presetGridEl.querySelectorAll(".preset-check").forEach((el) => {
      el.addEventListener("change", async () => {
        const item = templates.find((x) => x.id === el.dataset.id);
        if (!item) return;
        item.enabled = el.checked;
        try {
          await UpdateTemplate(item);
          log.append(`${item.enabled ? "已启用" : "已禁用"}：${item.name}`);
        } catch (e) {
          log.append("更新失败: " + e);
          el.checked = !el.checked;
        }
      });
    });
  }

  async function refreshCustomTemplates() {
    const { customs } = groupTemplates(templates);
    templateListEl.innerHTML = "";
    if (!customs.length) {
      templateListEl.innerHTML = '<p class="hint">暂无自定义监控，点击上方按钮添加。</p>';
      return;
    }

    for (const t of customs) {
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
        const item = templates.find((x) => x.id === el.dataset.id);
        if (!item) return;
        item.threshold = parseFloat(el.value);
        await UpdateTemplate(item);
        log.append(`已更新阈值：${item.name}`);
      });
    });

    templateListEl.querySelectorAll(".tpl-enabled").forEach((el) => {
      el.addEventListener("change", async () => {
        const item = templates.find((x) => x.id === el.dataset.id);
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
          await reloadTemplates();
          log.append("已删除自定义监控");
        } catch (e) {
          log.append("删除失败: " + e);
        } finally {
          el.disabled = false;
        }
      });
    });
  }

  async function reloadTemplates() {
    templates = await ListTemplates();
    await renderPresetGrid();
    await refreshCustomTemplates();
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
      cropCtx.strokeStyle = "#D97706";
      cropCtx.lineWidth = 2;
      cropCtx.strokeRect(cropRect.x, cropRect.y, cropRect.w, cropRect.h);
    }
  }

  function canvasPoint(evt) {
    const r = cropCanvas.getBoundingClientRect();
    return { x: evt.clientX - r.left, y: evt.clientY - r.top };
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

  root.querySelector("#captureTplBtn").addEventListener("click", async () => {
    try {
      const b64 = await CaptureScreenBase64();
      if (!b64) {
        log.append("截图失败：当前平台可能非 Windows");
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
      log.append("截图失败: " + e);
    }
  });

  root.querySelector("#cropCancelBtn").addEventListener("click", () => {
    cropOverlay.classList.add("hidden");
    cropImage = null;
  });

  root.querySelector("#cropSaveBtn").addEventListener("click", async () => {
    if (!cropImage || !cropRect || cropRect.w < 4 || cropRect.h < 4) {
      log.append("请先框选有效区域");
      return;
    }
    const name = root.querySelector("#cropName").value.trim() || "自定义监控";
    const threshold = parseFloat(root.querySelector("#cropThreshold").value) || 0.85;
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
    const b64 = off.toDataURL("image/png").split(",")[1];
    try {
      await SaveTemplate(name, b64, threshold);
      cropOverlay.classList.add("hidden");
      await reloadTemplates();
      log.append(`已添加自定义监控：${name}`);
    } catch (e) {
      log.append("保存失败: " + e);
    }
  });

  root.querySelector("#testMatchBtn").addEventListener("click", async () => {
    try {
      const scores = await TestMatch();
      if (!scores?.length) {
        log.append("测试匹配：无启用模板或无结果");
        return;
      }
      for (const s of scores) {
        log.append(`测试 ${s.templateName}: ${s.score?.toFixed(3)} ${s.matched ? "[命中]" : ""}`);
      }
    } catch (e) {
      log.append("测试失败: " + e);
    }
  });

  root.querySelector("#saveSettingsBtn").addEventListener("click", async () => {
    try {
      const base = await loadSettings();
      await saveSettings(mergeMonitorSettings(base, readMonitorFromForm(root, templates)));
      log.append("检测参数已保存");
    } catch (e) {
      log.append("保存失败: " + e);
    }
  });

  root.querySelector("#startBtn").addEventListener("click", async () => {
    const local = readMonitorFromForm(root, templates);
    if (!local.templates?.some((t) => t.enabled)) {
      log.append("请至少启用一个监控项");
      return;
    }
    try {
      const base = await loadSettings();
      const merged = mergeMonitorSettings(base, local);
      merged.consecutiveHits = 1;
      await StartMonitoring(merged.channelKey, merged);
      log.append("监控已启动", { highlight: true });
    } catch (e) {
      log.append("启动失败: " + e);
    }
  });

  root.querySelector("#stopBtn").addEventListener("click", () => {
    StopMonitoring();
    log.append("已请求停止");
  });

  async function init() {
    try {
      await EnsureBuiltinTemplates();
      const s = await GetSettings();
      applyMonitorToForm(root, s);
      templates = s.templates || [];
      await reloadTemplates();
    } catch (e) {
      log.append("加载设置失败: " + e);
    }

    statusTimer = setInterval(async () => {
      try {
        const st = await GetStatus();
        statusEl.textContent = formatStatus(st);
      } catch (_) {}
    }, 2000);
  }

  init();

  return {
    destroy() {
      if (statusTimer) clearInterval(statusTimer);
    },
    onMatch(payload) {
      if (payload?.scores?.length) {
        const brief = payload.scores
          .map((x) => `${x.templateName}:${x.score?.toFixed(3)}`)
          .join(" ");
        statusEl.textContent = `最近匹配 ${payload.at || ""}\n${brief}`;
      }
    },
  };
}

function escapeHtml(s) {
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}
