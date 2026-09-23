# Token 跨域

HTTP 直接返回 Access-Control-Allow-Origin: *，不读取配置、不设域名白名单，不设置 Access-Control-Allow-Credentials。API 继续验证 Authorization: Bearer；前端请求必须 withCredentials: false。支持 OPTIONS 和 Authorization、Content-Type、Token、session 等头部。

WebSocket 不校验 Origin：任意值（包括 null、file://）或不传均允许握手。连接绑定仍须提供有效 Token；业务权限校验保持不变。

注意：本模式不支持跨域 Cookie 请求。当前图片 Canvas 中 use-credentials 的加载方式不能用于此模式；需改为携带 Authorization 请求图片 Blob，再交给 Canvas，或改用同源代理。本次未改动 H5 Canvas。

替换程序后重启；Nginx 不要重复添加跨域响应头。
