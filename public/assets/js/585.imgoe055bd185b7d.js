"use strict";(self["webpackChunkRaingad_IM"]=self["webpackChunkRaingad_IM"]||[]).push([[585],{4585:function(t,a,s){s.r(a),s.d(a,{default:function(){return d}});var e=function(){var t=this,a=t._self._c;return a("div",{staticClass:"pd-20"},[a("el-row",{attrs:{gutter:20}},[t.globalConfig&&t.globalConfig.demon_mode?a("el-col",{attrs:{span:10}},[a("el-card",{staticClass:"mb-20",attrs:{shadow:"hover",header:"欢迎"}},[a("div",{staticClass:"welcome"},[a("div",{staticClass:"logo"},[a("img",{attrs:{src:s(5080)}}),a("h2",[t._v("欢迎体验 "+t._s(t.$packageData.name))])]),a("div",{staticClass:"tips"},t._l(t.$packageData.funcList,(function(s){return a("div",{key:s.icon,staticClass:"tips-item"},[a("div",{staticClass:"tips-item-icon"},[a("i",{class:s.icon})]),a("div",{staticClass:"tips-item-message",domProps:{textContent:t._s(s.text)}})])})),0),a("div",{staticClass:"actions"},[a("router-link",{attrs:{to:"/chat"}},[a("el-button",{attrs:{type:"primary",icon:"el-icon-s-promotion",size:"large"}},[t._v("去聊天")])],1)],1)])])],1):t._e(),t.globalConfig&&t.globalConfig.demon_mode?a("el-col",{attrs:{span:8}},[a("el-card",{staticClass:"item-background mb-20",attrs:{shadow:"hover",header:"关于项目"}},[a("p",[t._v(t._s(t.$packageData.name)+"是一个"),a("b",{staticClass:"c-red"},[t._v("开源的即时通信demo，主要用于学习交流，为大家提供即时通讯的开发思路")]),t._v("，许多功能需要自行开发，开发的初衷旨在快速建立企业内部通讯系统、内网交流、社区交流。不建议用于商业用途，如确有需要商用，请联系作者授权，自行开发代码量必须要高于原代码量的30%以上，重构UI，并注明相关的版权问题。")]),a("div",{staticClass:"mt-15 ml-15 mb-15"},[t._v(" 前端地址："),a("a",{attrs:{href:t.$packageData.frontUrl,target:"_blank"}},[a("el-image",{attrs:{src:t.$packageData.frontUrl+"/badge/star.svg?theme=white",alt:"star"}})],1)]),a("div",{staticClass:"ml-15 mb-15"},[t._v(" 后端地址："),a("a",{attrs:{href:t.$packageData.backstageUrl,target:"_blank"}},[a("el-image",{attrs:{src:t.$packageData.backstageUrl+"/badge/star.svg?theme=dark",alt:"star"}})],1)])])],1):t._e(),t.globalConfig&&t.globalConfig.demon_mode?a("el-col",{attrs:{span:6}},[a("el-card",{staticClass:"mb-20",attrs:{shadow:"hover",header:"数据概览"}},[a("div",{staticClass:"mb-15"},[t._v("用户总数：xxxx")]),a("div",{staticClass:"mb-15"},[t._v("群聊总数：xxxx")]),a("div",{staticClass:"mb-15"},[t._v("文件总数：xxxx")])])],1):t._e(),a("el-col",{attrs:{span:12}},[a("el-card",{staticClass:"mb-20",attrs:{shadow:"hover",header:""}},[a("div",{attrs:{slot:"header"},slot:"header"},[a("span",[t._v("系统公告")]),a("el-button",{staticStyle:{float:"right",padding:"3px 0"},attrs:{type:"text"},on:{click:function(a){t.noticeBox=!0}}},[t._v("发布公告")])],1),t._l(t.noticeList,(function(s,e){return a("div",{key:e,staticClass:"lz-flex lz-space-between"},[a("div",{staticClass:"mb-5 text-overflow cur-handle",staticStyle:{width:"70%"},on:{click:function(a){return t.viewNotice(s)}}},[a("span",{staticClass:"el-icon el-icon-collection-tag mr-5"}),t._v(" "+t._s(s.title)+" ")]),a("div",{staticClass:"imgo-notice-actions"},[a("span",{staticClass:"c-999"},[t._v(t._s(s.create_time))]),a("el-button",{attrs:{type:"text",size:"small"},staticClass:"imgo-notice-delete",on:{click:function(e){e.stopPropagation();return t.imgoDeleteNotice(s)}}},[t._v("删除")])],1)])})),a("el-pagination",{staticClass:"mt-10",attrs:{background:"",layout:"prev, pager, next","page-size":t.noticeParam.limit,"current-page":t.noticeParam.page,total:t.noticeTotal},on:{"update:currentPage":function(a){return t.$set(t.noticeParam,"page",a)},"update:current-page":function(a){return t.$set(t.noticeParam,"page",a)},"current-change":t.getNoticeList}})],2),a("el-dialog",{attrs:{title:(t.notice.msgId?"编辑":"发布")+"公告",width:"500px",visible:t.noticeBox},on:{"update:visible":function(a){t.noticeBox=a}}},[a("el-form",{attrs:{model:t.notice}},[a("el-form-item",{attrs:{label:"公告标题"}},[a("el-input",{model:{value:t.notice.title,callback:function(a){t.$set(t.notice,"title",a)},expression:"notice.title"}})],1),a("el-form-item",{attrs:{label:"公告内容"}},[a("el-input",{attrs:{type:"textarea",autosize:{minRows:10,maxRows:20}},model:{value:t.notice.content,callback:function(a){t.$set(t.notice,"content",a)},expression:"notice.content"}})],1)],1),a("div",{staticClass:"dialog-footer",attrs:{slot:"footer"},slot:"footer"},[a("el-button",{on:{click:t.cancelPublish}},[t._v("取 消")]),a("el-button",{attrs:{type:"primary"},on:{click:t.publishNotice}},[t._v(t._s(t.notice.msgId?"更新":"发布"))])],1)],1)],1),a("el-col",{attrs:{span:12}},[a("el-card",{directives:[{name:"loading",rawName:"v-loading",value:t.loading,expression:"loading"}],staticClass:"task task-item mb-20",attrs:{shadow:"hover"}},[a("div",{attrs:{slot:"header"},slot:"header"},[a("span",[t._v("系统服务")]),a("span",{staticClass:"handler",staticStyle:{float:"right","margin-top":"-3px"}},[a("i",{staticClass:"f-24 c-999 cur-handle",class:t.taskStatus?"el-icon-video-pause stop-task":"el-icon-video-play start-task",staticStyle:{padding:"3px"},attrs:{type:"primary"},on:{click:t.startService}})])]),a("el-alert",{attrs:{type:"warning",title:"系统服务使用要求运行的PHP的版本必须是默认的，并且可以直接执行PHP命令。如果启动失败可能是某些函数被禁用或者runtime的目录没有写入权限，可以在终端中运行 ‘php think task start’ 来调试程序的错误。","show-icon":"",closable:!1}}),t._l(t.taskList,(function(s,e){return a("div",{key:e,staticClass:"lz-flex lz-space-between mt-10 mb-10 lz-align-items-center"},[a("div",{staticClass:"task-name el-icon-timer"},[t._v(" "+t._s(s.remark)+" ")]),a("div",{staticClass:"el-icon-alarm-clock"},[t._v(" "+t._s(s.started)+" ")]),"active"==s.status?a("div",{staticClass:"c-green"},[t._v("运行中")]):a("div",{staticClass:"c-red"},[t._v("未启动")]),a("el-button",{staticClass:"ml-10",attrs:{size:"mini",type:"text"},on:{click:function(a){return t.showLog(s.name)}}},[t._v("日志")])],1)})),a("el-dialog",{attrs:{width:"900px",title:"运行日志",visible:t.dialogTableVisible},on:{"update:visible":function(a){t.dialogTableVisible=a}}},[a("el-button",{on:{click:t.clearTaskLog}},[t._v("清除进程日志")]),a("div",{staticClass:"mt-10",staticStyle:{height:"500px"}},[a("el-scrollbar",[a("div",{staticClass:"task-log pd-10"},[t._v(t._s(t.taskLog))])])],1)],1)],2)],1)],1)],1)},i=[],o=s(3822),l={components:{},computed:{...(0,o.rn)({globalConfig:t=>t.globalConfig})},data(){return{loading:!1,taskStatus:!1,taskList:[],curName:"",dialogTableVisible:!1,taskLog:"",noticeBox:!1,noticeList:[],notice:{msgId:0,title:"",content:""},noticeParam:{page:1,limit:10},noticeTotal:0,task:[{name:"im_task_schedule",started:"--",status:"stop",remark:"计划任务"},{name:"im_task_queue",started:"--",status:"stop",remark:"消息队列"},{name:"im_task_worker",started:"--",status:"stop",remark:"消息推送"}]}},mounted(){this.resetTask(),this.getTaskList(),this.getNoticeList()},methods:{resetTask(){let t=this.task;this.taskList=t},getTaskList(){this.$api.taskApi.getTaskList().then((t=>{400==t.code?this.taskStatus=!1:0==t.code&&(this.taskStatus=!0,this.taskList=t.data)}))},getNoticeList(){this.$api.commonApi.getNoticeList(this.noticeParam).then((t=>{0==t.code&&(this.noticeList=t.data,this.noticeTotal=t.count)}))},startService(){this.loading=!0,0==this.taskStatus?this.$api.taskApi.startTask().then((t=>{this.loading=!1,0==t.code&&this.getTaskList()})):this.$confirm("确定要停止服务吗？","提示",{confirmButtonText:"确定",cancelButtonText:"取消",type:"warning"}).then((()=>{this.$api.taskApi.stopTask().then((t=>{this.loading=!1,0==t.code&&(this.taskStatus=!1,this.resetTask())}))})).catch((()=>{this.loading=!1,this.$message({type:"info",message:"已取消停止"})}))},showLog(t){this.curName=t,this.$api.taskApi.getTaskLog({name:t}).then((t=>{if(0==t.code){if(""==t.data)return this.$message.error("暂无日志");this.dialogTableVisible=!0,this.taskLog=t.data}}))},clearTaskLog(){this.$confirm("确定要清除日志吗？","提示",{confirmButtonText:"确定",cancelButtonText:"取消",type:"warning"}).then((()=>{this.$api.taskApi.clearTaskLog({name:this.curName}).then((t=>{0==t.code&&(this.dialogTableVisible=!1,this.taskLog="")}))})).catch((()=>{}))},publishNotice(){this.$api.commonApi.publishNotice(this.notice).then((t=>{this.noticeBox=!1,0==t.code&&(this.cancelPublish(),this.getNoticeList())}))},cancelPublish(){this.noticeBox=!1,this.notice={msgId:0,title:"",content:""}},viewNotice(t){this.noticeBox=!0,this.notice={msgId:t.msg_id,title:t.extends.title,content:t.extends.notice??""}}}},n=l,c=s(1001),r=(0,c.Z)(n,e,i,!1,null,"3f58b2b1",null),d=r.exports;/* IMGO_MAINTENANCE_BEGIN */
// Vue 2: retain the original notice editor and list; add a guarded delete action.
d.methods.imgoDeleteNotice = async function (notice) {
  if (this._imgoNoticeDeleting) return;
  try {
    await this.$confirm('确定删除这条系统公告吗？', '删除公告', {
      confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning'
    });
  } catch (_) { return; }
  this._imgoNoticeDeleting = true;
  try {
    const result = await this.$api.commonApi.delNotice({id: notice.msg_id});
    if (result.code !== 0) throw new Error(result.msg || '删除失败');
    if (this.noticeList.length === 1 && this.noticeParam.page > 1) this.noticeParam.page--;
    this.getNoticeList();
    this.$message.success('公告已删除');
  } catch (error) {
    this.$message.error(error.message || '删除失败，请重试');
  } finally { this._imgoNoticeDeleting = false; }
};
;
// Vue 2 Options API: matches the bundled management application.
const ImgoMaintenancePanel = {
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
;
// Presentational SVG chart. Null values are gaps, never fabricated zeros.
const ImgoOverviewChart = {
  props: { title: String, labels: Array, series: Array, bars: Boolean, note: String },
  render(h) {
    const labels = this.labels || [], series = this.series || []
    const max = Math.max(1, ...series.flatMap(s => s.values.filter(v => v != null)))
    const top = Math.max(1, Math.ceil(max / 4)) * 4
    const x = i => 48 + i * 680 / Math.max(1, labels.length - 1)
    const y = v => 225 - Number(v) / top * 185
    const marks = []
    for (let i = 0; i <= 4; i++) {
      const v = top * i / 4
      marks.push(h('line', { attrs: { x1: 48, x2: 728, y1: y(v), y2: y(v), stroke: '#e5ebf3' } }), h('text', { attrs: { x: 37, y: y(v) + 4, 'text-anchor': 'end' } }, String(v)))
    }
    labels.forEach((label, i) => {
      if (i % Math.max(1, Math.ceil(labels.length / 6)) === 0 || i === labels.length - 1) marks.push(h('text', { attrs: { x: x(i), y: 249, 'text-anchor': 'middle' } }, label))
    })
    series.forEach((s, si) => {
      if (this.bars) {
        const w = Math.min(18, 600 / Math.max(1, labels.length) / series.length)
        s.values.forEach((v, i) => { if (v != null) marks.push(h('rect', { attrs: { x: x(i) + (si - series.length / 2) * w, y: y(v), width: w - 2, height: 225 - y(v), rx: 3, fill: s.color } }, [h('title', `${labels[i]} ${s.name}：${v}`)])) })
      } else {
        let path = '', connected = false
        s.values.forEach((v, i) => { if (v == null) { connected = false; return }; path += `${connected ? 'L' : 'M'}${x(i)},${y(v)} `; connected = true })
        marks.push(h('path', { attrs: { d: path, fill: 'none', stroke: s.color, 'stroke-width': 2 } }))
        s.values.forEach((v, i) => { if (v != null) marks.push(h('circle', { attrs: { cx: x(i), cy: y(v), r: 3, fill: s.color } }, [h('title', `${labels[i]} ${s.name}：${v}`)])) })
      }
    })
    return h('section', { class: 'imgo-chart' }, [
      h('header', [h('span', this.title), h('div', this.$slots.controls)]),
      h('div', { class: 'imgo-chart-legend' }, series.map(s => h('span', [h('i', { style: { background: s.color } }), s.name]))),
      h('svg', { attrs: { viewBox: '0 0 770 275', role: 'img', 'aria-label': this.title } }, [h('title', this.title), ...marks]),
      h('p', { class: 'imgo-chart-note' }, this.note || '统计时区：北京时间；鼠标移至数据点查看数值。')
    ])
  }
}

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
;
// Vue 2 component for the existing compiled management application.
// The authorized management detail supplies the full card number for editing.
const ImgoBankPanel = {
  name: 'ImgoBankPanel',
  data() {
    return {
      rows: [], total: 0, page: 1, keywords: '', filter: '',
      loading: false, saving: false, dialog: false,
      draft: { user_id: 0, version: 0, receipt_name: '', receipt_account: '', original_account: '', bank_name: '', branch_name: '', status: 0, remark: '' }
    }
  },
  mounted() { this.refresh() },
  methods: {
    async refresh() {
      this.loading = true
      try {
        const res = await this.$api.bankApi.index({ page: this.page, limit: 20, keywords: this.keywords.trim(), status: this.filter })
        if (res.code !== 0) throw Error(res.msg || '读取绑卡列表失败')
        this.rows = Array.isArray(res.data) ? res.data : []
        this.total = Number(res.count || 0)
      } catch (error) {
        this.$message.error(error.message || '读取绑卡列表失败')
      } finally { this.loading = false }
    },
    search() { this.page = 1; this.refresh() },
    async open(row) {
      try {
        const res = await this.$api.bankApi.detail({ user_id: row.user_id })
        if (res.code !== 0) throw Error(res.msg || '读取绑卡资料失败')
        const card = res.data
        this.draft = {
          user_id: Number(card.user_id), version: Number(card.version),
          receipt_name: card.receipt_name || '', receipt_account: card.receipt_account || '', original_account: card.receipt_account || '',
          bank_name: card.bank_name || '', branch_name: card.branch_name || '',
          status: Number(card.status), remark: card.remark || ''
        }
        this.dialog = true
      } catch (error) { this.$message.error(error.message || '读取绑卡资料失败') }
    },
    async save() {
      const d = this.draft, account = d.receipt_account.replace(/\s/g, '')
      if (d.receipt_name.trim().length < 2 || !/^\d{12,30}$/.test(account) ||
          (d.bank_name.trim() && (d.bank_name.trim().length < 2 || d.bank_name.trim().length > 120)) ||
          (d.branch_name.trim() && (d.branch_name.trim().length < 2 || d.branch_name.trim().length > 120)) || d.remark.length > 500) {
        this.$message.warning('请检查收款姓名、卡号、收款银行、支行名称和备注')
        return
      }
      this.saving = true
      try {
        const payload = {
          user_id: d.user_id, version: d.version,
          receipt_name: d.receipt_name.trim(),
          bank_name: d.bank_name.trim(), branch_name: d.branch_name.trim(),
          status: d.status, remark: d.remark.trim()
        }
        if (account !== d.original_account) payload.receipt_account = account
        const res = await this.$api.bankApi.edit(payload)
        if (res.code !== 0) throw Error(res.msg || '保存失败')
        this.dialog = false
        this.$message.success('绑卡资料已更新')
        await this.refresh()
      } catch (error) { this.$message.error(error.message || '保存失败') }
      finally { this.saving = false }
    },
    statusLabel(value) { return ['未处理', '同意', '拒绝'][Number(value)] || '未知' },
    date(value) { return value ? new Date(Number(value) * 1000).toLocaleString('zh-CN', { hour12: false }) : '—' }
  },
  render(h) {
    const field = (label, key, options = {}) => h('el-form-item', { props: { label, labelWidth: '100px' } }, [
      h('el-input', { props: { value: this.draft[key], placeholder: options.placeholder || '', type: options.type || 'text', maxlength: options.maxlength, showWordLimit: !!options.maxlength }, on: { input: value => { this.draft[key] = value } } })
    ])
    const column = (label, prop, width) => h('el-table-column', { props: { label, prop, width } })
    const dialog = h('el-dialog', {
      props: { title: '编辑绑卡', visible: this.dialog, width: '520px', closeOnClickModal: false },
      on: { close: () => { this.dialog = false; this.draft.receipt_account = ''; this.draft.original_account = '' } }
    }, this.dialog ? [
      h('el-form', { class: 'imgo-bank-form' }, [
        h('el-form-item', { props: { label: '用户ID', labelWidth: '100px' } }, String(this.draft.user_id)),
        field('收款姓名', 'receipt_name', { placeholder: '银行卡持有人姓名', maxlength: 40 }),
        field('卡号', 'receipt_account', { placeholder: '填写12至30位银行卡号', maxlength: 30 }),
        field('收款银行', 'bank_name', { placeholder: '请输入开户行', maxlength: 120 }),
        field('支行名称', 'branch_name', { placeholder: '请输入支行名称', maxlength: 120 }),
        h('el-form-item', { props: { label: '状态', labelWidth: '100px' } }, [
          h('el-radio-group', { props: { value: this.draft.status }, on: { input: value => { this.draft.status = Number(value) } } },
            [0, 1, 2].map(value => h('el-radio', { key: value, props: { label: value } }, this.statusLabel(value))))
        ]),
        field('备注', 'remark', { type: 'textarea', maxlength: 500, placeholder: '处理说明（用户可见）' })
      ]),
      h('span', { slot: 'footer' }, [
        h('el-button', { on: { click: () => { this.dialog = false } } }, '取消'),
        h('el-button', { props: { type: 'primary', loading: this.saving }, on: { click: this.save } }, '保存')
      ])
    ] : [])
    return h('section', { class: 'imgo-bank' }, [
      h('div', { class: 'imgo-bank-heading' }, [
        h('div', [h('h1', '绑卡管理'), h('p', '查看用户提交的收款资料，编辑并设置处理状态')]),
        h('el-button', { props: { icon: 'el-icon-refresh', loading: this.loading }, on: { click: this.refresh } }, '刷新')
      ]),
      h('div', { class: 'imgo-bank-toolbar' }, [
        h('el-input', { props: { value: this.keywords, placeholder: '搜索账号、姓名、银行或支行', clearable: true }, on: { input: value => { this.keywords = value }, keyup: event => { if (event.key === 'Enter') this.search() } } }),
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
        column('收款姓名', 'receipt_name', '140'), column('收款账号', 'receipt_account_masked', '200'),
        column('收款银行', 'bank_name', '180'), column('支行名称', 'branch_name', '220'),
        h('el-table-column', { props: { label: '状态', width: '100' }, scopedSlots: { default: scope => h('el-tag', { props: { type: ['warning', 'success', 'danger'][Number(scope.row.status)] } }, this.statusLabel(scope.row.status)) } }),
        column('备注', 'remark'),
        h('el-table-column', { props: { label: '更新时间', width: '180' }, scopedSlots: { default: scope => this.date(scope.row.updated_at) } }),
        h('el-table-column', { props: { label: '操作', width: '85', fixed: 'right', align: 'center' }, scopedSlots: { default: scope => h('el-button', { props: { type: 'text' }, on: { click: () => this.open(scope.row) } }, '编辑') } })
      ]),
      h('el-pagination', { class: 'imgo-bank-pagination', props: { currentPage: this.page, pageSize: 20, total: this.total, layout: 'total, prev, pager, next' }, on: { 'current-change': page => { this.page = page; this.refresh() } } }),
      dialog
    ])
  }
}
;
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
;
const ImgoRolePanel = {
  name: 'ImgoRolePanel',
  data() {
    return {
      roles: [], permissions: [], activeRoleID: null, loading: false, saving: false,
      form: { role_id: 0, name: '', remark: '', status: 1, agent_mode: 1, role_code: '', permissions: [], builtin: false }
    }
  },
  mounted() { this.load() },
  methods: {
    async load(preferredID) {
      this.loading = true
      try {
        const [rolesResult, permissionsResult] = await Promise.all([
          this.$api.roleApi.index({}), this.$api.roleApi.permissions({})
        ])
        if (rolesResult.code !== 0) throw Error(rolesResult.msg || '读取角色失败')
        if (permissionsResult.code !== 0) throw Error(permissionsResult.msg || '读取权限失败')
        this.roles = Array.isArray(rolesResult.data) ? rolesResult.data : []
        this.permissions = Array.isArray(permissionsResult.data) ? permissionsResult.data : []
        const wantedID = preferredID !== undefined && preferredID !== null ? Number(preferredID) : this.activeRoleID
        const selected = wantedID !== null ? this.roles.find(role => Number(role.role_id) === Number(wantedID)) : null
        if (selected) this.select(selected)
        else if (this.roles.length) this.select(this.roles[0])
        else this.create()
      } catch (error) { this.$message.error(error.message || '读取角色失败') }
      finally { this.loading = false }
    },
    create() {
      this.activeRoleID = null
      this.form = { role_id: 0, name: '', remark: '', status: 1, agent_mode: 1, role_code: '', permissions: [], builtin: false }
    },
    select(role) {
      this.activeRoleID = Number(role.role_id)
      this.form = {
        role_id: Number(role.role_id), name: role.name || '', remark: role.remark || '',
        status: Number(role.status), agent_mode: Number(role.agent_mode || 0), role_code: role.role_code || '',
        permissions: Array.isArray(role.permissions) ? [...role.permissions] : [],
        builtin: Boolean(role.builtin)
      }
    },
    async save() {
      if (this.form.builtin) return
      const name = this.form.name.trim()
      if (!name || name.length > 64) { this.$message.warning('请输入1-64个字的角色名称'); return }
      if (this.form.remark.length > 255) { this.$message.warning('角色备注最多255个字'); return }
      this.saving = true
      try {
        const result = await this.$api.roleApi.save({ ...this.form, name, remark: this.form.remark.trim() })
        if (result.code !== 0) throw Error(result.msg || '保存角色失败')
        this.$message.success('角色已保存')
        window.__imgoRoleOptionsPromise = null
        await this.load(Number(result.data.role_id))
      } catch (error) { this.$message.error(error.message || '保存角色失败') }
      finally { this.saving = false }
    },
    async remove() {
      if (!this.form.role_id || this.form.builtin || this.form.role_code === 'mentor') return
      try {
        await this.$confirm('确定删除该角色？已绑定用户的角色不能删除。', '删除角色', { type: 'warning' })
        const result = await this.$api.roleApi.del({ role_id: this.form.role_id })
        if (result.code !== 0) throw Error(result.msg || '删除角色失败')
        this.$message.success('角色已删除')
        window.__imgoRoleOptionsPromise = null
        this.activeRoleID = null
        await this.load()
      } catch (error) {
        if (error !== 'cancel' && error !== 'close') this.$message.error(error.message || '删除角色失败')
      }
    }
  },
  render(h) {
    const roleCards = this.roles.map(role => h('button', {
      key: role.role_id,
      class: ['imgo-role-card', { 'is-active': Number(role.role_id) === this.activeRoleID }],
      attrs: { type: 'button' }, on: { click: () => this.select(role) }
    }, [
      h('span', { class: 'imgo-role-card-name' }, role.name),
      h('span', { class: 'imgo-role-card-count' }, `${Number(role.user_count || 0)} 人`),
      h('span', { class: ['imgo-role-card-status', Number(role.status) === 1 ? 'is-enabled' : 'is-disabled'] }, role.builtin ? '系统角色' : (Number(role.status) === 1 ? '已启用' : '已禁用'))
    ]))
    const permissionBoxes = this.permissions.map(permission => h('el-checkbox', {
      key: permission.permission_key, props: { label: permission.permission_key }
    }, [h('span', { class: 'imgo-role-permission-name' }, permission.name), h('small', permission.menu_path)]))
    return h('section', { class: 'imgo-role-page' }, [
      h('header', { class: 'imgo-role-heading' }, [
        h('div', [h('h1', '角色'), h('p', '为后台人员分配一个角色，并同时限制菜单和接口权限。')]),
        h('el-button', { props: { type: 'primary', icon: 'el-icon-plus' }, on: { click: this.create } }, '新增角色')
      ]),
      h('div', { class: 'imgo-role-layout', directives: [{ name: 'loading', value: this.loading }] }, [
        h('aside', { class: 'imgo-role-list' }, roleCards.length ? roleCards : [h('div', { class: 'imgo-role-empty' }, '暂无角色')]),
        h('main', { class: 'imgo-role-editor' }, [
          h('div', { class: 'imgo-role-editor-title' }, this.form.builtin ? this.form.name : (this.form.role_id ? '编辑角色' : '新增角色')),
          h('el-form', { props: { labelWidth: '92px' } }, [
            h('el-form-item', { props: { label: '角色名称', required: !this.form.builtin } }, [h('el-input', { props: { value: this.form.name, disabled: this.form.builtin, maxlength: 64, showWordLimit: !this.form.builtin, placeholder: '例如：客服、群聊管理员' }, on: { input: value => { this.form.name = value } } })]),
            h('el-form-item', { props: { label: '角色状态' } }, [h('el-switch', { props: { value: this.form.status, disabled: this.form.builtin, activeValue: 1, inactiveValue: 0, activeText: '启用', inactiveText: '禁用' }, on: { input: value => { this.form.status = Number(value) } } })]),
            h('el-form-item', { props: { label: '代理模式' } }, [
              h('el-switch', { props: { value: this.form.agent_mode, disabled: this.form.builtin, activeValue: 1, inactiveValue: 0, activeText: '开启', inactiveText: '关闭' }, on: { input: value => { this.form.agent_mode = Number(value) } } }),
              h('p', { class: 'imgo-role-agent-help' }, '开启后，该角色只能管理自己的邀请下级')
            ]),
            h('el-form-item', { props: { label: '角色备注' } }, [h('el-input', { props: { value: this.form.remark, disabled: this.form.builtin, type: 'textarea', rows: 3, maxlength: 255, showWordLimit: !this.form.builtin, placeholder: '说明该角色的使用范围' }, on: { input: value => { this.form.remark = value } } })]),
            h('el-form-item', { props: { label: '菜单权限' } }, [h('el-checkbox-group', { class: 'imgo-role-permissions', props: { value: this.form.permissions, disabled: this.form.builtin }, on: { input: value => { this.form.permissions = value } } }, permissionBoxes)])
          ]),
          this.form.builtin ? h('div', { class: 'imgo-role-system-note' }, '系统角色为固定权限，不可编辑或删除。') : h('div', { class: 'imgo-role-actions' }, [
            this.form.role_id && this.form.role_code !== 'mentor' ? h('el-button', { props: { type: 'danger', plain: true }, on: { click: this.remove } }, '删除角色') : null,
            h('el-button', { props: { type: 'primary', loading: this.saving }, on: { click: this.save } }, '保存角色')
          ])
        ])
      ])
    ])
  }
}
;
const LegacyManagement = d; d = { name: 'ImgoManagement', render(h) { const bank = this.$route.path === '/manage/bank', finance = this.$route.path.startsWith('/manage/finance/'), role = this.$route.path === '/manage/role'; return h('div', {class: 'imgo-management'}, [role ? h(ImgoRolePanel) : finance ? h(ImgoFinanceShell) : bank ? h(ImgoBankPanel) : h(ImgoOverview, [h(LegacyManagement), h(ImgoMaintenancePanel)])]); } };
/* IMGO_MAINTENANCE_END */},5080:function(t,a,s){t.exports=s.p+"assets/img/logo.e8099414.png"}}]);