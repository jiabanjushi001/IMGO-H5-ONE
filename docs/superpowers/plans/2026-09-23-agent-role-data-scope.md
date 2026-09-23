# 导师代理角色与数据范围 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 自动创建“导师专员”角色，为角色增加代理模式，并让导师后台、概况统计和注册自动分配严格限制在该导师的邀请下级范围内。

**Architecture:** 后端新增集中式 `adminScope`，从登录用户角色解析全局或导师范围，并为所有管理查询和事务提供统一的下级校验。导师自动客服和自动群聊使用可空覆盖配置；空值继承全局配置，覆盖值使用独立轮换状态。Vue 2 编译产物继续通过可重复构建适配器注入角色开关和导师设置弹窗。

**Tech Stack:** Go 1.23、Gin、MySQL 8、`sqlmock`、Vue 2 编译产物适配、Node.js 断言脚本、Linux amd64 交叉编译。

**Spec:** `docs/superpowers/specs/2026-09-23-agent-role-data-scope-design.md`

## Global Constraints

- 超级管理员 `user_id=1` 始终拥有全站权限。
- 普通用户没有后台管理权限。
- 已有自定义角色升级后保持 `agent_mode=0`；后台新建角色默认勾选代理模式。
- 导师范围不包含导师本人，只包含 `imgo_referral_path` 中以导师为祖先的账号。
- 列表、详情和修改接口必须执行同一服务端范围校验。
- 导师设置的空值表示继承全局，显式关闭必须保存为覆盖 JSON。
- 自动数据库升级必须幂等，不覆盖管理员修改过的导师角色权限。
- API 路径保持现有兼容性；新增接口使用 `/manage/agentSetting/*`。

---

### Task 1: 数据库字段、导师表与预置角色

**Files:**
- Modify: `internal/server/addon_schema.go`
- Modify: `internal/server/addon_schema_test.go`
- Modify: `internal/server/schema.sql`
- Modify: `internal/server/migrate.go`

**Interfaces:**
- Produces: `imgo_admin_role.agent_mode`, `imgo_admin_role.role_code`。
- Produces: `imgo_agent_setting`、`imgo_agent_auto_state`、`imgo_agent_online_sample`。
- Produces: `seedMentorRole(ctx context.Context, db DB) error`。

- [ ] **Step 1: 写导师数据库升级失败测试**

在 `addon_schema_test.go` 增加断言，要求升级语句包含：

```go
for _, name := range []string{
    "imgo_agent_setting", "imgo_agent_auto_state", "imgo_agent_online_sample",
} {
    if !strings.Contains(joinedDDL, name) { t.Fatalf("missing %s", name) }
}
```

并使用 `sqlmock` 验证 `seedMentorRole` 首次创建 `role_code='mentor'`，随后写入除 `manage.settings` 外的七项权限。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/server -run 'Test.*(Addon|MentorRole)' -count=1`

Expected: FAIL，缺少代理字段、导师表或 `seedMentorRole`。

- [ ] **Step 3: 实现幂等数据库升级**

在 `ensureAddonTables` 中创建：

```sql
CREATE TABLE IF NOT EXISTS `prefix_imgo_agent_setting` (
  agent_user_id BIGINT PRIMARY KEY,
  auto_add_user JSON NULL,
  auto_add_group JSON NULL,
  updated_by BIGINT NOT NULL,
  updated_at BIGINT NOT NULL
)
```

```sql
CREATE TABLE IF NOT EXISTS `prefix_imgo_agent_auto_state` (
  agent_user_id BIGINT PRIMARY KEY,
  last_customer_user_id BIGINT NOT NULL DEFAULT 0,
  group_id BIGINT NOT NULL DEFAULT 0,
  group_num INT NOT NULL DEFAULT 1,
  updated_at BIGINT NOT NULL
)
```

```sql
CREATE TABLE IF NOT EXISTS `prefix_imgo_agent_online_sample` (
  agent_user_id BIGINT NOT NULL,
  sample_at BIGINT NOT NULL,
  users INT NOT NULL,
  devices INT NOT NULL,
  PRIMARY KEY(agent_user_id,sample_at),
  INDEX(sample_at)
)
```

使用 `information_schema.columns` 为角色表增加 `agent_mode TINYINT NOT NULL DEFAULT 0` 和可空 `role_code VARCHAR(32)`，再检查并创建 `role_code` 唯一索引。

`seedMentorRole` 先按 `role_code='mentor'` 查询；不存在时插入导师角色，再从权限目录中插入 `permission_key<>'manage.settings'` 的权限。存在时直接返回，不改写角色或权限。

- [ ] **Step 4: 更新全新数据库 schema 与 schema 检查**

在 `schema.sql` 写入相同字段和三张表。`CheckSchema` 的附加表清单加入三张导师表，保证残缺升级不能静默启动。

- [ ] **Step 5: 运行数据库相关测试**

Run: `go test ./internal/server -run 'Test.*(Addon|MentorRole|Schema)' -count=1`

Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add internal/server/addon_schema.go internal/server/addon_schema_test.go internal/server/schema.sql internal/server/migrate.go
git commit -m "feat: seed mentor role and agent schema"
```

