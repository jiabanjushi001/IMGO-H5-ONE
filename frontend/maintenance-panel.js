// Vue 2 Options API: matches the bundled management application.
export default {
  name: 'ImgoMaintenancePanel',
  data() {
    return {
      loading: false, ready: false, settings: {},
      draft: { interval_minutes: 60, retention_days: 30 },
      logVisible: false, log: '', timer: null
    }
  },
  mounted() {
    this.refresh(true)
    this.timer = setInterval(() => this.refresh(false), 15000)
  },
  beforeDestroy() { clearInterval(this.timer) },
  methods: {
    async refresh(reset) {
      try {
        const res = await this.$api.taskApi.getTaskList()
        if (res.code !== 0) return
        const task = res.data.find(item => item.name === 'schedule')
        if (!task || !task.settings) return
        this.settings = task.settings
        if (reset) this.draft = {
          interval_minutes: Number(task.settings.interval_minutes),
          retention_days: Number(task.settings.retention_days)
        }
        this.ready = true
      } catch (error) {
        if (reset) this.$message.error('无法读取清理任务设置，请刷新重试')
      }
    },
    async save(action) {
      if (action !== 'stop' && (!Number.isInteger(this.draft.interval_minutes) || this.draft.interval_minutes < 1 || this.draft.interval_minutes > 43200 ||
          !Number.isInteger(this.draft.retention_days) || this.draft.retention_days < 1 || this.draft.retention_days > 3650)) {
        this.$message.error('请输入有效的执行间隔和消息保留天数')
        return
      }
      this.loading = true
      try {
        const payload = { ...this.draft, clear_messages: true }
        const api = this.$api.taskApi
        const res = action === 'start' ? await api.startTask(payload)
          : action === 'stop' ? await api.stopTask() : await api.setTaskConfig(payload)
        if (res.code === 0) {
          await this.refresh(true)
          this.$message.success(action === 'stop' ? '清理任务已停止' : action === 'start' ? '已保存并开启定时清理' : '清理设置已保存')
        }
      } catch (error) {
        this.$message.error('保存失败，请检查连接后重试')
      } finally { this.loading = false }
    },
    time(value) { return value ? new Date(Number(value) * 1000).toLocaleString() : '尚无记录' },
    async showLog() {
      try {
        const res = await this.$api.taskApi.getTaskLog({ name: 'schedule' })
        if (res.code === 0) { this.log = res.data || '任务尚未执行'; this.logVisible = true }
      } catch (error) { this.$message.error('读取日志失败') }
    }
  },
  render(h) {
    const running = !!this.settings.enabled
    const disabled = this.loading || !this.ready
    const button = (text, type, action) => h('el-button', {
      props: { type, disabled, loading: this.loading }, on: { click: () => this.save(action) }
    }, text)
    const field = (title, key, max, unit) => h('el-form-item', { props: { label: title } }, [
      h('el-input-number', {
        props: { value: this.draft[key], min: 1, max, precision: 0, disabled },
        on: { input: value => { this.draft[key] = value } }
      }), h('span', { class: 'imgo-unit' }, unit)
    ])
    return h('section', { class: 'imgo-maintenance' }, [
      h('el-card', { props: { shadow: 'never' } }, [
        h('div', { slot: 'header', class: 'imgo-task-header' }, [
          h('strong', 'Go 定时清理'),
          h('el-tag', { props: { type: running ? 'success' : 'info' } }, this.ready ? (running ? '已开启' : '未开启') : '读取中')
        ]),
        h('p', { class: 'imgo-task-help' }, '开启后定期清理过期登录会话，并隐藏超过保留天数的聊天消息。系统公告和实体附件保留。'),
        this.ready ? h('el-form', { props: { labelPosition: 'top' }, class: 'imgo-task-form' }, [
          field('多久执行一次', 'interval_minutes', 43200, '分钟'),
          field('消息保留多久', 'retention_days', 3650, '天')
        ]) : h('p', '正在读取设置…'),
        h('p', { class: 'imgo-task-help' }, '例如：每 60 分钟执行一次，保留最近 30 天的消息。保存或开启后，等待一个执行间隔再首次清理；设置在服务重启后仍生效。'),
        h('div', { class: 'imgo-task-actions' }, [
          running ? button('停止清理', 'danger', 'stop') : button('保存并开启', 'primary', 'start'),
          button('保存设置', 'default', 'save'),
          h('el-button', { props: { disabled }, on: { click: this.showLog } }, '查看日志')
        ]),
        h('dl', { class: 'imgo-task-times' }, [
          h('dt', '上次执行'), h('dd', this.time(this.settings.last_run_at)),
          h('dt', '下次执行'), h('dd', running ? (this.settings.next_run_at ? this.time(this.settings.next_run_at) : '等待调度') : '任务未开启'),
          h('dt', '上次结果'), h('dd', this.settings.last_run_at ? `清理 ${this.settings.last_sessions || 0} 条会话，隐藏 ${this.settings.last_messages || 0} 条消息` : '尚未执行')
        ]),
        running && !this.settings.clear_messages ? h('el-alert', { props: { type: 'warning', closable: false, title: '当前聊天配置关闭了消息清理，仅清理过期会话。点击“保存设置”可按上述天数启用消息清理。' } }) : null,
        this.settings.last_error ? h('el-alert', { props: { type: 'error', closable: false, title: this.settings.last_error } }) : null,
        h('p', { class: 'imgo-task-help' }, 'WebSocket 随 Go 主服务运行，不受此开关影响。执行时间可能有约 10 秒的调度偏差。'),
        h('el-dialog', { props: { title: '清理日志（最近 20 次）', visible: this.logVisible, width: '80%' }, on: { 'update:visible': value => { this.logVisible = value } } }, [
          h('pre', { class: 'imgo-task-log' }, this.log)
        ])
      ])
    ])
  }
}
