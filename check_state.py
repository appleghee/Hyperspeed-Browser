import sys, json, urllib.request, urllib.parse, os, tempfile

sys.stdout.reconfigure(encoding='utf-8')

port_file = os.path.join(tempfile.gettempdir(), "hyperspeed-browser.port")
if not os.path.exists(port_file):
    print("Lỗi: browser chưa chạy (không tìm thấy", port_file, ")")
    sys.exit(1)
with open(port_file) as f:
    lines = f.read().strip().splitlines()
    port = lines[0].strip()
    token = lines[1].strip() if len(lines) > 1 else ""
base = f"http://127.0.0.1:{port}"

def deep_parse(s):
    """Parse potentially double-encoded JSON strings."""
    d = json.loads(s)
    while isinstance(d, str):
        try:
            d = json.loads(d)
        except json.JSONDecodeError:
            break
    return d

def api(method, path, data=None):
    url = f"{base}{path}"
    headers = {"X-API-Token": token}
    if data is not None:
        headers["Content-Type"] = "application/json"
        req = urllib.request.Request(url, data=json.dumps(data).encode(), headers=headers, method=method)
    else:
        req = urllib.request.Request(url, method=method)
        for k, v in headers.items():
            req.add_header(k, v)
    with urllib.request.urlopen(req) as r:
        return json.loads(r.read())

# 1. Snapshot - find filled input
print("=== Snapshot - filled input ===")
snap = api("GET", "/api/snapshot")
if snap.get("ok"):
    nodes = snap["result"]
    for n in nodes:
        t = str(n.get("text", ""))
        if "Hello from AI agent" in t:
            print(f"FOUND: uid={n['uid']} text={repr(t)}")
            break
        if n.get("uid") == "s_2359":
            print(f"Input s_2359: {n}")
            break
    else:
        print("Input not found")

# 2. Clickable elements
print("\n=== Clickable elements ===")
for n in nodes:
    tag = n.get("tag", "").lower()
    role = n.get("role", "").lower()
    txt = str(n.get("text", ""))[:80]
    if tag in ("button", "a", "input") or "button" in role:
        print(f"{n['uid']:>10} | {n['tag']:<10} | {txt}")

# 3. Check localStorage for popup blocker config
print("\n=== Popup blocker config ===")
r = api("POST", "/api/eval", {"js": "JSON.stringify({ls:Object.keys(localStorage),ss:Object.keys(sessionStorage)})"})
if r.get("ok"):
    d = deep_parse(r["result"])
    print(f"localStorage: {d['ls']}")
    print(f"sessionStorage: {d['ss']}")
else:
    print(f"Error: {r}")

# 4. Runtime info
print("\n=== Runtime ===")
r = api("GET", "/api/runtime")
print(r)

# 5. Cookies
print("\n=== Cookies ===")
r = api("GET", "/api/cookies")
if r.get("ok"):
    for c in r["result"]:
        print(f"  {c['key']} = {c.get('value','')[:50]}")

# 6. Popup blocker config check
print("\n=== Popup Blocker Config ===")
r = api("POST", "/api/eval", {"js": "JSON.stringify({blocked:localStorage.getItem('blockedWindows')||'none',browsers:localStorage.getItem('knownBrowsers')||'none',warnCount:localStorage.getItem('warnCount')||'0'})"})
if r.get("ok"):
    d = deep_parse(r["result"])
    for k,v in d.items():
        print(f"  {k} = {v}")

print("\n=== Done ===")
