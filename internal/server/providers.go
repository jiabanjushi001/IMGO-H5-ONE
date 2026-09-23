package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/qiniu/go-sdk/v7/auth"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

func percent(s string) string { return strings.ReplaceAll(url.QueryEscape(s), "+", "%20") }
func aliyunRPCSignature(method string, v url.Values, secret string) string {
	canonical := strings.ReplaceAll(v.Encode(), "+", "%20")
	mac := hmac.New(sha1.New, []byte(secret+"&"))
	mac.Write([]byte(method + "&%2F&" + percent(canonical)))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
func ucloudSignature(v url.Values, secret string) string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString(v.Get(k))
	}
	b.WriteString(secret)
	sum := sha1.Sum([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}
func smsRequest(ctx context.Context, driver string, c M, action, phone, code string, now time.Time, nonce string) (*http.Request, error) {
	template := str(obj(obj(c["actions"])[action])["template_id"])
	if template == "" {
		return nil, clientError{"短信模板未配置：" + action, 503}
	}
	method := http.MethodPost
	endpoint := ""
	body := ""
	headers := http.Header{}
	headers.Set("Content-Type", "application/x-www-form-urlencoded")
	require := func(keys ...string) error {
		for _, k := range keys {
			if str(c[k]) == "" {
				return clientError{"短信缺少配置：" + driver + "." + k, 503}
			}
		}
		return nil
	}
	form := url.Values{}
	switch driver {
	case "aliyun":
		if e := require("access_key", "access_secret", "sign_name"); e != nil {
			return nil, e
		}
		endpoint = "https://dysmsapi.aliyuncs.com/"
		region := str(c["region_id"])
		if region == "" {
			region = "cn-hangzhou"
		}
		form = url.Values{"Action": {"SendSms"}, "Version": {"2017-05-25"}, "Format": {"JSON"}, "AccessKeyId": {str(c["access_key"])}, "SignatureMethod": {"HMAC-SHA1"}, "SignatureVersion": {"1.0"}, "SignatureNonce": {nonce}, "Timestamp": {now.UTC().Format("2006-01-02T15:04:05Z")}, "RegionId": {region}, "PhoneNumbers": {phone}, "SignName": {str(c["sign_name"])}, "TemplateCode": {template}, "TemplateParam": {js(M{"code": code})}}
		form.Set("Signature", aliyunRPCSignature(method, form, str(c["access_secret"])))
		body = form.Encode()
	case "qcloud":
		if e := require("appid", "appkey", "sign_name"); e != nil {
			return nil, e
		}
		random := fmt.Sprint(now.UnixNano() & 0x7fffffff)
		timestamp := fmt.Sprint(now.Unix())
		sum := sha256.Sum256([]byte("appkey=" + str(c["appkey"]) + "&random=" + random + "&time=" + timestamp + "&mobile=" + phone))
		endpoint = "https://yun.tim.qq.com/v5/tlssmssvr/sendsms?sdkappid=" + url.QueryEscape(str(c["appid"])) + "&random=" + random
		body = js(M{"tel": M{"nationcode": "86", "mobile": phone}, "sig": hex.EncodeToString(sum[:]), "tpl_id": number(template), "params": []string{code}, "sign": str(c["sign_name"]), "time": now.Unix(), "extend": "", "ext": ""})
		headers.Set("Content-Type", "application/json")
	case "ucloud":
		if e := require("public_key", "private_key", "project_id", "sign_name"); e != nil {
			return nil, e
		}
		method = http.MethodGet
		form = url.Values{"Action": {"SendUSMSMessage"}, "PublicKey": {str(c["public_key"])}, "ProjectId": {str(c["project_id"])}, "PhoneNumbers.0": {phone}, "SigContent": {str(c["sign_name"])}, "TemplateId": {template}, "TemplateParams.0": {code}}
		form.Set("Signature", ucloudSignature(form, str(c["private_key"])))
		endpoint = "https://api.ucloud.cn/?" + form.Encode()
	case "qiniu":
		if e := require("AccessKey", "SecretKey"); e != nil {
			return nil, e
		}
		endpoint = "https://sms.qiniuapi.com/v1/message"
		body = js(M{"template_id": template, "mobiles": []string{phone}, "parameters": M{"code": code}})
		headers.Set("Content-Type", "application/json")
	case "upyun":
		if e := require("token"); e != nil {
			return nil, e
		}
		endpoint = "https://sms-api.upyun.com/api/messages"
		body = url.Values{"mobile": {phone}, "template_id": {template}, "vars[code]": {code}}.Encode()
		headers.Set("Authorization", str(c["token"]))
	case "huawei":
		if e := require("url", "appKey", "appSecret", "sender"); e != nil {
			return nil, e
		}
		u, e := url.Parse(str(c["url"]))
		if e != nil || u.Scheme != "https" || !strings.HasSuffix(u.Hostname(), ".myhuaweicloud.com") {
			return nil, clientError{"华为短信 URL 必须为 HTTPS 官方域名", 400}
		}
		endpoint = strings.TrimRight(str(c["url"]), "/") + "/sms/batchSendSms/v1"
		created := now.UTC().Format("2006-01-02T15:04:05Z")
		sum := sha256.Sum256([]byte(nonce + created + str(c["appSecret"])))
		digest := base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(sum[:])))
		headers.Set("Authorization", `WSSE realm="SDP",profile="UsernameToken",type="Appkey"`)
		headers.Set("X-WSSE", fmt.Sprintf(`UsernameToken Username=%q,PasswordDigest=%q,Nonce=%q,Created=%q`, str(c["appKey"]), digest, nonce, created))
		body = url.Values{"from": {str(c["sender"])}, "to": {phone}, "templateId": {template}, "templateParas": {js([]string{code})}, "signature": {str(c["signature"])}, "statusCallback": {str(c["statusCallback"])}}.Encode()
	default:
		return nil, clientError{"短信 driver 无效", 400}
	}
	req, e := http.NewRequestWithContext(ctx, method, endpoint, strings.NewReader(body))
	if e != nil {
		return nil, e
	}
	req.Header = headers
	if driver == "qiniu" {
		token, e := auth.New(str(c["AccessKey"]), str(c["SecretKey"])).SignRequestV2(req)
		if e != nil {
			return nil, e
		}
		req.Header.Set("Authorization", "Qiniu "+token)
	}
	return req, nil
}
func (a *App) providerCall(req *http.Request) (M, error) {
	client := a.outbound
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	res, e := client.Do(req)
	if e != nil {
		return nil, clientError{"外部服务连接失败，请稍后重试", 503}
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, clientError{"外部服务拒绝请求，请检查配置", 502}
	}
	var data M
	if e = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&data); e != nil {
		return nil, clientError{"外部服务返回格式错误", 502}
	}
	return data, nil
}
func (a *App) sendSMS(ctx context.Context, phone, kind, code string) error {
	if !regexp.MustCompile(`^1[3-9][0-9]{9}$`).MatchString(phone) {
		return clientError{"手机号格式错误", 400}
	}
	cfg := a.config(ctx, "sms")
	driver := str(cfg["driver"])
	action := map[string]string{"1": "login", "2": "register", "3": "changePassword", "4": "changeUserinfo"}[kind]
	req, e := smsRequest(ctx, driver, obj(cfg[driver]), action, phone, code, time.Now(), randomID())
	if e != nil {
		return e
	}
	data, e := a.providerCall(req)
	if e != nil {
		return e
	}
	ok := false
	switch driver {
	case "aliyun":
		ok = str(data["Code"]) == "OK"
	case "qcloud":
		v, exists := data["result"]
		ok = exists && number(v) == 0
	case "ucloud":
		v, exists := data["RetCode"]
		ok = exists && number(v) == 0
	case "qiniu":
		ok = str(data["job_id"]) != ""
	case "upyun":
		list, exists := data["message_ids"].([]any)
		ok = exists && len(list) > 0 && str(obj(list[0])["message_id"]) != ""
	case "huawei":
		ok = str(data["code"]) == "000000"
	}
	if !ok {
		return clientError{"短信发送失败，请检查供应商返回状态和模板配置", 502}
	}
	return nil
}
func (a *App) moderate(ctx context.Context, content, service string) error {
	token := a.cfg.ModerationToken
	if token == "" {
		return nil
	}
	plain := []rune(regexp.MustCompile(`<[^>]*>`).ReplaceAllString(content, ""))
	if len(plain) > 500 {
		plain = plain[:500]
	}
	form := url.Values{"service": {service}, "content": {string(plain)}}
	req, e := http.NewRequestWithContext(ctx, "POST", "https://api.topthink.com/green/text", strings.NewReader(form.Encode()))
	if e != nil {
		return e
	}
	req.Header.Set("Authorization", "AppCode "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	data, e := a.providerCall(req)
	if e != nil {
		return e
	}
	v, exists := data["code"]
	if !exists || number(v) != 0 {
		return clientError{"内容审核服务暂不可用", 503}
	}
	if str(obj(data["data"])["labels"]) != "" {
		return clientError{"内容未通过审核，请修改后重试", 400}
	}
	return nil
}
