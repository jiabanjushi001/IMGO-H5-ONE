# Imgo Linux x86_64 部署

此包包含 Linux amd64 静态编译程序及 PC 网页/管理后台，不包含 uniapp H5 构建产物、数据库数据、本机密钥或上传附件。无需安装 Go 或 PHP，需要 MySQL 8。

## 配置和启动

```sh
cp .env.example .env
chmod 600 .env
chmod +x start.sh run.sh bin/imgo
```

编辑 `.env` 后，先创建空数据库。为 `ADMIN_PASSWORD` 设置独立的强密码（6–30 字节）。`ADMIN_ACCOUNT` 省略时默认为 `administrator`。全新空库执行以下命令，一次完成建表和超级管理员创建。

```sh
./start.sh -init
```

如果已用旧版完成建表，且数据库还没有用户，在 `.env` 中设置 `ADMIN_ACCOUNT` 和 `ADMIN_PASSWORD` 后执行 `./start.sh -create-admin`。

已有数据库直接启动。程序会自动补齐 Go 扩展表、角色权限表和兼容字段；涉及旧数据重写的迁移仍按 `docs/MIGRATION.md` 执行。

启动使用的数据库账号除日常读写权限外，必须具有 `CREATE`、`ALTER`、`INSERT` 权限，以完成自动升级。启动日志出现 `automatic database upgrade failed` 时，先修复数据库权限或报错原因，再重新启动。

前台运行：`./start.sh`。守护运行：`./run.sh start`。可用 `status`、`restart`、`stop`、`logs` 管理进程。不要同时使用 `run.sh`、systemd 和宝塔进程管理器启动多个实例。

默认使用 Nginx 反向代理 `127.0.0.1:8080`，并为 `/wss` 配置 WebSocket Upgrade。如果修改了 `.env` 中的 `IMGO_ADDR`，Nginx 的 `proxy_pass` 也要使用同一端口。API 域名和后台域名分离的示例位于 `deploy/domain-split/`。

原生 Android TCP 是新增的可选通道。设置 `IMGO_TCP_ADDR=0.0.0.0:9090` 后，Go 进程会在保留全部 HTTP/WebSocket 功能的同时监听 TCP；留空则完全不启动 TCP。公网环境同时设置 `IMGO_TCP_TLS_CERT` 和 `IMGO_TCP_TLS_KEY`，并在防火墙中仅开放实际使用的 TCP 端口。TLS 两项必须同时配置。

## H5 和后台直传阿里云 OSS

超级管理员在“系统设置 → 文件上传设置”选择“阿里云”，填写 OSS 区域 `endpoint` 和已经绑定当前 Bucket 的 ESA/CNAME HTTPS 域名，再开启“客户端直传 OSS”。新版 H5 与管理后台会把图片、视频、语音和文件直接上传到 ESA，A 反向代理和 B Go 服务只处理上传凭证申请、文件确认等很小的 JSON 请求，文件内容不再经过 A 或 B。关闭开关后，新版客户端自动退回原上传接口；未升级的旧 H5 继续使用原接口，但 B 服务向 OSS 上传、校验和生成签名下载地址时同样会走配置的 ESA 域名。

在阿里云 OSS Bucket 的跨域设置中，为实际使用的 H5 和后台域名添加规则：

- 来源（Origins）：填写完整的 H5、管理后台域名，生产环境建议不要使用 `*`。
- 允许方法（Methods）：`POST`、`GET`、`HEAD`。
- 允许请求头（Allowed Headers）：`*`。
- 暴露响应头（Expose Headers）：可填写 `ETag`。
- 缓存时间（Max Age）：建议 `600` 秒或以上。

直传使用服务端签发的短期 POST Policy，不会把 OSS `AccessKeySecret` 下发给浏览器。ESA/CNAME 域名必须使用 HTTPS，不能填写 `*.aliyuncs.com` 公共 endpoint，也不能带路径、端口或查询参数。修改 ESA/OSS 跨域规则后应同时用 H5 和管理后台各上传一张图片、一个视频和一个普通文件；如果浏览器控制台显示 CORS 错误，先核对当前页面的协议、域名和端口是否与 Origins 完全一致，并确认允许 `POST`、`GET`、`HEAD` 和 `Content-Type` 请求头。

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
