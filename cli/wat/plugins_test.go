package wat

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

type pluginAB struct{}

func (pluginAB) name() string { return "pluginAB" }
func (pluginAB) run(ctx context.Context, root string) ([]WatCommand, error) {
	return []WatCommand{cmdA, cmdB}, nil
}

type pluginC struct{}

func (pluginC) name() string { return "pluginC" }
func (pluginC) run(ctx context.Context, root string) ([]WatCommand, error) {
	return []WatCommand{cmdC}, nil
}

type pluginErr struct{}

func (pluginErr) name() string { return "pluginErr" }
func (pluginErr) run(ctx context.Context, root string) ([]WatCommand, error) {
	return []WatCommand{}, errors.New("Oh noes!")
}

func TestBuiltinGoList(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()

	f.write("src/github.com/fake/repo/repo.go", `
package repo

func One() { return 1 }
`)
	f.write("src/github.com/fake/repo/repo_test.go", `
package repo

import "testing"

func TestOne(t *testing.T) {}
`)

	os.Setenv("GOPATH", f.root.Path())

	cmds, err := PluginGo{}.run(context.Background(), f.root.Path())
	if err != nil {
		f.t.Fatalf("populateAt: %v", err)
	}

	if len(cmds) != 1 {
		f.t.Fatalf("Expected 1 command, got %d", len(cmds))
	}

	expected := WatCommand{
		FilePattern: "src/github.com/fake/repo/*",
		Command:     "go test github.com/fake/repo",
	}
	if cmds[0] != expected {
		f.t.Fatalf("Expected %+v. Actual: %+v", expected, cmds[0])
	}
}

func TestRunBuiltinPlugins(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ws := f.watInit()

	actual := runPlugins(context.Background(), ws,
		[]plugin{pluginAB{}, pluginErr{}, pluginC{}})

	// NOTE: we expect to ignore the err thrown by pluginErr and continue running the rest
	expected := []WatCommand{cmdA, cmdB, cmdC}

	if !reflect.DeepEqual(actual, expected) {
		f.t.Fatalf("Expected command list %+v. Actual: %+v", expected, actual)
	}
}

func TestGetUserPlugins(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ws := f.watInit()
	f.write(filepath.Join(kWatDirName, fnameUserPlugins), "foo\nbar\nbaz")

	actual, err := getUserPlugins(ws)
	if err != nil {
		f.t.Fatal("getUserPlugins:", err)
	}

	expected := []plugin{
		userPlugin{cmd: "foo"},
		userPlugin{cmd: "bar"},
		userPlugin{cmd: "baz"},
	}
	if !reflect.DeepEqual(actual, expected) {
		f.t.Fatalf("Expected to retrieve user plugins %+v, got %+v", expected, actual)
	}
}

func TestGetUserPluginsNoFile(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ws := f.watInit()

	actual, err := getUserPlugins(ws)
	if err != nil {
		f.t.Fatal("getUserPlugins:", err)
	}

	if len(actual) != 0 {
		f.t.Fatalf("Expected to retrieve 0 user plugins, instead got %d: %+v", len(actual), actual)
	}
}

func TestRunUserPlugins(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ws := f.watInit()

	// Instead of paths to executables, these "plugins" just echo results straight to stdout.
	f.write(filepath.Join(kWatDirName, fnameUserPlugins),
		fmt.Sprintf(`echo '%s'
echo 'not valid json'
echo '%s'`, MustJson([]WatCommand{cmdA}), MustJson([]WatCommand{cmdB, cmdC})))

	actual, err := RunUserPlugins(context.Background(), ws)
	if err != nil {
		f.t.Fatal("RunUserPlugins:", err)
	}

	// NOTE: we expect to ignore the err from invalid json and continue running the rest
	expected := []WatCommand{cmdA, cmdB, cmdC}
	if !reflect.DeepEqual(actual, expected) {
		f.t.Fatalf("Expected to retrieve user plugins %+v, got %+v", expected, actual)
	}
}

func TestRunUserPluginReturnsErr(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()

	plugin := userPlugin{cmd: "not-a-valid-command"}

	_, err := plugin.run(context.Background(), f.root.Path())
	if err == nil {
		f.t.Fatal("Expected error b/c of bad command, but none returned")
	}
}

func TestRunUserPluginReturnsTimeoutErr(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	plugin := userPlugin{cmd: "sleep 2"}

	_, err := plugin.run(ctx, f.root.Path())
	if err == nil {
		f.t.Fatal("Expected error b/c of timeout, but none returned")
	}
}
