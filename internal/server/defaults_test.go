package server

import "testing"

func TestDefaultChatSendInterval(t *testing.T) {
	if got := number(defaultConfig("chatInfo")["sendInterval"]); got != 1 {
		t.Fatalf("default chat send interval = %d seconds, want 1", got)
	}
}

func TestDefaultGroupCreateRoleIsMentor(t *testing.T) {
	if got := str(defaultConfig("chatInfo")["groupCreateRole"]); got != groupCreateRoleMentor {
		t.Fatalf("default group create role = %q, want %q", got, groupCreateRoleMentor)
	}
}
