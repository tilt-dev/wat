package wat

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"sort"

	"github.com/spf13/cobra"
	"github.com/windmilleng/wat/os/ospath"
)

var kNumMostRecent = 1

var recentCmd = &cobra.Command{
	Use:   "recent",
	Short: "Print the N most recently edited files",
	Run:   recent,
}

type fileInfo struct {
	name    string
	modTime time.Time
}

func (f fileInfo) String() string {
	return fmt.Sprintf("%s (%s)", f.name, f.modTime.String())
}

type fileInfos []fileInfo

func (infos fileInfos) Len() int {
	return len(infos)
}

func (infos fileInfos) Less(i, j int) bool {
	return infos[i].modTime.Before(infos[j].modTime)
}

func (infos fileInfos) Swap(i, j int) {
	infos[i], infos[j] = infos[j], infos[i]
}

var _ sort.Interface = fileInfos{}

func recent(cmd *cobra.Command, args []string) {
	ws, err := GetOrInitWatWorkspace()
	if err != nil {
		ws.Fatal("GetWatWorkspace", err)
	}

	files, err := RecentFileNames(ws)
	if err != nil {
		ws.Fatal("RecentFileNames", err)
	}
	for _, f := range files {
		fmt.Println(f)
	}
}

func RecentFileInfos(ws WatWorkspace) ([]fileInfo, error) {
	files, err := ws.WalkRoot()
	if err != nil {
		return nil, err
	}

	sort.Sort(fileInfos(files))
	// Might want to make this accept an arg for # of files to return...
	return files[len(files)-kNumMostRecent:], nil
}

func RecentFileNames(ws WatWorkspace) ([]string, error) {
	files, err := RecentFileInfos(ws)
	if err != nil {
		return nil, err
	}
	strs := []string{}
	for _, f := range files {
		strs = append(strs, f.name)
	}
	return strs, nil
}

func (w WatWorkspace) WalkRoot() ([]fileInfo, error) {
	return w.WalkDir(w.root)
}

func (w WatWorkspace) WalkDir(dir string) ([]fileInfo, error) {
	return walkDir(dir)
}

func walkDir(dir string) ([]fileInfo, error) {
	// Assume watIgnore lives in this directory
	ignore := watIgnoreOrDummy(filepath.Join(dir, fnameWatIgnore))

	dir, err := ospath.RealAbs(dir)
	if err != nil {
		return nil, err
	}

	files := []fileInfo{}
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		shouldIgnore := ignore.Match(path, info.IsDir())
		if shouldIgnore {
			if info.IsDir() {
				// skip whole dir
				return filepath.SkipDir
			}
			// ignored file, don't add to result
			return nil
		}
		if info.Mode().IsRegular() {
			name, _ := ospath.Child(dir, path)
			files = append(files, fileInfo{name: name, modTime: info.ModTime()})
		}
		return nil
	})

	return files, err
}
