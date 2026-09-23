# Imgo Linux x86_64 部署

此包包含 Linux amd64 静态编译程序及 PC 网页/管理后台，不包含 uniapp H5 构建产物、数据库数据、本机密钥或上传附件。无需安装 Go 或 PHP，需要 MySQL 8。

## 配置和启动

```sh
cp .env.example .env
chmod 600 .env
chmod +x start.sh run.sh bin/imgo
```

编辑 `.env` 后，先创建空数据库。全新空库执行：

```sh
./start.sh -init
ADMIN_ACCOUNT=administrator ADMIN_PASSWORD='替换为强密码' ./start.sh -create-admin
```

已有数据库直接启动。程序会自动补齐 Go 扩展表、角色权限表和兼容字段；涉及旧数据重写的迁移仍按 `docs/MIGRATION.md` 执行。

启动使用的数据库账号除日常读写权限外，必须具有 `CREATE`、`ALTER`、`INSERT` 权限，以完成自动升级。启动日志出现 `automatic database upgrade failed` 时，先修复数据库权限或报错原因，再重新启动。

前台运行：`./start.sh`。守护运行：`./run.sh start`。可用 `status`、`restart`、`stop`、`logs` 管理进程。不要同时使用 `run.sh`、systemd 和宝塔进程管理器启动多个实例。

默认使用 Nginx 反向代理 `127.0.0.1:8080`，并为 `/wss` 配置 WebSocket Upgrade。如果修改了 `.env` 中的 `IMGO_ADDR`，Nginx 的 `proxy_pass` 也要使用同一端口。API 域名和后台域名分离的示例位于 `deploy/domain-split/`。

## 更新已有部署

保留服务器原来的 `.env` 与 `public/storage`。停止旧进程后更新 `bin/imgo`、`public`、`start.sh` 和 `run.sh`，再按原来的进程管理方式启动。不要用 `.env.example` 覆盖生产配置。

更新前备份数据库。此次普通启动会自动创建导师设置、状态和在线采样表，并幂等创建“导师专员”角色：默认代理模式开启，默认七项菜单权限（不含系统设置）。已有自定义角色保持非代理模式；已存在的导师预置角色及其权限不会被重置。无需导入附加 SQL，不要对已有数据库运行 `-init`。

超级管理员可在成员列表授予导师角色并打开“导师设置”。自动客服与自动群聊默认分别继承全局设置；关闭继承后可独立配置或显式关闭，覆盖配置使用导师自己的自动分配状态。代理后台仅显示导师邀请树中的下级，在线趋势从升级后开始采样。完整规则见 `docs/MIGRATION.md`。

## 验证

```sh
curl http://127.0.0.1:8080/healthz
file bin/imgo
```

`bin/imgo` 应显示为 Linux x86-64 ELF。

使用超级管理员登录后台，检查角色列表中只有一个 `role_code=mentor` 的角色；首次创建时 `agent_mode=1` 且有七项权限。为测试账号分配该角色后，未保存覆盖配置的导师设置应显示两项均继承全局。也可在已登录的管理员会话中调用 `POST /manage/role/index` 和 `POST /manage/agentSetting/detail`（传入 `agent_user_id`）核对。

在解压前可执行 `sha256sum -c Imgo-linux-amd64.tar.gz.sha256` 校验发布包。包内不包含开发源码、测试库、`.env`、运行日志或上传文件；升级时仍须保留服务器原配置和附件。
