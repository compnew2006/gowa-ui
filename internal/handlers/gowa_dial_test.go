package handlers_test

import (
	"testing"

	"github.com/compnew2006/gowa-ui/internal/config"
	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/stretchr/testify/assert"
)

func TestGowaDialBaseURL(t *testing.T) {
	t.Parallel()
	internal := "http://127.0.0.1:3000"
	tests := []struct {
		name string
		cfg  *config.Config
		in   string
		want string
	}{
		{
			name: "nil config passes URL through",
			cfg:  nil,
			in:   "https://gowa.example.com",
			want: "https://gowa.example.com",
		},
		{
			name: "no override passes URL through",
			cfg:  &config.Config{},
			in:   "https://gowa.example.com",
			want: "https://gowa.example.com",
		},
		{
			name: "override wins and drops trailing slash",
			cfg:  &config.Config{GOWA: config.GOWAConfig{InternalBaseURL: internal + "/"}},
			in:   "https://gowa.example.com",
			want: internal,
		},
		{
			name: "override applies to empty URL (legacy accounts)",
			cfg:  &config.Config{GOWA: config.GOWAConfig{InternalBaseURL: internal}},
			in:   "",
			want: internal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, handlers.GowaDialBaseURL(tt.cfg, tt.in))
		})
	}
}
