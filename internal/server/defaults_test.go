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

func TestDefaultCheckInRewardSettings(t *testing.T) {
	settings, err := decodeCheckInSettings(defaultConfig("checkIn"))
	if err != nil {
		t.Fatal(err)
	}
	if !settings.Enabled || settings.CycleDays != 10 || settings.HighDays != 3 || settings.HighMinYuan != 7 || settings.HighMaxYuan != 10 || settings.NormalMinYuan != 3 || settings.NormalMaxYuan != 5 {
		t.Fatalf("unexpected default check-in settings: %#v", settings)
	}
}

func TestDefaultWithdrawalSettings(t *testing.T) {
	settings, err := decodeWithdrawalSettings(defaultConfig("withdrawal"))
	if err != nil {
		t.Fatal(err)
	}
	if !settings.Enabled || settings.MinYuan != 100 || settings.PasswordRequired {
		t.Fatalf("unexpected default withdrawal settings: %#v", settings)
	}
}
