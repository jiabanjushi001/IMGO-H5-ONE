// Backend chat-only dialog for joining a group whose owner enabled group-number access.
component.components.ImgoJoinGroup = {
  name: 'ImgoJoinGroup',
  data() {
    return {visible: false, joining: false, value: ''};
  },
  methods: {
    open() {
      this.value = '';
      this.visible = true;
      this.$nextTick(() => {
        const input = this.$refs.input;
        if (input && typeof input.focus === 'function') input.focus();
      });
    },
    parse(value) {
      const raw = String(value || '').trim();
      if (!/^\d+$/.test(raw) || Number(raw) <= 0) return null;
      return {group_id: Number(raw)};
    },
    async submit() {
      if (this.joining) return;
      const params = this.parse(this.value);
      if (!params) {
        this.$message.warning('请输入正确的群号');
        return;
      }
      this.joining = true;
      try {
        const joined = await this.$api.imApi.joinGroupAPI(params);
        if (Number(joined.code) !== 0) return;
        const id = 'group-' + params.group_id;
        const result = await this.$api.imApi.contactInfo({id});
        if (Number(result.code) !== 0) return;
        this.$emit('joined', result.data);
        this.visible = false;
        this.$message.success('群聊已添加');
      } finally {
        this.joining = false;
      }
    }
  },
  render(h) {
    return h('el-dialog', {
      props: {title: '添加群聊', visible: this.visible, width: '480px', appendToBody: true, closeOnClickModal: false},
      on: {'update:visible': value => { this.visible = value; }}
    }, [
      h('div', {class: 'imgo-join-group'}, [
        h('div', {class: 'imgo-join-group-label'}, ['群号']),
        h('el-input', {
          ref: 'input',
          props: {value: this.value, clearable: true, placeholder: '请输入数字群号，例如：12'},
          on: {input: value => { this.value = value; }},
          nativeOn: {keyup: event => { if (event.key === 'Enter') this.submit(); }}
        }),
        h('p', {class: 'imgo-join-group-tip'}, ['群主开启“群号加入”后，私有群也可以直接添加。'])
      ]),
      h('span', {slot: 'footer'}, [
        h('el-button', {props: {disabled: this.joining}, on: {click: () => { this.visible = false; }}}, ['取消']),
        h('el-button', {props: {type: 'primary', loading: this.joining}, on: {click: this.submit}}, ['添加'])
      ])
    ]);
  }
};

const imgoOriginalHandleCommand = component.methods.handleCommand;
component.methods.handleCommand = function (command) {
  if (command === 'joinGroup') {
    const dialog = this.$refs && this.$refs.ImgoJoinGroup;
    if (dialog && typeof dialog.open === 'function') dialog.open();
    return;
  }
  return imgoOriginalHandleCommand.call(this, command);
};
component.methods.imgoJoinedGroup = function (contact) {
  const ui = this.$refs && this.$refs.IMUI;
  if (!ui || !contact || !contact.id) return;
  const exists = ui.getContacts().some(item => String(item.id) === String(contact.id));
  if (exists) ui.updateContact(contact);
  else ui.appendContact(contact);
  ui.changeContact(contact.id);
};
