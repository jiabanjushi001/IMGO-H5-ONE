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
