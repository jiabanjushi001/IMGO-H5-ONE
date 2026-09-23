package server

import "testing"

func TestRelativeMediaPath(t *testing.T) {
	a := &App{cfg: Config{BaseURL: "https://new.example"}}
	for _, tc := range []struct{ in, want string }{
		{"", ""}, {"/storage/image/a.png", "/storage/image/a.png"},
		{"./storage/a.png", "/storage/a.png"},
		{"https://old.example/storage/a.png?v=2", "/storage/a.png?v=2"},
		{"https://old.example/avatar/a/120/1", "/avatar/a/120/1"},
		{"https://cdn.example/external.png", "https://cdn.example/external.png"},
		{"data:image/png;base64,AA", "data:image/png;base64,AA"},
	} {
		if got := a.mediaPath(tc.in); got != tc.want {
			t.Errorf("%q: got %q want %q", tc.in, got, tc.want)
		}
	}
	if got := a.videoPoster("https://old.example/storage/cover/a.jpg"); got != "/storage/cover/a.jpg" {
		t.Fatal(got)
	}
	if got := a.userAvatar(M{"avatar": "https://old.example/storage/image/a.png"}); got != "/storage/image/a.png" {
		t.Fatal(got)
	}
	if got := a.url("/scan/1"); got != "https://new.example/scan/1" {
		t.Fatal("navigation changed", got)
	}
}
