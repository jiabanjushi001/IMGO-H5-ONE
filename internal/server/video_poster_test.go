package server

import (
	"strings"
	"testing"
)

func TestVideoPosterPlaceholder(t *testing.T) {
	a := &App{}
	for _, input := range []string{"", "/static/common/img/video.png", "https://old.example/static/common/img/video.png?v=1"} {
		got := a.videoPoster(input)
		if !strings.HasSuffix(got, "/static/common/img/video-placeholder.svg") {
			t.Fatalf("unexpected poster %q", got)
		}
	}
	actual := "https://example.com/storage/cover/real.jpg"
	if got := a.videoPoster(actual); got != "/storage/cover/real.jpg" {
		t.Fatalf("actual thumbnail changed: %s", got)
	}
}
