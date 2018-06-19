package wat

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestTrainMatch(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ws := f.watInit()

	f.write("src/github.com/fake/repo/repo_test.go", `
package repo

import "testing"

func TestOne(t *testing.T) {}
`)

	cmd := WatCommand{
		Command:     "go test github.com/fake/repo",
		FilePattern: "src/github.com/fake/repo/*.go",
	}
	cmds := []WatCommand{cmd}
	groups, err := trainAt(context.Background(), ws, cmds)
	if err != nil {
		t.Fatalf("Train: %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("Expected 1 groups. Actual: %+v", groups)
	}

	for _, g := range groups {
		if len(g.Logs) != 1 {
			t.Fatalf("Expected each group to have 1 log. Actual: %+v", g.Logs)
		}
	}
}

func TestTrainToFile(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ws := f.watInit()

	f.write("src/github.com/fake/repo/repo_test.go", `
package repo

import "testing"

func TestOne(t *testing.T) {}
`)

	cmd := WatCommand{
		Command:     "go test github.com/fake/repo",
		FilePattern: "src/github.com/fake/repo/*.go",
	}
	cmds := []WatCommand{cmd}
	_, err := Train(context.Background(), ws, cmds, 0)
	if err != nil {
		t.Fatalf("Train: %v", err)
	}

	// Make sure Train() wrote to a file.
	groups, err := ReadCmdLogGroups(ws)
	if err != nil {
		t.Fatal(err)
	}

	if len(groups) != 1 {
		t.Fatalf("Expected 1 groups. Actual: %+v", groups)
	}

	for _, g := range groups {
		if len(g.Logs) != 1 {
			t.Fatalf("Expected each group to have 1 log. Actual: %+v", g.Logs)
		}
	}

	// Make sure Train() re-uses the files on-disk.
	groups2, err := Train(context.Background(), ws, []WatCommand{}, time.Hour)
	if err != nil {
		t.Fatalf("Train: %v", err)
	}

	if !reflect.DeepEqual(groups, groups2) {
		t.Fatalf("Train must not have reused files on disk because the results were different:\n%v\n%v", groups, groups2)
	}
}

func TestTrainNoMatch(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ws := f.watInit()

	cmd := WatCommand{
		Command:     "go test github.com/fake/repo",
		FilePattern: "src/github.com/fake/repo/*.go",
	}
	cmds := []WatCommand{cmd}
	groups, err := trainAt(context.Background(), ws, cmds)
	if err != nil {
		t.Fatalf("Train: %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("Expected 1 groups. Actual: %+v", groups)
	}

	for _, g := range groups {
		if len(g.Logs) != 1 {
			t.Fatalf("Expected each group to have 1 log. Actual: %+v", g.Logs)
		}
	}
}

func TestTrainFuzz(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()

	f.write("src/github.com/fake/repo/repo.go", `
package repo

func F() bool { return false }
`)
	f.write("src/github.com/fake/repo/repo_test.go", `
package repo

import "testing"

func TestF(t *testing.T) {
  if F() {
    t.Fatal("Expected false, got true")
  }
}
`)

	cmd := WatCommand{
		Command:     "go test github.com/fake/repo",
		FilePattern: "src/github.com/fake/repo/*.go",
	}
	cmds := []WatCommand{cmd}
	os.Setenv("GOPATH", f.root.Path())

	group, err := fuzzAndRun(context.Background(), cmds, f.root.Path(), "src/github.com/fake/repo/repo.go")
	if err != nil {
		t.Fatalf("fuzzAndRun: %v", err)
	}

	if len(group.Logs) != 1 || group.Logs[0].Success {
		t.Fatalf("Expected failure. Actual log group: %+v", group)
	}

	group, err = fuzzAndRun(context.Background(), cmds, f.root.Path(), "src/github.com/fake/repo/repo_test.go")
	if err != nil {
		t.Fatalf("fuzzAndRun: %v", err)
	}

	if len(group.Logs) != 1 || !group.Logs[0].Success {
		t.Fatalf("Expected success. Actual log group: %+v", group)
	}
}

type fuzzCase struct {
	original string
	expected string
}

func TestFuzz(t *testing.T) {
	cases := []fuzzCase{
		fuzzCase{"x := false", "x := true"},
		fuzzCase{"x := 0", "x := 1"},
		fuzzCase{"x := 100", "x := 100"},
		fuzzCase{"x := falsey", "x := falsey"},
	}

	for i, c := range cases {
		c := c
		t.Run(fmt.Sprintf("TestFuzz%d", i), func(t *testing.T) {
			actual := string(fuzz([]byte(c.original)))
			if c.expected != actual {
				t.Fatalf("Expected fuzz(%q) = %q. Actual: %q", c.original, c.expected, actual)
			}
		})
	}
}
