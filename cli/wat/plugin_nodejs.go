package wat

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// A simple whitelist of common nodejs testing frameworks and how they
// are typically called. This doesn't need to be exhaustive.
var commonNodeTestScripts = map[string]bool{
	"jest":  true,
	"mocha": true,
	"mocha --require babel-register": true,
	"eslint .":                       true,
	"jasmine":                        true,
}

type PackageJSON struct {
	Scripts map[string]string `json:"scripts"`
}

type PluginNodeJS struct {
}

func (p PluginNodeJS) name() string {
	return "nodejs"
}

func (p PluginNodeJS) parsePackageJSON(root string) (PackageJSON, error) {
	packageJSONPath := filepath.Join(root, "package.json")
	packageJSONContents, err := ioutil.ReadFile(packageJSONPath)
	if err != nil {
		if os.IsNotExist(err) {
			return PackageJSON{}, nil
		}
		return PackageJSON{}, fmt.Errorf("Read package.json: %v", err)
	}

	packageJSON := PackageJSON{}
	err = json.Unmarshal(packageJSONContents, &packageJSON)
	if err != nil {
		return PackageJSON{}, fmt.Errorf("Parse package.json: %v", err)
	}

	return packageJSON, nil
}

func (p PluginNodeJS) run(ctx context.Context, root string) ([]WatCommand, error) {
	packageJSON, err := p.parsePackageJSON(root)
	if len(packageJSON.Scripts) == 0 || err != nil {
		return nil, err
	}

	cmds := make([]WatCommand, 0)
	for key, value := range packageJSON.Scripts {
		value = strings.TrimSpace(value)

		if key == "test" {
			cmds = append(cmds, p.toWatCommand(value))
			continue
		}

		if commonNodeTestScripts[value] {
			cmds = append(cmds, p.toWatCommand(value))
			continue
		}
	}
	return cmds, nil
}

func (p PluginNodeJS) toWatCommand(test string) WatCommand {
	return WatCommand{
		Command:     fmt.Sprintf(`PATH="node_modules/.bin:$PATH" %s`, test),
		FilePattern: "**/*.js",
	}
}

var _ plugin = PluginNodeJS{}
