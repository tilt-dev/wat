package wat

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestPluginNodeJSEmptyRepo(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()

	plugin := PluginNodeJS{}
	cmds, err := plugin.run(f.ctx, f.root.Path())
	if err != nil {
		t.Fatal(err)
	}

	if len(cmds) != 0 {
		t.Errorf("Expected 0 commands. Actual: %v", cmds)
	}
}

func TestPluginNodeJSBasic(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()

	f.write("package.json", `
{
  "name": "my-package",
  "scripts": {
    "test": "jest"
  }
}`)

	plugin := PluginNodeJS{}
	cmds, err := plugin.run(f.ctx, f.root.Path())
	if err != nil {
		t.Fatal(err)
	}

	if len(cmds) != 1 || !strings.Contains(cmds[0].Command, "jest") {
		t.Errorf("Expected 1 jest command. Actual: %v", cmds)
	}
}

func TestPluginNodeJSMultiple(t *testing.T) {
	f := newWatFixture(t)
	defer f.tearDown()

	// Adapted from https://github.com/react-navigation/react-navigation/blob/master/package.json
	f.write("package.json", `
{
  "name": "my-package",
  "scripts": {
    "test": "npm run lint && npm run jest",
    "eslint": "eslint .",
    "format": "eslint --fix .",
    "jest": "jest",
    "ios": "cd examples/NavigationPlayground && yarn && yarn ios"
  }
}`)

	plugin := PluginNodeJS{}
	cmds, err := plugin.run(f.ctx, f.root.Path())
	if err != nil {
		t.Fatal(err)
	}

	if len(cmds) != 3 {
		t.Errorf("Expected 3 commands. Actual: %v", cmds)
	}

	cmdStrs := make([]string, 0)
	for _, cmd := range cmds {
		// Strip out the part of the command that sets PATH
		cmdStr := strings.Join(strings.Split(cmd.Command, " ")[1:], " ")
		cmdStrs = append(cmdStrs, cmdStr)
	}
	sort.Strings(cmdStrs)

	expected := []string{
		"eslint .",
		"jest",
		"npm run lint && npm run jest",
	}
	if !reflect.DeepEqual(expected, cmdStrs) {
		t.Errorf("Expected %v. Actual: %v", expected, cmdStrs)
	}
}
