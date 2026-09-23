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

前台运行：`./start.sh`。守护运行：`./run.sh start`。可用 `status`、`restart`、`stop`、`logs` 管理进程。不要同时使用 `run.sh`、systemd 和宝塔进程管理器启动多个实例。

使用 Nginx 反向代理 `127.0.0.1:8088`，并为 `/wss` 配置 WebSocket Upgrade。API 域名和后台域名分离的示例位于 `deploy/domain-split/`。

## 更新已有部署

保留服务器原来的 `.env` 与 `public/storage`。停止旧进程后更新 `bin/imgo`、`public`、`start.sh` 和 `run.sh`，再按原来的进程管理方式启动。不要用 `.env.example` 覆盖生产配置。

## 验证

```sh
curl http://127.0.0.1:8088/healthz
file bin/imgo
```

`bin/imgo` 应显示为 Linux x86-64 ELF。