### Task 2: 角色代理模式 API 与角色页面

**Files:**
- Modify: `internal/server/admin_roles.go`
- Modify: `internal/server/admin_roles_test.go`
- Modify: `internal/server/admin.go`
- Modify: `frontend/role-panel.js`
- Modify: `frontend/role-panel.css`
- Modify: `frontend/member-role-select.js`
- Modify: `scripts/build-maintenance.cjs`
- Modify: `scripts/test-rbac.cjs`

**Interfaces:**
- Role list fields: `agent_mode int64`, `role_code string`。
- Role save input: `agent_mode` must be `0` or `1`。
- Member list fields: `admin_role_name`, `admin_role_agent_mode`。

- [ ] **Step 1: 写角色代理模式失败测试**

增加测试：

```go
func TestSaveAdminRolePersistsAgentMode(t *testing.T) { /* expect INSERT agent_mode=1 */ }
func TestMentorPresetCannotBeDeleted(t *testing.T) { /* role_code=mentor => 409 */ }
func TestAttachAdminRoleNamesIncludesAgentMode(t *testing.T) { /* row field is 1 */ }
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/server -run 'Test.*(AgentMode|MentorPreset)' -count=1`

Expected: FAIL。

- [ ] **Step 3: 扩展角色 API**

角色列表和详情查询返回 `agent_mode,role_code`。保存时验证 `agent_mode` 只能是 0 或 1，并写入角色表。删除前查询 `role_code`，导师预置角色返回 `409 导师专员角色不能删除`。成员角色装饰查询同时读取 `agent_mode`，写入 `admin_role_agent_mode`。

- [ ] **Step 4: 添加角色页面代理开关**

`ImgoRolePanel.create()` 默认：

```js
{ role_id: 0, name: '', remark: '', status: 1, agent_mode: 1, permissions: [], builtin: false }
```

表单增加“代理模式”开关和说明“开启后，该角色只能管理自己的邀请下级”。固定的超级管理员、普通用户仍只读；导师预置角色允许修改开关和权限，但不显示删除按钮。

- [ ] **Step 5: 更新前端断言并重建**

`test-rbac.cjs` 断言角色源文件包含 `agent_mode`、默认值 `1`、导师不可删除逻辑。运行：

```bash
node scripts/build-maintenance.cjs
node scripts/build-maintenance.cjs
node scripts/test-rbac.cjs
```

Expected: 两次构建 hash 相同，断言 PASS。

- [ ] **Step 6: 提交**

```bash
git add internal/server/admin_roles.go internal/server/admin_roles_test.go internal/server/admin.go frontend/role-panel.js frontend/role-panel.css frontend/member-role-select.js scripts/build-maintenance.cjs scripts/test-rbac.cjs public
git commit -m "feat: add agent mode to admin roles"
```

### Task 3: 统一代理数据范围组件

