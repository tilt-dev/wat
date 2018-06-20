package wat

import (
	"path/filepath"
	"testing"

	"github.com/windmilleng/wat/os/ospath"
)

func TestInit(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()

	err := Init(f.a, f.root.Path())
	if err != nil {
		f.t.Fatalf("Error calling init: %v", err)
	}

	expectedPath := filepath.Join(f.root.Path(), kWatDirName)
	if !ospath.IsDir(expectedPath) {
		f.t.Fatalf("Expected directory '%s' does not exist", expectedPath)
	}
}
