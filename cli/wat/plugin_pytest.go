package wat

import (
	"context"
	"os/exec"

	"fmt"
	"regexp"
)

var testRegexps = []*regexp.Regexp{
	regexp.MustCompile("test_.*\\.py"),
	regexp.MustCompile(".*_test\\.py"),
}

type PluginPytest struct{}

func (PluginPytest) name() string { return "python+pytest populate" }

func (PluginPytest) run(ctx context.Context, root string) ([]WatCommand, error) {
	// 1. is pytest relevant?
	if !projUsesPytest(ctx, root) {
		return nil, nil
	}

	// 2. where are the test files? what are the commands?
	cmds, err := findTestFiles(ctx, root)
	if err != nil {
		return nil, fmt.Errorf("findTestFiles: %v", err)
	}

	watCmds := []WatCommand{}
	for _, c := range cmds {
		watCmds = append(watCmds, WatCommand{Command: c})
	}

	// 3. which tests --> which files?
	// TBD ( ⚆ _ ⚆ )

	return watCmds, nil
}

// Here's the naive implementation. Other possibilities:
// a. are there py files?
// b. is some other framework configured here that takes precedence?
// c. pytest-related files in .gitignore, requirements.txt, setup.py?
// d. config files
// + path/pytest.ini
// + path/setup.cfg  (must also contain [tool:pytest] section to match)
// + path/tox.ini    (must also contain [pytest] section to match)
// + pytest.ini
func projUsesPytest(ctx context.Context, root string) bool {
	// is there a pytest executable?
	cmd := exec.CommandContext(ctx, "python", "-c", "import pytest")
	err := cmd.Run()
	if err != nil {
		return false
	}

	return true
}

// This is the naive function that just finds test_*.py files, returns their
// invocations (NOT caring about associated code)
func findTestFiles(ctx context.Context, root string) ([]string, error) {
	allFiles, err := walkDir(root)
	if err != nil {
		return nil, fmt.Errorf("walkDirWithRegexp: %v", err)
	}
	testFiles := filterFilesMatchAny(allFiles, testRegexps)

	cmds := []string{}
	for _, info := range testFiles {
		cmds = append(cmds, fmt.Sprintf("pytest %s", info.name))
	}
	return cmds, nil
}

var _ plugin = PluginPytest{}
