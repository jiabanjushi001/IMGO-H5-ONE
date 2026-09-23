# 后台角色与菜单权限 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 增加一个用户绑定一个后台角色的 RBAC，并让后台菜单与 Go API 使用同一套权限。

**Architecture:** 使用规范化角色、权限和关联表，用户表以 `admin_role_id` 指向单个角色。Gin 路由声明所需权限并在统一 dispatch 中校验；Vue 2 发行版通过可复现适配器增加角色页面、动态成员角色选择和菜单过滤。

**Tech Stack:** Go、Gin、MySQL、Vue 2 Options API/render functions、Element UI、Node.js 构建适配器

**Spec:** `docs/superpowers/specs/2026-09-23-role-menu-permissions.md`

## Global Constraints

- 一个用户只能绑定一个角色。
- 菜单权限和对应后台 API 必须同步限制。
- `user_id=1` 超级管理员始终拥有全部权限且不能被降级。
- 普通用户只能聊天。
- 数据库升级自动补表和字段。
- 角色管理及角色分配仅允许超级管理员。

---

### Task 1: 数据库角色模型与自动迁移

**Files:**
- Modify: `internal/server/addon_schema.go`
- Modify: `internal/server/schema.sql`
- Test: `internal/server/addon_schema_test.go`

**Interfaces:**
- Produces: `user.admin_role_id`、`imgo_admin_role`、`imgo_admin_permission`、`imgo_admin_role_permission`
- Produces: `seedAdminPermissions(ctx context.Context, db dbExecutor) error`

- [ ] **Step 1: 写失败测试**

在 `addon_schema_test.go` 增加测试，断言空附加库迁移后存在三张 RBAC 表、用户表存在 `admin_role_id`，并且八个 `manage.*` 权限键均存在。

- [ ] **Step 2: 运行失败测试**

Run: `GOCACHE=/tmp/imgo-rbac-go-cache go test ./internal/server -run 'TestEnsureAddonTables.*Role' -count=1`

Expected: FAIL，缺少角色表或字段。

- [ ] **Step 3: 实现最小迁移**

在 `ensureAddonTables` 中创建三张表，按 `information_schema.columns` 自动增加 `admin_role_id`，并以 `INSERT ... ON DUPLICATE KEY UPDATE name=VALUES(name),menu_path=VALUES(menu_path),sort=VALUES(sort)` 补齐内置权限。

- [ ] **Step 4: 更新全新安装结构**

在 `schema.sql` 用户表加入 `admin_role_id`，附加表仍由启动迁移创建，确保全新安装与升级路径一致。

- [ ] **Step 5: 运行测试**

Run: `GOCACHE=/tmp/imgo-rbac-go-cache go test ./internal/server -run 'TestEnsureAddonTables' -count=1`

Expected: PASS。

### Task 2: 权限服务与统一 API 拦截

**Files:**
- Create: `internal/server/rbac.go`
- Create: `internal/server/rbac_test.go`
- Modify: `internal/server/app.go`
- Modify: `internal/server/routes.go`

**Interfaces:**
- Produces: `func (a *App) menuPermissions(ctx context.Context, user M) ([]string, error)`
- Produces: `func (a *App) authorizeManage(ctx context.Context, user M, permission string) error`
- Changes: `endpoint` includes `permission string`

- [ ] **Step 1: 写权限矩阵失败测试**

覆盖超级管理员绕过、普通用户拒绝、自定义角色允许、缺少权限拒绝、禁用角色拒绝五种情况；403 必须返回“无权操作”。

- [ ] **Step 2: 运行失败测试**

Run: `GOCACHE=/tmp/imgo-rbac-go-cache go test ./internal/server -run 'TestAuthorizeManage' -count=1`

Expected: FAIL，权限函数不存在。

- [ ] **Step 3: 实现权限查询和校验**

查询用户的 `admin_role_id`、角色状态以及关联权限。`user_id=1` 返回全部权限；`admin_role_id=0` 返回空权限。禁止仅依赖旧 `role>0`。

- [ ] **Step 4: 为路由声明权限**

将 `/manage/index`、`config/task`、`user`、`message`、`group`、`bank`、`wallet` 分别映射到规格中的权限键；角色 CRUD 和 `setRole` 保持超级管理员专用。

