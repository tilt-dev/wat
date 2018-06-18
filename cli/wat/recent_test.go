package wat

import (
	"testing"

	"time"

	"sort"

	"reflect"

	"fmt"
)

func TestWalk(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()

	fileset := map[string]bool{
		"foo":       true,
		"bar":       true,
		"baz":       true,
		"beep/boop": true,
		"beep/bop":  true,
	}
	for file := range fileset {
		f.writeFile(file)
	}

	files, err := walkDir(f.root.Path())
	if err != nil {
		f.t.Fatalf("Error testing walk: %v", err)
	}

	f.assertExpectedFiles(fileset, files)
}

func TestSort(t *testing.T) {
	now := fileInfo{name: "now", modTime: time.Now()}
	yesterday := fileInfo{name: "yesterday", modTime: time.Now().Add(-24 * time.Hour)}
	tomorrow := fileInfo{name: "tomorrow", modTime: time.Now().Add(24 * time.Hour)}
	files := []fileInfo{now, yesterday, tomorrow}

	sort.Sort(fileInfos(files))

	expected := []fileInfo{yesterday, now, tomorrow}

	if !reflect.DeepEqual(expected, files) {
		t.Fatalf("Expected %v, got %v", expected, files)
	}
}

func TestIgnore(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()

	filesToExpected := map[string]bool{
		"foo/a":           false, // ignore "a"
		"foo/b":           true,
		"foo/c":           true,
		"bar/a":           false, // ignore "a"
		"bar/b":           true,
		"thingA.yes":      true,
		"thingB.no":       false, // ignore "*.no"
		"stuff/thingC.no": false, // ignore "*.no"
		"hello":           true,
		"world":           true,
		"beep/boop/bzz":   false, // ignore "beep"
		"beep/boop/bees":  false, // ignore "beep"
		"blah/beep/bork":  false, // ignore "beep"
		"blah/blorp":      true,
	}
	for file := range filesToExpected {
		f.writeFile(file)
	}

	f.write(fnameWatIgnore, fmt.Sprintf(`*.no
a
beep
%s`, fnameWatIgnore))

	files, err := walkDir(f.root.Path())
	if err != nil {
		f.t.Fatalf("Error testing walk: %v", err)
	}

	f.assertExpectedFiles(filesToExpected, files)
}

// Make sure that WalkDir (public func.) looks in the right place for
// .watignore (should be at wat root)
func TestWalkDirFindsIgnore(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()
	ws := f.watInit()

	filesToExpected := map[string]bool{
		"foo": false, // ignore "foo"
		"bar": true,
	}
	for file := range filesToExpected {
		f.writeFile(file)
	}

	f.write(fnameWatIgnore, fmt.Sprintf("foo\n%s", fnameWatIgnore))

	files, err := ws.WalkRoot()
	if err != nil {
		f.t.Fatalf("Error testing walk: %v", err)
	}

	f.assertExpectedFiles(filesToExpected, files)
}
