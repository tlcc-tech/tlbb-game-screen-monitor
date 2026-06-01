const ICONS = {
  keyboard: `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M4 7h16a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V9a2 2 0 0 1 2-2zm2 4v2h2v-2H6zm4 0v2h2v-2h-2zm4 0v2h2v-2h-2zm4 0v2h2v-2h-2zM6 15v2h2v-2H6zm4 0v2h2v-2h-2zm4 0v2h2v-2h-2z" fill="currentColor"/></svg>`,
  cursor: `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M5 3l14 7.5L12 12l-1.5 7L5 3z" fill="currentColor"/></svg>`,
  monitor: `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M3 5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5zm4 16h10v2H7v-2z" fill="currentColor"/></svg>`,
  broom: `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M14.5 3a5.5 5.5 0 0 1 3.9 9.4L8 22.5 1.5 16l9.4-10.4A5.5 5.5 0 0 1 14.5 3zm-1 2.1a3.5 3.5 0 0 0-2.5 6l8.9 8.9 1.4-1.4-8.9-8.9a3.5 3.5 0 0 0 1.1-4.6z" fill="currentColor"/></svg>`,
  grid: `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><rect x="4" y="4" width="6" height="6" rx="1" stroke="currentColor" stroke-width="2"/><rect x="14" y="4" width="6" height="6" rx="1" stroke="currentColor" stroke-width="2"/><rect x="4" y="14" width="6" height="6" rx="1" stroke="currentColor" stroke-width="2"/><rect x="14" y="14" width="6" height="6" rx="1" stroke="currentColor" stroke-width="2"/></svg>`,
  settings: `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M12 8a4 4 0 1 1 0 8 4 4 0 0 1 0-8zm8.5 4a7.5 7.5 0 0 0-.2-1.7l2-1.5-2-3.5-2.4 1a7.6 7.6 0 0 0-1.5-1l-.4-2.6H9l-.4 2.6a7.6 7.6 0 0 0-1.5 1l-2.4-1-2 3.5 2 1.5a7.5 7.5 0 0 0 0 3.4l-2 1.5 2 3.5 2.4-1a7.6 7.6 0 0 0 1.5 1l.4 2.6h6l.4-2.6a7.6 7.6 0 0 0 1.5-1l2.4 1 2-3.5-2-1.5c.1-.6.2-1.1.2-1.7z" fill="currentColor"/></svg>`,
  info: `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z" fill="currentColor"/></svg>`,
  back: `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M15 6l-6 6 6 6" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>`,
};

export function icon(name, className = "icon") {
  const svg = ICONS[name] || "";
  return `<span class="${className}" aria-hidden="true">${svg}</span>`;
}

export const FEATURES = [
  {
    id: "keyboard",
    title: "按键打怪",
    desc: "自动战斗",
    color: "#D97706",
    route: null,
  },
  {
    id: "cursor",
    title: "鼠标连点",
    desc: "高速连点",
    color: "#F59E0B",
    route: null,
  },
  {
    id: "monitor",
    title: "挂机监控",
    desc: "实时监控",
    color: "#7C3AED",
    route: "monitor",
  },
  {
    id: "broom",
    title: "副本扫荡",
    desc: "批量清图",
    color: "#DC2626",
    route: null,
  },
  {
    id: "grid",
    title: "更多功能",
    desc: "扩展工具",
    color: "#64748B",
    route: null,
  },
];
