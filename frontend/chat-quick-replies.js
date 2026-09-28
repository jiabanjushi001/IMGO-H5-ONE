// Mentor-owned quick replies shown in the ordinary chat composer.
component.components.ImgoChatQuickReplies = {
  name: 'ImgoChatQuickReplies',
  data() {
    return {
      loading: false,
      saving: false,
      popoverVisible: false,
      managerVisible: false,
      items: [],
      draft: [],
      canManage: false
    };
  },
  created() {
    this.load();
  },
  methods: {
    async load() {
      if (this.loading) return;
      this.loading = true;
      try {
        const result = await this.$api.imApi.getQuickRepliesAPI({});
        if (Number(result.code) !== 0) return;
        const data = result.data || {};
        this.items = (data.items || []).map(String);
        this.canManage = Boolean(data.can_manage);
      } finally {
        this.loading = false;
      }
    },
    select(item) {
      const value = String(item || '').trim();
      if (!value) return;
      this.$emit('select', value);
      this.popoverVisible = false;
    },
    openManager() {
      if (!this.canManage) return;
      this.draft = this.items.slice();
      this.managerVisible = true;
    },
    add() {
      if (this.draft.length >= 50) {
        this.$message.warning('快捷用语最多保存 50 条');
        return;
      }
      this.draft.push('');
    },
    remove(index) {
      this.draft.splice(index, 1);
    },
    async save() {
      if (this.saving) return;
      const items = this.draft.map(value => String(value || '').trim()).filter(Boolean);
      if (items.some(value => Array.from(value).length > 500)) {
        this.$message.warning('单条快捷用语不能超过 500 个字符');
        return;
      }
      this.saving = true;
      try {
        const result = await this.$api.imApi.saveQuickRepliesAPI({items});
        if (Number(result.code) !== 0) return;
        this.items = (result.data && result.data.items || items).map(String);
        this.managerVisible = false;
        this.$message.success('快捷用语已保存');
      } finally {
        this.saving = false;
      }
    }
  },
  render(h) {
    const replyList = this.items.length
      ? h('div', {class: 'imgo-chat-quick-list'}, this.items.map((item, index) => h('button', {
          key: item + index,
          class: 'imgo-chat-quick-item',
          attrs: {type: 'button', title: item},
          on: {click: () => this.select(item)}
        }, [item])))
      : h('div', {class: 'imgo-chat-quick-empty'}, [this.canManage ? '还没有快捷用语，请先添加' : '导师暂未设置快捷用语']);
    const popover = h('el-popover', {
      props: {value: this.popoverVisible, placement: 'top-start', width: 360, trigger: 'click', popperClass: 'imgo-chat-quick-popover'},
      on: {input: value => { this.popoverVisible = value; }, show: this.load}
    }, [
      h('div', {class: 'imgo-chat-quick-head'}, [
        h('strong', ['快捷用语']),
        this.canManage ? h('el-button', {props: {type: 'text', icon: 'el-icon-setting'}, on: {click: this.openManager}}, ['管理']) : h('span', ['导师共享'])
      ]),
      replyList,
      h('button', {slot: 'reference', class: 'imgo-chat-quick-trigger', attrs: {type: 'button'}}, [
        h('i', {class: 'el-icon-chat-line-square'}), h('span', ['快捷用语'])
      ])
    ]);
    const rows = this.draft.map((item, index) => h('div', {key: index, class: 'imgo-chat-quick-row'}, [
      h('el-input', {props: {value: item, maxlength: 500, placeholder: '请输入快捷用语'}, on: {input: value => this.$set(this.draft, index, value)}}),
      h('el-button', {props: {type: 'text'}, class: 'imgo-chat-quick-remove', on: {click: () => this.remove(index)}}, ['删除'])
    ]));
    const manager = h('el-dialog', {
      props: {title: '管理快捷用语', visible: this.managerVisible, width: '560px', appendToBody: true, closeOnClickModal: false},
      on: {'update:visible': value => { this.managerVisible = value; }}
    }, [
      h('div', {class: 'imgo-chat-quick-manager'}, [
        rows.length ? h('div', {class: 'imgo-chat-quick-rows'}, rows) : h('div', {class: 'imgo-chat-quick-manager-empty'}, ['暂未添加快捷用语']),
        h('el-button', {props: {type: 'primary', plain: true, icon: 'el-icon-plus', disabled: this.draft.length >= 50}, on: {click: this.add}}, ['新增快捷用语'])
      ]),
      h('span', {slot: 'footer'}, [
        h('el-button', {props: {disabled: this.saving}, on: {click: () => { this.managerVisible = false; }}}, ['取消']),
        h('el-button', {props: {type: 'primary', loading: this.saving}, on: {click: this.save}}, ['保存'])
      ])
    ]);
    return h('span', {class: 'imgo-chat-quick-replies'}, [popover, manager]);
  }
};

component.methods.imgoUseQuickReply = function (value) {
  const text = String(value || '').trim();
  const imui = this.$refs && this.$refs.IMUI;
  if (!text || !imui) return;

  // Lemon IMUI's setEditorValue expects its serialized rich-text markup.
  // Passing plain text leaves a raw text node in the editor, so getHtml()
  // reports an empty message and the send button stays disabled. Use the
  // chat area's text insertion path so it builds the expected grid markup.
  const composer = imui.$refs && imui.$refs.editor;
  if (composer && composer.chatArea && typeof composer.chatArea.insertText === 'function') {
    if (typeof composer.clear === 'function') composer.clear();
    composer.chatArea.insertText(text);
    if (typeof composer._checkSubmitDisabled === 'function') {
      composer._checkSubmitDisabled();
    } else {
      composer.submitDisabled = false;
    }
    return;
  }

  if (typeof imui.setEditorValue === 'function') {
    imui.setEditorValue(text);
  }
};
