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
