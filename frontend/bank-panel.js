// Vue 2 component for the existing compiled management application.
// The authorized management detail supplies the full card number for editing.
const ImgoBankPanel = {
  name: 'ImgoBankPanel',
  data() {
    return {
      rows: [], total: 0, page: 1, keywords: '', filter: '', typeFilter: '',
      loading: false, saving: false, dialog: false,
      draft: { user_id: 0, payment_type: 'bank', version: 0, receipt_name: '', receipt_account: '', original_account: '', bank_name: '', branch_name: '', qr_image: '', status: 0, remark: '' }
    }
  },
  mounted() { this.refresh() },
  methods: {
    async refresh() {
      this.loading = true
      try {
        const res = await this.$api.bankApi.index({ page: this.page, limit: 20, keywords: this.keywords.trim(), status: this.filter, payment_type: this.typeFilter })
        if (res.code !== 0) throw Error(res.msg || '读取收款审核列表失败')
        this.rows = Array.isArray(res.data) ? res.data : []
        this.total = Number(res.count || 0)
      } catch (error) {
        this.$message.error(error.message || '读取收款审核列表失败')
      } finally { this.loading = false }
    },
    search() { this.page = 1; this.refresh() },
    async open(row) {
      try {
        const type = row.payment_type || 'bank'
        const res = await this.$api.bankApi.detail({ user_id: row.user_id, payment_type: type })
        if (res.code !== 0) throw Error(res.msg || '读取收款资料失败')
        const card = res.data
        this.draft = {
          user_id: Number(card.user_id), payment_type: card.payment_type || type, version: Number(card.version),
          receipt_name: card.receipt_name || '', receipt_account: card.receipt_account || '', original_account: card.receipt_account || '',
          bank_name: card.bank_name || '', branch_name: card.branch_name || '', qr_image: card.qr_image || '',
          status: Number(card.status), remark: card.remark || ''
        }
        this.dialog = true
      } catch (error) { this.$message.error(error.message || '读取收款资料失败') }
    },
    async save() {
      const d = this.draft, account = d.receipt_account.replace(/\s/g, '')
      const invalidBank = d.payment_type === 'bank' && (!/^\d{12,30}$/.test(account) ||
        (d.bank_name.trim() && (d.bank_name.trim().length < 2 || d.bank_name.trim().length > 120)) ||
        (d.branch_name.trim() && (d.branch_name.trim().length < 2 || d.branch_name.trim().length > 120)))
      if (d.receipt_name.trim().length < 2 || invalidBank || d.remark.length > 500) {
        this.$message.warning('请检查收款姓名、收款账号和备注')
        return
      }
      this.saving = true
      try {
        const payload = {
          user_id: d.user_id, payment_type: d.payment_type, version: d.version,
          receipt_name: d.receipt_name.trim(),
          status: d.status, remark: d.remark.trim()
        }
        if (d.payment_type === 'bank') {
          payload.bank_name = d.bank_name.trim(); payload.branch_name = d.branch_name.trim()
        }
        if (d.payment_type === 'bank' && account !== d.original_account) payload.receipt_account = account
        const res = await this.$api.bankApi.edit(payload)
        if (res.code !== 0) throw Error(res.msg || '保存失败')
        this.dialog = false
        this.$message.success('收款资料已更新')
        await this.refresh()
      } catch (error) { this.$message.error(error.message || '保存失败') }
      finally { this.saving = false }
    },
    statusLabel(value) { return ['未处理', '同意', '拒绝'][Number(value)] || '未知' },
    typeLabel(value) { return { bank: '银行卡', alipay: '支付宝', wechat: '微信' }[value] || value },
    date(value) { return value ? new Date(Number(value) * 1000).toLocaleString('zh-CN', { hour12: false }) : '—' }
  },
  render(h) {
    const field = (label, key, options = {}) => h('el-form-item', { props: { label, labelWidth: '100px' } }, [
      h('el-input', { props: { value: this.draft[key], placeholder: options.placeholder || '', type: options.type || 'text', maxlength: options.maxlength, showWordLimit: !!options.maxlength }, on: { input: value => { this.draft[key] = value } } })
    ])
    const column = (label, prop, width) => h('el-table-column', { props: { label, prop, width } })
    const d = this.draft
    const formItems = [
      h('el-form-item', { props: { label: '用户ID', labelWidth: '100px' } }, String(d.user_id)),
      h('el-form-item', { props: { label: '收款类型', labelWidth: '100px' } }, this.typeLabel(d.payment_type)),
      field('收款姓名', 'receipt_name', { placeholder: '收款人姓名', maxlength: 40 })
    ]
    if (d.payment_type === 'bank') formItems.push(field('卡号', 'receipt_account', { placeholder: '请输入收款账号', maxlength: 100 }))
    if (d.payment_type === 'bank') {
      formItems.push(field('收款银行', 'bank_name', { placeholder: '请输入开户行', maxlength: 120 }))
      formItems.push(field('支行名称', 'branch_name', { placeholder: '请输入支行名称', maxlength: 120 }))
    }
    if (d.payment_type === 'wechat' || d.payment_type === 'alipay') formItems.push(h('el-form-item', { props: { label: d.payment_type === 'alipay' ? '支付宝收款码' : '微信收款码', labelWidth: '110px' } }, [
      d.qr_image ? h('img', { class: 'imgo-bank-qr', attrs: { src: d.qr_image, alt: d.payment_type === 'alipay' ? '支付宝收款码' : '微信收款码' } }) : h('span', { class: 'imgo-bank-empty' }, '图片不可用')
    ]))
    formItems.push(h('el-form-item', { props: { label: '状态', labelWidth: '100px' } }, [
      h('el-radio-group', { props: { value: d.status }, on: { input: value => { d.status = Number(value) } } },
        [0, 1, 2].map(value => h('el-radio', { key: value, props: { label: value } }, this.statusLabel(value))))
    ]))
    formItems.push(field('备注', 'remark', { type: 'textarea', maxlength: 500, placeholder: '处理说明（用户可见）' }))
    const dialog = h('el-dialog', {
      props: { title: '审核收款资料', visible: this.dialog, width: '520px', closeOnClickModal: false },
      on: { close: () => { this.dialog = false; this.draft.receipt_account = ''; this.draft.original_account = '' } }
    }, this.dialog ? [
      h('el-form', { class: 'imgo-bank-form' }, formItems),
      h('span', { slot: 'footer' }, [
        h('el-button', { on: { click: () => { this.dialog = false } } }, '取消'),
        h('el-button', { props: { type: 'primary', loading: this.saving }, on: { click: this.save } }, '保存')
      ])
    ] : [])
    return h('section', { class: 'imgo-bank' }, [
      h('div', { class: 'imgo-bank-heading' }, [
        h('div', [h('h1', '收款审核'), h('p', '统一审核银行卡、支付宝和微信收款码')]),
        h('el-button', { props: { icon: 'el-icon-refresh', loading: this.loading }, on: { click: this.refresh } }, '刷新')
      ]),
      h('div', { class: 'imgo-bank-toolbar' }, [
        h('el-input', { props: { value: this.keywords, placeholder: '搜索账号、昵称或收款姓名', clearable: true }, on: { input: value => { this.keywords = value }, keyup: event => { if (event.key === 'Enter') this.search() } } }),
        h('el-select', { props: { value: this.typeFilter, placeholder: '全部收款类型' }, on: { input: value => { this.typeFilter = value; this.search() } } }, [
          h('el-option', { props: { label: '全部收款类型', value: '' } }),
          ['bank', 'alipay', 'wechat'].map(value => h('el-option', { key: value, props: { label: this.typeLabel(value), value } }))
        ]),
        h('el-select', { props: { value: this.filter, placeholder: '全部状态' }, on: { input: value => { this.filter = value; this.search() } } }, [
          h('el-option', { props: { label: '全部状态', value: '' } }),
          [0, 1, 2].map(value => h('el-option', { props: { label: this.statusLabel(value), value: String(value), key: value } }))
        ]),
        h('el-button', { props: { type: 'primary' }, on: { click: this.search } }, '查询')
      ]),
      h('el-table', { props: { data: this.rows, border: true, stripe: true, vLoading: this.loading } }, [
        column('用户ID', 'user_id', '85'),
        h('el-table-column', { props: { label: '账号/昵称', width: '180' }, scopedSlots: { default: scope => h('div', { class: 'imgo-bank-user' }, [
          h('div', { class: 'imgo-bank-user-account' }, scope.row.login_account || '—'),
          h('div', { class: 'imgo-bank-user-name' }, scope.row.user_name || '—')
        ]) } }),
        h('el-table-column', { props: { label: '收款类型', width: '105' }, scopedSlots: { default: scope => this.typeLabel(scope.row.payment_type) } }),
        column('收款姓名', 'receipt_name', '140'), column('收款账号', 'receipt_account_masked', '180'),
        column('收款银行', 'bank_name', '180'), column('支行名称', 'branch_name', '220'),
        h('el-table-column', { props: { label: '状态', width: '100' }, scopedSlots: { default: scope => h('el-tag', { props: { type: ['warning', 'success', 'danger'][Number(scope.row.status)] } }, this.statusLabel(scope.row.status)) } }),
        column('备注', 'remark'),
        h('el-table-column', { props: { label: '更新时间', width: '180' }, scopedSlots: { default: scope => this.date(scope.row.updated_at) } }),
        h('el-table-column', { props: { label: '操作', width: '85', fixed: 'right', align: 'center' }, scopedSlots: { default: scope => h('el-button', { props: { type: 'text' }, on: { click: () => this.open(scope.row) } }, '审核') } })
      ]),
      h('el-pagination', { class: 'imgo-bank-pagination', props: { currentPage: this.page, pageSize: 20, total: this.total, layout: 'total, prev, pager, next' }, on: { 'current-change': page => { this.page = page; this.refresh() } } }),
      dialog
    ])
  }
}
