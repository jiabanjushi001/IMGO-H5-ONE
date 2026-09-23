package server

import (
	"context"
	"database/sql"
	"github.com/go-sql-driver/mysql"
	"os"
	"strings"
	"testing"
	"time"
)

func TestStartupAddonUpgrade(t *testing.T) {
	password := os.Getenv("IMGO_TEST_MYSQL_PASSWORD")
	if password == "" {
		t.Skip("requires isolated MySQL test database")
	}
	cfg := mysql.NewConfig()
	cfg.User = "root"
	cfg.Passwd = password
	cfg.Net = "tcp"
	cfg.Addr = "127.0.0.1:3306"
	root, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	name := "imgo_upgrade_test_" + randomID()[:12]
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if _, err = root.ExecContext(ctx, "CREATE DATABASE `"+name+"`"); err != nil {
		t.Fatal(err)
	}
	defer root.Exec("DROP DATABASE `" + name + "`")
	cfg.DBName = name
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	a := &App{db: db, cfg: Config{Prefix: "test_", JWTKey: strings.Repeat("x", 32)}}
	// A pre-existing user and balance must survive two automatic upgrades.
	for _, q := range []string{
		"CREATE TABLE test_user (user_id INT PRIMARY KEY, account VARCHAR(32))",
		"INSERT INTO test_user VALUES (7,'existing')",
		"CREATE TABLE test_imgo_wallet (user_id INT PRIMARY KEY,available_cents BIGINT,pending_cents BIGINT,updated_at BIGINT)",
		"INSERT INTO test_imgo_wallet VALUES (7,12345,678,1)",
	} {
		if _, err = db.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	if err = a.EnsureAddonSchema(ctx); err != nil {
		t.Fatal(err)
	}
	var invite string
	if err = db.QueryRow("SELECT invite_code FROM test_imgo_referral WHERE user_id=7").Scan(&invite); err != nil {
		t.Fatal(err)
	}
	if !validInviteCode(invite) {
		t.Fatal("new invite code must be six digits", invite)
	}
	// Existing five-letter invitations are converted, but old shared links still work.
	if _, err = db.Exec("UPDATE test_imgo_referral SET invite_code='ABCDE' WHERE user_id=7"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("ALTER TABLE test_imgo_referral MODIFY invite_code CHAR(5) CHARACTER SET ascii COLLATE ascii_bin NOT NULL"); err != nil {
		t.Fatal(err)
	}
	// Simulate an older extension table missing a later-added column.
	if _, err = db.Exec("ALTER TABLE test_imgo_withdrawal DROP COLUMN request_note"); err != nil {
		t.Fatal(err)
	}
	if err = a.EnsureAddonSchema(ctx); err != nil {
		t.Fatal(err)
	}
	var code, account string
	var balance int64
	if err = db.QueryRow("SELECT r.invite_code,u.account,w.available_cents FROM test_imgo_referral r JOIN test_user u ON u.user_id=r.user_id JOIN test_imgo_wallet w ON w.user_id=u.user_id WHERE u.user_id=7").Scan(&code, &account, &balance); err != nil {
		t.Fatal(err)
	}
	if !validInviteCode(code) || account != "existing" || balance != 12345 {
		t.Fatal("existing data changed", code, account, balance)
	}
	var legacyUserID int64
	if err = db.QueryRow("SELECT user_id FROM test_imgo_referral_legacy_code WHERE invite_code='ABCDE'").Scan(&legacyUserID); err != nil || legacyUserID != 7 {
		t.Fatal("old invitation link was not preserved", legacyUserID, err)
	}
	if err = a.EnsureAddonSchema(ctx); err != nil {
		t.Fatal("repeat upgrade failed", err)
	}
	var repeatCode string
	if err = db.QueryRow("SELECT invite_code FROM test_imgo_referral WHERE user_id=7").Scan(&repeatCode); err != nil || repeatCode != code {
		t.Fatal("repeat upgrade changed invite code", repeatCode, err)
	}
	if _, err = db.Exec("SELECT request_note FROM test_imgo_withdrawal LIMIT 0"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("SELECT last_chat_time,last_chat_ip FROM test_user LIMIT 0"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("SELECT admin_role_id FROM test_user LIMIT 0"); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"test_imgo_admin_role", "test_imgo_admin_permission", "test_imgo_admin_role_permission"} {
		var exists int
		if err = db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name=?", table).Scan(&exists); err != nil || exists != 1 {
			t.Fatalf("missing RBAC table %s: %v", table, err)
		}
	}
	var permissionCount int
	if err = db.QueryRow("SELECT COUNT(*) FROM test_imgo_admin_permission WHERE permission_key LIKE 'manage.%'").Scan(&permissionCount); err != nil || permissionCount != 8 {
		t.Fatalf("expected 8 seeded permissions, got %d: %v", permissionCount, err)
	}
}
