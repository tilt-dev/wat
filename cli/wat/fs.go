package wat

import (
	"bytes"
	"encoding/json"
	"path/filepath"

	"fmt"

	"io/ioutil"

	"os"

	"strings"

	"github.com/pkg/errors"
	"github.com/windmilleng/wat/cli/analytics"
	"github.com/windmilleng/wat/os/ospath"
)

var ErrNoWatRoot = errors.New("No wat root found")

const (
	fnameCmdLog      = "cmdlog"
	fnameList        = "list"
	fnameUserPlugins = "user_plugins"
	fnameGitIgnore   = ".gitignore"
	fnameWatIgnore   = ".watignore"
)

const (
	permDir  = os.FileMode(0755)
	permFile = os.FileMode(0640)
)

type WatWorkspace struct {
	root string
	a    analytics.Analytics
}

func (ws WatWorkspace) Fatal(msg string, err error) {
	tags := map[string]string{tagError: fmt.Sprintf("%v", err)}
	ws.a.Incr(statFatal, tags)
	fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	os.Exit(1)
}

func watRoot(dir string) (string, error) {
	watDir := filepath.Join(dir, kWatDirName)
	if ospath.IsDir(watDir) {
		// .wat/ exists here, yay!
		return dir, nil
	}
	if dir == filepath.Dir(dir) {
		// Visited all parent directories w/o finding watRoot
		return "", ErrNoWatRoot
	}

	return watRoot(filepath.Dir(dir))
}

func (w WatWorkspace) Root() string {
	return w.root
}

// Write (over)writes the specified file in the .wat dir (<wat_root>/.wat/<name>)
// with the given contents
func (w WatWorkspace) Write(name, contents string) error {
	path := filepath.Join(w.root, kWatDirName, name)

	// make any intervening dirs
	err := os.MkdirAll(filepath.Dir(path), permDir)
	if err != nil {
		return fmt.Errorf("[write] os.MkdirAll: %v", err)
	}

	bytes := []byte(contents)
	return ioutil.WriteFile(path, bytes, permFile)
}

// Append appends the given contents to the specified file in the .wat dir (<wat_root>/.wat/<name>)
func (w WatWorkspace) Append(name, contents string) error {
	path := filepath.Join(w.root, kWatDirName, name)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, permFile)
	if err != nil {
		return fmt.Errorf("os.OpenFile: %v", err)
	}
	defer file.Close()

	_, err = file.WriteString(contents)
	return err

}

// AppendLine appends the given contents as a newline
func (w WatWorkspace) AppendLine(name, contents string) error {
	return w.Append(name, fmt.Sprintf("%s\n", contents))
}

func (w WatWorkspace) Exists(name string) (bool, error) {
	_, err := w.Stat(name)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	return !os.IsNotExist(err), nil
}

func (w WatWorkspace) Stat(name string) (os.FileInfo, error) {
	path := filepath.Join(w.root, kWatDirName, name)
	return os.Stat(path)
}

// Read reads the specified file from the .wat dir (<wat_root>/.wat/<name>)
func (w WatWorkspace) Read(name string) ([]byte, error) {
	path := filepath.Join(w.root, kWatDirName, name)
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadFile: %v (while reading file '%s')", err, name)
	}
	return contents, nil
}

func ReadCmdLogGroups(ws WatWorkspace) ([]CommandLogGroup, error) {
	contents, err := ws.Read(fnameCmdLog)
	if err != nil {
		return nil, err
	}

	result := make([]CommandLogGroup, 0)
	decoder := json.NewDecoder(bytes.NewBuffer(contents))
	for decoder.More() {
		var entry CommandLogGroup
		err = decoder.Decode(&entry)
		if err != nil {
			return nil, err
		}
		result = append(result, entry)
	}

	return result, nil
}

func CmdLogGroupsToFile(ws WatWorkspace, gs []CommandLogGroup) error {
	if len(gs) == 0 {
		return nil
	}

	lines := []string{}
	for _, g := range gs {
		lines = append(lines, MustJson(g))
	}
	writeStr := strings.Join(lines, "\n")
	return ws.Append(fnameCmdLog, fmt.Sprintf("%s\n", writeStr))
}
