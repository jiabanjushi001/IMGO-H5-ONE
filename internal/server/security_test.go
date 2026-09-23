package server

import (
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordMigration(t *testing.T) {
	old := legacyPassword("old-password", "abcd")
	if !checkPassword(old, "abcd", "old-password") || checkPassword(old, "abcd", "wrong") {
		t.Fatal("legacy password verification")
	}
	h, e := hashPassword("new-password")
	if e != nil || !checkPassword(h, "", "new-password") {
		t.Fatal("bcrypt verification", e)
	}
}

func TestNewPasswordLength(t *testing.T) {
	for _, tc := range []struct {
		name    string
		length  int
		wantErr bool
	}{
		{"too short", 5, true},
		{"minimum", 6, false},
		{"maximum", 30, false},
		{"too long", 31, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			password := strings.Repeat("a", tc.length)
			hash, err := hashPassword(password)
			if (err != nil) != tc.wantErr {
				t.Fatalf("hashPassword(%d characters) error = %v, wantErr %v", tc.length, err, tc.wantErr)
			}
			if err == nil && !checkPassword(hash, "", password) {
				t.Fatal("new password cannot be verified")
			}
		})
	}
	legacyHash, err := bcrypt.GenerateFromPassword([]byte(strings.Repeat("a", 31)), bcrypt.MinCost)
	if err != nil || !checkPassword(string(legacyHash), "", strings.Repeat("a", 31)) {
		t.Fatal("existing password over the new limit must still log in", err)
	}
}
func TestTokens(t *testing.T) {
	key := strings.Repeat("k", 32)
	token := signToken(key, 7, "session", time.Now().Add(time.Hour))
	c, e := parseToken(key, token)
	if e != nil || c.UID != 7 {
		t.Fatal(c, e)
	}
	if _, e = parseToken("different-key", token); e == nil {
		t.Fatal("accepted forged signature")
	}
	if _, e = parseToken(key, signToken(key, 7, "s", time.Now().Add(-time.Second))); e == nil {
		t.Fatal("accepted expired token")
	}
}
func TestLegacyEncryption(t *testing.T) {
	for _, plain := range []string{"", "你好，世界", "1234567890123456"} {
		encrypted, e := encryptContent("1234567890123456", plain)
		if e != nil {
			t.Fatal(e)
		}
		got, e := decryptContent("1234567890123456", encrypted)
		if e != nil || got != plain {
			t.Fatal(got, e)
		}
	}
	if _, e := decryptContent("secret", "not ciphertext"); e == nil {
		t.Fatal("accepted invalid ciphertext")
	}
}
func TestSafePath(t *testing.T) {
	for _, p := range []string{"../.env", "a.php", "a.PHp", "dir/a.phtml", ".env", "a.php.jpg"} {
		if safeAsset(p) {
			t.Errorf("unsafe asset %q", p)
		}
	}
	if !safeAsset("assets/js/app.js") || !safeAsset("storage/image/a.png") {
		t.Fatal("valid assets rejected")
	}
}
func TestRichTextRejectsScript(t *testing.T) {
	s := sanitizeText(`<b>你好</b><img src="x" onerror="alert(1)"><script>alert(2)</script><a href="javascript:alert(3)">x</a>`)
	if strings.Contains(s, "onerror") || strings.Contains(s, "javascript:") || strings.Contains(s, "<script") {
		t.Fatal(s)
	}
	if !strings.Contains(s, "<b>你好</b>") {
		t.Fatal("formatting lost", s)
	}
}