**Files:**
- Create: `internal/server/admin_scope.go`
- Create: `internal/server/admin_scope_test.go`
- Modify: `internal/server/rbac.go`
- Modify: `internal/server/auth.go`

**Interfaces:**
- Produces:

```go
type adminScope struct {
    Global bool
    AgentUserID int64
}
func (a *App) adminScope(ctx context.Context, user M) (adminScope, error)
func (s adminScope) userPredicate(alias string) (string, []any)
func (a *App) requireScopedUser(ctx context.Context, db DB, scope adminScope, userID int64) error
func (a *App) nearestAgent(ctx context.Context, db DB, userID int64) (int64, error)
```

- [ ] **Step 1: 写范围矩阵失败测试**

覆盖：超级管理员全局、非代理自定义角色全局、代理角色返回当前用户 ID、普通用户拒绝、下级允许、非下级拒绝、最近导师按最小深度选择。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/server -run 'Test(AdminScope|NearestAgent|RequireScopedUser)' -count=1`

Expected: FAIL，类型与函数尚不存在。

- [ ] **Step 3: 实现范围解析**

`adminScope` 查询启用角色的 `agent_mode`。`userPredicate("u")` 对全局返回 `1=1`，对导师返回：

```sql
EXISTS (
  SELECT 1 FROM prefix_imgo_referral_path scope_path
  WHERE scope_path.ancestor_user_id=?
    AND scope_path.descendant_user_id=u.user_id
)
```

`requireScopedUser` 用单条 `SELECT 1 FROM user u WHERE u.user_id=? AND ...` 校验。失败统一返回 `deny()`。

`nearestAgent` 连接邀请路径、用户和角色表，要求角色启用及 `agent_mode=1`，按 `depth ASC LIMIT 1`。

- [ ] **Step 4: 登录返回代理标志**

`adminAccessInfo` 增加 `agent_mode`，使前端可以显示当前账号数据范围。保持超级管理员 `agent_mode=0`。

- [ ] **Step 5: 运行范围与 RBAC 测试**

Run: `go test ./internal/server -run 'Test(AdminScope|NearestAgent|RequireScopedUser|AdminAccess)' -count=1`

Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add internal/server/admin_scope.go internal/server/admin_scope_test.go internal/server/rbac.go internal/server/auth.go
git commit -m "feat: centralize mentor data scope"
```

### Task 4: 成员列表、详情与修改范围

**Files:**
- Modify: `internal/server/admin.go`
- Modify: `internal/server/admin_check_in_test.go`
- Create: `internal/server/admin_scope_users_test.go`
- Modify: `frontend/member-referral-filter.js`
- Modify: `scripts/test-rbac.cjs`

**Interfaces:**
- Consumes: `adminScope`, `userPredicate`, `requireScopedUser`。
- Member request: `referral_scope` accepts empty, `direct`, `all`。

- [ ] **Step 1: 写导师成员查询失败测试**

