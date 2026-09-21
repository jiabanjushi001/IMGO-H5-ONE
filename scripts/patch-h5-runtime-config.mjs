// 兼容旧 H5 产物；网站包不内置 config.js，一门 APK 包由 release-yimen-apk 单独复制。
import { createHash } from 'node:crypto'
import { lstatSync, readdirSync, readFileSync, readlinkSync, symlinkSync, writeFileSync } from 'node:fs'
import { relative, resolve } from 'node:path'

const projectRoot = resolve(import.meta.dirname, '..')
const sourceHTML = readFileSync(resolve(projectRoot, 'index.html'), 'utf8')
const begin = '<!-- IMGO_RUNTIME_CONFIG_BEGIN -->'
const end = '<!-- IMGO_RUNTIME_CONFIG_END -->'
const from = sourceHTML.indexOf(begin)
const to = sourceHTML.indexOf(end, from)
if (from < 0 || to < 0) throw Error('Runtime config loader is missing from source index.html')
const loader = sourceHTML.slice(from, to + end.length).trim()
const hash = value => createHash('sha256').update(value).digest('hex').slice(0, 12)

function findEntry(html) {
  return html.match(/src="\.?\/?assets\/(index-[^"/]+\.js)"/) ||
    html.match(/imgoStartApp\("\.?\/?assets\/(index-[^"/]+\.js)"\)/)
}

