const ImgoMemberGoogleAuthDialog = {
  name: 'ImgoMemberGoogleAuthDialog',
  data() {
    return { visible: false, loading: false, row: {}, detail: {}, code: '' }
  },
  methods: {
    async open(row) {
      this.row = row || {}
      this.detail = {}
      this.code = ''
      this.visible = true
      await this.load(false)
    },
    async load(reset) {
      this.loading = true
      try {
        const response = await this.$api.userApi.googleAuthDetail({ user_id: Number(this.row.user_id), reset: reset ? 1 : 0 })
        if (Number(response.code) === 0) {
          this.detail = response.data || {}
          this.code = ''
        }
      } finally { this.loading = false }
    },
    async bind() {
      if (!/^\d{6}$/.test(String(this.code || ''))) return this.$message.error('请输入 Google Authenticator 中的 6 位验证码')
      this.loading = true
      try {
        const response = await this.$api.userApi.googleAuthBind({ user_id: Number(this.row.user_id), setup_token: this.detail.setup_token, code: this.code })
        if (Number(response.code) === 0) {
          this.detail = { ...this.detail, bound: true, setup: false, secret: '', qr_data: '' }
          this.code = ''
          this.$message.success('谷歌验证绑定成功')
        }
      } finally { this.loading = false }
    },
    async unbind() {
      try {
        await this.$confirm('解除后，该成员在重新绑定前无法通过已开启的后台谷歌验证。', '解除谷歌验证', { type: 'warning', confirmButtonText: '确认解除', cancelButtonText: '取消' })
        this.loading = true
        const response = await this.$api.userApi.googleAuthUnbind({ user_id: Number(this.row.user_id) })
        if (Number(response.code) === 0) {
          this.$message.success('已解除谷歌验证')
          await this.load(false)
        }
      } catch (_) {
      } finally { this.loading = false }
    },
    copySecret() {
      this.$clipboard(this.detail.secret || '')
      this.$message.success('密钥已复制')
    }
  },
  render(h) {
    const setup = !!this.detail.setup
    return h('el-dialog', {
      props: { title: `${this.row.account || this.row.realname || '成员'} · 谷歌设置`, visible: this.visible, width: '520px', appendToBody: true, closeOnClickModal: false },
      on: { 'update:visible': value => { this.visible = value } }
    }, [
      h('div', { class: 'imgo-google-auth-dialog', directives: [{ name: 'loading', value: this.loading }] }, [
        h('el-alert', { props: { type: this.detail.bound && !setup ? 'success' : 'warning', closable: false, title: this.detail.bound && !setup ? '该成员已绑定独立谷歌验证' : '请让该成员使用 Google Authenticator 扫描二维码' } }),
        setup ? h('div', { class: 'imgo-google-auth-setup' }, [
          h('img', { class: 'imgo-google-auth-qr', attrs: { src: this.detail.qr_data, alt: 'Google Authenticator 二维码' } }),
          h('div', { class: 'imgo-google-auth-fields' }, [
            h('p', ['账号：', h('strong', [this.detail.account || this.row.account || '—'])]),
            h('p', ['密钥：']),
            h('div', { class: 'imgo-google-auth-secret' }, [
              h('code', [this.detail.secret || '']),
              h('el-button', { props: { type: 'text' }, on: { click: this.copySecret } }, ['复制'])
            ]),
            h('el-input', { props: { value: this.code, maxlength: 6, placeholder: '输入 6 位验证码完成绑定' }, on: { input: value => { this.code = String(value || '').replace(/\D/g, '').slice(0, 6) }, keyup: event => { if (event.key === 'Enter') this.bind() } } }),
            h('div', { class: 'imgo-google-auth-expire' }, ['二维码和密钥 10 分钟内有效'])
          ])
        ]) : null
      ]),
      h('span', { slot: 'footer' }, setup ? [
        this.detail.bound ? h('el-button', { on: { click: () => this.load(false) } }, ['取消重新绑定']) : null,
        h('el-button', { props: { type: 'primary', loading: this.loading }, on: { click: this.bind } }, ['验证并绑定'])
      ] : [
        h('el-button', { props: { type: 'danger', plain: true }, on: { click: this.unbind } }, ['解除绑定']),
        h('el-button', { props: { type: 'primary' }, on: { click: () => this.load(true) } }, ['重新绑定'])
      ])
    ])
  }
}
