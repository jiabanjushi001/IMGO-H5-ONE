const ImgoAuditPanel = {
  name: 'ImgoAuditPanel',
  data() {
    return {
      loading: false, rows: [], total: 0,
      requestVisible: false, activeRequest: null,
      query: { page: 1, limit: 20, category: '', risk_level: '', status: '', keywords: '' },
      categories: ['登录', '成员', '实名认证', '群聊', '设置', '概况', '消息', '任务', '角色', '绑卡', '财务']
    }
  },
  mounted() { this.load() },
  methods: {
    async load(reset) {
      if (reset) this.query.page = 1
      this.loading = true
      try {
        const result = await this.$api.auditApi.index(this.query)
        if (Number(result.code) !== 0) throw Error(result.msg || '读取日志失败')
        this.rows = Array.isArray(result.data) ? result.data : []
        this.total = Number(result.count || 0)
      } catch (error) { this.$message.error(error.message || '读取日志失败') }
      finally { this.loading = false }
    },
    time(value) {
      const date = new Date(Number(value || 0) * 1000)
      if (Number.isNaN(date.getTime())) return '-'
      const pad = number => String(number).padStart(2, '0')
      return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
    },
    risk(value) {
      return ({ low: ['普通', 'info'], medium: ['注意', 'warning'], high: ['高敏', 'danger'], critical: ['严重', 'danger'] })[value] || ['普通', 'info']
    },
    pretty(value) {
      if (!value) return '{}'
      try { return JSON.stringify(typeof value === 'string' ? JSON.parse(value) : value, null, 2) }
      catch (_) { return String(value) }
    },
    openRequest(row) {
      this.activeRequest = row
      this.requestVisible = true
    }
  },
  render(h) {
    const columns = [
      h('el-table-column', { props: { label: '时间', width: 170 }, scopedSlots: { default: ({ row }) => h('span', this.time(row.created_at)) } }),
      h('el-table-column', { props: { label: '操作账号', minWidth: 160 }, scopedSlots: { default: ({ row }) => h('div', [h('strong', row.actor_account || '-'), h('small', { class: 'imgo-audit-sub' }, `${row.actor_name || ''} · ${row.actor_role || ''}`)]) } }),
      h('el-table-column', { props: { label: '操作分类', minWidth: 150 }, scopedSlots: { default: ({ row }) => h('span', { class: 'imgo-audit-action' }, [h('b', row.category), h('i', '→'), h('span', row.action)]) } }),
      h('el-table-column', { props: { label: '敏感等级', width: 92 }, scopedSlots: { default: ({ row }) => { const risk = this.risk(row.risk_level); return h('el-tag', { props: { size: 'mini', type: risk[1], effect: row.risk_level === 'critical' ? 'dark' : 'light' } }, risk[0]) } } }),
      h('el-table-column', { props: { label: '操作对象', minWidth: 160 }, scopedSlots: { default: ({ row }) => h('span', row.target_name ? `${row.target_name}（${row.target_type || '用户'} #${row.target_id}）` : `${row.target_type || '-'}${row.target_id ? ` #${row.target_id}` : ''}`) } }),
      h('el-table-column', { props: { label: 'IP', minWidth: 125 }, scopedSlots: { default: ({ row }) => h('code', row.ip || '-') } }),
      h('el-table-column', { props: { label: '结果', width: 90 }, scopedSlots: { default: ({ row }) => h('el-tag', { props: { size: 'mini', type: Number(row.status) === 1 ? 'success' : 'danger' } }, Number(row.status) === 1 ? '成功' : '失败') } }),
      h('el-table-column', { props: { label: '说明', minWidth: 150, showOverflowTooltip: true }, scopedSlots: { default: ({ row }) => h('span', row.error_message || row.detail || '-') } }),
      h('el-table-column', { props: { label: '请求数据', width: 100, fixed: 'right' }, scopedSlots: { default: ({ row }) => h('el-button', { props: { type: 'text', size: 'mini' }, on: { click: () => this.openRequest(row) } }, '查看') } })
    ]
    return h('section', { class: 'imgo-audit-page' }, [
      h('header', { class: 'imgo-audit-heading' }, [h('div', [h('h1', '日志'), h('p', '记录后台账号的重要操作，并按业务和敏感等级分类。')])]),
      h('div', { class: 'imgo-audit-filters' }, [
        h('el-input', { props: { value: this.query.keywords, clearable: true, placeholder: '账号、姓名、动作、对象或 IP' }, on: { input: value => { this.query.keywords = value }, clear: () => this.load(true) }, nativeOn: { keyup: event => { if (event.key === 'Enter') this.load(true) } } }),
        h('el-select', { props: { value: this.query.category, clearable: true, placeholder: '业务分类' }, on: { input: value => { this.query.category = value }, change: () => this.load(true) } }, this.categories.map(value => h('el-option', { key: value, props: { label: value, value } }))),
        h('el-select', { props: { value: this.query.risk_level, clearable: true, placeholder: '敏感等级' }, on: { input: value => { this.query.risk_level = value }, change: () => this.load(true) } }, [
          h('el-option', { props: { label: '普通', value: 'low' } }), h('el-option', { props: { label: '注意', value: 'medium' } }),
          h('el-option', { props: { label: '高敏', value: 'high' } }), h('el-option', { props: { label: '严重', value: 'critical' } })
        ]),
        h('el-select', { props: { value: this.query.status, clearable: true, placeholder: '操作结果' }, on: { input: value => { this.query.status = value }, change: () => this.load(true) } }, [h('el-option', { props: { label: '成功', value: '1' } }), h('el-option', { props: { label: '失败', value: '0' } })]),
        h('el-button', { props: { type: 'primary', icon: 'el-icon-search' }, on: { click: () => this.load(true) } }, '查询')
      ]),
      h('el-table', { class: 'imgo-audit-table', props: { data: this.rows, border: true, stripe: true }, directives: [{ name: 'loading', value: this.loading }] }, columns),
      h('div', { class: 'imgo-audit-pagination' }, [h('el-pagination', { props: { currentPage: this.query.page, pageSize: this.query.limit, total: this.total, layout: 'total, prev, pager, next' }, on: { 'current-change': page => { this.query.page = page; this.load() } } })]),
      h('el-dialog', {
        props: { title: '请求详情', visible: this.requestVisible, width: '760px', appendToBody: true, closeOnClickModal: false },
        on: { 'update:visible': value => { this.requestVisible = value } }
      }, this.activeRequest ? [
        h('div', { class: 'imgo-audit-request-line' }, [h('el-tag', { props: { size: 'mini' } }, this.activeRequest.request_method || 'POST'), h('code', this.activeRequest.request_path || '-')]),
        h('div', { class: 'imgo-audit-request-block' }, [h('h3', '请求头'), h('pre', this.pretty(this.activeRequest.request_headers))]),
        h('div', { class: 'imgo-audit-request-block' }, [h('h3', '请求数据'), h('pre', this.pretty(this.activeRequest.request_data))]),
        h('el-alert', { props: { title: '密码、Token、Cookie、验证码和银行卡号等敏感值已隐藏', type: 'info', showIcon: true, closable: false } })
      ] : [])
    ])
  }
}
