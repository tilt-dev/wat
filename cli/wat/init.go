package wat

import (
	"fmt"
	"os"

	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/windmilleng/wat/cli/analytics"
	"github.com/windmilleng/wat/os/ospath"
)

const kWatDirName = ".wat"

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Make the current directory into a wat project root",
	Run:   initWat,
}

// Init makes the given directory into a wat project root (i.e. creates a .wat/ directory)
func Init(a analytics.Analytics, dir string) error {
	a.Count(statInit, map[string]string{tagDir: dir}, 1)
	path := filepath.Join(dir, kWatDirName)
	return os.MkdirAll(path, permDir)
}

func initWat(cmd *cobra.Command, args []string) {
	if ospath.IsDir(kWatDirName) {
		fmt.Fprintf(os.Stderr, "%s directory already exists here, nothing to do.\n", kWatDirName)
		return
	}

	ws, err := GetOrInitWatWorkspace()
	if err != nil {
		ws.Fatal("initWat", err)
	}

	fmt.Fprintln(os.Stderr, "Successfully initialized wat")
}

func GetOrInitWatWorkspace() (WatWorkspace, error) {
	a := analytics.NewMemoryAnalytics() // ANALYTICS: should be remote analytics
	wd, err := ospath.Realwd()
	if err != nil {
		// Even if there's an error, we guarantee that the returned workspace will have a valid Analytics
		return WatWorkspace{a: a}, err
	}

	return GetOrInitWatWorkspaceAt(wd, a)

}

func GetOrInitWatWorkspaceAt(wd string, a analytics.Analytics) (WatWorkspace, error) {
	// Even if there's an error, we guarantee that the returned workspace will have a valid Analytics
	ws := WatWorkspace{a: a}

	root, err := watRoot(wd)
	if err == nil {
		ws.root = root
		return ws, nil
	}

	if err != ErrNoWatRoot {
		return ws, err
	}

	err = Init(a, wd)
	if err != nil {
		return ws, err
	}

	err = MakeWatIgnore(wd)
	if err != nil {
		return ws, err
	}

	ws.root = wd
	return ws, nil
}
