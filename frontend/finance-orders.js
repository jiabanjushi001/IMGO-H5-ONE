// Vue 2 components loaded through the existing compiled management shell.
function imgoFinanceMoney(cents) { return '¥' + (Number(cents || 0) / 100).toFixed(2) }
function imgoFinanceOrderNumber(prefix, id) { return prefix + '-' + String(id || 0).padStart(8, '0') }
function imgoFinanceDate(seconds) {
  return Number(seconds) > 0
    ? new Date(Number(seconds) * 1000).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai', hour12: false })
    : '—'
}
function imgoFinanceSearchInput(h, value, onInput, onSearch) {
  return h('el-input', {
    props: { value, placeholder: '搜索成员账号或姓名', clearable: true },
    on: { input: onInput, keyup: event => { if (event.key === 'Enter') onSearch() } }
  })
}
function imgoFinancePagination(h, view) {
  return h('el-pagination', {
    class: 'imgo-finance-pagination',
    props: { currentPage: view.page, pageSize: 20, total: view.total, layout: 'total, prev, pager, next' },
    on: { 'current-change': page => { view.page = page; view.refresh() } }
  })
}

const ImgoRechargeOrders = {
  name: 'ImgoRechargeOrders',
  data() { return { rows: [], total: 0, page: 1, keywords: '', loading: false } },
  mounted() { this.refresh() },
  methods: {
    async refresh() {
      this.loading = true
      try {
        const result = await this.$api.walletApi.recharges({ page: this.page, limit: 20, keywords: this.keywords.trim() })
        if (result.code !== 0) throw Error(result.msg || '读取充值订单失败')
        this.rows = Array.isArray(result.data) ? result.data : []
        this.total = Number(result.count || 0)
      } catch (error) { this.$message.error(error.message || '读取充值订单失败') }
      finally { this.loading = false }
    },
    search() { this.page = 1; this.refresh() },
    statusName(value) { return ({ 0: '待处理', 1: '已到账', 2: '已取消' })[Number(value)] || '未知' },
    statusColor(value) { return ({ 0: 'warning', 1: 'success', 2: 'info' })[Number(value)] || 'info' }
  },
  render(h) {
    const col = (label, prop, width) => h('el-table-column', { props: { label, prop, width } })
    return h('section', { class: 'imgo-finance' }, [
      h('header', { class: 'imgo-finance-heading' }, [
        h('div', [h('h1', '充值订单'), h('p', '成员操作中的充值立即入账；本金和赠送分别记入余额账变。')])
      ]),
      h('div', { class: 'imgo-finance-card' }, [
        h('div', { class: 'imgo-finance-toolbar' }, [
          imgoFinanceSearchInput(h, this.keywords, value => { this.keywords = value }, this.search),
          h('el-button', { props: { type: 'primary' }, on: { click: this.search } }, '查询'),
          h('el-button', { props: { icon: 'el-icon-refresh', loading: this.loading }, on: { click: this.refresh } }, '刷新')
        ]),
        h('el-table', { props: { data: this.rows, border: true, stripe: true }, directives: [{ name: 'loading', value: this.loading }] }, [
          h('el-table-column', { props: { label: '充值订单号', width: '150' }, scopedSlots: { default: scope => imgoFinanceOrderNumber('CZ', scope.row.order_id) } }),
          col('成员账号', 'account', '150'),
          h('el-table-column', { props: { label: '订单金额', width: '135' }, scopedSlots: { default: scope => h('strong', imgoFinanceMoney(scope.row.total_cents)) } }),
          h('el-table-column', { props: { label: '本金', width: '120' }, scopedSlots: { default: scope => imgoFinanceMoney(scope.row.amount_cents) } }),
          h('el-table-column', { props: { label: '赠送', width: '120' }, scopedSlots: { default: scope => imgoFinanceMoney(scope.row.bonus_cents) } }),
          h('el-table-column', { props: { label: '时间', width: '190' }, scopedSlots: { default: scope => imgoFinanceDate(scope.row.created_at) } }),
          h('el-table-column', { props: { label: '状态', width: '105' }, scopedSlots: { default: scope => h('el-tag', { props: { type: this.statusColor(scope.row.status), size: 'small' } }, this.statusName(scope.row.status)) } }),
          col('操作员ID', 'created_by', '110'),
          col('备注', 'note')
        ]),
        imgoFinancePagination(h, this)
      ])
    ])
  }
}

