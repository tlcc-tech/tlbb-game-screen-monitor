import { GetSettings, SaveSettings } from "../wailsjs/go/main/App";

let cachedSettings = null;

export async function loadSettings() {
  cachedSettings = await GetSettings();
  return cachedSettings;
}

export function getCachedSettings() {
  return cachedSettings;
}

export async function saveSettings(settings) {
  await SaveSettings(settings);
  cachedSettings = settings;
}

export function readGlobalFromForm(root, windowBindState) {
  return {
    channelKey: root.querySelector("#channelKey").value.trim(),
    pingHost: root.querySelector("#pingHost").value.trim(),
    httpProbeUrl: root.querySelector("#httpProbeUrl").value.trim(),
    usePing: root.querySelector("#usePing").checked,
    useHttp: root.querySelector("#useHttp").checked,
    networkWaitMaxMin: parseInt(root.querySelector("#networkWaitMax").value, 10) || 30,
    gameWindowTitle: windowBindState?.title || "",
    gameWindowHwnd: windowBindState?.hwnd || 0,
  };
}

export function applyGlobalToForm(root, settings, windowBindState) {
  root.querySelector("#channelKey").value = settings.channelKey || "";
  root.querySelector("#pingHost").value = settings.pingHost || "xz.qqoq.net";
  root.querySelector("#httpProbeUrl").value = settings.httpProbeUrl || "https://xz.qqoq.net";
  root.querySelector("#usePing").checked = settings.usePing !== false;
  root.querySelector("#useHttp").checked = settings.useHttp !== false;
  root.querySelector("#networkWaitMax").value = settings.networkWaitMaxMin || 30;
  if (windowBindState) {
    windowBindState.hwnd = settings.gameWindowHwnd || 0;
    windowBindState.title = settings.gameWindowTitle || "";
  }
}

export function readMonitorFromForm(root, templates) {
  return {
    pollIntervalSec: parseInt(root.querySelector("#pollInterval").value, 10) || 2,
    consecutiveHits: parseInt(root.querySelector("#consecutiveHits").value, 10) || 3,
    pushCooldownMin: parseInt(root.querySelector("#pushCooldown").value, 10) || 10,
    hotkeyStart: root.querySelector("#hotkeyStart").value.trim() || "Home",
    hotkeyStop: root.querySelector("#hotkeyStop").value.trim() || "End",
    notifyOnRecover: root.querySelector("#notifyOnRecover").checked,
    templates,
  };
}

export function applyMonitorToForm(root, settings) {
  root.querySelector("#pollInterval").value = settings.pollIntervalSec || 2;
  root.querySelector("#consecutiveHits").value = settings.consecutiveHits || 3;
  root.querySelector("#pushCooldown").value = settings.pushCooldownMin || 10;
  root.querySelector("#hotkeyStart").value = settings.hotkeyStart || "Home";
  root.querySelector("#hotkeyStop").value = settings.hotkeyStop || "End";
  root.querySelector("#notifyOnRecover").checked = !!settings.notifyOnRecover;
}

export function mergeMonitorSettings(global, monitorPartial) {
  return { ...global, ...monitorPartial };
}

export function mergeGlobalSettings(base, globalPartial) {
  return { ...base, ...globalPartial };
}

export function formatWindowBindLabel(windowBindState) {
  if (windowBindState.hwnd) {
    return `已绑定：${windowBindState.title} (0x${windowBindState.hwnd.toString(16).toUpperCase()})`;
  }
  if (windowBindState.title) {
    return `标题回退：${windowBindState.title}`;
  }
  return "未绑定（将使用全屏截图）";
}
