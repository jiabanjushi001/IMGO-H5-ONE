	t("el-table-column",{attrs:{fixed:"right",label:"操作",width:"210"},scopedSlots:e._u([{key:"default",fn:function(s){return[ t("div",{staticClass:"imgo-member-actions-inline"},[
  t("el-button",{attrs:{type:"text",size:"small",disabled:true,title:"功能暂未开放"}},[e._v("会话")]),
  (Number((e.$store.state.userInfo||{}).user_id)===1||(Number((e.$store.state.userInfo||{}).agent_mode)===1&&Number((e.$store.state.userInfo||{}).user_id)!==Number(s.row.user_id)))?t("el-button",{attrs:{type:"text",size:"small"},on:{click:function(){return e.$refs.memberFinance.open(s.row,"recharge")}}},[e._v("充值")]):e._e(),
	  (Number((e.$store.state.userInfo||{}).user_id)===1||(Number((e.$store.state.userInfo||{}).agent_mode)===1&&Number((e.$store.state.userInfo||{}).user_id)!==Number(s.row.user_id)))?t("el-button",{attrs:{type:"text",size:"small"},on:{click:function(){return e.$refs.memberFinance.open(s.row,"debit")}}},[e._v("扣款")]):e._e(),
  (Number((e.$store.state.userInfo||{}).user_id)===1||(Number((e.$store.state.userInfo||{}).agent_mode)===1&&Number((e.$store.state.userInfo||{}).user_id)!==Number(s.row.user_id)))?t("el-button",{attrs:{type:"text",size:"small"},on:{click:function(){return e.$refs.memberFinance.open(s.row,"withdraw")}}},[e._v("提现")]):e._e(),
  t("el-dropdown",{attrs:{trigger:"click",placement:"bottom-end"},on:{command:function(command){
   if(command==="googleAuth"&&Number((e.$store.state.userInfo||{}).user_id)===1)return e.$refs.memberGoogleAuth.open(s.row);
   if(Number(s.row.is_self)===1&&Number((e.$store.state.userInfo||{}).user_id)===Number(s.row.user_id)){
    if(command==="agentSetting"&&Number(s.row.admin_role_agent_mode)===1)return e.$refs.memberAgentSetting.open(s.row);
    return;
   }
   if(command==="dialogue")return e.openDialogue(s.row);
   if(command==="view")return e.handleClick(s.row);
   if(command==="edit")return e.editUser(s.row);
   if(command==="password")return e.editPass(s.row);
   if(command==="inviteCode")return e.$refs.memberInviteCode.open(s.row);
   if(command==="withdrawalEnable"||command==="withdrawalDisable"||command==="withdrawalInherit"){
    const mode=command==="withdrawalEnable"?"enabled":command==="withdrawalDisable"?"disabled":"inherit";
    const actionText=mode==="enabled"?"开启此人提现":mode==="disabled"?"关闭此人提现":"恢复继承团队提现";
    const detail=mode==="enabled"?"该成员将可以提现，即使其导师团队已关闭提现。":mode==="disabled"?"该成员将不能提交新的提现申请，即使其导师团队已开启提现。":"该成员不再使用个人开关，之后跟随导师团队或系统总开关。";
    return e.$confirm(detail,actionText,{confirmButtonText:"确定",cancelButtonText:"取消",type:mode==="disabled"?"warning":"info"}).then(function(){
     return e.$api.userApi.setWithdrawal({user_id:s.row.user_id,mode:mode});
    }).then(function(result){
     if(!result||Number(result.code)!==0)throw Error(result&&result.msg||"设置失败");
     e.$set(s.row,"withdrawal_enabled",result.data?result.data.withdrawal_enabled:null);
     e.$message.success(actionText+"成功");
    }).catch(function(error){if(error&&error!=="cancel"&&error!=="close")e.$message.error(error.msg||error.message||"设置失败");});
   }
   if(command==="groupFriendAdd"){
    const enabled=Number(s.row.group_friend_add)===1,next=enabled?0:1;
    return e.$confirm(enabled?"关闭后，该账号将不能绕过群成员隐私添加群友，确定关闭吗？":"开启后，该账号可在群内查看成员资料并添加为好友，但不会获得拉人进群或群管理权限，确定开启吗？","群添加权限",{confirmButtonText:"确定",cancelButtonText:"取消",type:"warning"}).then(function(){
     return e.$api.userApi.setGroupFriendAdd({user_id:s.row.user_id,group_friend_add:next});
    }).then(function(result){
     if(result&&Number(result.code)===0){e.$set(s.row,"group_friend_add",next);e.$message.success(next?"已开启群添加权限":"已关闭群添加权限");}
    }).catch(function(error){if(error&&error!=="cancel"&&error!=="close")e.$message.error(error.msg||error.message||"设置失败");});
   }
   if(command==="agentSetting"&&Number(s.row.admin_role_agent_mode)===1&&(Number((e.$store.state.userInfo||{}).user_id)===1||(Number((e.$store.state.userInfo||{}).user_id)===Number(s.row.user_id)&&Number(s.row.is_self)===1)))return e.$refs.memberAgentSetting.open(s.row);
  }}},[
   t("el-button",{attrs:{type:"text",size:"small"}},[e._v("更多"),t("i",{staticClass:"el-icon-arrow-down"})]),
   t("el-dropdown-menu",{slot:"dropdown"},[
    Number(s.row.is_self)===1&&Number((e.$store.state.userInfo||{}).user_id)===Number(s.row.user_id)?e._e():t("el-dropdown-item",{attrs:{command:"dialogue"}},[e._v("会话列表")]),
    Number(s.row.is_self)===1&&Number((e.$store.state.userInfo||{}).user_id)===Number(s.row.user_id)?e._e():t("el-dropdown-item",{attrs:{command:"view"}},[e._v("查看")]),
    Number(s.row.is_self)!==1&&s.row.user_id>1?t("el-dropdown-item",{attrs:{command:"edit"}},[e._v("编辑")]):e._e(),
    Number(s.row.is_self)===1&&Number((e.$store.state.userInfo||{}).user_id)===Number(s.row.user_id)?e._e():t("el-dropdown-item",{attrs:{command:"password"}},[e._v("改密")]),
    Number(s.row.is_self)===1&&Number((e.$store.state.userInfo||{}).user_id)===Number(s.row.user_id)?e._e():t("el-dropdown-item",{attrs:{command:"inviteCode"}},[e._v("修改邀请码")]),
    (Number((e.$store.state.userInfo||{}).user_id)===1||Number((e.$store.state.userInfo||{}).agent_mode)===1)&&Number(s.row.is_self)!==1&&Number(s.row.user_id)>1&&Number(s.row.withdrawal_enabled)!==1?t("el-dropdown-item",{attrs:{command:"withdrawalEnable"}},[e._v("个人开启提现")]):e._e(),
    (Number((e.$store.state.userInfo||{}).user_id)===1||Number((e.$store.state.userInfo||{}).agent_mode)===1)&&Number(s.row.is_self)!==1&&Number(s.row.user_id)>1&&(null==s.row.withdrawal_enabled||Number(s.row.withdrawal_enabled)!==0)?t("el-dropdown-item",{attrs:{command:"withdrawalDisable"}},[e._v("个人关闭提现")]):e._e(),
    (Number((e.$store.state.userInfo||{}).user_id)===1||Number((e.$store.state.userInfo||{}).agent_mode)===1)&&Number(s.row.is_self)!==1&&Number(s.row.user_id)>1&&null!=s.row.withdrawal_enabled?t("el-dropdown-item",{attrs:{command:"withdrawalInherit"}},[e._v("恢复团队提现设置")]):e._e(),
    Number(s.row.is_self)!==1&&Number(s.row.user_id)>1?t("el-dropdown-item",{attrs:{command:"groupFriendAdd"}},[e._v(Number(s.row.group_friend_add)===1?"关闭群添加":"开启群添加")]):e._e(),
    Number((e.$store.state.userInfo||{}).user_id)===1?t("el-dropdown-item",{attrs:{command:"googleAuth"}},[e._v("谷歌设置")]):e._e(),
    Number(s.row.admin_role_agent_mode)===1&&(Number((e.$store.state.userInfo||{}).user_id)===1||(Number((e.$store.state.userInfo||{}).user_id)===Number(s.row.user_id)&&Number(s.row.is_self)===1))?t("el-dropdown-item",{attrs:{command:"agentSetting"}},[e._v("导师设置")]):e._e()
   ],1)
  ],1)
 ],1)
]}}])})
