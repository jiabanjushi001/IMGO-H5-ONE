package server

import (
	"strings"
	"testing"
)

func TestIdentityNumberValidationEncryptionAndMasking(t *testing.T) {
	valid := "11010519491231002X"
	for _, invalid := range []string{"", "110105194912310021", "11010519490230002X", "11010519491231002A"} {
		if validIdentityNumber(invalid) {
			t.Fatalf("accepted invalid identity number %q", invalid)
		}
	}
	if !validIdentityNumber(valid) || maskIdentityNumber(valid) != "110105********002X" {
		t.Fatal("valid identity number was rejected or masked incorrectly")
	}
	if !validIdentityName("测试用户") || validIdentityName("A") || validIdentityName("测试\n用户") {
		t.Fatal("identity name validation failed")
	}
	key := strings.Repeat("k", 32)
	ciphertext, err := encryptIdentityNumber(key, valid)
	if err != nil || strings.Contains(ciphertext, valid) {
		t.Fatalf("identity encryption failed: %v", err)
	}
	plain, err := decryptIdentityNumber(key, ciphertext)
	if err != nil || plain != valid {
		t.Fatalf("identity decryption failed: %v", err)
	}
	row := M{"id_number_cipher": ciphertext, "id_number_masked": maskIdentityNumber(valid)}
	if err = exposeIdentityNumber(row, key); err != nil || str(row["id_number"]) != valid {
		t.Fatalf("identity number was not exposed to its authorized viewer: %v", err)
	}
	if _, exists := row["id_number_cipher"]; exists {
		t.Fatal("identity ciphertext leaked into the response")
	}
	if identityNumberHash(key, valid) == identityNumberHash(key, "110105194912310010") {
		t.Fatal("identity hash does not distinguish numbers")
	}
}

func TestIdentityRoutesUseAuthenticatedUserAndMemberPermission(t *testing.T) {
	a, _ := testApp(t)
	for _, path := range []string{"/enterprise/identity/get", "/enterprise/identity/save"} {
		route, ok := a.routes[normalizedPath(path)]
		if !ok || route.public || route.super {
			t.Fatalf("user identity route is not authenticated correctly: %s %#v", path, route)
		}
	}
	for _, path := range []string{"/manage/identity/index", "/manage/identity/detail", "/manage/identity/review"} {
		route, ok := a.routes[normalizedPath(path)]
		if !ok || route.public || route.super || route.permission != "manage.users" {
			t.Fatalf("management identity route is not scoped correctly: %s %#v", path, route)
		}
	}
}
