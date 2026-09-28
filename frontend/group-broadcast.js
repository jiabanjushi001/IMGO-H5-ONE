// Adds the management group broadcast dialog to the legacy Vue 2 component.
const imgoGroupBroadcastData = component.data;
component.data = function () {
  return Object.assign(imgoGroupBroadcastData.call(this), {
    broadcastVisible: false,
    broadcastContent: '',
    broadcastSending: false
  });
};
Object.assign(component.methods, {
  async setNumberJoin(group, enabled) {
    if (!group || !group.can_set_number_join) return;
    const value = enabled ? 1 : 0;
    const result = await this.$api.groupApi.setNumberJoin({group_id: group.group_id, enabled: value});
    if (Number(result.code) !== 0) return;
    this.$set(group, 'number_join', value);
    this.$message.success(value ? '已允许通过群号加入' : '已关闭群号加入');
  },
  openBroadcast() {
    if (!this.active) {
      this.$message.warning('请选择群聊');
      return;
    }
    this.broadcastContent = '';
    this.broadcastVisible = true;
  },
  closeBroadcast() {
    if (!this.broadcastSending) this.broadcastVisible = false;
  },
  async submitBroadcast() {
    const content = String(this.broadcastContent || '').trim();
    if (!content) {
      this.$message.warning('请输入群发内容');
      return;
    }
    if (content.length > 2048 || this.broadcastSending) return;
    this.broadcastSending = true;
    try {
      const result = await this.$api.groupApi.broadcast({group_id: this.active, content});
      if (Number(result.code) !== 0) return;
      const data = result.data || {};
      this.$message.success(data.message || ('群发完成：成功 ' + Number(data.sent || 0) + ' 人'));
      this.broadcastVisible = false;
      this.broadcastContent = '';
    } finally {
      this.broadcastSending = false;
    }
  }
});
