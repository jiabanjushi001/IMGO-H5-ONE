# 后台定时清理面板

`maintenance-panel.js` 是与现有后台匹配的 Vue 2 Options API 组件，`maintenance.css` 是限定作用域的样式。

运行 `node scripts/build-maintenance.cjs` 后更新发行包的管理页模块和任务 API，保留原公告管理及其他页面。因为现有部署使用 webpack 构建产物，此脚本是针对当前发行包的适配器；更换原发行包时须重新核对 chunk/module/API 标识，不能直接套用。生成的管理页 chunk 和首页引用包含版本信息。

面板使用原 getTaskList/startTask/stopTask/getTaskLog 路径，增加 setTaskConfig；这些接口均限初始管理员。保存设置不会自动开启关闭中的任务；“保存并开启”才会启用。

后台每 10 秒检查数据库中的 next_run_at，到期后串行执行一次。消息清理沿用软删除（status=0），不删除系统公告和实体附件；过期登录会话被删除。仅保存最近 20 次执行日志。

## 概况页

`overview-panel.js` 负责数据加载、统计卡片和筛选；`overview-chart.js` 负责 SVG 图表；`maintenance-panel.js` 保留任务设置。构建脚本组合以上组件及原公告组件。无需新增前端依赖。

新增管理员接口 `/manage/index/overview`，所有既有路径保持不变。统计使用北京时间。用户统计排除已删除用户（包含禁用用户）；群聊、文件统计有效记录；消息排除隐藏消息和系统公告。趋势反映当前保留记录，并非不可变审计数据。

部署前运行 `-migrate` 添加 `imgo_online_sample` 表。主服务每分钟采样已认证 WebSocket 连接，按用户去重；设备数表示连接数。采样独立于清理开关，持久化到数据库；不补造上线前或停机时段的数据。当前仅支持单 Go 实例统计。采样表不自动删除历史记录。

## 后台样式

`admin-theme.css` 统一后台导航、卡片、表格、表单和弹窗，限定在 `imgo-admin` 管理布局。修改后执行 `node scripts/build-maintenance.cjs` 生成带版本号的资源。成员表日期仅在前端格式化为北京时间；Logo 加载失败时使用本地 `imgo-mark.svg`。

## 绑卡管理

新增的 `bank_name`（收款银行）和 `branch_name`（支行名称）会随 H5 绑卡提交、查询和后台列表/编辑返回。新提交或拒绝后重提必须填写这两项；旧记录迁移后两项为空，管理员仍可处理并补录。升级时先运行 `./bin/imgo -migrate`，该迁移可重复执行且不会改写旧卡号。

H5 的“我的 → 绑定银行卡”调用需登录的 `/enterprise/bank/get` 与 `/enterprise/bank/save`。保存参数为 `name`（收款姓名）、`card_number`（银行卡号）、`bank_name`（收款银行）、`branch_name`（支行名称）；首次提交必须填写卡号。每个用户保留一条记录：未处理（`status=0`）或同意（`status=1`）时用户不能再次提交，只有管理员拒绝（`status=2`）后才能修改，重提时可留空卡号沿用旧号。用户接口按登录用户的数据库状态校验，非拒绝状态一律返回 409；重新提交转为未处理并清空旧备注，条件更新时再次要求 `status=2`，防止并发绕过。此限制只针对用户接口，不限制管理员。后台左侧“绑卡”菜单调用 `/manage/bank/index`、`/manage/bank/detail`、`/manage/bank/edit`；编辑参数为 `user_id`、`version`、`receipt_name`、`bank_name`、`branch_name`、可选 `receipt_account`、`status`（0 未处理、1 同意、2 拒绝）、`remark`。编辑弹窗预填完整卡号并允许管理员修改，状态仍为单选；仅变更卡号时发送 `receipt_account`，服务端验证并加密新卡号。版本冲突返回 409，需要刷新后再编辑。

先运行 `./bin/imgo -migrate` 创建 `imgo_bank_card` 表，再启动新版服务。表中的卡号使用 AES-GCM 加密，仅保存末四位用于列表和 H5 脱敏展示；已鉴权后台的 `/manage/bank/detail` 单条详情会解密返回完整卡号以便核对，响应禁用缓存，不返回密文。绑卡密钥由现有 `JWT_KEY` 按独立用途派生，已有绑卡记录期间不能直接更换 `JWT_KEY`，否则旧记录无法解密；更换前必须执行数据迁移或保留旧密钥。后台构建时 `bank-panel.js` 与 `bank-panel.css` 会由 `scripts/build-maintenance.cjs` 注入现有构建包。

## 成员签到信息

成员列表现在可输入“上级用户名”（账号，如 `wuhuA1`），并选择“直属下级”或“所有下级”后查询；右侧原有关键字搜索仍可进一步筛选结果。未填上级用户名时保持普通成员列表。服务端 `/manage/user/index` 使用 `referrer_account` 和 `referral_scope`（`direct`/`all`）过滤 `imgo_referral_path`，分页总数与列表使用相同条件；升级后重新编译 Go 服务并执行 `node scripts/build-maintenance.cjs`。

签到按北京时间按账号每日记录一次，`imgo_check_in` 的 `(user_id, sign_date)` 主键防止重复。后台“成员”表格和编辑弹窗只读展示累计签到天数、今日状态、最近签到日期；`/manage/user/index` 在分页结果上批量统计，`/manage/user/detail` 返回相同字段 `checkin_days`、`checkin_today`、`checkin_last_date`。未签到成员分别返回 `0`、`false`、空字符串。成员表“签到信息”列的“查看详情”弹窗调用管理员专用 `/manage/user/checkInHistory`，按日期倒序分页展示签到日期与北京时间签到时间，响应的 `count` 为该成员总签到天数；无记录时显示空状态。修改 Go 服务后需重新构建，并在新环境先运行 `./bin/imgo -migrate` 创建签到表；后台静态资源通过 `node scripts/build-maintenance.cjs` 生成。

## 钱包与提现

钱包初始余额为 0，只有初始管理员可在后台“钱包”页按用户 ID 手动入账。每次入账要求填写金额、原因，并记录操作人和流水；请求编号用于防止网络重试重复入账。用户“我的”页面显示可提现余额和处理中金额，可进入提现及提现记录页。提现要求银行卡已审核通过；申请时把金额从可用余额移入待处理余额，并保存当时的加密银行卡信息快照。管理员拒绝会原路退回可用余额；确认已线下打款则扣除待处理余额并留下处理记录。**系统不会自动向银行发起转账；管理员必须先完成线下打款，再点击“确认已线下打款”。**

金额以整数分存储，不使用浮点数计算余额。升级时先运行 `./bin/imgo -migrate` 新建 `imgo_wallet`、`imgo_withdrawal`、`imgo_wallet_entry` 三张表，再启动新版 Go 服务；后台静态资源运行 `node scripts/build-maintenance.cjs` 生成。接口为用户 `/enterprise/wallet/status`、`withdraw`、`history`，初始管理员 `/manage/wallet/account`、`credit`、`index`、`detail`、`review`；提现及入账都接受幂等请求编号。完整收款卡号仅在已鉴权的后台单条提现详情中解密展示，列表和 H5 只显示末四位，响应禁用缓存。