测试代理账号在空范围时自动增加 `ancestor_user_id=当前账号`；`direct` 再增加 `depth=1`；用户名查询在同一范围内精确优先、模糊回退。测试详情和修改非下级返回 403。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/server -run 'TestAgentManageUser' -count=1`

Expected: FAIL，当前查询仍可访问全站用户。

- [ ] **Step 3: 修改成员列表查询**

代理模式下始终叠加当前导师的邀请路径。空值与 `all` 都查询全部深度，`direct` 查询深度 1。非代理全局角色保留现有超级管理员按指定用户名查看某邀请人下级的能力。

- [ ] **Step 4: 修改成员详情和所有写操作**

在 `detail`、`checkInHistory`、`edit`、`setRemark`、`setInviteCode`、`del`、`setStatus`、`editPassword` 前调用 `requireScopedUser`。导师不能操作自己、其他导师或非下级账号。

导师执行 `add` 时，在同一事务中调用 `bindInviter(ctx, tx, newUserID, r.uid())`，让新成员成为直属下级。

- [ ] **Step 5: 更新成员筛选前端**

代理账号进入成员页时默认 `referral_scope='all'`，只显示“直属下级/全部下级”；超级管理员继续允许空白范围。用户名输入保持最左侧。

- [ ] **Step 6: 运行成员测试和前端断言**

```bash
go test ./internal/server -run 'Test(AgentManageUser|ManageUserReferral)' -count=1
node scripts/build-maintenance.cjs
node scripts/test-rbac.cjs
```

Expected: PASS。

- [ ] **Step 7: 提交**

```bash
git add internal/server/admin.go internal/server/admin_check_in_test.go internal/server/admin_scope_users_test.go frontend/member-referral-filter.js scripts/test-rbac.cjs public
git commit -m "feat: scope member management to mentor descendants"
```

### Task 5: 导师设置 API 与注册继承逻辑

**Files:**
- Create: `internal/server/agent_settings.go`
- Create: `internal/server/agent_settings_test.go`
- Modify: `internal/server/routes.go`
- Modify: `internal/server/registration.go`
- Modify: `internal/server/integration_test.go`

**Interfaces:**
- Produces: `manageAgentSetting(r *request) (any, error)`。
- Produces:

```go
type registrationAutomation struct {
    AgentUserID int64
    AutoUser M
    AutoGroup M
    UserInherited bool
    GroupInherited bool
}
func (a *App) registrationAutomation(ctx context.Context, db DB, inviterID int64) (registrationAutomation, error)
```

- [ ] **Step 1: 写导师设置 API 失败测试**

测试 `detail` 返回两个继承标志、覆盖配置和全局摘要；`save` 将继承项写为 SQL `NULL`，将显式关闭写为 JSON；非导师目标拒绝；覆盖客服和群主超出导师范围拒绝。

- [ ] **Step 2: 写注册继承失败测试**

覆盖四种组合：两项继承、仅客服覆盖、仅群聊覆盖、两项覆盖；再覆盖通过导师孙级邀请码注册仍解析到导师，以及嵌套导师选择最近导师。

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./internal/server -run 'Test(AgentSetting|RegistrationAutomation)' -count=1`

Expected: FAIL。

- [ ] **Step 4: 实现导师设置接口**

注册 `/manage/agentSetting/detail` 和 `/manage/agentSetting/save` 为超级管理员专用。保存前确认目标用户角色启用且 `agent_mode=1`。复用全局自动配置的字段校验，并调用 `requireScopedUser` 验证客服与群主；导师本人作为客服或群主时明确允许。

- [ ] **Step 5: 重构注册自动分配**

先绑定邀请路径，再调用 `registrationAutomation`。继承项使用全局 `autoTask`，覆盖项使用 `imgo_agent_auto_state` 并以 `SELECT ... FOR UPDATE` 锁定。客服与群聊分别读取自己的继承状态，允许一项继承、另一项覆盖。

将现有自动好友、欢迎消息和自动群代码提取为接受配置与状态存储器的内部函数，保持事务原子性。

- [ ] **Step 6: 运行单元与集成测试**

Run: `go test ./internal/server -run 'Test(AgentSetting|RegistrationAutomation|RegistrationAuto)' -count=1`

Expected: PASS。

- [ ] **Step 7: 提交**

```bash
git add internal/server/agent_settings.go internal/server/agent_settings_test.go internal/server/routes.go internal/server/registration.go internal/server/integration_test.go
git commit -m "feat: add mentor registration automation overrides"
```

### Task 6: 群聊、消息、文件、银行卡与财务范围

**Files:**
- Modify: `internal/server/admin.go`
- Modify: `internal/server/chat.go`
- Modify: `internal/server/files.go`
- Modify: `internal/server/bank_card.go`
- Modify: `internal/server/wallet_admin.go`
- Modify: `internal/server/wallet_orders.go`
- Create: `internal/server/admin_scope_resources_test.go`

**Interfaces:**
- Consumes: `adminScope`, `requireScopedUser`。
- Produces resource predicates for group owner, message participant, file owner and financial user.

- [ ] **Step 1: 写各资源允许/拒绝失败测试**

