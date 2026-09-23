const ImgoMemberRemark = {
 name: 'ImgoMemberRemark',
 props: {row: {type: Object, required: true}},
 data() { return {editing: false, draft: '', saving: false}; },
 methods: {
  open() { this.draft = this.row.remark || ''; this.editing = true; this.$nextTick(() => this.$refs.input.focus()); },
  async save() {
   if (this.saving) return;
   if (this.draft === (this.row.remark || '')) { this.editing = false; return; }
   this.saving = true;
   try {
    const result = await this.$api.userApi.setRemark({user_id: this.row.user_id, remark: this.draft});
    if (Number(result.code) !== 0) throw new Error(result.msg || '保存失败');
    this.$set(this.row, 'remark', this.draft); this.editing = false; this.$message.success('备注已保存');
   } catch (error) { this.$message.error(error.message || '保存失败，请重试'); }
   finally { this.saving = false; }
  }
 },
 render(h) {
  if (!this.editing) return h('button', {class: 'imgo-inline-remark', attrs: {type: 'button', title: '点击编辑备注'}, on: {click: this.open}}, [
   h('span', this.row.remark || '添加备注'), h('i', {class: 'el-icon-edit'})
  ]);
  return h('div', {class: 'imgo-inline-remark-form'}, [
   h('el-input', {ref: 'input', props: {type: 'textarea', value: this.draft, maxlength: 191, rows: 2, disabled: this.saving, placeholder: '填写备注'}, on: {input: value => {this.draft = value;}}}),
   h('el-button', {props: {type: 'text', size: 'mini', loading: this.saving}, on: {click: this.save}}, '保存'),
   h('el-button', {props: {type: 'text', size: 'mini', disabled: this.saving}, on: {click: () => {this.editing = false;}}}, '取消')
  ]);
 }
};
