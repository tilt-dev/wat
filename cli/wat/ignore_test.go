package wat

import (
	"testing"
)

func TestWatIgnoreOrDummy(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()

	// Nil path --> dummy
	ignore := watIgnoreOrDummy("")
	if _, ok := ignore.(dummyWatIgnore); !ok {
		f.t.Fatal("watIgnoreOrDummy called with nil path should return dummyWatIgnore")
	}

	// Path to non-existent file --> dummy
	ignore = watIgnoreOrDummy("file/does/not/exist")
	if _, ok := ignore.(dummyWatIgnore); !ok {
		f.t.Fatal("watIgnoreOrDummy called with nonexistant filepath should return dummyWatIgnore")
	}

	// Real and parse-able ignore file --> real Ignore object
	path := "myIgnore"
	f.write(path, "foo")
	ignore = watIgnoreOrDummy(path)
	if _, ok := ignore.(dummyWatIgnore); ok {
		f.t.Fatal("watIgnoreOrDummy called with valid ignorePath should NOT return dummyWatIgnore")
	}
	if !ignore.Match("foo", false) {
		f.t.Fatal("this ignore should match 'foo'")
	}
}

func TestMakeWatIgnoreContentsWithDefaults(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()

	defaults := []string{"foo", "bar", "baz"}
	actual, err := makeWatIgnoreContents(f.root.Path(), defaults)
	if err != nil {
		f.t.Fatal("makeWatIgnoreContents", err)
	}

	expected := "foo\nbar\nbaz"
	if actual != expected {
		f.t.Fatalf("expected watIgnore actual:\n%s\nGot:\n%s", expected, actual)
	}
}

func TestMakeWatIgnoreContentsFromGitIgnore(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()

	gitIgnore := "hello\nworld"
	f.write(fnameGitIgnore, gitIgnore)

	actual, err := makeWatIgnoreContents(f.root.Path(), nil)
	if err != nil {
		f.t.Fatal("makeWatIgnoreContents", err)
	}

	if actual != gitIgnore {
		f.t.Fatalf("expected watIgnore actual:\n%s\nGot:\n%s", gitIgnore, actual)
	}
}

func TestMakeWatIgnoreContentsDedupe(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()

	defaults := []string{"foo", "bar", "baz"}
	gitIgnore := `foo
hello
world
bar`
	f.write(fnameGitIgnore, gitIgnore)

	actual, err := makeWatIgnoreContents(f.root.Path(), defaults)
	if err != nil {
		f.t.Fatal("makeWatIgnoreContents", err)
	}

	expected := `foo
hello
world
bar

baz`

	if actual != expected {
		f.t.Fatalf("expected watIgnore actual:\n%s\nGot:\n%s", expected, actual)
	}

}