- [ ] **Step 5: 在 dispatch 中统一执行**

认证成功后，若 endpoint 含权限键则调用 `authorizeManage`。保留 `super` 作为角色管理等不可委派操作的硬限制。

- [ ] **Step 6: 运行测试**

Run: `GOCACHE=/tmp/imgo-rbac-go-cache go test ./internal/server -run 'TestAuthorizeManage|TestRoutes' -count=1`

Expected: PASS。

### Task 3: 角色 CRUD 与单角色分配

**Files:**
- Create: `internal/server/admin_roles.go`
- Create: `internal/server/admin_roles_test.go`
- Modify: `internal/server/routes.go`
- Modify: `internal/server/admin.go`

**Interfaces:**
- Produces endpoints: `/manage/role/index|detail|save|setStatus|del|permissions`
- Changes endpoint: `/manage/user/setRole` accepts `{user_id, admin_role_id}`

- [ ] **Step 1: 写角色 CRUD 失败测试**

测试新增角色、编辑权限、名称重复、禁用角色、删除空角色、拒绝删除已绑定角色以及非超级管理员访问 403。

- [ ] **Step 2: 写分配失败测试**

测试 `admin_role_id=0` 还原普通用户；大于零时校验角色存在且启用；禁止修改 `user_id=1`；保存时同步旧 `role` 为 `0/2`。

- [ ] **Step 3: 运行失败测试**

Run: `GOCACHE=/tmp/imgo-rbac-go-cache go test ./internal/server -run 'TestManageRole|TestSetAdminRole' -count=1`

Expected: FAIL，路由或 handler 不存在。

- [ ] **Step 4: 事务实现保存**

角色基本信息和权限关联在同一事务内保存。权限键只允许来自内置权限表；禁止写入 `manage.roles`。

- [ ] **Step 5: 实现删除与状态约束**

删除前统计 `user.admin_role_id`；大于零返回 409。状态仅允许 `0/1`。

- [ ] **Step 6: 实现成员列表角色信息**

成员列表和详情返回 `admin_role_id`、`admin_role_name`。旧数据中 `role=2` 且没有角色的用户迁移为普通用户，避免保留无限权限管理员。

- [ ] **Step 7: 运行测试**

Run: `GOCACHE=/tmp/imgo-rbac-go-cache go test ./internal/server -run 'TestManageRole|TestSetAdminRole|TestManageUser' -count=1`

Expected: PASS。

### Task 4: 登录权限数据和路由落点

**Files:**
- Modify: `internal/server/auth.go`
- Modify: `internal/server/auth_test.go`
- Modify: `internal/server/integration_test.go`

**Interfaces:**
- Produces userInfo fields: `admin_role_id int64`, `admin_role_name string`, `menu_permissions []string`

- [ ] **Step 1: 写登录响应失败测试**

分别断言超级管理员返回全部权限、普通用户返回空数组、自定义角色只返回已授权键。

- [ ] **Step 2: 运行失败测试**

Run: `GOCACHE=/tmp/imgo-rbac-go-cache go test ./internal/server -run 'Test.*Login.*Permission' -count=1`

Expected: FAIL，登录响应缺少权限字段。

- [ ] **Step 3: 扩展登录 userInfo**

登录成功后调用 `menuPermissions` 并附加角色名称与权限数组；查询失败时登录失败，避免默认放行。

- [ ] **Step 4: 增加越权集成测试**

创建仅有成员权限的角色，断言 `/manage/user/index` 成功、`/manage/group/index` 返回 403；禁用角色后前一个接口也返回 403。

- [ ] **Step 5: 运行测试**

Run: `GOCACHE=/tmp/imgo-rbac-go-cache go test ./internal/server -run 'Test.*Login.*Permission|Test.*Role.*Access' -count=1`

Expected: PASS。

### Task 5: 角色权限管理页面

**Files:**
- Create: `frontend/role-panel.js`
- Create: `frontend/role-panel.css`
- Modify: `scripts/build-maintenance.cjs`
- Modify: `public/index.html`
- Test: `scripts/build-maintenance.test.cjs`

