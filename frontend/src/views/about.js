import { BrowserOpenURL } from "../../wailsjs/runtime/runtime";

export function renderAboutModal(container, appInfo, onClose) {
  const repoUrl =
    appInfo.repoUrl || "https://github.com/tlcc-tech/tlbb-game-screen-monitor";

  container.innerHTML = `
    <div class="modal modal--wide">
      <h2 class="modal-title">关于我们</h2>
      <div class="modal-scroll">
        <p class="modal-text">
          争取用免费的工具，鞭策畅游的开发人员和策划，把游戏做得更好。CC科技更专注《怀旧天龙八部》玩家服务，玩转江湖更轻松；更新抢先看，第一时间解读游戏公告，分析版本变动；
          打造思路全分享，从入门到精通，门派养成、装备搭配、珍兽打造。CC科技，永久免费。
        </p>
        <p class="modal-meta">
          作者：${escapeHtml(appInfo.author || "怀旧天龙CC科技")}　版本：${escapeHtml(appInfo.version || "-")}
        </p>
        <p class="modal-meta">
          源码：
          <button class="link-btn inline-link" id="repoLinkBtn" type="button">${escapeHtml(repoUrl)}</button>
        </p>

        <div class="about-qrcodes">
          <div class="about-qrcode-item">
            <span class="about-qrcode-label">公众号</span>
            <img class="about-qrcode-img" src="/qrcode_gzh.jpg" alt="公众号二维码" />
          </div>
          <div class="about-qrcode-item">
            <span class="about-qrcode-label">小程序</span>
            <img class="about-qrcode-img" src="/qrcode.jpg" alt="小程序二维码" />
          </div>
        </div>

        <div class="about-disclaimer">
          <strong>免责声明</strong>
          <p>本工具完全免费，仅供学习交流使用，请勿用于任何商业用途。使用本工具所产生的一切后果由使用者自行承担。</p>
        </div>
      </div>
      <div class="btn-row modal-actions">
        <button class="btn primary" id="aboutCloseBtn" type="button">关闭</button>
      </div>
    </div>
  `;

  container.querySelector("#repoLinkBtn").addEventListener("click", () => {
    BrowserOpenURL(repoUrl);
  });
  container.querySelector("#aboutCloseBtn").addEventListener("click", onClose);
}

function escapeHtml(s) {
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}
