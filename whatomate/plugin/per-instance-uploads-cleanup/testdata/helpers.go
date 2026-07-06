package testdata

import (
	"log/slog"
	"testing"

	"github.com/compnew2006/whatomate/plugin/per-instance-uploads-cleanup"
	"github.com/compnew2006/whatomate/test/testutil"
)

func SetupTestEnv(t *testing.T) *perinstanceuploadscleanup.Plugin {
	t.Helper()
	db := testutil.SetupTestDB(t)
	rdb := testutil.SetupTestRedis(t)
	p := &perinstanceuploadscleanup.Plugin{}
	if err := p.Init(nil, db, rdb, slog.Default()); err != nil {
		t.Fatalf("plugin init failed: %v", err)
	}
	return p
}
