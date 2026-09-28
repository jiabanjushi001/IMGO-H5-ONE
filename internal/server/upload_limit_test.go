package server

import "testing"

func TestUploadLimitMBUsesSeparateVideoLimit(t *testing.T) {
	tests := []struct {
		name   string
		config M
		kind   string
		want   int64
	}{
		{name: "ordinary file keeps configured limit", config: M{"size": "25"}, kind: "file", want: 25},
		{name: "ordinary file remains capped", config: M{"size": "500"}, kind: "file", want: maxFileUploadMB},
		{name: "video defaults to 200 MB for old config", config: M{"size": "50"}, kind: "video", want: defaultVideoUploadMB},
		{name: "video uses explicit lower limit", config: M{"videoSize": "120"}, kind: "video", want: 120},
		{name: "video remains capped", config: M{"videoSize": "500"}, kind: "video", want: maxVideoUploadMB},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := uploadLimitMB(test.config, test.kind); got != test.want {
				t.Fatalf("uploadLimitMB() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestUploadRequestBodyLimitLeavesMultipartHeadroom(t *testing.T) {
	if maxUploadRequestBodyMB <= maxVideoUploadMB {
		t.Fatalf("request-body limit %d MB must exceed video limit %d MB", maxUploadRequestBodyMB, maxVideoUploadMB)
	}
}
