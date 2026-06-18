import sys, json

def main():
    data = json.load(sys.stdin)
    result = data.get('result')
    if result is None:
        print('Error:', data.get('error'))
        return
    if isinstance(result, dict) and 'error' in result:
        print('JS Error:', result['error'])
        return
    if not isinstance(result, list):
        print('Unexpected:', str(result)[:200])
        return
    
    print(f'Total nodes: {len(result)}')
    for n in result:
        uid = n.get('uid', '')
        tag = n.get('tag', '')
        a = n.get('a', False)
        text = n.get('text', '')[:60]
        role = n.get('role', '')
        href = n.get('href', '')[:60]
        placeholder = n.get('placeholder', '')
        typ = n.get('type', '')
        val = n.get('value', '')[:30]
        
        if a or role or tag in ['h1','h2','h3','p','a','button','input','textarea']:
            extra = ''
            if placeholder: extra += f' placeholder={placeholder}'
            if href: extra += f' href={href}'
            if typ: extra += f' type={typ}'
            if val: extra += f' value={val}'
            print(f'  {uid} {tag} a={a} role={role} text={text}{extra}')

if __name__ == '__main__':
    main()
