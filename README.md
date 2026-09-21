# IMGO H5

基于 uni-app（Vue 3）的即时通讯前端。正式交付**不走 uni-app 云打包，也不做 App 原生壳**。前端打成静态站点，再交给一门 APP「网页打包 / HTML 离线」打进 APK。

后续改动必须继续满足下面的约束。一门离线包跑在 `file://` 或 `fs://www/` 下，普通 H5 的写法在这里会直接白屏或 CORS。

## 核心需求：一门 APP 静态离线打包

HTML、JS、CSS、图片、`static`、`hybrid` 全部作为静态文件打进 APK，打开即可用，前端资源不依赖网站托管。

必须同时满足：

1. **相对路径。** 路由 `base` 为 `./`。`assets`、`static`、`hybrid` 以及页面里的本地资源都按相对路径解析（`utils/asset-url.js` 的 `assetUrl`，入口里的 `window.__IMGO_BASE__`）。禁止写死站点根路径 `/assets/...`。
2. **Hash 路由。** `manifest.json` 的 H5 `router.mode` 保持 `hash`。一门 WebView 没有服务端 rewrite，history 模式会 404。
3. **入口是经典脚本，不是 ES Module。** `file://` 加载 `type="module"` 会报 CORS。`npm run release:apk` 会把入口打成单个 IIFE（`scripts/bundle-apk-classic.mjs`），并设置 `window.__IMGO_CLASSIC_SCRIPT__`。
4. **不要给脚本加 `crossorigin`，也不要对分包 JS 做 `modulepreload`。** 页面 JS 已内联进 IIFE；离线包只保留 CSS 的 stylesheet 注入。
5. **聊天编辑器使用完整 Quill UMD。** 经典包先加载 `./assets/quill.min.js`，再启动入口。ESM 动态 `import('quill')` 在 `file://` 下会被掏空，输入框无法输入、无法发送。
6. **包内自带 `config.js`。** APK 里不能像网站那样单独挂配置。服务器地址写在项目根目录 `config.js`，打包时复制进离线包。

上传一门时：

1. 执行 `npm run release:apk`。
2. 产物是 `release/yimen-apk/` 和同级 `release/yimen-apk.zip`。
3. 一门开发者中心选择「网页打包 / HTML 离线 / 混合打包」。
4. 首页文件选 `index.html`。
5. 上传 zip，或把该目录全部文件放到项目 `www` 根目录。
6. 改服务器：改根目录 `config.js` 后重新打包；或只替换包内 `config.js` 再重新压缩。

## 两种发布形态

| 命令 | 产物 | `config.js` | 适用 |
| --- | --- | --- | --- |
| `npm run release:apk` | `release/yimen-apk/`、`release/yimen-apk.zip` | 打进包内 | 一门静态离线 APK |
| `npm run release:h5` | `release/h5/` | **不包含**，由网站目录单独放置 | Nginx 等 HTTP 部署 |

两种产物共用同一套相对路径和 hash 路由，可以挂在站点根目录或子目录。H5 站点包仍使用 ES Module；只有一门包改成 IIFE。

本地调试：`npm run dev:h5`。仅编译、不整理发布目录：`npm run build:h5`。

## 服务器配置

根目录 `config.js` 提供 `window.apiServers`。每一项的 HTTP API 与 WebSocket 必须是同一套服务，可按顺序写主站和备用站。

```js
window.apiServers = [
  { httpUrl: 'https://example.com', wsUrl: 'wss://example.com/wss' },
]
```

启动顺序：

1. 同步加载 `config.js`，必须早于主程序。
2. 配置缺失或地址无效时，停在「配置文件缺失或无效」，不发接口、不加载主包。
3. 探测 `GET {httpUrl}/common/pub/getSystemInfo`，只有 `code === 0` 才启动。
4. 多站点时，当前站失败才切到已验证可用的备用站，并写入 `sessionStorage`。

正式包不要写 `127.0.0.1`、`localhost`、`::1`。探测失败后若误切到本机，WebSocket 会一直连不上。

## 离线包里仍必须可用的能力

- 登录后的会话、联系人、「我的」能打开。
- 聊天能输入、能发送；发出的图片在当前窗口显示。
- 用户头像、群头像、「我的」头像能显示。需要登录才能访问的媒体（`img` / `audio` / `video` / 下载）不能裸链直出，走带鉴权的请求再转成可显示地址（`AuthImage`、`utils/avatar.js`）。`<img>` 等标签不会自动带鉴权头，直接当图片地址会 401。
- 从会话返回消息列表时，保留进入前的滚动位置，不要回到列表顶部。
- 从消息页进入会话后，左上角返回可点；通信重连后输入框仍可点击、可输入。

界面主色为蓝紫渐变 `#4c63e9` → `#5e70f5`。账号安全、通用设置、个人资料、银行卡等页与整体保持同一套颜色。

## 已知限制

一门是 WebView 壳，下面这些能力不要按 uni-app 原生插件来依赖：

- 系统推送、后台保活、原生文件选择器，在壳里可能不可用。
- 音视频通话依赖已打进包内的 `hybrid` 页面。
- `file://` 的页面 Origin 是 `null`。若服务端校验 WebSocket 的 Origin，握手会 403，表现为连上后没有业务数据。需要服务端放行 `Origin: null`（或一门实际发出的 Origin），前端无法伪造浏览器 Origin。部分低版本 Android WebView 对 `wss` 的支持也不稳定，需要在真机上确认。
