一门 APP 静态离线打包说明
========================

1. 本目录（或同级 yimen-apk.zip）即为可上传的静态资源包。
2. 在一门开发者中心选择「网页打包 / HTML 离线 / 混合打包」模式。
3. 首页文件选择：index.html
4. 上传本 zip 或把本目录全部文件上传到项目 www 根目录。
5. 包内已含 config.js（API / WebSocket）。若要改服务器，编辑本目录 config.js 后重新打 zip，或在一门后台替换该文件。
6. 资源均为相对路径（./assets ./static ./hybrid），兼容 file:// 与 fs://www/。
7. 入口 JS 已打成经典 IIFE（非 ES Module），避免 file:// CORS。
8. 路由为 hash 模式，无需服务端 rewrite。

注意：推送、保活、原生文件选择等 App 原生插件在一门 WebView 壳中可能不可用；音视频通话依赖 hybrid 页，已打入本包。
