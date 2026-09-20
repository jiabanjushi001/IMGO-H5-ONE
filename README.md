

技术栈：`vue3` + `pinia` + `color-UI`

#### H5 运行时接口配置

H5 在启动前单独请求必需的 `config.js?t=当前时间戳`，从 `window.apiServers` 读取按优先级排列的 API、WebSocket 地址。文件缺失、加载失败或没有有效地址时，不启动应用，也不发送 API 请求；页面显示配置错误。配置有效时，启动阶段依次探测 API；运行中每 15 秒检测当前 API，故障时探测备用站并自动重新载入页面，使 API 和 WebSocket 同时切换。上次可用的地址会在当前浏览器会话中优先使用。所有地址均不可达时显示可重试的离线页。

```js
window.apiServers = [
  { httpUrl: 'https://api.example.com', wsUrl: 'wss://api.example.com/wss' },
  { httpUrl: 'https://backup.example.com', wsUrl: 'wss://backup.example.com/wss' },
]
```

`wsUrl` 可省略，默认从 `httpUrl` 推导为同域的 `/wss`。单地址的旧配置 `window.httpUrl` / `window.wsUrl` 仍兼容。自动切换会重新载入页面，不会自动重发失败的请求，以免重复提交。项目根目录的 `config.js` 是本地示例配置，不放在 `static`、不导入主程序，也不包含在 `release/h5` 发布包里。

部署时，把 `config.js` **单独**放到服务器上，与发布后的 `index.html` 同级，并自行填写实际可用的主、备服务器。后续只更新服务器上的这个文件，用户刷新页面即可生效，无须重新打包。本地预览目录的 `config.js` 是指向项目根目录文件的链接，因此本地也只需修改项目根目录的 `config.js`。

若要兼容已生成的旧 H5 产物，可运行 `node scripts/patch-h5-runtime-config.mjs` 更新入口和主程序；该脚本不会把 `config.js` 复制到发布包。

#### 目录结构
```
Raingad
├─api                   接口目录
│  ├─index.js           总的接口文件
│  └─message.js         消息接口
│
├─common                公共目录
│  ├─socket.js          websocket配置
│  └─config.js          服务器地址配置
│ 
│─components            符合vue组件规范的uni-app组件目录
│  └─comp-a.vue         可复用的a组件
│
├─pages                 业务页面文件存放的目录
│  ├─index
│  │  └─index.vue       index页面
│  ├─message
│  │   └─index.vue      消息列表页面
│  ├─ ...               更多目录
│  └─contact
│     └─index.vue       联系人列表页面
│
├─static                存放应用引用的本地静态资源（如图片、视频等）的目录，注意：静态资源只能存放于此
│
├─uni_modules           存放[uni_module](/uni_modules)。
│
├─hybrid                App端存放本地html文件的目录，详见
│
├─nativeplugins         远程插件目录
│
├─uniCloud              云函数目录，里面有unipush的推送功能，如果不需要可以删除（已删除）
│
├─unpackage             非工程代码，一般存放运行或发行的编译结果
│
├─main.js               Vue初始化入口文件
├─App.vue               应用配置，用来配置App全局样式以及监听 应用生命周期
├─manifest.json         配置应用名称、appid、logo、版本等打包信息，详见
├─pages.json            配置页面路由、导航条、选项卡等页面类信息，详见
└─uni.scss              这里是uni-app内置的常用样式变量
```
