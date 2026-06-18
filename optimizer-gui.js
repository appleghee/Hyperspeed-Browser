(function () {
  if (document.getElementById("__mb_opt_panel")) return;
  var s = document.createElement("style");
  s.textContent =
    "#__mb_opt_wrap{font-family:Segoe UI,sans-serif;font-size:12px;z-index:2147483646;position:fixed;bottom:70px;right:20px;pointer-events:none}" +
    "#__mb_opt_btn{pointer-events:auto;width:38px;height:38px;border-radius:50%;background:rgba(26,26,46,0.92);backdrop-filter:blur(16px);border:1px solid rgba(255,255,255,0.1);box-shadow:0 4px 16px rgba(0,0,0,0.4);cursor:pointer;display:flex;align-items:center;justify-content:center;font-size:18px;color:#e0e0e0;position:absolute;bottom:0;right:0}" +
    "#__mb_opt_btn:hover{transform:scale(1.1)}#__mb_opt_btn.active{background:rgba(79,195,247,0.2)}" +
    "#__mb_opt_panel{pointer-events:auto;position:absolute;bottom:46px;right:0;width:320px;max-height:70vh;background:rgba(22,22,40,0.96);backdrop-filter:blur(24px);border:1px solid rgba(255,255,255,0.08);border-radius:14px;box-shadow:0 8px 40px rgba(0,0,0,0.5);color:#e0e0e0;display:none;flex-direction:column;overflow:hidden}" +
    "#__mb_opt_panel.show{display:flex}" +
    "#__mb_opt_panel ::-webkit-scrollbar{width:3px}#__mb_opt_panel ::-webkit-scrollbar-thumb{background:rgba(79,195,247,0.3);border-radius:2px}" +
    ".opt-h{padding:12px 14px 8px;border-bottom:1px solid rgba(255,255,255,0.06);display:flex;align-items:center;justify-content:space-between}" +
    ".opt-hl{font-weight:700;font-size:13px;display:flex;align-items:center;gap:6px}.opt-hl s{color:#4fc3f7}" +
    ".opt-b{padding:6px 14px 12px;overflow-y:auto;flex:1;display:flex;flex-direction:column;gap:6px}" +
    ".opt-g{display:flex;flex-direction:column;gap:3px}" +
    ".opt-gl{font-size:9px;text-transform:uppercase;letter-spacing:0.05em;color:#888;font-weight:600}" +
    ".opt-sel{width:100%;padding:5px 8px;border-radius:6px;border:1px solid rgba(255,255,255,0.1);background:rgba(255,255,255,0.04);color:#ddd;font-family:inherit;font-size:11px;outline:none;cursor:pointer}" +
    ".opt-sel:focus{border-color:#4fc3f7}.opt-sel option{background:#1a1a2e;color:#ddd}" +
    ".opt-row{display:flex;align-items:center;justify-content:space-between;padding:3px 0}" +
    ".opt-lbl{font-size:11px;color:#bbb}" +
    ".opt-tg{position:relative;width:34px;height:18px;flex-shrink:0}.opt-tg input{opacity:0;width:0;height:0}" +
    ".opt-tgs{position:absolute;cursor:pointer;top:0;left:0;right:0;bottom:0;background:rgba(255,255,255,0.12);border-radius:18px}" +
    ".opt-tgs:before{content:'';position:absolute;height:12px;width:12px;left:3px;bottom:3px;background:#fff;border-radius:50%}" +
    ".opt-tg input:checked+.opt-tgs{background:#4fc3f7}" +
    ".opt-tg input:checked+.opt-tgs:before{transform:translateX(16px)}" +
    ".opt-act{display:flex;gap:4px;flex-wrap:wrap}" +
    ".opt-btn{padding:5px 10px;border-radius:6px;border:none;cursor:pointer;font-size:10px;font-weight:600;font-family:inherit;white-space:nowrap}" +
    ".opt-btn-p{background:#4fc3f7;color:#000}.opt-btn-s{background:rgba(255,255,255,0.08);color:#bbb}" +
    ".opt-btn-s2{background:rgba(79,195,247,0.15);color:#4fc3f7}" +
    ".opt-stat{display:grid;grid-template-columns:1fr 1fr 1fr;gap:3px;text-align:center}" +
    ".opt-stat div{padding:5px 2px;background:rgba(255,255,255,0.03);border-radius:5px}" +
    ".opt-stat b{display:block;font-size:14px;color:#4fc3f7}.opt-stat s{font-size:8px;color:#888}" +
    ".opt-badge{font-size:8px;padding:1px 5px;border-radius:7px;font-weight:600}" +
    ".opt-bd-green{background:rgba(102,187,106,0.2);color:#66bb6a}.opt-bd-yellow{background:rgba(255,167,38,0.2);color:#ffa726}.opt-bd-red{background:rgba(239,83,80,0.2);color:#ef5350}" +
    ".opt-ta{width:100%;height:60px;border-radius:6px;border:1px solid rgba(255,255,255,0.1);background:rgba(255,255,255,0.04);color:#ccc;font-family:monospace;font-size:10px;padding:5px;resize:vertical;outline:none}" +
    ".opt-ta:focus{border-color:#4fc3f7}" +
    ".opt-hint{font-size:8px;color:#666;margin-top:1px}";
  document.head.appendChild(s);
  var p = document.createElement("div");
  p.id = "__mb_opt_wrap";
  p.innerHTML =
    '<button id="__mb_opt_btn" title="Optimizer (Ctrl+Shift+Space)">\u2699</button>' +
    '<div id="__mb_opt_panel">' +
    '<div class="opt-h"><div class="opt-hl"><s>\u2699</s>Optimizer</div><span id="__mb_opt_badge" class="opt-badge opt-bd-green">Balanced</span></div>' +
    '<div class="opt-b">' +
    '<div class="opt-g"><div class="opt-gl">Mode</div>' +
    '<select id="__mb_opt_mode" class="opt-sel">' +
    '<optgroup label="Performance">' +
    '<option value="turbo">Turbo</option><option value="aggressive">Aggressive</option><option value="speed">Speed</option>' +
    '<option value="balanced" selected>Balanced</option>' +
    '</optgroup><optgroup label="Special">' +
    '<option value="eco">Eco</option><option value="mobile">Mobile</option><option value="compat">Compat</option>' +
    "</optgroup></select></div>" +
    '<div class="opt-g"><div class="opt-gl">Live Stats</div>' +
    '<div class="opt-stat"><div><b id="__mb_opt_s">-</b><s>Score</s></div><div><b id="__mb_opt_l">-</b><s>Load</s></div><div><b id="__mb_opt_r">-</b><s>Reqs</s></div></div></div>' +
    '<div class="opt-g"><div class="opt-gl">Actions</div>' +
    '<div class="opt-act"><button class="opt-btn opt-btn-p" id="__mb_opt_m">\u25b6 Do</button>' +
    '<button class="opt-btn opt-btn-s" id="__mb_opt_x">\u26a1 Optimize</button>' +
    '<button class="opt-btn opt-btn-s" id="__mb_opt_n">\u2630 Snap</button></div></div>' +
    '<div class="opt-g"><div class="opt-gl">Toggles</div>' +
    '<div class="opt-row"><span class="opt-lbl">Lazy images</span><label class="opt-tg"><input type="checkbox" checked id="__mb_t_i"><span class="opt-tgs"></span></label></div>' +
    '<div class="opt-row"><span class="opt-lbl">Defer JS</span><label class="opt-tg"><input type="checkbox" checked id="__mb_t_j"><span class="opt-tgs"></span></label></div>' +
    '<div class="opt-row"><span class="opt-lbl">Block trackers</span><label class="opt-tg"><input type="checkbox" checked id="__mb_t_t"><span class="opt-tgs"></span></label></div>' +
    '<div class="opt-row"><span class="opt-lbl">Smart cache</span><label class="opt-tg"><input type="checkbox" checked id="__mb_t_c"><span class="opt-tgs"></span></label></div>' +
    '<div class="opt-g"><div class="opt-gl">Custom Script</div>' +
    '<textarea id="__mb_script_ta" class="opt-ta" placeholder="Paste JS here..."></textarea>' +
    '<div class="opt-act" style="margin-top:3px">' +
    '<button class="opt-btn opt-btn-s2" id="__mb_script_save">\u2714 Save</button>' +
    '<label class="opt-tg" style="margin-left:auto"><input type="checkbox" id="__mb_script_auto"><span class="opt-tgs"></span></label>' +
    '<span class="opt-lbl" style="font-size:9px">Auto</span></div>' +
    '<span class="opt-hint">Ctrl+Shift+R to run script</span></div>' +
    '<div class="opt-g" style="border-top:1px solid rgba(255,255,255,0.06);padding-top:5px"><div class="opt-gl">PVDS Value Density</div>' +
    '<div class="opt-stat"><div><b id="__mb_vd_avg">-</b><s>Avg VD</s></div><div><b id="__mb_vd_hi">-</b><s>High</s></div><div><b id="__mb_vd_lo">-</b><s>Low</s></div></div>' +
    '<div class="opt-stat" style="margin-top:2px"><div><b id="__mb_vd_mem">-</b><s>Mem MB</s></div><div><b id="__mb_vd_bud">-</b><s>Budget</s></div><div><b id="__mb_vd_frz">-</b><s>Frozen</s></div></div>' +
    '<div class="opt-act"><button class="opt-btn opt-btn-s2" id="__mb_vd_scan">\u25b6 Scan</button>' +
    '<button class="opt-btn opt-btn-s" id="__mb_vd_opt">\u26a1 Schedule</button></div></div>' +
    '<div class="opt-g" style="border-top:1px solid rgba(255,255,255,0.06);padding-top:5px"><div class="opt-gl">CRG Reuse Graph</div>' +
    '<div class="opt-stat"><div><b id="__mb_crg_h">-</b><s>Hits</s></div><div><b id="__mb_crg_m">-</b><s>Miss</s></div><div><b id="__mb_crg_s">-</b><s>Saved</s></div></div>' +
    '<div class="opt-stat" style="margin-top:2px"><div><b id="__mb_crg_r">-</b><s>Reused</s></div><div><b id="__mb_crg_st">-</b><s>Stale</s></div><div><b id="__mb_crg_c">-</b><s>Cached</s></div></div>' +
    '<div class="opt-act"><button class="opt-btn opt-btn-s2" id="__mb_crg_scan">\u25b6 Scan</button>' +
    '<button class="opt-btn opt-btn-s" id="__mb_crg_opt">\u267b Cache</button></div></div>' +
    '<div class="opt-g" style="border-top:1px solid rgba(255,255,255,0.06);padding-top:5px"><div class="opt-gl">\u26a1 QuickOpt (5 features)</div>' +
    '<div class="opt-stat"><div><b id="__mb_q_mddp">-</b><s>MDDP</s></div><div><b id="__mb_q_htp">-</b><s>HTP</s></div><div><b id="__mb_q_psfq">-</b><s>PSFQ</s></div></div>' +
    '<div class="opt-stat" style="margin-top:2px"><div><b id="__mb_q_dae">-</b><s>DAE</s></div><div><b id="__mb_q_fdtf">-</b><s>FDTF</s></div><div><b id="__mb_q_s">-</b><s>Status</s></div></div>' +
    '<div class="opt-act"><button class="opt-btn opt-btn-p" id="__mb_q_inj">\u25b6 Inject</button>' +
    '<button class="opt-btn opt-btn-s" id="__mb_q_st">\u2139 Stats</button></div></div>' +
    '<div class="opt-g" style="border-top:1px solid rgba(255,255,255,0.06);padding-top:5px"><div class="opt-gl">\u267b RHD-GC + PVC</div>' +
    '<div class="opt-stat"><div><b id="__mb_rhd_d">-</b><s>Detach</s></div><div><b id="__mb_rhd_f">-</b><s>Frozen</s></div><div><b id="__mb_rhd_x">-</b><s>Destroy</s></div></div>' +
    '<div class="opt-stat" style="margin-top:2px"><div><b id="__mb_pvc_f">-</b><s>Full</s></div><div><b id="__mb_pvc_s">-</b><s>Skel</s></div><div><b id="__mb_pvc_c">-</b><s>Coll</s></div></div>' +
    '<div class="opt-act"><button class="opt-btn opt-btn-p" id="__mb_rhd_start">\u25b6 Start</button>' +
    '<button class="opt-btn opt-btn-s" id="__mb_pvc_start">\u25b6 PVC</button>' +
    '<button class="opt-btn opt-btn-s" id="__mb_rhd_st">\u2139</button></div></div>' +
    '<div class="opt-g" style="border-top:1px solid rgba(255,255,255,0.06);padding-top:5px"><div class="opt-gl">\u26a1 EHS Scheduler</div>' +
    '<div class="opt-stat"><div><b id="__mb_ehs_h">-</b><s>Hot</s></div><div><b id="__mb_ehs_w">-</b><s>Warm</s></div><div><b id="__mb_ehs_c">-</b><s>Cold</s></div></div>' +
    '<div class="opt-stat" style="margin-top:2px"><div><b id="__mb_ehs_d">-</b><s>Dormant</s></div><div><b id="__mb_ehs_s">-</b><s>Split</s></div><div><b id="__mb_ehs_t">-</b><s>Total</s></div></div>' +
    '<div class="opt-act"><button class="opt-btn opt-btn-p" id="__mb_ehs_start">\u25b6 EHS</button>' +
    '<button class="opt-btn opt-btn-s" id="__mb_ehs_st">\u2139</button></div></div>' +
    '<div class="opt-g" style="border-top:1px solid rgba(255,255,255,0.06);padding-top:4px"><div class="opt-gl" style="color:#ffa726">\u2728 Easter Eggs</div>' +
    '<span style="font-size:9px;color:#888;display:block;padding:2px 0">Konami code \u2192 rainbow panel | type "opencode"</span></div>' +
    "</div></div></div>";
  document.body.appendChild(p);
  var port = window.__mbPort || 0;
  var token = window.__mbToken || "";
  function api(m, path, bd) {
    return fetch("http://127.0.0.1:" + port + path, {
      method: m || "GET",
      headers: { "Content-Type": "application/json", "X-API-Token": token },
      body: bd ? JSON.stringify(bd) : null,
    })
      .then(function (r) {
        return r.json();
      })["catch"](function () {
        return null;
      });
  }
  var btn = document.getElementById("__mb_opt_btn"),
    panel = document.getElementById("__mb_opt_panel");
  function togglePanel() {
    panel.classList.toggle("show");
    btn.classList.toggle("active");
    if (panel.classList.contains("show")) rs();
  }
  btn.onclick = togglePanel;
  document.addEventListener("keydown", function (e) {
    if (e.ctrlKey && e.shiftKey && e.code === "Space") {
      e.preventDefault();
      togglePanel();
    }
    if (e.ctrlKey && e.shiftKey && e.code === "KeyR") {
      e.preventDefault();
      runScript();
    }
  });
  var md = document.getElementById("__mb_opt_mode");
  md.onchange = function () {
    var v = md.value;
    api("POST", "/api/opt/profile", { profile: v }).then(function (d) {
      if (d && d.ok) {
        bg(v);
        rs();
      }
    });
  };
  function bg(p) {
    var e = document.getElementById("__mb_opt_badge");
    e.textContent = p.charAt(0).toUpperCase() + p.slice(1);
    e.className =
      "opt-badge " +
      (p === "turbo" || p === "aggressive"
        ? "opt-bd-red"
        : p === "speed"
          ? "opt-bd-yellow"
          : "opt-bd-green");
  }
  function rs() {
    api("GET", "/api/opt/metrics").then(function (d) {
      if (d && d.metrics) {
        var m = d.metrics;
        document.getElementById("__mb_opt_s").textContent = m.score || "-";
        document.getElementById("__mb_opt_l").textContent = m.loadTimeMs
          ? Math.round(m.loadTimeMs)
          : "-";
        document.getElementById("__mb_opt_r").textContent =
          m.requestCount || "-";
      }
    });
    vdScan();crgScan();qStats();
  }
  document.getElementById("__mb_opt_m").onclick = rs;
  document.getElementById("__mb_opt_x").onclick = function () {
    api("POST", "/api/opt/run").then(function (d) {
      if (d) rs();
    });
  };
  document.getElementById("__mb_opt_n").onclick = function () {
    api("GET", "/api/snapshot").then(function (d) {
      if (d && d.result)
        document.getElementById("__mb_opt_s").textContent =
          d.result.length + "n";
    });
  };
  ["i", "j", "t", "c"].forEach(function (k) {
    var el = document.getElementById("__mb_t_" + k);
    if (el) el.onchange = function () {};
  });
  var ta = document.getElementById("__mb_script_ta");
  var autoCb = document.getElementById("__mb_script_auto");
  try {
    var saved = JSON.parse(localStorage.getItem("__mb_customScript") || "{}");
    if (saved.code) ta.value = saved.code;
    if (saved.auto) autoCb.checked = saved.auto;
  } catch (e) {}
  function saveScript() {
    var code = ta.value.trim();
    var auto = autoCb.checked;
    try {
      localStorage.setItem(
        "__mb_customScript",
        JSON.stringify({ code: code, auto: auto })
      );
    } catch (e) {}
  }
  function runScript() {
    var code = ta.value.trim();
    if (code) {
      try {
        new Function(code)();
      } catch (e) {
        console.error("[CustomScript]", e);
      }
    }
  }
  function vdScan() {
    api("GET", "/api/vd/snapshot").then(function (d) {
      if (d && d.stats) {
        var s = d.stats;
        document.getElementById("__mb_vd_avg").textContent = s.avgVD;
        document.getElementById("__mb_vd_hi").textContent = s.highValue;
        document.getElementById("__mb_vd_lo").textContent = s.lowValue;
        document.getElementById("__mb_vd_mem").textContent =
          s.usedMB + " / " + s.budgetMB;
        document.getElementById("__mb_vd_frz").textContent = s.freezeZones;
      }
    });
  }
  function vdOpt() {
    api("POST", "/api/vd/optimize").then(function (d) {
      if (d && d.stats) vdScan();
    });
  }
  function crgScan() {
    api("GET", "/api/crg/snapshot").then(function (d) {
      if (d && d.stats) {
        var s = d.stats;
        document.getElementById("__mb_crg_h").textContent = s.cacheHits;
        document.getElementById("__mb_crg_m").textContent = s.cacheMisses;
        document.getElementById("__mb_crg_s").textContent = s.totalSaved;
        document.getElementById("__mb_crg_r").textContent = s.reusedNodes;
        document.getElementById("__mb_crg_st").textContent = s.staleNodes;
        document.getElementById("__mb_crg_c").textContent = s.cacheSize;
      }
    });
  }
  function crgOpt() {
    api("POST", "/api/crg/optimize").then(function (d) {
      if (d && d.stats) crgScan();
    });
  }
  function qStats() {
    api("GET", "/api/quick/stats").then(function (d) {
      if (d && d.stats) {
        var s = d.stats;
        document.getElementById("__mb_q_mddp").textContent =
          (s.mddp ? s.mddp.dnsHits + s.mddp.tcpHits : "-");
        document.getElementById("__mb_q_htp").textContent =
          (s.htp ? s.htp.preconnects : "-");
        document.getElementById("__mb_q_psfq").textContent =
          (s.psfq ? s.psfq.total : "-");
        document.getElementById("__mb_q_dae").textContent =
          (s.dae ? s.dae.decoded + "/" + s.dae.evicted : "-");
        document.getElementById("__mb_q_fdtf").textContent =
          (s.fdtf ? s.fdtf.fallbacks + "/" + s.fdtf.swapped : "-");
        document.getElementById("__mb_q_s").textContent = "OK";
      }
    });
  }
  function qInject() {
    api("POST", "/api/quick/inject").then(function (d) {
      if (d && d.ok) {
        document.getElementById("__mb_q_s").textContent = "Injected";
        qStats();
      }
    });
  }
  function rhdStats() {
    api("GET", "/api/rhd/stats").then(function (d) {
      if (d && d.stats) {
        var s = d.stats;
        document.getElementById("__mb_rhd_d").textContent = s.detached;
        document.getElementById("__mb_rhd_f").textContent = s.frozen;
        document.getElementById("__mb_rhd_x").textContent = s.destroyed;
      }
    });
    api("GET", "/api/pvc/stats").then(function (d) {
      if (d && d.stats) {
        var s = d.stats;
        document.getElementById("__mb_pvc_f").textContent = s.full;
        document.getElementById("__mb_pvc_s").textContent = s.skeleton;
        document.getElementById("__mb_pvc_c").textContent = s.collapsed;
      }
    });
  }
  document.getElementById("__mb_rhd_start").onclick = function() {
    api("POST", "/api/rhd/start").then(function(d) { if(d && d.ok) rhdStats(); });
  };
  document.getElementById("__mb_pvc_start").onclick = function() {
    api("POST", "/api/pvc/start").then(function(d) { if(d && d.ok) rhdStats(); });
  };
  function ehsStats() {
    api("GET", "/api/ehs/stats").then(function (d) {
      if (d && d.stats) {
        var s = d.stats;
        document.getElementById("__mb_ehs_h").textContent = s.hot;
        document.getElementById("__mb_ehs_w").textContent = s.warm;
        document.getElementById("__mb_ehs_c").textContent = s.cold;
        document.getElementById("__mb_ehs_d").textContent = s.dormant;
        document.getElementById("__mb_ehs_s").textContent = s.splits;
        document.getElementById("__mb_ehs_t").textContent = s.total;
      }
    });
  }
  document.getElementById("__mb_ehs_start").onclick = function() {
    api("POST", "/api/ehs/start").then(function(d) { if(d && d.ok) { ehsStats(); } });
  };
  document.getElementById("__mb_ehs_st").onclick = ehsStats;
  document.getElementById("__mb_rhd_st").onclick = rhdStats;
  document.getElementById("__mb_q_inj").onclick = qInject;
  document.getElementById("__mb_q_st").onclick = qStats;
  document.getElementById("__mb_crg_scan").onclick = crgScan;
  document.getElementById("__mb_crg_opt").onclick = crgOpt;
  document.getElementById("__mb_vd_scan").onclick = vdScan;
  document.getElementById("__mb_vd_opt").onclick = vdOpt;
  document.getElementById("__mb_script_save").onclick = saveScript;
  window.__mbRunCustomScript = runScript;
  bg("balanced");
  qInject(); rhdStats();
  api("POST", "/api/rhd/start"); api("POST", "/api/pvc/start"); api("POST", "/api/ehs/start");
  window.__mbOptGUI = true;
})();
(function () {
  if (window.__mbScriptWatcher) return;
  window.__mbScriptWatcher = true;
  try {
    var saved = JSON.parse(localStorage.getItem("__mb_customScript") || "{}");
    if (saved.code && saved.auto) {
      if (document.readyState === "loading") {
        document.addEventListener("DOMContentLoaded", function () {
          try {
            new Function(saved.code)();
          } catch (e) {
            console.error("[CustomScript]", e);
          }
        });
      } else {
        try {
          new Function(saved.code)();
        } catch (e) {
          console.error("[CustomScript]", e);
        }
      }
    }
  } catch (e) {}
  var _pushState = history.pushState;
  history.pushState = function () {
    _pushState.apply(this, arguments);
    window.__mbRunScriptNav && window.__mbRunScriptNav();
  };
  var _replaceState = history.replaceState;
  history.replaceState = function () {
    _replaceState.apply(this, arguments);
    window.__mbRunScriptNav && window.__mbRunScriptNav();
  };
  window.addEventListener("popstate", function () {
    window.__mbRunScriptNav && window.__mbRunScriptNav();
  });
  window.__mbRunScriptNav = function () {
    try {
      var saved = JSON.parse(
        localStorage.getItem("__mb_customScript") || "{}"
      );
      if (saved.code && saved.auto) {
        setTimeout(function () {
          try {
            new Function(saved.code)();
          } catch (e) {
            console.error("[CustomScript]", e);
          }
        }, 500);
      }
    } catch (e) {}
  };
})();
