namespace Imgo.Desktop;

internal sealed record AdminColumn(string Label, string Path, string Format = "");
internal sealed record AdminPage(string Title, string Permission, string Api, string Web,
    bool Paged, bool Searchable, AdminColumn[] Columns, bool SuperOnly = false)
{
    public string Icon => Title switch
    {
        "系统概况" => "\uE9D2",
        "系统公告" => "\uE7E7",
        "系统设置" => "\uE713",
        "签到设置" => "\uE787",
        "成员管理" => "\uE77B",
        "消息管理" => "\uE8BD",
        "群聊管理" => "\uE902",
        "文件管理" => "\uE8B7",
        "实名认证" => "\uE8FA",
        "绑卡管理" => "\uE8C7",
        "充值订单" => "\uE896",
        "提现订单" => "\uE897",
        "日志" => "\uE9D5",
        "角色" => "\uE72E",
        _ => "\uE8A5"
    };
}

internal static class AdminPages
{
    private static AdminColumn C(string label, string key, string format = "") => new(label, key, format);

    public static readonly AdminPage[] All =
    [
        new("系统概况", "manage.overview", "/manage/index/overview", "/manage/index", false, false, []),
        new("系统公告", "manage.overview", "/manage/index/noticelist", "/manage/index", true, false,
            [C("编号", "msg_id"), C("标题", "title"), C("内容", "content"), C("发布时间", "create_time", "date")], true),
        new("系统设置", "manage.settings", "/manage/config/getconfig", "/manage/setting", false, false, []),
        new("签到设置", "", "/manage/config/getcheckinsettings", "/manage/setting", false, false, []),
        new("成员管理", "manage.users", "/manage/user/index", "/manage/user", true, true,
            [C("ID", "user_id"), C("用户名", "account"), C("昵称", "realname"), C("角色", "admin_role_name"),
             C("备注", "remark"), C("注册时间", "create_time", "date"), C("最后登录", "last_login_time", "date")]),
        new("消息管理", "manage.messages", "/manage/message/index", "/manage/message", true, true,
            [C("编号", "msg_id"), C("发送人", "fromUser.displayName"), C("发送人 ID", "from_user"),
             C("类型", "type"), C("内容", "content", "message"), C("时间", "sendTime", "date")]),
        new("群聊管理", "manage.groups", "/manage/group/index", "/manage/group", true, true,
            [C("群号", "group_id"), C("群名", "name"), C("群主", "owner_id_info.displayName"),
             C("人数", "user_count"), C("按群号加入", "number_join", "switch"), C("创建时间", "create_time", "date")]),
        new("文件管理", "manage.files", "/manage/files/index", "/manage/files", true, true,
            [C("ID", "file_id"), C("文件名", "name"), C("类别", "msg_type"), C("上传人 ID", "user_id"),
             C("大小", "size", "bytes"), C("时间", "create_time", "date")]),
        new("实名认证", "manage.users", "/manage/identity/index", "/manage/identity", true, true,
            [C("用户 ID", "user_id"), C("用户名", "login_account"), C("实名姓名", "real_name"),
             C("审核状态", "status", "review"), C("备注", "remark"), C("提交时间", "submitted_at", "date")]),
        new("绑卡管理", "manage.bank", "/manage/bank/index", "/manage/bank", true, true,
            [C("用户 ID", "user_id"), C("用户名", "login_account"), C("类型", "payment_type"),
             C("收款姓名", "receipt_name"), C("收款账号", "receipt_account_masked"), C("状态", "status", "review")]),
        new("充值订单", "manage.finance", "/manage/wallet/recharges", "/manage/finance/recharges", true, true,
            [C("订单", "order_id"), C("用户名", "account"), C("本金", "amount_cents", "money"),
             C("赠送", "bonus_cents", "money"), C("实入账", "total_cents", "money"), C("备注", "note"), C("时间", "created_at", "date")]),
        new("提现订单", "manage.finance", "/manage/wallet/index", "/manage/finance/withdrawals", true, true,
            [C("订单", "withdrawal_id"), C("用户名", "account"), C("金额", "amount_cents", "money"),
             C("状态", "status", "withdrawal"), C("备注", "remark"), C("时间", "created_at", "date")]),
        new("日志", "", "/manage/audit/index", "/manage/audit", true, true,
            [C("时间", "created_at", "date"), C("操作账号", "actor_account"), C("分类", "category"),
             C("操作", "action"), C("敏感等级", "risk_level"), C("对象", "target_name"), C("IP", "ip"),
             C("结果", "status", "result"), C("说明", "detail")], true),
        new("角色", "", "/manage/role/index", "/manage/role", false, false,
            [C("ID", "role_id"), C("角色名称", "name"), C("成员数", "user_count"),
             C("代理模式", "agent_mode", "switch"), C("状态", "status", "switch"), C("备注", "remark")], true)
    ];

    public static bool IsAllowed(AdminPage page, AccountSession account) => page.SuperOnly ? account.IsSuperAdmin :
        page.Permission.Length == 0 ? account.IsAgent || account.HasPermission("manage.settings") :
        account.HasPermission(page.Permission);
}
