package wat

import (
	"fmt"
	"os"

	"strings"

	"io/ioutil"

	"path/filepath"

	"github.com/monochromegane/go-gitignore"
)

var defaultIgnores = []string{
	"# WAT DEFAULT IGNORES", // just a comment...
	".git",
	".wat",

	// language-specific
	"node_modules",
	"vendor",
	"*.pyc",

	// editor-specific
	".idea",

	// ...windmill specific -- TODO: remove
	"frontend",
	"build",
	"sphinx",
}

func MakeWatIgnore(root string) error {
	contents, err := makeWatIgnoreContents(root, defaultIgnores)
	if err != nil {
		return err
	}

	watIgnorePath := filepath.Join(root, fnameWatIgnore)
	return ioutil.WriteFile(watIgnorePath, []byte(contents), permFile)
}

func makeWatIgnoreContents(root string, defaults []string) (string, error) {
	gitIgnorePath := filepath.Join(root, fnameGitIgnore)

	// populate .watignore w/ contents of .gitignore
	existing, err := ioutil.ReadFile(gitIgnorePath)
	if err != nil {
		if os.IsNotExist(err) {
			// .gitignore doesn't exist, populate .watignore with defaults stuff
			return strings.Join(defaults, "\n"), nil
		}
		return "", fmt.Errorf("ioutil.ReadFile(%s): %v", gitIgnorePath, err)
	}
	lines := strings.Split(string(existing), "\n")

	// add defaults (deduped against existing lines)
	defaultsToAdd := dedupeAgainst(defaults, lines)
	if len(defaultsToAdd) > 0 {
		lines = append(lines, "") // line break
		lines = append(lines, defaultsToAdd...)
	}

	return strings.Join(lines, "\n"), nil
}

func watIgnoreOrDummy(ignorePath string) matcher {
	// if file doesn't exist (or we can't parse it for some reason), return a dummy
	ignore, err := gitignore.NewGitIgnore(ignorePath)
	if err != nil {
		if !os.IsNotExist(err) {
			// File exists, something weird is going on
			fmt.Fprintf(os.Stderr,
				"ERROR: ignore file at %s exists, but err calling NewGitIgnore: %v",
				ignorePath, err)
		}
		return dummyWatIgnore{}
	}
	return ignore
}

type matcher interface {
	Match(string, bool) bool
}

type dummyWatIgnore struct{}

// Dummy Wat Ignore doesn't match any files (i.e. doesn't ignore anything)
func (dummyWatIgnore) Match(string, bool) bool {
	return false
}
