const ImgoRolePanel = {
  name: 'ImgoRolePanel',
  data() {
    return {
      roles: [], permissions: [], activeRoleID: null, loading: false, saving: false,
      form: { role_id: 0, name: '', remark: '', status: 1, permissions: [], builtin: false }
    }
  },
  mounted() { this.load() },
  methods: {
    async load(preferredID) {
      this.loading = true
      try {
        const [rolesResult, permissionsResult] = await Promise.all([
          this.$api.roleApi.index({}), this.$api.roleApi.permissions({})
        ])
        if (rolesResult.code !== 0) throw Error(rolesResult.msg || '读取角色失败')
        if (permissionsResult.code !== 0) throw Error(permissionsResult.msg || '读取权限失败')
        this.roles = Array.isArray(rolesResult.data) ? rolesResult.data : []
        this.permissions = Array.isArray(permissionsResult.data) ? permissionsResult.data : []
        const wantedID = preferredID !== undefined && preferredID !== null ? Number(preferredID) : this.activeRoleID
        const selected = wantedID !== null ? this.roles.find(role => Number(role.role_id) === Number(wantedID)) : null
        if (selected) this.select(selected)
        else if (this.roles.length) this.select(this.roles[0])
        else this.create()
      } catch (error) { this.$message.error(error.message || '读取角色失败') }
      finally { this.loading = false }
    },
    create() {
      this.activeRoleID = null
      this.form = { role_id: 0, name: '', remark: '', status: 1, permissions: [], builtin: false }
    },
    select(role) {
      this.activeRoleID = Number(role.role_id)
      this.form = {
        role_id: Number(role.role_id), name: role.name || '', remark: role.remark || '',
        status: Number(role.status), permissions: Array.isArray(role.permissions) ? [...role.permissions] : [],
        builtin: Boolean(role.builtin)
      }
    },
    async save() {
      if (this.form.builtin) return
      const name = this.form.name.trim()
      if (!name || name.length > 64) { this.$message.warning('请输入1-64个字的角色名称'); return }
      if (this.form.remark.length > 255) { this.$message.warning('角色备注最多255个字'); return }
      this.saving = true
      try {
        const result = await this.$api.roleApi.save({ ...this.form, name, remark: this.form.remark.trim() })
        if (result.code !== 0) throw Error(result.msg || '保存角色失败')
        this.$message.success('角色已保存')
        window.__imgoRoleOptionsPromise = null
        await this.load(Number(result.data.role_id))
      } catch (error) { this.$message.error(error.message || '保存角色失败') }
      finally { this.saving = false }
    },
    async remove() {
      if (!this.form.role_id || this.form.builtin) return
      try {
        await this.$confirm('确定删除该角色？已绑定用户的角色不能删除。', '删除角色', { type: 'warning' })
        const result = await this.$api.roleApi.del({ role_id: this.form.role_id })
        if (result.code !== 0) throw Error(result.msg || '删除角色失败')
        this.$message.success('角色已删除')
        window.__imgoRoleOptionsPromise = null
        this.activeRoleID = null
        await this.load()
      } catch (error) {
        if (error !== 'cancel' && error !== 'close') this.$message.error(error.message || '删除角色失败')
      }
    }
  },
  render(h) {
    const roleCards = this.roles.map(role => h('button', {
      key: role.role_id,
      class: ['imgo-role-card', { 'is-active': Number(role.role_id) === this.activeRoleID }],
      attrs: { type: 'button' }, on: { click: () => this.select(role) }
    }, [
      h('span', { class: 'imgo-role-card-name' }, role.name),
      h('span', { class: 'imgo-role-card-count' }, `${Number(role.user_count || 0)} 人`),
      h('span', { class: ['imgo-role-card-status', Number(role.status) === 1 ? 'is-enabled' : 'is-disabled'] }, role.builtin ? '系统角色' : (Number(role.status) === 1 ? '已启用' : '已禁用'))
    ]))
    const permissionBoxes = this.permissions.map(permission => h('el-checkbox', {
      key: permission.permission_key, props: { label: permission.permission_key }
    }, [h('span', { class: 'imgo-role-permission-name' }, permission.name), h('small', permission.menu_path)]))
    return h('section', { class: 'imgo-role-page' }, [
      h('header', { class: 'imgo-role-heading' }, [
        h('div', [h('h1', '角色'), h('p', '为后台人员分配一个角色，并同时限制菜单和接口权限。')]),
        h('el-button', { props: { type: 'primary', icon: 'el-icon-plus' }, on: { click: this.create } }, '新增角色')
      ]),
      h('div', { class: 'imgo-role-layout', directives: [{ name: 'loading', value: this.loading }] }, [
        h('aside', { class: 'imgo-role-list' }, roleCards.length ? roleCards : [h('div', { class: 'imgo-role-empty' }, '暂无角色')]),
        h('main', { class: 'imgo-role-editor' }, [
          h('div', { class: 'imgo-role-editor-title' }, this.form.builtin ? this.form.name : (this.form.role_id ? '编辑角色' : '新增角色')),
          h('el-form', { props: { labelWidth: '92px' } }, [
            h('el-form-item', { props: { label: '角色名称', required: !this.form.builtin } }, [h('el-input', { props: { value: this.form.name, disabled: this.form.builtin, maxlength: 64, showWordLimit: !this.form.builtin, placeholder: '例如：客服、群聊管理员' }, on: { input: value => { this.form.name = value } } })]),
            h('el-form-item', { props: { label: '角色状态' } }, [h('el-switch', { props: { value: this.form.status, disabled: this.form.builtin, activeValue: 1, inactiveValue: 0, activeText: '启用', inactiveText: '禁用' }, on: { input: value => { this.form.status = Number(value) } } })]),
            h('el-form-item', { props: { label: '角色备注' } }, [h('el-input', { props: { value: this.form.remark, disabled: this.form.builtin, type: 'textarea', rows: 3, maxlength: 255, showWordLimit: !this.form.builtin, placeholder: '说明该角色的使用范围' }, on: { input: value => { this.form.remark = value } } })]),
            h('el-form-item', { props: { label: '菜单权限' } }, [h('el-checkbox-group', { class: 'imgo-role-permissions', props: { value: this.form.permissions, disabled: this.form.builtin }, on: { input: value => { this.form.permissions = value } } }, permissionBoxes)])
          ]),
          this.form.builtin ? h('div', { class: 'imgo-role-system-note' }, '系统角色为固定权限，不可编辑或删除。') : h('div', { class: 'imgo-role-actions' }, [
            this.form.role_id ? h('el-button', { props: { type: 'danger', plain: true }, on: { click: this.remove } }, '删除角色') : null,
            h('el-button', { props: { type: 'primary', loading: this.saving }, on: { click: this.save } }, '保存角色')
          ])
        ])
      ])
    ])
  }
}
