package wat

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/windmilleng/wat/cli/analytics"
	"github.com/windmilleng/wat/os/ospath"
	"github.com/windmilleng/wat/os/temp"
)

type watFixture struct {
	t      *testing.T
	ctx    context.Context
	root   *temp.TempDir
	origWd string // For resetting wd at teardown
}

func newWatFixture(t *testing.T) *watFixture {
	wd, err := ospath.Realwd()
	if err != nil {
		t.Fatalf("Error getting wd: %v", err)
	}

	root, err := temp.NewDir(t.Name())
	if err != nil {
		t.Fatalf("Error making temp dir: %v", err)
	}

	err = os.Chdir(root.Path())
	if err != nil {
		t.Fatalf("Error setting wd to temp dir: %v", err)
	}

	w := analytics.NewNoopAnyWriter()
	watlytics = &watAnalytics{
		init: analytics.NewStringWriter(w),
		recs: &recEventWriter{del: w},
		errs: analytics.NewErrorWriter(w),
	}

	return &watFixture{
		t:      t,
		ctx:    context.Background(),
		root:   root,
		origWd: wd,
	}
}

func (f *watFixture) tearDown() {
	err := f.root.TearDown()
	if err != nil {
		f.t.Fatalf("Error tearing down temp dir: %v", err)
	}

	err = os.Chdir(f.origWd)
	if err != nil {
		f.t.Fatalf("Error resetting wd: %v", err)
	}
}

// watInit adds a .wat directory in the fixture's temp dir.
func (f *watFixture) watInit() WatWorkspace {
	err := os.Mkdir(filepath.Join(f.root.Path(), kWatDirName), os.FileMode(0777))
	if err != nil {
		f.t.Fatalf("Error making dir: %v", err)
	}
	return WatWorkspace{root: f.root.Path()}
}

func (f *watFixture) writeFile(name string) {
	f.write(name, "hello world")
}

func (f *watFixture) write(name, contents string) {
	rtPath := f.root.Path()
	path := filepath.Join(rtPath, name)

	err := os.MkdirAll(filepath.Dir(path), os.FileMode(0777))
	if err != nil {
		f.t.Fatalf("Error making dir: %v", err)
	}

	err = ioutil.WriteFile(path, []byte(contents), os.FileMode(0777))
	if err != nil {
		f.t.Fatalf("Error writing file: %v", err)
	}
}

func (f *watFixture) assertExpectedFiles(expected map[string]bool, actual []fileInfo) {
	expectedCount := 0
	for _, v := range expected {
		if v {
			expectedCount++
		}
	}

	if len(actual) != expectedCount {
		f.t.Fatalf("expected %d files returned, got %d", expectedCount, len(actual))
	}
	for _, file := range actual {
		expected, ok := expected[file.name]
		if !ok {
			f.t.Fatalf("Returned file that wasn't even in our map?? %s", file.name)
		}
		if !expected {
			f.t.Fatalf("File %s should have been ignored", file.name)
		}
	}
}

func assertCommandLogs(t *testing.T, expected, actual []CommandLog) {
	if len(actual) != len(expected) {
		t.Fatalf("Expected %d logs but got %d", len(expected), len(actual))
	}
	for i, l := range expected {
		if actual[i].Command != l.Command || actual[i].Success != l.Success {
			t.Fatalf("Expected log %+v, got %+v (NOTE: not comparing duration)", l, actual[i])
		}
	}
}

func assertContainsStrings(t *testing.T, expectedStrs []string, output, outputName string) {
	for _, s := range expectedStrs {
		if !strings.Contains(output, s) {
			t.Fatalf("%s does not contain expected string: %q", outputName, s)
		}
	}

}

func assertCmdLogFileContents(t *testing.T, expected CommandLogGroup) {
	logPath := filepath.Join(kWatDirName, fnameCmdLog)
	logContents, err := ioutil.ReadFile(logPath)
	if err != nil {
		t.Fatalf("[ioutil.ReadFile] %v", err)
	}
	actual := CommandLogGroup{}
	err = json.Unmarshal(logContents, &actual)
	if err != nil {
		t.Fatalf("[ioutil.ReadFile] %v", err)
	}

	assertCommandLogGroupsEqual(t, expected, actual)
}

func assertCommandLogGroupsEqual(t *testing.T, expected CommandLogGroup, actual CommandLogGroup) {
	assertLogContext(t, expected.Context, actual.Context)

	if len(expected.Logs) != len(actual.Logs) {
		t.Fatalf("CommandLogGroups not equal: expected %d logs, got %d",
			len(expected.Logs), len(actual.Logs))
	}
	for i, eLog := range expected.Logs {
		assertCommandLog(t, eLog, actual.Logs[i])
	}
}

func assertCommandLog(t *testing.T, expected CommandLog, actual CommandLog) {
	// zero out Duration cuz we can't reliably compare it
	cleanActual := actual
	cleanActual.Duration = 0

	if !reflect.DeepEqual(expected, cleanActual) {
		t.Fatalf("Logs not equal: expected %+v, got %+v (NOTE: not comparing duration)", expected, actual)
	}
}

func assertLogContext(t *testing.T, expected LogContext, actual LogContext) {
	// we don't care about testing StartTime
	cleanExpected := actual
	cleanExpected.StartTime = time.Unix(0, 0)
	cleanActual := actual
	cleanActual.StartTime = time.Unix(0, 0)

	if !reflect.DeepEqual(cleanExpected, cleanActual) {
		t.Fatalf("Log contexts not equal: expected %+v, got %+v", cleanExpected, cleanActual)
	}
}

func assertCommandList(t *testing.T, exp, act CommandList) {
	// We don't care about Timestamp, it gets wonky between json un/marshal
	cleanExp := exp
	cleanExp.Timestamp = time.Unix(0, 0)
	cleanAct := act
	cleanAct.Timestamp = time.Unix(0, 0)

	if !reflect.DeepEqual(cleanExp, cleanAct) {
		t.Fatalf("Expected CommandList %v; got %v (NOTE: not comparing timestamps)", exp, act)
	}

}
