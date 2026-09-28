package server

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAdminAuditSensitiveClassification(t *testing.T) {
	tests := map[string]struct{ category, risk string }{
		"/manage/user/add":            {"成员", "medium"},
		"/manage/user/editpassword":   {"成员", "high"},
		"/manage/user/googleauthbind": {"成员", "high"},
		"/manage/wallet/recharge":     {"财务", "critical"},
		"/manage/wallet/withdraw":     {"财务", "critical"},
		"/manage/config/setsecurity":  {"设置", "critical"},
	}
	for path, want := range tests {
		definition, ok := adminAuditDefinitions[path]
		if !ok || definition.Category != want.category || definition.Risk != want.risk {
			t.Fatalf("audit definition %s = %#v, want category=%s risk=%s", path, definition, want.category, want.risk)
		}
	}
}

func TestWriteAdminAuditStoresSnapshotWithoutSecrets(t *testing.T) {
	a, mock := testApp(t)
	r := bankRequest(a, "/manage/user/editPassword", 1, M{"user_id": 7, "password": "plain-secret", "google_code": "123456", "remark": "visible"})
	r.user = M{"user_id": 1, "account": "administrator", "realname": "管理员"}
	r.c.Request.RemoteAddr = "203.0.113.8:1234"
	r.c.Request.Header.Set("Authorization", "bearer secret-token")
	r.c.Request.Header.Set("Cookie", "session=secret")
	r.c.Request.Header.Set("X-Trace-Id", "trace-123")
	r.c.Request.Header.Set("User-Agent", "test-agent")

	method, path, headers, data := auditRequestSnapshot(r)
	if method != "POST" || path != "/manage/user/editpassword" {
		t.Fatalf("unexpected request identity: %s %s", method, path)
	}
	if !strings.Contains(headers, "trace-123") || strings.Contains(headers, "secret-token") || strings.Contains(headers, "session=secret") {
		t.Fatalf("request headers were not redacted correctly: %s", headers)
	}
	if !strings.Contains(data, "visible") || !strings.Contains(data, "[已隐藏]") || strings.Contains(data, "plain-secret") || strings.Contains(data, "123456") {
		t.Fatalf("request data was not redacted correctly: %s", data)
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `yu_imgo_admin_audit_log` (actor_user_id,actor_account,actor_name,actor_role,category,action,risk_level,target_type,target_id,target_name,detail,ip,user_agent,request_method,request_path,request_headers,request_data,status,error_message,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")).
		WithArgs(int64(1), "administrator", "管理员", "超级管理员", "成员", "修改密码", "high", "用户", "7", "wuhu1", "", "203.0.113.8", "test-agent", "POST", "/manage/user/editpassword", sqlmock.AnyArg(), sqlmock.AnyArg(), 1, "", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	a.writeAdminAudit(context.Background(), r, r.user, auditDefinition{"成员", "修改密码", "high", "用户"}, "7", "wuhu1", "", true, "")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAuditDetailDescribesSecurityChanges(t *testing.T) {
	a, _ := testApp(t)
	r := bankRequest(a, "/manage/config/setSecurity", 1, M{"google_auth_enabled": 1})
	if got := a.auditDetail(auditDefinition{}, r, M{"google_auth_enabled": true}); got != "开启全局谷歌验证" {
		t.Fatalf("unexpected google detail: %s", got)
	}
	r = bankRequest(a, "/manage/config/setSecurity", 1, M{"ip_whitelist_enabled": 1, "ip_whitelist": "203.0.113.1,203.0.113.2"})
	r.auditBefore = M{"ip_whitelist_enabled": true, "ip_whitelist": []string{"203.0.113.1", "198.51.100.8"}}
	got := a.auditDetail(auditDefinition{}, r, M{"ip_whitelist_enabled": true, "ip_whitelist": []string{"203.0.113.1", "203.0.113.2"}})
	if got != "更新后台 IP 白名单；新增：203.0.113.2；删除：198.51.100.8；当前允许：203.0.113.1、203.0.113.2" {
		t.Fatalf("unexpected whitelist detail: %s", got)
	}
}
