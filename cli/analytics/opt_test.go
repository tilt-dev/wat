package analytics_test

import (
	"os"
	"testing"

	"github.com/windmilleng/wat/cli/analytics"
	"github.com/windmilleng/wat/os/temp"
)

func TestSetOptStr(t *testing.T) {
	oldWindmillDir := os.Getenv("WINDMILL_DIR")
	defer os.Setenv("WINDMILL_DIR", oldWindmillDir)
	tmpdir, err := temp.NewDir("TestOpt")
	if err != nil {
		t.Fatalf("Error making temp dir: %v", err)
	}

	f := setup(t)

	os.Setenv("WINDMILL_DIR", tmpdir.Path())

	f.assertOptStatus(analytics.OptDefault)

	analytics.SetOptStr("opt-in")
	f.assertOptStatus(analytics.OptIn)

	analytics.SetOptStr("opt-out")
	f.assertOptStatus(analytics.OptOut)

	analytics.SetOptStr("in")
	f.assertOptStatus(analytics.OptIn)

	analytics.SetOptStr("out")
	f.assertOptStatus(analytics.OptOut)

	analytics.SetOptStr("foo")
	f.assertOptStatus(analytics.OptDefault)
}

func TestSetOpt(t *testing.T) {
	oldWindmillDir := os.Getenv("WINDMILL_DIR")
	defer os.Setenv("WINDMILL_DIR", oldWindmillDir)
	tmpdir, err := temp.NewDir("TestOpt")
	if err != nil {
		t.Fatalf("Error making temp dir: %v", err)
	}

	f := setup(t)

	os.Setenv("WINDMILL_DIR", tmpdir.Path())

	f.assertOptStatus(analytics.OptDefault)

	analytics.SetOpt(analytics.OptIn)
	f.assertOptStatus(analytics.OptIn)

	analytics.SetOpt(analytics.OptOut)
	f.assertOptStatus(analytics.OptOut)

	analytics.SetOpt(99999)
	f.assertOptStatus(analytics.OptDefault)
}

type fixture struct {
	t *testing.T
}

func setup(t *testing.T) *fixture {
	return &fixture{t: t}
}

func (f *fixture) assertOptStatus(expected analytics.Opt) {
	actual, err := analytics.OptStatus()
	if err != nil {
		f.t.Fatal(err)
	}
	if actual != expected {
		f.t.Errorf("got opt status %v, expected %v", actual, expected)
	}
}
