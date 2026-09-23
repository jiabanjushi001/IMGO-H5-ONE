// Vue 2 distribution adapter: data orchestration and section composition.
const ImgoOverview = {
  data() { return { data: null, error: '', registration: 'month', onlineDays: 1, timer: null, busy: false } },
  mounted() { this.refresh(); this.timer = setInterval(this.refresh, 60000) },
  beforeDestroy() { clearInterval(this.timer) },
  methods: {
    async refresh() {
      if (this.busy) return
      this.busy = true
      try {
        const res = await this.$api.taskApi.getOverview({ online_days: this.onlineDays })
        if (res.code !== 0) throw Error(res.msg || '概况读取失败')
        this.data = res.data; this.error = ''
      } catch (e) { this.error = '概况数据加载失败，请重试。已有数据可能不是最新。' }
      finally { this.busy = false }
    },
    switchOnline(days) { if (this.busy) return; this.onlineDays = days; this.refresh() }
  },
  render(h) {
    const d = this.data, total = d ? d.totals : {}, today = d ? d.today : {}
    const card = (title, key, sub, color, icon, unit = '个') => h('section', { class: 'imgo-stat', style: { '--stat-color': color } }, [
      h('p', title), h('div', [h('strong', d ? String(total[key]) : '—'), h('span', ` ${unit}`)]), h('p', d ? sub : '正在读取'), h('i', { class: `imgo-stat-icon ${icon}` })
    ])
    const buttons = (items, selected, action) => h('el-button-group', items.map(([text, value]) => h('el-button', { props: { size: 'small', type: selected === value ? 'primary' : 'default', disabled: this.busy }, on: { click: () => action(value) } }, text)))
    const chart = (title, labels, series, extra = {}, controls) => h(ImgoOverviewChart, { props: { title, labels, series, ...extra } }, controls ? [h('div', { slot: 'controls' }, [controls])] : [])
    const series = (name, values, color) => ({ name, values: values.map(p => Number(p.value)), color })
    let charts = []
    if (d) {
      const reg = d['registration_' + this.registration]
      const samples = new Map(d.online.map(p => [Number(p.time), p]))
      const step = d.online_bucket_seconds, start = Math.floor(d.online_since / step) * step, end = Math.floor(d.generated_at / step) * step
      const labels = [], users = [], devices = []
      for (let t = start; t <= end; t += step) {
        const date = new Date(t * 1000)
        labels.push(new Intl.DateTimeFormat('zh-CN', { timeZone: 'Asia/Shanghai', ...(this.onlineDays === 1 ? { hour: '2-digit', minute: '2-digit', hour12: false } : { month: '2-digit', day: '2-digit', hour: '2-digit', hour12: false }) }).format(date))
        const p = samples.get(t); users.push(p ? Number(p.users) : null); devices.push(p ? Number(p.devices) : null)
      }
      charts = [
        chart('用户注册趋势', reg.map(p => p.label), [series('注册数', reg, '#409eff')], {}, buttons([['按月', 'month'], ['按日', 'day']], this.registration, v => { this.registration = v })),
        chart('在线趋势', labels, [{ name: '在线用户', values: users, color: '#409eff' }, { name: '在线设备（连接数）', values: devices, color: '#67c23a' }], { note: '每分钟采样；今日按5分钟峰值、近7/30天按小时峰值展示。未采集或停机时段留空。' }, buttons([['今日', 1], ['近7天', 7], ['近30天', 30]], this.onlineDays, this.switchOnline)),
        chart('消息发送趋势（近30天）', d.messages.map(p => p.label.slice(5)), [series('消息数', d.messages, '#e6a23c')]),
        chart('群聊 / 文件增长趋势（近12个月）', d.groups.map(p => p.label), [series('新增群聊', d.groups, '#409eff'), series('新增文件', d.files, '#e6a23c')], { bars: true })
      ]
    }
    return h('div', { class: 'imgo-overview' }, [
      h('div', {class: 'imgo-page-heading'}, [
        h('div', [h('h1', '系统概况'), h('p', [h('i', {class: 'imgo-live-dot'}), '掌握运行状态，让每一次连接清晰可见'])]),
        h('el-button', {props: {icon: 'el-icon-refresh', size: 'small', loading: this.busy}, on: {click: this.refresh}}, '刷新数据')
      ]),
      this.error ? h('el-alert', { props: { title: this.error, type: 'error', closable: false } }, [h('el-button', { on: { click: this.refresh } }, '重试')]) : null,
      h('div', { class: 'imgo-stats' }, [
        card('用户总数', 'users', `今日新增 +${today.users}`, '#409eff', 'el-icon-user'),
        card('在线用户', 'online_users', `今日采样峰值 ${today.peak_users}`, '#67c23a', 'el-icon-user-solid'),
        card('在线设备', 'online_devices', `今日采样峰值 ${today.peak_devices}`, '#13c2c2', 'el-icon-monitor'),
        card('群聊总数', 'groups', `今日创建 +${today.groups}`, '#e6a23c', 'el-icon-s-custom'),
        card('消息总数', 'messages', `今日 ${today.messages} 条`, '#8959d1', 'el-icon-chat-dot-round', '条'),
        card('文件总数', 'files', `今日上传 +${today.files}`, '#ff4785', 'el-icon-document')
      ]),
      h('div', { class: 'imgo-overview-middle' }, this.$slots.default),
      h('div', { class: 'imgo-charts' }, charts),
      h('p', { class: 'imgo-overview-footnote' }, '统计当前未删除的用户、有效群聊和文件、可见聊天消息（不含系统公告）。删除或清理后，相关历史统计也会变化。在线统计为当前 Go 实例的已认证连接，每60秒刷新。')
    ])
  }
}
