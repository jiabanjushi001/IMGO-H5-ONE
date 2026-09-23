// Private-chat profile for every signed-in web user.
component.components.ImgoChatProfile = {
  name: 'ImgoChatProfile',
  props: {contact: {type: Object, required: true}},
  data() { return {profile: null, loading: false, failed: false}; },
  watch: {
    'contact.id': {immediate: true, handler() { this.load(); }},
    '$store.state.socketAction': function (event) {
      if (event && event.type === 'userIPChanged' && event.data &&
          String(event.data.id) === String(this.contact && this.contact.id)) this.load();
    }
  },
  methods: {
    async load() {
      const id = this.contact && this.contact.id;
      const request = (this._request || 0) + 1;
      this._request = request;
      this.profile = null; this.failed = false; this.loading = false;
      if (!/^\d+$/.test(String(id)) || !Number.isSafeInteger(Number(id)) || Number(id) <= 0 ||
          (this.contact.is_group != null && Number(this.contact.is_group) !== 0)) return;
      this.loading = true;
      try {
        const result = await this.$api.imApi.getUserInfo({user_id: id});
        if (this._isDestroyed || request !== this._request) return;
        if (Number(result.code) !== 0) throw new Error('load failed');
        this.profile = result.data;
      } catch (_) { if (request === this._request && !this._isDestroyed) this.failed = true; }
      finally { if (request === this._request && !this._isDestroyed) this.loading = false; }
    },
    time(value) {
      if (!value || value === '0') return '暂无';
      const date = /^\d+$/.test(String(value)) ? new Date(Number(value) * 1000) : new Date(value);
      return Number.isNaN(date.getTime()) ? '暂无' : date.toLocaleString('zh-CN', {hour12: false});
    }
  },
  render(h) {
    const p = this.profile || {}, empty = value => value || '暂无';
    const nickname = this.contact.displayName || p.realname || '';
    const name = p.account ? p.account + '（' + nickname + '）' : nickname;
    const region = value => [...new Set(String(value || '暂无').split(/\s+/))].join(' ');
    const chatIP = p.last_chat_ip || p.last_login_ip;
    const chatArea = p.last_chat_ip ? p.chat_location : p.location;
    const row = (label, value) => h('div', {class: 'imgo-profile-field'}, [h('span', label), h('strong', String(value))]);
    const section = (label, time, ip, area) => h('section', {class: 'imgo-profile-section'}, [
      h('h4', label), row('时间', this.time(time)), row('IP 地址', empty(ip)), row('地区', region(area))
    ]);
    return h('div', {class: 'imgo-chat-profile'}, [
      h('div', {class: 'imgo-chat-profile-name', attrs: {title: name}}, [
        h('i', {class: ['imgo-profile-status', this.contact.is_online ? 'is-online' : ''], attrs: {title: this.contact.is_online ? '在线' : '离线'}}),
        h('strong', p.account || nickname), p.account ? h('span', {class: 'imgo-profile-nickname'}, '（' + nickname + '）') : null
      ]),
      this.loading ? h('span', {class: 'imgo-profile-meta'}, '加载中…') :
      this.failed ? h('button', {class: 'imgo-profile-trigger', on: {click: this.load}}, '重试资料') :
      h('div', {class: 'imgo-profile-actions'}, [
        h('span', {class: 'imgo-profile-meta', attrs: {title: '最近聊天 IP：' + empty(chatIP) + ' · ' + region(chatArea)}},
          empty(chatIP) + ' · ' + region(chatArea)),
        h('el-popover', {key: this.contact.id, props: {placement: 'bottom-end', trigger: 'click', width: 320, popperClass: 'imgo-profile-popover'}}, [
          h('div', {class: 'imgo-profile-popover-title'}, '用户资料'),
          section('注册信息', p.create_time, p.register_ip, p.reg_location),
          section('上次登录', p.last_login_time, p.last_login_ip, p.location),
          section('最近聊天', p.last_chat_time, p.last_chat_ip, p.chat_location),
          h('button', {slot: 'reference', class: 'imgo-profile-trigger', attrs: {type: 'button', 'aria-label': '查看注册和登录详情'}}, [
            h('i', {class: 'el-icon-info'}), ' 详细资料'
          ])
        ])
      ])
    ]);
  }
};
