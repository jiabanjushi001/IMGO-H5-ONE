package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// Uses a newly created database only. Existing schemas are never reset or modified.
func TestMySQLIntegration(t *testing.T) {
	password := os.Getenv("IMGO_TEST_MYSQL_PASSWORD")
	if password == "" {
		t.Skip("set IMGO_TEST_MYSQL_PASSWORD to run MySQL integration")
	}
	dc := mysql.NewConfig()
	dc.User = "root"
	dc.Passwd = password
	dc.Net = "tcp"
	dc.Addr = env("IMGO_TEST_MYSQL_ADDR", "127.0.0.1:3306")
	dc.Timeout = 5 * time.Second
	root, e := sql.Open("mysql", dc.FormatDSN())
	if e != nil {
		t.Fatal(e)
	}
	defer root.Close()
	schema := "imgo_test_" + randomID()[:12]
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if _, e = root.ExecContext(ctx, "CREATE DATABASE `"+schema+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); e != nil {
		t.Fatal(e)
	}
	defer func() {
		if _, err := root.Exec("DROP DATABASE `" + schema + "`"); err != nil {
			t.Errorf("test database cleanup: %v", err)
		}
	}()
	dc.DBName = schema
	cfg := Config{DSN: dc.FormatDSN(), Prefix: "yu_", JWTKey: strings.Repeat("test-key", 8), ChatKey: "legacy-chat-key", PublicDir: t.TempDir()}
	a, e := New(cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	if e = a.Migrate(ctx, true); e != nil {
		t.Fatal(e)
	}
	if e = a.Migrate(ctx, false); e != nil {
		t.Fatal("migration not repeatable", e)
	}
	if e = a.CreateAdmin(ctx, "administrator", "test-admin-password"); e != nil {
		t.Fatal(e)
	}
	if e = a.CheckSchema(ctx); e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"alice", "bob", "outsider"} {
		if _, e = a.createUser(ctx, a.db, M{"account": name, "realname": name, "password": "test-password"}, "127.0.0.1"); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = a.db.Exec("UPDATE yu_user SET password=?,salt='salt' WHERE account='outsider'", legacyPassword("test-password", "salt")); e != nil {
		t.Fatal(e)
	}
	srv := httptest.NewServer(a.Router())
	defer srv.Close()
	call := func(path, token string, p M) M {
		t.Helper()
		b, _ := json.Marshal(p)
		req, _ := http.NewRequest("POST", srv.URL+path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", token)
		}
		resp, e := http.DefaultClient.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer resp.Body.Close()
		var result M
		if e = json.NewDecoder(resp.Body).Decode(&result); e != nil {
			t.Fatal(path, e)
		}
		return result
	}
	success := func(path, token string, p M) any {
		t.Helper()
		v := call(path, token, p)
		if number(v["code"]) != 0 {
			t.Fatalf("%s failed: %s", path, js(v))
		}
		return v["data"]
	}
	login := func(account, password string) string {
		t.Helper()
		v := obj(success("/common/pub/login", "", M{"account": account, "password": password}))
		time.Sleep(320 * time.Millisecond)
		return str(v["authToken"])
	}
	admin := login("administrator", "test-admin-password")
	alice := login("alice", "test-password")
	bob := login("bob", "test-password")
	outsider := login("outsider", "test-password")

	t.Run("overview statistics and persisted online samples", func(t *testing.T) {
		if number(call("/manage/index/overview", alice, M{})["code"]) != 403 {
			t.Fatal("overview permission bypass")
		}
		if err := a.recordOnline(ctx, time.Now()); err != nil {
			t.Fatal(err)
		}
		data := obj(success("/manage/index/overview", admin, M{"online_days": 7}))
		if number(obj(data["totals"])["users"]) != 4 {
			t.Fatal(data)
		}
		if len(data["registration_month"].([]any)) != 12 || len(data["messages"].([]any)) != 30 || len(data["online"].([]any)) == 0 {
			t.Fatal("missing trends", data)
		}
		if number(call("/manage/index/overview", admin, M{"online_days": 99})["code"]) != 400 {
			t.Fatal("invalid online range accepted")
		}
	})
	t.Run("admin user quotas round trip", func(t *testing.T) {
		data := obj(success("/manage/user/add", admin, M{"account": "quota_user", "password": "test-password", "friend_limit": 10, "group_limit": 10}))
		id := number(data["user_id"])
		detail := obj(success("/manage/user/detail", admin, M{"user_id": id}))
		if number(detail["friend_limit"]) != 10 || number(detail["group_limit"]) != 10 {
			t.Fatal("creation quotas lost", detail)
		}
		success("/manage/user/edit", admin, M{"user_id": id, "friend_limit": 7, "group_limit": -1})
		detail = obj(success("/manage/user/detail", admin, M{"user_id": id}))
		if number(detail["friend_limit"]) != 7 || number(detail["group_limit"]) != -1 {
			t.Fatal("edit quotas lost", detail)
		}
		if number(call("/manage/user/add", admin, M{"account": "invalid_quota", "password": "test-password", "friend_limit": 1.5})["code"]) != 400 {
			t.Fatal("fraction accepted")
		}
	})
	var upgraded string
	if e = a.db.QueryRow("SELECT password FROM yu_user WHERE account='outsider'").Scan(&upgraded); e != nil || !strings.HasPrefix(upgraded, "$2") {
		t.Fatal("legacy password migration failed", e)
	}
	if number(call("/manage/User/index", alice, M{})["code"]) != 403 {
		t.Fatal("admin authorization bypass")
	}
	success("/manage/User/index", admin, M{})
	// Retain numeric JSON user ids and ignore spoofed fromUser.
	msg := obj(success("/enterprise/im/sendMessage", alice, M{"id": "client-message-1", "toContactId": 3, "content": "hello <script>alert(1)</script>", "type": "text", "fromUser": M{"id": 1}}))
	if number(msg["from_user"]) != 2 || strings.Contains(str(msg["content"]), "script") {
		t.Fatal("spoofed sender or unsanitized content", msg)
	}
	duplicate := obj(success("/enterprise/im/sendMessage", alice, M{"id": "client-message-1", "toContactId": 3, "content": "hello", "type": "text"}))
	if number(duplicate["msg_id"]) != number(msg["msg_id"]) {
		t.Fatal("retry duplicated message")
	}
	history := success("/enterprise/im/getMessageList", bob, M{"toContactId": 2, "is_group": 0}).([]any)
	if len(history) != 1 {
		t.Fatal("history mismatch", history)
	}
	success("/enterprise/im/getContacts", alice, M{})
	// Bob cannot delete Alice's message for both parties; outsider cannot undo it.
	if number(call("/enterprise/im/delMessage", bob, M{"id": "client-message-1"})["code"]) != 403 {
		t.Fatal("recipient deleted sender message")
	}
	if number(call("/enterprise/im/undoMessage", outsider, M{"id": "client-message-1"})["code"]) != 403 {
		t.Fatal("outsider accessed message")
	}
	group := obj(success("/enterprise/group/add", alice, M{"name": "Integration group", "user_ids": []any{3}}))
	gid := str(group["id"])
	success("/enterprise/im/sendMessage", alice, M{"id": "group-message-1", "toContactId": gid, "is_group": 1, "type": "text", "content": "hello group"})
	if number(call("/enterprise/im/sendMessage", outsider, M{"toContactId": gid, "is_group": 1, "content": "intrusion", "type": "text"})["code"]) != 403 {
		t.Fatal("outsider group send")
	}
	if number(call("/enterprise/group/changeOwner", bob, M{"id": gid, "user_id": 3})["code"]) != 403 {
		t.Fatal("member changed group owner")
	}
	success("/enterprise/group/groupInfo", bob, M{"group_id": gid})
	success("/enterprise/group/groupuserlist", alice, M{"group_id": gid})
	success("/enterprise/im/getMessageList", bob, M{"toContactId": gid, "is_group": 1})
	success("/enterprise/group/changeOwner", alice, M{"id": gid, "user_id": 3})
	// Friend request lifecycle and form compatibility.
	friend := obj(success("/enterprise/friend/add", alice, M{"user_id": 4, "remark": "hi"}))
	success("/enterprise/friend/update", outsider, M{"friend_id": friend["friend_id"], "status": 1})
	success("/enterprise/friend/index", outsider, M{})
	success("/enterprise/friend/getApplyMsg", outsider, M{})
	// Simultaneous retries with one client id must persist once.
	var wg sync.WaitGroup
	errs := make(chan string, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b := bytes.NewBufferString(`{"id":"concurrent-id","toContactId":3,"content":"concurrent","type":"text"}`)
			req, _ := http.NewRequest("POST", srv.URL+"/enterprise/im/sendMessage", b)
			req.Header.Set("Authorization", alice)
			req.Header.Set("Content-Type", "application/json")
			res, e := http.DefaultClient.Do(req)
			if e != nil {
				errs <- e.Error()
				return
			}
			defer res.Body.Close()
			var m M
			if json.NewDecoder(res.Body).Decode(&m) != nil || number(m["code"]) != 0 {
				errs <- js(m)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
	var duplicates int
	if e = a.db.QueryRow("SELECT COUNT(*) FROM yu_message WHERE id='concurrent-id'").Scan(&duplicates); e != nil || duplicates != 1 {
		t.Fatal("concurrent idempotency", duplicates, e)
	}
	// Actual multipart upload followed by authenticated attachment download.
	var uploadBody bytes.Buffer
	writer := multipart.NewWriter(&uploadBody)
	part, _ := writer.CreateFormFile("file", "sample.txt")
	_, _ = part.Write([]byte("integration attachment"))
	_ = writer.Close()
	req, _ := http.NewRequest("POST", srv.URL+"/common/upload/uploadFile", &uploadBody)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", alice)
	resp, e := http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	var uploaded M
	_ = json.NewDecoder(resp.Body).Decode(&uploaded)
	resp.Body.Close()
	if number(uploaded["code"]) != 0 {
		t.Fatal("upload", uploaded)
	}
	file := obj(uploaded["data"])
	fid := number(file["file_id"])
	success("/enterprise/im/sendMessage", alice, M{"id": "file-message-1", "toContactId": 3, "type": "file", "content": file["src"], "file_id": fid})
	for _, tc := range []struct {
		token string
		want  int
	}{{bob, 200}, {outsider, 403}, {"", 401}} {
		req, _ := http.NewRequest("GET", srv.URL+"/filedown/"+a.hashID(fid), nil)
		req.Header.Set("Authorization", tc.token)
		res, e := http.DefaultClient.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		_, _ = io.Copy(io.Discard, res.Body)
		res.Body.Close()
		if res.StatusCode != tc.want {
			t.Fatalf("download: got %d want %d", res.StatusCode, tc.want)
		}
	}
	success("/enterprise/im/forwardMessage", bob, M{"msg_id": number(obj(success("/enterprise/im/sendMessage", alice, M{"id": "forward-source", "toContactId": 3, "type": "file", "content": file["src"], "file_id": fid}))["msg_id"]), "user_ids": []any{gid}})
	// Real WebSocket binding, recipient push, and revoked HTTP session.
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/wss"
	ws, _, e := websocket.DefaultDialer.Dial(wsURL, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer ws.Close()
	_ = ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	var init M
	if e = ws.ReadJSON(&init); e != nil {
		t.Fatal(e)
	}
	success("/common/pub/bindUid", bob, M{"client_id": init["client_id"], "user_id": 1})
	success("/enterprise/im/sendMessage", alice, M{"id": "websocket-message", "toContactId": 3, "type": "text", "content": "live delivery"})
	var event M
	for {
		if e = ws.ReadJSON(&event); e != nil {
			t.Fatal(e)
		}
		if str(event["type"]) != "isOnline" && str(event["type"]) != "userIPChanged" {
			break
		}
	}
	if str(event["type"]) != "simple" || number(obj(event["data"])["toContactId"]) != 2 {
		t.Fatal("WebSocket payload", event)
	}
	success("/common/pub/logout", outsider, M{})
	if number(call("/enterprise/im/getContacts", outsider, M{})["code"]) != -1 {
		t.Fatal("logged-out token accepted")
	}

	// Additional parity checks run against real encrypted MySQL records.
	t.Run("encrypted search and mentions", func(t *testing.T) {
		success("/enterprise/im/sendMessage", alice, M{"id": "search-private", "toContactId": 3, "type": "text", "content": "Needle 私聊"})
		found := call("/enterprise/im/getMessageList", bob, M{"toContactId": 2, "keywords": "needle"})
		if number(found["code"]) != 0 || number(found["count"]) != 1 || len(found["data"].([]any)) != 1 {
			t.Fatal("encrypted search", found)
		}
		groupMsg := obj(success("/enterprise/im/sendMessage", bob, M{"id": "mention-all", "toContactId": gid, "type": "text", "content": "大家好", "at": []any{0}}))
		if number(groupMsg["role"]) != 1 || !strings.Contains(js(groupMsg["at"]), "2") || obj(groupMsg["contactInfo"])["id"] != gid {
			t.Fatal("group push metadata", groupMsg)
		}
		at := call("/enterprise/im/getMessageList", alice, M{"toContactId": gid, "is_at": 1})
		if number(at["count"]) != 1 {
			t.Fatal("mention filter", at)
		}
		success("/enterprise/im/readAtMsg", alice, M{"toContactId": gid})
		at = call("/enterprise/im/getMessageList", alice, M{"toContactId": gid, "is_at": 1})
		if number(at["count"]) != 0 {
			t.Fatal("mention not cleared", at)
		}
	})
	t.Run("group avatar", func(t *testing.T) {
		for _, tc := range []struct {
			token  string
			status int
		}{{alice, 200}, {"", 401}} {
			req, _ := http.NewRequest("GET", srv.URL+a.groupAvatarURL(number(strings.TrimPrefix(gid, "group-"))), nil)
			req.Header.Set("Authorization", tc.token)
			res, e := http.DefaultClient.Do(req)
			if e != nil {
				t.Fatal(e)
			}
			if res.StatusCode != tc.status {
				t.Fatalf("avatar status %d", res.StatusCode)
			}
			if tc.status == 200 {
				cfg, e := png.DecodeConfig(res.Body)
				if e != nil || cfg.Width != 120 || cfg.Height != 120 {
					t.Fatal("bad avatar", cfg, e)
				}
			}
			res.Body.Close()
		}
	})

	t.Run("WebRTC legacy event protocol", func(t *testing.T) {
		started := obj(success("/enterprise/im/sendToMsg", alice, M{"id": "rtc-test", "toContactId": 3, "type": 1, "event": "calling"}))
		if number(obj(started["extends"])["code"]) != 901 || number(obj(started["extends"])["status"]) != 3 {
			t.Fatal("call defaults", started)
		}
		success("/enterprise/im/sendToMsg", bob, M{"id": "rtc-test", "msg_id": started["msg_id"], "toContactId": 2, "type": 1, "event": "acceptRtc", "code": 904})
		success("/enterprise/im/sendToMsg", alice, M{"id": "rtc-test", "msg_id": started["msg_id"], "toContactId": 3, "type": 1, "event": "hangup", "code": 906, "callTime": 65})
		m, e := one(ctx, a.db, "SELECT content,extends FROM yu_message WHERE id='rtc-test'")
		if e != nil {
			t.Fatal(e)
		}
		plain, e := decryptContent(a.cfg.ChatKey, str(m["content"]))
		if e != nil || !strings.Contains(plain, "01:05") || number(obj(m["extends"])["code"]) != 906 {
			t.Fatal("hangup history", m, e)
		}
	})
	t.Run("legacy media and emoji", func(t *testing.T) {
		content, _ := encryptContent(a.cfg.ChatKey, "/storage/voice/old.mp3")
		mid, e := insert(ctx, a.db, a.t("message"), M{"id": "legacy-voice", "from_user": 2, "to_user": 3, "chat_identify": "2-3", "content": content, "type": "voice", "file_id": 0, "status": 1})
		if e != nil {
			t.Fatal(e)
		}
		if e = a.migrateLegacyMedia(ctx); e != nil {
			t.Fatal(e)
		}
		if e = a.migrateLegacyMedia(ctx); e != nil {
			t.Fatal("media migration not repeatable", e)
		}
		m, e := one(ctx, a.db, "SELECT file_id FROM yu_message WHERE msg_id=?", mid)
		if e != nil || number(m["file_id"]) == 0 {
			t.Fatal("voice not registered", m, e)
		}
		f, e := one(ctx, a.db, "SELECT * FROM yu_file WHERE file_id=?", m["file_id"])
		if e != nil {
			t.Fatal(e)
		}
		if !a.canReadFile(ctx, 3, f) || a.canReadFile(ctx, 4, f) {
			t.Fatal("legacy voice permission")
		}
		fid, e := insert(ctx, a.db, a.t("file"), M{"src": "/storage/image/test.png", "name": "test", "ext": "png", "cate": 2, "user_id": 2, "status": 1})
		if e != nil {
			t.Fatal(e)
		}
		success("/enterprise/im/sendMessage", alice, M{"id": "collect-image", "toContactId": 3, "type": "image", "content": "/storage/image/test.png", "file_id": fid})
		first := obj(success("/enterprise/emoji/add", bob, M{"file_id": fid}))
		second := obj(success("/enterprise/emoji/add", bob, M{"file_id": fid}))
		if number(first["id"]) != number(second["id"]) {
			t.Fatal("emoji duplicated")
		}
		success("/enterprise/emoji/del", bob, M{"ids": []any{first["id"]}})
		success("/enterprise/emoji/add", bob, M{"file_id": fid})
		list := success("/enterprise/emoji/index", bob, M{}).([]any)
		if len(list) != 1 {
			t.Fatal("emoji could not be restored")
		}
	})
	t.Run("ordinary administrator permissions", func(t *testing.T) {
		uid, e := a.createUser(ctx, a.db, M{"account": "moderator", "password": "moderator-password"}, "127.0.0.1")
		if e != nil {
			t.Fatal(e)
		}
		if e = update(ctx, a.db, a.t("user"), M{"role": 2}, "user_id=?", uid); e != nil {
			t.Fatal(e)
		}
		token := login("moderator", "moderator-password")
		success("/manage/user/add", token, M{"account": "managed-user", "password": "managed-password"})
		target, e := one(ctx, a.db, "SELECT user_id FROM yu_user WHERE account='managed-user'")
		if e != nil {
			t.Fatal(e)
		}
		success("/manage/user/edit", token, M{"user_id": target["user_id"], "realname": "已修改", "role": 1})
		u, e := one(ctx, a.db, "SELECT realname,role FROM yu_user WHERE user_id=?", target["user_id"])
		if e != nil || str(u["realname"]) != "已修改" || number(u["role"]) != 0 {
			t.Fatal("ordinary admin edit/escalation", u, e)
		}
		for _, path := range []string{"edit", "setStatus", "editPassword", "del", "setRole"} {
			if number(call("/manage/user/"+path, token, M{"user_id": 1, "password": "attacker-password", "status": 0, "role": 2})["code"]) != 403 {
				t.Fatal("privileged user not protected", path)
			}
		}
		success("/manage/group/index", token, M{})
		success("/manage/message/index", token, M{})
		success("/manage/user/del", token, M{"user_id": target["user_id"]})
	})
	t.Run("automatic registration rules", func(t *testing.T) {
		oldSys := a.config(ctx, "sysInfo")
		oldChat := a.config(ctx, "chatInfo")
		setConfig := func(name string, value M) {
			t.Helper()
			if e := update(ctx, a.db, a.t("config"), M{"value": js(value)}, "name=?", name); e != nil {
				t.Fatal(e)
			}
		}
		defer setConfig("sysInfo", oldSys)
		defer setConfig("chatInfo", oldChat)
		sys := a.config(ctx, "sysInfo")
		sys["runMode"] = 2
		setConfig("sysInfo", sys)
		chat := a.config(ctx, "chatInfo")
		chat["autoAddUser"] = M{"status": 1, "user_ids": []any{2, 3}, "welcome": "欢迎新成员"}
		chat["autoAddGroup"] = M{"status": 1, "owner_uid": 1, "userMax": 2, "name": "自动群"}
		setConfig("chatInfo", chat)
		var previousGroup int64
		for i, wantCS := range []int64{2, 3, 2} {
			uid, e := a.createRegisteredUser(ctx, M{"account": fmt.Sprintf("registered%d", i), "realname": "张三", "password": "registration-test"}, "127.0.0.1")
			if e != nil {
				t.Fatal(e)
			}
			u, e := one(ctx, a.db, "SELECT cs_uid,name_py FROM yu_user WHERE user_id=?", uid)
			if e != nil || number(u["cs_uid"]) != wantCS || str(u["name_py"]) != "zhangsan" {
				t.Fatal("assignment", u, e)
			}
			friends, e := one(ctx, a.db, "SELECT COUNT(*) n FROM yu_friend WHERE status=1 AND ((create_user=? AND friend_user_id=?) OR (create_user=? AND friend_user_id=?))", wantCS, uid, uid, wantCS)
			if e != nil || number(friends["n"]) != 2 {
				t.Fatal("automatic mutual friendship", friends, e)
			}
			g, e := one(ctx, a.db, "SELECT group_id FROM yu_group_user WHERE user_id=?", uid)
			if e != nil || number(g["group_id"]) == previousGroup {
				t.Fatal("group rollover", g, e)
			}
			previousGroup = number(g["group_id"])
			m, e := one(ctx, a.db, "SELECT content FROM yu_message WHERE from_user=? AND to_user=? AND is_group=0", wantCS, uid)
			if e != nil {
				t.Fatal(e)
			}
			plain, e := decryptContent(a.cfg.ChatKey, str(m["content"]))
			if e != nil || plain != "欢迎新成员" {
				t.Fatal("welcome", e)
			}
			notice, err := one(ctx, a.db, "SELECT COUNT(*) n FROM yu_message WHERE from_user=? AND is_group=1 AND type='event'", uid)
			if err != nil || number(notice["n"]) != 0 {
				t.Fatal("automatic group join should not create a chat notice", notice, err)
			}
		}
		chat["autoAddGroup"] = M{"status": 1, "owner_uid": 999999, "userMax": 2}
		setConfig("chatInfo", chat)
		if _, e := a.createRegisteredUser(ctx, M{"account": "rollback-register", "password": "registration-test"}, "127.0.0.1"); e == nil {
			t.Fatal("invalid rule accepted")
		}
		if _, e := one(ctx, a.db, "SELECT user_id FROM yu_user WHERE account='rollback-register'"); e != sql.ErrNoRows {
			t.Fatal("failed registration was not rolled back", e)
		}
	})
	t.Run("registration automation follows nearest mentor with independent overrides", func(t *testing.T) {
		oldSys, oldChat := a.config(ctx, "sysInfo"), a.config(ctx, "chatInfo")
		setConfig := func(name string, value M) {
			t.Helper()
			if err := update(ctx, a.db, a.t("config"), M{"value": js(value)}, "name=?", name); err != nil {
				t.Fatal(err)
			}
		}
		defer setConfig("sysInfo", oldSys)
		defer setConfig("chatInfo", oldChat)
		sys := a.config(ctx, "sysInfo")
		sys["runMode"] = 2
		setConfig("sysInfo", sys)
		chat := a.config(ctx, "chatInfo")
		chat["autoAddUser"] = M{"status": 1, "user_ids": []any{2}, "welcome": "全局欢迎"}
		chat["autoAddGroup"] = M{"status": 1, "owner_uid": 1, "userMax": 5, "name": "全局群"}
		setConfig("chatInfo", chat)
		roleID, err := insert(ctx, a.db, a.t("imgo_admin_role"), M{"name": "自动分配导师", "status": 1, "agent_mode": 1, "created_at": time.Now().Unix(), "updated_at": time.Now().Unix()})
		if err != nil {
			t.Fatal(err)
		}
		create := func(account string) int64 {
			t.Helper()
			id, err := a.createUser(ctx, a.db, M{"account": account, "password": "test-password"}, "127.0.0.1")
			if err != nil {
				t.Fatal(err)
			}
			return id
		}
		bind := func(child, parent int64) {
			t.Helper()
			tx, err := a.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			if err := a.bindInviter(ctx, tx, child, parent); err != nil {
				t.Fatal(err)
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
		}
		mentor := create("automation-mentor")
		if err := update(ctx, a.db, a.t("user"), M{"admin_role_id": roleID}, "user_id=?", mentor); err != nil {
			t.Fatal(err)
		}
		child := create("automation-child")
		bind(child, mentor)
		grandchild := create("automation-grandchild")
		bind(grandchild, child)
		codeRow, err := one(ctx, a.db, "SELECT invite_code FROM "+a.t("imgo_referral")+" WHERE user_id=?", grandchild)
		if err != nil {
			t.Fatal(err)
		}
		grandchildCode := str(codeRow["invite_code"])
		if number(call("/manage/agentSetting/detail", bob, M{"agent_user_id": mentor})["code"]) != 403 {
			t.Fatal("ordinary user can read mentor settings")
		}
		if number(call("/manage/agentSetting/detail", admin, M{"agent_user_id": child})["code"]) != 403 {
			t.Fatal("non mentor accepted")
		}
		if number(call("/manage/agentSetting/save", admin, M{"agent_user_id": mentor, "inherit_auto_user": false, "inherit_auto_group": true, "auto_add_user": M{"status": 1, "user_ids": []any{3}}})["code"]) != 403 {
			t.Fatal("out of scope customer accepted")
		}
		userOverride := M{"status": 1, "user_ids": []any{mentor}, "welcome": "导师欢迎"}
		groupOverride := M{"status": 1, "owner_uid": mentor, "userMax": 5, "name": "导师群"}
		for i, tc := range []struct {
			userInherited, groupInherited bool
			wantCustomer, wantOwner       int64
		}{
			{true, true, 2, 1},
			{false, true, mentor, 1},
			{true, false, 2, mentor},
			{false, false, mentor, mentor},
		} {
			config := M{"agent_user_id": mentor, "inherit_auto_user": tc.userInherited, "inherit_auto_group": tc.groupInherited, "auto_add_user": userOverride, "auto_add_group": groupOverride}
			success("/manage/agentSetting/save", admin, config)
			detail := obj(success("/manage/agentSetting/detail", admin, M{"agent_user_id": mentor}))
			if detail["inherit_auto_user"] != tc.userInherited || detail["inherit_auto_group"] != tc.groupInherited {
				t.Fatalf("case %d detail: %#v", i, detail)
			}
			uid, err := a.createRegisteredUser(ctx, M{"account": fmt.Sprintf("automation-new-%d", i), "password": "test-password", "inviteCode": grandchildCode}, "127.0.0.1")
			if err != nil {
				t.Fatal(err)
			}
			user, err := one(ctx, a.db, "SELECT cs_uid FROM "+a.t("user")+" WHERE user_id=?", uid)
			if err != nil || number(user["cs_uid"]) != tc.wantCustomer {
				t.Fatalf("case %d customer: %#v, %v", i, user, err)
			}
			group, err := one(ctx, a.db, "SELECT g.owner_id FROM "+a.t("group_user")+" gu JOIN "+a.t("group")+" g ON g.group_id=gu.group_id WHERE gu.user_id=?", uid)
			if err != nil || number(group["owner_id"]) != tc.wantOwner {
				t.Fatalf("case %d group: %#v, %v", i, group, err)
			}
		}
		success("/manage/agentSetting/save", admin, M{"agent_user_id": mentor, "inherit_auto_user": false, "inherit_auto_group": true, "auto_add_user": M{"status": 0}})
		uid, err := a.createRegisteredUser(ctx, M{"account": "automation-disabled", "password": "test-password", "inviteCode": grandchildCode}, "127.0.0.1")
		if err != nil {
			t.Fatal(err)
		}
		user, err := one(ctx, a.db, "SELECT cs_uid FROM "+a.t("user")+" WHERE user_id=?", uid)
		if err != nil || number(user["cs_uid"]) != 0 {
			t.Fatalf("explicitly disabled customer still assigned: %#v, %v", user, err)
		}
		nested := create("automation-nested-mentor")
		bind(nested, child)
		if err := update(ctx, a.db, a.t("user"), M{"admin_role_id": roleID}, "user_id=?", nested); err != nil {
			t.Fatal(err)
		}
		leaf := create("automation-nested-leaf")
		bind(leaf, nested)
		leafCode, err := one(ctx, a.db, "SELECT invite_code FROM "+a.t("imgo_referral")+" WHERE user_id=?", leaf)
		if err != nil {
			t.Fatal(err)
		}
		success("/manage/agentSetting/save", admin, M{"agent_user_id": nested, "inherit_auto_user": false, "inherit_auto_group": false, "auto_add_user": M{"status": 1, "user_ids": []any{nested}}, "auto_add_group": M{"status": 1, "owner_uid": nested, "userMax": 5, "name": "近导师群"}})
		uid, err = a.createRegisteredUser(ctx, M{"account": "automation-nested-new", "password": "test-password", "inviteCode": leafCode["invite_code"]}, "127.0.0.1")
		if err != nil {
			t.Fatal(err)
		}
		user, err = one(ctx, a.db, "SELECT cs_uid FROM "+a.t("user")+" WHERE user_id=?", uid)
		if err != nil || number(user["cs_uid"]) != nested {
			t.Fatalf("nearest mentor not selected: %#v, %v", user, err)
		}
	})

	t.Run("persistent maintenance scheduler", func(t *testing.T) {
		if number(call("/manage/task/startTask", alice, M{})["code"]) != 403 {
			t.Fatal("non-admin enabled cleanup")
		}
		if number(call("/manage/task/setTaskConfig", admin, M{"interval_minutes": 0})["code"]) != 400 {
			t.Fatal("invalid interval accepted")
		}
		if number(call("/manage/task/setTaskConfig", admin, M{"retention_days": -1})["code"]) != 400 {
			t.Fatal("invalid retention accepted")
		}
		now := time.Now()
		encrypted, _ := encryptContent(a.cfg.ChatKey, "maintenance fixture")
		fixture := func(id, key string, when time.Time) int64 {
			t.Helper()
			mid, e := insert(ctx, a.db, a.t("message"), M{"id": id, "from_user": 2, "to_user": 3, "chat_identify": key, "type": "text", "content": encrypted, "status": 1, "create_time": when.Unix()})
			if e != nil {
				t.Fatal(e)
			}
			return mid
		}
		old := fixture("cleanup-old", "2-3", now.Add(-72*time.Hour))
		recent := fixture("cleanup-recent", "2-3", now)
		notice := fixture("cleanup-notice", "admin_notice", now.Add(-72*time.Hour))
		if _, e := insert(ctx, a.db, a.t("imgo_session"), M{"sid": "expired-fixture", "user_id": 2, "expires_at": now.Add(-time.Hour).Unix()}); e != nil {
			t.Fatal(e)
		}
		state := obj(success("/manage/task/startTask", admin, M{"interval_minutes": 5, "retention_days": 2, "clear_messages": true}))
		due := number(state["next_run_at"])
		if number(state["enabled"]) != 1 || due < now.Unix()+299 {
			t.Fatal("schedule not persisted", state)
		}
		status := func(id int64) int64 {
			t.Helper()
			m, e := one(ctx, a.db, "SELECT status FROM yu_message WHERE msg_id=?", id)
			if e != nil {
				t.Fatal(e)
			}
			return number(m["status"])
		}
		if e := a.maintenanceTick(ctx, time.Unix(due-1, 0)); e != nil {
			t.Fatal(e)
		}
		if status(old) != 1 {
			t.Fatal("cleanup ran before due time")
		}
		if e := a.maintenanceTick(ctx, time.Unix(due, 0)); e != nil {
			t.Fatal(e)
		}
		if status(old) != 0 || status(recent) != 1 || status(notice) != 1 {
			t.Fatal("retention boundary or announcement protection failed")
		}
		if _, e := one(ctx, a.db, "SELECT sid FROM yu_imgo_session WHERE sid='expired-fixture'"); e != sql.ErrNoRows {
			t.Fatal("expired session not removed", e)
		}
		persisted, e := a.loadMaintenance(ctx)
		if e != nil || number(persisted["last_run_at"]) != due || number(persisted["next_run_at"]) != due+300 {
			t.Fatal("run state", persisted, e)
		}
		fresh, e := New(cfg)
		if e != nil {
			t.Fatal(e)
		}
		loaded, e := fresh.loadMaintenance(ctx)
		fresh.Close()
		if e != nil || number(loaded["enabled"]) != 1 || number(loaded["interval_minutes"]) != 5 {
			t.Fatal("restart lost settings", loaded, e)
		}
		stopped := obj(success("/manage/task/stopTask", admin, M{}))
		if number(stopped["enabled"]) != 0 || number(stopped["next_run_at"]) != 0 {
			t.Fatal("not stopped", stopped)
		}
		if e := a.maintenanceTick(ctx, now.Add(10*24*time.Hour)); e != nil {
			t.Fatal(e)
		}
		if status(recent) != 1 {
			t.Fatal("stopped job executed")
		}
		if log := str(success("/manage/task/getTaskLog", admin, M{})); !strings.Contains(log, "过期消息") {
			t.Fatal("missing persistent log", log)
		}
	})
	t.Run("invitation hierarchy and direct-only H5 summary", func(t *testing.T) {
		codeFor := func(userID int64) string {
			t.Helper()
			row, err := one(ctx, a.db, "SELECT invite_code FROM yu_imgo_referral WHERE user_id=?", userID)
			if err != nil || !validInviteCode(str(row["invite_code"])) {
				t.Fatalf("missing six-digit code for %d: %v %#v", userID, err, row)
			}
			return str(row["invite_code"])
		}
		parent := int64(2) // alice
		for _, name := range []string{"referral_b", "referral_c", "referral_d", "referral_e"} {
			child, err := a.createRegisteredUser(ctx, M{"account": name, "realname": name, "password": "test-password", "inviteCode": codeFor(parent)}, "127.0.0.1")
			if err != nil {
				t.Fatal(err)
			}
			parent = child
		}
		root, err := one(ctx, a.db, "SELECT COUNT(*) team_count,COALESCE(SUM(depth=1),0) direct_count FROM yu_imgo_referral_path WHERE ancestor_user_id=2")
		if err != nil || number(root["team_count"]) != 4 || number(root["direct_count"]) != 1 {
			t.Fatalf("wrong A→B→C→D→E tree: %#v %v", root, err)
		}
		status := obj(success("/enterprise/invite/status", alice, M{}))
		if status["invite_code"] != codeFor(2) || number(status["direct_count"]) != 1 || status["team_count"] != nil || status["earned_points"] != nil || status["surpassed_percent"] != nil {
			t.Fatalf("H5 leaked team count or lost direct count: %#v", status)
		}
		if number(status["streak_days"]) < 0 {
			t.Fatalf("invitation streak is inconsistent: %#v", status)
		}
		members := success("/manage/user/index", admin, M{"keywords": "alice"}).([]any)
		if len(members) == 0 {
			t.Fatal("admin member search returned nothing")
		}
		member := obj(members[0])
		if number(member["direct_invite_count"]) != 1 || number(member["team_count"]) != 4 || member["invite_code"] != codeFor(2) {
			t.Fatalf("admin tree summary: %#v", member)
		}
		if _, err = a.createRegisteredUser(ctx, M{"account": "referral_invalid", "password": "test-password", "inviteCode": "NOPEQ"}, "127.0.0.1"); err == nil {
			t.Fatal("unknown code was accepted")
		}
		if _, err = one(ctx, a.db, "SELECT user_id FROM yu_user WHERE account='referral_invalid'"); err != sql.ErrNoRows {
			t.Fatalf("failed referral registration was not rolled back: %v", err)
		}
	})
	if !t.Failed() {
		t.Log(fmt.Sprintf("MySQL migration, authentication, DM, groups, uploads, file authorization, WebSocket and logout passed (%s)", schema))
	}
}
