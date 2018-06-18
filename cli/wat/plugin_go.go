package wat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/windmilleng/wat/os/ospath"
)

type PluginGo struct{}

func (PluginGo) name() string { return "go populate" }

func (PluginGo) run(ctx context.Context, root string) ([]WatCommand, error) {
	var result []WatCommand

	goListEntries, err := goList(ctx, root)
	if err != nil {
		return nil, err
	}

	for _, e := range goListEntries {
		cmd := e.toWatCommand(root)
		if cmd.Empty() {
			continue
		}
		result = append(result, cmd)
	}
	return result, nil
}

type goListEntry struct {
	Dir         string
	ImportPath  string
	TestGoFiles []string
}

func (e goListEntry) toWatCommand(root string) WatCommand {
	if len(e.TestGoFiles) == 0 {
		return WatCommand{}
	}

	// We do not expect to have commands that reach outside
	// the wat workspace, but if we do, skipping them for
	// now seems like the right way to handle the error.
	child, isUnderRoot := ospath.Child(root, e.Dir)
	if !isUnderRoot {
		return WatCommand{}
	}

	return WatCommand{
		Command:     fmt.Sprintf("go test %s", e.ImportPath),
		FilePattern: filepath.Join(child, "*"),
	}
}

func goList(ctx context.Context, root string) ([]goListEntry, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-json", "./...")
	cmd.Dir = root

	output, err := cmd.Output()
	if err != nil {
		exitErr, isExit := err.(*exec.ExitError)
		if isExit {
			return nil, fmt.Errorf("go list: %s (%q)", err.Error(), string(exitErr.Stderr))
		}
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewBuffer(output))
	var result []goListEntry
	for decoder.More() {
		var entry goListEntry
		err = decoder.Decode(&entry)
		if err != nil {
			return nil, err
		}
		result = append(result, entry)
	}

	return result, nil
}

var _ plugin = PluginGo{}
