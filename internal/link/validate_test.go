package link

import (
	"strings"
	"testing"
)

func TestValidateURL(t *testing.T) {
	const prefix = "https://example.com/"

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name: "http",
			url:  "http://example.com",
		},
		{
			name: "https with path, query and fragment",
			url:  "https://example.com/a/b?x=1#top",
		},
		{
			name: "uppercase scheme",
			url:  "HTTPS://EXAMPLE.COM",
		},
		{
			name: "with port",
			url:  "http://localhost:8080/path",
		},
		{
			name: "max length",
			url:  prefix + strings.Repeat("a", maxURLLength-len(prefix)),
		},
		{
			name:    "over max length",
			url:     prefix + strings.Repeat("a", maxURLLength-len(prefix)+1),
			wantErr: true,
		},
		{
			name:    "empty",
			url:     "",
			wantErr: true,
		},
		{
			name:    "no scheme",
			url:     "example.com",
			wantErr: true,
		},
		{
			name:    "scheme without host",
			url:     "https://",
			wantErr: true,
		},
		{
			name:    "relative path",
			url:     "/just/a/path",
			wantErr: true,
		},
		{
			name:    "ftp scheme",
			url:     "ftp://example.com",
			wantErr: true,
		},
		{
			name:    "javascript scheme",
			url:     "javascript:alert(1)",
			wantErr: true,
		},
		{
			name:    "file scheme",
			url:     "file:///etc/passwd",
			wantErr: true,
		},
		{
			name:    "space in host",
			url:     "http://exa mple.com",
			wantErr: true,
		},
		{
			name:    "garbage",
			url:     "ht!tp://%%%",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.url)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateURL(%.60q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}