const ImgoWithdrawalDetail = {
  name: 'ImgoWithdrawalDetail',
  props: { visible: Boolean, detail: { type: Object, default: () => ({}) }, processing: Boolean },
  render(h) {
    const detail = this.detail || {}
    const info = (label, value) => h('div', [h('span', label), value || '—'])
    return h('el-dialog', {
      props: { title: '提现订单详情', visible: this.visible, width: '560px', closeOnClickModal: false, appendToBody: true },
      on: { close: () => this.$emit('close') }
    }, this.visible ? [
      h('div', { class: 'imgo-finance-detail' }, [
        info('订单号', imgoFinanceOrderNumber('TX', detail.withdrawal_id)),
        info('成员', `${detail.account || '—'}（ID ${detail.user_id || '—'}）`),
        info('金额', h('strong', imgoFinanceMoney(detail.amount_cents))),
        info('申请时间', imgoFinanceDate(detail.created_at)),
        info('状态', ['待处理', '已打款', '已拒绝'][Number(detail.status)] || '未知'),
        info('收款姓名', detail.receipt_name),
        info('收款银行', detail.bank_name),
        info('支行名称', detail.branch_name),
        info('收款卡号', h('strong', { class: 'imgo-finance-account' }, detail.receipt_account || '—'))
      ]),
      h('div', { class: 'imgo-finance-remark' }, [
        h('label', '处理备注 / 拒绝原因'),
        h('el-input', {
          props: { type: 'textarea', rows: 3, value: detail.remark || '', disabled: Number(detail.status) !== 0, showWordLimit: true },
          attrs: { maxlength: 500, placeholder: '拒绝时须填写原因；确认打款可填写转账备注' },
          on: { input: value => this.$emit('remark-change', value) }
        })
      ]),
      h('span', { slot: 'footer' }, Number(detail.status) === 0 ? [
        h('el-button', { on: { click: () => this.$emit('close') } }, '关闭'),
        h('el-button', { props: { type: 'danger', loading: this.processing }, on: { click: () => this.$emit('review', 2) } }, '拒绝并退款'),
        h('el-button', { props: { type: 'primary', loading: this.processing }, on: { click: () => this.$emit('review', 1) } }, '确认已线下打款')
      ] : [h('el-button', { on: { click: () => this.$emit('close') } }, '关闭')])
    ] : [])
  }
}

