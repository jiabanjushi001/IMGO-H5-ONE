const ImgoMemberBatchCreateDialog = {
  name: 'ImgoMemberBatchCreateDialog',
  data() {
    return {
      visible: false,
      loading: false,
      submitting: false,
      copying: false,
      polling: false,
      pollTimer: null,
      pollFailures: 0,
      candidates: [],
      inviterOptions: [],
      results: [],
      task: null,
      form: { count: 10, account_prefix: '', password: '123456', parent_invite_code: '', customer_user_id: 0 }
    }
  },
  computed: {
    mentorMode() {
      const user = this.$store.state.userInfo || {}
      return Number(user.user_id) !== 1 && Number(user.agent_mode) === 1
    },
    activeCandidates() {
      return this.candidates.filter(item => Number(item.status) === 1)
    },
    taskActive() {
      return !!this.task && (this.task.status === 'queued' || this.task.status === 'running')
    },
    taskFinished() {
      return !!this.task && (this.task.status === 'completed' || this.task.status === 'failed')
    },
    progressPercent() {
      if (!this.task) return 0
      return Math.max(0, Math.min(100, Number(this.task.percent || 0)))
    },
    taskStatusText() {
      if (!this.task) return ''
      return { queued: '等待开始', running: '正在创建', completed: '创建完成', failed: '创建中断' }[this.task.status] || '处理中'
    },
    accountPrefixHint() {
      const prefix = String(this.form.account_prefix || '').trim()
      if (!prefix) return '不填写前缀时，账号使用随机字符生成'
      return `将按 ${prefix}001、${prefix}002… 的顺序生成；如已有同前缀账号，会自动接续下一编号`
    }
  },
  beforeDestroy() {
    this.stopPolling()
  },
  methods: {
    emptyForm() {
      return { count: 10, account_prefix: '', password: '123456', parent_invite_code: '', customer_user_id: 0 }
    },
    storageKey() {
      const user = this.$store.state.userInfo || {}
      return `imgo:member-batch-task:${Number(user.user_id || 0)}`
    },
    savedTaskID() {
      try {
        return String(window.localStorage.getItem(this.storageKey()) || '')
      } catch (_) {
        return ''
      }
    },
    rememberTask(taskID) {
      try {
        if (taskID) window.localStorage.setItem(this.storageKey(), taskID)
        else window.localStorage.removeItem(this.storageKey())
      } catch (_) {}
    },
    async loadOptions() {
      const user = this.$store.state.userInfo || {}
      const response = await this.$api.userApi.getUserList({ page: 1, limit: 200, keywords: '', referral_scope: this.mentorMode ? 'all' : '' })
      if (Number(response.code) !== 0) throw Error(response.msg || '读取成员列表失败')
      this.candidates = Array.isArray(response.data) ? response.data : []
      if (!this.mentorMode) return
      this.inviterOptions = this.candidates.filter(item => Number(item.status) === 1 && /^\d{6}$/.test(String(item.invite_code || '')))
      const self = this.inviterOptions.find(item => Number(item.user_id) === Number(user.user_id))
      if (!self) throw Error('未读取到导师本人的邀请码，请刷新后重试')
      if (!this.form.parent_invite_code) this.form.parent_invite_code = String(self.invite_code)
    },
    async open() {
      this.stopPolling()
      this.form = this.emptyForm()
      this.results = []
      this.task = null
      this.candidates = []
      this.inviterOptions = []
      this.visible = true
      this.loading = true
      try {
        await Promise.all([this.loadOptions(), this.restoreTask()])
        if (this.task && this.task.status === 'completed' && this.$message.info) this.$message.info('已恢复上次批量创建结果')
      } catch (error) {
        this.$message.error(error.message || '读取批量创建配置失败')
      } finally {
        this.loading = false
      }
    },
    async restoreTask() {
      let taskID = this.savedTaskID()
      let response = await this.$api.userApi.batchStatus(taskID ? { task_id: taskID } : {})
      if (Number(response.code) !== 0) throw Error(response.msg || '读取创建进度失败')
      if ((!response.data || !response.data.found) && taskID) {
        this.rememberTask('')
        taskID = ''
        response = await this.$api.userApi.batchStatus({})
        if (Number(response.code) !== 0) throw Error(response.msg || '读取创建进度失败')
      }
      if (!response.data || !response.data.found) return
      this.applyTask(response.data, false)
      if (this.taskActive) this.startPolling()
    },
    async searchCustomers(keyword) {
      if (this.mentorMode) return
      this.loading = true
      try {
        const response = await this.$api.userApi.getUserList({ page: 1, limit: 200, keywords: String(keyword || '').trim(), referral_scope: '' })
        if (Number(response.code) !== 0) throw Error(response.msg || '搜索客服失败')
        this.candidates = Array.isArray(response.data) ? response.data : []
      } catch (error) {
        this.$message.error(error.message || '搜索客服失败')
      } finally {
        this.loading = false
      }
    },
    optionLabel(item) {
      const name = item.realname || '未命名'
      return `${name}（${item.account || item.user_id}）`
    },
    validate() {
      const count = Number(this.form.count)
      const password = String(this.form.password || '')
      const prefix = String(this.form.account_prefix || '').trim()
      if (!Number.isInteger(count) || count < 1 || count > 100) return '账号个数须为 1–100'
      if (password.length < 6 || password.length > 30) return '统一密码长度须为 6–30 位'
      if (prefix.length > 20 || !/^[A-Za-z0-9_-]*$/.test(prefix)) return '账号前缀只能填写 20 位以内的字母、数字、下划线或短横线'
      if (this.mentorMode && !/^\d{6}$/.test(String(this.form.parent_invite_code || ''))) return '请选择本人或自己团队的上级邀请码'
      if (!this.mentorMode && this.form.parent_invite_code && !/^\d{6}$/.test(String(this.form.parent_invite_code))) return '上级邀请码须为 6 位数字'
      return ''
    },
    async submit() {
      if (this.submitting || this.taskActive) return
      const message = this.validate()
      if (message) return this.$message.error(message)
      this.submitting = true
      try {
        const payload = {
          count: Number(this.form.count),
          account_prefix: String(this.form.account_prefix || '').trim(),
          password: String(this.form.password),
          parent_invite_code: String(this.form.parent_invite_code || '').trim(),
          customer_user_id: this.mentorMode ? 0 : Number(this.form.customer_user_id || 0)
        }
        const response = await this.$api.userApi.batchAdd(payload)
        if (Number(response.code) !== 0) throw Error(response.msg || '批量创建失败')
        this.applyTask(response.data || {}, false)
        if (!this.task || !this.task.task_id) throw Error('服务器未返回任务编号')
        this.$message.success(response.data.resumed ? '已恢复正在执行的创建任务' : '创建任务已提交，可关闭弹窗或刷新页面')
        this.startPolling()
      } catch (error) {
        this.$message.error(error.message || '批量创建失败')
      } finally {
        this.submitting = false
      }
    },
    applyTask(data, announce) {
      const previousStatus = this.task && this.task.status
      this.task = Object.assign({}, data)
      this.results = Array.isArray(data.items) ? data.items : []
      if (data.task_id) this.rememberTask(data.task_id)
      const finishedNow = data.status === 'completed' || data.status === 'failed'
      if (finishedNow) this.stopPolling()
      if (!announce || previousStatus === data.status || !finishedNow) return
      this.$emit('saved')
      if (data.status === 'completed') this.$message.success(`成功创建 ${Number(data.success_count || this.results.length)} 个账号`)
      else this.$message.error(data.error_message || '创建任务已中断，请查看已完成结果')
    },
    async refreshTask(silent) {
      if (!this.task || !this.task.task_id || this.polling) return
      this.polling = true
      try {
        const response = await this.$api.userApi.batchStatus({ task_id: this.task.task_id })
        if (Number(response.code) !== 0 || !response.data || !response.data.found) throw Error(response.msg || '任务记录不存在')
        this.pollFailures = 0
        this.applyTask(response.data, true)
        if (!silent && this.taskActive && this.$message.info) this.$message.info('进度已更新')
      } catch (error) {
        this.pollFailures += 1
        if (!silent || this.pollFailures === 1) this.$message.error(error.message || '刷新创建进度失败')
      } finally {
        this.polling = false
        if (this.taskActive && this.visible) this.schedulePoll()
      }
    },
    schedulePoll() {
      this.stopPollTimer()
      const delay = this.pollFailures > 2 ? 3000 : 1200
      this.pollTimer = setTimeout(() => this.refreshTask(true), delay)
    },
    startPolling() {
      if (!this.taskActive || !this.visible) return
      this.schedulePoll()
    },
    stopPollTimer() {
      if (this.pollTimer) clearTimeout(this.pollTimer)
      this.pollTimer = null
    },
    stopPolling() {
      this.stopPollTimer()
      this.polling = false
    },
    resultText() {
      const header = '账号\t密码\t姓名\t性别\t上级邀请码\t客服'
      const lines = this.results.map(item => [
        item.account || '', item.password || '', item.realname || '', item.sex_label || '',
        item.parent_invite_code || '无', item.customer_account || '未指定'
      ].join('\t'))
      return [header].concat(lines).join('\n')
    },
    async copyResults() {
      if (!this.results.length || this.copying) return
      this.copying = true
      const text = this.resultText()
      try {
        if (typeof navigator !== 'undefined' && navigator.clipboard && window.isSecureContext) {
          await navigator.clipboard.writeText(text)
        } else {
          const area = document.createElement('textarea')
          area.value = text
          area.style.position = 'fixed'
          area.style.opacity = '0'
          document.body.appendChild(area)
          area.select()
          if (!document.execCommand('copy')) throw Error('浏览器拒绝复制')
          document.body.removeChild(area)
        }
        this.$message.success(`已复制 ${this.results.length} 个账号`)
      } catch (error) {
        this.$message.error('复制失败，请手动选择结果复制')
      } finally {
        this.copying = false
      }
    },
    createAgain() {
      const inviteCode = this.mentorMode ? this.form.parent_invite_code : ''
      this.stopPolling()
      this.rememberTask('')
      this.form = this.emptyForm()
      this.form.parent_invite_code = inviteCode
      this.results = []
      this.task = null
    },
    close(done) {
      if (this.submitting) return
      const wasVisible = this.visible
      if (wasVisible && this.taskActive && this.$message.info) this.$message.info('任务将在后台继续，重新打开可查看进度')
      this.stopPolling()
      this.visible = false
      if (typeof done === 'function') done()
    }
  },
  render(h) {
    const field = (label, child, wide) => h('el-form-item', { class: { 'is-wide': !!wide }, props: { label } }, [child])
    const countControl = h('div', { class: 'imgo-member-batch-count' }, [
      h('el-input-number', { props: { value: this.form.count, min: 1, max: 100, step: 1 }, on: { input: value => { this.form.count = value } } }),
      h('div', { class: 'imgo-member-batch-presets' }, [10, 20, 50, 100].map(value => h('button', { key: value, class: { 'is-active': Number(this.form.count) === value }, attrs: { type: 'button' }, on: { click: () => { this.form.count = value } } }, String(value))))
    ])
    const form = h('div', { class: 'imgo-member-batch-form' }, [
      h('div', { class: 'imgo-member-batch-intro' }, [
        h('div', { class: 'imgo-member-batch-intro-icon' }, [h('i', { class: 'el-icon-user-solid' })]),
        h('div', [h('strong', '批量生成成员账号'), h('p', `本次将创建 ${Number(this.form.count || 0)} 个账号。提交后可关闭弹窗或刷新网页，任务不会中断。`)])
      ]),
      field('账号个数', countControl),
      field('统一密码', h('el-input', { props: { value: this.form.password, showPassword: true, maxlength: 30, placeholder: '6–30 位密码' }, on: { input: value => { this.form.password = value } } })),
      field('账号前缀', h('div', { class: 'imgo-member-batch-prefix' }, [
        h('el-input', { props: { value: this.form.account_prefix, maxlength: 20, clearable: true, placeholder: '选填，例如 wuhu' }, on: { input: value => { this.form.account_prefix = value } } }),
        h('div', { class: 'imgo-member-batch-prefix-hint' }, this.accountPrefixHint)
      ])),
      field('姓名与性别', h('div', { class: 'imgo-member-batch-static' }, '中文姓名随机 2–4 个字，性别随机生成')),
      this.mentorMode
        ? field('上级邀请码', h('el-select', { props: { value: this.form.parent_invite_code, filterable: true, placeholder: '请选择本人或团队成员', loading: this.loading }, on: { input: value => { this.form.parent_invite_code = value } } }, this.inviterOptions.map(item => h('el-option', { key: item.user_id, props: { value: String(item.invite_code), label: `${this.optionLabel(item)} · ${item.invite_code}` } }))), true)
        : field('上级邀请码', h('el-input', { props: { value: this.form.parent_invite_code, maxlength: 6, clearable: true, placeholder: '选填；不填则无上级' }, on: { input: value => { this.form.parent_invite_code = String(value || '').replace(/\D/g, '').slice(0, 6) } } }), true),
      this.mentorMode
        ? field('专属客服', h('div', { class: 'imgo-member-batch-static is-required' }, '由导师设置中的默认客服自动分配，不允许手动修改'), true)
        : field('专属客服', h('el-select', { props: { value: this.form.customer_user_id || '', filterable: true, clearable: true, remote: true, remoteMethod: this.searchCustomers, loading: this.loading, placeholder: '选填；可搜索并指定客服' }, on: { input: value => { this.form.customer_user_id = Number(value || 0) } } }, this.activeCandidates.map(item => h('el-option', { key: item.user_id, props: { value: Number(item.user_id), label: this.optionLabel(item) } }))), true)
    ])
    const resultTable = () => h('div', { class: 'imgo-member-batch-result-scroll' }, [
      h('div', { class: 'imgo-member-batch-result-row is-header' }, ['账号', '密码', '姓名', '性别', '上级邀请码', '客服'].map(text => h('span', text))),
      ...this.results.map(item => h('div', { key: item.user_id, class: 'imgo-member-batch-result-row' }, [
        h('span', { class: 'is-account' }, item.account || '—'), h('span', item.password || '—'),
        h('span', item.realname || '—'), h('span', item.sex_label || '—'),
        h('span', item.parent_invite_code || '无'), h('span', item.customer_account || '未指定')
      ]))
    ])
    const progress = this.task ? h('section', { class: 'imgo-member-batch-progress' }, [
      h('div', { class: 'imgo-member-batch-progress-head' }, [
        h('div', [h('h3', this.taskStatusText), h('p', this.task.status === 'completed' ? '账号已经全部创建，可复制后保存。' : this.task.status === 'failed' ? '任务已停止，已创建的账号仍然有效。' : '正在后台逐个创建账号，请稍候。')]),
        h('el-tag', { props: { type: this.task.status === 'completed' ? 'success' : this.task.status === 'failed' ? 'danger' : 'primary', effect: 'dark' } }, this.taskStatusText)
      ]),
      h('el-progress', { props: { percentage: this.progressPercent, strokeWidth: 14, status: this.task.status === 'completed' ? 'success' : this.task.status === 'failed' ? 'exception' : undefined } }),
      h('div', { class: 'imgo-member-batch-metrics' }, [
        ['计划创建', Number(this.task.total_count || 0), '个'],
        ['已经创建', Number(this.task.success_count || 0), '个'],
        ['剩余数量', Math.max(0, Number(this.task.total_count || 0) - Number(this.task.completed_count || 0)), '个']
      ].map(metric => h('div', [h('span', metric[0]), h('strong', String(metric[1])), h('small', metric[2])]))),
      h('div', { class: 'imgo-member-batch-recovery' }, [
        h('i', { class: 'el-icon-refresh-right' }),
        h('span', '任务已保存到服务器，关闭弹窗或刷新网页后，再次点击“批量创建账号”即可恢复。'),
        h('code', `任务 ${String(this.task.task_id || '').slice(0, 10)}…`)
      ]),
      this.task.status === 'failed' ? h('el-alert', { props: { type: 'error', showIcon: true, closable: false, title: this.task.error_message || '创建任务已中断' } }) : null,
      this.results.length ? h('div', { class: 'imgo-member-batch-created' }, [h('div', { class: 'imgo-member-batch-created-title' }, `已生成账号（${this.results.length}）`), resultTable()]) : null
    ]) : null
    let footer
    if (!this.task) {
      footer = [
        h('el-button', { props: { disabled: this.submitting }, on: { click: () => this.close() } }, '取消'),
        h('el-button', { props: { type: 'primary', loading: this.submitting, disabled: this.loading }, on: { click: this.submit } }, `开始创建 ${Number(this.form.count || 0)} 个账号`)
      ]
    } else if (this.taskActive) {
      footer = [
        h('el-button', { props: { loading: this.polling }, on: { click: () => this.refreshTask(false) } }, '刷新进度'),
        h('el-button', { props: { type: 'primary' }, on: { click: () => this.close() } }, '关闭（后台继续）')
      ]
    } else {
      footer = [
        h('el-button', { on: { click: this.createAgain } }, '继续创建'),
        h('el-button', { props: { type: 'primary', loading: this.copying, disabled: !this.results.length }, on: { click: this.copyResults } }, '一键复制全部'),
        h('el-button', { on: { click: () => this.close() } }, '关闭')
      ]
    }
    return h('el-dialog', {
      class: 'imgo-member-batch-dialog-host',
      props: { title: '批量创建账号', visible: this.visible, width: '820px', appendToBody: true, closeOnClickModal: false, customClass: 'imgo-member-batch-dialog', beforeClose: this.close },
      on: { 'update:visible': value => { if (!value) this.close() } }
    }, [
      this.task ? progress : h('el-form', { directives: [{ name: 'loading', value: this.loading }], props: { labelPosition: 'top' } }, [form]),
      h('span', { slot: 'footer' }, footer)
    ])
  }
}
