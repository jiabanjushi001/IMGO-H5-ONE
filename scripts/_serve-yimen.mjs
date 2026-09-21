import { createServer } from 'node:http'
import { readFileSync, existsSync, statSync } from 'node:fs'
import { extname, join, normalize, resolve } from 'node:path'
const root = resolve('release/yimen-apk')
const port = 8787
const mime = { '.html':'text/html;charset=utf-8','.js':'text/javascript;charset=utf-8','.css':'text/css;charset=utf-8','.png':'image/png','.jpg':'image/jpeg','.svg':'image/svg+xml','.json':'application/json','.woff':'font/woff','.woff2':'font/woff2','.ico':'image/x-icon' }
createServer((req,res)=>{
  try {
    const u = new URL(req.url,'http://127.0.0.1')
    let p = decodeURIComponent(u.pathname)
    if (p.endsWith('/')) p += 'index.html'
    const file = normalize(join(root, p.replace(/^\//,'')))
    if (!file.startsWith(root) || !existsSync(file) || statSync(file).isDirectory()) { res.writeHead(404); res.end('404'); return }
    res.writeHead(200, {'Content-Type': mime[extname(file)]||'application/octet-stream', 'Cache-Control':'no-store'})
    res.end(readFileSync(file))
  } catch(e){ res.writeHead(500); res.end(String(e)) }
}).listen(port, '127.0.0.1', ()=>console.log('serving http://127.0.0.1:'+port))
