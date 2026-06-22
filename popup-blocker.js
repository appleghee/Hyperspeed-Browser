(function() {
  if (window.__mbSPB) return;
  window.__mbSPB = true;

  const STORAGE_KEY = 'spb_config';
  const DEFAULT_CONFIG = {
    enabled: true,
    smartMode: true,
    blockAllPopups: false,
    autoBlockAds: true,
    notificationDuration: 8000,
    notificationPosition: 'br',
    maxNotifications: 3,
    whitelist: [],
    blacklist: [],
    blockedCount: 0,
    showBlockedBadge: true,
    theme: 'dark',
    fontSize: 'small',
    soundOnBlock: false,
    logBlockedToConsole: false
  };

  // ========== Cấu hình ==========
  function loadConfig() {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (raw) {
        const parsed = JSON.parse(raw);
        if (parsed && typeof parsed === 'object') {
          const merged = {};
          for (const key in DEFAULT_CONFIG) merged[key] = DEFAULT_CONFIG[key];
          for (const key in parsed) if (key in merged) merged[key] = parsed[key];
          return merged;
        }
      }
    } catch (e) {}
    return Object.assign({}, DEFAULT_CONFIG);
  }

  function saveConfig(cfg) {
    try { localStorage.setItem(STORAGE_KEY, JSON.stringify(cfg)); } catch (e) {}
  }

  window.__mbSPBConfig = loadConfig();

  // ========== Tiện ích ==========
  function getDomain(url) {
    try {
      if (!url || url === 'about:blank' || url === '') return '';
      if (url.indexOf('javascript:') === 0 || url.indexOf('data:') === 0) return 'special';
      return new URL(url, location.href).hostname.toLowerCase();
    } catch (e) {
      const m = url.match(/^(?:https?:\/\/)?([^/:?#]+)/i);
      return m ? m[1].toLowerCase() : '';
    }
  }

  function matchDomain(domain, entry) {
    // khớp chính xác hoặc domain con: entry = "example.com" khớp "example.com" và "www.example.com"
    if (domain === entry) return true;
    if (domain.endsWith('.' + entry)) return true;
    return false;
  }

  function isAdsUrl(url) {
    if (!url) return false;
    const patterns = [
      /doubleclick\.net/i, /googleadservices/i, /googlesyndication/i,
      /adnxs\.com/i, /adsystem/i, /adserver/i, /advertisement/i,
      /popads/i, /popunder/i, /tabunder/i, /trafficjunky/i,
      /click\.php/i, /track\.php/i, /redirect\.php/i, /\/ad\//i,
      /\/ads\//i, /\/banner\//i, /\/popup\//i, /sponsor/i,
      /affiliate/i, /campaign/i, /utm_/i, /\.exe$/i, /\.apk$/i,
      /\.dmg$/i, /\.msi$/i, /download\.php/i, /file\.php/i
    ];
    return patterns.some(p => p.test(url));
  }

  const originalWindowOpen = window.open;
  const popupQueue = [];
  window.__mbSPBQueue = popupQueue;
  function capQueue() { while(popupQueue.length > 50) popupQueue.shift() }

  // ========== Thông báo UI ==========
  let notificationContainer = null;

  function createNotificationContainer() {
    if (notificationContainer) return;
    const cfg = window.__mbSPBConfig;
    const container = document.createElement('div');
    container.id = 'mb-spb-notifications';
    const pos = cfg.notificationPosition || 'br';
    const positions = {
      tr: 'top: 12px; right: 12px;',
      tl: 'top: 12px; left: 12px;',
      br: 'bottom: 12px; right: 12px;',
      bl: 'bottom: 12px; left: 12px;'
    };
    container.style.cssText = `
      position: fixed; z-index: 2147483647; ${positions[pos] || positions.br}
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      pointer-events: auto;
    `;
    document.body.appendChild(container);
    notificationContainer = container;
  }

  function getThemeStyles() {
    const cfg = window.__mbSPBConfig;
    const dark = cfg.theme === 'dark';
    return {
      bg: dark ? 'rgba(30,30,30,0.9)' : 'rgba(255,255,255,0.9)',
      text: dark ? '#e0e0e0' : '#222',
      border: dark ? '1px solid #555' : '1px solid #ccc',
      closeColor: dark ? '#aaa' : '#666',
      titleColor: dark ? '#ff5f5f' : '#d32f2f'
    };
  }

  function getFontSizeClass() {
    const size = window.__mbSPBConfig.fontSize;
    if (size === 'large') return '16px';
    if (size === 'medium') return '14px';
    return '12px';
  }

  function playBlockSound() {
    try {
      const ctx = new (window.AudioContext || window.webkitAudioContext)();
      const osc = ctx.createOscillator();
      const gain = ctx.createGain();
      osc.type = 'square';
      osc.frequency.setValueAtTime(800, ctx.currentTime);
      osc.frequency.exponentialRampToValueAtTime(400, ctx.currentTime + 0.15);
      gain.gain.setValueAtTime(0.1, ctx.currentTime);
      gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.15);
      osc.connect(gain);
      gain.connect(ctx.destination);
      osc.start(ctx.currentTime);
      osc.stop(ctx.currentTime + 0.15);
    } catch(e) {}
  }

  function showNotification(popupInfo) {
    const cfg = window.__mbSPBConfig;
    if (!cfg.showBlockedBadge) return;

    if (cfg.soundOnBlock) playBlockSound();
    if (cfg.logBlockedToConsole) {
      console.log('[Smart Popup Blocker] Blocked:', popupInfo.url);
    }

    createNotificationContainer();
    const theme = getThemeStyles();
    const fontSize = getFontSizeClass();

    // Giới hạn số thông báo hiển thị
    const currentItems = notificationContainer.querySelectorAll('.mb-spb-item');
    if (currentItems.length >= cfg.maxNotifications) {
      currentItems[0].remove();
    }

    const item = document.createElement('div');
    item.className = 'mb-spb-item';
    item.style.cssText = `
      background: ${theme.bg};
      color: ${theme.text};
      border: ${theme.border};
      border-radius: 8px;
      padding: 8px 12px;
      margin-bottom: 6px;
      font-size: ${fontSize};
      box-shadow: 0 4px 12px rgba(0,0,0,0.3);
      display: flex;
      align-items: center;
      gap: 8px;
      max-width: 320px;
      backdrop-filter: blur(6px);
      transition: opacity 0.2s;
    `;

    const textSpan = document.createElement('span');
    textSpan.textContent = `🚫 Đã chặn: ${cfg.blockedCount} popup`;
    textSpan.style.flex = '1';

    const closeBtn = document.createElement('span');
    closeBtn.textContent = '✕';
    closeBtn.style.cssText = `
      cursor: pointer; font-weight: bold; color: ${theme.closeColor};
      font-size: 14px; line-height: 1; padding: 2px;
    `;
    closeBtn.onclick = () => item.remove();

    item.appendChild(textSpan);
    item.appendChild(closeBtn);
    notificationContainer.appendChild(item);

    if (cfg.notificationDuration > 0) {
      setTimeout(() => {
        if (item.parentNode) item.remove();
      }, cfg.notificationDuration);
    }
  }

  // ========== Xử lý popup ==========
  function handlePopupRequest(url, target, features) {
    const cfg = window.__mbSPBConfig;
    if (!cfg.enabled) return originalWindowOpen.call(window, url, target, features);

    // Các loại URL đặc biệt luôn được cho phép
    if (!url || url === 'about:blank' || url === '') return originalWindowOpen.call(window, url, target, features);
    if (url.indexOf('data:') === 0 || url.indexOf('blob:') === 0 || url.indexOf('javascript:') === 0) {
      return originalWindowOpen.call(window, url, target, features);
    }

    const td = getDomain(url);
    const cd = location.hostname.toLowerCase();

    // Blacklist
    for (let i = 0; i < cfg.blacklist.length; i++) {
      if (matchDomain(td, cfg.blacklist[i])) {
        cfg.blockedCount++;
        saveConfig(cfg);
        capQueue();popupQueue.push({
          id: Date.now() + '_' + Math.random().toString(36).substr(2, 9),
          url, target, features, targetDomain: td, currentDomain: cd, timestamp: Date.now()
        });
        showNotification(popupQueue[popupQueue.length - 1]);
        return null;
      }
    }

    // Whitelist
    for (let i = 0; i < cfg.whitelist.length; i++) {
      if (matchDomain(td, cfg.whitelist[i])) {
        return originalWindowOpen.call(window, url, target, features);
      }
    }

    // Smart mode: cùng domain
    if (cfg.smartMode && td === cd) {
      return originalWindowOpen.call(window, url, target, features);
    }

    // Tự động chặn quảng cáo
    if (cfg.autoBlockAds && isAdsUrl(url)) {
      cfg.blockedCount++;
      saveConfig(cfg);
      capQueue();popupQueue.push({
        id: Date.now() + '_' + Math.random().toString(36).substr(2, 9),
        url, target, features, targetDomain: td, currentDomain: cd, timestamp: Date.now()
      });
      showNotification(popupQueue[popupQueue.length - 1]);
      return null;
    }

    // Chặn tất cả popup
    if (cfg.blockAllPopups) {
      cfg.blockedCount++;
      saveConfig(cfg);
      capQueue();popupQueue.push({
        id: Date.now() + '_' + Math.random().toString(36).substr(2, 9),
        url, target, features, targetDomain: td, currentDomain: cd, timestamp: Date.now()
      });
      showNotification(popupQueue[popupQueue.length - 1]);
      return null;
    }

    // Mặc định: cho phép popup
    return originalWindowOpen.call(window, url, target, features);
  }

  window.open = function(url, target, features) {
    return handlePopupRequest(url, target, features);
  };
  window._originalOpen = originalWindowOpen;

  // ========== Bắt sự kiện click lên link ==========
  document.addEventListener('click', function(e) {
    const cfg = window.__mbSPBConfig;
    if (!cfg.enabled) return;

    const a = e.target.closest('a');
    if (!a) return;

    const targetAttr = a.getAttribute('target');
    if (targetAttr !== '_blank' && targetAttr !== '_new' && targetAttr !== 'popup') return;

    const href = a.getAttribute('href');
    if (!href || href.indexOf('javascript:') === 0) return;
    if (href.indexOf('#') === 0 && a.pathname === location.pathname && a.search === location.search) return;

    e.preventDefault();
    e.stopImmediatePropagation();

    // Dùng lại hàm window.open đã ghi đè để áp dụng toàn bộ luật
    window.open(a.href, targetAttr, '');
  }, true);
})();