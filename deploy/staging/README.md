# IMGo Go 预发布部署

该目录用于独立预发布环境。预发布必须与正式环境隔离：独立 Linux 用户、目录、服务、数据库、上传附件、JWT/CHAT 密钥、域名和 TCP 端口。

## 需要提前准备

- 一台预发布 Linux x86_64 服务器；
- 独立 MySQL 8 数据库和最小权限账号，数据库名建议 `imgo_staging`；
- `api-staging`、`tcp-staging`、可选的 `app-staging` 域名；
- API 与 TCP 域名对应的受信任 TLS 证书；
- 防火墙开放 `443/tcp` 与 `19090/tcp`，不要开放 MySQL 或 `18080`；
- 不含真实隐私数据的测试账号/脱敏数据。

## 1. 安装服务文件

```sh
sudo useradd --system --home /opt/imgo-staging --shell /usr/sbin/nologin imgo-staging
sudo mkdir -p /opt/imgo-staging /etc/imgo-staging
sudo chown -R imgo-staging:imgo-staging /opt/imgo-staging
```

将 Linux 发布包解压到 `/opt/imgo-staging`。升级时必须保留 `/opt/imgo-staging/public/storage`。

```sh
sudo cp deploy/staging/.env.example /etc/imgo-staging/imgo.env
sudo chmod 600 /etc/imgo-staging/imgo.env
sudo chown root:imgo-staging /etc/imgo-staging/imgo.env
```

填写真实预发布数据库、随机密钥、域名与证书路径，然后检查：

```sh
sudo deploy/staging/check-env.sh /etc/imgo-staging/imgo.env
```

## 2. 初始化独立数据库

仅对全新空的预发布数据库执行一次：

```sh
sudo -u imgo-staging sh -c 'set -a; . /etc/imgo-staging/imgo.env; exec /opt/imgo-staging/bin/imgo -init'
sudo -u imgo-staging sh -c 'set -a; . /etc/imgo-staging/imgo.env; ADMIN_ACCOUNT=administrator ADMIN_PASSWORD="临时强密码" exec /opt/imgo-staging/bin/imgo -create-admin'
```

不要复制正式用户、聊天、银行卡、提现、实名认证或上传附件。确需使用正式结构时，只使用经过审批的脱敏快照，并保留独立数据库账号与密钥。

## 3. 启动 systemd

```sh
sudo cp deploy/staging/imgo-staging.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now imgo-staging
sudo systemctl status imgo-staging --no-pager
curl --fail http://127.0.0.1:18080/healthz
```

日志：

```sh
sudo journalctl -u imgo-staging -f
```

## 4. 配置 Nginx 与 TLS TCP

替换 `api-staging.nginx.conf` 中的示例域名和证书路径，再复制到 Nginx `http` 配置目录：

```sh
sudo cp deploy/staging/api-staging.nginx.conf /etc/nginx/conf.d/imgo-staging.conf
sudo nginx -t
sudo systemctl reload nginx
```

Go 直接在 `19090` 提供 TLS TCP。`IMGO_TCP_TLS_CERT` 与 `IMGO_TCP_TLS_KEY` 必须属于 `tcp-staging` 域名且能被 `imgo-staging` 用户读取。

## 5. 冒烟验证

```sh
deploy/staging/smoke-test.sh \
  https://api-staging.your-domain.com \
  tcp-staging.your-domain.com \
  19090
```

随后安装 `com.imgo.mobile.staging` 包，执行完整 Excel 测试清单。正式环境和正式 App 在预发布验收期间不得改动。

## 6. 更新与回滚

更新前备份预发布数据库和上传目录，保留上一版 `bin/imgo`：

```sh
sudo systemctl stop imgo-staging
sudo cp /opt/imgo-staging/bin/imgo /opt/imgo-staging/bin/imgo.rollback
# 更新 bin/imgo 与 public（保留 public/storage）
sudo systemctl start imgo-staging
```

回滚：

```sh
sudo systemctl stop imgo-staging
sudo mv /opt/imgo-staging/bin/imgo.rollback /opt/imgo-staging/bin/imgo
sudo systemctl start imgo-staging
deploy/staging/smoke-test.sh https://api-staging.your-domain.com tcp-staging.your-domain.com 19090
```

预发布通过后应使用正式环境配置重新构建，而不是把预发布 `.env`、Debug Key 或数据库直接提升到正式环境。
