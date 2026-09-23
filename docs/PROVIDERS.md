# 扩展配置

业务接口地址不变。通过初始管理员的原管理后台设置 `fileUpload`、`chatInfo`、`sysInfo` 等 JSON 配置。供应商密钥放在数据库/环境变量中，不放进前端代码；下列值都是占位示例。

## 文件存储

`fileUpload.disk` 可选 `local`、`aliyun`、`qcloud`、`qiniu`。沿用原对应字段：

- aliyun：`bucket`、`endpoint`（如 `oss-cn-hangzhou.aliyuncs.com`）、`accessId`、`accessSecret`；可填 `region`。
- qcloud：`bucket`、`region`（如 `ap-guangzhou`）、`secretId`、`secretKey`、`appId`。已含 appId 后缀的桶名不重复添加。
- qiniu：`bucket`、`url`（HTTPS 下载域名）、`accessKey`、`secretKey`。

新上传先落本机暂存文件，再上传云端并记录 `imgo_object`。附件 GET 经会话/参与者鉴权后跳转至有效期 5 分钟的签名链接；修改供应商密钥不会改变对象名。历史对象根据旧 `src` 与当前存储配置解析。切换旧桶前应完成对象映射迁移，不能只修改 disk 后假设旧附件仍在新桶。

目前不自动删除远端对象。云上传成功但数据库事务失败时可能留下孤立对象，可按供应商生命周期/库存工具处理；不得直接删除被消息引用的文件。

## 短信

原 `sms` 配置结构：

```json
{
  "driver": "aliyun",
  "aliyun": {
    "access_key": "YOUR_ACCESS_KEY",
    "access_secret": "YOUR_ACCESS_SECRET",
    "sign_name": "YOUR_SIGN",
    "region_id": "cn-hangzhou",
    "actions": {
      "login": {"template_id": "YOUR_LOGIN_TEMPLATE"},
      "register": {"template_id": "YOUR_REGISTER_TEMPLATE"},
      "changePassword": {"template_id": "YOUR_RESET_TEMPLATE"},
      "changeUserinfo": {"template_id": "YOUR_ACCOUNT_TEMPLATE"}
    }
  }
}
```

其他 driver 同样使用 `actions`，凭据字段沿用 PHP：

| driver | 字段 |
| --- | --- |
| qcloud | appid、appkey、sign_name（旧腾讯短信接口凭据） |
| ucloud | public_key、private_key、project_id、sign_name |
| qiniu | AccessKey、SecretKey（注意大小写） |
| upyun | token |
| huawei | url、appKey、appSecret、sender；可选 signature、statusCallback |

`/common/pub/sendCode` 使用 account 判断邮箱/大陆手机号；code 类型 1 登录、2 注册、3 改密码、4 改账号。仅供应商明确接受后保存本机验证码。真实签名、模板及额度需在供应商账户验证，本次没有发短信。

## 审核、视频和注册

- `THINKAPI_TOKEN`：原 ThinkAPI AppCode，启用昵称、签名、群公告、文本聊天审核；未配置时不调用，配置后服务故障拒绝提交。
- `FFMPEG_BIN`、`FFPROBE_BIN`：受信任二进制绝对路径。未配置沿用默认视频占位图；配置后读取时长、抽帧并生成受原视频权限约束的封面。
- `sysInfo.runMode=2` 社区模式时，沿用 `chatInfo.autoAddUser` 的 status/user_ids/welcome，以及 `autoAddGroup` 的 status/owner_uid/name/userMax。启用自动添加好友后，新注册用户与轮询分配的客服会建立双向好友关系，再由客服私聊发送欢迎语。`autoTask` 保存轮询/群序号。用户、好友关系、欢迎消息和自动规则在同一事务内提交。
- `IP_DATABASE`：读取随项目提供的旧 17mon 格式数据，原说明保存在 `data/IP_DATABASE_NOTICE.txt`。它不是实时属地服务。

## 适配参考

云端 SDK 分别采用 [阿里云 OSS Go V2](https://www.alibabacloud.com/help/en/oss/developer-reference/manual-for-go-sdk-v2/)、[七牛 Go SDK](https://developer-doc.qiniu.com/products/kodo/go)、[腾讯云 COS Go SDK](https://intl.cloud.tencent.com/document/product/436/31528?lang=en)。短信与审核字段/协议对照本次 PHP 包内驱动；旧供应商接口的账户适用性仍需真实联调。
