// The invitation panel stays mounted while chatting; refresh when its data changes.
component.watch = Object.assign({}, component.watch, {
  '$store.state.socketAction': function (event) {
    if (event && (event.type === 'friendApply' || event.type === 'friendApplyChanged')) {
      if (event.type === 'friendApply' && !this.params.is_mine) this.params.page = 1;
      this.getList();
    }
  }
});
// Avoid an older page request overwriting a newly received invitation.
component.methods.getList = async function () {
  const request = (this._imgoApplyRequest || 0) + 1;
  this._imgoApplyRequest = request;
  try {
    const result = await this.$api.friendApi.getApplyList(Object.assign({}, this.params));
    if (this._isDestroyed || request !== this._imgoApplyRequest || Number(result.code) !== 0) return;
    this.list = result.data;
    this.total = result.count;
    this.singlePage = this.total <= this.params.limit;
  } catch (_) {
    if (!this._isDestroyed && request === this._imgoApplyRequest) this.$message.error('好友申请加载失败，请重试');
  }
};
component.methods.acceptApply = async function (id, accepted) {
  try {
    const result = await this.$api.friendApi.acceptFriend({friend_id: id, status: accepted ? 1 : 0});
    if (Number(result.code) !== 0) return;
    this.$message.success('操作成功');
    this.$store.commit('catchSocketAction', {type: 'friendApplyChanged'});
  } catch (_) { this.$message.error('操作失败，请重试'); }
};
