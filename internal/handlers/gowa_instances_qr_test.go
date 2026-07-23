package handlers

import (
	"testing"
)

func TestRebaseGowaQRLink(t *testing.T) {
	cases := []struct {
		name    string
		qrLink  string
		baseURL string
		want    string
	}{
		{
			name:    "localhost link rebased onto configured docker host",
			qrLink:  "http://localhost:3000/statics/images/qrcode/scan-qr-abc.png",
			baseURL: "http://gowa:8080",
			want:    "http://gowa:8080/statics/images/qrcode/scan-qr-abc.png",
		},
		{
			name:    "link already on the right host is unchanged",
			qrLink:  "http://gowa:8080/statics/images/qrcode/scan-qr-abc.png",
			baseURL: "http://gowa:8080",
			want:    "http://gowa:8080/statics/images/qrcode/scan-qr-abc.png",
		},
		{
			name:    "relative link is prefixed with the base",
			qrLink:  "/statics/images/qrcode/scan-qr-abc.png",
			baseURL: "http://gowa:8080",
			want:    "http://gowa:8080/statics/images/qrcode/scan-qr-abc.png",
		},
		{
			name:    "query string preserved on rebase",
			qrLink:  "http://localhost:3000/statics/qr.png?token=xyz",
			baseURL: "https://gowa.example.com",
			want:    "https://gowa.example.com/statics/qr.png?token=xyz",
		},
		{
			name:    "empty base url leaves link untouched",
			qrLink:  "http://localhost:3000/x.png",
			baseURL: "",
			want:    "http://localhost:3000/x.png",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := rebaseGowaQRLink(c.qrLink, c.baseURL)
			if got != c.want {
				t.Errorf("rebaseGowaQRLink(%q, %q) = %q, want %q", c.qrLink, c.baseURL, got, c.want)
			}
		})
	}
}
