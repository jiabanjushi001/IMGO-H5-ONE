# 后台角色与菜单权限设计

## 目标

在现有“超级管理员 / 普通用户”模型上增加可配置后台角色。一个用户只能绑定一个角色；菜单可见性和对应后台 API 必须使用同一套权限判断。

## 已确认规则

- `user_id=1` 为超级管理员，始终拥有全部后台权限，不能被降级、禁用或删除。
- 普通用户不绑定后台角色，只能使用聊天功能，不能访问 `/manage/*`。
- 一个用户最多绑定一个自定义角色。
- 角色被禁用后，该角色下用户的后台权限立即失效。
- 角色管理和用户角色分配仅限超级管理员，避免角色自行提权。
- 已绑定用户的角色不能删除；必须先将用户改为普通用户或转移到其他角色。
- 数据库结构由服务启动迁移自动创建，不要求部署者手动执行 SQL。

## 数据模型

### `imgo_admin_role`

- `role_id BIGINT AUTO_INCREMENT PRIMARY KEY`
- `name VARCHAR(64) NOT NULL UNIQUE`
- `remark VARCHAR(255) NOT NULL DEFAULT ''`
- `status TINYINT NOT NULL DEFAULT 1`
- `created_at BIGINT NOT NULL`
- `updated_at BIGINT NOT NULL`

### `imgo_admin_permission`

- `permission_id BIGINT AUTO_INCREMENT PRIMARY KEY`
- `permission_key VARCHAR(64) NOT NULL UNIQUE`
- `name VARCHAR(64) NOT NULL`
- `menu_path VARCHAR(128) NOT NULL DEFAULT ''`
- `sort INT NOT NULL DEFAULT 0`

程序启动时按权限键补齐内置权限，不覆盖角色已有授权。

### `imgo_admin_role_permission`

- `role_id BIGINT NOT NULL`
- `permission_id BIGINT NOT NULL`
- 联合主键 `(role_id, permission_id)`

### 用户表

增加 `admin_role_id BIGINT NOT NULL DEFAULT 0`。`0` 表示普通用户；大于 `0` 表示绑定一个自定义后台角色。保留旧 `role` 字段用于兼容聊天和群聊逻辑：超级管理员为 `1`，自定义后台角色用户为 `2`，普通用户为 `0`。

## 权限清单

| 权限键 | 菜单 | 路由 |
| --- | --- | --- |
| `manage.overview` | 概况 | `/manage/index` |
| `manage.settings` | 设置 | `/manage/setting` |
| `manage.users` | 成员 | `/manage/user` |
| `manage.messages` | 消息 | `/manage/message` |
| `manage.groups` | 群聊 | `/manage/group` |
| `manage.files` | 文件 | `/manage/files` |
| `manage.bank` | 绑卡 | `/manage/bank` |
| `manage.finance` | 财务 | `/manage/finance/*` |

角色管理使用保留权限 `manage.roles`，但不出现在可分配复选框中，只允许超级管理员访问。

## 后端鉴权

`endpoint` 增加权限键。请求流程为：

1. 验证 token 并读取当前用户。
2. 超级管理员直接通过。
3. 普通用户访问后台接口返回 403。
4. 自定义角色必须存在且处于启用状态。
5. 接口要求的权限键必须存在于角色权限表，否则返回 403。

鉴权每次请求读取当前角色状态和授权，因此禁用角色、删除授权、重新分配角色后立即生效。可使用短时缓存降低查询量，但任何角色写操作必须主动清除缓存。

聊天使用的 `/enterprise/*` 接口继续按原有规则工作。若后台页面复用了聊天接口，则为后台管理增加独立 `/manage/*` 接口，避免限制普通用户的聊天功能。

## 后台接口

- `POST /manage/role/index`：角色列表及用户数量。
- `POST /manage/role/detail`：角色详情和已选权限。
- `POST /manage/role/save`：新增或编辑角色。
- `POST /manage/role/setStatus`：启用或禁用角色。
- `POST /manage/role/del`：删除未绑定用户的角色。
- `POST /manage/role/permissions`：返回可分配权限清单。
- `POST /manage/user/setRole`：参数改为 `user_id`、`admin_role_id`；`0` 表示普通用户。

登录响应的 `userInfo` 增加：

- `admin_role_id`
- `admin_role_name`
- `menu_permissions`：允许访问的权限键数组

超级管理员返回全部权限；普通用户返回空数组。

## 前端交互

侧边栏增加“角色权限”，仅超级管理员可见。角色页面左侧为角色列表，右侧为角色名称、备注、状态和菜单权限复选框。

成员页面角色列显示“超级管理员”“普通用户”或自定义角色名称。超级管理员可直接为用户选择角色。角色修改成功后立即更新当前表格行。

路由规则：

- 普通用户登录后进入 `/chat`。
- 自定义角色进入第一个有权限的后台页面。
- 用户手动打开无权限路由时，提示“无权操作”并跳到第一个可访问页面。
- 服务端仍执行相同权限校验，前端隐藏菜单不能替代接口鉴权。

## 错误处理

- 角色名称为空或重复：400。
- 角色不存在：404。
- 角色已绑定用户：409，并返回绑定数量。
- 非超级管理员操作角色或分配角色：403。
- 访问未授权菜单接口：403，消息为“无权操作”。
- 数据库迁移失败：启动失败并记录具体表或字段。

## 验证

- Go 单元测试覆盖普通用户、超级管理员、自定义角色、禁用角色和权限变更。
- 集成测试覆盖角色 CRUD、重复名称、删除已使用角色、用户单角色分配和 API 越权。
- 前端构建断言覆盖角色路由、菜单过滤、成员角色选择和登录跳转。
- 浏览器分别使用超级管理员、普通用户和受限角色验证菜单与直接 URL 访问。
