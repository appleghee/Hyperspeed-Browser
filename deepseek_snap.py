import sys, json, urllib.request, os, tempfile

port_file = os.path.join(tempfile.gettempdir(), "hyperspeed-browser.port")
port = open(port_file).read().strip() if os.path.exists(port_file) else "52954"
base = f"http://127.0.0.1:{port}"

def api(method, path, data=None):
    url = f"{base}{path}"
    if data is not None:
        req = urllib.request.Request(url, data=json.dumps(data).encode(), headers={"Content-Type": "application/json"}, method=method)
    else:
        req = urllib.request.Request(url, method=method)
    with urllib.request.urlopen(req) as r:
        return json.loads(r.read())

def show_login_state():
    snap = api("GET", "/api/snapshot")
    if not snap.get("ok"):
        print("Error:", snap.get("error"))
        return

    nodes = snap["result"]
    print(f"Total nodes: {len(nodes)}")
    
    # Check current URL
    r = api("POST", "/api/eval", {"js": "window.location.href"})
    url = json.loads(r["result"]) if r.get("ok") else "?"
    print(f"URL: {url}")
    
    # Debug: check if there's a form element and what events it listens to
    r = api("POST", "/api/eval", {"js": "JSON.stringify({hasForm:document.querySelector('form')!==null, forms:document.querySelectorAll('form').length, btnTag:document.querySelector('[data-si=\"s_56\"]').tagName, btnClasses:document.querySelector('[data-si=\"s_56\"]').className, parentTag:document.querySelector('[data-si=\"s_56\"]').parentElement.tagName})"})
    if r.get("ok"):
        try:
            info = json.loads(r["result"])
            print(f"FORM INFO: {info}")
        except:
            print(f"FORM INFO RAW: {r['result']}")
    
    # Search for any form in the page
    r = api("POST", "/api/eval", {"js": "JSON.stringify(Array.from(document.querySelectorAll('form')).map(f => ({id:f.id, action:f.action, inputs:Array.from(f.querySelectorAll('input,button')).length})))"})
    if r.get("ok"):
        try:
            infos = json.loads(r["result"])
            for info in infos if isinstance(infos, list) else [infos]:
                print(f"FORM: {info}")
        except:
            print(f"FORMS RAW: {r['result'][:200]}")
    
    # Show form elements
    for n in nodes:
        tag = n.get("tag","").lower()
        txt = str(n.get("text",""))
        role = n.get("role","").lower()
        typ = n.get("type","")
        uid = n["uid"]
        placeholder = n.get("placeholder","")
        
        # Inputs
        if tag == "input":
            print(f"INPUT    {uid:>6} | type={typ:<12} | placeholder={placeholder:<35} | value={n.get('value','')[:50]}")
        
        # Buttons (including role=button divs)
        if "button" in role or tag == "button":
            print(f"BUTTON   {uid:>6} | tag={tag:<10} | {txt[:80]}")
        
        # Error/success messages
        if txt.strip() and any(w in txt.lower() for w in ["error","incorrect","invalid","wrong","failed","success","welcome","verified"]):
            print(f"MSG      {uid:>6} | {tag:<10} | {txt[:100]}")
        
        # Links that matter
        if tag == "a" and "sign" in txt.lower():
            print(f"LINK     {uid:>6} | {txt[:80]}")

if __name__ == "__main__":
    show_login_state()
