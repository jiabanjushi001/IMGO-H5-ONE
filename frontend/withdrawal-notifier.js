function imgoInstallWithdrawalNotifier(store) {
  if (!store || typeof store.watch !== 'function' || store.__imgoWithdrawalNotifierInstalled) return
  store.__imgoWithdrawalNotifierInstalled = true
  const delivered = new Set()
  const playFallback = () => {
    try {
      const audio = new window.Audio('/assets/media/notify.3d49176d.mp3')
      const result = audio.play()
      if (result && typeof result.catch === 'function') result.catch(() => {})
    } catch (_) {}
  }
  store.watch(
    state => state.socketAction,
    event => {
      if (!event || !['withdrawalSubmitted', 'paymentMethodSubmitted'].includes(event.type)) return
      const data = event.data && typeof event.data === 'object' ? event.data : {}
      const key = event.type + ':' + String(data.withdrawal_id || [data.user_id, data.payment_type, data.amount_cents, event.time].join('-'))
      if (delivered.has(key)) return
      delivered.add(key)
      if (delivered.size > 100) delivered.delete(delivered.values().next().value)

      const account = String(data.account || data.realname || '团队成员').trim()
      const amount = (Number(data.amount_cents || 0) / 100).toFixed(2)
      const message = event.type === 'paymentMethodSubmitted'
        ? `收到新的${String(data.payment_label || '收款码')}审核申请，账号${account}，请及时审核。`
        : `收到新的提现申请，账号${account}，金额${amount}元，请及时处理。`
      try {
        const notify = store._vm && store._vm.$message
        if (typeof notify === 'function') notify({ message, type: 'warning', duration: 8000, showClose: true })
      } catch (_) {}
      try {
        if (window.speechSynthesis && window.SpeechSynthesisUtterance) {
          const speech = new window.SpeechSynthesisUtterance(message)
          speech.lang = 'zh-CN'
          speech.rate = 0.95
          speech.onerror = playFallback
          window.speechSynthesis.speak(speech)
        } else playFallback()
      } catch (_) {
        playFallback()
      }
    }
  )
}
