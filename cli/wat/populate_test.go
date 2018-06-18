package wat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestPopulate(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ws := f.watInit()

	oldBuiltins := builtins
	defer func() { builtins = oldBuiltins }()
	builtins = []plugin{pluginAB{}}

	// Instead of paths to executables, this "plugin" just echos results straight to stdout.
	f.write(filepath.Join(kWatDirName, fnameUserPlugins),
		fmt.Sprintf("echo '%s'", MustJson([]WatCommand{cmdC})))

	cmds, err := List(context.Background(), ws, 0)
	if err != nil {
		f.t.Fatal("Populate", err)
	}
	count := len(cmds.Commands)
	if count != 3 {
		f.t.Fatalf("Expected 3 commands to be populated, got %d", count)
	}
	exists, err := ws.Exists(fnameList)
	if err != nil {
		f.t.Fatal("Exists", err)
	}
	if !exists {
		f.t.Fatal("Expect list file to exist")
	}

}

// Make sure we don't explode if the list file doesn't exist yet.
func TestCommandListNotExists(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ws := f.watInit()
	ctx := context.Background()
	cmdListOut, err := List(ctx, ws, listTTL)
	if err != nil {
		f.t.Fatal("List()", err)
	}

	assertCommandList(f.t, CommandList{}, cmdListOut)
}

func TestCommandListWriteAndRead(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ws := f.watInit()
	ctx := context.Background()

	cmds := []WatCommand{
		WatCommand{
			Command:     "echo hello",
			FilePattern: "abc",
		},
	}
	cmdListIn := CommandList{
		time.Now(),
		cmds,
	}

	err := cmdListIn.toFile(ws)
	if err != nil {
		f.t.Fatal("cmdList.toFile", err)
	}

	cmdListOut, err := List(ctx, ws, listTTL)
	if err != nil {
		f.t.Fatal("List()", err)
	}

	assertCommandList(f.t, cmdListIn, cmdListOut)
}

func TestListRepopulatesStaleInfo(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ws := f.watInit()

	oldData := CommandList{
		Timestamp: time.Now().Add(time.Hour * -60),
		Commands: []WatCommand{
			WatCommand{
				Command:     "echo hello",
				FilePattern: "abc",
			},
		},
	}
	err := oldData.toFile(ws)
	if err != nil {
		f.t.Fatal("CommandList.toFile()", err)
	}

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

	actual, err := List(f.ctx, ws, listTTL)

	expectedCommands := []WatCommand{
		WatCommand{
			FilePattern: "src/github.com/fake/repo/*",
			Command:     "go test github.com/fake/repo",
		},
	}

	if !reflect.DeepEqual(actual.Commands, expectedCommands) {
		f.t.Fatalf("expected: %v, actual: %v", expectedCommands, actual)
	}
	if !actual.Timestamp.After(oldData.Timestamp) {
		f.t.Fatal("Timestamp was not updated.")
	}
}
