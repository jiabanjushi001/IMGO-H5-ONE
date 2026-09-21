/**
 * 把 Vite ES Module 产物打成单文件 IIFE，去掉 crossorigin，
 * 使 file:// / 一门 APP WebView 可离线加载（避免 CORS）。
 */
import { createHash } from 'node:crypto'
import { copyFileSync, readFileSync, writeFileSync, unlinkSync, existsSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { rollup } from 'rollup'
import * as esbuild from 'esbuild'

const projectRoot = resolve(fileURLToPath(import.meta.url), '../..')
const root = resolve(process.argv[2] || 'release/yimen-apk')
const htmlPath = resolve(root, 'index.html')
let html = readFileSync(htmlPath, 'utf8')

html = html.replace(/\s+crossorigin(?:=["'][^"']*["'])?/g, '')

const boot = html.match(/imgoStartApp\("(\.?\/?assets\/[^"]+\.js)"\)/)
if (!boot) throw new Error(`${root}: 未找到 imgoStartApp 入口`)

const entryRel = boot[1].replace(/^\.\//, '')
const entryAbs = resolve(root, entryRel)
if (!existsSync(entryAbs)) throw new Error(`入口不存在: ${entryAbs}`)

const assetsDir = dirname(entryAbs)
const quillSrc = resolve(projectRoot, 'node_modules/quill/dist/quill.min.js')
const quillCssSrc = resolve(projectRoot, 'node_modules/quill/dist/quill.snow.css')
if (!existsSync(quillSrc)) throw new Error('缺少 node_modules/quill/dist/quill.min.js')
copyFileSync(quillSrc, resolve(assetsDir, 'quill.min.js'))
if (existsSync(quillCssSrc)) copyFileSync(quillCssSrc, resolve(assetsDir, 'quill.snow.css'))

const tmpEsm = resolve(assetsDir, `index-inline-tmp.js`)
const bundle = await rollup({
	input: entryAbs,
	onwarn(warning, warn) {
		if (warning.code === 'THIS_IS_UNDEFINED') return
		if (warning.code === 'CIRCULAR_DEPENDENCY') return
		warn(warning)
	},
})
await bundle.write({
	file: tmpEsm,
	format: 'es',
	inlineDynamicImports: true,
	sourcemap: false,
})
await bundle.close()

const tmpOut = resolve(assetsDir, `index-classic-tmp.js`)
await esbuild.build({
	entryPoints: [tmpEsm],
	bundle: true,
	format: 'iife',
	platform: 'browser',
	target: ['es2018'],
	outfile: tmpOut,
	logLevel: 'warning',
	loader: {
		'.css': 'css',
		'.png': 'dataurl',
		'.jpg': 'dataurl',
		'.jpeg': 'dataurl',
		'.gif': 'dataurl',
		'.svg': 'dataurl',
		'.webp': 'dataurl',
		'.woff': 'dataurl',
		'.woff2': 'dataurl',
		'.ttf': 'dataurl',
		'.eot': 'dataurl',
	},
})
unlinkSync(tmpEsm)

let classic = readFileSync(tmpOut, 'utf8')
unlinkSync(tmpOut)

// 保留 UMD Quill（含 parchment），避免 ESM 动态 import 在 file:// 下被掏空导致聊天框无法输入。
if (!classic.startsWith('(() => {')) {
	throw new Error('意外的 IIFE 包裹格式，无法注入 chunk base')
}
classic = classic.replace(
	'(() => {',
	'(() => { var __IMGO_KEEP_QUILL__=window.__IMGO_QUILL_UMD__||window.Quill; var __IMGO_CHUNK_BASE__=(document.currentScript&&document.currentScript.src)||((window.__IMGO_BASE__||document.baseURI||"")+ "assets/");',
)
classic = classic.replace(/\bwindow\.Quill\s*=\s*/g, 'window.Quill=__IMGO_KEEP_QUILL__||')

// esbuild 会把 import.meta.url 清成空字符串；Vite preload 必须用入口脚本 URL 作基址。
classic = classic.replace(
	/return new URL\((\w+), (\w+)\)\.href;/g,
	'return new URL($1, $2 || __IMGO_CHUNK_BASE__).href;',
)

// Vite preload 会给分包 JS 打 modulepreload；file:// 下必 CORS。
// 页面 JS 已 inlineDynamicImports 进 IIFE，只需保留 CSS stylesheet 注入。
// 兼容两种产物：旧版 ternary modulepreload、新版 scriptRel 变量。
{
	const before = classic
	classic = classic.replace(
		/const (\w+) = (\w+)\.endsWith\("\.css"\),\s*(\w+) = \1 \? '\[rel="stylesheet"\]' : ""/g,
		'const $1 = $2.endsWith(".css"); if (!$1) return; const $3 = \'[rel="stylesheet"]\'',
	)
	const hits = (before.match(/\.endsWith\("\.css"\),/g) || []).length
	const afterHits = (classic.match(/if \(!\w+\) return; const \w+ = '\[rel="stylesheet"\]'/g) || []).length
	if (afterHits > 0) {
		console.log(`已跳过 ${afterHits} 处分包 JS modulepreload（仅加载 CSS）`)
	} else if (hits > 0) {
		console.warn('警告: 检测到 endsWith(.css) 但未能修补，file:// 下仍可能 CORS')
	} else {
		console.warn('警告: 未找到 Vite preload CSS 判定点')
	}
}

const leftover = [...classic.matchAll(/\bimport\s*\(\s*["'`]\.?\.?\//g)].length
if (leftover > 0) {
	console.warn(`警告: IIFE 中仍有 ${leftover} 处相对路径动态 import()，file:// 下懒加载可能失败`)
}
const leftoverJsPreload = [...classic.matchAll(/\.endsWith\("\.css"\),/g)].length
if (leftoverJsPreload > 0) {
	console.warn(`警告: 仍有 ${leftoverJsPreload} 处未改写的 CSS 判定，可能继续 preload JS`)
}

const digest = createHash('sha256').update(classic).digest('hex').slice(0, 12)
const classicName = `index-classic-${digest}.js`
const classicRel = `./assets/${classicName}`
writeFileSync(resolve(assetsDir, classicName), classic)

if (!html.includes('__IMGO_CLASSIC_SCRIPT__=true')) {
	html = html.replace(
		'window.__IMGO_BASE__',
		'window.__IMGO_CLASSIC_SCRIPT__=true;\n      window.__IMGO_BASE__',
	)
}
if (existsSync(resolve(assetsDir, 'quill.snow.css')) && !html.includes('quill.snow.css')) {
	html = html.replace(
		'</head>',
		'    <link rel="stylesheet" href="./assets/quill.snow.css">\n  </head>',
	)
}
html = html.replace(boot[0], `imgoStartApp("${classicRel}")`)
writeFileSync(htmlPath, html)

console.log(`${root}: classic IIFE -> ${classicName} (${classic.length} bytes); leftoverImport=${leftover}; leftoverJsPreload=${leftoverJsPreload}`)