function toRelativeAssetPaths(html, entryName) {
  return html
    .replace(/(href|src)="\/assets\//g, '$1="./assets/')
    .replace(/imgoStartApp\("\/assets\//g, 'imgoStartApp("./assets/')
    .replace(new RegExp(`imgoStartApp\\("assets/${entryName}"\\)`), `imgoStartApp("./assets/${entryName}")`)
}

function rewriteAbsolutePublicPaths(content) {
  // 仅改写本地包资源路径。不要改 apiUrl+"/static/..."，
  // 否则会变成 apiUrl+". /static/..." → https://host./static/...
  return content
    .replace(/(?<!\+)(["'`])\/hybrid\//g, '$1./hybrid/')
    .replace(/(?<!\+)(["'`])\/assets\//g, '$1./assets/')
    .replace(/(?<!\+)(["'`])\/static\//g, '$1./static/')
    .replace(/(url\(\s*['"]?)\/hybrid\//g, '$1./hybrid/')
    .replace(/(url\(\s*['"]?)\/assets\//g, '$1./assets/')
    .replace(/(url\(\s*['"]?)\/static\//g, '$1./static/')
}

function patchBuild(directory, { relativeAssets = true } = {}) {
  const root = resolve(projectRoot, directory)
  const read = file => readFileSync(resolve(root, file), 'utf8')
  const write = (file, content) => writeFileSync(resolve(root, file), content)
  let html = read('index.html')
  const entry = findEntry(html)
  if (!entry) throw Error(`${directory}: H5 entry not found`)
  let main = read(`assets/${entry[1]}`)

  const anchor = 'const Wk=Uk.replace(/^http/,"ws")+"/wss"'
  const oldPatch = String.raw`const imgoRuntimeConfig=window.IMGO_CONFIG||{};const imgoRuntimeApi=String(imgoRuntimeConfig.apiUrl||"").trim();if(/^https?:\/\//i.test(imgoRuntimeApi))Uk=imgoRuntimeApi.replace(/\/+$/,'');const imgoRuntimeWs=String(imgoRuntimeConfig.wssUrl||"").trim();const Wk=/^wss?:\/\//i.test(imgoRuntimeWs)?imgoRuntimeWs:Uk.replace(/^http/,"ws")+"/wss"`
  const replacement = String.raw`const imgoRuntimeApi=String(window.httpUrl||"").trim();if(/^https?:\/\//i.test(imgoRuntimeApi))Uk=imgoRuntimeApi.replace(/\/+$/,'');const imgoRuntimeWs=String(window.wsUrl||"").trim();const Wk=/^wss?:\/\//i.test(imgoRuntimeWs)?imgoRuntimeWs:Uk.replace(/^http/,"ws")+"/wss"`
  if (main.includes(oldPatch)) {
    main = main.replace(oldPatch, replacement)
  } else if (!main.includes('window.httpUrl') || !main.includes('window.wsUrl')) {
    if (!main.includes(anchor)) throw Error(`${directory}: bundled API configuration changed`)
    main = main.replace(anchor, replacement)
  }

  // 旧版 H5 产物仍包含每 5 秒永久重复的重连定时器。API / WS 不通时，
  // 每次断线都会再创建一个 interval；保留一个一次性重试，避免计时器堆积。
  const legacyReconnect = 'reconnect(){console.info("检查是否手动断开，并重新连接"),clearInterval(this.heartbeatInterval),this.is_open_socket||2!=this.traderDetailIndex&&0!=this.accountStateIndex&&!this.followFlake||(console.info("5秒后重新连接..."),this.reconnectTimeOut=setInterval(()=>{this.connectSocketInit(this.data)},5e3))}'
  const safeReconnect = 'reconnect(){console.info("检查是否手动断开，并重新连接"),clearInterval(this.heartbeatInterval),this.heartbeatInterval=null,this.reconnectTimeOut&&clearTimeout(this.reconnectTimeOut),this.reconnectTimeOut=null,this.is_open_socket||2!=this.traderDetailIndex&&0!=this.accountStateIndex&&!this.followFlake||(console.info("5秒后重新连接..."),this.reconnectTimeOut=setTimeout(()=>{this.reconnectTimeOut=null,this.connectSocketInit(this.data)},5e3))}'
  if (main.includes(legacyReconnect)) main = main.replace(legacyReconnect, safeReconnect)
  const legacyConnect = 'connectSocketInit(e){this.data=e,this.socketTask=J_('
  const guardedConnect = 'connectSocketInit(e){if(this.socketTask&&[0,1].includes(this.socketTask.readyState))return this.socketTask;this.data=e,this.socketTask=J_('
  if (main.includes(legacyConnect)) main = main.replace(legacyConnect, guardedConnect)

  if (relativeAssets) main = rewriteAbsolutePublicPaths(main)

  const entryName = `index-imgo${hash(main)}.js`
  write(`assets/${entryName}`, main)
  html = html.replace(entry[1], entryName)
  const loaderStart = html.indexOf(begin)
  const loaderEnd = html.indexOf(end, loaderStart)
  if (loaderStart >= 0 && loaderEnd >= 0) {
    html = html.slice(0, loaderStart) + loader + html.slice(loaderEnd + end.length)
  } else {
    const moduleScript = /<script type="module"[^>]+><\/script>/
    if (!moduleScript.test(html)) throw Error(`${directory}: module script not found`)
    html = html.replace(moduleScript, `${loader}\n    $&`)
  }
  // 用探测通过后才注入的模块脚本代替静态入口，避免不可达 API 触发旧包的启动死循环。
  const staticEntry = /<script type="module"[^>]*src="\.?\/?assets\/index-[^"/]+\.js"[^>]*><\/script>/
  if (staticEntry.test(html)) {
    html = html.replace(staticEntry, '')
    const bootPath = relativeAssets ? `./assets/${entryName}` : `/assets/${entryName}`
    html = html.replace('</body>', `    <script>window.imgoStartApp("${bootPath}")</script>\n  </body>`)
  } else if (!html.includes(`imgoStartApp("./assets/${entryName}")`) && !html.includes(`imgoStartApp("/assets/${entryName}")`)) {
    throw Error(`${directory}: H5 bootstrap not found`)
  }
  if (relativeAssets) html = toRelativeAssetPaths(html, entryName)
  write('index.html', html)

  // CSS / 其它 JS 里也可能有 /static/ 绝对路径
  if (relativeAssets) {
    for (const name of readdirSync(resolve(root, 'assets'))) {
      if (!name.endsWith('.css') && !name.endsWith('.js')) continue
      if (name === entryName) continue
      const file = `assets/${name}`
      const raw = read(file)
      const next = rewriteAbsolutePublicPaths(raw)
      if (next !== raw) write(file, next)
    }
  }

  // 本地预览目录：把 config.js 链到项目根，方便改一处即可生效。
  if (
    root === resolve(projectRoot, 'dist/build/h5') ||
    root === resolve(projectRoot, 'unpackage/dist/build/h5')
  ) {
    const configPath = resolve(root, 'config.js')
    const sourcePath = resolve(projectRoot, 'config.js')
    const linkTarget = relative(root, sourcePath)
    let info
    try {
      info = lstatSync(configPath)
    } catch (error) {
      if (error.code !== 'ENOENT') throw error
    }
    if (!info) symlinkSync(linkTarget, configPath)
    else if (!info.isSymbolicLink() || readlinkSync(configPath) !== linkTarget) {
      throw Error(`${directory}: config.js must link to the project root config.js`)
    }
  }
  console.log(`${directory}: ${entryName}; relativeAssets=${relativeAssets}`)
}

const args = process.argv.slice(2).filter(a => a !== '--absolute')
const relativeAssets = !process.argv.includes('--absolute')
for (const directory of args.length
  ? args
  : ['dist/build/h5', 'unpackage/dist/build/h5', 'release/h5']) {
  patchBuild(directory, { relativeAssets })
}
