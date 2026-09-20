# 一门 App 内置 H5

该分支保留普通网页 H5 的 `config.js` 外置方式，并为一门 App 提供独立静态目录。内置包的 API/WS 配置来自项目根目录 `config.yimen.js`；不会使用普通网页的本地 `config.js` 或旧的 `imh5.myad.top` 默认值。

1. 在 HBuilderX 重新发行 H5（目标目录 `unpackage/dist/build/h5`）。本分支的 `manifest.json` 使用 hash 路由和 `./` 资源基路径。准备脚本会拒绝早于本分支源码的旧编译产物。
2. 检查 `config.yimen.js` 中的 API 和 WebSocket 地址。APK 内的 `127.0.0.1` 指手机自身，不是开发电脑。
3. 执行 `node scripts/prepare-yimen-h5.mjs`，会生成 `release/yimen-h5`。若目录已存在，脚本会拒绝覆盖；可指定新的输出目录：`node scripts/prepare-yimen-h5.mjs <H5编译目录> <新输出目录> <配置文件>`。
4. 在一门 App 的 HTML/静态资源模式上传输出目录**内部的全部文件**，确保 `index.html`、`config.js`、`jsbridge-mini.js`、`assets/`、`static/` 同级。不要只上传 `index.html`，也不必再手动移动 `assets` 到 `h5` 子目录。

本步骤不生成 APK。修改内置配置或代码后，需要重新准备静态目录并重新生成 APK；与网页部署时刷新读取外部 `config.js` 不同。

一门 App 中的“扫一扫”调用官方 JS Bridge `scan({needResult:true})`，把结果交由现有二维码接口核验。浏览器中仍使用 Web 摄像头或选图识别。邀请链接在内置包里仅使用 API 返回的有效 HTTP(S) 注册链接；若 API 没有提供可用公网链接，只能复制邀请码，避免发出 `fs://`/`file://` 或 `127.0.0.1` 链接。

需要在真实 Android APK 中核验本地资源加载、API/WS 请求、登录保持、拍照/相册权限、原生扫码与群二维码入群。部分 WebView 以 `file://`/自定义 `fs://` 加载资源时会产生跨源限制；如果 API 请求被拦截，需要在 Go 服务端按实际 `Origin` 配置 CORS，或在一门 App 平台选择其支持的本地资源映射方式。不要把 API 改为 `127.0.0.1` 来绕过此问题。
