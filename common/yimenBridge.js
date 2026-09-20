// Only the separately prepared Yimen bundle sets this marker and loads its SDK.
export function yimenBridge() {
	if (typeof window === 'undefined' || window.imgoPackagedApp !== true) return null
	const bridge = window.jsBridge
	return bridge?.inApp && typeof bridge.ready === 'function' && typeof bridge.scan === 'function'
		? bridge : null
}

export function startYimenScan(onResult) {
	const bridge = yimenBridge()
	if (!bridge) return false
	bridge.ready(() => {
		// needResult prevents the container opening the QR URL outside our group flow.
		bridge.scan({ needResult: true }, code => {
			if (typeof code === 'string' && code.trim()) onResult(code.trim())
		})
	})
	return true
}
