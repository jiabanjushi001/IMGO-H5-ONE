// Vue 2 management panel for manual wallet credits and offline withdrawal review.
function imgoWalletRequestId() { return 'admin-' + Date.now().toString(36) + '-' + Math.random().toString(36).slice(2, 14) }
function imgoWalletAmountCents(amount) {
  const parts = /^(\d{1,9})(?:\.(\d{1,2}))?$/.exec(amount)
  return parts ? Number(parts[1]) * 100 + Number((parts[2] || '').padEnd(2, '0')) : 0
}

const ImgoWalletPanel = {
  name: 'ImgoWalletPanel',
  data() {
    return {
      rows: [], total: 0, page: 1, status: '0', keywords: '', loading: false,
      dialog: false, detail: {}, processing: false,
      accountID: '', account: null, accountLoading: false,
      creditAmount: '', creditNote: '', creditRequestID: imgoWalletRequestId(), creditSaving: false
    }
  },
  mounted() { this.refresh() },
  methods: {
    money(cents) { return '¥' + (Number(cents || 0) / 100).toFixed(2) },
    date(seconds) { return Number(seconds) > 0 ? new Date(Number(seconds) * 1000).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai', hour12: false }) : '—' },
    statusName(status) { return ['待处理', '已打款', '已拒绝'][Number(status)] || '未知' },
    async refresh() {
      this.loading = true
      try {
        const res = await this.$api.walletApi.index({ page: this.page, limit: 20, status: this.status, keywords: this.keywords.trim() })
        if (res.code !== 0) throw Error(res.msg || '读取提现列表失败')
        this.rows = Array.isArray(res.data) ? res.data : []
        this.total = Number(res.count || 0)
      } catch (error) { this.$message.error(error.message || '读取提现列表失败') }
      finally { this.loading = false }
    },
    search() { this.page = 1; this.refresh() },
    async lookup() {
      const id = Number(this.accountID)
      this.account = null
      if (!Number.isInteger(id) || id < 1) { this.$message.warning('请输入有效用户 ID'); return }
      this.accountLoading = true
      try {
        const res = await this.$api.walletApi.account({ user_id: id })
        if (res.code !== 0) throw Error(res.msg || '读取用户钱包失败')
        this.account = res.data
      } catch (error) { this.$message.error(error.message || '读取用户钱包失败') }
      finally { this.accountLoading = false }
    },
    async credit() {
      if (!this.account || this.creditSaving) return
      const amount = this.creditAmount.trim(), note = this.creditNote.trim()
      const cents = imgoWalletAmountCents(amount)
      if (!cents || note.length < 2 || note.length > 500) {
        this.$message.warning('请填写正确金额和至少 2 个字的入账说明')
        return
      }
      try {
        await this.$confirm(`确认给 ${this.account.account}（ID ${this.account.user_id}）手动入账 ${this.money(cents)}？此操作会增加用户可提现余额。`, '确认手动入账', { type: 'warning', confirmButtonText: '确认入账', cancelButtonText: '取消' })
      } catch (_) { return }
      this.creditSaving = true
      try {
        const res = await this.$api.walletApi.credit({ user_id: this.account.user_id, amount, note, request_id: this.creditRequestID })
        if (res.code !== 0) throw Error(res.msg || '入账失败')
        this.creditRequestID = imgoWalletRequestId()
        this.creditAmount = ''
        this.creditNote = ''
        this.$message.success('已入账，流水已记录')
        await this.lookup()
      } catch (error) { this.$message.error(error.message || '入账失败，请重试') }
      finally { this.creditSaving = false }
    },
    async open(row) {
      try {
        const res = await this.$api.walletApi.detail({ withdrawal_id: row.withdrawal_id })
        if (res.code !== 0) throw Error(res.msg || '读取提现详情失败')
        this.detail = { ...res.data }
        this.dialog = true
      } catch (error) { this.$message.error(error.message || '读取提现详情失败') }
    },
    close() {
      this.dialog = false
      this.detail.receipt_account = ''
      this.detail = {}
    },
    async review(status) {
      if (!this.detail.withdrawal_id || this.processing) return
      const remark = String(this.detail.remark || '').trim()
      if (status === 2 && remark.length < 2) { this.$message.warning('拒绝时请填写至少 2 个字的原因'); return }
      if (remark.length > 500) { this.$message.warning('备注不能超过 500 字'); return }
      try {
        await this.$confirm(status === 1
          ? '请先完成线下银行转账，再点击确认。此按钮只更新系统记录，不会自动转账。'
          : '确认拒绝并将冻结金额退回用户钱包？',
        status === 1 ? '确认已打款' : '确认拒绝', { type: 'warning', confirmButtonText: status === 1 ? '已完成打款' : '确认拒绝', cancelButtonText: '取消' })
      } catch (_) { return }
      this.processing = true
      try {
        const res = await this.$api.walletApi.review({ withdrawal_id: this.detail.withdrawal_id, status, remark })
        if (res.code !== 0) throw Error(res.msg || '处理失败')
        this.$message.success(status === 1 ? '已记录线下打款' : '已拒绝并退回余额')
        this.close()
        await this.refresh()
      } catch (error) { this.$message.error(error.message || '处理失败，请重试') }
      finally { this.processing = false }
    }
  },
  render(h) {
    const input = (value, placeholder, onInput, options = {}) => h('el-input', {
      props: { value, clearable: true, type: options.type || 'text' },
      attrs: { placeholder, maxlength: options.maxlength },
      on: { input: onInput, keyup: event => { if (event.key === 'Enter' && options.enter) options.enter() } }
    })
    const column = (label, prop, width) => h('el-table-column', { props: { label, prop, width } })
    const detail = this.detail || {}
    const detailDialog = h('el-dialog', {
      props: { title: '提现详情', visible: this.dialog, width: '560px', closeOnClickModal: false, appendToBody: true },
      on: { close: this.close }
    }, this.dialog ? [
      h('div', { class: 'imgo-wallet-detail' }, [
        h('div', [h('span', '用户'), `${detail.account || '—'}（ID ${detail.user_id}）`]),
        h('div', [h('span', '提现金额'), h('strong', this.money(detail.amount_cents))]),
        h('div', [h('span', '申请时间'), this.date(detail.created_at)]),
        h('div', [h('span', '状态'), this.statusName(detail.status)]),
        h('div', [h('span', '收款姓名'), detail.receipt_name || '—']),
        h('div', [h('span', '收款银行'), detail.bank_name || '—']),
        h('div', [h('span', '支行名称'), detail.branch_name || '—']),
        h('div', [h('span', '收款卡号'), h('strong', { class: 'imgo-wallet-account' }, detail.receipt_account || '—')])
      ]),
      h('div', { class: 'imgo-wallet-remark' }, [
        h('label', '处理备注 / 拒绝原因'),
        h('el-input', { props: { type: 'textarea', value: detail.remark || '', rows: 3, showWordLimit: true, disabled: Number(detail.status) !== 0 }, attrs: { maxlength: 500, placeholder: '拒绝时须填写原因；打款时可填写转账备注' }, on: { input: value => { this.$set(this.detail, 'remark', value) } } })
      ]),
      h('span', { slot: 'footer' }, Number(detail.status) === 0 ? [
        h('el-button', { on: { click: this.close } }, '关闭'),
        h('el-button', { props: { type: 'danger', loading: this.processing }, on: { click: () => this.review(2) } }, '拒绝并退款'),
        h('el-button', { props: { type: 'primary', loading: this.processing }, on: { click: () => this.review(1) } }, '确认已线下打款')
      ] : [h('el-button', { on: { click: this.close } }, '关闭')])
    ] : [])

    return h('section', { class: 'imgo-wallet' }, [
      h('div', { class: 'imgo-wallet-heading' }, [h('div', [h('h1', '钱包与提现'), h('p', '手动入账、核对提现申请并记录线下打款')])]),
      h('div', { class: 'imgo-wallet-credit' }, [
        h('h2', '余额入账'),
        h('p', '仅初始管理员可操作；每次入账都会记录金额、操作人和原因。'),
        h('div', { class: 'imgo-wallet-credit-fields' }, [
          input(this.accountID, '用户 ID', value => { this.accountID = value; this.account = null }, { enter: this.lookup }),
          h('el-button', { props: { loading: this.accountLoading }, on: { click: this.lookup } }, '查询用户'),
          this.account ? h('span', { class: 'imgo-wallet-account-summary' }, `${this.account.account} · ${this.account.realname || '—'} · 可用 ${this.money(this.account.available_cents)}`) : null
        ]),
        h('div', { class: 'imgo-wallet-credit-fields' }, [
          input(this.creditAmount, '入账金额（元）', value => { this.creditAmount = value; this.creditRequestID = imgoWalletRequestId() }),
          input(this.creditNote, '入账原因（必填）', value => { this.creditNote = value; this.creditRequestID = imgoWalletRequestId() }, { maxlength: 500 }),
          h('el-button', { props: { type: 'primary', disabled: !this.account, loading: this.creditSaving }, on: { click: this.credit } }, '确认入账')
        ])
      ]),
      h('div', { class: 'imgo-wallet-withdrawals' }, [
        h('div', { class: 'imgo-wallet-toolbar' }, [
          h('h2', '提现申请'),
          input(this.keywords, '搜索账号或姓名', value => { this.keywords = value }, { enter: this.search }),
          h('el-select', { props: { value: this.status }, on: { input: value => { this.status = value; this.search() } } }, [
            h('el-option', { props: { label: '全部状态', value: '' } }),
            [0, 1, 2].map(value => h('el-option', { key: value, props: { label: this.statusName(value), value: String(value) } }))
          ]),
          h('el-button', { props: { type: 'primary' }, on: { click: this.search } }, '查询'),
          h('el-button', { props: { icon: 'el-icon-refresh', loading: this.loading }, on: { click: this.refresh } }, '刷新')
        ]),
        h('el-table', { props: { data: this.rows, border: true, stripe: true }, directives: [{ name: 'loading', value: this.loading }] }, [
          column('申请ID', 'withdrawal_id', '90'), column('用户账号', 'account', '140'),
          h('el-table-column', { props: { label: '金额', width: '130' }, scopedSlots: { default: scope => this.money(scope.row.amount_cents) } }),
          h('el-table-column', { props: { label: '状态', width: '100' }, scopedSlots: { default: scope => h('el-tag', { props: { type: ['warning', 'success', 'danger'][Number(scope.row.status)] } }, this.statusName(scope.row.status)) } }),
          column('收款银行', 'bank_name', '150'),
          h('el-table-column', { props: { label: '卡号末四位', width: '130' }, scopedSlots: { default: scope => '•••• ' + scope.row.account_last4 } }),
          h('el-table-column', { props: { label: '申请时间', width: '190' }, scopedSlots: { default: scope => this.date(scope.row.created_at) } }),
          column('备注', 'remark'),
          h('el-table-column', { props: { label: '操作', width: '95', fixed: 'right', align: 'center' }, scopedSlots: { default: scope => h('el-button', { props: { type: 'text' }, on: { click: () => this.open(scope.row) } }, '查看处理') } })
        ]),
        h('el-pagination', { class: 'imgo-wallet-pagination', props: { currentPage: this.page, pageSize: 20, total: this.total, layout: 'total, prev, pager, next' }, on: { 'current-change': page => { this.page = page; this.refresh() } } })
      ]),
      detailDialog
    ])
  }
}
