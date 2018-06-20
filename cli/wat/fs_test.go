package wat

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestWatRootCurDir(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()

	// wat root at f.root
	f.watInit()

	rt, err := watRoot(f.root.Path())
	if err != nil {
		f.t.Fatalf("Got error: %v", err)
	}
	if rt != f.root.Path() {
		f.t.Fatalf("Expected wat root '%s', got '%s'", f.root.Path(), rt)
	}
}

func TestWatRootParentDir(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()

	// wat root at f.root
	f.watInit()

	f.writeFile("foo/bar/baz")
	childDir := filepath.Join(f.root.Path(), "foo/bar")

	rt, err := watRoot(childDir)
	if err != nil {
		f.t.Fatalf("Got error: %v", err)
	}
	if rt != f.root.Path() {
		f.t.Fatalf("Expected wat root '%s', got '%s'", f.root.Path(), rt)
	}
}

func TestWatRootNotExists(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()

	_, err := watRoot(f.root.Path())
	if err == nil || err != ErrNoWatRoot {
		f.t.Fatalf("Expected 'no wat root found' error but instead got error: %v", err)
	}
}

func TestRead(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ws := f.watInit()

	for i := 0; i < 5; i++ {
		f.write(filepath.Join(kWatDirName, strconv.Itoa(i)), strconv.Itoa(i))
	}

	for i := 0; i < 5; i++ {
		contents, err := ws.Read(strconv.Itoa(i))
		if err != nil {
			f.t.Fatalf("[wat.Read] Got error reading file '%d': %v", i, err)
		}
		if string(contents) != strconv.Itoa(i) {
			f.t.Fatalf("Expected file '%d' to have contents '%d'; got '%s'", i, i, contents)
		}
	}
}

func TestWrite(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ws := f.watInit()

	for i := 0; i < 5; i++ {
		err := ws.Write(strconv.Itoa(i), strconv.Itoa(i))
		if err != nil {
			f.t.Fatalf("Got error writing file '%d': %v", i, err)
		}
	}

	for i := 0; i < 5; i++ {
		expectedPath := filepath.Join(kWatDirName, strconv.Itoa(i))
		contents, err := ioutil.ReadFile(expectedPath)
		if err != nil {
			f.t.Fatalf("[ioutil.ReadFile]: %v", err)
		}
		if string(contents) != strconv.Itoa(i) {
			f.t.Fatalf("Expected file '%d' to have contents '%d'; got '%s'", i, i, contents)
		}
	}
}

func TestWriteNested(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ws := f.watInit()

	for i := 0; i < 5; i++ {
		path := filepath.Join(strconv.Itoa(i), strconv.Itoa(i))
		err := ws.Write(path, strconv.Itoa(i))
		if err != nil {
			f.t.Fatalf("[wat.write]: %v", err)
		}
	}

	for i := 0; i < 5; i++ {
		expectedPath := filepath.Join(kWatDirName, strconv.Itoa(i), strconv.Itoa(i))
		contents, err := ioutil.ReadFile(expectedPath)
		if err != nil {
			f.t.Fatalf("[ioutil.ReadFile] %v", err)
		}
		if string(contents) != strconv.Itoa(i) {
			f.t.Fatalf("Expected file '%d' to have contents '%d'; got '%s'",
				i, i, contents)
		}
	}
}

func TestAppend(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ws := f.watInit()

	fullPath := filepath.Join(kWatDirName, "test")
	f.write(fullPath, "foo")

	err := ws.Append("test", "bar")
	if err != nil {
		f.t.Fatalf("[Append] %v", err)
	}
	err = ws.Append("test", "baz")
	if err != nil {
		f.t.Fatalf("[Append] %v", err)
	}

	expectedContents := "foobarbaz"
	contents, err := ioutil.ReadFile(fullPath)
	if err != nil {
		f.t.Fatalf("[ioutil.ReadFile]: %v", err)
	}
	if string(contents) != expectedContents {
		f.t.Fatalf("Expected contents '%s'; got '%s'", expectedContents, contents)
	}
}

func TestAppendNonexistentFile(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ws := f.watInit()

	// Appending to file that doesn't exist yet
	err := ws.Append("test", "foobarbaz")

	fullPath := filepath.Join(kWatDirName, "test")
	expectedContents := "foobarbaz"
	contents, err := ioutil.ReadFile(fullPath)
	if err != nil {
		f.t.Fatalf("[ioutil.ReadFile]: %v", err)
	}
	if string(contents) != expectedContents {
		f.t.Fatalf("Expected contents '%s'; got '%s'", expectedContents, contents)
	}
}

func TestAppendLine(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ws := f.watInit()

	fullPath := filepath.Join(kWatDirName, "test")
	f.write(fullPath, `foo
`)

	err := ws.AppendLine("test", "bar")
	if err != nil {
		f.t.Fatalf("[AppendLine] %v", err)
	}
	err = ws.AppendLine("test", "baz")
	if err != nil {
		f.t.Fatalf("[AppendLine] %v", err)
	}

	expectedContents := `foo
bar
baz
`
	contents, err := ioutil.ReadFile(fullPath)
	if err != nil {
		f.t.Fatalf("[ioutil.ReadFile]: %v", err)
	}
	if string(contents) != expectedContents {
		f.t.Fatalf("Expected contents '%s'; got '%s'", expectedContents, contents)
	}
}

func TestCmdLogsToFile(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ws := f.watInit()

	groupA := CommandLogGroup{
		Context: LogContext{
			RecentEdits: []string{"a.txt", "b.txt"},
			StartTime:   time.Now(),
			Source:      LogSourceBootstrap,
		},
		Logs: []CommandLog{
			{
				Command:  "cat foo",
				Success:  true,
				Duration: time.Minute,
			},
		},
	}
	groupB := CommandLogGroup{
		Context: LogContext{
			RecentEdits: []string{"a.txt", "c.txt"},
			StartTime:   time.Now(),
			Source:      LogSourceBootstrap,
		},
		Logs: []CommandLog{
			{
				Command:  "cat bar",
				Success:  false,
				Duration: time.Minute * 2,
			},
		},
	}

	err := CmdLogGroupsToFile(ws, []CommandLogGroup{groupA, groupB})
	if err != nil {
		f.t.Fatalf("[CmdLogGroupsToFile]: %v", err)
	}

	expectedContents := fmt.Sprintf("%s\n%s\n", MustJson(groupA), MustJson(groupB))
	contents, err := ioutil.ReadFile(filepath.Join(kWatDirName, fnameCmdLog))
	if err != nil {
		f.t.Fatalf("[ioutil.ReadFile]: %v", err)
	}
	if string(contents) != expectedContents {
		f.t.Fatalf("Expected contents '%s'; got '%s'", expectedContents, contents)
	}
}
