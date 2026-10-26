package app_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

func TestArch_GoSecurityAndNormalizationVersions(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(moduleRoot(t), "go.mod"))
	require.NoError(t, err)
	mod, err := modfile.Parse("go.mod", data, nil)
	require.NoError(t, err)
	require.NotNil(t, mod.Go)
	require.GreaterOrEqual(t, semver.Compare("v"+mod.Go.Version, "v1.27.3"), 0,
		"release builds must include the October 2026 Go security fixes")

	floors := map[string]string{
		"golang.org/x/net":                                      "v0.61.0",
		"google.golang.org/grpc":                                "v1.83.2",
		"github.com/hashicorp/go-getter":                        "v1.8.6",
		"github.com/ulikunitz/xz":                               "v0.5.15",
		"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream": "v1.7.8",
		"github.com/aws/aws-sdk-go-v2/service/s3":               "v1.97.3",
	}
	resolved := make(map[string]string)
	for _, req := range mod.Require {
		resolved[req.Mod.Path] = req.Mod.Version
	}
	for _, rep := range mod.Replace {
		if _, guarded := floors[rep.Old.Path]; !guarded && rep.Old.Path != "golang.org/x/text" {
			continue
		}
		if rep.Old.Version != "" && rep.Old.Version != resolved[rep.Old.Path] {
			continue
		}
		require.Equal(t, rep.Old.Path, rep.New.Path, "security checks require the upstream module")
		resolved[rep.Old.Path] = rep.New.Version
	}
	for path, floor := range floors {
		require.True(t, semver.IsValid(resolved[path]), path)
		require.GreaterOrEqual(t, semver.Compare(resolved[path], floor), 0,
			"%s must retain its security fixes", path)
	}
	require.Equal(t, "v0.28.0", resolved["golang.org/x/text"],
		"normalization is consensus state: dependency bumps must not change its implementation")
}
