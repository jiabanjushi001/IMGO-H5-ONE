const ImgoWithdrawalSettingsPanel = {
  name: 'ImgoWithdrawalSettingsPanel',
  data() {
    return { loading: false, saving: false, loaded: false, form: { enabled: 1, minYuan: 100, passwordRequired: 0 } }
  },
  mounted() { this.load() },
  methods: {
    apply(value) {
      const source = value || {}
      this.form = {
        enabled: Number(source.enabled) === 0 ? 0 : 1,
        minYuan: Number(source.minYuan) || 100,
        passwordRequired: Number(source.passwordRequired) === 1 ? 1 : 0
      }
      this.loaded = true
    },
    async load() {
      this.loading = true
      try {
        const response = await this.$api.configApi.getWithdrawalSettings({})
        if (Number(response.code) === 0) this.apply(response.data)
        else this.$message.error(response.msg || '读取提现设置失败')
      } finally { this.loading = false }
    },
    async save() {
      const minimum = Number(this.form.minYuan)
      if (!Number.isInteger(minimum) || minimum < 1 || minimum > 100000000) {
        this.$message.warning('最低提现金额必须是1到100000000元的整数')
        return
      }
      this.saving = true
      try {
        const payload = {
          enabled: Number(this.form.enabled) === 0 ? 0 : 1,
          minYuan: minimum,
          passwordRequired: Number(this.form.passwordRequired) === 1 ? 1 : 0
        }
        const response = await this.$api.configApi.setWithdrawalSettings(payload)
        if (Number(response.code) === 0) {
          this.apply(response.data || payload)
          this.$message.success('提现设置已保存并立即生效')
        } else this.$message.error(response.msg || '保存提现设置失败')
      } finally { this.saving = false }
    }
  },
  render(h) {
    return h('div', { class: 'imgo-withdrawal-settings', directives: [{ name: 'loading', value: this.loading }] }, [
      h('el-alert', { props: { title: '该设置仅超级管理员可修改；服务端会强制校验提现开关、最低金额和提现密码。', type: 'info', showIcon: true, closable: false } }),
      h('el-form', { props: { labelWidth: '180px' } }, [
        h('el-form-item', { props: { label: '提现功能' } }, [
          h('el-switch', {
            props: { value: this.form.enabled, activeValue: 1, inactiveValue: 0, disabled: !this.loaded || this.saving },
            on: { input: value => { this.form.enabled = Number(value) } }
          }),
          h('span', { class: 'imgo-withdrawal-setting-state' }, [this.form.enabled ? '已开启' : '已关闭'])
        ]),
        h('el-form-item', { props: { label: '最低提现金额' } }, [
          h('el-input-number', {
            props: { value: this.form.minYuan, min: 1, max: 100000000, step: 1, precision: 0, disabled: !this.loaded || this.saving },
            on: { input: value => { this.form.minYuan = Number(value) } }
          }),
          h('span', { class: 'imgo-withdrawal-setting-unit' }, ['元'])
        ]),
        h('el-form-item', { props: { label: '提现密码' } }, [
          h('el-switch', {
            props: { value: this.form.passwordRequired, activeValue: 1, inactiveValue: 0, disabled: !this.loaded || this.saving },
            on: { input: value => { this.form.passwordRequired = Number(value) } }
          }),
          h('span', { class: 'imgo-withdrawal-setting-state' }, [this.form.passwordRequired ? '必须验证' : '无需验证'])
        ]),
        h('el-form-item', { props: { label: '生效说明' } }, [
          h('div', { class: 'imgo-withdrawal-setting-description' }, [
            this.form.enabled
              ? `用户单笔提现金额不得低于 ¥${Number(this.form.minYuan || 0).toFixed(2)}。`
              : '用户端提现入口将显示为不可用，所有新的提现提交都会被服务端拒绝。',
            h('br'),
            this.form.passwordRequired
              ? '用户未设置密码时需先设置 6 位数字密码；设置后每次提现都必须验证。'
              : '用户提现时无需验证提现密码。',
            h('br'),
            '已提交的提现订单不受开关影响，仍可在“财务 → 提现订单”中继续审核。'
          ])
        ]),
        h('el-form-item', [
          h('el-button', { props: { type: 'primary', loading: this.saving, disabled: !this.loaded }, on: { click: this.save } }, ['保存提现设置'])
        ])
      ])
    ])
  }
}
