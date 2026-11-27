// Vue 2 dialog injected into the compiled member page.
const ImgoMemberAgentSettingDialog = {
  name: 'ImgoMemberAgentSettingDialog',
  data() {
    return {
      visible: false, row: null, loading: false, saving: false, error: '', serial: 0,
      inheritAutoUser: true, inheritAutoGroup: true,
      identityEnabled: true,
      withdrawalEnabled: true,
      withdrawalEnabledInherited: true,
      withdrawalPasswordRequired: false,
      withdrawalPasswordInherited: true,
      globalWithdrawalPasswordRequired: false,
      withdrawalMethods: { bank: true, alipay: true, wechat: true },
      autoAddUser: { status: 0, user_ids: [], welcome: '' },
      autoAddGroup: { status: 0, mode: 'create', group_id: 0, owner_uid: 0, userMax: 5, name: '' },
      globalAutoUser: {}, globalAutoGroup: {}, options: [], groupOptions: [], groupLoadError: '', unavailableNotice: ''
    }
  },
  computed: {
    isSuperOperator() { return Number((this.$store.state.userInfo || {}).user_id) === 1 }
  },
  methods: {
    canManageRow(row) {
      const user = this.$store.state.userInfo || {}
      return !!row && Number(row.admin_role_agent_mode) === 1 && (Number(user.user_id) === 1 || (Number(user.user_id) === Number(row.user_id) && Number(row.is_self) === 1))
    },
    async open(row) {
      if (!this.canManageRow(row) || this.saving) return
      if (this.visible && this.row && Number(this.row.user_id) === Number(row.user_id) && !this.error) return
      const serial = ++this.serial
      this.row = row
      this.visible = true
      this.loading = true
      this.error = ''
      this.options = []
      this.groupOptions = []
      this.groupLoadError = ''
      this.unavailableNotice = ''
      try {
        const [detail, options, groups] = await Promise.all([
          this.$api.agentSettingApi.detail({ agent_user_id: Number(row.user_id) }),
          this.loadOptions(row),
          this.loadGroupOptions().catch(error => {
            this.groupLoadError = error.message || '读取已有群聊失败'
            return []
          })
        ])
        if (serial !== this.serial) return
        if (Number(detail.code) !== 0) throw Error(detail.msg || '读取导师设置失败')
        this.options = options
        const allowedOwners = new Set(options.map(option => Number(option.user_id)))
        this.groupOptions = this.isSuperOperator ? groups : groups.filter(group => allowedOwners.has(Number(group.owner_id)))
        this.applyDetail(detail.data || {})
      } catch (error) {
        if (serial === this.serial) this.error = error.message || '读取导师设置失败'
      } finally {
        if (serial === this.serial) this.loading = false
      }
    },
    async loadOptions(row) {
      const options = Number(row.status) === 1 ? [{ user_id: Number(row.user_id), account: row.account || String(row.user_id), status: row.status }] : []
      let page = 1, fetched = 0
      while (true) {
        const response = await this.$api.userApi.getUserList({ keywords: this.isSuperOperator ? row.account : '', referral_scope: 'all', page, limit: 200 })
        if (Number(response.code) !== 0) throw Error(response.msg || '读取下级账号失败')
        if (!Array.isArray(response.data)) throw Error('下级账号数据无效')
        fetched += response.data.length
        options.push(...response.data.filter(item => Number(item.status) === 1))
        if (response.data.length < 200 || fetched >= Number(response.count || 0)) break
        page++
      }
      return options.filter((item, index, all) => all.findIndex(candidate => Number(candidate.user_id) === Number(item.user_id)) === index)
    },
    async loadGroupOptions() {
      const groups = []
      let page = 1
      while (page <= 50) {
        const response = await this.$api.groupApi.getGroupList({ page, limit: 200 })
        if (Number(response.code) !== 0) throw Error(response.msg || '读取已有群聊失败')
        if (!Array.isArray(response.data)) throw Error('已有群聊数据无效')
        groups.push(...response.data)
        if (response.data.length < 200 || groups.length >= Number(response.count || 0)) break
        page++
      }
      return groups.filter((item, index, all) => Number(item.group_id) > 0 && all.findIndex(candidate => Number(candidate.group_id) === Number(item.group_id)) === index)
    },
    applyDetail(detail) {
      this.inheritAutoUser = detail.inherit_auto_user === true
      this.inheritAutoGroup = detail.inherit_auto_group === true
      this.identityEnabled = detail.identity_enabled !== false && Number(detail.identity_enabled) !== 0
      this.withdrawalEnabled = detail.withdrawal_enabled !== false && Number(detail.withdrawal_enabled) !== 0
      this.withdrawalEnabledInherited = detail.withdrawal_enabled_inherited === true
      this.withdrawalPasswordRequired = detail.withdrawal_password_required === true || Number(detail.withdrawal_password_required) === 1
      this.withdrawalPasswordInherited = detail.withdrawal_password_required_inherited !== false
      this.globalWithdrawalPasswordRequired = detail.global_withdrawal_password_required === true || Number(detail.global_withdrawal_password_required) === 1
      const withdrawal = detail.withdrawal_methods || {}
      this.withdrawalMethods = {
        bank: withdrawal.bank !== false && Number(withdrawal.bank) !== 0,
        alipay: withdrawal.alipay !== false && Number(withdrawal.alipay) !== 0,
        wechat: withdrawal.wechat !== false && Number(withdrawal.wechat) !== 0
      }
      this.globalAutoUser = detail.global_auto_add_user || {}
      this.globalAutoGroup = detail.global_auto_add_group || {}
      const user = this.inheritAutoUser ? this.globalAutoUser : detail.auto_add_user || {}
      const group = this.inheritAutoGroup ? this.globalAutoGroup : detail.auto_add_group || {}
      const allowed = new Set(this.options.map(option => Number(option.user_id)))
      const userIDs = Array.isArray(user.user_ids) ? user.user_ids.map(Number) : []
      const ownerID = Number(group.owner_uid || 0)
      const mode = group.mode === 'existing' ? 'existing' : 'create'
      const groupID = Number(group.group_id || 0)
      const allowedGroups = new Set(this.groupOptions.map(option => Number(option.group_id)))
      this.unavailableNotice = (!this.inheritAutoUser && userIDs.some(id => !allowed.has(id))) || (!this.inheritAutoGroup && mode === 'create' && ownerID > 0 && !allowed.has(ownerID)) || (!this.inheritAutoGroup && mode === 'existing' && groupID > 0 && !allowedGroups.has(groupID))
        ? '已保存的客服、群主或已有群已停用或不在导师范围，已从可编辑配置移除；请重新选择后保存。' : ''
      this.autoAddUser = { status: Number(user.status) === 1 ? 1 : 0, user_ids: userIDs.filter(id => allowed.has(id)), welcome: user.welcome || '' }
      this.autoAddGroup = { status: Number(group.status) === 1 ? 1 : 0, mode, group_id: allowedGroups.has(groupID) ? groupID : 0, owner_uid: allowed.has(ownerID) ? ownerID : 0, userMax: Number(group.userMax || 5), name: group.name || '' }
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
      if (!this.row || this.loading || this.saving || this.error || !this.canManageRow(this.row)) return
      if (!this.inheritAutoUser && this.autoAddUser.status === 1 && !this.autoAddUser.user_ids.length) {
        this.$message.error('请选择至少一位自动添加的客服账号')
        return
      }
      if (!this.inheritAutoGroup && this.autoAddGroup.status === 1) {
        if (!Number.isInteger(Number(this.autoAddGroup.userMax)) || Number(this.autoAddGroup.userMax) < 5 || Number(this.autoAddGroup.userMax) > 10000) {
          this.$message.error('请填写 5 到 10000 的成员上限')
          return
        }
        if (this.autoAddGroup.mode === 'existing' && !this.autoAddGroup.group_id) {
          this.$message.error('请选择要自动加入的已有群聊')
          return
        }
        if (this.autoAddGroup.mode !== 'existing' && (!this.autoAddGroup.owner_uid || !this.autoAddGroup.name.trim())) {
          this.$message.error('请选择群主并填写群名称')
          return
        }
      }
      const payload = {
        agent_user_id: Number(this.row.user_id),
        inherit_auto_user: this.inheritAutoUser,
        inherit_auto_group: this.inheritAutoGroup,
        identity_enabled: this.identityEnabled,
        withdrawal_enabled: this.withdrawalEnabled,
        withdrawal_password_required: this.withdrawalPasswordRequired,
        withdrawal_password_required_inherited: this.withdrawalPasswordInherited,
        withdrawal_methods: { ...this.withdrawalMethods },
        auto_add_user: { status: Number(this.autoAddUser.status), user_ids: this.autoAddUser.user_ids.map(Number), welcome: this.autoAddUser.welcome },
        auto_add_group: { status: Number(this.autoAddGroup.status), mode: this.autoAddGroup.mode === 'existing' ? 'existing' : 'create', group_id: Number(this.autoAddGroup.group_id), owner_uid: Number(this.autoAddGroup.owner_uid), userMax: Number(this.autoAddGroup.userMax), name: this.autoAddGroup.name.trim() }
      }
      this.saving = true
      try {
        const response = await this.$api.agentSettingApi.save(payload)
        if (Number(response.code) !== 0) throw Error(response.msg || '保存导师设置失败')
      } catch (error) {
        this.$message.error(error.message || '保存导师设置失败')
        this.saving = false
        return
      }
      this.$set(this.row, 'agent_setting_inherit_auto_user', payload.inherit_auto_user)
      this.$set(this.row, 'agent_setting_inherit_auto_group', payload.inherit_auto_group)
      this.$set(this.row, 'agent_setting_identity_enabled', payload.identity_enabled)
      this.$set(this.row, 'agent_setting_withdrawal_enabled', payload.withdrawal_enabled)
      this.$set(this.row, 'agent_setting_withdrawal_password_required', payload.withdrawal_password_required)
      this.$set(this.row, 'agent_setting_withdrawal_password_required_inherited', payload.withdrawal_password_required_inherited)
      this.$set(this.row, 'agent_setting_withdrawal_methods', payload.withdrawal_methods)
      this.$emit('saved', payload.agent_user_id)
      this.$message.success('导师设置已保存')
      try {
        const fresh = await this.$api.agentSettingApi.detail({ agent_user_id: payload.agent_user_id })
        if (Number(fresh.code) !== 0) throw Error(fresh.msg || '刷新导师设置失败')
        this.applyDetail(fresh.data || {})
      } catch (error) {
        this.$message.warning('导师设置已保存，刷新失败；请重新打开查看最新设置')
        this.visible = false
        this.row = null
        this.serial++
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
    const groupOptionNodes = this.groupOptions.map(item => h('el-option', { key: item.group_id, props: { value: Number(item.group_id), label: `${item.name || '未命名群聊'}（群号 ${item.group_id}，${Number(item.user_count || 0)}人）` } }))
    const inherited = (config, isGroup) => h('div', { class: 'imgo-agent-global-summary' }, [
      h('span', {}, Number(config.status) === 1 ? '全局已开启' : '全局已关闭'),
      isGroup ? h('span', {}, config.mode === 'existing' ? `已有群 · 群号 ${config.group_id || '—'}` : config.name || '未设置群名称') : h('span', {}, (Array.isArray(config.user_ids) ? config.user_ids.length : 0) + ' 位客服'),
      isGroup ? h('span', {}, config.mode === 'existing' ? ` · 人数上限 ${config.userMax || '—'}` : ` · 群主 ID ${config.owner_uid || '—'} · 人数上限 ${config.userMax || '—'}`) : h('span', {}, ` · 欢迎语：${config.welcome || '无'}`)
    ])
    const disabledUser = this.loading || this.saving
    const disabledGroup = this.loading || this.saving
    const userFields = [
      field('自动添加好友', switchField(this.autoAddUser.status, disabledUser, value => { this.inheritAutoUser = false; this.autoAddUser.status = value }, '自动添加好友')),
      field('客服账号', h('el-select', { props: { value: this.autoAddUser.user_ids, multiple: true, filterable: true, disabled: disabledUser || this.autoAddUser.status !== 1, placeholder: '选择导师或下级账号' }, on: { input: value => { this.inheritAutoUser = false; this.autoAddUser.user_ids = value } } }, optionNodes)),
      field('欢迎语', h('el-input', { props: { value: this.autoAddUser.welcome, disabled: disabledUser || this.autoAddUser.status !== 1 }, attrs: { maxlength: 500 }, on: { input: value => { this.inheritAutoUser = false; this.autoAddUser.welcome = value } } }))
    ]
    const groupFields = [
      field('自动加入群聊', switchField(this.autoAddGroup.status, disabledGroup, value => { this.inheritAutoGroup = false; this.autoAddGroup.status = value }, '自动加入群聊')),
      field('加入方式', h('el-radio-group', { props: { value: this.autoAddGroup.mode, disabled: disabledGroup || this.autoAddGroup.status !== 1 }, on: { input: value => { this.inheritAutoGroup = false; this.autoAddGroup.mode = value } } }, [
        h('el-radio-button', { props: { label: 'create' } }, '自动新建群'),
        h('el-radio-button', { props: { label: 'existing' } }, '加入已有群')
      ]))
    ]
    if (this.autoAddGroup.mode === 'existing') {
      groupFields.push(field('已有群聊', h('el-select', { props: { value: this.autoAddGroup.group_id || '', filterable: true, clearable: true, disabled: disabledGroup || this.autoAddGroup.status !== 1, placeholder: '选择已经创建好的群聊' }, on: { input: value => { this.inheritAutoGroup = false; this.autoAddGroup.group_id = Number(value || 0) } } }, groupOptionNodes)))
    } else {
      groupFields.push(field('群主', h('el-select', { props: { value: this.autoAddGroup.owner_uid, filterable: true, disabled: disabledGroup || this.autoAddGroup.status !== 1, placeholder: '选择导师或下级账号' }, on: { input: value => { this.inheritAutoGroup = false; this.autoAddGroup.owner_uid = value } } }, optionNodes)))
      groupFields.push(field('群名称', h('el-input', { props: { value: this.autoAddGroup.name, disabled: disabledGroup || this.autoAddGroup.status !== 1 }, attrs: { maxlength: 100 }, on: { input: value => { this.inheritAutoGroup = false; this.autoAddGroup.name = value } } })))
    }
    groupFields.push(field('人数上限', h('el-input-number', { props: { value: this.autoAddGroup.userMax, min: 5, max: 10000, disabled: disabledGroup || this.autoAddGroup.status !== 1 }, on: { input: value => { this.inheritAutoGroup = false; this.autoAddGroup.userMax = value } } })))
    return h('el-dialog', {
      props: { title: `导师设置 · ${this.row ? this.row.account || this.row.user_id : ''}`, visible: this.visible, width: '680px', appendToBody: true, closeOnClickModal: false, showClose: !this.saving },
      on: { close: this.close }
    }, this.visible ? [
      this.error ? h('el-alert', { props: { title: this.error, type: 'error', showIcon: true, closable: false } }) : null,
      this.error ? h('el-button', { props: { type: 'text' }, on: { click: () => this.open(this.row) } }, '重试') : null,
      this.unavailableNotice ? h('el-alert', { props: { title: this.unavailableNotice, type: 'warning', showIcon: true, closable: false } }) : null,
      h('div', { class: 'imgo-agent-setting-body', directives: [{ name: 'loading', value: this.loading }] }, [
        h('section', { class: 'imgo-agent-setting-section' }, [
          h('h3', {}, 'H5 实名认证'),
          h('el-form', { class: 'imgo-agent-identity-form' }, [
            field('向下级显示', h('el-switch', {
              props: { value: this.identityEnabled, disabled: this.loading || this.saving },
              attrs: { 'aria-label': '下级实名认证' },
              on: { input: value => { this.identityEnabled = Boolean(value) } }
            }))
          ]),
          h('div', { class: 'imgo-agent-setting-hint' }, '开启后，下级 H5 显示实名认证入口和状态；关闭后隐藏相关信息，并禁止直接提交实名认证。')
        ]),
        h('section', { class: 'imgo-agent-setting-section' }, [
          h('h3', {}, '团队提现'),
          h('el-form', { class: 'imgo-agent-withdrawal-form' }, [
            field('允许提现', h('el-switch', {
              props: { value: this.withdrawalEnabled, disabled: this.loading || this.saving },
              attrs: { 'aria-label': '允许团队提现' },
              on: { input: value => { this.withdrawalEnabled = Boolean(value); this.withdrawalEnabledInherited = false } }
            })),
            field('密码规则', h('div', { class: 'imgo-agent-withdrawal-password-rule' }, [
              h('el-switch', {
                props: { value: this.withdrawalPasswordInherited, disabled: this.loading || this.saving },
                attrs: { 'aria-label': '继承全局提现密码' },
                on: { input: value => {
                  this.withdrawalPasswordInherited = Boolean(value)
                  if (this.withdrawalPasswordInherited) this.withdrawalPasswordRequired = this.globalWithdrawalPasswordRequired
                } }
              }),
              h('span', {}, this.withdrawalPasswordInherited ? '继承全局' : '独立设置')
            ])),
            field('要求密码', h('el-switch', {
              props: { value: this.withdrawalPasswordRequired, disabled: this.loading || this.saving || this.withdrawalPasswordInherited || !this.withdrawalEnabled },
              attrs: { 'aria-label': '团队提现密码' },
              on: { input: value => { this.withdrawalPasswordRequired = Boolean(value) } }
            })),
            field('银行卡', h('el-switch', {
              props: { value: this.withdrawalMethods.bank, disabled: this.loading || this.saving || !this.withdrawalEnabled },
              attrs: { 'aria-label': '下级银行卡提现' },
              on: { input: value => { this.withdrawalMethods.bank = Boolean(value) } }
            })),
            field('支付宝收款码', h('el-switch', {
              props: { value: this.withdrawalMethods.alipay, disabled: this.loading || this.saving || !this.withdrawalEnabled },
              attrs: { 'aria-label': '下级支付宝提现' },
              on: { input: value => { this.withdrawalMethods.alipay = Boolean(value) } }
            })),
            field('微信收款码', h('el-switch', {
              props: { value: this.withdrawalMethods.wechat, disabled: this.loading || this.saving || !this.withdrawalEnabled },
              attrs: { 'aria-label': '下级微信提现' },
              on: { input: value => { this.withdrawalMethods.wechat = Boolean(value) } }
            }))
          ]),
          h('div', { class: 'imgo-agent-setting-hint' }, [
            this.withdrawalEnabledInherited
              ? '提现功能当前继承超级管理员的全局开关；保存后将成为本导师团队的独立开关。'
              : '提现功能开关只影响本导师及其团队成员，不影响其他导师团队。',
            h('br'),
            this.withdrawalPasswordInherited
              ? `提现密码当前继承全局设置（${this.globalWithdrawalPasswordRequired ? '必须验证' : '无需验证'}）。`
              : `提现密码已由本导师独立设置为“${this.withdrawalPasswordRequired ? '必须验证' : '无需验证'}”。`
          ])
        ]),
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
          this.groupLoadError ? h('el-alert', { props: { title: this.groupLoadError, type: 'warning', showIcon: true, closable: false } }) : null,
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
