package main

import (
	"path/filepath"
	"testing"

	"github.com/compnew2006/whatomate/internal/licenseissuer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultStudioDataDir(t *testing.T) {
	dir, err := licenseissuer.DefaultStudioDataDir()

	require.NoError(t, err)
	assert.NotEmpty(t, dir)
	assert.Contains(t, dir, licenseissuer.DefaultStudioDirName)
}

func TestDefaultStudioDataDirContainsHome(t *testing.T) {
	dir, err := licenseissuer.DefaultStudioDataDir()

	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(dir))
}
