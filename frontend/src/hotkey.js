export function setupHotkeyRecord(root, inputId, btnId, log, label) {
  const input = root.querySelector(`#${inputId}`);
  const btn = root.querySelector(`#${btnId}`);
  if (!input || !btn) return;

  btn.addEventListener("click", () => {
    log.append(`请按下新的${label}热键…`);
    input.value = "…";
    const handler = (ev) => {
      ev.preventDefault();
      ev.stopPropagation();
      const key = ev.key.length === 1 ? ev.key.toUpperCase() : ev.key;
      input.value = key;
      window.removeEventListener("keydown", handler, true);
    };
    window.addEventListener("keydown", handler, true);
  });
}
