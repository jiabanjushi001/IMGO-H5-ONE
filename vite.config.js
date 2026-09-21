import { defineConfig } from 'vite'
import uni from '@dcloudio/vite-plugin-uni'
import { resolve } from 'node:path'

const vueDemiV3 = resolve(__dirname, 'node_modules/vue-demi/lib/v3/index.mjs')

export default defineConfig({
	plugins: [
		{
			name: 'force-vue-demi-v3',
			enforce: 'pre',
			resolveId(id) {
				if (id === 'vue-demi' || id.includes('vue-demi/lib/index')) {
					return vueDemiV3
				}
			},
		},
		uni(),
	],
})
