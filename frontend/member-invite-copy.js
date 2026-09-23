// Small Vue 2 cell component: display one invite code and copy it on demand.
const ImgoInviteCodeCopy = {
  name: 'ImgoInviteCodeCopy',
  props: {
    code: { type: [String, Number], default: '' }
  },
  computed: {
    normalizedCode() { return String(this.code || '').trim() }
  },
  methods: {
    copyWithLegacyAPI(text) {
      const input = document.createElement('textarea')
      input.value = text
      input.setAttribute('readonly', '')
      input.style.position = 'fixed'
      input.style.left = '-9999px'
      input.style.opacity = '0'
      document.body.appendChild(input)
      input.select()
      const copied = document.execCommand('copy')
      document.body.removeChild(input)
      if (!copied) throw Error('copy command failed')
    },
    async copy() {
      const text = this.normalizedCode
      if (!text) {
        this.$message.warning('暂无邀请码')
        return
      }
      try {
        if (navigator.clipboard && window.isSecureContext) {
          await navigator.clipboard.writeText(text)
        } else {
          this.copyWithLegacyAPI(text)
        }
        this.$message.success('邀请码已复制')
      } catch (_) {
        this.$message.error('复制失败，请手动复制')
      }
    }
  },
  render(h) {
    const code = this.normalizedCode
    const children = [h('span', { class: 'imgo-invite-code-value' }, code || '—')]
    if (code) {
      children.push(h('el-button', {
        props: { type: 'text', size: 'mini' },
        attrs: { type: 'button', title: '复制邀请码', 'aria-label': `复制邀请码 ${code}` },
        on: { click: this.copy }
      }, '复制'))
    }
    return h('div', { class: 'imgo-invite-code-copy' }, children)
  }
}
