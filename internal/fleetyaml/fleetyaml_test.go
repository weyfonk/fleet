//go:build !windows
// +build !windows

package fleetyaml_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rancher/fleet/internal/fleetyaml"
)

func TestBundleYaml(t *testing.T) {
	a := assert.New(t)
	for _, path := range []string{"/foo", "foo", "/foo/", "foo/", "../foo/bar"} {

		// Test both the primary extension and the fallback extension.
		for _, fullPath := range []string{fleetyaml.GetFleetYamlPath(path, false), fleetyaml.GetFleetYamlPath(path, true)} {
			a.True(fleetyaml.IsFleetYaml(filepath.Base(fullPath)))
			a.True(fleetyaml.IsFleetYamlSuffix(fullPath))
		}
	}

	// Test expected failure payloads.
	for _, fullPath := range []string{"fleet.yaaaaaaaaaml", "", ".", "weakmonkey.yaml", "../fleet.yaaaaml"} {
		a.False(fleetyaml.IsFleetYaml(filepath.Base(fullPath)))
		a.False(fleetyaml.IsFleetYamlSuffix(fullPath))
	}
}
