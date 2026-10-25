package app_test

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"cosmossdk.io/x/upgrade/plan"
	"github.com/stretchr/testify/require"
	"github.com/ulikunitz/xz"
)

// Keep the SDK/go-getter/archive integration covered with local, checksummed
// artifacts. The payload is never executed and no external URL is contacted.
func TestUpgradeDownloadSecurityCompatibility(t *testing.T) {
	payload := []byte("structsd download fixture\n")
	var archive bytes.Buffer
	compressor, err := xz.NewWriter(&archive)
	require.NoError(t, err)
	writer := tar.NewWriter(compressor)
	require.NoError(t, writer.WriteHeader(&tar.Header{Name: "bin/structsd", Mode: 0755, Size: int64(len(payload))}))
	_, err = writer.Write(payload)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.NoError(t, compressor.Close())
	for _, fixture := range []struct {
		name string
		data []byte
	}{
		{"structsd", payload}, {"release.tar.xz", archive.Bytes()},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(fixture.data)
			}))
			defer server.Close()
			url := fmt.Sprintf("%s/%s?checksum=sha256:%x", server.URL, fixture.name, sha256.Sum256(fixture.data))
			destination := filepath.Join(t.TempDir(), "upgrade")
			require.NoError(t, plan.DownloadUpgrade(destination, url, "structsd"))
			path := filepath.Join(destination, "bin", "structsd")
			got, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, payload, got)
			info, err := os.Stat(path)
			require.NoError(t, err)
			require.EqualValues(t, 0111, info.Mode().Perm()&0111)
			badURL := fmt.Sprintf("%s/%s?checksum=sha256:%064x", server.URL, fixture.name, 0)
			require.Error(t, plan.DownloadUpgrade(filepath.Join(t.TempDir(), "bad"), badURL, "structsd"))
		})
	}
}
