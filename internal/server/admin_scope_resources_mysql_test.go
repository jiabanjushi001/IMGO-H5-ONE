package server

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

// Called by TestMySQLIntegration, whose fixture owns a fresh disposable database.
func testAgentScopeResourcesMySQL(t *testing.T, a *App) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	create := func(name string) int64 {
		t.Helper()
		id, err := a.createUser(ctx, a.db, M{"account": name, "password": "test-password"}, "127.0.0.1")
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	agent, subject, peerID, foreign, unrelated := create("scope_agent"), create("scope_subject"), create("scope_peer"), create("scope_foreign"), create("scope_unrelated")
	roleID, err := insert(ctx, a.db, a.t("imgo_admin_role"), M{"name": "scope_test_role", "agent_mode": 1, "status": 1, "created_at": time.Now().Unix(), "updated_at": time.Now().Unix()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.db.ExecContext(ctx, "UPDATE "+a.t("user")+" SET admin_role_id=? WHERE user_id=?", roleID, agent); err != nil {
		t.Fatal(err)
	}
	for _, uid := range []int64{subject, peerID, unrelated} {
		if _, err = a.db.ExecContext(ctx, "INSERT INTO "+a.t("imgo_referral_path")+" (ancestor_user_id,descendant_user_id,depth) VALUES (?,?,1)", agent, uid); err != nil {
			t.Fatal(err)
		}
	}
	requestFor := func(path string, params M) *request {
		r := bankRequest(a, path, agent, params)
		r.user["admin_role_id"] = roleID
		r.c.Request = r.c.Request.WithContext(ctx)
		return r
	}
	t.Run("contacts omit foreign and unrelated users", func(t *testing.T) {
		config, err := one(ctx, a.db, "SELECT value FROM "+a.t("config")+" WHERE name='sysInfo'")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = a.db.ExecContext(ctx, "UPDATE "+a.t("config")+" SET value=JSON_SET(value,'$.runMode',1) WHERE name='sysInfo'"); err != nil {
			t.Fatal(err)
		}
		defer a.db.ExecContext(ctx, "UPDATE "+a.t("config")+" SET value=? WHERE name='sysInfo'", config["value"])
		content, err := encryptContent(a.cfg.ChatKey, "仅限已有下级会话")
		if err != nil {
			t.Fatal(err)
		}
		for _, target := range []int64{peerID, foreign} {
			if _, err = insert(ctx, a.db, a.t("message"), M{"id": randomID()[:32], "from_user": subject, "to_user": target, "is_group": 0, "chat_identify": chatKey(subject, target, 0), "content": content, "type": "text", "status": 1, "create_time": time.Now().Unix()}); err != nil {
				t.Fatal(err)
			}
		}
		result, err := a.manageMessage(requestFor("/manage/message/getContacts", M{"user_id": subject}))
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, contact := range result.([]M) {
			id := number(contact["user_id"])
			if id == peerID {
				found = true
			}
			if id == foreign || id == unrelated || id == agent {
				t.Fatalf("unauthorized or unrelated contact: %v", contact)
			}
		}
		if !found {
			t.Fatal("existing descendant conversation was hidden")
		}
	})
	for _, act := range []string{"credit", "review"} {
		t.Run(act+" keeps authorization locked while waiting for wallet", func(t *testing.T) {
			if _, err := a.db.ExecContext(ctx, "INSERT INTO "+a.t("imgo_wallet")+" (user_id,available_cents,pending_cents,updated_at) VALUES (?,1000,100,0) ON DUPLICATE KEY UPDATE available_cents=1000,pending_cents=100", subject); err != nil {
				t.Fatal(err)
			}
			params := M{"user_id": subject, "amount": "1.00", "request_id": "scope-credit-retry", "note": "范围锁测试"}
			if act == "review" {
				orderID, err := insert(ctx, a.db, a.t("imgo_withdrawal"), M{"user_id": subject, "request_id": "scope-review-order", "amount_cents": 100, "status": 0, "receipt_name": "测试用户", "bank_name": "测试银行", "branch_name": "测试支行", "account_cipher": "unused", "account_last4": "1234", "created_at": time.Now().Unix()})
				if err != nil {
					t.Fatal(err)
				}
				params = M{"withdrawal_id": orderID, "status": 1, "remark": "范围锁测试"}
			}
			blocker, err := a.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer blocker.Rollback()
			if _, err = one(ctx, blocker, "SELECT user_id FROM "+a.t("imgo_wallet")+" WHERE user_id=? FOR UPDATE", subject); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { _, err := a.manageWallet(requestFor("/manage/wallet/"+act, params)); done <- err }()
			// NOWAIT observes the handler's user lock while its wallet query is blocked.
			deadline := time.Now().Add(3 * time.Second)
			locked := false
			for time.Now().Before(deadline) {
				_, err = one(ctx, a.db, "SELECT user_id FROM "+a.t("user")+" WHERE user_id=? FOR UPDATE NOWAIT", subject)
				var mysqlErr *mysql.MySQLError
				if errors.As(err, &mysqlErr) && mysqlErr.Number == 3572 {
					locked = true
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				select {
				case err := <-done:
					t.Fatalf("handler returned before wallet lock release: %v", err)
				default:
				}
				time.Sleep(10 * time.Millisecond)
			}
			if !locked {
				t.Fatal("financial operation did not hold authorized user lock")
			}
			deleter, err := a.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = deleter.ExecContext(ctx, "SET innodb_lock_wait_timeout=1"); err != nil {
				deleter.Rollback()
				t.Fatal(err)
			}
			_, err = deleter.ExecContext(ctx, "UPDATE "+a.t("user")+" SET delete_time=1 WHERE user_id=?", subject)
			var mysqlErr *mysql.MySQLError
			if !errors.As(err, &mysqlErr) || mysqlErr.Number != 1205 {
				deleter.Rollback()
				t.Fatalf("deletion must wait for financial authorization lock, got %v", err)
			}
			_, _ = deleter.ExecContext(ctx, "SET innodb_lock_wait_timeout=DEFAULT")
			_ = deleter.Rollback()
			if err = blocker.Rollback(); err != nil {
				t.Fatal(err)
			}
			select {
			case err = <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			if _, err = a.db.ExecContext(ctx, "UPDATE "+a.t("user")+" SET delete_time=1 WHERE user_id=?", subject); err != nil {
				t.Fatal(err)
			}
			if err = a.requireScopedUser(ctx, a.db, adminScope{AgentUserID: agent}, subject); err == nil {
				t.Fatal("deleted user remained authorized")
			}
			if _, err = a.db.ExecContext(ctx, "UPDATE "+a.t("user")+" SET delete_time=0 WHERE user_id=?", subject); err != nil {
				t.Fatal(err)
			}
			t.Log(fmt.Sprintf("%s blocked concurrent deletion until its wallet transaction committed", act))
		})
	}
}
