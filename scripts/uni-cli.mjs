// 兼容 HBuilderX 根目录工程：将 UNI_INPUT_DIR 指向项目根而非默认 src/
import { spawnSync } from 'node:child_process'
import { resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const projectRoot = resolve(fileURLToPath(import.meta.url), '../..')
const args = process.argv.slice(2)
const uniBin = resolve(projectRoot, 'node_modules/@dcloudio/vite-plugin-uni/bin/uni.js')

const result = spawnSync(process.execPath, [uniBin, ...args], {
	cwd: projectRoot,
	stdio: 'inherit',
	env: {
		...process.env,
		UNI_INPUT_DIR: projectRoot,
		VITE_ROOT_DIR: projectRoot,
	},
})

process.exit(result.status ?? 1)