const ImgoWithdrawalOrders = {
  name: 'ImgoWithdrawalOrders',
  data() {
    return { rows: [], total: 0, page: 1, keywords: '', status: '', loading: false, visible: false, detail: {}, processing: false }
  },
  mounted() { this.refresh() },
  methods: {
    statusName(value) { return ['待处理', '已打款', '已拒绝'][Number(value)] || '未知' },
    async refresh() {
      this.loading = true
      try {
        const result = await this.$api.walletApi.index({ page: this.page, limit: 20, status: this.status, keywords: this.keywords.trim() })
        if (result.code !== 0) throw Error(result.msg || '读取提现订单失败')
        this.rows = Array.isArray(result.data) ? result.data : []
        this.total = Number(result.count || 0)
      } catch (error) { this.$message.error(error.message || '读取提现订单失败') }
      finally { this.loading = false }
    },
    search() { this.page = 1; this.refresh() },
    async open(row) {
      try {
        const result = await this.$api.walletApi.detail({ withdrawal_id: row.withdrawal_id })
        if (result.code !== 0) throw Error(result.msg || '读取提现详情失败')
        this.detail = { ...result.data }
        this.visible = true
      } catch (error) { this.$message.error(error.message || '读取提现详情失败') }
    },
    close() { this.visible = false; this.detail.receipt_account = ''; this.detail = {} },
    async review(status) {
      if (!this.detail.withdrawal_id || this.processing) return
      const remark = String(this.detail.remark || '').trim()
      if (status === 2 && remark.length < 2) { this.$message.warning('拒绝时请填写至少 2 个字的原因'); return }
      if (remark.length > 500) { this.$message.warning('备注不能超过 500 字'); return }
      try {
        await this.$confirm(status === 1
          ? '请先完成线下银行转账，再确认已打款。此操作只记录线下结果，不会自动转账。'
          : '确认拒绝并将冻结金额退回成员钱包？',
        status === 1 ? '确认已打款' : '确认拒绝', { type: 'warning', confirmButtonText: status === 1 ? '已完成打款' : '确认拒绝', cancelButtonText: '取消' })
      } catch (_) { return }
      this.processing = true
      try {
        const result = await this.$api.walletApi.review({ withdrawal_id: this.detail.withdrawal_id, status, remark })
        if (result.code !== 0) throw Error(result.msg || '处理失败')
        this.$message.success(status === 1 ? '已记录线下打款' : '已拒绝并退回余额')
        this.close()
        await this.refresh()
      } catch (error) { this.$message.error(error.message || '处理失败，请重试') }
      finally { this.processing = false }
    }
  },
  render(h) {
    const col = (label, prop, width) => h('el-table-column', { props: { label, prop, width } })
    return h('section', { class: 'imgo-finance' }, [
      h('header', { class: 'imgo-finance-heading' }, [
        h('div', [h('h1', '提现订单'), h('p', '核对冻结金额和收款银行卡，人工线下打款后再确认。')])
      ]),
      h('div', { class: 'imgo-finance-card' }, [
        h('div', { class: 'imgo-finance-toolbar' }, [
          imgoFinanceSearchInput(h, this.keywords, value => { this.keywords = value }, this.search),
          h('el-select', { props: { value: this.status }, on: { input: value => { this.status = value; this.search() } } }, [
            h('el-option', { props: { label: '全部状态', value: '' } }),
            [0, 1, 2].map(value => h('el-option', { key: value, props: { label: this.statusName(value), value: String(value) } }))
          ]),
          h('el-button', { props: { type: 'primary' }, on: { click: this.search } }, '查询'),
          h('el-button', { props: { icon: 'el-icon-refresh', loading: this.loading }, on: { click: this.refresh } }, '刷新')
        ]),
        h('el-table', { props: { data: this.rows, border: true, stripe: true }, directives: [{ name: 'loading', value: this.loading }] }, [
          h('el-table-column', { props: { label: '提现订单号', width: '150' }, scopedSlots: { default: scope => imgoFinanceOrderNumber('TX', scope.row.withdrawal_id) } }),
          col('成员账号', 'account', '145'),
          h('el-table-column', { props: { label: '订单金额', width: '135' }, scopedSlots: { default: scope => h('strong', imgoFinanceMoney(scope.row.amount_cents)) } }),
          h('el-table-column', { props: { label: '时间', width: '190' }, scopedSlots: { default: scope => imgoFinanceDate(scope.row.created_at) } }),
          h('el-table-column', { props: { label: '状态', width: '105' }, scopedSlots: { default: scope => h('el-tag', { props: { type: ['warning', 'success', 'danger'][Number(scope.row.status)] || 'info', size: 'small' } }, this.statusName(scope.row.status)) } }),
          col('收款银行', 'bank_name', '160'),
          h('el-table-column', { props: { label: '卡号末四位', width: '130' }, scopedSlots: { default: scope => '•••• ' + scope.row.account_last4 } }),
          col('处理备注', 'remark'),
          h('el-table-column', { props: { label: '操作', width: '115', fixed: 'right', align: 'center' }, scopedSlots: { default: scope => h('el-button', { props: { type: 'text' }, on: { click: () => this.open(scope.row) } }, Number(scope.row.status) === 0 ? '查看处理' : '查看详情') } })
        ]),
        imgoFinancePagination(h, this)
      ]),
      h(ImgoWithdrawalDetail, {
        props: { visible: this.visible, detail: this.detail, processing: this.processing },
        on: { close: this.close, 'remark-change': value => { this.$set(this.detail, 'remark', value) }, review: this.review }
      })
    ])
  }
}

const ImgoFinanceShell = {
  name: 'ImgoFinanceShell',
  render(h) { return h(this.$route.path === '/manage/finance/recharges' ? ImgoRechargeOrders : ImgoWithdrawalOrders) }
}