**Interfaces:**
- Consumes role APIs from Task 3
- Produces component: `ImgoRolePanel`
- Produces route: `/manage/role`

- [ ] **Step 1: 添加构建失败断言**

断言生成包包含 `/manage/role`、`角色权限`、角色 API、权限复选框和 `ImgoRolePanel`。

- [ ] **Step 2: 运行失败测试**

Run: `node --test scripts/build-maintenance.test.cjs`

Expected: FAIL，角色页面尚未注入。

- [ ] **Step 3: 实现 Options API render 组件**

组件状态包含 `roles`、`activeRoleId`、`form`、`permissions`、`loading`；方法包含 `loadRoles`、`selectRole`、`newRole`、`saveRole`、`setStatus`、`deleteRole`。子组件不直接修改父级 props。

- [ ] **Step 4: 注入路由和样式**

通过构建适配器把角色路由注入管理路由列表；仅 `user_id=1` 时保留该菜单。CSS 使用 `imgo-role-*` 类名隔离。

- [ ] **Step 5: 运行构建测试**

Run: `node --test scripts/build-maintenance.test.cjs && npm run build`

Expected: PASS，构建输出不包含未替换锚点错误。

### Task 6: 菜单过滤和成员角色分配

**Files:**
- Create: `frontend/rbac-menu.js`
- Create: `frontend/member-role-select.js`
- Modify: `scripts/build-maintenance.cjs`
- Modify: `frontend/admin-theme.css`
- Test: `scripts/build-maintenance.test.cjs`

**Interfaces:**
- Consumes `userInfo.menu_permissions`
- Consumes `/manage/role/index` and `/manage/user/setRole`

- [ ] **Step 1: 添加失败断言**

断言菜单映射包含八个权限键；普通用户目标路由为 `/chat`；成员角色请求发送 `admin_role_id`。

- [ ] **Step 2: 运行失败测试**

Run: `node --test scripts/build-maintenance.test.cjs`

Expected: FAIL，旧菜单仍按 `role>0` 展示。

- [ ] **Step 3: 实现菜单过滤**

超级管理员保留所有菜单；其他用户根据 `menu_permissions` 过滤管理路由。无后台权限时跳转 `/chat`，直接访问无权限管理路由时提示并跳到第一个允许路由。

- [ ] **Step 4: 实现成员角色选择**

角色列加载角色列表，显示角色名称。选择后调用 `setRole`，成功立即更新行数据；失败则恢复原值。超级管理员行不可修改。

- [ ] **Step 5: 运行构建和静态验证**

Run: `node --test scripts/build-maintenance.test.cjs && npm run build`

Expected: PASS。

### Task 7: 全量验证和 Linux 打包

**Files:**
- Modify if needed: `docs/MIGRATION.md`
- Output: `dist/Imgo-linux-amd64.tar.gz`

**Interfaces:**
- Consumes all previous tasks
- Produces deployable Linux x86_64 archive

- [ ] **Step 1: 运行 Go 全量测试和静态检查**

Run: `GOCACHE=/tmp/imgo-rbac-go-cache go test ./... && GOCACHE=/tmp/imgo-rbac-go-cache go vet ./... && test -z "$(gofmt -l internal cmd)"`

Expected: 全部成功且 gofmt 无输出。

- [ ] **Step 2: 运行前端构建验证**

Run: `node --test scripts/build-maintenance.test.cjs && npm run build`

Expected: PASS。

- [ ] **Step 3: 本地接口烟雾验证**

使用超级管理员、普通用户、仅成员权限角色各登录一次，验证菜单响应和允许/拒绝接口与规格一致。

- [ ] **Step 4: 更新迁移说明**

写明服务启动自动创建角色表和 `admin_role_id`；旧普通管理员不会自动获得无限权限，需超级管理员重新分配角色。

- [ ] **Step 5: 生成 Linux x86_64 包**

Run: `npm run package:linux`

Expected: `dist/Imgo-linux-amd64.tar.gz` 存在，归档内没有 macOS `._*` 文件，`bin/imgo` 为 Linux x86_64 ELF。

- [ ] **Step 6: 输出校验值**

Run: `shasum -a 256 dist/Imgo-linux-amd64.tar.gz`

Expected: 输出单一 SHA-256 校验值。
