t("el-table-column",{attrs:{fixed:"right",label:"操作",width:"164"},scopedSlots:e._u([{key:"default",fn:function(s){return[ t("div",{staticClass:"imgo-member-actions-inline"},[
  t("el-button",{attrs:{type:"text",size:"small",disabled:true,title:"功能暂未开放"}},[e._v("会话")]),
  Number((e.$store.state.userInfo||{}).user_id)===1?t("el-button",{attrs:{type:"text",size:"small"},on:{click:function(){return e.$refs.memberFinance.open(s.row,"recharge")}}},[e._v("充值")]):e._e(),
  Number((e.$store.state.userInfo||{}).user_id)===1?t("el-button",{attrs:{type:"text",size:"small"},on:{click:function(){return e.$refs.memberFinance.open(s.row,"withdraw")}}},[e._v("提现")]):e._e(),
  t("el-dropdown",{attrs:{trigger:"click",placement:"bottom-end"},on:{command:function(command){
   if(Number(s.row.is_self)===1&&Number((e.$store.state.userInfo||{}).user_id)===Number(s.row.user_id)){
    if(command==="agentSetting"&&Number(s.row.admin_role_agent_mode)===1)return e.$refs.memberAgentSetting.open(s.row);
    return;
   }
   if(command==="dialogue")return e.openDialogue(s.row);
   if(command==="view")return e.handleClick(s.row);
   if(command==="edit")return e.editUser(s.row);
   if(command==="password")return e.editPass(s.row);
   if(command==="inviteCode")return e.$refs.memberInviteCode.open(s.row);
   if(command==="agentSetting"&&Number(s.row.admin_role_agent_mode)===1&&(Number((e.$store.state.userInfo||{}).user_id)===1||(Number((e.$store.state.userInfo||{}).user_id)===Number(s.row.user_id)&&Number(s.row.is_self)===1)))return e.$refs.memberAgentSetting.open(s.row);
  }}},[
   t("el-button",{attrs:{type:"text",size:"small"}},[e._v("更多"),t("i",{staticClass:"el-icon-arrow-down"})]),
   t("el-dropdown-menu",{slot:"dropdown"},[
    Number(s.row.is_self)===1&&Number((e.$store.state.userInfo||{}).user_id)===Number(s.row.user_id)?e._e():t("el-dropdown-item",{attrs:{command:"dialogue"}},[e._v("会话列表")]),
    Number(s.row.is_self)===1&&Number((e.$store.state.userInfo||{}).user_id)===Number(s.row.user_id)?e._e():t("el-dropdown-item",{attrs:{command:"view"}},[e._v("查看")]),
    Number(s.row.is_self)!==1&&s.row.user_id>1?t("el-dropdown-item",{attrs:{command:"edit"}},[e._v("编辑")]):e._e(),
    Number(s.row.is_self)===1&&Number((e.$store.state.userInfo||{}).user_id)===Number(s.row.user_id)?e._e():t("el-dropdown-item",{attrs:{command:"password"}},[e._v("改密")]),
    Number(s.row.is_self)===1&&Number((e.$store.state.userInfo||{}).user_id)===Number(s.row.user_id)?e._e():t("el-dropdown-item",{attrs:{command:"inviteCode"}},[e._v("修改邀请码")]),
    Number(s.row.admin_role_agent_mode)===1&&(Number((e.$store.state.userInfo||{}).user_id)===1||(Number((e.$store.state.userInfo||{}).user_id)===Number(s.row.user_id)&&Number(s.row.is_self)===1))?t("el-dropdown-item",{attrs:{command:"agentSetting"}},[e._v("导师设置")]):e._e()
   ],1)
  ],1)
 ],1)
]}}])})
