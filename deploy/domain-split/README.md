# 宝塔：API 与后台域名分离

- `https://imh5.myad.top`：原 H5 静态站点，不修改。
- `https://imgo.myad.top`：API、WebSocket、头像、文件和二维码入口。根路径和 PC index.html 返回 404。
- `https://imadmin.myad.top`：PC／后台站点；与 API 使用同一个 Go 进程和数据库。

这些配置是 server 内的片段，不是完整站点文件。保留宝塔生成的证书、443 监听和 ACME 校验。不要整份替换站点配置。

## 安装

1. 备份两个站点的 Nginx 配置。
2. 将本目录 `proxy.inc` 上传到 `/www/server/panel/vhost/nginx/imgo-snippets/proxy.inc`（先创建目录）。如 Go 监听端口不是 8088，在该文件中修改。
3. 宝塔打开 `imgo.myad.top` 配置，移除原先代理至 Go 的 `location /`（如果来自宝塔反向代理 include，则先移除对应 include），将 `api.inc` 内容放在该域名的 HTTPS `server {}` 内。
4. 打开 `imadmin.myad.top` 配置，以同样方式使用 `admin.inc` 内容。域名 DNS 和 TLS 证书须已配置。
5. 若 server 内已有 `client_max_body_size`，修改已有值即可，不重复添加。移除这两个代理站点原来的 PHP 配置、SPA rewrite 和缓存静态文件的 location，保留 ACME 校验。媒体请求必须经过 Go 权限验证。
6. 如果 HTTP 站点没有强制跳转 HTTPS，也需添加相同入口限制；推荐沿用宝塔强制 HTTPS。
7. 检查通过后重载：

```sh
/www/server/nginx/sbin/nginx -t && /www/server/nginx/sbin/nginx -s reload
```

## Go 配置

保留现有数据库、JWT_KEY 等配置，只核对下面项目：

```sh
IMGO_ADDR=127.0.0.1:8088
BASE_URL=https://imgo.myad.top
TRUSTED_PROXIES=127.0.0.1,::1
```

Go 默认只信任同机 Nginx，并从 Nginx 设置的 `X-Forwarded-For` 读取客户端 IP。旧版本如果把该项留空，会把所有登录 IP 记录成 `127.0.0.1`。

HTTP 跨域采用 *，仅支持显式 Authorization Token，不开放 Cookie 凭证跨域；不读取 ALLOWED_ORIGINS，见 docs/CORS.md。H5 apiUrl 使用 `https://imgo.myad.top`，WebSocket 使用 `wss://imgo.myad.top/wss`。后台页面的管理请求可通过自己的域名转发；不要在 API 域名直接禁止 `/manage/`，以免现有前端显式配置 API 域名时中断管理功能。修改 Go 环境配置后用原进程管理方式重启。

## 验证

```sh
curl -I https://imgo.myad.top/                 # 404
curl -I https://imgo.myad.top/index.html       # 404
curl -I https://imgo.myad.top/index.html/      # 404
curl -I https://imadmin.myad.top/              # 200
```

浏览器分别验证 H5 登录、收发消息、图片访问和后台登录。URL 中的 `#/manage/...` 是浏览器路由，不会发送给 Nginx，限制 index.html 即关闭 API 域名的 PC 页面入口。

本方案分离网页入口，不替代管理员身份验证，也不屏蔽 Go 所有静态资源。服务器 8088 仅监听本地，避免外部绕过 Nginx 直接访问网页。
