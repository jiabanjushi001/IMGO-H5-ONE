package server

import "strings"

func fileCategory(ext string, voice bool) (int, string) {
	switch strings.ToLower(ext) {
	case "ppt", "pptx", "doc", "docx", "xls", "xlsx", "pdf", "txt", "md":
		return 1, "file"
	case "jpg", "jpeg", "png", "bmp", "gif", "webp", "ico":
		return 2, "image"
	case "mp3", "wav", "wmv", "amr":
		if voice {
			return 3, "voice"
		}
		return 3, "file"
	case "mp4", "3gp", "avi", "m2v", "mkv", "mov", "webm":
		return 4, "video"
	}
	return 9, "file"
}
func imageMetadata(width, height int) M {
	mode := 1
	if width < height {
		mode = 2
	}
	if width < 200 && height < 240 {
		mode = 3
	}
	return M{"width": width, "height": height, "fixMode": mode}
}
