async addUser(){
  const user=this.$store.state.userInfo||{};
  const locked=Number(user.user_id)!==1&&Number(user.agent_mode)===1;
  let code='';
  if(locked){
    try{
      const response=await this.$api.userApi.getUserList({page:1,limit:20,keywords:user.account||'',referral_scope:''});
      const self=Number(response.code)===0&&(response.data||[]).find(row=>Number(row.user_id)===Number(user.user_id));
      if(!self||!/^\d{6}$/.test(String(self.invite_code||'')))throw Error('获取本人邀请码失败，请刷新后重试');
      code=String(self.invite_code);
    }catch(error){this.$message.error(error.message||'获取本人邀请码失败');return;}
  }
  this.detail=Object.assign({},this.originDetail,{account:'',realname:'',parent_invite_code:code});
  this.formTitle='添加成员';this.formType='add';this.dialogVisible=true;
}