每类资源至少覆盖：导师下级记录允许、其他导师记录拒绝、超级管理员允许。写操作同时断言 SQL `UPDATE` 或事务锁查询带范围校验。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/server -run 'TestAgentScope(Resources|Group|Message|File|Bank|Wallet)' -count=1`

Expected: FAIL。

- [ ] **Step 3: 限制群聊和消息**

群列表与群操作要求 `group.owner_id` 属于导师下级。转让群主、添加或管理群成员时目标用户也必须属于下级。

后台消息查询使用：私聊的 `from_user` 或 `to_user` 属于下级；群聊的群主属于下级。`getContacts` 与 `dealMsg` 在读取内容前执行相同校验。

- [ ] **Step 4: 限制文件和银行卡**

后台文件 `is_all=1` 查询增加 `file.user_id` 下级条件；下载和预览的后台授权路径叠加范围。银行卡列表、详情和编辑按 `bank_card.user_id` 校验。

- [ ] **Step 5: 限制钱包和财务事务**

账户、流水、充值单、提现单和详情按所属 `user_id` 过滤。充值、人工入账、提现代办和审核在事务外先校验，并在锁定订单后对订单所属用户再次调用事务内范围查询。

- [ ] **Step 6: 禁止代理执行全站修改**

代理模式对 `publishNotice`、`delNotice`、`clearMessage`、任务配置和任务启停返回 403；`noticeList` 保持只读。系统设置本来没有默认权限，后端仍以 `manage.settings` 校验。

- [ ] **Step 7: 运行资源测试**

Run: `go test ./internal/server -run 'TestAgentScope(Resources|Group|Message|File|Bank|Wallet)' -count=1`

Expected: PASS。

- [ ] **Step 8: 提交**

```bash
git add internal/server/admin.go internal/server/chat.go internal/server/files.go internal/server/bank_card.go internal/server/wallet_admin.go internal/server/wallet_orders.go internal/server/admin_scope_resources_test.go
git commit -m "feat: enforce mentor scope across admin resources"
```

### Task 7: 导师概况和在线趋势

**Files:**
- Modify: `internal/server/overview.go`
- Modify: `internal/server/overview_test.go`
- Modify: `internal/server/hub.go`

**Interfaces:**
- Consumes: `adminScope`。
- Produces: `Hub.onlineCountsForUsers(allowed map[int64]bool, now time.Time) (int, int)`。
- Produces: minute samples in `imgo_agent_online_sample`。

- [ ] **Step 1: 写概况范围失败测试**

断言代理概况的用户、群、消息、文件总数与趋势 SQL 均包含下级谓词；在线人数只计算下级连接；超级管理员仍读取全局统计。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/server -run 'Test(AgentOverview|OnlineCountsForUsers)' -count=1`

Expected: FAIL。

- [ ] **Step 3: 实现范围化总数与趋势**

让 `trend` 接受查询参数，避免拼接用户输入。按规范分别构造用户、群主、消息参与者和文件上传者范围。代理概况不读取全局在线峰值。

- [ ] **Step 4: 实现导师在线采样**

每分钟查询所有启用的代理角色账号，按其下级 ID 集合统计 Hub 连接并 `INSERT ... ON DUPLICATE KEY UPDATE` 到 `imgo_agent_online_sample`。代理概况读取自己的样本；升级前时段自然为空。

- [ ] **Step 5: 运行概况测试**

