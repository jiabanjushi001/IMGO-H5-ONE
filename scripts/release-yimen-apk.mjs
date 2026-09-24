/**
 * 一门 APP 静态离线包：
 * - html / js / css / img / hybrid 全部相对路径
 * - 内置 config.js（APK 内无法像网站那样单独挂配置时也能启动）
 * - 输出 release/yimen-apk/ 与 release/yimen-apk.zip，可直接上传一门「网页打包 / HTML 离线」
 */
import { cpSync, existsSync, mkdirSync, readFileSync, rmSync, writeFileSync, copyFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const projectRoot = resolve(fileURLToPath(import.meta.url), '../..')
const distH5 = resolve(projectRoot, 'dist/build/h5')
const outDir = resolve(projectRoot, 'release/yimen-apk')
const outZip = resolve(projectRoot, 'release/yimen-apk.zip')
const configSource = resolve(projectRoot, 'config.js')
const faviconSource = resolve(projectRoot, 'favicon.ico')
const readmePath = resolve(outDir, 'README-一门打包说明.txt')

if (!existsSync(resolve(distH5, 'index.html'))) {
	throw new Error('未找到 dist/build/h5，请先执行 npm run build:h5 或 npm run release:apk')
}

const patch = spawnSync(
	process.execPath,
	[resolve(projectRoot, 'scripts/patch-h5-runtime-config.mjs'), 'dist/build/h5'],
	{ cwd: projectRoot, stdio: 'inherit' },
)
if (patch.status !== 0) process.exit(patch.status ?? 1)

rmSync(outDir, { recursive: true, force: true })
mkdirSync(outDir, { recursive: true })
cpSync(distH5, outDir, {
	recursive: true,
	filter: (src) => {
		const name = src.replace(/\\/g, '/').split('/').pop()
		// 去掉指向源码的软链，后面改为真实文件
		return name !== 'config.js' && name !== 'favicon.ico'
	},
})

if (!existsSync(configSource)) {
	throw new Error('缺少项目根目录 config.js，一门 APK 包需要内置服务器地址')
}
writeFileSync(resolve(outDir, 'config.js'), readFileSync(configSource))
if (!existsSync(faviconSource)) {
	throw new Error('缺少项目根目录 favicon.ico，登录页 Logo 依赖该文件')
}
copyFileSync(faviconSource, resolve(outDir, 'favicon.ico'))

const classic = spawnSync(
	process.execPath,
	[resolve(projectRoot, 'scripts/bundle-apk-classic.mjs'), outDir],
	{ cwd: projectRoot, stdio: 'inherit' },
)
if (classic.status !== 0) process.exit(classic.status ?? 1)

writeFileSync(readmePath, `一门 APP 静态离线打包说明
========================

1. 本目录（或同级 yimen-apk.zip）即为可上传的静态资源包。
2. 在一门开发者中心选择「网页打包 / HTML 离线 / 混合打包」模式。
3. 首页文件选择：index.html
4. 上传本 zip 或把本目录全部文件上传到项目 www 根目录。
5. 包内已含 config.js（API / WebSocket）。若要改服务器，编辑本目录 config.js 后重新打 zip，或在一门后台替换该文件。
6. 资源均为相对路径（./assets ./static ./hybrid），兼容 file:// 与 fs://www/。
7. 入口 JS 已打成经典 IIFE（非 ES Module），避免 file:// CORS。
8. 路由为 hash 模式，无需服务端 rewrite。

注意：推送、保活、原生文件选择等 App 原生插件在一门 WebView 壳中可能不可用；音视频通话依赖 hybrid 页，已打入本包。
`)

rmSync(outZip, { force: true })
const zip = spawnSync(
	'powershell.exe',
	[
		'-NoProfile',
		'-Command',
		`Compress-Archive -Path '${outDir}\\*' -DestinationPath '${outZip}' -Force`,
	],
	{ cwd: projectRoot, stdio: 'inherit' },
)
if (zip.status !== 0) {
	console.warn('zip 压缩失败，仍可手动压缩 release/yimen-apk 目录后上传一门')
} else {
	console.log('已生成', outZip)
}

console.log('一门静态包目录:', outDir)
