// 将 CLI 产物打补丁并同步到 release/h5（不复制 config.js）
import { cpSync, existsSync, mkdirSync, rmSync } from 'node:fs'
import { resolve } from 'node:path'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const projectRoot = resolve(fileURLToPath(import.meta.url), '../..')
const distH5 = resolve(projectRoot, 'dist/build/h5')
const releaseH5 = resolve(projectRoot, 'release/h5')

if (!existsSync(resolve(distH5, 'index.html'))) {
	throw new Error('未找到 dist/build/h5，请先执行 npm run build:h5')
}

const patch = spawnSync(
	process.execPath,
	[resolve(projectRoot, 'scripts/patch-h5-runtime-config.mjs'), 'dist/build/h5'],
	{ cwd: projectRoot, stdio: 'inherit' },
)
if (patch.status !== 0) process.exit(patch.status ?? 1)

rmSync(releaseH5, { recursive: true, force: true })
mkdirSync(releaseH5, { recursive: true })
cpSync(distH5, releaseH5, {
	recursive: true,
	filter: (src) => {
		const name = src.replace(/\\/g, '/').split('/').pop()
		return name !== 'config.js'
	},
})

console.log('release/h5 已更新（未包含 config.js，请在服务器单独放置）')
