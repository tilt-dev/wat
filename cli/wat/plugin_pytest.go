package wat

import (
	"context"
	"os/exec"

	"fmt"
	"regexp"
)

var rePrefix = regexp.MustCompile("(^|/)test_(.*)\\.py")
var rePostfix = regexp.MustCompile("(.*)_test\\.py")
var testRegexps = []*regexp.Regexp{rePrefix, rePostfix}

type PluginPytest struct{}

func (PluginPytest) name() string { return "python+pytest populate" }

func (PluginPytest) run(ctx context.Context, root string) ([]WatCommand, error) {
	// 1. is pytest relevant?
	if !projUsesPytest(ctx, root) {
		return nil, nil
	}

	// 2. where are the test files?
	files, err := findTestFiles(ctx, root)
	if err != nil {
		return nil, fmt.Errorf("findTestFiles: %v", err)
	}

	// 3. which tests --> which files?
	watCmds := []WatCommand{}
	for _, f := range files {
		filePattern := testFileToPattern(root, f.name)
		watCmds = append(watCmds, WatCommand{
			Command:     fmt.Sprintf("pytest %s", f.name),
			FilePattern: filePattern,
		})
	}

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
func findTestFiles(ctx context.Context, root string) ([]fileInfo, error) {
	allFiles, err := walkDir(root)
	if err != nil {
		return nil, fmt.Errorf("walkDir: %v", err)
	}
	return filterFilesMatchAny(allFiles, testRegexps), nil
}

func testFileToPattern(root, test string) string {
	// Fallthrough: associate with all .py files
	return "**/*.py"
}

func baseFile(test string) string {
	if substrs := rePrefix.FindStringSubmatch(test); len(substrs) > 1 {
		fmt.Println(substrs)
		return fmt.Sprintf("%s.py", substrs[1])
	}
	if substrs := rePostfix.FindStringSubmatch(test); len(substrs) > 1 {
		fmt.Println(substrs)
		return fmt.Sprintf("%s.py", substrs[1])
	}
	return ""
}

var _ plugin = PluginPytest{}
