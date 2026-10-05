// Vue 2 component injected into the existing management application.
// Full identity numbers and document images are fetched only for one authorized detail view.
const ImgoIdentityPanel = {
  name: 'ImgoIdentityPanel',
  data() {
    return {
      rows: [], total: 0, page: 1, keywords: '', filter: '',
      loading: false, reviewing: false, dialog: false,
      detail: null, remark: ''
    }
  },
  mounted() { this.refresh() },
  methods: {
    async refresh() {
      this.loading = true
      try {
        const res = await this.$api.identityApi.index({ page: this.page, limit: 20, keywords: this.keywords.trim(), status: this.filter })
        if (res.code !== 0) throw Error(res.msg || '读取实名认证列表失败')
        this.rows = Array.isArray(res.data) ? res.data : []
        this.total = Number(res.count || 0)
      } catch (error) { this.$message.error(error.message || '读取实名认证列表失败') }
      finally { this.loading = false }
    },
    search() { this.page = 1; this.refresh() },
    async open(row) {
      try {
        const res = await this.$api.identityApi.detail({ user_id: row.user_id })
        if (res.code !== 0) throw Error(res.msg || '读取实名认证资料失败')
        this.detail = res.data || null
        this.remark = this.detail && this.detail.remark || ''
        this.dialog = true
      } catch (error) { this.$message.error(error.message || '读取实名认证资料失败') }
    },
    async review(status) {
      if (!this.detail || this.reviewing) return
      if (status === 2 && !this.remark.trim()) {
        this.$message.warning('拒绝时请填写原因')
        return
      }
      const label = status === 1 ? '同意' : '拒绝'
      try {
        await this.$confirm(`确定${label}该用户的实名认证吗？`, '实名认证审核', { type: status === 1 ? 'success' : 'warning' })
      } catch (error) { return }
      this.reviewing = true
      try {
        const res = await this.$api.identityApi.review({
          user_id: this.detail.user_id,
          version: this.detail.version,
          status,
          remark: this.remark.trim()
        })
        if (res.code !== 0) throw Error(res.msg || '审核失败')
        this.dialog = false
        this.$message.success(`已${label}`)
        await this.refresh()
      } catch (error) { this.$message.error(error.message || '审核失败') }
      finally { this.reviewing = false }
    },
    statusLabel(value) { return ['未处理', '已同意', '已拒绝'][Number(value)] || '未知' },
    tagType(value) { return ['warning', 'success', 'danger'][Number(value)] || 'info' },
    date(value) { return value ? new Date(Number(value) * 1000).toLocaleString('zh-CN', { hour12: false }) : '—' }
  },
  render(h) {
    const column = (label, prop, width) => h('el-table-column', { props: { label, prop, width } })
    const detail = this.detail
    const info = (label, value) => h('div', { class: 'imgo-identity-info-row' }, [h('span', label), h('strong', value || '—')])
    const image = (label, src) => h('div', { class: 'imgo-identity-document' }, [
      h('div', { class: 'imgo-identity-document-title' }, label),
      h('el-image', { props: { src, fit: 'cover', previewSrcList: src ? [src] : [] } }, [h('div', { slot: 'error', class: 'imgo-identity-image-error' }, '图片读取失败')])
    ])
    const dialog = h('el-dialog', {
      props: { title: '实名认证审核', visible: this.dialog, width: '760px', closeOnClickModal: false, appendToBody: true },
      on: { close: () => { this.dialog = false; this.detail = null; this.remark = '' } }
    }, this.dialog && detail ? [
      h('div', { class: 'imgo-identity-detail' }, [
        h('div', { class: 'imgo-identity-info-grid' }, [
          info('账号', detail.login_account), info('昵称', detail.user_name),
          info('真实姓名', detail.real_name), info('身份证号', detail.id_number),
          info('提交时间', this.date(detail.submitted_at)),
          h('div', { class: 'imgo-identity-info-row' }, [h('span', '当前状态'), h('el-tag', { props: { type: this.tagType(detail.status) } }, this.statusLabel(detail.status))])
        ]),
        h('div', { class: 'imgo-identity-documents' }, [image('身份证人像面', detail.front_image), image('身份证国徽面', detail.back_image)]),
        h('el-form', { class: 'imgo-identity-review-form' }, [
          h('el-form-item', { props: { label: '审核备注', labelWidth: '90px' } }, [
            h('el-input', { props: { value: this.remark, type: 'textarea', rows: 3, maxlength: 500, showWordLimit: true, placeholder: '拒绝时必须填写原因，用户可见' }, on: { input: value => { this.remark = value } } })
          ])
        ])
      ]),
      h('span', { slot: 'footer' }, Number(detail.status) === 0 ? [
        h('el-button', { props: { loading: this.reviewing, type: 'danger', plain: true }, on: { click: () => this.review(2) } }, '拒绝'),
        h('el-button', { props: { loading: this.reviewing, type: 'success' }, on: { click: () => this.review(1) } }, '同意实名')
      ] : [h('el-button', { on: { click: () => { this.dialog = false } } }, '关闭')])
    ] : [])

    return h('section', { class: 'imgo-identity' }, [
      h('div', { class: 'imgo-identity-heading' }, [
        h('div', [h('h1', '实名认证'), h('p', '核对用户姓名、身份证号和证件正反面，审核结果会同步到用户端')]),
        h('el-button', { props: { icon: 'el-icon-refresh', loading: this.loading }, on: { click: this.refresh } }, '刷新')
      ]),
      h('div', { class: 'imgo-identity-toolbar' }, [
        h('el-input', { props: { value: this.keywords, placeholder: '搜索账号、昵称、姓名或完整身份证号', clearable: true }, on: { input: value => { this.keywords = value }, keyup: event => { if (event.key === 'Enter') this.search() } } }),
        h('el-select', { props: { value: this.filter, placeholder: '全部状态' }, on: { input: value => { this.filter = value; this.search() } } }, [
          h('el-option', { props: { label: '全部状态', value: '' } }),
          [0, 1, 2].map(value => h('el-option', { key: value, props: { label: this.statusLabel(value), value: String(value) } }))
        ]),
        h('el-button', { props: { type: 'primary' }, on: { click: this.search } }, '查询')
      ]),
      h('el-table', { props: { data: this.rows, border: true, stripe: true } }, [
        column('用户ID', 'user_id', '82'),
        h('el-table-column', { props: { label: '账号/昵称', width: '180' }, scopedSlots: { default: scope => h('div', { class: 'imgo-identity-user' }, [
          h('div', { class: 'imgo-identity-user-account' }, scope.row.login_account || '—'),
          h('div', { class: 'imgo-identity-user-name' }, scope.row.user_name || '—')
        ]) } }),
        column('真实姓名', 'real_name', '140'),
        column('身份证号', 'id_number_masked', '205'),
        h('el-table-column', { props: { label: '状态', width: '100' }, scopedSlots: { default: scope => h('el-tag', { props: { type: this.tagType(scope.row.status) } }, this.statusLabel(scope.row.status)) } }),
        column('审核备注', 'remark'),
        h('el-table-column', { props: { label: '提交时间', width: '180' }, scopedSlots: { default: scope => this.date(scope.row.submitted_at) } }),
        h('el-table-column', { props: { label: '操作', width: '90', fixed: 'right', align: 'center' }, scopedSlots: { default: scope => h('el-button', { props: { type: 'text' }, on: { click: () => this.open(scope.row) } }, Number(scope.row.status) === 0 ? '审核' : '查看') } })
      ]),
      h('el-pagination', { class: 'imgo-identity-pagination', props: { currentPage: this.page, pageSize: 20, total: this.total, layout: 'total, prev, pager, next' }, on: { 'current-change': page => { this.page = page; this.refresh() } } }),
      dialog
    ])
  }
}
