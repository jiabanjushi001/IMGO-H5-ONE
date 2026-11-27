package server

import (
	"database/sql"
	"encoding/base64"
	"image/png"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestMergeGroupSettings(t *testing.T) {
	current := M{"manage": 1, "history": 0, "number_join": 1, "profile": 0, "legacy": "keep"}
	settings, err := mergeGroupSettings(current, M{"profile": "1", "nospeak": "2", "number_join": 0, "legacy": "changed"}, 1)
	if err != nil || number(settings["profile"]) != 1 || number(settings["nospeak"]) != 2 ||
		number(settings["number_join"]) != 1 || number(settings["history"]) != 0 ||
		number(settings["manage"]) != 1 || settings["legacy"] != "keep" || number(current["profile"]) != 0 {
		t.Fatalf("owner update settings=%v current=%v err=%v", settings, current, err)
	}
	settings, err = mergeGroupSettings(current, M{"nospeak": 1}, 2)
	if err != nil || number(settings["nospeak"]) != 1 || number(settings["number_join"]) != 1 {
		t.Fatalf("manager mute settings=%v err=%v", settings, err)
	}
	for _, edit := range []M{{"profile": 1}, {"history": 1}, {"number_join": 0}} {
		if _, err := mergeGroupSettings(current, edit, 2); err == nil {
			t.Fatalf("manager was allowed to change %v", edit)
		}
	}
	for _, edit := range []M{{"profile": -1}, {"nospeak": 3}, {"nospeak": "garbage"}} {
		if _, err := mergeGroupSettings(current, edit, 1); err == nil {
			t.Fatalf("accepted invalid setting %v", edit)
		}
	}
}

func TestGroupInvitationQRIsPNG(t *testing.T) {
	data, err := qrPNGData("https://example.test/scan/g/5.1800000000.signature", 320)
	if err != nil {
		t.Fatal(err)
	}
	raw, ok := strings.CutPrefix(data, "data:image/png;base64,")
	if !ok {
		t.Fatal("missing PNG data URL")
	}
	config, err := png.DecodeConfig(base64.NewDecoder(base64.StdEncoding, strings.NewReader(raw)))
	if err != nil || config.Width != 320 || config.Height != 320 {
		t.Fatalf("config=%v err=%v", config, err)
	}
}

func TestGroupMemberInteractionRespectsPrivacy(t *testing.T) {
	for _, test := range []struct {
		name                     string
		role, profile, exception int64
		allowed, targetPresent   bool
	}{
		{"private group", 3, 0, 0, false, true},
		{"open member profile", 3, 1, 0, true, true},
		{"privileged member", 3, 0, 1, true, true},
		{"group manager", 2, 0, 0, true, true},
		{"target outside group", 3, 1, 0, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			a, mock := testApp(t)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/enterprise/friend/add", nil)
			mock.ExpectQuery("SELECT gu.*").WithArgs(int64(5), int64(8)).
				WillReturnRows(sqlmock.NewRows([]string{"role", "group_setting"}).
					AddRow(test.role, js(M{"profile": test.profile})))
			if test.allowed || !test.targetPresent {
				query := mock.ExpectQuery("SELECT gu.*").WithArgs(int64(5), int64(9))
				if test.targetPresent {
					query.WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow(3))
				} else {
					query.WillReturnError(sql.ErrNoRows)
				}
			}
			r := &request{app: a, c: c, user: M{"user_id": 8, "group_friend_add": test.exception}, p: M{"group_id": "group-5"}}
			if err := a.requireGroupMemberInteraction(r, 9); (err == nil) != test.allowed {
				t.Fatalf("allowed=%v, err=%v", test.allowed, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
