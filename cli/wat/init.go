package wat

import (
	"fmt"
	"os"

	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/windmilleng/wat/os/ospath"
)

const kWatDirName = ".wat"

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Make the current directory into a wat project root",
	Run:   initWat,
}

// Init makes the given directory into a wat project root (i.e. creates a .wat/ directory)
func Init(dir string) error {
	// ANALYTICS: log stat
	path := filepath.Join(dir, kWatDirName)
	return os.MkdirAll(path, permDir)
}

func initWat(cmd *cobra.Command, args []string) {
	if ospath.IsDir(kWatDirName) {
		fmt.Fprintf(os.Stderr, "%s directory already exists here, nothing to do.\n", kWatDirName)
		return
	}

	_, err := GetOrInitWatWorkspace()
	if err != nil {
		Fatal("initWat", err)
	}

	fmt.Fprintln(os.Stderr, "Successfully initialized wat")
}

func GetOrInitWatWorkspace() (WatWorkspace, error) {
	wd, err := ospath.Realwd()
	if err != nil {
		return WatWorkspace{}, err
	}

	return GetOrInitWatWorkspaceAt(wd)
}

func GetOrInitWatWorkspaceAt(wd string) (WatWorkspace, error) {
	ws, err := GetWatWorkspaceAt(wd)
	if err == nil {
		return ws, nil
	}

	if err != ErrNoWatRoot {
		return WatWorkspace{}, err
	}

	err = Init(wd)
	if err != nil {
		return WatWorkspace{}, err
	}

	err = MakeWatIgnore(wd)
	if err != nil {
		return WatWorkspace{}, err
	}

	return WatWorkspace{root: wd}, nil
}
