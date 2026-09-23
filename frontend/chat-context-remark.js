// Add an editable friend remark to the private-chat context menu.
const imgoChatData = component.data;
component.data = function () {
  const state = imgoChatData.call(this);
  const vm = this;
  const menus = state.contactContextmenu || [];
  if (!menus.some(item => item && item.imgoRemark)) {
    menus.splice(1, 0, {
      imgoRemark: true,
      icon: 'el-icon-edit-outline',
      text: '设置备注',
      visible: event => Number(event.contact.is_group) === 0,
      click(event, context, close) {
        const {IMUI, contact} = context;
        close();
        vm.$prompt('请输入好友备注，留空可清除备注', '设置备注', {
          confirmButtonText: '保存',
          cancelButtonText: '取消',
          inputValue: contact.displayName || '',
          inputAttributes: {maxlength: 100},
          closeOnClickModal: false
        }).then(async ({value}) => {
          const nickname = String(value == null ? '' : value).trim();
          if ([...nickname].length > 100) {
            vm.$message.warning('备注不能超过100个字');
            return;
          }
          const result = await vm.$api.friendApi.setNickname({user_id: contact.id, nickname});
          if (Number(result.code) !== 0) return;
          const displayName = nickname || contact.realname || contact.account || String(contact.id);
          vm.$set(contact, 'displayName', displayName);
          IMUI.updateContact({id: contact.id, displayName});
          const current = IMUI.getCurrentContact();
          if (current && String(current.id) === String(contact.id)) {
            vm.$set(current, 'displayName', displayName);
          }
          vm.$message.success(nickname ? '备注已保存' : '备注已清除');
        }).catch(() => {});
      }
    });
  }
  return state;
};
