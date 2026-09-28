const ImgoSecurityPanel = {
  name: 'ImgoSecurityPanel',
  data() {
    return { loading: false, loaded: false, enabled: false, whitelistEnabled: false, whitelistText: '', whitelistDraft: '', currentIP: '' }
  },
  computed: {
    whitelistRules() {
      return [...new Set(String(this.whitelistText || '').split(/[,;\s]+/).map(value => value.trim()).filter(Boolean))]
    }
  },
  mounted() { this.load() },
  methods: {
    async load() {
      this.loading = true
      try {
        const response = await this.$api.configApi.getSecurity({})
        if (Number(response.code) === 0) {
          this.enabled = !!response.data.google_auth_enabled
          this.whitelistEnabled = !!response.data.ip_whitelist_enabled
          this.whitelistText = Array.isArray(response.data.ip_whitelist) ? response.data.ip_whitelist.join(', ') : ''
          this.currentIP = response.data.current_ip || ''
          this.loaded = true
        }
      } finally { this.loading = false }
    },
    async changeGoogle(value) {
      const action = value ? '开启' : '关闭'
      const message = value
        ? '开启后，所有未绑定谷歌验证的账号会立即退出登录；以后从后台登录必须输入各自绑定的谷歌验证码。超级管理员必须先完成绑定才能开启。'
        : '关闭后，后台登录将不再校验谷歌验证码，已绑定的个人密钥会保留。'
      try {
        await this.$confirm(message, `${action}谷歌验证`, { type: value ? 'warning' : 'info', confirmButtonText: `确认${action}`, cancelButtonText: '取消' })
        this.loading = true
        const response = await this.$api.configApi.setSecurity({ google_auth_enabled: value ? 1 : 0 })
        if (Number(response.code) === 0) {
          this.enabled = !!response.data.google_auth_enabled
          this.$message.success(`已${action}后台谷歌验证`)
        } else {
          this.enabled = !value
          this.$message.error(response.msg || `${action}失败`)
        }
      } catch (_) {
        this.enabled = !value
      } finally { this.loading = false }
    },
    async persistWhitelist(enabled) {
      this.loading = true
      try {
        const rules = this.mergeWhitelistDraft()
        const response = await this.$api.configApi.setSecurity({
          ip_whitelist_enabled: enabled ? 1 : 0,
          ip_whitelist: rules.join(', ')
        })
        if (Number(response.code) !== 0) {
          this.$message.error(response.msg || '保存失败')
          return false
        }
        this.whitelistEnabled = !!response.data.ip_whitelist_enabled
        this.whitelistText = Array.isArray(response.data.ip_whitelist) ? response.data.ip_whitelist.join(', ') : this.whitelistText
        this.currentIP = response.data.current_ip || this.currentIP
        this.$message.success('后台登录 IP 白名单已保存')
        return true
      } finally { this.loading = false }
    },
    async changeWhitelist(value) {
      try {
        const message = value
          ? `开启后，仅白名单 IP 可以登录后台和访问后台管理 API。当前 IP：${this.currentIP || '未知'}，必须包含在白名单内。H5 用户端不受影响。`
          : '关闭后，后台登录和后台管理 API 将不再限制来源 IP，已填写的白名单会保留。'
        await this.$confirm(message, `${value ? '开启' : '关闭'}后台 IP 白名单`, { type: value ? 'warning' : 'info', confirmButtonText: `确认${value ? '开启' : '关闭'}`, cancelButtonText: '取消' })
        if (!await this.persistWhitelist(value)) this.whitelistEnabled = !value
      } catch (_) {
        this.whitelistEnabled = !value
      }
    },
    async saveWhitelist() {
      await this.persistWhitelist(this.whitelistEnabled)
    },
    mergeWhitelistDraft() {
      const incoming = String(this.whitelistDraft || '').split(/[,;\s]+/).map(value => value.trim()).filter(Boolean)
      const rules = [...new Set([...this.whitelistRules, ...incoming])]
      this.whitelistText = rules.join(', ')
      this.whitelistDraft = ''
      return rules
    },
    addWhitelistRules() {
      if (!String(this.whitelistDraft || '').trim()) return
      this.mergeWhitelistDraft()
    },
    removeWhitelistRule(rule) {
      this.whitelistText = this.whitelistRules.filter(value => value !== rule).join(', ')
    }
  },
  render(h) {
    return h('div', { class: 'imgo-security-settings', directives: [{ name: 'loading', value: this.loading }] }, [
      h('el-alert', { props: { title: '后台登录安全设置', type: 'info', showIcon: true, closable: false } }),
      h('el-form', { props: { labelWidth: '180px' } }, [
        h('el-form-item', { props: { label: '后台谷歌验证' } }, [
          h('el-switch', { props: { value: this.enabled, disabled: !this.loaded || this.loading }, on: { change: this.changeGoogle } }),
          h('span', { class: 'imgo-security-state' }, [this.enabled ? '已开启' : '已关闭'])
        ]),
        h('el-form-item', { props: { label: '说明' } }, [
          h('div', { class: 'imgo-security-description' }, [
            h('p', ['开启后，所有后台账号登录时都必须输入各自绑定的谷歌验证码。']),
            h('p', ['谷歌验证可由超级管理员在“成员 → 更多 → 谷歌设置”中管理。'])
          ])
        ]),
        h('el-divider'),
        h('el-form-item', { props: { label: '后台登录 IP 白名单' } }, [
          h('el-switch', { props: { value: this.whitelistEnabled, disabled: !this.loaded || this.loading }, on: { change: this.changeWhitelist } }),
          h('span', { class: 'imgo-security-state' }, [this.whitelistEnabled ? '已开启' : '已关闭'])
        ]),
        h('el-form-item', { props: { label: '当前访问 IP' } }, [h('span', [this.currentIP || '读取中...'])]),
        h('el-form-item', { props: { label: '允许的 IP / CIDR' } }, [
          h('div', { class: 'imgo-security-ip-editor' }, [
            this.whitelistRules.length
              ? h('div', { class: 'imgo-security-ip-list' }, this.whitelistRules.map(rule => h('div', { class: 'imgo-security-ip-row', key: rule }, [
                h('span', [rule]),
                h('button', { class: 'imgo-security-ip-remove', attrs: { type: 'button', title: `删除 ${rule}`, 'aria-label': `删除 ${rule}` }, on: { click: () => this.removeWhitelistRule(rule) } }, ['×'])
              ])))
              : h('div', { class: 'imgo-security-ip-empty' }, ['暂未添加 IP 白名单']),
            h('div', { class: 'imgo-security-ip-add' }, [
              h('el-input', {
                props: { value: this.whitelistDraft, disabled: this.loading, placeholder: '输入 IP，多个请用英文逗号分隔' },
                on: { input: value => { this.whitelistDraft = value } },
                nativeOn: { keyup: event => { if (event.key === 'Enter') this.addWhitelistRules() } }
              }),
              h('el-button', { props: { disabled: this.loading || !String(this.whitelistDraft || '').trim() }, on: { click: this.addWhitelistRules } }, ['添加'])
            ])
          ]),
          h('div', { class: 'imgo-security-description' }, [
            h('p', ['多个 IP 使用英文逗号（,）分隔，一行可以填写多个。']),
            h('p', ['开启时白名单必须包含当前访问 IP，最多 200 条。'])
          ]),
          h('el-button', { class: 'imgo-security-save', props: { type: 'primary', loading: this.loading }, on: { click: this.saveWhitelist } }, ['保存白名单'])
        ])
      ])
    ])
  }
}
