package server

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestTCPEventFanoutUsesVersionedEnvelope(t *testing.T) {
	a, _ := testApp(t)
	a.tcpHub = newTCPHub(a)
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()
	peer := a.tcpHub.add(serverConn)
	defer a.tcpHub.remove(peer)

	a.tcpHub.mu.Lock()
	peer.uid = 7
	peer.claims = claims{UID: 7, SID: "tcp-push", Exp: time.Now().Add(time.Hour).Unix()}
	a.tcpHub.mu.Unlock()
	a.hub.send([]int64{7}, "simple", M{"id": "message-1"})

	select {
	case payload := <-peer.send:
		var event tcpEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			t.Fatal(err)
		}
		if event.Version != tcpProtocolVersion || event.Event != "simple" || str(obj(event.Data)["id"]) != "message-1" {
			t.Fatalf("event = %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("TCP event was not delivered")
	}
}

func tcpTestLine(t *testing.T, id, action string, data any) []byte {
	t.Helper()
	line, err := json.Marshal(M{"version": tcpProtocolVersion, "id": id, "action": action, "data": data})
	if err != nil {
		t.Fatal(err)
	}
	return line
}

func TestTCPPingUsesVersionedEnvelope(t *testing.T) {
	a, _ := testApp(t)
	response := a.handleTCPLine(context.Background(), &tcpConnectionState{}, "203.0.113.8:42000", tcpTestLine(t, "ping-1", "system.ping", M{}))
	if response.Code != 0 || response.ID != "ping-1" || response.Version != tcpProtocolVersion {
		t.Fatalf("response = %#v", response)
	}
	if str(obj(response.Data)["type"]) != "pong" {
		t.Fatalf("data = %#v", response.Data)
	}
}

func TestTCPLoginUsesExistingGoUserAndSessionTables(t *testing.T) {
	a, mock := testApp(t)
	password, err := hashPassword("test-password")
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT \\* FROM `yu_user` WHERE account=\\? AND delete_time=0").
		WithArgs("wuhu1").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "account", "realname", "password", "salt", "admin_role_id", "status", "delete_time"}).
			AddRow(7, "wuhu1", "五湖一号", password, "", 0, 1, 0))
	mock.ExpectExec("UPDATE `yu_user` SET login_count=").
		WithArgs(sqlmock.AnyArg(), "203.0.113.8", int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `yu_imgo_session`").
		WithArgs(false, sqlmock.AnyArg(), "203.0.113.8", sqlmock.AnyArg(), int64(7)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	response := a.handleTCPLine(context.Background(), &tcpConnectionState{}, "203.0.113.8:42000", tcpTestLine(t, "login-1", "auth.login", M{
		"account":  "wuhu1",
		"password": "test-password",
	}))
	if response.Code != 0 {
		t.Fatalf("response = %#v", response)
	}
	result := obj(response.Data)
	if !strings.HasPrefix(str(result["authToken"]), "bearer ") {
		t.Fatalf("missing shared Go auth token: %#v", result)
	}
	user := obj(result["userInfo"])
	if str(user["account"]) != "wuhu1" || str(user["displayName"]) != "五湖一号" {
		t.Fatalf("userInfo = %#v", user)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTCPSessionBindValidatesPersistentGoSession(t *testing.T) {
	a, mock := testApp(t)
	sid := "tcp-session"
	token := "bearer " + signToken(a.cfg.JWTKey, 7, sid, time.Now().Add(time.Hour))
	mock.ExpectQuery("SELECT user_id FROM `yu_imgo_session`").
		WithArgs(sid, int64(7), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7))
	mock.ExpectQuery("SELECT \\* FROM `yu_user` WHERE user_id=\\? AND status=1 AND delete_time=0").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "account", "realname", "status", "delete_time"}).
			AddRow(7, "wuhu1", "五湖一号", 1, 0))
	mock.ExpectExec("UPDATE `yu_user` SET last_chat_time=\\?,last_chat_ip=\\? WHERE user_id=\\?").
		WithArgs(sqlmock.AnyArg(), "203.0.113.8", int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	state := &tcpConnectionState{}
	response := a.handleTCPLine(context.Background(), state, "203.0.113.8:42000", tcpTestLine(t, "bind-1", "session.bind", M{
		"authToken": token,
		"deviceId":  "android-xiaomi-15-pro",
	}))
	if response.Code != 0 || state.authToken != token || state.deviceID != "android-xiaomi-15-pro" {
		t.Fatalf("response = %#v, state = %#v", response, state)
	}
	if str(obj(response.Data)["sessionId"]) != sid {
		t.Fatalf("data = %#v", response.Data)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTCPConfigRequiresBoundSession(t *testing.T) {
	a, _ := testApp(t)
	response := a.handleTCPLine(context.Background(), &tcpConnectionState{}, "203.0.113.8:42000", tcpTestLine(t, "config-1", "config.get", M{}))
	if response.Code != 401 {
		t.Fatalf("response = %#v", response)
	}
}
