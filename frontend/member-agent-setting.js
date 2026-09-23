// Vue 2 dialog injected into the compiled member page.
const ImgoMemberAgentSettingDialog = {
  name: 'ImgoMemberAgentSettingDialog',
  data() {
    return {
      visible: false, row: null, loading: false, saving: false, error: '', serial: 0,
      inheritAutoUser: true, inheritAutoGroup: true,
      autoAddUser: { status: 0, user_ids: [], welcome: '' },
      autoAddGroup: { status: 0, owner_uid: 0, userMax: 5, name: '' },
      globalAutoUser: {}, globalAutoGroup: {}, options: []
    }
  },
  computed: {
    isSuperOperator() { return Number((this.$store.state.userInfo || {}).user_id) === 1 }
  },
  methods: {
    async open(row) {
      if (!this.isSuperOperator || !row || Number(row.admin_role_agent_mode) !== 1 || this.saving) return
      if (this.visible && this.row && Number(this.row.user_id) === Number(row.user_id) && !this.error) return
      const serial = ++this.serial
      this.row = row
      this.visible = true
      this.loading = true
      this.error = ''
      this.options = []
      try {
        const [detail, options] = await Promise.all([
          this.$api.agentSettingApi.detail({ agent_user_id: Number(row.user_id) }),
          this.loadOptions(row)
        ])
        if (serial !== this.serial) return
        if (Number(detail.code) !== 0) throw Error(detail.msg || '读取导师设置失败')
        this.options = options
        this.applyDetail(detail.data || {})
      } catch (error) {
        if (serial === this.serial) this.error = error.message || '读取导师设置失败'
      } finally {
        if (serial === this.serial) this.loading = false
      }
    },
    async loadOptions(row) {
      const options = [{ user_id: Number(row.user_id), account: row.account || String(row.user_id) }]
      let page = 1
      while (true) {
        const response = await this.$api.userApi.getUserList({ keywords: row.account, referral_scope: 'all', page, limit: 200 })
        if (Number(response.code) !== 0) throw Error(response.msg || '读取下级账号失败')
        if (!Array.isArray(response.data)) throw Error('下级账号数据无效')
        options.push(...response.data)
        if (response.data.length < 200 || options.length - 1 >= Number(response.count || 0)) break
        page++
      }
      return options.filter((item, index, all) => all.findIndex(candidate => Number(candidate.user_id) === Number(item.user_id)) === index)
    },
    applyDetail(detail) {
      this.inheritAutoUser = detail.inherit_auto_user === true
      this.inheritAutoGroup = detail.inherit_auto_group === true
      this.globalAutoUser = detail.global_auto_add_user || {}
      this.globalAutoGroup = detail.global_auto_add_group || {}
      const user = this.inheritAutoUser ? this.globalAutoUser : detail.auto_add_user || {}
      const group = this.inheritAutoGroup ? this.globalAutoGroup : detail.auto_add_group || {}
      const allowed = new Set(this.options.map(option => Number(option.user_id)))
      const userIDs = Array.isArray(user.user_ids) ? user.user_ids.map(Number) : []
      const ownerID = Number(group.owner_uid || 0)
      this.autoAddUser = { status: Number(user.status) === 1 ? 1 : 0, user_ids: userIDs.filter(id => allowed.has(id)), welcome: user.welcome || '' }
      this.autoAddGroup = { status: Number(group.status) === 1 ? 1 : 0, owner_uid: allowed.has(ownerID) ? ownerID : 0, userMax: Number(group.userMax || 5), name: group.name || '' }
      if (this.row) {
        this.$set(this.row, 'agent_setting_inherit_auto_user', this.inheritAutoUser)
        this.$set(this.row, 'agent_setting_inherit_auto_group', this.inheritAutoGroup)
      }
    },
    close() {
      if (this.saving) return
      this.visible = false
      this.row = null
      this.serial++
    },
    async submit() {
      if (!this.row || this.loading || this.saving || this.error || !this.isSuperOperator) return
      if (!this.inheritAutoUser && this.autoAddUser.status === 1 && !this.autoAddUser.user_ids.length) {
        this.$message.error('请选择至少一位自动添加的客服账号')
        return
      }
      if (!this.inheritAutoGroup && this.autoAddGroup.status === 1 && (!this.autoAddGroup.owner_uid || !this.autoAddGroup.name.trim() || !Number.isInteger(Number(this.autoAddGroup.userMax)) || Number(this.autoAddGroup.userMax) < 5 || Number(this.autoAddGroup.userMax) > 10000)) {
        this.$message.error('请选择群主、填写群名称和 5 到 10000 的成员上限')
        return
      }
      const payload = {
        agent_user_id: Number(this.row.user_id),
        inherit_auto_user: this.inheritAutoUser,
        inherit_auto_group: this.inheritAutoGroup,
        auto_add_user: { status: Number(this.autoAddUser.status), user_ids: this.autoAddUser.user_ids.map(Number), welcome: this.autoAddUser.welcome },
        auto_add_group: { status: Number(this.autoAddGroup.status), owner_uid: Number(this.autoAddGroup.owner_uid), userMax: Number(this.autoAddGroup.userMax), name: this.autoAddGroup.name.trim() }
      }
      this.saving = true
      try {
        const response = await this.$api.agentSettingApi.save(payload)
        if (Number(response.code) !== 0) throw Error(response.msg || '保存导师设置失败')
        this.$set(this.row, 'agent_setting_inherit_auto_user', payload.inherit_auto_user)
        this.$set(this.row, 'agent_setting_inherit_auto_group', payload.inherit_auto_group)
        this.$emit('saved', payload.agent_user_id)
        const fresh = await this.$api.agentSettingApi.detail({ agent_user_id: payload.agent_user_id })
        if (Number(fresh.code) !== 0) throw Error(fresh.msg || '刷新导师设置失败')
        this.applyDetail(fresh.data || {})
        this.$message.success('导师设置已保存')
      } catch (error) {
        this.$message.error(error.message || '保存导师设置失败')
      } finally { this.saving = false }
    }
  },
  render(h) {
    const field = (label, child) => h('el-form-item', { props: { label, labelWidth: '115px' } }, [child])
    const switchField = (value, disabled, change, label) => h('el-switch', {
      props: { value, disabled, activeValue: 1, inactiveValue: 0 }, attrs: { 'aria-label': label }, on: { input: change }
    })
    const inheritSwitch = (value, change) => h('el-switch', {
      props: { value, disabled: this.loading || this.saving }, attrs: { 'aria-label': '继承全局设置' }, on: { input: change }
    })
    const optionNodes = this.options.map(item => h('el-option', { key: item.user_id, props: { value: Number(item.user_id), label: `${item.account || item.user_id}（ID ${item.user_id}）` } }))
    const inherited = (config, isGroup) => h('div', { class: 'imgo-agent-global-summary' }, [
      h('span', {}, Number(config.status) === 1 ? '全局已开启' : '全局已关闭'),
      isGroup ? h('span', {}, config.name || '未设置群名称') : h('span', {}, (Array.isArray(config.user_ids) ? config.user_ids.length : 0) + ' 位客服'),
      isGroup ? h('span', {}, ` · 群主 ID ${config.owner_uid || '—'} · 人数上限 ${config.userMax || '—'}`) : h('span', {}, ` · 欢迎语：${config.welcome || '无'}`)
    ])
    const disabledUser = this.loading || this.saving || this.inheritAutoUser
    const disabledGroup = this.loading || this.saving || this.inheritAutoGroup
    const userFields = [
      field('自动添加好友', switchField(this.autoAddUser.status, disabledUser, value => { this.autoAddUser.status = value }, '自动添加好友')),
      field('客服账号', h('el-select', { props: { value: this.autoAddUser.user_ids, multiple: true, filterable: true, disabled: disabledUser || this.autoAddUser.status !== 1, placeholder: '选择导师或下级账号' }, on: { input: value => { this.autoAddUser.user_ids = value } } }, optionNodes)),
      field('欢迎语', h('el-input', { props: { value: this.autoAddUser.welcome, disabled: disabledUser || this.autoAddUser.status !== 1 }, attrs: { maxlength: 500 }, on: { input: value => { this.autoAddUser.welcome = value } } }))
    ]
    const groupFields = [
      field('自动加入群聊', switchField(this.autoAddGroup.status, disabledGroup, value => { this.autoAddGroup.status = value }, '自动加入群聊')),
      field('群主', h('el-select', { props: { value: this.autoAddGroup.owner_uid, filterable: true, disabled: disabledGroup || this.autoAddGroup.status !== 1, placeholder: '选择导师或下级账号' }, on: { input: value => { this.autoAddGroup.owner_uid = value } } }, optionNodes)),
      field('群名称', h('el-input', { props: { value: this.autoAddGroup.name, disabled: disabledGroup || this.autoAddGroup.status !== 1 }, attrs: { maxlength: 100 }, on: { input: value => { this.autoAddGroup.name = value } } })),
      field('人数上限', h('el-input-number', { props: { value: this.autoAddGroup.userMax, min: 5, max: 10000, disabled: disabledGroup || this.autoAddGroup.status !== 1 }, on: { input: value => { this.autoAddGroup.userMax = value } } }))
    ]
    return h('el-dialog', {
      props: { title: `导师设置 · ${this.row ? this.row.account || this.row.user_id : ''}`, visible: this.visible, width: '680px', appendToBody: true, closeOnClickModal: false, showClose: !this.saving },
      on: { close: this.close }
    }, this.visible ? [
      this.error ? h('el-alert', { props: { title: this.error, type: 'error', showIcon: true, closable: false } }) : null,
      this.error ? h('el-button', { props: { type: 'text' }, on: { click: () => this.open(this.row) } }, '重试') : null,
      h('div', { class: 'imgo-agent-setting-body', directives: [{ name: 'loading', value: this.loading }] }, [
        h('section', { class: 'imgo-agent-setting-section' }, [
          h('h3', {}, '自动添加好友'),
          field('继承全局设置', inheritSwitch(this.inheritAutoUser, value => { this.inheritAutoUser = value })),
          this.inheritAutoUser ? inherited(this.globalAutoUser, false) : null,
          h('el-form', {}, userFields)
        ]),
        h('section', { class: 'imgo-agent-setting-section' }, [
          h('h3', {}, '自动加入群聊'),
          field('继承全局设置', inheritSwitch(this.inheritAutoGroup, value => { this.inheritAutoGroup = value })),
          this.inheritAutoGroup ? inherited(this.globalAutoGroup, true) : null,
          h('el-form', {}, groupFields)
        ])
      ]),
      h('span', { slot: 'footer' }, [
        h('el-button', { props: { disabled: this.saving }, on: { click: this.close } }, '关闭'),
        h('el-button', { props: { type: 'primary', loading: this.saving, disabled: this.loading || !!this.error }, on: { click: this.submit } }, '保存')
      ])
    ] : [])
  }
}
