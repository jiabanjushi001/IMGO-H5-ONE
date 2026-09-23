"use strict";(self["webpackChunkRaingad_IM"]=self["webpackChunkRaingad_IM"]||[]).push([[687],{7076:function(e,t,s){s.d(t,{Z:function(){return p}});var a=function(){var e=this,t=e._self._c;return t("div",[t("el-container",[t("el-aside",{attrs:{width:"320px"}},[t("div",{staticClass:"lz-flex group-box"},[e.showSearch?t("div",{staticClass:"group-box-header"},[t("el-input",{staticStyle:{width:"300px"},attrs:{placeholder:"请输入关键字搜索"},nativeOn:{keyup:function(t){return!t.type.indexOf("key")&&e._k(t.keyCode,"enter",13,t.key,"Enter")?null:e.handleChange.apply(null,arguments)}},model:{value:e.params.keywords,callback:function(t){e.$set(e.params,"keywords",t)},expression:"params.keywords"}},[t("el-button",{attrs:{slot:"prepend",icon:"el-icon-back"},on:{click:function(t){e.showSearch=!1}},slot:"prepend"}),t("el-button",{attrs:{slot:"append",icon:"el-icon-search"},on:{click:e.handleChange},slot:"append"})],1)],1):t("div",{staticClass:"group-box-header"},[t("div",[t("el-button",{attrs:{type:"primary",size:"small",plain:""},on:{click:function(t){return e.chooseTab(1)}}},[e._v("TA的会话")]),t("el-button",{staticClass:"ml-10",attrs:{type:"success",size:"small",plain:""},on:{click:function(t){return e.chooseTab(0)}}},[e._v("TA的联系人")])],1),t("div",[t("el-button",{attrs:{plain:"",circle:"",icon:"el-icon-search",title:"搜索"},on:{click:function(t){e.showSearch=!0}}})],1)]),t("div",{staticClass:"group-box-list"},[t("el-scrollbar",e._l(e.list,(function(s){return t("div",{key:s.user_id,staticClass:"chat-item",class:e.active==s.user_id?"active":"",on:{click:function(t){return e.openChat(s)}}},[t("div",{staticClass:"chat-avatar"},[t("img",{attrs:{src:s.avatar,alt:"avatar"}})]),t("div",{staticClass:"chat-content"},[t("span",{staticClass:"chat-name"},[e._v(e._s(s.realname))])])])})),0)],1),t("div",{staticClass:"group-box-page",attrs:{align:"center"}},[t("el-pagination",{attrs:{background:"",total:e.total,"current-page":e.params.page,"page-size":e.params.limit,layout:"total, prev, next , jumper"},on:{"current-change":e.getList,"update:currentPage":function(t){return e.$set(e.params,"page",t)},"update:current-page":function(t){return e.$set(e.params,"page",t)},"update:pageSize":function(t){return e.$set(e.params,"limit",t)},"update:page-size":function(t){return e.$set(e.params,"limit",t)}}})],1)])]),t("el-main",{staticStyle:{padding:"0"}},[t("div",{staticClass:"lz-flex group-box group-user-box"},[t("div",{staticClass:"group-box-header"},[t("div",[e._v("聊天记录")])]),t("div",{staticClass:"group-box-list",staticStyle:{padding:"15px"}},[e.currentChat.user_id?t("ChatRecord",{key:e.componentKey,attrs:{contact:e.currentChat,condition:e.condition,manage:!0}}):e._e()],1)])])],1)],1)},r=[],i=s(284),l=s(8100),o=(s(2325),{components:{Group:i.Z,ChatRecord:l.Z},props:{userInfo:{type:Object,default:{}}},data(){return{componentKey:99,messageBox:!1,isAdd:!0,dialogTitle:"创建群聊",createChatBox:!1,userIds:[],showSearch:!1,value:!1,active:0,currentChat:{},params:{hasConvo:1,user_id:0,page:1,limit:20,keywords:""},total:0,list:[],condition:{}}},created(){this.getList()},methods:{openChat(e){this.active=e.user_id,this.currentChat=e,this.componentKey++,this.condition={user_id:this.userInfo.user_id}},chooseTab(e){this.params.page=1,this.params.hasConvo=e,this.getList()},getList(){this.params.user_id=this.userInfo.user_id,this.$api.messageApi.getContacts(this.params).then((e=>{0==e.code&&(this.list=e.data,this.total=e.count,this.params.page=e.page)}))},handleChange(){this.params.page=1,this.getList()}}}),n=o,d=s(1001),c=(0,d.Z)(n,a,r,!1,null,"61b0eeda",null),p=c.exports},4368:function(e,t,s){/* IMGO_MEMBER_AGENT_SETTING_BEGIN */// Vue 2 dialog injected into the compiled member page.
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
/* IMGO_MEMBER_AGENT_SETTING_END *//* IMGO_MEMBER_ROLE_BEGIN */const ImgoMemberRoleSelect = {
  name: 'ImgoMemberRoleSelect',
  props: { row: { type: Object, required: true } },
  data() { return { roles: [], saving: false } },
  computed: {
    isSuperOperator() { return Number((this.$store.state.userInfo || {}).user_id) === 1 },
    roleValue() { return Number(this.row.admin_role_id || 0) },
    roleLabel() { return this.row.admin_role_name || (Number(this.row.user_id) === 1 ? '超级管理员' : '普通用户') }
  },
  mounted() {
    if (!this.isSuperOperator || Number(this.row.user_id) === 1) return
    if (!window.__imgoRoleOptionsPromise) {
      window.__imgoRoleOptionsPromise = this.$api.roleApi.index({}).then(result => {
        if (result.code !== 0) throw Error(result.msg || '读取角色失败')
        return Array.isArray(result.data) ? result.data.filter(role => !role.builtin && Number(role.role_id) > 0) : []
      }).catch(error => { window.__imgoRoleOptionsPromise = null; throw error })
    }
    window.__imgoRoleOptionsPromise.then(roles => { this.roles = roles }).catch(error => this.$message.error(error.message || '读取角色失败'))
  },
  methods: {
    async change(adminRoleID) {
      if (this.saving || Number(adminRoleID) === this.roleValue) return
      const previousID = this.roleValue
      const previousName = this.roleLabel
      const previousAgentMode = Number(this.row.admin_role_agent_mode || 0)
      const selected = this.roles.find(role => Number(role.role_id) === Number(adminRoleID))
      this.$set(this.row, 'admin_role_id', Number(adminRoleID))
      this.$set(this.row, 'admin_role_name', selected ? selected.name : '普通用户')
      this.$set(this.row, 'admin_role_agent_mode', selected ? Number(selected.agent_mode || 0) : 0)
      this.saving = true
      try {
        const result = await this.$api.userApi.setRole({ user_id: this.row.user_id, admin_role_id: Number(adminRoleID) })
        if (result.code !== 0) throw Error(result.msg || '角色设置失败')
        this.$message.success('角色已更新')
      } catch (error) {
        this.$set(this.row, 'admin_role_id', previousID)
        this.$set(this.row, 'admin_role_name', previousName)
        this.$set(this.row, 'admin_role_agent_mode', previousAgentMode)
        this.$message.error(error.message || '角色设置失败')
      } finally { this.saving = false }
    }
  },
  render(h) {
    if (!this.isSuperOperator || Number(this.row.user_id) === 1) {
      return h('el-tag', { props: { size: 'mini', type: Number(this.row.user_id) === 1 ? 'danger' : 'info' } }, this.roleLabel)
    }
    const options = [h('el-option', { key: 0, props: { label: '普通用户', value: 0 } })]
    for (const role of this.roles) {
      options.push(h('el-option', { key: role.role_id, props: { label: role.name, value: Number(role.role_id), disabled: Number(role.status) !== 1 } }))
    }
    return h('el-select', {
      class: 'imgo-member-role-select',
      props: { value: this.roleValue, size: 'mini', loading: this.saving, disabled: this.saving },
      on: { input: this.change }
    }, options)
  }
}
/* IMGO_MEMBER_ROLE_END *//* IMGO_INVITE_COPY_BEGIN */// Small Vue 2 cell component: display one invite code and copy it on demand.
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
/* IMGO_INVITE_COPY_END *//* IMGO_MEMBER_INVITE_CODE_BEGIN */// Small Vue 2 dialog owned by the existing member list's More menu.
const ImgoMemberInviteCodeDialog = {
  name: 'ImgoMemberInviteCodeDialog',
  data() {
    return { visible: false, member: null, draft: '', saving: false }
  },
  computed: {
    valid() { return /^\d{6}$/.test(this.draft) },
    changed() { return this.member && this.draft !== this.member.invite_code }
  },
  methods: {
    open(row) {
      if (!row || Number(row.user_id) < 1) return
      this.member = row
      this.draft = String(row.invite_code || '')
      this.visible = true
      this.$nextTick(() => this.$refs.codeInput && this.$refs.codeInput.focus())
    },
    close() { if (!this.saving) this.visible = false },
    async save() {
      if (this.saving || !this.member) return
      if (!this.valid) { this.$message.warning('邀请码必须是 6 位纯数字'); return }
      if (!this.changed) { this.close(); return }
      this.saving = true
      try {
        const result = await this.$api.userApi.setInviteCode({ user_id: this.member.user_id, invite_code: this.draft })
        if (Number(result.code) !== 0) throw Error(result.msg || '保存邀请码失败')
        this.$set(this.member, 'invite_code', this.draft)
        this.$message.success('邀请码已修改')
        this.visible = false
      } catch (error) {
        this.$message.error(error.message || '保存邀请码失败')
      } finally {
        this.saving = false
      }
    }
  },
  render(h) {
    const member = this.member || {}
    return h('el-dialog', {
      props: { title: '修改邀请码', visible: this.visible, width: '420px', appendToBody: true, closeOnClickModal: false },
      on: { close: this.close }
    }, this.visible ? [
      h('p', { style: { marginTop: '0', color: '#606266' } }, `成员：${member.account || member.realname || member.user_id}`),
      h('el-input', {
        ref: 'codeInput',
        props: { value: this.draft, maxlength: 6, clearable: true, placeholder: '请输入 6 位数字邀请码', disabled: this.saving },
        attrs: { inputmode: 'numeric' },
        on: { input: value => { this.draft = String(value) }, keyup: event => { if (event.key === 'Enter') this.save() } }
      }),
      h('p', { style: { color: '#909399', fontSize: '12px' } }, '保存前会检查唯一性；已被其他成员使用的邀请码不能保存。'),
      h('span', { slot: 'footer' }, [
        h('el-button', { props: { disabled: this.saving }, on: { click: this.close } }, '取消'),
        h('el-button', { props: { type: 'primary', disabled: !this.valid || !this.changed, loading: this.saving }, on: { click: this.save } }, '保存')
      ])
    ] : [])
  }
}
/* IMGO_MEMBER_INVITE_CODE_END *//* IMGO_MEMBER_REMARK_BEGIN */const ImgoMemberRemark = {
 name: 'ImgoMemberRemark',
 props: {row: {type: Object, required: true}},
 data() { return {editing: false, draft: '', saving: false}; },
 methods: {
  open() { this.draft = this.row.remark || ''; this.editing = true; this.$nextTick(() => this.$refs.input.focus()); },
  async save() {
   if (this.saving) return;
   if (this.draft === (this.row.remark || '')) { this.editing = false; return; }
   this.saving = true;
   try {
    const result = await this.$api.userApi.setRemark({user_id: this.row.user_id, remark: this.draft});
    if (Number(result.code) !== 0) throw new Error(result.msg || '保存失败');
    this.$set(this.row, 'remark', this.draft); this.editing = false; this.$message.success('备注已保存');
   } catch (error) { this.$message.error(error.message || '保存失败，请重试'); }
   finally { this.saving = false; }
  }
 },
 render(h) {
  if (!this.editing) return h('button', {class: 'imgo-inline-remark', attrs: {type: 'button', title: '点击编辑备注'}, on: {click: this.open}}, [
   h('span', this.row.remark || '添加备注'), h('i', {class: 'el-icon-edit'})
  ]);
  return h('div', {class: 'imgo-inline-remark-form'}, [
   h('el-input', {ref: 'input', props: {type: 'textarea', value: this.draft, maxlength: 191, rows: 2, disabled: this.saving, placeholder: '填写备注'}, on: {input: value => {this.draft = value;}}}),
   h('el-button', {props: {type: 'text', size: 'mini', loading: this.saving}, on: {click: this.save}}, '保存'),
   h('el-button', {props: {type: 'text', size: 'mini', disabled: this.saving}, on: {click: () => {this.editing = false;}}}, '取消')
  ]);
 }
};
/* IMGO_MEMBER_REMARK_END */// Read-only Vue 2 dialog for the existing compiled member page.
const ImgoCheckInHistory = {
  name: 'ImgoCheckInHistory',
  data() {
    return { visible: false, member: null, rows: [], total: 0, page: 1, loading: false, requestSerial: 0 }
  },
  methods: {
    open(member) {
      if (!member || Number(member.user_id) < 1) return
      this.member = { user_id: Number(member.user_id), account: member.account || '', realname: member.realname || '' }
      this.page = 1
      this.rows = []
      this.total = Number(member.checkin_days || 0)
      this.visible = true
      this.refresh()
    },
    close() {
      this.visible = false
      this.requestSerial++
      this.loading = false
    },
    async refresh() {
      if (!this.member) return
      const serial = ++this.requestSerial
      this.loading = true
      try {
        const res = await this.$api.userApi.checkInHistory({ user_id: this.member.user_id, page: this.page, limit: 20 })
        if (serial !== this.requestSerial) return
        if (res.code !== 0) throw Error(res.msg || '读取签到记录失败')
        this.rows = Array.isArray(res.data) ? res.data : []
        this.total = Number(res.count || 0)
      } catch (error) {
        if (serial !== this.requestSerial) return
        this.rows = []
        this.$message.error(error.message || '读取签到记录失败')
      } finally {
        if (serial === this.requestSerial) this.loading = false
      }
    },
    signedAt(value) {
      const seconds = Number(value)
      return seconds > 0 ? new Date(seconds * 1000).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai', hour12: false }) : '—'
    }
  },
  render(h) {
    const member = this.member || {}
    return h('el-dialog', {
      props: { title: '签到详情', visible: this.visible, width: '560px', appendToBody: true, closeOnClickModal: false },
      on: { close: this.close }
    }, this.visible ? [
      h('div', { class: 'imgo-checkin-history-head' }, [
        h('div', [h('strong', member.realname || member.account || `用户 ${member.user_id}`), h('span', `账号：${member.account || '—'} · ID：${member.user_id}`)]),
        h('div', { class: 'imgo-checkin-history-total' }, [h('strong', String(this.total)), h('span', '累计签到天数')])
      ]),
      h('el-table', { class: 'imgo-checkin-history-table', props: { data: this.rows, stripe: true, border: true, maxHeight: 420 }, directives: [{ name: 'loading', value: this.loading }] }, [
        h('el-table-column', { props: { label: '签到日期', prop: 'sign_date', minWidth: '160' } }),
        h('el-table-column', { props: { label: '签到时间（北京时间）', minWidth: '220' }, scopedSlots: { default: scope => this.signedAt(scope.row.signed_at) } })
      ]),
      h('el-pagination', {
        class: 'imgo-checkin-history-pagination',
        props: { currentPage: this.page, pageSize: 20, total: this.total, layout: 'total, prev, pager, next' },
        on: { 'current-change': page => { this.page = page; this.refresh() } }
      }),
      h('span', { slot: 'footer' }, [
        h('el-button', { on: { click: this.close } }, '关闭')
      ])
    ] : [])
  }
}

// Vue 2 dialog attached to each row's finance buttons on the legacy member page.
function imgoMemberFinanceRequestID() {
  return 'admin-' + Date.now().toString(36) + '-' + Math.random().toString(36).slice(2, 15)
}
function imgoMemberFinanceCents(value) {
  const match = /^(\d{1,9})(?:\.(\d{1,2}))?$/.exec(String(value).trim())
  return match ? Number(match[1]) * 100 + Number((match[2] || '').padEnd(2, '0')) : 0
}
const ImgoMemberFinanceDialog = {
  name: 'ImgoMemberFinanceDialog',
  data() {
    return {
      visible: false, mode: 'recharge', member: null, account: null, loading: false, saving: false,
      amount: '', note: '', bonusMode: 'none', bonusValue: '', requestID: imgoMemberFinanceRequestID(),
      entries: [], recharges: [], withdrawals: [], historyLoading: false, serial: 0
    }
  },
  computed: {
    amountCents() { return imgoMemberFinanceCents(this.amount) },
    bonusCents() {
      if (!this.amountCents) return 0
      if (this.bonusMode === 'none') return 0
      if (this.bonusMode === 'fixed') return imgoMemberFinanceCents(this.bonusValue)
      const match = /^(\d{1,4})(?:\.(\d{1,2}))?$/.exec(String(this.bonusValue).trim())
      if (!match) return 0
      const basisPoints = Number(match[1]) * 100 + Number((match[2] || '').padEnd(2, '0'))
      return basisPoints > 0 && basisPoints <= 100000 ? Math.floor((this.amountCents * basisPoints + 5000) / 10000) : 0
    }
  },
  methods: {
    money(cents) { return '¥' + (Number(cents || 0) / 100).toFixed(2) },
    date(value) { return value ? new Date(Number(value) * 1000).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai', hour12: false }) : '—' },
    async open(member, mode) {
      if (!member || Number(member.user_id) < 1 || !['recharge', 'withdraw'].includes(mode)) return
      this.member = { user_id: Number(member.user_id), account: member.account || '', realname: member.realname || '' }
      this.mode = mode
      this.amount = ''
      this.note = ''
      this.bonusMode = 'none'
      this.bonusValue = ''
      this.requestID = imgoMemberFinanceRequestID()
      this.account = null
      this.entries = []
      this.recharges = []
      this.withdrawals = []
      this.visible = true
      await this.refresh()
    },
    close() { if (!this.saving) { this.visible = false; this.serial++; this.member = null; this.account = null } },
    async refresh() {
      if (!this.member) return
      const serial = ++this.serial, userID = this.member.user_id
      this.loading = true
      this.historyLoading = true
      try {
        const response = await this.$api.walletApi.account({ user_id: userID })
        if (serial !== this.serial) return
        if (response.code !== 0) throw Error(response.msg || '读取钱包失败')
        this.account = response.data
      } catch (error) {
        if (serial === this.serial) this.$message.error(error.message || '读取钱包失败')
      } finally { if (serial === this.serial) this.loading = false }
      try {
        const [entries, recharges, withdrawals] = await Promise.all([
          this.$api.walletApi.entries({ user_id: userID, page: 1, limit: 10 }),
          this.$api.walletApi.recharges({ user_id: userID, page: 1, limit: 10 }),
          this.$api.walletApi.index({ user_id: userID, status: '', page: 1, limit: 10 })
        ])
        if (serial !== this.serial) return
        for (const item of [entries, recharges, withdrawals]) if (item.code !== 0) throw Error(item.msg || '读取资金记录失败')
        this.entries = Array.isArray(entries.data) ? entries.data : []
        this.recharges = Array.isArray(recharges.data) ? recharges.data : []
        this.withdrawals = Array.isArray(withdrawals.data) ? withdrawals.data : []
      } catch (error) {
        if (serial === this.serial) this.$message.error(error.message || '读取资金记录失败')
      } finally { if (serial === this.serial) this.historyLoading = false }
    },
    change(key, value) { this[key] = value; this.requestID = imgoMemberFinanceRequestID() },
    async submit() {
      if (!this.member || this.saving || !this.account) return
      const amount = String(this.amount).trim(), note = String(this.note).trim()
      if (!this.amountCents || note.length > 500 || (this.mode === 'recharge' && note.length < 2)) {
        this.$message.warning('请输入正确金额；充值备注至少 2 字，最多 500 字')
        return
      }
      let bonusValue = ''
      if (this.mode === 'recharge' && this.bonusMode !== 'none') {
        bonusValue = String(this.bonusValue).trim()
        const valid = this.bonusMode === 'fixed'
          ? imgoMemberFinanceCents(bonusValue) > 0
          : /^(\d{1,4})(?:\.(\d{1,2}))?$/.test(bonusValue) && Number(bonusValue) > 0 && Number(bonusValue) <= 1000 && this.bonusCents > 0
        if (!valid) { this.$message.warning('请填写有效的赠送金额或百分比（最多 1000%）'); return }
      }
      const recharge = this.mode === 'recharge'
      const title = recharge ? '确认充值并立即到账' : '确认提现并立即冻结'
      const message = recharge
        ? `将为 ${this.member.account} 充值 ${this.money(this.amountCents)}，赠送 ${this.money(this.bonusCents)}，合计 ${this.money(this.amountCents + this.bonusCents)} 立即增加可用余额。`
        : `将为 ${this.member.account} 创建提现订单 ${this.money(this.amountCents)}，立即从可用余额冻结，待线下打款审核。`
      try { await this.$confirm(message, title, { type: 'warning', confirmButtonText: '确认操作', cancelButtonText: '取消' }) }
      catch (_) { return }
      this.saving = true
      try {
        const payload = { user_id: this.member.user_id, amount, note, request_id: this.requestID }
        if (recharge) { payload.bonus_mode = this.bonusMode; payload.bonus_value = bonusValue }
        const result = await (recharge ? this.$api.walletApi.recharge(payload) : this.$api.walletApi.withdraw(payload))
        if (result.code !== 0) throw Error(result.msg || '订单提交失败')
        const orderID = recharge ? result.data.order_id : result.data.withdrawal_id
        this.$message.success(`${recharge ? '充值' : '提现'}订单 ${recharge ? 'CZ' : 'TX'}-${String(orderID).padStart(8, '0')} 已生成，余额与账变已记录`)
        this.amount = ''
        this.note = ''
        this.bonusMode = 'none'
        this.bonusValue = ''
        this.requestID = imgoMemberFinanceRequestID()
        await this.refresh()
      } catch (error) { this.$message.error(error.message || '提交失败；保留本次请求编号，请核实后重试') }
      finally { this.saving = false }
    },
    eventName(event) {
      return ({ recharge: '充值本金', recharge_bonus: '充值赠送', credit: '人工入账', withdraw: '提现冻结', refund: '提现退回', paid: '提现打款' })[event] || event
    }
  },
  render(h) {
    const member = this.member || {}, recharge = this.mode === 'recharge'
    const field = (value, placeholder, change, type = 'text') => h('el-input', {
      props: { value, type, clearable: true, rows: type === 'textarea' ? 2 : undefined },
      attrs: { placeholder, maxlength: type === 'textarea' ? 500 : undefined },
      on: { input: change }
    })
    const formItem = (label, child) => h('el-form-item', { props: { label, labelWidth: '105px' } }, [child])
    const col = (label, prop, width) => h('el-table-column', { props: { label, prop, width } })
    const history = (title, rows, columns) => h('div', { class: 'imgo-member-finance-history' }, [
      h('h3', title),
      h('el-table', { props: { data: rows, border: true, stripe: true, size: 'mini', maxHeight: 210 }, directives: [{ name: 'loading', value: this.historyLoading }] }, columns)
    ])
    return h('el-dialog', {
      props: { title: `${recharge ? '充值' : '提现'} · ${member.account || ''}（ID ${member.user_id || '—'}）`, visible: this.visible, width: '700px', appendToBody: true, closeOnClickModal: false, closeOnPressEscape: !this.saving, showClose: !this.saving },
      on: { close: this.close }
    }, this.visible ? [
      h('div', { class: 'imgo-member-finance-summary' }, [
        h('span', '可用余额 '), h('strong', this.account ? this.money(this.account.available_cents) : '加载中…'),
        h('span', ' · 冻结中 '), h('strong', this.account ? this.money(this.account.pending_cents) : '—'),
        h('el-button', { props: { type: 'text', loading: this.loading }, on: { click: this.refresh } }, '刷新')
      ]),
      h('el-form', { class: 'imgo-member-finance-form' }, [
        formItem(recharge ? '充值金额' : '提现金额', field(this.amount, '请输入金额（元）', value => this.change('amount', value))),
        recharge ? formItem('赠送方式', h('el-radio-group', { props: { value: this.bonusMode }, on: { input: value => { this.change('bonusMode', value); this.bonusValue = '' } } }, [
          h('el-radio', { props: { label: 'none' } }, '不赠送'),
          h('el-radio', { props: { label: 'percent' } }, '百分比'),
          h('el-radio', { props: { label: 'fixed' } }, '固定金额')
        ])) : null,
        recharge && this.bonusMode !== 'none' ? formItem(this.bonusMode === 'percent' ? '赠送百分比' : '赠送金额', field(this.bonusValue, this.bonusMode === 'percent' ? '如 10 或 2.5（%）' : '固定赠送金额（元）', value => this.change('bonusValue', value))) : null,
        formItem('备注', field(this.note, recharge ? '必填，至少 2 字；记录本次充值原因' : '可选，记录本次提现原因', value => this.change('note', value), 'textarea')),
        h('div', { class: 'imgo-member-finance-preview' }, recharge
          ? `本金 ${this.money(this.amountCents)} + 赠送 ${this.money(this.bonusCents)} = 立即到账 ${this.money(this.amountCents + this.bonusCents)}`
          : `提交后冻结 ${this.money(this.amountCents)}；不会自动向银行卡转账。`)
      ]),
      history('最近充值订单', this.recharges, [
        h('el-table-column', { props: { label: '订单号', width: '145' }, scopedSlots: { default: scope => 'CZ-' + String(scope.row.order_id).padStart(8, '0') } }),
        h('el-table-column', { props: { label: '本金 / 赠送 / 合计', minWidth: '205' }, scopedSlots: { default: scope => `${this.money(scope.row.amount_cents)} / ${this.money(scope.row.bonus_cents)} / ${this.money(scope.row.total_cents)}` } }),
        col('备注', 'note'),
        h('el-table-column', { props: { label: '时间', width: '170' }, scopedSlots: { default: scope => this.date(scope.row.created_at) } })
      ]),
      history('最近提现订单', this.withdrawals, [
        h('el-table-column', { props: { label: '订单号', width: '145' }, scopedSlots: { default: scope => 'TX-' + String(scope.row.withdrawal_id).padStart(8, '0') } }),
        h('el-table-column', { props: { label: '金额', width: '110' }, scopedSlots: { default: scope => this.money(scope.row.amount_cents) } }),
        h('el-table-column', { props: { label: '状态', width: '100' }, scopedSlots: { default: scope => ['待处理', '已打款', '已拒绝'][Number(scope.row.status)] || '未知' } }),
        h('el-table-column', { props: { label: '时间', width: '170' }, scopedSlots: { default: scope => this.date(scope.row.created_at) } })
      ]),
      history('最近余额账变', this.entries, [
        h('el-table-column', { props: { label: '类型', width: '120' }, scopedSlots: { default: scope => this.eventName(scope.row.event) } }),
        h('el-table-column', { props: { label: '可用变化', width: '120' }, scopedSlots: { default: scope => (Number(scope.row.available_delta) >= 0 ? '+' : '-') + this.money(Math.abs(Number(scope.row.available_delta))) } }),
        h('el-table-column', { props: { label: '冻结变化', width: '120' }, scopedSlots: { default: scope => (Number(scope.row.pending_delta) >= 0 ? '+' : '-') + this.money(Math.abs(Number(scope.row.pending_delta))) } }),
        col('备注', 'note'),
        h('el-table-column', { props: { label: '时间', width: '170' }, scopedSlots: { default: scope => this.date(scope.row.created_at) } })
      ]),
      h('span', { slot: 'footer' }, [
        h('el-button', { props: { disabled: this.saving }, on: { click: this.close } }, '关闭'),
        h('el-button', { props: { type: recharge ? 'primary' : 'warning', disabled: !this.account, loading: this.saving }, on: { click: this.submit } }, recharge ? '确认充值并到账' : '确认提现并冻结')
      ])
    ] : [])
  }
}

/* IMGO_REFERRAL_FILTER_BEGIN */
// Vue 2 component embedded in the existing compiled member page.
// The parent owns query state; this toolbar only emits filter changes and searches.
const ImgoMemberReferralFilter = {
  name: 'ImgoMemberReferralFilter',
  props: {
    scope: { type: String, default: '' },
    agentMode: { type: Boolean, default: false },
    referrerAccount: { type: String, default: '' }
  },
  render(h) {
    return h('div', { class: 'imgo-member-referral-filter' }, [
      h('span', { class: 'imgo-member-referral-label' }, '层级'),
      h('el-select', {
        props: { value: this.scope, clearable: !this.agentMode, placeholder: this.agentMode ? '全部下级' : '不选则查用户本人' },
        on: {
          input: value => this.$emit('update:scope', value),
          change: () => this.$emit('search')
        }
      }, [
        h('el-option', { props: { label: '直属下级', value: 'direct' } }),
        h('el-option', { props: { label: '全部下级', value: 'all' } })
      ]),
      h('el-button', {
        props: { type: 'primary', icon: 'el-icon-search' },
        on: { click: () => this.$emit('search') }
      }, '查询')
    ])
  }
};

/* IMGO_REFERRAL_FILTER_END */
s.r(t),s.d(t,{default:function(){return u}});var a=function(){var e=this,t=e._self._c;return t("div",{staticClass:"m-20"},[t("div",{staticClass:"mb-15 lz-flex lz-space-between"},[t("div",[t("el-button",{staticClass:"mr-15",on:{click:e.addUser}},[e._v("添加成员")]),t("imgo-member-referral-filter",{attrs:{scope:e.params.referral_scope,"agent-mode":Number((e.$store.state.userInfo||{}).agent_mode)===1,"referrer-account":e.params.referrer_account},on:{"update:scope":function(t){e.$set(e.params,"referral_scope",t)},"update:referrer-account":function(t){e.$set(e.params,"referrer_account",t)},search:function(){return e.handleChange()}}})],1),t("div",[t("el-input",{staticStyle:{width:"300px"},attrs:{placeholder:"请输入关键字搜索","prefix-icon":"el-icon-search"},nativeOn:{keyup:function(t){return!t.type.indexOf("key")&&e._k(t.keyCode,"enter",13,t.key,"Enter")?null:e.handleChange.apply(null,arguments)}},model:{value:e.params.keywords,callback:function(t){e.$set(e.params,"keywords",t)},expression:"params.keywords"}},[t("el-button",{attrs:{slot:"append",icon:"el-icon-search"},on:{click:e.handleChange},slot:"append"})],1)],1)]),t("el-table",{staticStyle:{width:"100%",border:"solid 1px #e3e3e3"},attrs:{data:e.userList,stripe:"",height:"calc(100vh - 200px)","header-cell-style":{"background-color":"#f5f7fa",color:"#909399"}},on:{"sort-change":e.sortChange,"row-dblclick":e.handleClick}},[t("el-table-column",{attrs:{fixed:"",prop:"user_id",label:"ID",sortable:"custom",width:"52"}}),t("el-table-column",{attrs:{prop:"realname",label:"姓名",width:"78"}}),t("el-table-column",{attrs:{prop:"account",label:"账号",width:"100"}}),t("el-table-column",{attrs:{prop:"sex",label:"性别",sortable:"custom",width:"68"},scopedSlots:e._u([{key:"default",fn:function(s){return[0==s.row.sex?t("span",{staticClass:"el-dropdown-link"},[e._v("女")]):e._e(),1==s.row.sex?t("span",{staticClass:"el-dropdown-link"},[e._v("男")]):e._e(),2==s.row.sex?t("span",{staticClass:"el-dropdown-link"},[e._v("未知")]):e._e()]}}])}),t("el-table-column",{attrs:{label:"角色",width:"132"},scopedSlots:e._u([{key:"default",fn:function(s){return[t("imgo-member-role-select",{attrs:{row:s.row}})]}}])}),t("el-table-column",{attrs:{label:"签到信息",width:"136"},scopedSlots:e._u([{key:"default",fn:function(s){return[t("div",{staticClass:"imgo-checkin-summary"},[t("span",{staticClass:"imgo-checkin-days"},[e._v("累计 "+e._s(s.row.checkin_days||0)+" 天")]),t("span",{staticClass:"imgo-checkin-status",class:s.row.checkin_today?"is-signed":"is-unsigned"},[e._v(s.row.checkin_today?"今日已签到":"今日未签到")])]),t("div",{staticClass:"imgo-checkin-last"},[e._v("最近："+e._s(s.row.checkin_last_date||"—"))]),t("el-button",{staticClass:"imgo-checkin-open",attrs:{type:"text",size:"mini"},on:{click:function(){return e.$refs.checkinHistory.open(s.row)}}},[e._v("查看详情")])]}}])}),t("el-table-column",{attrs:{label:"邀请码",width:"118"},scopedSlots:e._u([{key:"default",fn:function(s){return[t("imgo-invite-code-copy",{attrs:{code:s.row.invite_code}})]}}])}),t("el-table-column",{attrs:{prop:"direct_invite_count",label:"直属下级人数",width:"92"}}),t("el-table-column",{attrs:{prop:"team_count",label:"团队人数",width:"76"}}),t("el-table-column",{attrs:{prop:"create_time",label:"注册时间",width:"132"},scopedSlots:e._u([{key:"default",fn:function(s){return[t("el-popover",{attrs:{placement:"top-start",title:"位置信息",width:"250",trigger:"hover"}},[e._v(" IP: "+e._s(s.row.register_ip)+" "),t("br"),e._v(" 位置："+e._s(s.row.reg_location||"--")+" "),t("span",{attrs:{slot:"reference"},slot:"reference"},[e._v(e._s((v=>!v?'—':/^\d+$/.test(String(v))?new Date(Number(v)*1000).toLocaleString('zh-CN',{timeZone:'Asia/Shanghai',hour12:false}):v)(s.row.create_time)))])])]}}])}),t("el-table-column",{attrs:{prop:"last_login_time",label:"最后登录时间",width:"132"},scopedSlots:e._u([{key:"default",fn:function(s){return[t("el-popover",{attrs:{placement:"top-start",title:"位置信息",width:"250",trigger:"hover"}},[e._v(" IP: "+e._s(s.row.last_login_ip)+" "),t("br"),e._v(" 位置："+e._s(s.row.location||"--")+" "),t("span",{attrs:{slot:"reference"},slot:"reference"},[e._v(e._s((v=>!v?'—':/^\d+$/.test(String(v))?new Date(Number(v)*1000).toLocaleString('zh-CN',{timeZone:'Asia/Shanghai',hour12:false}):v)(s.row.last_login_time)))])])]}}])}),t("el-table-column",{attrs:{prop:"remark",label:"备注","min-width":"120"},scopedSlots:e._u([{key:"default",fn:function(s){return[t("imgo-member-remark",{key:s.row.user_id,attrs:{row:s.row}})]}}])}),t("el-table-column",{attrs:{prop:"status",label:"状态",width:"70"},scopedSlots:e._u([{key:"default",fn:function(s){return[1!=s.row.user_id?t("el-switch",{attrs:{"active-value":1,"inactive-value":0},on:{change:function(t){return e.setStatus(s.row)}},model:{value:s.row.status,callback:function(t){e.$set(s.row,"status",t)},expression:"scope.row.status"}}):t("span",[e._v("--")])]}}])}),t("el-table-column",{attrs:{fixed:"right",label:"操作",width:"164"},scopedSlots:e._u([{key:"default",fn:function(s){return[ t("div",{staticClass:"imgo-member-actions-inline"},[
  t("el-button",{attrs:{type:"text",size:"small",disabled:true,title:"功能暂未开放"}},[e._v("会话")]),
  Number((e.$store.state.userInfo||{}).user_id)===1?t("el-button",{attrs:{type:"text",size:"small"},on:{click:function(){return e.$refs.memberFinance.open(s.row,"recharge")}}},[e._v("充值")]):e._e(),
  Number((e.$store.state.userInfo||{}).user_id)===1?t("el-button",{attrs:{type:"text",size:"small"},on:{click:function(){return e.$refs.memberFinance.open(s.row,"withdraw")}}},[e._v("提现")]):e._e(),
  t("el-dropdown",{attrs:{trigger:"click",placement:"bottom-end"},on:{command:function(command){
   if(command==="dialogue")return e.openDialogue(s.row);
   if(command==="view")return e.handleClick(s.row);
   if(command==="edit")return e.editUser(s.row);
   if(command==="password")return e.editPass(s.row);
   if(command==="inviteCode")return e.$refs.memberInviteCode.open(s.row);
   if(command==="agentSetting"&&Number((e.$store.state.userInfo||{}).user_id)===1&&Number(s.row.admin_role_agent_mode)===1)return e.$refs.memberAgentSetting.open(s.row);
  }}},[
   t("el-button",{attrs:{type:"text",size:"small"}},[e._v("更多"),t("i",{staticClass:"el-icon-arrow-down"})]),
   t("el-dropdown-menu",{slot:"dropdown"},[
    t("el-dropdown-item",{attrs:{command:"dialogue"}},[e._v("会话列表")]),
    t("el-dropdown-item",{attrs:{command:"view"}},[e._v("查看")]),
    s.row.user_id>1?t("el-dropdown-item",{attrs:{command:"edit"}},[e._v("编辑")]):e._e(),
    t("el-dropdown-item",{attrs:{command:"password"}},[e._v("改密")]),
    t("el-dropdown-item",{attrs:{command:"inviteCode"}},[e._v("修改邀请码")]),
    Number((e.$store.state.userInfo||{}).user_id)===1&&Number(s.row.admin_role_agent_mode)===1?t("el-dropdown-item",{attrs:{command:"agentSetting"}},[e._v("导师设置")]):e._e()
   ],1)
  ],1)
 ],1)
]}}])})],1),t("div",{staticClass:"mt-15"},[t("el-pagination",{attrs:{background:"","current-page":e.params.page,"page-sizes":[20,50,100,200,300,400,500],"page-size":e.params.limit,layout:"total, sizes, prev, pager, next, jumper",total:e.total},on:{"size-change":e.handleChange,"current-change":e.getUserList,"update:currentPage":function(t){return e.$set(e.params,"page",t)},"update:current-page":function(t){return e.$set(e.params,"page",t)},"update:pageSize":function(t){return e.$set(e.params,"limit",t)},"update:page-size":function(t){return e.$set(e.params,"limit",t)}}})],1),t("imgo-check-in-history",{ref:"checkinHistory"}),t("imgo-member-finance-dialog",{ref:"memberFinance"}),t("imgo-member-invite-code-dialog",{ref:"memberInviteCode"}),t("imgo-member-agent-setting-dialog",{ref:"memberAgentSetting",on:{saved:e.handleChange}}),t("el-dialog",{attrs:{title:e.currentUser.realname+" 的会话管理",visible:e.dialogueBox,modal:!0,width:"80%","append-to-body":""},on:{"update:visible":function(t){e.dialogueBox=t},close:function(t){e.dialogueBox=!1}}},[t("dialogue",{key:e.componentKey,attrs:{userInfo:e.currentUser}})],1),t("el-dialog",{attrs:{title:e.formTitle,visible:e.dialogVisible,modal:!0,width:"500px","append-to-body":""},on:{"update:visible":function(t){e.dialogVisible=t},close:function(t){e.dialogVisible=!1}}},[t("el-form",{ref:"userinfo",attrs:{model:e.detail,rules:e.rules,"label-width":"100px"}},[t("el-form-item",{attrs:{label:"账号",prop:"account"}},[t("el-input",{attrs:{placeholder:"请输入邮箱或者手机号"},model:{value:e.detail.account,callback:function(t){e.$set(e.detail,"account",t)},expression:"detail.account"}})],1),t("el-form-item",{directives:[{name:"show",rawName:"v-show",value:"add"==e.formType,expression:"formType=='add'"}],attrs:{label:"密码",prop:"password"}},[t("el-input",{attrs:{"show-password":"",placeholder:"请输入密码"},model:{value:e.detail.password,callback:function(t){e.$set(e.detail,"password",t)},expression:"detail.password"}})],1),t("el-form-item",{attrs:{label:"姓名",prop:"realname"}},[t("el-input",{attrs:{placeholder:"请输入用户名称"},model:{value:e.detail.realname,callback:function(t){e.$set(e.detail,"realname",t)},expression:"detail.realname"}})],1),t("el-form-item",{attrs:{label:"e-mail",prop:"email"}},[t("el-input",{attrs:{placeholder:"请输入邮箱地址"},model:{value:e.detail.email,callback:function(t){e.$set(e.detail,"email",t)},expression:"detail.email"}})],1),t("el-form-item",{attrs:{label:"专属客服",prop:"cs_uid"}},[t("user-select",{attrs:{width:"180px",radio:!0},model:{value:e.detail.cs_uid,callback:function(t){e.$set(e.detail,"cs_uid",t)},expression:"detail.cs_uid"}})],1),t("el-form-item",{attrs:{label:"性别",prop:"sex"}},[t("el-radio-group",{model:{value:e.detail.sex,callback:function(t){e.$set(e.detail,"sex",t)},expression:"detail.sex"}},[t("el-radio",{attrs:{label:2,border:""}},[e._v("未知")]),t("el-radio",{attrs:{label:1,border:""}},[e._v("男")]),t("el-radio",{attrs:{label:0,border:""}},[e._v("女")])],1)],1),t("el-form-item",{attrs:{label:"状态",prop:"status"}},[t("el-radio-group",{model:{value:e.detail.status,callback:function(t){e.$set(e.detail,"status",t)},expression:"detail.status"}},[t("el-radio",{attrs:{label:1,border:""}},[e._v("正常")]),t("el-radio",{attrs:{label:0,border:""}},[e._v("禁用")])],1)],1),t("el-form-item",{attrs:{label:"好友上限",prop:"friend_limit"}},[t("el-input-number",{staticClass:"ml-10",attrs:{min:-1,max:1e3},model:{value:e.detail.friend_limit,callback:function(t){e.$set(e.detail,"friend_limit",t)},expression:"detail.friend_limit"}}),t("span",{staticClass:"ml-10 c-999 f-12"},[e._v("个，0表示不限制，-1表示禁止创建")])],1),t("el-form-item",{attrs:{label:"群聊上限",prop:"group_limit"}},[t("el-input-number",{staticClass:"ml-10",attrs:{min:-1,max:1e3},model:{value:e.detail.group_limit,callback:function(t){e.$set(e.detail,"group_limit",t)},expression:"detail.group_limit"}}),t("span",{staticClass:"ml-10 c-999 f-12"},[e._v("个，0表示不限制-1表示禁止创建")])],1),"edit"==e.formType?t("el-form-item",{attrs:{label:"签到信息"}},[t("div",{staticClass:"imgo-checkin-detail"},[e._v("累计 "+e._s(e.detail.checkin_days||0)+" 天 · "+(e.detail.checkin_today?"今日已签到":"今日未签到"))]),t("div",{staticClass:"imgo-checkin-last"},[e._v("最近签到："+e._s(e.detail.checkin_last_date||"—"))])]):e._e(),"edit"==e.formType?t("el-form-item",{attrs:{label:"邀请关系"}},[t("div",{staticClass:"imgo-referral-detail"},[e._v("邀请码："+e._s(e.detail.invite_code||"—")+" · 直属下级："+e._s(e.detail.direct_invite_count||0)+" 人 · 团队："+e._s(e.detail.team_count||0)+" 人")])]):e._e(),t("el-form-item",{attrs:{label:"备注",prop:"remark"}},[t("el-input",{attrs:{type:"textarea",rows:2},model:{value:e.detail.remark,callback:function(t){e.$set(e.detail,"remark",t)},expression:"detail.remark"}})],1),t("el-form-item",[t("el-button",{attrs:{type:"primary"},on:{click:function(t){return e.submitForm("userinfo")}}},[e._v("保存")])],1)],1)],1),t("el-dialog",{attrs:{title:"修改密码",visible:e.dialogPass,modal:!0,width:"400px","append-to-body":""},on:{"update:visible":function(t){e.dialogPass=t}}},[t("el-form",{attrs:{"label-width":"100px"}},[t("el-form-item",{attrs:{label:"新密码"}},[t("el-input",{attrs:{"show-password":"",placeholder:"请输入密码"},model:{value:e.password,callback:function(t){e.password=t},expression:"password"}})],1),t("el-form-item",{attrs:{label:"重复密码"}},[t("el-input",{attrs:{"show-password":"",placeholder:"请输入重复输入密码"},model:{value:e.repass,callback:function(t){e.repass=t},expression:"repass"}})],1),t("el-form-item",[t("el-button",{attrs:{type:"primary"},on:{click:function(t){return e.editPassword()}}},[e._v("保存")])],1)],1)],1)],1)},r=[],i=(s(7658),s(3822)),l=s(6647),o=s(7076),n={components:{userSelect:l.Z,dialogue:o.Z,ImgoCheckInHistory,ImgoMemberFinanceDialog,ImgoMemberReferralFilter,ImgoMemberRemark,ImgoMemberInviteCodeDialog,ImgoInviteCodeCopy,ImgoMemberRoleSelect,ImgoMemberAgentSettingDialog},data(){return{componentKey:0,total:0,params:{page:1,limit:20,keywords:"",order_field:"",order_type:1,referral_scope:Number((this.$store.state.userInfo||{}).agent_mode)===1?"all":"",referrer_account:""},userList:[],formTitle:"添加成员",formType:"add",dialogVisible:!1,dialogueBox:!1,detail:{},originDetail:{realname:"",password:"123456",email:"",sex:2,role:0,friend_limit:0,group_limit:0,remark:"",status:1},rules:{account:[{min:4,max:32,message:"长度在 4 到 32 个字符",trigger:"blur"}],realname:[{required:!0,message:"请输入用户名称",trigger:"blur"},{min:2,max:16,message:"长度在 2 到 16 个字符",trigger:"blur"}],email:[{type:"email",message:"请输入正确的邮箱地址",trigger:["blur","change"]}],password:[{required:!0,message:"请输入密码",trigger:"blur"},{min:6,max:30,message:"长度在 6 到 30 个字符",trigger:"blur"}]},dialogPass:!1,password:"",repass:"",currentUser:{}}},computed:{...(0,i.rn)({globalConfig:e=>e.globalConfig})},watch:{dialogVisible(e){e||(this.detail=this.originDetail)}},mounted(){this.detail=this.originDetail,this.getUserList();let e=this.globalConfig.sysInfo.regauth??0,t="请输入账号：4-32个字符";switch(parseInt(e)){case 1:t="请输入正确的手机号";break;case 2:t="请输入正确的邮箱";break;case 3:t="请输入正确的手机号或者邮箱";break;default:t="请输入正确的账号";break}let s={required:!0,message:t,trigger:"blur"};this.rules.account.push(s);let a={type:"email",message:t,trigger:"blur",validator:this.validateContact},r={type:"phone",message:t,trigger:"blur",validator:this.validateContact};1==e?this.rules.account.push(r):2==e?this.rules.account.push(a):3==e&&(this.rules.account.push(a),this.rules.account.push(r))},methods:{getUserList(){this.$api.userApi.getUserList(this.params).then((e=>{0==e.code&&(this.userList=e.data,this.total=e.count,this.params.page=e.page)}))},sortChange(e){this.params.order_field=e.prop,null==e.order&&(this.params.order_field=null),this.params.order_type="ascending"==e.order?1:2,this.getUserList()},handleClick(e){this.$user(e.user_id,{isManage:!0,editDataCallbak:e=>{this.editUser(e)}})},openDialogue(e){this.currentUser=e,this.componentKey++,this.dialogueBox=!0},handleChange(){this.params.page=1,this.getUserList()},addUser(){this.formTitle="添加成员",this.formType="add",this.dialogVisible=!0},editUser(e){let t=e;this.formTitle="修改成员",this.formType="edit",this.dialogVisible=!0,t.password="rainagd",this.detail=t},validateContact(e,t,s){t?/^1[3456789]\d{9}$/.test(t)||/^\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*$/.test(t)?s():s(new Error("请输入正确的手机号或邮箱")):s()},submitForm(e){this.$refs[e].validate((e=>{if(!e)return console.log("error submit!!"),!1;if("add"==this.formType)this.$api.userApi.addUser(this.detail).then((e=>{0==e.code&&(this.dialogVisible=!1,this.getUserList(),this.$message({message:e.msg,type:"success"}))}));else{let e=JSON.parse(JSON.stringify(this.detail));delete e.password,this.$api.userApi.editUser(e).then((e=>{0==e.code&&(this.dialogVisible=!1,this.getUserList(),this.$message({message:e.msg,type:"success"}))}))}}))},editPass(e){this.currentUser=e,this.dialogPass=!0},editPassword(){if(""==this.password||this.password.length<6||this.password.length>30)return this.$message({message:"请输入6-30位密码",type:"warning"}),!1;if(this.password!=this.repass)return this.$message({message:"两次密码不一致",type:"warning"}),!1;let e={user_id:this.currentUser.user_id,password:this.password};this.$api.userApi.editPassword(e).then((e=>{0==e.code&&(this.dialogPass=!1,this.password="",this.repass="",this.$message({message:e.msg,type:"success"}))}))},setStatus(e){let t={user_id:e.user_id,status:e.status};this.$api.userApi.setStatus(t).then((e=>{0==e.code&&this.$message({message:e.msg,type:"success"})}))},handleCommand(e,t){let s={user_id:e.user_id,role:t};this.$api.userApi.setRole(s).then((s=>{0==s.code&&(e.role=t,this.$message({message:s.msg,type:"success"}))}))},delUser(e){this.$confirm("此操作将永久删除该用户, 是否继续?","提示",{confirmButtonText:"确定",cancelButtonText:"取消",type:"warning"}).then((()=>{let t={user_id:e.user_id};this.$api.userApi.delUser(t).then((e=>{0==e.code&&(this.$message({message:e.msg,type:"success"}),this.getUserList())}))})).catch((()=>{this.$message({type:"info",message:"已取消删除"})}))}}},d=n,c=s(1001),p=(0,c.Z)(d,a,r,!1,null,"e0b9ac40",null),u=p.exports}}]);