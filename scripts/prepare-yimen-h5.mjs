import { cpSync, existsSync, lstatSync, mkdirSync, readFileSync, readdirSync, renameSync, rmSync, statSync, writeFileSync } from 'node:fs'
import { basename, dirname, isAbsolute, join, relative, resolve, sep } from 'node:path'
import { randomUUID } from 'node:crypto'

const projectRoot = resolve(import.meta.dirname, '..')
const source = resolve(process.argv[2] || join(projectRoot, 'unpackage/dist/build/h5'))
const destination = resolve(process.argv[3] || join(projectRoot, 'release/yimen-h5'))
const config = resolve(process.argv[4] || join(projectRoot, 'config.yimen.js'))
const sdk = join(projectRoot, 'vendor/yimen/jsbridge-mini.js')

function inside(parent, child) {
  const subpath = relative(parent, child)
  return !subpath || (!subpath.startsWith(`..${sep}`) && subpath !== '..' && !isAbsolute(subpath))
}

function prepare() {
  if (!existsSync(join(source, 'index.html'))) throw new Error(`H5 编译产物不存在：${source}`)
  if (!existsSync(config) || !existsSync(sdk)) throw new Error('缺少一门版 config.yimen.js 或官方 jsbridge-mini.js')
  if (existsSync(destination)) throw new Error(`输出目录已存在，请指定新目录，避免覆盖：${destination}`)
  if (inside(source, destination) || inside(destination, source)) throw new Error('输入和输出目录不能互相包含')
  const sourceFiles = [
    'index.html', 'manifest.json', 'common/scan.js', 'common/yimenBridge.js',
    'pages/index/scan.vue', 'pages/index/index.vue', 'pages/index/qrcode.vue',
    'pages/mine/invite.vue', 'pages/login/index.vue', 'pages/login/register.vue',
    'components/Empty.vue', 'components/invite/InviteShareActions.vue',
    'utils/inviteShareUrl.js'
  ]
  const buildTime = statSync(join(source, 'index.html')).mtimeMs
  if (sourceFiles.some(file => statSync(join(projectRoot, file)).mtimeMs > buildTime + 1000)) {
    throw new Error('H5 编译产物早于本分支源码修改；请先在 HBuilderX 重新发行 H5')
  }
  if (!existsSync(join(source, 'assets')) || !existsSync(join(source, 'static'))) {
    throw new Error('H5 编译产物缺少 assets 或 static 目录')
  }

  const configText = readFileSync(config, 'utf8')
  if (!configText.includes('window.imgoPackagedApp = true') ||
      !/window\.(apiServers|httpUrl)\s*=/.test(configText)) {
    throw new Error('一门配置必须设置 imgoPackagedApp 和 API 地址')
  }

  let html = readFileSync(join(source, 'index.html'), 'utf8')
  if (!html.includes('IMGO_RUNTIME_CONFIG_BEGIN') || !html.includes('window.imgoStartApp =')) {
    throw new Error('编译产物缺少运行时 config.js 启动门，不能准备一门版')
  }
  html = html.replace(/(["'(])\/assets\//g, '$1./assets/')
  html = html.replace(/(["'(])\/static\//g, '$1./static/')
  const moduleEntry = /<script\b(?=[^>]*\btype=["']module["'])(?=[^>]*\bsrc=["'](\.\/assets\/[^"']+\.js)["'])[^>]*><\/script>/
  const staticMatch = html.match(moduleEntry)
  if (staticMatch) {
    html = html.replace(moduleEntry, `<script>window.imgoStartApp(${JSON.stringify(staticMatch[1])})</script>`)
  }
  if (!/window\.imgoStartApp\(["']\.\/assets\/[^"']+\.js["']\)/.test(html)) {
    throw new Error('无法找到 H5 主程序入口；请先重新编译 H5')
  }
  html = html.replace('<!-- IMGO_RUNTIME_CONFIG_BEGIN -->',
    '<script src="./jsbridge-mini.js"></script>\n    <!-- IMGO_RUNTIME_CONFIG_BEGIN -->')
  if (/(["'(])\/(?:assets|static)\//.test(html)) throw new Error('index.html 仍有站点根路径资源')

  // Stage first so a failed preparation never leaves a partially usable output.
  mkdirSync(dirname(destination), { recursive: true })
  const staging = join(dirname(destination), `.imgo-yimen-${randomUUID()}`)
  try {
    // Never copy the development config.js symlink: the APK needs a real file.
    cpSync(source, staging, {
      recursive: true,
      filter(path) {
        if (path === join(source, 'config.js')) return false
        if (lstatSync(path).isSymbolicLink()) throw new Error(`静态资源含符号链接：${path}`)
        return true
      }
    })

    const assets = join(staging, 'assets')
    for (const name of readdirSync(assets)) {
      const file = join(assets, name)
      if (!lstatSync(file).isFile()) continue
      if (name.endsWith('.css')) {
        const css = readFileSync(file, 'utf8')
          .replace(/url\((['"]?)\/assets\//g, 'url($1./')
          .replace(/url\((['"]?)\/static\//g, 'url($1../static/')
        writeFileSync(file, css)
      } else if (name.endsWith('.js')) {
        // uni-app generates root-relative tabBar icon paths from pages.json.
        const js = readFileSync(file, 'utf8')
          .replace(/(["'])\/static\/image\//g, '$1./static/image/')
        writeFileSync(file, js)
      }
    }
    writeFileSync(join(staging, 'index.html'), html)
    writeFileSync(join(staging, 'config.js'), configText)
    cpSync(sdk, join(staging, basename(sdk)))
    if (existsSync(destination)) throw new Error(`输出目录已存在，请指定新目录，避免覆盖：${destination}`)
    renameSync(staging, destination)
  } finally {
    if (existsSync(staging)) rmSync(staging, { recursive: true, force: true })
  }
  console.log(`一门 App 静态目录已准备：${destination}`)
  console.log('此步骤只准备 HTML/JS/CSS/图片，没有生成 APK；请用一门 App 平台上传该目录的全部内容。')
}

prepare()
