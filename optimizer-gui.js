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
    ".opt-stat{display:grid;grid-template-columns:1fr 1fr 1fr;gap:3px;text-align:center}" +
    ".opt-stat div{padding:5px 2px;background:rgba(255,255,255,0.03);border-radius:5px}" +
    ".opt-stat b{display:block;font-size:14px;color:#4fc3f7}.opt-stat s{font-size:8px;color:#888}" +
    ".opt-badge{font-size:8px;padding:1px 5px;border-radius:7px;font-weight:600}" +
    ".opt-bd-green{background:rgba(102,187,106,0.2);color:#66bb6a}.opt-bd-yellow{background:rgba(255,167,38,0.2);color:#ffa726}.opt-bd-red{background:rgba(239,83,80,0.2);color:#ef5350}";
  document.head.appendChild(s);
  var p = document.createElement("div");
  p.id = "__mb_opt_wrap";
  p.innerHTML =
    '<button id="__mb_opt_btn">\u2699</button>' +
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
      })
      ["catch"](function () {
        return null;
      });
  }
  var btn = document.getElementById("__mb_opt_btn"),
    panel = document.getElementById("__mb_opt_panel");
  btn.onclick = function () {
    panel.classList.toggle("show");
    btn.classList.toggle("active");
    if (panel.classList.contains("show")) rs();
  };
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
  bg("balanced");
  window.__mbOptGUI = true;
})();
