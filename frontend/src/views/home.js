import { FEATURES, icon } from "../icons.js";

export function renderHome(root, { onOpenFeature, onClearLog }) {
  root.innerHTML = `
    <div class="home-body">
      <section class="home-left">
        <h2 class="section-label">功能模块</h2>
        <div class="func-grid">
          ${FEATURES.map(
            (f) => `
            <button class="func-card" type="button" data-feature="${f.id}" data-route="${f.route || ""}">
              <span class="func-icon-wrap" style="--accent: ${f.color}">
                ${icon(f.id, "func-icon")}
              </span>
              <span class="func-title">${f.title}</span>
              <span class="func-desc">${f.desc}</span>
            </button>
          `,
          ).join("")}
        </div>
      </section>
      <aside class="home-right">
        <h2 class="section-label">运行日志</h2>
        <div class="log-box" id="globalLog"></div>
        <button class="log-clear" type="button" id="clearLogBtn">清空日志</button>
      </aside>
    </div>
  `;

  root.querySelectorAll(".func-card").forEach((btn) => {
    btn.addEventListener("click", () => {
      onOpenFeature(btn.dataset.feature, btn.dataset.route);
    });
  });

  root.querySelector("#clearLogBtn").addEventListener("click", onClearLog);

  return root.querySelector("#globalLog");
}
