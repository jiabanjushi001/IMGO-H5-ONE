package server

import (
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
)

func TestMemberAddInviter(t *testing.T) {
	for _, tc := range []struct {
		name, code  string
		global      bool
		owner, want int64
		bad         bool
	}{
		{"super empty", "", true, 0, 0, false},
		{"super valid", "123456", true, 12, 12, false},
		{"super missing", "123456", true, 0, 0, true},
		{"bad format", "abcdef", true, 0, 0, true},
		{"mentor omitted", "", false, 0, 7, false},
		{"mentor own", "123456", false, 7, 7, false},
		{"mentor foreign", "123456", false, 12, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, mock := testApp(t)
			if validInviteCode(tc.code) {
				rows := sqlmock.NewRows([]string{"user_id"})
				if tc.owner > 0 {
					rows.AddRow(tc.owner)
				}
				mock.ExpectQuery("SELECT r.user_id FROM `yu_imgo_referral`").WithArgs(tc.code).WillReturnRows(rows)
				if tc.owner > 0 {
					mock.ExpectQuery("SELECT user_id FROM `yu_user`.*status=1 AND delete_time=0").WithArgs(tc.owner).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(tc.owner))
				}
			}
			r := agentMemberRequest(a, "/manage/user/add", M{"parent_invite_code": tc.code})
			got, err := a.memberAddInviter(r, a.db, adminScope{Global: tc.global, AgentUserID: 7})
			if (err != nil) != tc.bad || got != tc.want {
				t.Fatalf("got %d %v", got, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
