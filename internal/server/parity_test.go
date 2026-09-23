package server

import (
	"strings"
	"testing"
)

func TestLegacyFileCategories(t *testing.T) {
	for _, tc := range []struct {
		ext  string
		cate int
		kind string
	}{{"pdf", 1, "file"}, {"png", 2, "image"}, {"mp3", 3, "file"}, {"mov", 4, "video"}, {"zip", 9, "file"}} {
		cate, kind := fileCategory(tc.ext, false)
		if cate != tc.cate || kind != tc.kind {
			t.Errorf("%s: %d %s", tc.ext, cate, kind)
		}
	}
	if _, kind := fileCategory("mp3", true); kind != "voice" {
		t.Fatal(kind)
	}
}
func TestImageMetadata(t *testing.T) {
	if number(imageMetadata(100, 100)["fixMode"]) != 3 || number(imageMetadata(400, 200)["fixMode"]) != 1 || number(imageMetadata(200, 400)["fixMode"]) != 2 {
		t.Fatal("image sizing incompatible")
	}
}
func TestCustomerServiceRotation(t *testing.T) {
	users := []int64{4, 6, 8}
	for _, tc := range []struct{ last, want int64 }{{0, 4}, {4, 6}, {6, 8}, {8, 4}, {9, 4}} {
		if got := nextCustomerService(users, tc.last); got != tc.want {
			t.Fatal(got, tc)
		}
	}
}
func TestChineseNameIndex(t *testing.T) {
	if got := namePinyin("张三Alice"); !strings.Contains(got, "zhang") || !strings.Contains(got, "san") || !strings.Contains(got, "Alice") {
		t.Fatal(got)
	}
}
