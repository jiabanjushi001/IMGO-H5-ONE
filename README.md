# Imgo

将 Raingad-IM 5.5.2 的 PHP 后端迁移为 **Go + Gin + MySQL + WebSocket**。保留原有前端构建产物、`yu_` 表名及原业务 API 路径，路径匹配不区分大小写。

## 当前本地环境

- 数据库：本机 MySQL 的独立 `imgo` 库；没有修改其他业务库。
- 地址：`http://127.0.0.1:8088`。
- 管理员账号：`administrator`。
- 管理员随机密码：查看本地 `.env` 中的 `ADMIN_PASSWORD`。密码不在源码或文档里。
- `.env` 权限为 `600`，已被 Git 忽略。该文件包含用户提供的本地数据库连接信息。

在项目目录执行：

```sh
./start-local.sh
```

停止：终端 `Ctrl+C`。修改 Go 代码后先 `make build`，再启动。`start-local.sh` 不会自动覆盖已有二进制。

## 目录

```text
cmd/imgo/           入口、启动与数据库管理命令
internal/server/    Gin 路由、业务、数据库、鉴权、WebSocket、测试
data/              原项目 IP 数据库及说明
migrations/        原始表结构（不含演示用户和默认密码）
public/            保留的前端静态资源；不包含 PHP 运行环境
public/storage/    运行时上传文件
 .env              本机密钥、数据库、SMTP、可信代理等
go.mod / go.sum    依赖版本与校验和
start-local.sh     本地启动
Makefile           build / test / vet / race
Dockerfile         容器构建
deploy/           Nginx 和 systemd 配置示例
 docs/             接口清单、迁移说明、验证结果、原许可说明
```

## 新环境启动

需要 Go 1.23 或兼容版本、MySQL 8。不要把本地开发用的 root 账号作为生产运行账号。

1. 手动建立空数据库，为应用配置仅访问该库的账号。
2. `cp .env.example .env`，设置 `MYSQL_DSN`、`JWT_KEY`、`BASE_URL`。用 `openssl rand -hex 32` 生成 JWT 密钥。
3. `make build`，加载配置后初始化：

```sh
set -a
. ./.env
set +a
./bin/imgo -init
```

4. 在终端环境中设置 `ADMIN_ACCOUNT` 和随机 `ADMIN_PASSWORD`，执行 `./bin/imgo -create-admin`。此命令只允许在没有用户的库中运行。
5. `./start-local.sh`，访问对应地址。

## 已有 PHP 数据库迁移

**先备份并在副本演练。** 迁移命令不会删除旧用户或聊天记录，但会改变密码字段、表行格式、增加索引和 Go 附加表。

1. 把数据库副本导入测试库；保留原表前缀。
2. 配置该测试库 DSN。原 PHP 聊天加密密钥必须原样填写到 `CHAT_KEY`；旧数据没有加密时才留空。JWT 使用新随机密钥。
3. 运行 `./bin/imgo -migrate`，再启动服务。
4. 旧 MD5 密码仍可验证，正常登录后升级为 bcrypt；旧聊天密文兼容 AES-128-ECB / PKCS7 格式。新增密码使用 bcrypt。
5. 将经过检查的原 `public/storage/` 内容迁入新 `public/storage/`，保留数据库 `src` 对应的相对路径；云存储沿用原桶与配置。不要整体复制旧 `public`、`.env`、PHP、日志、SQL 或定时任务。
6. 验证两人聊天、群聊、文件、管理页和目标客户端后，再安排生产切换。

旧 PHP 不支持升级后的 bcrypt 密码。回滚应恢复切换前的数据库快照及原服务配置，并处理切换后的新增数据；不能只把反向代理切回去。

详细变化见 [迁移与兼容说明](docs/MIGRATION.md)。

## API 与实现范围

- 保留扫描出的 **108 个控制器路径**，以及前端引用的 **81 个 API 路径**；另有大小写别名和 `/index.php?s=/...` 兼容。
- 返回基本格式保留为 `{code,msg,data,count,page}`；业务错误通常 HTTP 200，`code` 表示业务状态。
- 登录、用户、好友、私聊、群聊、消息记录、已读、撤回、双向删除、转发、表情、附件、管理后台已实现。
- WebSocket 继续使用 `/wss`，支持 `init/client_id`、绑定、心跳和消息事件。
- `/avatar/...`、`/filedown/...`、`/scan/...`、`/view`、`/downapp`、`/downloadApp/...` 保留。
- 网页安装器原路径返回 `410`；部署改为本机 CLI，避免远程修改数据库配置。