Run: `go test ./internal/server -run 'Test(AgentOverview|OnlineCountsForUsers|Overview)' -count=1`

Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add internal/server/overview.go internal/server/overview_test.go internal/server/hub.go
git commit -m "feat: scope dashboard metrics to mentor teams"
```

### Task 8: 成员列表导师设置弹窗

**Files:**
- Create: `frontend/member-agent-setting.js`
- Create: `frontend/member-agent-setting.css`
- Modify: `frontend/member-actions.js`
- Modify: `scripts/build-maintenance.cjs`
- Modify: `scripts/test-rbac.cjs`

**Interfaces:**
- Consumes: `/manage/agentSetting/detail`、`/manage/agentSetting/save`。
- Opens only for rows with `admin_role_agent_mode=1`。

- [ ] **Step 1: 添加前端失败断言**

`test-rbac.cjs` 断言：导师行显示“导师设置”；普通行不显示；弹窗含两个“继承全局设置”开关；保存调用导师设置 API。

- [ ] **Step 2: 运行断言确认失败**

Run: `node scripts/test-rbac.cjs`

Expected: FAIL，缺少组件和 API。

- [ ] **Step 3: 实现导师设置弹窗**

组件打开时并行读取导师设置详情和下级用户选项。每项继承开关开启时禁用覆盖字段并显示全局摘要；关闭时允许显式开关功能及编辑详细配置。保存发送：

```js
{
  agent_user_id,
  inherit_auto_user,
  inherit_auto_group,
  auto_add_user,
  auto_add_group
}
```

成员“更多”菜单仅在超级管理员查看 `admin_role_agent_mode=1` 的行时显示入口。

- [ ] **Step 4: 注入 API、组件和样式**

在构建适配器中加入 `ImgoAgentSettingApi`、弹窗组件、成员操作事件和 CSS。保持构建幂等并清理旧 hash 资源。

- [ ] **Step 5: 重建和验证前端**

```bash
node scripts/build-maintenance.cjs
node scripts/build-maintenance.cjs
node scripts/test-rbac.cjs
node scripts/test-group-avatar.cjs
for f in public/assets/js/app.85372e4e.js public/assets/js/585.imgo*.js public/assets/js/687.imgo*.js; do node --check "$f"; done
```

Expected: 两次 hash 相同，所有断言和语法检查 PASS。

- [ ] **Step 6: 提交**

```bash
git add frontend/member-agent-setting.js frontend/member-agent-setting.css frontend/member-actions.js scripts/build-maintenance.cjs scripts/test-rbac.cjs public
git commit -m "feat: add mentor automation settings dialog"
```

### Task 9: 完整验证、本地迁移与 Linux 打包

**Files:**
- Modify: `docs/MIGRATION.md`
- Modify: `docs/DEPLOY-LINUX.md`
- Verify: `scripts/package-linux.sh`

**Interfaces:**
- Produces: `dist/Imgo-linux-amd64.tar.gz` and checksum。

- [ ] **Step 1: 更新部署和迁移说明**

记录自动创建导师角色、已有角色保持非代理模式、导师设置继承规则、启动账号需 `CREATE/ALTER/INSERT` 权限，以及在线趋势从升级后开始采样。

- [ ] **Step 2: 运行完整 Go 验证**

```bash
gofmt -w internal/server/*.go
GOCACHE=/tmp/imgo-agent-go-cache go test ./... -count=1
GOCACHE=/tmp/imgo-agent-go-cache go vet ./...
```

Expected: PASS。

- [ ] **Step 3: 运行完整前端验证**

```bash
node scripts/build-maintenance.cjs
node scripts/build-maintenance.cjs
for f in scripts/test-*.cjs; do node "$f"; done
```

Expected: 两次构建一致，全部脚本 PASS。

- [ ] **Step 4: 在本地数据库验证自动升级**

重启 `go run ./cmd/imgo`，用管理员登录并调用：

```text
POST /manage/role/index
POST /manage/agentSetting/detail
```

确认导师专员只出现一次、默认七项权限、`agent_mode=1`，且未配置导师返回继承全局。

- [ ] **Step 5: 打包 Linux x86_64**

Run: `./scripts/package-linux.sh`

验证：

```bash
file dist/Imgo-linux-amd64/bin/imgo
tar -tzf dist/Imgo-linux-amd64.tar.gz | rg '(^|/)\._|\.DS_Store' && exit 1 || true
cat dist/Imgo-linux-amd64.tar.gz.sha256
```

Expected: ELF 64-bit x86-64、归档无 macOS 元数据、生成 SHA256。

- [ ] **Step 6: 提交并推送**

```bash
git add docs/MIGRATION.md docs/DEPLOY-LINUX.md public
git commit -m "docs: document mentor agent deployment"
git push origin IMGO
```
