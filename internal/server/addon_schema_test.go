package server

import (
	"context"
	"database/sql"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
)

func TestAddonDDLIncludesAgentTables(t *testing.T) {
	a := &App{cfg: Config{Prefix: "test_"}}
	joinedDDL := strings.Join(a.addonTableDDL(), "\n")
	for _, name := range []string{
		"imgo_agent_setting", "imgo_agent_auto_state", "imgo_agent_online_sample",
	} {
		if !strings.Contains(joinedDDL, name) {
			t.Fatalf("missing %s", name)
		}
	}
}

func TestSeedMentorRoleCreatesSevenPermissionsOnce(t *testing.T) {
	a, mock := testApp(t)
	ctx := context.Background()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT role_id FROM `yu_imgo_admin_role` WHERE role_code=?")).
		WithArgs("mentor").WillReturnRows(sqlmock.NewRows([]string{"role_id"}))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT role_id FROM `yu_imgo_admin_role` WHERE name=?")).
		WithArgs("导师专员").WillReturnRows(sqlmock.NewRows([]string{"role_id"}))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `yu_imgo_admin_role` ")).
		WithArgs("导师专员", "", 1, 1, "mentor", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(9, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `yu_imgo_admin_role_permission` (role_id,permission_id) SELECT ?,permission_id FROM `yu_imgo_admin_permission` WHERE permission_key IN (?,?,?,?,?,?,?)")).
		WithArgs(int64(9), "manage.overview", "manage.users", "manage.messages", "manage.groups", "manage.files", "manage.bank", "manage.finance").
		WillReturnResult(sqlmock.NewResult(0, 7))
	if err := a.seedMentorRole(ctx, a.db); err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT role_id FROM `yu_imgo_admin_role` WHERE role_code=?")).
		WithArgs("mentor").WillReturnRows(sqlmock.NewRows([]string{"role_id"}).AddRow(9))
	if err := a.seedMentorRole(ctx, a.db); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSeedMentorRolePreservesCustomRoleWithSameName(t *testing.T) {
	a, mock := testApp(t)
	ctx := context.Background()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT role_id FROM `yu_imgo_admin_role` WHERE role_code=?")).
		WithArgs("mentor").WillReturnRows(sqlmock.NewRows([]string{"role_id"}))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT role_id FROM `yu_imgo_admin_role` WHERE name=?")).
		WithArgs("导师专员").WillReturnRows(sqlmock.NewRows([]string{"role_id"}).AddRow(5))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT role_id FROM `yu_imgo_admin_role` WHERE name=?")).
		WithArgs("导师专员（预置）").WillReturnRows(sqlmock.NewRows([]string{"role_id"}))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `yu_imgo_admin_role` ")).
		WithArgs("导师专员（预置）", "", 1, 1, "mentor", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(9, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `yu_imgo_admin_role_permission` (role_id,permission_id) SELECT ?,permission_id FROM `yu_imgo_admin_permission` WHERE permission_key IN (?,?,?,?,?,?,?)")).
		WithArgs(int64(9), "manage.overview", "manage.users", "manage.messages", "manage.groups", "manage.files", "manage.bank", "manage.finance").
		WillReturnResult(sqlmock.NewResult(0, 7))
	if err := a.seedMentorRole(ctx, a.db); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

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
		"CREATE TABLE test_imgo_admin_role (role_id BIGINT AUTO_INCREMENT PRIMARY KEY,name VARCHAR(64) NOT NULL,remark VARCHAR(255) NOT NULL DEFAULT '',status TINYINT NOT NULL DEFAULT 1,created_at BIGINT NOT NULL,updated_at BIGINT NOT NULL,UNIQUE KEY imgo_admin_role_name(name))",
		"INSERT INTO test_imgo_admin_role (role_id,name,remark,status,created_at,updated_at) VALUES (41,'导师专员','自定义角色',1,1,1)",
		"CREATE TABLE test_imgo_admin_permission (permission_id BIGINT AUTO_INCREMENT PRIMARY KEY,permission_key VARCHAR(64) NOT NULL,name VARCHAR(64) NOT NULL,menu_path VARCHAR(128) NOT NULL DEFAULT '',sort INT NOT NULL DEFAULT 0,UNIQUE KEY imgo_admin_permission_key(permission_key))",
		"INSERT INTO test_imgo_admin_permission (permission_key,name,menu_path,sort) VALUES ('manage.extra','历史额外权限','',99)",
	} {
		if _, err = db.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	if err = a.EnsureAddonSchema(ctx); err != nil {
		t.Fatal(err)
	}
	var customName, customRemark string
	var customMode int
	var customCode sql.NullString
	if err = db.QueryRow("SELECT name,remark,agent_mode,role_code FROM test_imgo_admin_role WHERE role_id=41").Scan(&customName, &customRemark, &customMode, &customCode); err != nil || customName != "导师专员" || customRemark != "自定义角色" || customMode != 0 || customCode.Valid {
		t.Fatalf("custom role changed: name=%q remark=%q mode=%d code=%v err=%v", customName, customRemark, customMode, customCode, err)
	}
	var mentorID int64
	var mentorName string
	var mentorMode, mentorStatus int
	if err = db.QueryRow("SELECT role_id,name,agent_mode,status FROM test_imgo_admin_role WHERE role_code='mentor'").Scan(&mentorID, &mentorName, &mentorMode, &mentorStatus); err != nil || mentorName != "导师专员（预置）" || mentorMode != 1 || mentorStatus != 1 {
		t.Fatalf("new mentor role: id=%d name=%q mode=%d status=%d err=%v", mentorID, mentorName, mentorMode, mentorStatus, err)
	}
	mentorPermissions := func() []string {
		t.Helper()
		r, queryErr := db.Query("SELECT p.permission_key FROM test_imgo_admin_role_permission rp JOIN test_imgo_admin_permission p ON p.permission_id=rp.permission_id WHERE rp.role_id=? ORDER BY p.permission_key", mentorID)
		if queryErr != nil {
			t.Fatal(queryErr)
		}
		defer r.Close()
		var keys []string
		for r.Next() {
			var key string
			if scanErr := r.Scan(&key); scanErr != nil {
				t.Fatal(scanErr)
			}
			keys = append(keys, key)
		}
		if rowErr := r.Err(); rowErr != nil {
			t.Fatal(rowErr)
		}
		return keys
	}
	if got, want := mentorPermissions(), []string{"manage.bank", "manage.files", "manage.finance", "manage.groups", "manage.messages", "manage.overview", "manage.users"}; !slices.Equal(got, want) {
		t.Fatalf("initial mentor permissions = %v, want %v", got, want)
	}
	for _, q := range []string{
		"UPDATE test_imgo_admin_role SET name='导师专员（管理员修改）',status=0,agent_mode=0 WHERE role_id=?",
		"DELETE FROM test_imgo_admin_role_permission WHERE role_id=? AND permission_id=(SELECT permission_id FROM test_imgo_admin_permission WHERE permission_key='manage.finance')",
		"INSERT INTO test_imgo_admin_role_permission (role_id,permission_id) SELECT ?,permission_id FROM test_imgo_admin_permission WHERE permission_key='manage.settings'",
	} {
		if _, err = db.ExecContext(ctx, q, mentorID); err != nil {
			t.Fatal(err)
		}
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
	for _, table := range []string{"test_imgo_agent_setting", "test_imgo_agent_auto_state", "test_imgo_agent_online_sample"} {
		var exists int
		if err = db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name=?", table).Scan(&exists); err != nil || exists != 1 {
			t.Fatalf("missing agent table %s: %v", table, err)
		}
	}
	for _, column := range []string{"agent_mode", "role_code"} {
		var exists int
		if err = db.QueryRow("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='test_imgo_admin_role' AND column_name=?", column).Scan(&exists); err != nil || exists != 1 {
			t.Fatalf("missing role column %s: %v", column, err)
		}
	}
	var mentorCount int
	if err = db.QueryRow("SELECT COUNT(*) FROM test_imgo_admin_role WHERE role_code='mentor'").Scan(&mentorCount); err != nil || mentorCount != 1 {
		t.Fatalf("mentor role count = %d: %v", mentorCount, err)
	}
	if err = db.QueryRow("SELECT name,agent_mode,status FROM test_imgo_admin_role WHERE role_id=?", mentorID).Scan(&mentorName, &mentorMode, &mentorStatus); err != nil || mentorName != "导师专员（管理员修改）" || mentorMode != 0 || mentorStatus != 0 {
		t.Fatalf("mentor edits overwritten: name=%q mode=%d status=%d err=%v", mentorName, mentorMode, mentorStatus, err)
	}
	if got, want := mentorPermissions(), []string{"manage.bank", "manage.files", "manage.groups", "manage.messages", "manage.overview", "manage.settings", "manage.users"}; !slices.Equal(got, want) {
		t.Fatalf("mentor permissions overwritten: got %v, want %v", got, want)
	}
	var permissionCount int
	if err = db.QueryRow("SELECT COUNT(*) FROM test_imgo_admin_permission WHERE permission_key LIKE 'manage.%'").Scan(&permissionCount); err != nil || permissionCount != 9 {
		t.Fatalf("expected 8 catalog and 1 historical permission, got %d: %v", permissionCount, err)
	}
}
