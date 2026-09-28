const ImgoSystemAlertPanel = {
  name: 'ImgoSystemAlertPanel',
  data() {
    return {
      loading: false,
      saving: false,
      testing: false,
      loaded: false,
      clearBotToken: false,
      form: { enabled: false, bot_token: '', bot_token_set: false, chat_id: '', events: [] },
      options: []
    }
  },
  mounted() { this.load() },
  methods: {
    apply(data) {
      const value = data || {}
      this.form = {
        enabled: !!value.enabled,
        bot_token: value.bot_token || '',
        bot_token_set: !!value.bot_token_set,
        chat_id: value.chat_id || '',
        events: Array.isArray(value.events) ? value.events.slice() : []
      }
      this.options = Array.isArray(value.options) ? value.options.slice() : []
      this.clearBotToken = false
      this.loaded = true
    },
    async load() {
      this.loading = true
      try {
        const response = await this.$api.configApi.getSystemAlert({})
        if (Number(response.code) === 0) this.apply(response.data)
        else this.$message.error(response.msg || '读取系统报警设置失败')
      } finally { this.loading = false }
    },
    changeToken(value) {
      this.form.bot_token = value
      if (String(value || '').trim()) this.clearBotToken = false
    },
    clearToken() {
      this.form.bot_token = ''
      this.form.bot_token_set = false
      this.clearBotToken = true
    },
    async save() {
      if (this.form.enabled && !String(this.form.chat_id || '').trim()) {
        this.$message.warning('请填写 Telegram 群 Chat ID')
        return
      }
      if (this.form.enabled && !this.form.bot_token_set && !String(this.form.bot_token || '').trim()) {
        this.$message.warning('请填写 Telegram Bot Token')
        return
      }
      if (this.form.enabled && !this.form.events.length) {
        this.$message.warning('至少勾选一项报警内容')
        return
      }
      const payload = {
        enabled: this.form.enabled ? 1 : 0,
        chat_id: String(this.form.chat_id || '').trim(),
        events: this.form.events.slice(),
        clear_bot_token: this.clearBotToken ? 1 : 0
      }
      if (String(this.form.bot_token || '').trim()) payload.bot_token = String(this.form.bot_token).trim()
      this.saving = true
      try {
        const response = await this.$api.configApi.setSystemAlert(payload)
        if (Number(response.code) === 0) {
          this.apply(response.data)
          this.$message.success('系统报警设置已保存')
        } else this.$message.error(response.msg || '保存失败')
      } finally { this.saving = false }
    },
    async testConnection() {
      if (!this.form.bot_token_set || !String(this.form.chat_id || '').trim()) {
        this.$message.warning('请先保存 Bot Token 和群 Chat ID')
        return
      }
      try {
        await this.$confirm('将立即向已配置的 Telegram 群发送一条测试报警。', '测试 Telegram 报警', { type: 'info', confirmButtonText: '发送测试消息', cancelButtonText: '取消' })
        this.testing = true
        const response = await this.$api.configApi.testSystemAlert({})
        if (Number(response.code) === 0) this.$message.success('测试报警已发送')
        else this.$message.error(response.msg || '测试发送失败')
      } catch (_) {
        // The confirmation dialog was cancelled.
      } finally { this.testing = false }
    }
  },
  computed: {
    groupedOptions() {
      return ['后台操作', '用户端操作'].map(label => ({
        label,
        options: this.options.filter(option => option.group === label)
      })).filter(group => group.options.length)
    }
  },
  render(h) {
    const tokenHint = 'Token 仅在超级管理员的系统报警设置中显示，可直接编辑。'
    return h('div', { class: 'imgo-system-alert', directives: [{ name: 'loading', value: this.loading }] }, [
      h('el-alert', { props: { title: '系统报警通过 Telegram Bot 发送到指定群', type: 'info', showIcon: true, closable: false } }),
      h('el-form', { props: { labelWidth: '170px' } }, [
        h('el-form-item', { props: { label: '启用系统报警' } }, [
          h('el-switch', { props: { value: this.form.enabled, disabled: !this.loaded || this.saving }, on: { change: value => { this.form.enabled = value } } }),
          h('span', { class: 'imgo-system-alert-state' }, [this.form.enabled ? '已开启' : '已关闭'])
        ]),
        h('el-form-item', { props: { label: 'Telegram Bot Token' } }, [
          h('div', { class: 'imgo-system-alert-secret' }, [
            h('el-input', {
              props: { value: this.form.bot_token, type: 'text', autocomplete: 'off', placeholder: '123456789:AA...' },
              on: { input: this.changeToken }
            }),
            this.form.bot_token_set ? h('el-button', { props: { type: 'danger', plain: true }, on: { click: this.clearToken } }, ['清除 Token']) : null
          ]),
          h('div', { class: 'imgo-system-alert-help' }, [tokenHint])
        ]),
        h('el-form-item', { props: { label: 'Telegram 群 Chat ID' } }, [
          h('el-input', { props: { value: this.form.chat_id, placeholder: '例如 -1001234567890 或 @channelname', clearable: true }, on: { input: value => { this.form.chat_id = value } } }),
          h('div', { class: 'imgo-system-alert-help' }, [
            h('p', ['保存 Token 并把 Bot 加入目标群后，在群里发送“@机器人用户名 id”（或“/id@机器人用户名”），Bot 会回复群 Chat ID。']),
            h('p', ['该功能不受报警总开关影响；若 Bot 已配置 webhook，需先停用 webhook 才能使用指令。'])
          ])
        ]),
        h('el-form-item', { props: { label: '报警内容' } }, [
          h('el-checkbox-group', { props: { value: this.form.events }, on: { input: value => { this.form.events = value } } }, this.groupedOptions.map(group => h('div', { key: group.label, class: 'imgo-system-alert-option-group' }, [
            h('div', { class: 'imgo-system-alert-option-title' }, [group.label]),
            h('div', { class: 'imgo-system-alert-option-items' }, group.options.map(option => h('el-checkbox', { key: option.value, props: { label: option.value } }, [option.label])))
          ]))),
          h('div', { class: 'imgo-system-alert-help' }, ['所有报警项默认不勾选，由超级管理员按需选择。'])
        ]),
        h('el-form-item', { props: { label: '消息内容' } }, [
          h('div', { class: 'imgo-system-alert-help' }, [
            h('p', ['每条报警都包含：时间、IP、操作人、操作对象、操作类型。']),
            h('p', ['充值、提现、提现审核和余额调整会额外显示金额。'])
          ])
        ]),
        h('el-form-item', [
          h('el-button', { props: { type: 'primary', loading: this.saving }, on: { click: this.save } }, ['保存设置']),
          h('el-button', { props: { loading: this.testing, disabled: this.saving }, on: { click: this.testConnection } }, ['发送测试报警'])
        ])
      ])
    ])
  }
}
