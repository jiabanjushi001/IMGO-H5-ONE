// Select recipients and send private bulk messages from configured customer accounts.
const ImgoMemberBroadcastDialog = {
  name: 'ImgoMemberBroadcastDialog',
  data() {
    return {
      visible: false,
      loading: false,
      sending: false,
      keyword: '',
      users: [],
      selectedIds: [],
      content: ''
    };
  },
  computed: {
    filteredUsers() {
      const keyword = String(this.keyword || '').trim().toLowerCase();
      if (!keyword) return this.users;
      return this.users.filter(user =>
        String(user.account || '').toLowerCase().includes(keyword) ||
        String(user.realname || '').toLowerCase().includes(keyword)
      );
    },
    allVisibleSelected() {
      return this.filteredUsers.length > 0 && this.filteredUsers.every(user => this.selectedIds.includes(Number(user.user_id)));
    }
  },
  methods: {
    async open() {
      this.visible = true;
      this.loading = true;
      this.keyword = '';
      this.selectedIds = [];
      this.content = '';
      try {
        const result = await this.$api.userApi.broadcastOptions({});
        if (Number(result.code) !== 0) {
          this.visible = false;
          return;
        }
        this.users = (result.data || []).map(user => Object.assign({}, user, {user_id: Number(user.user_id)}));
      } finally {
        this.loading = false;
      }
    },
    close() {
      if (!this.sending) this.visible = false;
    },
    toggleAll(checked) {
      const visibleIds = this.filteredUsers.map(user => Number(user.user_id));
      if (checked) this.selectedIds = Array.from(new Set(this.selectedIds.concat(visibleIds)));
      else this.selectedIds = this.selectedIds.filter(id => !visibleIds.includes(Number(id)));
    },
    avatar(user) {
      return user.avatar || '/assets/img/imgo-mark.svg';
    },
    async submit() {
      const content = String(this.content || '').trim();
      if (!this.selectedIds.length) {
        this.$message.warning('请选择要群发的用户');
        return;
      }
      if (!content) {
        this.$message.warning('请输入群发内容');
        return;
      }
      if (content.length > 2048 || this.sending) return;
      this.sending = true;
      try {
        const result = await this.$api.userApi.broadcast({user_ids: this.selectedIds, content});
        if (Number(result.code) !== 0) return;
        const data = result.data || {};
        this.$message.success(data.message || ('群发完成：成功 ' + Number(data.sent || 0) + ' 人'));
        this.visible = false;
      } finally {
        this.sending = false;
      }
    }
  },
  render(h) {
    const cards = this.filteredUsers.map(user => h('el-checkbox', {
      key: user.user_id,
      class: 'imgo-broadcast-user',
      props: {label: user.user_id}
    }, [
      h('el-avatar', {props: {size: 42, src: this.avatar(user)}}),
      h('span', {class: 'imgo-broadcast-user-name'}, [user.realname || user.account || ('用户' + user.user_id)]),
      h('small', {class: 'imgo-broadcast-user-account'}, [user.account || ''])
    ]));
    return h('el-dialog', {
      props: {title: '成员群发', visible: this.visible, width: '760px', appendToBody: true, closeOnClickModal: false},
      on: {'update:visible': value => { this.visible = value; }}
    }, [
      h('div', {class: 'imgo-broadcast-toolbar'}, [
        h('el-checkbox', {props: {value: this.allVisibleSelected}, on: {change: this.toggleAll}}, ['全选']),
        h('el-input', {class: 'imgo-broadcast-search', props: {value: this.keyword, clearable: true, prefixIcon: 'el-icon-search', placeholder: '搜索用户名或昵称'}, on: {input: value => { this.keyword = value; }}}),
        h('span', {class: 'imgo-broadcast-count'}, ['已选 ' + this.selectedIds.length + ' 人'])
      ]),
      h('div', {directives: [{name: 'loading', value: this.loading}], class: 'imgo-broadcast-users'}, [
        this.filteredUsers.length ? h('el-checkbox-group', {props: {value: this.selectedIds}, on: {input: value => { this.selectedIds = value.map(Number); }}}, [h('div', {class: 'imgo-broadcast-grid'}, cards)]) : h('el-empty', {props: {description: '没有可选择的成员'}})
      ]),
      h('el-input', {class: 'imgo-broadcast-editor', props: {type: 'textarea', rows: 4, maxlength: 2048, showWordLimit: true, value: this.content, placeholder: '请输入群发内容'}, on: {input: value => { this.content = value; }}}),
      h('span', {slot: 'footer'}, [
        h('el-button', {props: {disabled: this.sending}, on: {click: this.close}}, ['取消']),
        h('el-button', {props: {type: 'primary', loading: this.sending, disabled: !this.selectedIds.length}, on: {click: this.submit}}, ['发送给 ' + this.selectedIds.length + ' 人'])
      ])
    ]);
  }
};
