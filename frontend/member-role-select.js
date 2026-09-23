const ImgoMemberRoleSelect = {
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
        return Array.isArray(result.data) ? result.data : []
      }).catch(error => { window.__imgoRoleOptionsPromise = null; throw error })
    }
    window.__imgoRoleOptionsPromise.then(roles => { this.roles = roles }).catch(error => this.$message.error(error.message || '读取角色失败'))
  },
  methods: {
    async change(adminRoleID) {
      if (this.saving || Number(adminRoleID) === this.roleValue) return
      const previousID = this.roleValue
      const previousName = this.roleLabel
      const selected = this.roles.find(role => Number(role.role_id) === Number(adminRoleID))
      this.$set(this.row, 'admin_role_id', Number(adminRoleID))
      this.$set(this.row, 'admin_role_name', selected ? selected.name : '普通用户')
      this.saving = true
      try {
        const result = await this.$api.userApi.setRole({ user_id: this.row.user_id, admin_role_id: Number(adminRoleID) })
        if (result.code !== 0) throw Error(result.msg || '角色设置失败')
        this.$message.success('角色已更新')
      } catch (error) {
        this.$set(this.row, 'admin_role_id', previousID)
        this.$set(this.row, 'admin_role_name', previousName)
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