本轮已补齐云存储、六类短信适配、ThinkAPI 审核、自动客服/入群、群头像、可选视频封面、拼音/IP 属地、加密消息搜索及消息/通话协议。配置见 [PROVIDERS.md](docs/PROVIDERS.md)。

**路径覆盖不等于全部场景已验收。** 真实 MySQL 与并发测试已验证核心及新增业务。第三方云、短信、邮件、内容审核、FFmpeg 视频抽帧和双设备 TURN 通话尚未真实联调；单实例约束及安全行为差异见迁移说明。

## 验证

```sh
make test
make vet
make race
```

真实 MySQL 集成测试需要 `IMGO_TEST_MYSQL_PASSWORD` 环境变量，默认 root、`127.0.0.1:3306`（可用 `IMGO_TEST_MYSQL_ADDR` 更改）。测试会创建随机命名 `imgo_test_*` 库，结束后只删除它自己创建的库，不清空已有业务库。

```sh
# 先安全设置 IMGO_TEST_MYSQL_PASSWORD，避免把密码写入命令历史。
go test -race ./... -count=1 -v
```

验证项目和结果见 [TEST_REPORT.md](docs/TEST_REPORT.md)。尚未给出承载人数或吞吐量承诺；Go 本身不能替代容量测试。

## 部署

Go 独立提供 HTTP、静态文件和 WebSocket，不需要 PHP-FPM。Nginx 可保留作 TLS 反向代理。示例在 `deploy/`；所有 `/storage/` 和 `/filedown/` 请求必须经过 Go 鉴权，不能被 Nginx 静态目录绕过。

当前 WebSocket Hub、验证码、邀请和限流状态驻留单进程；会话存于 MySQL。支持一个进程内的并发连接，**不支持多个实例直接负载均衡**。横向扩容前需要共享事件总线和共享短期状态。

前端是原有编译文件，并非重新生成的 Vue 源码。保留的原许可说明见 `docs/ORIGINAL_LICENSE.txt`；原 PHP 后门及运行目录未复制。

## 定时清理

管理员进入后台首页 → Go 定时清理，填写执行间隔（分钟）和消息保留天数，再点“保存并开启”。默认未开启；保存设置本身不启用任务。任务隐藏超期消息并清理过期会话，保留系统公告及实体附件。配置和最近 20 次日志存于数据库，服务重启后仍生效。前端面板源码与可重复构建脚本见 `frontend/README.md`。

### 邀请注册链接

`BASE_URL` 为服务端资源地址。单独设置 `INVITE_URL` 可指定注册页面，后端仅追加邀请码 token，例如：

群二维码默认编码 `BASE_URL/scan/g/<签名令牌>`；如需手机从内网访问本机预览，可将 `QR_BASE_URL` 设为预览代理根地址（例如 `http://192.168.1.123:8765`），不影响其他 API 和头像的 `BASE_URL`。浏览器直接打开群二维码时会跳转到 `H5_URL` 的群信息页并保留令牌；预览代理会把此跳转改为自己的内网域名。`H5_URL` 应填写 H5 站点根地址（不含 `#` 路由），例如本地 `http://127.0.0.1:8765/`。未设置时尝试使用 `INVITE_URL` 的站点地址。内网 HTTP 不支持浏览器实时摄像头扫码，H5 可拍照或选图识别；实时扫码需要 HTTPS。

```env
INVITE_URL='http://127.0.0.1:8765/h5/#/pages/login/register?inviteCode='
```

修改后重启 Go 服务并重新生成邀请链接。留空沿用 PC 注册页。

## 程序更新与自动补表

更新二进制并重新启动时，程序自动补齐缺失的 Go 附加表及已定义的附加字段，并为缺少邀请码的旧用户补齐邀请码。重复启动不会覆盖已有用户、余额、上下级关系或邀请码。数据库账号需具备对应数据库的 CREATE 权限，以及补字段时的 ALTER 权限。升级失败会记录具体数据库错误并停止启动。

新空库仍先执行 `./start.sh -init`；从原 PHP 数据库迁移仍使用 `-migrate`。普通启动不执行 PHP 历史数据转换或重建业务表。
