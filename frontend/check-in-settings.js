const ImgoCheckInSettingsPanel = {
  name: 'ImgoCheckInSettingsPanel',
  data() {
    return {
      loading: false,
      saving: false,
      loaded: false,
      settingScope: 'global',
      inherited: false,
      form: {
        rewardEnabled: 1,
        cycleDays: 10,
        highDays: 3,
        highMinYuan: 7,
        highMaxYuan: 10,
        normalMinYuan: 3,
        normalMaxYuan: 5
      }
    }
  },
  mounted() { this.load() },
  methods: {
    apply(value) {
      const source = value || {}
      this.settingScope = source.setting_scope === 'mentor' ? 'mentor' : 'global'
      this.inherited = Boolean(source.inherited)
      this.form = {
        rewardEnabled: Number(source.rewardEnabled) === 0 ? 0 : 1,
        cycleDays: Number(source.cycleDays) || 10,
        highDays: Number(source.highDays) || 3,
        highMinYuan: Number(source.highMinYuan) || 7,
        highMaxYuan: Number(source.highMaxYuan) || 10,
        normalMinYuan: Number(source.normalMinYuan) || 3,
        normalMaxYuan: Number(source.normalMaxYuan) || 5
      }
      this.loaded = true
    },
    async load() {
      this.loading = true
      try {
        const response = await this.$api.configApi.getCheckInSettings({})
        if (Number(response.code) !== 0) {
          this.$message.error(response.msg || '读取签到设置失败')
          return
        }
        this.apply(response.data)
      } finally { this.loading = false }
    },
    validate() {
      const form = this.form
      if (form.highDays < 1 || form.highDays > form.cycleDays) return '高奖励天数不能超过奖励周期'
      if (form.highMinYuan > form.highMaxYuan) return '前段最低奖励不能大于最高奖励'
      if (form.normalMinYuan > form.normalMaxYuan) return '后段最低奖励不能大于最高奖励'
      return ''
    },
    async save() {
      const message = this.validate()
      if (message) {
        this.$message.warning(message)
        return
      }
      this.saving = true
      try {
        const payload = {
          rewardEnabled: Number(this.form.rewardEnabled),
          cycleDays: Number(this.form.cycleDays),
          highDays: Number(this.form.highDays),
          highMinYuan: Number(this.form.highMinYuan),
          highMaxYuan: Number(this.form.highMaxYuan),
          normalMinYuan: Number(this.form.normalMinYuan),
          normalMaxYuan: Number(this.form.normalMaxYuan)
        }
        const response = await this.$api.configApi.setCheckInSettings(payload)
        if (Number(response.code) === 0) {
          this.apply(response.data || payload)
          this.$message.success('签到设置已保存')
        } else this.$message.error(response.msg || '保存失败')
      } finally { this.saving = false }
    }
  },
  render(h) {
    const mentorScope = this.settingScope === 'mentor'
    const scopeTitle = mentorScope
      ? (this.inherited
          ? '当前显示全局默认值；首次保存后会形成您的独立签到规则，仅对您所属团队生效。'
          : '这是您的独立签到规则，仅对您所属团队生效，不影响其他导师。')
      : '这是系统全局签到规则；尚未单独设置的导师会继承此规则。'
    const numberField = (label, key, min, max, suffix) => h('el-form-item', { props: { label } }, [
      h('el-input-number', {
        props: { value: this.form[key], min, max, step: 1, precision: 0, disabled: !this.loaded || this.saving },
        on: { input: value => { this.form[key] = Number(value) } }
      }),
      h('span', { class: 'imgo-checkin-setting-unit' }, [suffix])
    ])
    return h('div', { class: 'imgo-checkin-settings', directives: [{ name: 'loading', value: this.loading }] }, [
      h('el-alert', { props: { title: scopeTitle, type: mentorScope ? 'warning' : 'info', showIcon: true, closable: false } }),
      h('el-alert', { props: { title: '用户每天签到后，奖励自动进入可用余额，并生成一条余额账变记录。', type: 'info', showIcon: true, closable: false } }),
      h('el-form', { props: { labelWidth: '170px' } }, [
        h('el-form-item', { props: { label: '启用签到奖励' } }, [
          h('el-switch', {
            props: { value: this.form.rewardEnabled, activeValue: 1, inactiveValue: 0, disabled: !this.loaded || this.saving },
            on: { input: value => { this.form.rewardEnabled = Number(value) } }
          }),
          h('span', { class: 'imgo-checkin-setting-state' }, [this.form.rewardEnabled ? '已开启' : '已关闭'])
        ]),
        numberField('奖励周期', 'cycleDays', 2, 31, '天，周期结束后从第1天重新计算'),
        numberField('高奖励天数', 'highDays', 1, Math.max(1, Number(this.form.cycleDays) || 10), '天，从每轮第1天开始计算'),
        h('el-divider', { props: { contentPosition: 'left' } }, ['前段奖励']),
        h('div', { class: 'imgo-checkin-setting-range' }, [
          numberField('最低金额', 'highMinYuan', 1, 10000, '元'),
          numberField('最高金额', 'highMaxYuan', 1, 10000, '元')
        ]),
        h('el-divider', { props: { contentPosition: 'left' } }, ['后段奖励']),
        h('div', { class: 'imgo-checkin-setting-range' }, [
          numberField('最低金额', 'normalMinYuan', 1, 10000, '元'),
          numberField('最高金额', 'normalMaxYuan', 1, 10000, '元')
        ]),
        h('el-form-item', { props: { label: '当前规则预览' } }, [
          h('div', { class: 'imgo-checkin-setting-preview' }, [
            `每${this.form.cycleDays}天一轮；第1–${this.form.highDays}天随机${this.form.highMinYuan}–${this.form.highMaxYuan}元，`,
            this.form.highDays < this.form.cycleDays
              ? `第${this.form.highDays + 1}–${this.form.cycleDays}天随机${this.form.normalMinYuan}–${this.form.normalMaxYuan}元。`
              : '整个周期均使用前段奖励。'
          ])
        ]),
        h('el-form-item', [
          h('el-button', { props: { type: 'primary', loading: this.saving, disabled: !this.loaded }, on: { click: this.save } }, ['保存签到设置'])
        ])
      ])
    ])
  }
}
