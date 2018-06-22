package wat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"os"

	"github.com/windmilleng/wat/errors"
)

// GetUserPlugins gets user-specified WatList Plugins
// (stored as a newline-separated list in .wat/plugins)
func getUserPlugins(ws WatWorkspace) ([]plugin, error) {
	exists, err := ws.Exists(fnameUserPlugins)
	if !exists || err != nil {
		return nil, err
	}

	contents, err := ws.Read(fnameUserPlugins)
	if err != nil {
		return nil, err
	}

	res := []plugin{}
	for _, cmd := range strings.Split(string(contents), "\n") {
		res = append(res, userPlugin{cmd: cmd})
	}

	return res, nil
}

type plugin interface {
	name() string
	run(ctx context.Context, root string) ([]WatCommand, error)
}

type userPlugin struct{ cmd string }

func (p userPlugin) name() string { return p.cmd }
func (p userPlugin) run(ctx context.Context, root string) ([]WatCommand, error) {
	result := []WatCommand{}

	rawRes, err := runAndCaptureStdout(ctx, root, p.cmd)
	if err != nil {
		return result, err
	}

	jErr := json.Unmarshal(rawRes, &result)
	if jErr != nil {
		return result, fmt.Errorf("[json.Unmarshal: %q] %v", rawRes, jErr)
	}

	return result, nil
}

var _ plugin = userPlugin{}

// NOTE: to register your builtin plugin with wat, add it to this array!
var builtins = []plugin{
	PluginGo{},
	PluginNodeJS{},
	//PluginPytest{}, BUG: should only match `test_` at beginning of file name
}

func RunBuiltinPlugins(ctx context.Context, ws WatWorkspace) (result []WatCommand) {
	return runPlugins(ctx, ws, builtins)
}

func RunUserPlugins(ctx context.Context, ws WatWorkspace) (result []WatCommand, err error) {
	plugins, err := getUserPlugins(ws)
	if err != nil {
		return []WatCommand{}, errors.Propagatef(err, "getUserPlugins")
	}

	return runPlugins(ctx, ws, plugins), nil
}

func runPlugins(ctx context.Context, ws WatWorkspace, plugins []plugin) (result []WatCommand) {
	for _, p := range plugins {
		res, err := p.run(ctx, ws.root)
		if err != nil {
			// TODO: what kind of err is serious enough to return up?
			fmt.Fprintf(os.Stderr, "ERROR (p: '%s'): %v\n", p.name(), err)
			continue
		}
		result = append(result, res...)
	}

	return result
}

func runAndCaptureStdout(ctx context.Context, root, invocation string) ([]byte, error) {
	outBuf := bytes.Buffer{}
	errBuf := bytes.Buffer{}

	err := runCmd(ctx, root, invocation, &outBuf, &errBuf)

	if ctx.Err() != nil {
		// Propagate cancel/timeout errors
		return []byte{}, ctx.Err()
	}
	if err != nil {
		return []byte{}, fmt.Errorf("ERROR %v (%s)", err, errBuf.String())
	}

	// The result of this plugin is just whatever we got into stdOut
	return outBuf.Bytes(), nil
}
