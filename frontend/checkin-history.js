// Read-only Vue 2 dialog for the existing compiled member page.
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
