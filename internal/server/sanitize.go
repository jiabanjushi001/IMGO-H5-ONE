package server

import "github.com/microcosm-cc/bluemonday"

var richTextPolicy = bluemonday.UGCPolicy()

func sanitizeText(s string) string { return richTextPolicy.Sanitize(s) }
