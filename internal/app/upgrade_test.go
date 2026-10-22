package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/google/go-github/v86/github"
	"github.com/stretchr/testify/require"
)

func TestUpgradeDoesNotReplaceEqualOrNewerVersions(t *testing.T) {
	for _, current := range []string{"v0.1.0", "v0.2.0"} {
		t.Run(current, func(t *testing.T) {
			source := &upgradeSource{}
			updater, err := selfupdate.NewUpdater(selfupdate.Config{
				Source: source, Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
			})
			require.NoError(t, err)
			var output bytes.Buffer
			err = upgradeBinary(t.Context(), &output, current, filepath.Join(t.TempDir(), "appsum"), updater)
			require.NoError(t, err)
			require.False(t, source.downloaded, "an equal or newer version must never be replaced")
			require.Contains(t, output.String(), "already up to date")
		})
	}
}

func TestUpgradeUsesHomebrewForCellarInstall(t *testing.T) {
	for _, status := range []int{0, 1} {
		t.Run(fmt.Sprintf("exit-%d", status), func(t *testing.T) {
			prefix := t.TempDir()
			bin := filepath.Join(prefix, "bin")
			require.NoError(t, os.MkdirAll(bin, 0o755))
			script := fmt.Sprintf("#!/bin/sh\n[ \"$*\" = 'upgrade sadiksaifi/tap/appsum' ] || exit 99\necho 'brew stdout'\necho 'brew stderr' >&2\nexit %d\n", status)
			require.NoError(t, os.WriteFile(filepath.Join(bin, "brew"), []byte(script), 0o755))
			path := filepath.Join(prefix, "Cellar", "appsum", "0.1.0", "bin", "appsum")
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
			require.NoError(t, os.WriteFile(path, []byte("original binary"), 0o755))

			var output bytes.Buffer
			err := upgradeBinary(t.Context(), &output, "dev", path, nil)
			if status == 0 {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "brew upgrade")
			}
			require.Contains(t, output.String(), "brew stdout")
			require.Contains(t, output.String(), "brew stderr")
			contents, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, "original binary", string(contents))
		})
	}
}

// Only the release-service boundary is replaced; selection uses the real updater.
type upgradeSource struct{ downloaded bool }

func (*upgradeSource) ListReleases(context.Context, selfupdate.Repository) ([]selfupdate.SourceRelease, error) {
	return []selfupdate.SourceRelease{selfupdate.NewGitHubRelease(&github.RepositoryRelease{
		TagName: new("v0.1.0"),
		Assets: []*github.ReleaseAsset{
			{ID: new(int64(1)), Name: new(fmt.Sprintf("appsum_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH))},
			{ID: new(int64(2)), Name: new("checksums.txt")},
		},
	})}, nil
}

func (s *upgradeSource) DownloadReleaseAsset(context.Context, *selfupdate.Release, int64) (io.ReadCloser, error) {
	s.downloaded = true
	return nil, fmt.Errorf("unexpected download")
}
