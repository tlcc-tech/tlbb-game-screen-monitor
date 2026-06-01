const MAX_LINES = 2000;

function formatTime(date = new Date()) {
  const h = String(date.getHours()).padStart(2, "0");
  const m = String(date.getMinutes()).padStart(2, "0");
  const s = String(date.getSeconds()).padStart(2, "0");
  return `${h}:${m}:${s}`;
}

export function createLog() {
  const lines = [];
  let containerEl = null;

  function render() {
    if (!containerEl) return;
    containerEl.innerHTML = lines
      .map(
        (line) =>
          `<div class="log-line${line.highlight ? " log-line--highlight" : ""}">${escapeHtml(line.text)}</div>`,
      )
      .join("");
    containerEl.scrollTop = containerEl.scrollHeight;
  }

  function append(text, { highlight = false, withTime = true } = {}) {
    const prefix = withTime ? `[${formatTime()}] ` : "";
    lines.push({ text: `${prefix}${text}`, highlight });
    if (lines.length > MAX_LINES) {
      lines.splice(0, lines.length - MAX_LINES);
    }
    render();
  }

  function clear() {
    lines.length = 0;
    render();
  }

  function mount(el) {
    containerEl = el;
    render();
  }

  function unmount() {
    containerEl = null;
  }

  return { append, clear, mount, unmount };
}

function escapeHtml(s) {
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}
