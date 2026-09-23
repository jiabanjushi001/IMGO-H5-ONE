// Small Vue 2 dialog owned by the existing member list's More menu.
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
